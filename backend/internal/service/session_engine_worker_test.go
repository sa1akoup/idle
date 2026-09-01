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

func TestValidateSessionEngineVersionAcceptsMigratableVersions(t *testing.T) {
	for _, version := range []string{engine.LegacyEngineVersionV16, engine.EngineVersion} {
		if err := validateSessionEngineVersion(version); err != nil {
			t.Fatalf("可处理的引擎版本 %q 不应被拒绝: %v", version, err)
		}
	}
	for _, version := range []string{"", "exploration-engine-v3", "unknown-engine"} {
		if err := validateSessionEngineVersion(version); err == nil {
			t.Fatalf("空或未知引擎版本 %q 应被拒绝", version)
		}
	}
}

func TestSessionEventRunIndexFallsBackToValidValue(t *testing.T) {
	tests := []struct {
		name         string
		sess         models.Session
		wantRunIndex int
	}{
		{name: "pending run", sess: models.Session{PendingRunIndex: 3, TotalRuns: 2}, wantRunIndex: 3},
		{name: "total runs", sess: models.Session{TotalRuns: 2}, wantRunIndex: 2},
		{name: "fresh session", sess: models.Session{}, wantRunIndex: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionEventRunIndex(tt.sess); got != tt.wantRunIndex {
				t.Fatalf("sessionEventRunIndex(%+v) = %d，期望 %d", tt.sess, got, tt.wantRunIndex)
			}
		})
	}
}
