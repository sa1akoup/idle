// Session worker 回归测试：验证调度版本与启动前置条件。
package service

import (
	"testing"

	"idle/internal/config"
	"idle/internal/engine"
	"idle/internal/models"
)

func TestStartSessionRejectsMissingAmmo(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-start-ammo")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	scheduler := newStartedTestScheduler(t, db)
	service := NewSessionServiceWithScheduler(db, models.DefaultUserID, scheduler)
	if _, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		AmmoID: "ammo_762x39_n4", AmmoRounds: 0,
	}); err == nil {
		t.Fatal("缺少最低携弹量时应拒绝启动 Session")
	}
}

func TestValidateSessionEngineVersionRejectsLegacy(t *testing.T) {
	if err := validateSessionEngineVersion(engine.EngineVersion); err != nil {
		t.Fatalf("当前引擎版本不应被拒绝: %v", err)
	}
	for _, version := range []string{"", "exploration-engine-v3", "unknown-engine"} {
		if err := validateSessionEngineVersion(version); err == nil {
			t.Fatalf("旧或空引擎版本 %q 应被拒绝", version)
		}
	}
}
