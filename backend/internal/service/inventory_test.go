package service

// 仓库交易测试：覆盖商人购买、失能丢装、预设补购成功与资金不足回滚。

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"idle/internal/models"
	"idle/internal/repository/database"

	"gorm.io/gorm"
)

func newInventoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-inventory-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("读取测试连接: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
	})

	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	if err := db.Create(&models.Character{UserID: models.DefaultUserID, Name: "测试角色"}).Error; err != nil {
		t.Fatalf("创建测试角色: %v", err)
	}
	for _, value := range []interface{}{
		&models.WeaponDef{ID: "weapon", Name: "测试步枪", Price: 100, MerchantCategory: "weapon"},
		&models.WeaponDef{ID: "weapon2", Name: "测试冲锋枪", Price: 150, MerchantCategory: "weapon"},
		&models.ArmorDef{ID: "armor", Name: "测试护甲", Price: 200, MaxDurability: 80, MerchantCategory: "clothing"},
		&models.ConsumableDef{ID: "smoke", Name: "测试烟雾弹", Price: 50, MerchantCategory: "medical"},
		&models.MerchantDef{ID: "weapon", Name: "武器商人", Category: "weapon", Open: true},
		&models.MerchantDef{ID: "clothing", Name: "服装商人", Category: "clothing", Open: true},
		&models.MerchantDef{ID: "medical", Name: "医疗商人", Category: "medical", Open: true},
	} {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("创建测试商品: %v", err)
		}
	}
	return db
}

func TestPurchaseItemDeductsCashAndAddsInventory(t *testing.T) {
	db := newInventoryTestDB(t)
	if err := db.Create(&models.Inventory{UserID: models.DefaultUserID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 1000, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}

	if err := PurchaseItemForUser(db, models.DefaultUserID, "weapon", 2); err != nil {
		t.Fatalf("购买武器: %v", err)
	}
	if err := PurchaseItemForUser(db, models.DefaultUserID, "armor", 1); err != nil {
		t.Fatalf("购买护甲: %v", err)
	}

	assertInventoryQuantity(t, db, models.DefaultUserID, "cash", 600)
	assertInventoryQuantity(t, db, models.DefaultUserID, "weapon", 2)
	assertInventoryQuantity(t, db, models.DefaultUserID, "armor", 1)
	var armorCount int64
	if err := db.Model(&models.ArmorInstance{}).Where("user_id = ?", models.DefaultUserID).Count(&armorCount).Error; err != nil {
		t.Fatal(err)
	}
	if armorCount != 1 {
		t.Fatalf("护甲实例数量 = %d，期望 1", armorCount)
	}
}

func TestReplaceLostLoadoutPurchasesPreset(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)

	paid, err := ReplaceLostLoadoutForUser(db, models.DefaultUserID, 1)
	if err != nil {
		t.Fatalf("自动补购: %v", err)
	}
	if paid != 350 {
		t.Fatalf("补购金额 = %d，期望 350", paid)
	}
	assertInventoryQuantity(t, db, models.DefaultUserID, "cash", 650)
	assertInventoryQuantity(t, db, models.DefaultUserID, "weapon", 1)
	assertInventoryQuantity(t, db, models.DefaultUserID, "armor", 1)
	assertInventoryQuantity(t, db, models.DefaultUserID, "smoke", 1)

	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if loadout.WeaponID != "weapon" || loadout.ArmorID != "armor" || len(loadout.Consumables) != 1 {
		t.Fatalf("补购后的装备配置异常: %+v", loadout)
	}
}

func TestReplaceLostLoadoutUsesChosenPreset(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)
	if err := db.Model(&models.PlayerLoadout{}).Where("id = ?", models.PlayerLoadoutID).Updates(map[string]interface{}{
		"preset2_weapon_id": "weapon2", "preset2_armor_id": "armor", "preset2_consumables": `["smoke"]`,
	}).Error; err != nil {
		t.Fatal(err)
	}

	paid, err := ReplaceLostLoadoutForUser(db, models.DefaultUserID, 2)
	if err != nil {
		t.Fatalf("按预设2补购: %v", err)
	}
	if paid != 400 {
		t.Fatalf("补购金额 = %d，期望 400（冲锋枪150+护甲200+烟雾弹50）", paid)
	}
	assertInventoryQuantity(t, db, models.DefaultUserID, "weapon2", 1)
	var lostCount int64
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", models.DefaultUserID, "weapon").Count(&lostCount).Error; err != nil {
		t.Fatal(err)
	}
	if lostCount != 0 {
		t.Fatalf("丢失武器记录仍在仓库中")
	}
	assertInventoryQuantity(t, db, models.DefaultUserID, "smoke", 1)

	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if loadout.WeaponID != "weapon2" || loadout.ArmorID != "armor" || len(loadout.Consumables) != 1 {
		t.Fatalf("按预设2补购后的装备配置异常: %+v", loadout)
	}
}

func TestInventoryUsageExcludesLoadoutAllocation(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)
	// 当前装备与预设1均占用 weapon/armor/smoke 各1，库存各1时全部扣除 -> 占用0
	// 多买备用武器与烟雾弹，超出装备占用的部分计入容量
	if err := PurchaseItemForUser(db, models.DefaultUserID, "weapon", 2); err != nil {
		t.Fatalf("购买备用武器: %v", err)
	}
	if err := PurchaseItemForUser(db, models.DefaultUserID, "smoke", 2); err != nil {
		t.Fatalf("购买备用烟雾弹: %v", err)
	}
	used, err := inventoryUsage(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("计算仓库容量: %v", err)
	}
	// weapon: 3 扣 2 -> 1；armor: 1 扣 2 -> 0；smoke: 3 扣 2 -> 1
	if used != 2 {
		t.Fatalf("仓库占用 = %d，期望 2", used)
	}
}

func TestReplaceLostLoadoutKeepsLossWhenCashIsInsufficient(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 100)

	_, err := ReplaceLostLoadoutForUser(db, models.DefaultUserID, 1)
	if !errors.Is(err, ErrPurchaseUnavailable) {
		t.Fatalf("错误 = %v，期望 ErrPurchaseUnavailable", err)
	}
	assertInventoryQuantity(t, db, models.DefaultUserID, "cash", 100)
	for _, itemID := range []string{"weapon", "armor", "smoke"} {
		var count int64
		if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", models.DefaultUserID, itemID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("丢失物品 %s 仍在仓库中", itemID)
		}
	}

	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if loadout.WeaponID != "" || loadout.ArmorID != "" || len(loadout.Consumables) != 0 {
		t.Fatalf("补购失败后当前装备未清空: %+v", loadout)
	}
	if loadout.PresetWeaponID != "weapon" || loadout.PresetArmorID != "armor" {
		t.Fatalf("补购失败后预设被改动: %+v", loadout)
	}
}

func seedTestLoadout(t *testing.T, db *gorm.DB, cash int) {
	t.Helper()
	values := []interface{}{
		&models.Inventory{UserID: models.DefaultUserID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: cash, Price: 1},
		&models.Inventory{UserID: models.DefaultUserID, ItemID: "weapon", Name: "测试步枪", Kind: "weapon", Quantity: 1, Price: 100},
		&models.Inventory{UserID: models.DefaultUserID, ItemID: "armor", Name: "测试护甲", Kind: "armor", Quantity: 1, Price: 200},
		&models.Inventory{UserID: models.DefaultUserID, ItemID: "smoke", Name: "测试烟雾弹", Kind: "consumable", Quantity: 1, Price: 50},
		&models.ArmorInstance{UserID: models.DefaultUserID, ArmorID: "armor", MaxDurability: 80, CurDurability: 80, Status: "normal"},
		&models.PlayerLoadout{
			UserID: models.DefaultUserID,
			ID:     models.PlayerLoadoutID, CharacterID: models.PlayerCharacterID,
			WeaponID: "weapon", ArmorID: "armor", Consumables: []string{"smoke"},
			PresetWeaponID: "weapon", PresetArmorID: "armor", PresetConsumables: []string{"smoke"},
		},
	}
	for _, value := range values {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("创建装备测试数据: %v", err)
		}
	}
}

func assertInventoryQuantity(t *testing.T, db *gorm.DB, userID uint, itemID string, want int) {
	t.Helper()
	var inventory models.Inventory
	if err := db.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inventory).Error; err != nil {
		t.Fatalf("读取库存 %s: %v", itemID, err)
	}
	if inventory.Quantity != want {
		t.Fatalf("库存 %s 数量 = %d，期望 %d", itemID, inventory.Quantity, want)
	}
}
