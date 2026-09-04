package service

import (
	"testing"
	"time"

	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReputationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:reputation-" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(
		&models.MerchantDef{}, &models.UserMerchantState{}, &models.Character{},
		&models.Inventory{}, &models.LootItemDef{}, &models.ItemUseDef{},
		&models.WeaponDef{}, &models.AmmoDef{}, &models.ArmorDef{}, &models.ConsumableDef{},
		&models.ChestRigDef{}, &models.BackpackDef{}, &models.HelmetDef{}, &models.HeadsetDef{},
		&models.FacilityJob{}, &models.HideoutFacility{}, &models.FacilityLevelDef{},
		&models.EconomicOperation{}, &models.Session{}, &models.PlayerLoadout{},
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

func TestSellReputationAndMechanicalCatalog(t *testing.T) {
	db := newReputationTestDB(t)
	const userID uint = 21
	if err := db.Create(&models.Character{UserID: userID, Name: "声望角色", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	merchant := models.MerchantDef{ID: "mechanical", Name: "机械商人", Category: "mechanical", Open: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: merchant.ID, Reputation: 0, Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{ID: "screwdriver", Name: "螺丝刀", Category: "tool", Price: 60, Weight: 1, Slots: 1, MerchantCategory: "mechanical"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{ID: "electric_drill", Name: "电钻", Category: "tool", Price: 420, Weight: 2, Slots: 2, MerchantCategory: "mechanical", RepRequirement: 20}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 0, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "screwdriver", Name: "螺丝刀", Kind: "loot", Quantity: 20, Price: 60, MerchantCategory: "mechanical"}).Error; err != nil {
		t.Fatal(err)
	}

	userMerchant, err := GetMerchantByIDForUser(db, userID, merchant.ID)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := MerchantCatalog(db, userMerchant)
	if err != nil {
		t.Fatal(err)
	}
	buyable := map[string]bool{}
	for _, item := range catalog {
		buyable[item.ID] = item.Buyable
	}
	if !buyable["screwdriver"] {
		t.Fatalf("LL1 应能购买螺丝刀")
	}
	if buyable["electric_drill"] {
		t.Fatalf("好感 0 不应能购买电钻")
	}

	if _, err := SellItemForUser(db, userID, merchant.ID, "screwdriver", 12); err != nil {
		t.Fatalf("出售失败: %v", err)
	}
	var state models.UserMerchantState
	if err := db.Where("user_id = ? AND merchant_id = ?", userID, merchant.ID).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.Reputation != 1 {
		t.Fatalf("出售好感 = %d，期望 1", state.Reputation)
	}
}

func TestSellReputationAmount(t *testing.T) {
	if sellReputationAmount(199) != 0 || sellReputationAmount(200) != 1 || sellReputationAmount(1000) != 2 {
		t.Fatalf("出售好感档位错误")
	}
}
