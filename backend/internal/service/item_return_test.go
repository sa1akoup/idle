package service

import (
	"testing"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

func TestReturnedRaidExtractConsumableStaysFIR(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	if err := db.Create(&models.Inventory{
		UserID: models.DefaultUserID, ItemID: "iskra", Name: "Iskra 口粮", Kind: "loot",
		Category: "food", Quantity: 1, Price: 220, Weight: 2, Slots: 1,
		RaidExtract: true, MerchantCategory: "medical",
	}).Error; err != nil {
		t.Fatal(err)
	}
	items, err := carriedItemsForLoadout(db, models.DefaultUserID, &models.PlayerLoadout{Consumables: []string{"iskra"}})
	if err != nil {
		t.Fatalf("读取携带补给: %v", err)
	}
	if len(items) != 1 || items[0].ItemID != "iskra" || !items[0].RaidExtract || items[0].Quantity != 1 {
		t.Fatalf("携带物品未保留局内带出标记: %+v", items)
	}
	snapshot := engine.ScenarioSnapshot{
		Items: map[string]engine.ItemDefinition{
			"iskra": {ID: "iskra", Name: "Iskra 口粮", Kind: "loot", Category: "food", Price: 220, Weight: 2, Slots: 1, MerchantCategory: "medical"},
		},
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := reserveCarriedItemsTx(tx, models.DefaultUserID, items, "session-1"); err != nil {
			return err
		}
		return returnCarriedItemsTx(tx, models.DefaultUserID, snapshot, items)
	}); err != nil {
		t.Fatalf("归还携带补给: %v", err)
	}
	var raid models.Inventory
	if err := db.Where("user_id = ? AND item_id = ? AND raid_extract = ?", models.DefaultUserID, "iskra", true).First(&raid).Error; err != nil {
		t.Fatalf("撤离归还应写回局内带出库存: %v", err)
	}
	if raid.Quantity != 1 {
		t.Fatalf("局内带出口粮数量 = %d，期望 1", raid.Quantity)
	}
	var shop int64
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND raid_extract = ?", models.DefaultUserID, "iskra", false).
		Count(&shop).Error; err != nil {
		t.Fatal(err)
	}
	if shop != 0 {
		t.Fatalf("不应把局内带出补给写成商店库存，商店行数 = %d", shop)
	}
}

func TestReturnedShopConsumableStaysNonFIR(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	if err := db.Create(&models.Inventory{
		UserID: models.DefaultUserID, ItemID: "iskra", Name: "Iskra 口粮", Kind: "loot",
		Category: "food", Quantity: 1, Price: 220, Weight: 2, Slots: 1,
		RaidExtract: false, MerchantCategory: "medical",
	}).Error; err != nil {
		t.Fatal(err)
	}
	items, err := carriedItemsForLoadout(db, models.DefaultUserID, &models.PlayerLoadout{Consumables: []string{"iskra"}})
	if err != nil {
		t.Fatalf("读取携带补给: %v", err)
	}
	snapshot := engine.ScenarioSnapshot{
		Items: map[string]engine.ItemDefinition{
			"iskra": {ID: "iskra", Name: "Iskra 口粮", Kind: "loot", Category: "food", Price: 220, Weight: 2, Slots: 1, MerchantCategory: "medical"},
		},
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := reserveCarriedItemsTx(tx, models.DefaultUserID, items, "session-1"); err != nil {
			return err
		}
		return returnCarriedItemsTx(tx, models.DefaultUserID, snapshot, items)
	}); err != nil {
		t.Fatalf("归还商店补给: %v", err)
	}
	var shop models.Inventory
	if err := db.Where("user_id = ? AND item_id = ? AND raid_extract = ?", models.DefaultUserID, "iskra", false).First(&shop).Error; err != nil {
		t.Fatalf("商店补给归还应保持非局内带出: %v", err)
	}
	if shop.Quantity != 1 {
		t.Fatalf("商店口粮数量 = %d，期望 1", shop.Quantity)
	}
}
