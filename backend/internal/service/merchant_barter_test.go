package service

import (
	"errors"
	"testing"
	"time"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newKeyCaseBarterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:keycase-barter-" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(
		&models.MerchantDef{}, &models.UserMerchantState{}, &models.Character{},
		&models.Inventory{}, &models.LootItemDef{}, &models.ItemUseDef{},
		&models.WeaponDef{}, &models.AmmoDef{}, &models.ArmorDef{}, &models.ConsumableDef{},
		&models.ChestRigDef{}, &models.BackpackDef{}, &models.HelmetDef{}, &models.HeadsetDef{},
		&models.KeyCaseDef{}, &models.FacilityJob{}, &models.HideoutFacility{}, &models.FacilityLevelDef{},
		&models.EconomicOperation{}, &models.Session{}, &models.PlayerLoadout{},
		&models.ItemInstance{},
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

func seedKeyCaseBarterFixtures(t *testing.T, db *gorm.DB, userID uint, reputation int) {
	t.Helper()
	if err := db.Create(&models.Character{UserID: userID, Name: "兑换角色", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MerchantDef{ID: "clothing", Name: "服装商人", Category: "clothing", Open: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: "clothing", Reputation: reputation, Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.KeyCaseDef{ID: "keycase_09", Name: "文件钥匙包", KeySlots: 9, Price: 1800, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 15}).Error; err != nil {
		t.Fatal(err)
	}
	for _, loot := range []models.LootItemDef{
		{ID: "hydrogen_peroxide", Name: "过氧化氢", Category: "medical", Price: 100, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "saline_solution", Name: "生理盐水", Category: "medical", Price: 170, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "cat_figurine", Name: "猫雕像", Category: "valuable", Price: 900, Weight: 1, Slots: 1, MerchantCategory: "black"},
	} {
		if err := db.Create(&loot).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 99999, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestKeyCaseBarterRequiresRaidExtractAndReputation(t *testing.T) {
	db := newKeyCaseBarterTestDB(t)
	const userID uint = 31
	seedKeyCaseBarterFixtures(t, db, userID, 15)

	if err := PurchaseFromMerchantForUser(db, userID, "clothing", "keycase_09", 1); err == nil {
		t.Fatal("没有局内带出材料时应拒绝兑换")
	}

	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "hydrogen_peroxide", Name: "过氧化氢", Kind: "loot", Quantity: 3, Price: 100, MerchantCategory: "medical", RaidExtract: false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "saline_solution", Name: "生理盐水", Kind: "loot", Quantity: 3, Price: 170, MerchantCategory: "medical", RaidExtract: false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cat_figurine", Name: "猫雕像", Kind: "loot", Quantity: 1, Price: 900, MerchantCategory: "black", RaidExtract: false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := PurchaseFromMerchantForUser(db, userID, "clothing", "keycase_09", 1); err == nil {
		t.Fatal("商店货不应能兑换钥匙包")
	}

	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", userID, "hydrogen_peroxide").Update("raid_extract", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", userID, "saline_solution").Update("raid_extract", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", userID, "cat_figurine").Update("raid_extract", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := PurchaseFromMerchantForUser(db, userID, "clothing", "keycase_09", 1); err != nil {
		t.Fatalf("局内带出材料兑换失败: %v", err)
	}

	var got models.Inventory
	if err := db.Where("user_id = ? AND item_id = ?", userID, "keycase_09").First(&got).Error; err != nil {
		t.Fatalf("应获得文件钥匙包: %v", err)
	}
	var cash models.Inventory
	if err := db.Where("user_id = ? AND item_id = ?", userID, "cash").First(&cash).Error; err != nil {
		t.Fatal(err)
	}
	if cash.Quantity != 99999 {
		t.Fatalf("兑换不应扣现金，剩余 %d", cash.Quantity)
	}
	var peroxide models.Inventory
	if err := db.Where("user_id = ? AND item_id = ? AND raid_extract = ?", userID, "hydrogen_peroxide", true).First(&peroxide).Error; err == nil && peroxide.Quantity > 0 {
		t.Fatalf("过氧化氢应被扣完，剩余 %d", peroxide.Quantity)
	}
}

func TestKeyCaseBarterBlockedByReputation(t *testing.T) {
	db := newKeyCaseBarterTestDB(t)
	const userID uint = 32
	seedKeyCaseBarterFixtures(t, db, userID, 0)
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "hydrogen_peroxide", Name: "过氧化氢", Kind: "loot", Quantity: 3, Price: 100, RaidExtract: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "saline_solution", Name: "生理盐水", Kind: "loot", Quantity: 3, Price: 170, RaidExtract: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cat_figurine", Name: "猫雕像", Kind: "loot", Quantity: 1, Price: 900, RaidExtract: true}).Error; err != nil {
		t.Fatal(err)
	}
	err := PurchaseFromMerchantForUser(db, userID, "clothing", "keycase_09", 1)
	if err == nil || !errors.Is(err, ErrMerchantUnavailable) {
		t.Fatalf("好感不足应拒绝，实际 %v", err)
	}
}

func TestKeyCaseBarterCatalogAndBlackMarketWeight(t *testing.T) {
	db := newKeyCaseBarterTestDB(t)
	const userID uint = 33
	seedKeyCaseBarterFixtures(t, db, userID, 15)
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "hydrogen_peroxide", Name: "过氧化氢", Kind: "loot", Quantity: 2, Price: 100, RaidExtract: true}).Error; err != nil {
		t.Fatal(err)
	}

	merchant, err := GetMerchantByIDForUser(db, userID, "clothing")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MerchantCatalog(db, userID, merchant)
	if err != nil {
		t.Fatal(err)
	}
	var found *MerchantCatalogItem
	for i := range result.Items {
		if result.Items[i].ID == "keycase_09" {
			found = &result.Items[i]
			break
		}
	}
	if found == nil {
		t.Fatal("目录应展示文件钥匙包")
	}
	if found.Price != 0 || len(found.BarterCosts) != 3 {
		t.Fatalf("文件钥匙包应为兑换商品: price=%d costs=%d", found.Price, len(found.BarterCosts))
	}
	haveByID := map[string]int{}
	for _, cost := range found.BarterCosts {
		haveByID[cost.ItemID] = cost.Have
	}
	if haveByID["hydrogen_peroxide"] != 2 || haveByID["cat_figurine"] != 0 {
		t.Fatalf("兑换材料持有量错误: %+v", haveByID)
	}

	item := catalog.Item{ID: "keycase_09", Kind: "keycase", Price: 1800}
	if blackMarketOfferWeight(item) != 0 {
		t.Fatal("兑换钥匙包不应进入黑市现金货架")
	}
	if blackMarketOfferWeight(catalog.Item{ID: "secure_04", Kind: "secure", Price: 1800}) != 0 {
		t.Fatal("兑换安全箱不应进入黑市现金货架")
	}
}
