// Session worker 回归测试：验证伤势恢复阶段资源不足时在同一事务内正常结束行动。
package service

import (
	"encoding/json"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/engine"
	"idle/internal/models"
)

func TestStartSessionRejectsMissingAmmo(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-start-ammo")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	service := NewSessionService(db, models.DefaultUserID)
	if _, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		AmmoID: "ammo_762x39_n4", AmmoRounds: 0,
	}); err == nil {
		t.Fatal("缺少最低携弹量时应拒绝启动 Session")
	}
}

func TestResumeAfterInjuryWithoutAmmoFinishesSession(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-injury-resource")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	snapshot, snapshotJSON, snapshotHash, err := buildScenarioSnapshot(db, models.DefaultUserID, "city_ruins")
	if err != nil {
		t.Fatalf("构建测试场景快照: %v", err)
	}
	var character models.Character
	if err := db.Where("user_id = ?", models.DefaultUserID).First(&character).Error; err != nil {
		t.Fatal(err)
	}
	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	state, err := buildEngineState(db, models.DefaultUserID, character, loadout, engine.CarriedAmmo{
		ID: "ammo_762x39_n4", CaliberID: "762x39", Level: 4, Rounds: 2,
		PreferredID: "ammo_762x39_n4", PreferredLevel: 4, TargetRounds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	state.Character.Injury = "heavy"
	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", models.DefaultUserID, "cash").Update("quantity", 0).Error; err != nil {
		t.Fatal(err)
	}
	ammoBefore, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_762x39_n4")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	nextRunAt := now.Add(-time.Second)
	sess := models.Session{
		UserID: models.DefaultUserID, CharacterID: character.ID, MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		Status: "waiting_injury", Seed: 20260825, StartTime: now.Add(-time.Hour), NextRunAt: &nextRunAt,
		EngineVersion: engine.EngineVersion, ScenarioSnapshot: snapshotJSON, ScenarioHash: snapshotHash, StateJSON: string(stateJSON),
	}
	if err := db.Create(&sess).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSessionService(db, models.DefaultUserID)
	if err := service.resumeAfterInjury(&sess, &state, snapshot, now); err != nil {
		t.Fatalf("伤势恢复资源不足: %v", err)
	}
	if sess.Status != "finished" {
		t.Fatalf("资源不足后的 Session 状态 = %s，期望 finished", sess.Status)
	}
	var event models.SessionEvent
	if err := db.Where("user_id = ? AND session_id = ? AND event_type = ?", models.DefaultUserID, sess.ID, sessionEventSessionFinished).First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["reason"] != "resource_unavailable" || payload["detail"] != "ammo_unavailable" {
		t.Fatalf("资源不足结束事件 = %+v", payload)
	}
	var storedCharacter models.Character
	if err := db.Where("user_id = ? AND id = ?", models.DefaultUserID, character.ID).First(&storedCharacter).Error; err != nil {
		t.Fatal(err)
	}
	if storedCharacter.Injury != "none" || storedCharacter.InjuryUntil != nil {
		t.Fatalf("资源不足结束后伤势未收口: injury=%s until=%v", storedCharacter.Injury, storedCharacter.InjuryUntil)
	}
	ammoAfter, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_762x39_n4")
	if err != nil {
		t.Fatal(err)
	}
	if ammoAfter != ammoBefore+2 {
		t.Fatalf("伤势恢复补给失败后的弹药 = %d，期望只返还 2 发为 %d", ammoAfter, ammoBefore+2)
	}
	if state.Ammo.ID != "" || state.Ammo.Rounds != 0 {
		t.Fatalf("伤势恢复结束后 Session 弹药未清空: %+v", state.Ammo)
	}
	if err := service.simulateSession(sess.ID); err != nil {
		t.Fatalf("重试已结束伤势恢复 Session: %v", err)
	}
	ammoAfterRetry, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_762x39_n4")
	if err != nil {
		t.Fatal(err)
	}
	if ammoAfterRetry != ammoAfter {
		t.Fatalf("重试 worker 后弹药 = %d，期望保持 %d", ammoAfterRetry, ammoAfter)
	}
}
