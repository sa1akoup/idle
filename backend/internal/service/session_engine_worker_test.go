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
	// 弹药槽全部为空且预设弹药未配置时，枪械开局应被拒绝并提示到角色页配置。
	var testLoadout models.PlayerLoadout
	if err := db.Where("user_id = ?", models.DefaultUserID).First(&testLoadout).Error; err != nil {
		t.Fatalf("读取测试配装: %v", err)
	}
	testLoadout.CarriedAmmo = []models.AmmoCell{{AmmoID: "", Rounds: 0}}
	testLoadout.PresetAmmoID = ""
	testLoadout.PresetAmmoRounds = 0
	if err := db.Model(&models.PlayerLoadout{}).Where("id = ?", testLoadout.ID).
		Select("CarriedAmmo", "PresetAmmoID", "PresetAmmoRounds").Updates(&testLoadout).Error; err != nil {
		t.Fatalf("清空携带弹药槽: %v", err)
	}
	scheduler := newStartedTestScheduler(t, db)
	service := NewSessionServiceWithScheduler(db, models.DefaultUserID, scheduler)
	if _, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
	}); err == nil {
		t.Fatal("未配置携带弹药时应拒绝启动 Session")
	}
}

func TestStartSessionUsesPresetAmmoWhenAllCellsAreEmpty(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-start-empty-ammo-cells")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	var loadout models.PlayerLoadout
	if err := db.Where("user_id = ?", models.DefaultUserID).First(&loadout).Error; err != nil {
		t.Fatalf("读取测试配装: %v", err)
	}
	loadout.CarriedAmmo = []models.AmmoCell{{}, {}, {}, {}}
	if err := db.Model(&models.PlayerLoadout{}).Where("id = ?", loadout.ID).
		Select("CarriedAmmo").Updates(&loadout).Error; err != nil {
		t.Fatalf("写入空弹药槽: %v", err)
	}

	scheduler := newStartedTestScheduler(t, db)
	service := NewSessionServiceWithScheduler(db, models.DefaultUserID, scheduler)
	sess, err := service.Start(StartReq{MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1})
	if err != nil {
		t.Fatalf("空弹药槽应回退预设弹药并启动: %v", err)
	}
	if sess.AmmoID != "ammo_762x39_n2" || sess.AmmoRounds != 30 {
		t.Fatalf("预设弹药回退结果异常: %s/%d", sess.AmmoID, sess.AmmoRounds)
	}
	waitSessionWorker(t, scheduler, sess.ID)
}

func TestStartSessionRejectsPartialLoadoutWithoutWeapon(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-start-partial-loadout")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	if err := db.Model(&models.PlayerLoadout{}).Where("user_id = ?", models.DefaultUserID).
		Updates(map[string]interface{}{"weapon_id": "", "carried_ammo": "[]"}).Error; err != nil {
		t.Fatalf("构造部分空配装失败: %v", err)
	}
	scheduler := newStartedTestScheduler(t, db)
	service := NewSessionServiceWithScheduler(db, models.DefaultUserID, scheduler)
	if _, err := service.Start(StartReq{MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1}); err == nil {
		t.Fatal("仍穿着其他装备但缺少武器时不应允许启动")
	}
}

func TestStartSessionBuysPresetWhenCurrentLoadoutIsEmpty(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-start-empty-loadout")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	if err := db.Model(&models.PlayerLoadout{}).Where("user_id = ?", models.DefaultUserID).
		Updates(map[string]interface{}{
			"weapon_id": "", "armor_id": "", "chest_rig_id": "", "backpack_id": "", "helmet_id": "", "headset_id": "",
			"consumables": "[]", "consumable_refs": "[]", "carried_ammo": "[]",
		}).Error; err != nil {
		t.Fatalf("构造空当前配装失败: %v", err)
	}
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", models.DefaultUserID, "cash").
		Update("quantity", 10000).Error; err != nil {
		t.Fatalf("补充测试现金失败: %v", err)
	}

	scheduler := newStartedTestScheduler(t, db)
	service := NewSessionServiceWithScheduler(db, models.DefaultUserID, scheduler)
	sess, err := service.Start(StartReq{MapID: "city_ruins", Style: "balanced", RecoveryPreset: 2})
	if err != nil {
		t.Fatalf("空当前配装应按预设自动补购并启动: %v", err)
	}
	if sess.WeaponID != "pistol_glock" || sess.ArmorID != "light_02" || sess.AmmoID != "ammo_9x19_n1" {
		t.Fatalf("空当前配装恢复结果异常: weapon=%s armor=%s ammo=%s", sess.WeaponID, sess.ArmorID, sess.AmmoID)
	}
	waitSessionWorker(t, scheduler, sess.ID)
}

func TestArmorlessEngineStateIsNotTerminalByArmorDurability(t *testing.T) {
	state := engine.EngineState{Loadout: engine.LoadoutState{WeaponID: "weapon_test"}, ArmorDurability: 0}
	if engineStateArmorBroken(state) {
		t.Fatal("无甲状态的 0 耐久不应触发 armor_broken 终局条件")
	}
	state.Loadout.ArmorID = "armor_test"
	if !engineStateArmorBroken(state) {
		t.Fatal("已装备护甲且耐久为 0 时应触发 armor_broken 终局条件")
	}
}

func TestEngineStateArmorBrokenDuringRunDistinguishesInitialDamage(t *testing.T) {
	startBroken := engine.EngineState{Loadout: engine.LoadoutState{ArmorID: "armor_test"}, ArmorDurability: 0}
	if engineStateArmorBrokenDuringRun(startBroken, startBroken, engine.RunResult{}) {
		t.Fatal("开局已损坏护甲不应被判定为本局新损坏")
	}

	startHealthy := engine.EngineState{Loadout: engine.LoadoutState{ArmorID: "armor_test"}, ArmorDurability: 100}
	endBroken := startHealthy
	endBroken.ArmorDurability = 0
	if !engineStateArmorBrokenDuringRun(startHealthy, endBroken, engine.RunResult{}) {
		t.Fatal("正常护甲本局降为 0 耐久时应触发终局判定")
	}

	if !engineStateArmorBrokenDuringRun(startBroken, startBroken, engine.RunResult{ArmorBrokenDuringRun: true}) {
		t.Fatal("开局破损后修复再损坏时应触发终局判定")
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
