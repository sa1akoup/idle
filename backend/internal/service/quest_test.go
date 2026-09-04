package service

import (
	"testing"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newQuestTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:quest-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Character{}, &models.QuestDef{}, &models.UserQuest{},
		&models.Inventory{}, &models.ItemInstance{}, &models.ItemUseDef{},
		&models.MerchantDef{}, &models.UserMerchantState{}, &models.LootItemDef{},
		&models.WeaponDef{}, &models.AmmoDef{}, &models.ArmorDef{}, &models.ConsumableDef{},
		&models.ChestRigDef{}, &models.BackpackDef{}, &models.HelmetDef{}, &models.HeadsetDef{},
		&models.NodeDef{},
	); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("读取测试连接: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestQuestTurnInRequiresRaidExtract(t *testing.T) {
	db := newQuestTestDB(t)
	const userID uint = 41
	if err := db.Create(&models.Character{UserID: userID, Name: "合同角色", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MerchantDef{ID: "medical", Name: "医疗商人", Category: "medical", Open: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: "medical", Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{ID: "salewa", Name: "Salewa 急救包", Category: "medical", Price: 400, Weight: 2, Slots: 2, MerchantCategory: "medical"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.QuestDef{
		ID: "medical_shortage", MerchantID: "medical", Name: "药品短缺", ObjectiveType: models.QuestObjectiveExtractItem,
		ObjectiveJSON: `{"itemId":"salewa","quantity":1}`, RewardJSON: `{"cash":1000,"merchantId":"medical","reputation":4}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 0, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{
		UserID: userID, ItemID: "salewa", Name: "Salewa 急救包", Kind: "loot", Quantity: 2, Price: 400,
		RaidExtract: false, MerchantCategory: "medical",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := AcceptQuestForUser(db, userID, "medical_shortage"); err != nil {
		t.Fatalf("接取失败: %v", err)
	}
	if err := TurnInQuestForUser(db, userID, "medical_shortage"); err == nil {
		t.Fatal("商店购买的 Salewa 不应能上交")
	}
	if err := db.Create(&models.Inventory{
		UserID: userID, ItemID: "salewa", Name: "Salewa 急救包", Kind: "loot", Quantity: 1, Price: 400,
		RaidExtract: true, MerchantCategory: "medical",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := TurnInQuestForUser(db, userID, "medical_shortage"); err != nil {
		t.Fatalf("局内带出上交失败: %v", err)
	}
	var shop models.Inventory
	if err := db.Where("user_id = ? AND item_id = ? AND raid_extract = ?", userID, "salewa", false).First(&shop).Error; err != nil {
		t.Fatalf("商店库存应保留: %v", err)
	}
	if shop.Quantity != 2 {
		t.Fatalf("商店库存 = %d，期望仍为 2", shop.Quantity)
	}
	var raid models.Inventory
	if err := db.Where("user_id = ? AND item_id = ? AND raid_extract = ?", userID, "salewa", true).First(&raid).Error; err == nil {
		t.Fatalf("局内带出库存应被扣完，仍剩 %d", raid.Quantity)
	}
	var state models.UserQuest
	if err := db.Where("user_id = ? AND quest_id = ?", userID, "medical_shortage").First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.Status != models.QuestStatusCompleted {
		t.Fatalf("合同状态 = %s，期望 completed", state.Status)
	}
	var cash models.Inventory
	if err := db.Where("user_id = ? AND item_id = ?", userID, "cash").First(&cash).Error; err != nil {
		t.Fatal(err)
	}
	if cash.Quantity != 1000 {
		t.Fatalf("奖励现金 = %d，期望 1000", cash.Quantity)
	}
	views, err := ListQuestsForUser(db, userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range views {
		if view.ID == "medical_shortage" && (view.Current != 1 || view.Required != 1) {
			t.Fatalf("已完成上交合同进度 = %d/%d，期望 1/1", view.Current, view.Required)
		}
	}
}

func TestQuestTurnInConsumesRaidExtractInstance(t *testing.T) {
	db := newQuestTestDB(t)
	const userID uint = 43
	if err := db.Create(&models.Character{UserID: userID, Name: "实例合同", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MerchantDef{ID: "medical", Name: "医疗商人", Category: "medical", Open: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: "medical", Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{ID: "salewa", Name: "Salewa 急救包", Category: "medical", Price: 400, Weight: 2, Slots: 2, MerchantCategory: "medical"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.QuestDef{
		ID: "medical_shortage", MerchantID: "medical", Name: "药品短缺", ObjectiveType: models.QuestObjectiveExtractItem,
		ObjectiveJSON: `{"itemId":"salewa","quantity":1}`, RewardJSON: `{"cash":1000,"merchantId":"medical","reputation":4}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 0, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ItemInstance{
		UserID: userID, ItemID: "salewa", CurrentDurability: 400, MaxDurability: 400,
		Status: "normal", LocationType: "inventory", RaidExtract: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := AcceptQuestForUser(db, userID, "medical_shortage"); err != nil {
		t.Fatalf("接取失败: %v", err)
	}
	if err := TurnInQuestForUser(db, userID, "medical_shortage"); err != nil {
		t.Fatalf("局内带出实例上交失败: %v", err)
	}
	var left int64
	if err := db.Model(&models.ItemInstance{}).Where("user_id = ? AND item_id = ?", userID, "salewa").Count(&left).Error; err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("上交后仍剩 %d 个 Salewa 实例", left)
	}
}

func TestQuestProgressUsesSnapshotContract(t *testing.T) {
	db := newQuestTestDB(t)
	const userID uint = 42
	if err := db.Create(&models.Character{UserID: userID, Name: "进度角色", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.QuestDef{
		ID: "weapon_customs", MerchantID: "weapon", Name: "海关调查",
		ObjectiveType: models.QuestObjectiveVisitNode, ObjectiveJSON: `{"nodeId":"city_ruins_node_1","quantity":1}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Create(&models.UserQuest{
		UserID: userID, QuestID: "weapon_customs", Status: models.QuestStatusActive,
		ProgressJSON: `{"count":0}`, AcceptedAt: &now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	result := engine.RunResult{
		Result: "success",
		Trace: []engine.TraceEvent{
			{Type: engine.TraceNodeEntered, NodeID: "city_ruins_node_8"},
		},
	}
	liveSnapshot := engine.ScenarioSnapshot{}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return applyQuestProgressTx(tx, userID, "balanced", liveSnapshot, result)
	}); err != nil {
		t.Fatalf("无快照合同时结算: %v", err)
	}
	var state models.UserQuest
	if err := db.Where("user_id = ? AND quest_id = ?", userID, "weapon_customs").First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.Status == models.QuestStatusCompleted {
		t.Fatal("快照未冻结的合同不应因本局访问而完成")
	}
	frozen := engine.ScenarioSnapshot{
		Contracts: []engine.QuestContract{{
			QuestID: "weapon_customs", Type: models.QuestObjectiveVisitNode,
			NodeID: "city_ruins_node_8", Quantity: 1,
		}},
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return applyQuestProgressTx(tx, userID, "balanced", frozen, result)
	}); err != nil {
		t.Fatalf("快照合同时结算: %v", err)
	}
	if err := db.Where("user_id = ? AND quest_id = ?", userID, "weapon_customs").First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.Status != models.QuestStatusCompleted {
		t.Fatalf("合同状态 = %s，期望 completed", state.Status)
	}
}
