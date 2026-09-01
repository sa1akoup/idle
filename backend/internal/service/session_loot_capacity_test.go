// Session 战利品与携行容量回归测试：锁定弹药组装箱、终局返还顺序和版本迁移收尾。
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

func TestFitEngineLootToStoragePacksIntoExistingPartialAmmoStack(t *testing.T) {
	db := newSessionEventsTestDB(t, "loot-capacity-partial-ammo")
	const userID uint = 31
	const ammoID = "ammo_test_partial"
	if err := db.Create(&models.AmmoDef{ID: ammoID, Name: "测试弹药", CaliberID: "9x19", Level: 1, RoundsPerSlot: 30}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "filler", Kind: "loot", Quantity: 478}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: ammoID, Name: "测试弹药", Kind: "ammo", Quantity: 15}).Error; err != nil {
		t.Fatal(err)
	}

	snapshot := engine.ScenarioSnapshot{
		Ammos:  map[string]engine.Ammo{ammoID: {ID: ammoID, Name: "测试弹药", RoundsPerSlot: 30}},
		Tuning: engine.DefaultTuning(),
	}
	stored, overflow, err := fitEngineLootToStorage(db, userID, snapshot, []engine.LootDrop{{ItemID: ammoID, Quantity: 30}})
	if err != nil {
		t.Fatalf("按已有部分弹药堆叠装箱失败: %v", err)
	}
	if len(stored) != 1 || stored[0].Quantity != 30 || len(overflow) != 0 {
		t.Fatalf("已有 15 发弹药时装箱结果异常: stored=%+v overflow=%+v", stored, overflow)
	}
}

func TestFitEngineLootToStorageLimitsAmmoByAvailableSlots(t *testing.T) {
	db := newSessionEventsTestDB(t, "loot-capacity-limit-ammo")
	const userID uint = 32
	const ammoID = "ammo_test_limit"
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "filler", Kind: "loot", Quantity: 478}).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := engine.ScenarioSnapshot{
		Ammos:  map[string]engine.Ammo{ammoID: {ID: ammoID, Name: "测试弹药", RoundsPerSlot: 30}},
		Tuning: engine.DefaultTuning(),
	}
	stored, overflow, err := fitEngineLootToStorage(db, userID, snapshot, []engine.LootDrop{{ItemID: ammoID, Quantity: 65}})
	if err != nil {
		t.Fatalf("按剩余仓位装箱失败: %v", err)
	}
	if len(stored) != 1 || stored[0].Quantity != 60 || len(overflow) != 1 || overflow[0].Quantity != 5 {
		t.Fatalf("2 格仓位装箱结果异常: stored=%+v overflow=%+v", stored, overflow)
	}
}

func TestFitEngineLootToStorageFillsExistingPartialAmmoStackWhenFull(t *testing.T) {
	db := newSessionEventsTestDB(t, "loot-capacity-fill-partial-ammo")
	const userID uint = 35
	const ammoID = "ammo_test_fill_partial"
	if err := db.Create(&models.AmmoDef{ID: ammoID, Name: "测试弹药", CaliberID: "9x19", Level: 1, RoundsPerSlot: 30}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "filler", Kind: "loot", Quantity: 499}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: ammoID, Name: "测试弹药", Kind: "ammo", Quantity: 15}).Error; err != nil {
		t.Fatal(err)
	}

	snapshot := engine.ScenarioSnapshot{
		Ammos:  map[string]engine.Ammo{ammoID: {ID: ammoID, Name: "测试弹药", RoundsPerSlot: 30}},
		Tuning: engine.DefaultTuning(),
	}
	stored, overflow, err := fitEngineLootToStorage(db, userID, snapshot, []engine.LootDrop{{ItemID: ammoID, Quantity: 15}})
	if err != nil {
		t.Fatalf("已有半堆弹药时装箱失败: %v", err)
	}
	if len(stored) != 1 || stored[0].Quantity != 15 || len(overflow) != 0 {
		t.Fatalf("仓库满但已有半堆时装箱结果异常: stored=%+v overflow=%+v", stored, overflow)
	}
}

func TestTerminalSettlementReturnsAmmoBeforeStoringLoot(t *testing.T) {
	db := newSessionEventsTestDB(t, "terminal-loot-order")
	const userID uint = 33
	character := models.Character{UserID: userID, Name: "终局测试角色", Strength: 50, HP: 100, Energy: 100, Hydration: 100}
	if err := db.Create(&character).Error; err != nil {
		t.Fatal(err)
	}
	const ammoID = "ammo_test_terminal"
	if err := db.Create(&models.AmmoDef{ID: ammoID, Name: "测试弹药", CaliberID: "9x19", Level: 1, RoundsPerSlot: 30}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "filler", Kind: "loot", Quantity: 479}).Error; err != nil {
		t.Fatal(err)
	}

	snapshot := engine.ScenarioSnapshot{
		Items: map[string]engine.ItemDefinition{
			"loot_test": {ID: "loot_test", Kind: "loot", Name: "测试战利品", Category: "tool", Slots: 1, Weight: 1},
		},
		LootItems: map[string]engine.LootItem{
			"loot_test": {ID: "loot_test", Name: "测试战利品", Category: "tool", Slots: 1, Weight: 1},
		},
		Ammos:  map[string]engine.Ammo{ammoID: {ID: ammoID, Name: "测试弹药", CaliberID: "9x19", Level: 1, RoundsPerSlot: 30}},
		Tuning: engine.DefaultTuning(),
	}
	state := engine.EngineState{
		Character: engine.CharacterState{Name: "终局测试角色", Strength: 50, HP: 100, Energy: 100, Hydration: 100},
		Ammo:      engine.CarriedAmmo{ID: ammoID, CaliberID: "9x19", Level: 1, Rounds: 30},
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	sess := models.Session{
		UserID: userID, CharacterID: character.ID, Status: "running", EngineVersion: engine.EngineVersion,
		RecoveryPolicyJSON: defaultRecoveryPolicyJSON(), StateJSON: string(stateJSON), StartTime: now,
	}
	if err := db.Create(&sess).Error; err != nil {
		t.Fatal(err)
	}

	result := engine.RunResult{
		Result: "success", Finished: true, DurationSec: 1,
		StartHP: 100, EndHP: 100, StartEnergy: 100, EndEnergy: 100, StartHydration: 100, EndHydration: 100,
		ExtractedLoot: []engine.LootDrop{{ItemID: "loot_test", Quantity: 1, Source: "容器"}},
		NextState:     state,
	}
	service := NewSessionService(db, userID)
	if err := service.settleEngineRun(&sess, &state, snapshot, result, 1, now, now.Add(time.Hour)); err != nil {
		t.Fatalf("终局返还与战利品结算失败: %v", err)
	}
	if state.Ammo.ID != "" || state.Ammo.Rounds != 0 {
		t.Fatalf("终局后携带弹药未清空: %+v", state.Ammo)
	}
	if quantity := inventoryQuantityForTest(t, db, userID, ammoID); quantity != 30 {
		t.Fatalf("终局返还弹药数量 = %d，期望 30", quantity)
	}
	if quantity := inventoryQuantityForTest(t, db, userID, "loot_test"); quantity != 0 {
		t.Fatalf("仓库已满时战利品不应入库，实际数量 = %d", quantity)
	}
	var run models.SessionRun
	if err := db.Where("user_id = ? AND session_id = ? AND run_index = ?", userID, sess.ID, 1).First(&run).Error; err != nil {
		t.Fatalf("读取终局行动记录: %v", err)
	}
	var overflow []map[string]interface{}
	if err := json.Unmarshal([]byte(run.OverflowLoot), &overflow); err != nil {
		t.Fatalf("解析溢出战利品: %v", err)
	}
	if len(overflow) != 1 || overflow[0]["itemId"] != "loot_test" || overflow[0]["quantity"] != float64(1) {
		t.Fatalf("终局战利品溢出记录异常: %+v", overflow)
	}
}

func TestBuildEngineStateRejectsAmmoOverCapacity(t *testing.T) {
	db := newSessionEventsTestDB(t, "engine-state-ammo-capacity")
	const userID uint = 34
	character := models.Character{UserID: userID, Name: "容量测试角色", Strength: 50, HP: 100, Energy: 100, Hydration: 100}
	if err := db.Create(&character).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.WeaponDef{ID: "weapon_test", Name: "测试步枪", Category: "rifle", CaliberID: "9x19", AmmoPerRound: 1, Slots: 1, Weight: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ArmorDef{ID: "armor_test", Name: "测试护甲", Slots: 1, Weight: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AmmoDef{ID: "ammo_test", Name: "测试弹药", CaliberID: "9x19", Level: 1, RoundsPerSlot: 30}).Error; err != nil {
		t.Fatal(err)
	}
	loadout := &models.PlayerLoadout{UserID: userID, CharacterID: character.ID, WeaponID: "weapon_test", ArmorID: "armor_test"}
	_, err := buildEngineState(db, userID, character, loadout, engine.CarriedAmmo{
		ID: "ammo_test", CaliberID: "9x19", Level: 1, Rounds: 9999,
	})
	if err == nil || !strings.Contains(err.Error(), "超过携行容量") {
		t.Fatalf("超容量弹药开局错误 = %v，期望拒绝", err)
	}
}

func TestLegacyV16SessionFinishesAndReturnsCarriedAssets(t *testing.T) {
	db := newSessionEventsTestDB(t, "legacy-v16-finish")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	initialAmmo, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_762x39_n4")
	if err != nil {
		t.Fatal(err)
	}
	scheduler := newStartedTestScheduler(t, db)
	service := NewSessionServiceWithScheduler(db, models.DefaultUserID, scheduler)
	sess, err := service.Start(StartReq{
		MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
		AmmoID: "ammo_762x39_n4", AmmoRounds: 60,
	})
	if err != nil {
		t.Fatalf("启动测试 Session: %v", err)
	}
	waitSessionWorker(t, scheduler, sess.ID)
	if result := db.Model(&models.Session{}).Where("id = ? AND user_id = ?", sess.ID, models.DefaultUserID).
		Update("engine_version", engine.LegacyEngineVersionV16); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("写入 v16 会话版本失败: error=%v rows=%d", result.Error, result.RowsAffected)
	}
	var legacySnapshot map[string]json.RawMessage
	if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &legacySnapshot); err != nil {
		t.Fatalf("解析 v17 场景快照: %v", err)
	}
	delete(legacySnapshot, "headsets")
	delete(legacySnapshot, "tuning")
	legacySnapshotJSON, err := json.Marshal(legacySnapshot)
	if err != nil {
		t.Fatalf("序列化 v16 场景快照: %v", err)
	}
	legacyDigest := sha256.Sum256(legacySnapshotJSON)
	legacyHash := hex.EncodeToString(legacyDigest[:])
	if result := db.Model(&models.Session{}).Where("id = ? AND user_id = ?", sess.ID, models.DefaultUserID).
		Updates(map[string]interface{}{"scenario_snapshot": string(legacySnapshotJSON), "scenario_hash": legacyHash}); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("写入 v16 旧格式场景快照失败: error=%v rows=%d", result.Error, result.RowsAffected)
	}

	legacyService := NewSessionService(db, models.DefaultUserID)
	if err := legacyService.simulateSession(sess.ID); err != nil {
		t.Fatalf("v16 Session 收尾失败: %v", err)
	}
	var finished models.Session
	if err := db.Where("id = ? AND user_id = ?", sess.ID, models.DefaultUserID).First(&finished).Error; err != nil {
		t.Fatal(err)
	}
	if finished.Status != "success" || finished.TerminalReason != "engine_migrated" || finished.AmmoID != "" || finished.AmmoRounds != 0 {
		t.Fatalf("v16 Session 收尾字段异常: status=%s reason=%s ammo=%s/%d", finished.Status, finished.TerminalReason, finished.AmmoID, finished.AmmoRounds)
	}
	if returned, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_762x39_n4"); err != nil || returned != initialAmmo {
		t.Fatalf("v16 Session 收尾后弹药 = %d，期望恢复为 %d，错误=%v", returned, initialAmmo, err)
	}
	var state engine.EngineState
	if err := json.Unmarshal([]byte(finished.StateJSON), &state); err != nil {
		t.Fatal(err)
	}
	if state.Ammo.ID != "" || state.Ammo.Rounds != 0 {
		t.Fatalf("v16 Session 持久化状态仍保留携弹: %+v", state.Ammo)
	}
	var recoveryPlans int64
	if err := db.Model(&models.RecoveryPlan{}).Where("user_id = ? AND source_session_id = ?", models.DefaultUserID, sess.ID).Count(&recoveryPlans).Error; err != nil {
		t.Fatal(err)
	}
	if recoveryPlans != 1 {
		t.Fatalf("v16 Session 恢复计划数 = %d，期望 1", recoveryPlans)
	}
}

func inventoryQuantityForTest(t *testing.T, db *gorm.DB, userID uint, itemID string) int {
	t.Helper()
	var quantity int
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", userID, itemID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&quantity).Error; err != nil {
		t.Fatalf("读取库存 %s: %v", itemID, err)
	}
	return quantity
}
