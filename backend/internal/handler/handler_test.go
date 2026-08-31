// 路由级测试：覆盖认证中间件、注册/登录/登出流转、核心接口状态码与用户隔离。
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/repository/database"
	"idle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// newTestRouter 构造带独立 SQLite 数据库的完整路由。
func newTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-handler-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}

	gin.SetMode(gin.TestMode)
	scheduler := service.NewSessionScheduler(db)
	h := NewHandler(db, false, scheduler)
	r := gin.New()
	h.Register(r)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("读取测试连接: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		for _, suffix := range []string{"", "-wal", "-shm"} {
			_ = os.Remove(dsn + suffix)
		}
	})
	return r, db
}

// doJSON 发起一次 JSON 请求；token 为空时不携带认证信息，否则以 Bearer 形式发送。
func doJSON(t *testing.T, router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

// sessionTokenFrom 从响应 Cookie 中取出认证 Token。
func sessionTokenFrom(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == authCookieName && cookie.Value != "" {
			return cookie.Value
		}
	}
	t.Fatal("响应中没有认证 Cookie")
	return ""
}

// decodeBody 将响应体解码到 out。
func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), out); err != nil {
		t.Fatalf("解析响应体 %s: %v", recorder.Body.String(), err)
	}
}

// registerAndLogin 注册一个唯一用户并返回认证 Token。
func registerAndLogin(t *testing.T, router http.Handler) string {
	t.Helper()
	username := fmt.Sprintf("h_%d", time.Now().UnixNano())
	registered := doJSON(t, router, http.MethodPost, "/api/auth/register",
		map[string]string{"username": username, "password": "test-password-123"}, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("注册状态码 = %d，期望 201，响应: %s", registered.Code, registered.Body.String())
	}
	return sessionTokenFrom(t, registered)
}

func TestProtectedRoutesRejectAnonymous(t *testing.T) {
	router, _ := newTestRouter(t)
	for _, path := range []string{"/api/player", "/api/maps", "/api/merchants", "/api/sessions", "/api/hideout"} {
		recorder := doJSON(t, router, http.MethodGet, path, nil, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("匿名访问 %s 状态码 = %d，期望 401", path, recorder.Code)
		}
	}
}

func TestAuthRegisterMeLogoutFlow(t *testing.T) {
	router, _ := newTestRouter(t)
	username := fmt.Sprintf("flow_%d", time.Now().UnixNano())
	registered := doJSON(t, router, http.MethodPost, "/api/auth/register",
		map[string]string{"username": username, "password": "test-password-123"}, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("注册状态码 = %d，期望 201，响应: %s", registered.Code, registered.Body.String())
	}
	token := sessionTokenFrom(t, registered)

	meRecorder := doJSON(t, router, http.MethodGet, "/api/auth/me", nil, token)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("读取当前用户状态码 = %d，期望 200", meRecorder.Code)
	}
	var me struct {
		Username string `json:"username"`
	}
	decodeBody(t, meRecorder, &me)
	if me.Username != username {
		t.Fatalf("当前用户 = %s，期望 %s", me.Username, username)
	}

	logoutRecorder := doJSON(t, router, http.MethodPost, "/api/auth/logout", nil, token)
	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("登出状态码 = %d，期望 204", logoutRecorder.Code)
	}

	afterLogout := doJSON(t, router, http.MethodGet, "/api/auth/me", nil, token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("登出后访问状态码 = %d，期望 401", afterLogout.Code)
	}
}

func TestRegisterDuplicateAndWrongPasswordRejected(t *testing.T) {
	router, _ := newTestRouter(t)
	const username = "duplicate_user"
	body := map[string]string{"username": username, "password": "test-password-123"}

	first := doJSON(t, router, http.MethodPost, "/api/auth/register", body, "")
	if first.Code != http.StatusCreated {
		t.Fatalf("首次注册状态码 = %d，期望 201", first.Code)
	}
	second := doJSON(t, router, http.MethodPost, "/api/auth/register", body, "")
	if second.Code != http.StatusBadRequest {
		t.Fatalf("重复注册状态码 = %d，期望 400", second.Code)
	}

	wrongPassword := doJSON(t, router, http.MethodPost, "/api/auth/login",
		map[string]string{"username": username, "password": "wrong-password"}, "")
	if wrongPassword.Code != http.StatusUnauthorized {
		t.Fatalf("错误密码登录状态码 = %d，期望 401", wrongPassword.Code)
	}
	correct := doJSON(t, router, http.MethodPost, "/api/auth/login", body, "")
	if correct.Code != http.StatusOK {
		t.Fatalf("正确密码登录状态码 = %d，期望 200", correct.Code)
	}
	if sessionTokenFrom(t, correct) == "" {
		t.Fatal("登录成功但未返回认证 Cookie")
	}
}

func TestPlayerUpdateAndCatalogListEndpoints(t *testing.T) {
	router, _ := newTestRouter(t)
	token := registerAndLogin(t, router)

	playerRecorder := doJSON(t, router, http.MethodGet, "/api/player", nil, token)
	if playerRecorder.Code != http.StatusOK {
		t.Fatalf("读取玩家状态码 = %d，期望 200", playerRecorder.Code)
	}

	updateRecorder := doJSON(t, router, http.MethodPut, "/api/player", map[string]string{"name": "测试幸存者"}, token)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("更新玩家状态码 = %d，期望 200，响应: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated struct {
		Name string `json:"name"`
	}
	decodeBody(t, updateRecorder, &updated)
	if updated.Name != "测试幸存者" {
		t.Fatalf("更新后玩家名 = %s，期望 测试幸存者", updated.Name)
	}

	for _, path := range []string{
		"/api/maps", "/api/enemies", "/api/weapons", "/api/ammos", "/api/armors", "/api/consumables",
		"/api/chestrigs", "/api/backpacks", "/api/helmets", "/api/headsets", "/api/loot",
	} {
		recorder := doJSON(t, router, http.MethodGet, path, nil, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s 状态码 = %d，期望 200", path, recorder.Code)
		}
	}
}

func TestMapGraphAndLoadoutInventoryEndpoints(t *testing.T) {
	router, _ := newTestRouter(t)
	token := registerAndLogin(t, router)

	var maps []struct {
		ID string `json:"id"`
	}
	mapsRecorder := doJSON(t, router, http.MethodGet, "/api/maps", nil, token)
	decodeBody(t, mapsRecorder, &maps)
	if len(maps) == 0 {
		t.Fatal("地图目录为空，无法验证地图图数据")
	}

	graphRecorder := doJSON(t, router, http.MethodGet, "/api/maps/"+maps[0].ID+"/graph", nil, token)
	if graphRecorder.Code != http.StatusOK {
		t.Fatalf("读取地图图数据状态码 = %d，期望 200", graphRecorder.Code)
	}

	var loadout struct {
		WeaponID string `json:"weaponId"`
	}
	loadoutRecorder := doJSON(t, router, http.MethodGet, "/api/loadout", nil, token)
	if loadoutRecorder.Code != http.StatusOK {
		t.Fatalf("读取装备配置状态码 = %d，期望 200", loadoutRecorder.Code)
	}
	decodeBody(t, loadoutRecorder, &loadout)
	if loadout.WeaponID == "" {
		t.Fatal("新用户的装备配置为空")
	}

	for _, path := range []string{
		"/api/loadout/capacity", "/api/inventory", "/api/inventory/capacity",
		"/api/item-instances", "/api/armor-instances", "/api/recovery/current", "/api/sessions",
	} {
		recorder := doJSON(t, router, http.MethodGet, path, nil, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s 状态码 = %d，期望 200，响应: %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestMerchantHideoutAndCraftingEndpoints(t *testing.T) {
	router, _ := newTestRouter(t)
	token := registerAndLogin(t, router)

	var merchants []struct {
		ID string `json:"id"`
	}
	merchantsRecorder := doJSON(t, router, http.MethodGet, "/api/merchants", nil, token)
	if merchantsRecorder.Code != http.StatusOK {
		t.Fatalf("读取商人列表状态码 = %d，期望 200", merchantsRecorder.Code)
	}
	decodeBody(t, merchantsRecorder, &merchants)
	if len(merchants) == 0 {
		t.Fatal("商人目录为空")
	}

	catalogRecorder := doJSON(t, router, http.MethodGet, "/api/merchants/"+merchants[0].ID+"/catalog", nil, token)
	if catalogRecorder.Code != http.StatusOK {
		t.Fatalf("读取商人目录状态码 = %d，期望 200", catalogRecorder.Code)
	}

	for _, path := range []string{"/api/hideout", "/api/crafting/recipes", "/api/inventory"} {
		recorder := doJSON(t, router, http.MethodGet, path, nil, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s 状态码 = %d，期望 200，响应: %s", path, recorder.Code, recorder.Body.String())
		}
	}

	// 维修接口路由与鉴权正确，但新用户没有可维修的归零护甲（初始护甲均满耐久），
// 应被业务规则拒绝；维修成功路径由服务层测试覆盖。
	var armorInstances []struct {
		ID uint `json:"id"`
	}
	instancesRecorder := doJSON(t, router, http.MethodGet, "/api/armor-instances", nil, token)
	if instancesRecorder.Code != http.StatusOK {
		t.Fatalf("读取护甲实例状态码 = %d，期望 200", instancesRecorder.Code)
	}
	decodeBody(t, instancesRecorder, &armorInstances)
	if len(armorInstances) == 0 {
		t.Fatal("新用户没有护甲实例")
	}

	repairRecorder := doJSON(t, router, http.MethodPost, "/api/hideout/repair",
		map[string]uint{"armorInstanceId": armorInstances[0].ID}, token)
	if repairRecorder.Code != http.StatusBadRequest {
		t.Fatalf("维修满耐久护甲状态码 = %d，期望 400（仅归零护甲可维修），响应: %s", repairRecorder.Code, repairRecorder.Body.String())
	}
}

func TestSessionStartQueryAndUserIsolation(t *testing.T) {
	router, _ := newTestRouter(t)
	token := registerAndLogin(t, router)
	otherToken := registerAndLogin(t, router)

	var maps []struct {
		ID string `json:"id"`
	}
	decodeBody(t, doJSON(t, router, http.MethodGet, "/api/maps", nil, token), &maps)
	if len(maps) == 0 {
		t.Fatal("地图目录为空，无法启动行动")
	}

	startRecorder := doJSON(t, router, http.MethodPost, "/api/session/start", map[string]interface{}{
		"mapId": maps[0].ID, "style": "balanced", "recoveryPreset": 1,
	}, token)
	if startRecorder.Code != http.StatusAccepted {
		t.Fatalf("启动行动状态码 = %d，期望 202，响应: %s", startRecorder.Code, startRecorder.Body.String())
	}
	var started struct {
		ID     uint   `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, startRecorder, &started)
	if started.ID == 0 || started.Status != "running" {
		t.Fatalf("启动行动结果异常: %+v", started)
	}

	sessionRecorder := doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/session/%d", started.ID), nil, token)
	if sessionRecorder.Code != http.StatusOK {
		t.Fatalf("读取行动状态码 = %d，期望 200", sessionRecorder.Code)
	}
	eventsRecorder := doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/session/%d/events", started.ID), nil, token)
	if eventsRecorder.Code != http.StatusOK {
		t.Fatalf("读取行动事件状态码 = %d，期望 200", eventsRecorder.Code)
	}

	// 已有行动运行时再次启动应被拒绝。
	againRecorder := doJSON(t, router, http.MethodPost, "/api/session/start", map[string]interface{}{
		"mapId": maps[0].ID, "style": "balanced", "recoveryPreset": 2,
	}, token)
	if againRecorder.Code != http.StatusBadRequest {
		t.Fatalf("重复启动行动状态码 = %d，期望 400", againRecorder.Code)
	}

	// 其他用户不能读取该行动。
	otherRecorder := doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/session/%d", started.ID), nil, otherToken)
	if otherRecorder.Code != http.StatusNotFound {
		t.Fatalf("跨用户读取行动状态码 = %d，期望 404", otherRecorder.Code)
	}
}