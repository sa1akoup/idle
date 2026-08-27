// Session 计划完整性测试：验证待结算结果只依赖持久化输入，不受实时状态和目录变化影响。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/database"

	"gorm.io/gorm"
)

func pendingRunTestData(t *testing.T) (models.Session, engine.EngineState) {
	t.Helper()
	state := engine.EngineState{
		Character: engine.CharacterState{Stress: 10},
		Loadout:   engine.LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
		Ammo:      engine.CarriedAmmo{ID: "ammo_test", CaliberID: "9x19", Level: 3, Rounds: 30, PreferredID: "ammo_test", PreferredLevel: 3, TargetRounds: 30},
	}
	result := engine.RunResult{
		Result:      "success",
		DurationSec: 60,
		Trace: []engine.TraceEvent{
			{Sequence: 1, Type: engine.TraceRunStarted, OffsetSec: 0},
			{Sequence: 2, Type: engine.TraceNodeEntered, OffsetSec: 60, NodeID: "node_test"},
		},
		NextState: state,
	}
	resultJSON, hash, err := marshalPendingRun(1, state, result)
	if err != nil {
		t.Fatalf("生成待结算计划: %v", err)
	}
	return models.Session{PendingRunIndex: 1, PendingRunResult: resultJSON, PendingRunHash: hash}, state
}

func TestPendingRunRejectsInputStateMismatch(t *testing.T) {
	sess, state := pendingRunTestData(t)
	state.Ammo.Rounds++
	_, err := decodePendingRun(sess, state)
	if !errors.Is(err, ErrPendingRunIntegrity) {
		t.Fatalf("错误 = %v，期望 ErrPendingRunIntegrity", err)
	}
}

func TestPendingRunSettlesAfterCatalogChanges(t *testing.T) {
	sess, state := pendingRunTestData(t)
	// 目录属于 Session 快照之外的实时数据变化，Pending Run 校验不应重新读取它。
	catalog := map[string]string{"weapon_test": "旧武器"}
	delete(catalog, "weapon_test")
	if _, err := decodePendingRun(sess, state); err != nil {
		t.Fatalf("目录变化后读取待结算计划: %v", err)
	}
}

func TestDuePendingRunSettlesExactlyOnce(t *testing.T) {
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-pending-run-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
	})

	if err := db.Create(&models.Character{UserID: models.DefaultUserID, Name: "测试角色"}).Error; err != nil {
		t.Fatal(err)
	}
	stateJSON, err := json.Marshal(engine.EngineState{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SessionRun{UserID: models.DefaultUserID, SessionID: 1, RunIndex: 1, NextState: string(stateJSON)}).Error; err != nil {
		t.Fatal(err)
	}
	sess := &models.Session{ID: 1, UserID: models.DefaultUserID, Status: "running"}
	state := engine.EngineState{}
	service := NewSessionService(db, models.DefaultUserID)
	now := time.Now()
	for i := 0; i < 2; i++ {
		if err := service.settleEngineRun(sess, &state, engine.ScenarioSnapshot{}, engine.RunResult{}, 1, now, now.Add(time.Hour)); err != nil {
			t.Fatalf("第%d次结算: %v", i+1, err)
		}
	}
	var count int64
	if err := db.Model(&models.SessionRun{}).Where("user_id = ? AND session_id = ? AND run_index = ?", models.DefaultUserID, 1, 1).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("同一 Pending Run 结算记录数 = %d，期望 1", count)
	}
}

func newSessionEventsTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-%s-%d.db", name, time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
	})
	return db
}

func TestSessionEventsCursorReturnsEveryEventOnce(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-events-cursor")
	now := time.Now()
	if err := db.Create(&models.Session{UserID: models.DefaultUserID, ID: 1, Status: "running", StartTime: now.Add(-time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	for sequence := 1; sequence <= 3; sequence++ {
		if err := db.Create(&models.SessionEvent{
			UserID: models.DefaultUserID, SessionID: 1, RunIndex: 1, Sequence: sequence,
			EventType: "node_entered", OffsetSec: int64(sequence), AvailableAt: now.Add(-time.Second), PayloadJSON: "{}", CreatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	first, err := ListSessionEvents(db, models.DefaultUserID, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 {
		t.Fatalf("首次读取事件数 = %d，期望 3", len(first))
	}
	second, err := ListSessionEvents(db, models.DefaultUserID, 1, first[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].ID != first[2].ID {
		t.Fatalf("游标读取结果异常: %+v", second)
	}
	third, err := ListSessionEvents(db, models.DefaultUserID, 1, first[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(third) != 0 {
		t.Fatalf("消费完游标后仍返回 %d 个事件", len(third))
	}
}

func TestTerminalSessionHidesFutureEvents(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-events-terminal")
	now := time.Now()
	endTime := now.Add(-time.Second)
	if err := db.Create(&models.Session{UserID: models.DefaultUserID, ID: 1, Status: "success", StartTime: now.Add(-time.Minute), EndTime: &endTime}).Error; err != nil {
		t.Fatal(err)
	}
	for sequence, availableAt := range []time.Time{endTime.Add(-time.Second), endTime.Add(time.Second)} {
		if err := db.Create(&models.SessionEvent{
			UserID: models.DefaultUserID, SessionID: 1, RunIndex: 1, Sequence: sequence + 1,
			EventType: "node_entered", AvailableAt: availableAt, PayloadJSON: "{}", CreatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	events, err := ListSessionEvents(db, models.DefaultUserID, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("终局后的可见事件数 = %d，期望 1", len(events))
	}
}
