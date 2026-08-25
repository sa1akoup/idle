// Session 弹药集成测试：覆盖启动预留、持久化携弹状态和中止后的精确返还。
package service

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/database"
)

func TestSessionReservesAndReturnsCarriedAmmo(t *testing.T) {
	dsn := fmt.Sprintf("file:session-ammo-%d?mode=memory&cache=shared", time.Now().UnixNano())
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

	const (
		ammoID        = "ammo_762x39_n4"
		carriedRounds = 60
	)
	service := NewSessionService(db, models.DefaultUserID)
	sess, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		AmmoID: ammoID, AmmoRounds: carriedRounds,
	})
	if err != nil {
		t.Fatalf("启动 Session: %v", err)
	}
	remaining, err := ammoInventoryQuantity(db, models.DefaultUserID, ammoID)
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 120 {
		t.Fatalf("启动后仓库弹药 = %d，期望 120", remaining)
	}
	if sess.AmmoID != ammoID || sess.AmmoRounds != carriedRounds {
		t.Fatalf("Session 携弹字段异常: id=%s rounds=%d", sess.AmmoID, sess.AmmoRounds)
	}
	var state engine.EngineState
	if err := json.Unmarshal([]byte(sess.StateJSON), &state); err != nil {
		t.Fatalf("解析 Session 状态: %v", err)
	}
	if state.Ammo.ID != ammoID || state.Ammo.Rounds != carriedRounds || state.Ammo.Level != 4 ||
		state.Ammo.PreferredID != ammoID || state.Ammo.PreferredLevel != 4 || state.Ammo.TargetRounds != carriedRounds {
		t.Fatalf("Session 携弹状态异常: %+v", state.Ammo)
	}
	var snapshot engine.ScenarioSnapshot
	if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &snapshot); err != nil {
		t.Fatalf("解析 Session 场景快照: %v", err)
	}
	if snapshot.SchemaVersion != engine.SchemaVersion || snapshot.AmmoSupplies[ammoID].UnitPrice <= 0 {
		t.Fatalf("Session 弹药补给快照异常: %+v", snapshot.AmmoSupplies[ammoID])
	}
	waitSessionWorker(t, sess.ID)

	if err := service.Abort(sess.ID); err != nil {
		t.Fatalf("中止 Session: %v", err)
	}
	// 重复中止必须保持幂等，不能再次返还弹药。
	if err := service.Abort(sess.ID); err != nil {
		t.Fatalf("重复中止 Session: %v", err)
	}
	returned, err := ammoInventoryQuantity(db, models.DefaultUserID, ammoID)
	if err != nil {
		t.Fatal(err)
	}
	if returned != 180 {
		t.Fatalf("中止后仓库弹药 = %d，期望 180", returned)
	}
	var stored models.Session
	if err := db.First(&stored, "id = ? AND user_id = ?", sess.ID, models.DefaultUserID).Error; err != nil {
		t.Fatalf("读取中止 Session: %v", err)
	}
	if stored.Status != "aborted" || stored.AmmoID != "" || stored.AmmoRounds != 0 {
		t.Fatalf("中止 Session 字段异常: status=%s id=%s rounds=%d", stored.Status, stored.AmmoID, stored.AmmoRounds)
	}
}

func TestSessionSettlementRefillsAmmoAndWritesEvent(t *testing.T) {
	dsn := fmt.Sprintf("file:session-ammo-refill-%d?mode=memory&cache=shared", time.Now().UnixNano())
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

	service := NewSessionService(db, models.DefaultUserID)
	sess, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		AmmoID: "ammo_762x39_n4", AmmoRounds: 60,
	})
	if err != nil {
		t.Fatalf("启动 Session: %v", err)
	}
	waitSessionWorker(t, sess.ID)

	var snapshot engine.ScenarioSnapshot
	var state engine.EngineState
	if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &snapshot); err != nil {
		t.Fatalf("解析场景快照: %v", err)
	}
	if err := json.Unmarshal([]byte(sess.StateJSON), &state); err != nil {
		t.Fatalf("解析 Session 状态: %v", err)
	}
	const fallbackID = "ammo_762x39_n2"
	supply := snapshot.AmmoSupplies[fallbackID]
	if !supply.Available {
		t.Fatalf("初始武器商人应提供 N2 弹药: %+v", supply)
	}
	cashBefore, err := ammoInventoryQuantity(db, models.DefaultUserID, "cash")
	if err != nil {
		t.Fatal(err)
	}

	// 模拟本局消耗到不足一次攻击，结算层应返还剩弹并按快照购买 N2。
	state.Ammo.Rounds = 2
	result := engine.RunResult{
		Result: "success", DurationSec: 1, AmmoUsed: 58, Injury: "none",
		NextState: state, SkipResourceConsumption: true,
	}
	runEndAt := time.Now()
	if err := service.settleEngineRun(sess, &state, snapshot, result, 1, runEndAt, runEndAt.Add(time.Hour)); err != nil {
		t.Fatalf("结算自动补给: %v", err)
	}
	if state.Ammo.ID != fallbackID || state.Ammo.Level != 2 || state.Ammo.Rounds != 60 ||
		state.Ammo.PreferredID != "ammo_762x39_n4" || state.Ammo.TargetRounds != 60 {
		t.Fatalf("结算后 Session 弹药异常: %+v", state.Ammo)
	}
	assertAmmoQuantity(t, db, models.DefaultUserID, "ammo_762x39_n4", 122)
	assertAmmoQuantity(t, db, models.DefaultUserID, "cash", cashBefore-supply.UnitPrice*60)

	var refillEvent models.SessionEvent
	if err := db.Where("user_id = ? AND session_id = ? AND event_type = ?", models.DefaultUserID, sess.ID, sessionEventAmmoRefilled).
		First(&refillEvent).Error; err != nil {
		t.Fatalf("读取自动补给事件: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(refillEvent.PayloadJSON), &payload); err != nil {
		t.Fatalf("解析自动补给事件: %v", err)
	}
	if payload["toAmmoId"] != fallbackID || payload["source"] != "merchant_fallback" {
		t.Fatalf("自动补给事件内容异常: %+v", payload)
	}
}

func TestSessionSettlementAmmoPurchaseFailureReturnsCarriedAmmoOnce(t *testing.T) {
	dsn := fmt.Sprintf("file:session-ammo-refill-failure-%d?mode=memory&cache=shared", time.Now().UnixNano())
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

	service := NewSessionService(db, models.DefaultUserID)
	sess, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		AmmoID: "ammo_762x39_n4", AmmoRounds: 60,
	})
	if err != nil {
		t.Fatalf("启动 Session: %v", err)
	}
	waitSessionWorker(t, sess.ID)

	var snapshot engine.ScenarioSnapshot
	var state engine.EngineState
	if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &snapshot); err != nil {
		t.Fatalf("解析场景快照: %v", err)
	}
	if err := json.Unmarshal([]byte(sess.StateJSON), &state); err != nil {
		t.Fatalf("解析 Session 状态: %v", err)
	}
	state.Ammo.Rounds = 2
	if result := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, "cash").
		Update("quantity", 0); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("清空测试现金: error=%v rows=%d", result.Error, result.RowsAffected)
	}

	runEndAt := time.Now()
	result := engine.RunResult{
		Result: "success", DurationSec: 1, AmmoUsed: 58, Injury: "none",
		NextState: state, SkipResourceConsumption: true,
	}
	if err := service.settleEngineRun(sess, &state, snapshot, result, 1, runEndAt, runEndAt.Add(time.Hour)); err != nil {
		t.Fatalf("现金不足时结算 Session: %v", err)
	}
	if sess.Status != "finished" {
		t.Fatalf("补给失败后的 Session 状态 = %s，期望 finished", sess.Status)
	}
	assertAmmoQuantity(t, db, models.DefaultUserID, "ammo_762x39_n4", 122)
	assertAmmoQuantity(t, db, models.DefaultUserID, "cash", 0)
	if state.Ammo.ID != "" || state.Ammo.Rounds != 0 {
		t.Fatalf("Session 终态仍保留携带弹药: %+v", state.Ammo)
	}
	var refillEvents int64
	if err := db.Model(&models.SessionEvent{}).
		Where("user_id = ? AND session_id = ? AND event_type = ?", models.DefaultUserID, sess.ID, sessionEventAmmoRefilled).
		Count(&refillEvents).Error; err != nil {
		t.Fatalf("统计自动补给事件: %v", err)
	}
	if refillEvents != 0 {
		t.Fatalf("补给失败时自动补给事件数量 = %d，期望 0", refillEvents)
	}

	// worker 重试已结束 Session 必须保持幂等，不能再次返还同一批剩余弹药。
	if err := service.simulateSession(sess.ID); err != nil {
		t.Fatalf("重试已结束 Session: %v", err)
	}
	assertAmmoQuantity(t, db, models.DefaultUserID, "ammo_762x39_n4", 122)
}

func waitSessionWorker(t *testing.T, sessionID uint) {
	t.Helper()
	workerKey := sessionWorkerKey(models.DefaultUserID, sessionID)
	workerDeadline := time.Now().Add(time.Second)
	for {
		if _, active := activeSessionWorkers.Load(workerKey); !active {
			return
		}
		if time.Now().After(workerDeadline) {
			t.Fatal("等待 Session 启动派发结束超时")
		}
		time.Sleep(time.Millisecond)
	}
}
