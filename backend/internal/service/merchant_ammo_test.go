// 商人弹药测试：验证目录只展示 N1-N4，并由后端强制执行等级与好感度限制。
package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMerchantAmmoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:merchant-ammo-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(
		&models.MerchantDef{}, &models.UserMerchantState{},
		&models.WeaponDef{}, &models.AmmoDef{}, &models.ArmorDef{}, &models.ConsumableDef{},
		&models.ChestRigDef{}, &models.BackpackDef{}, &models.HelmetDef{}, &models.HeadsetDef{},
		&models.Inventory{}, &models.PlayerLoadout{}, &models.Character{}, &models.ItemUseDef{}, &models.ItemInstance{}, &models.LootItemDef{}, &models.FacilityJob{},
		&models.FacilityDef{}, &models.FacilityLevelDef{}, &models.HideoutFacility{},
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

func TestMerchantAmmoCatalogAndPurchaseRules(t *testing.T) {
	db := newMerchantAmmoTestDB(t)
	const userID uint = 13
	merchant := models.MerchantDef{ID: "weapon", Name: "武器商人", Category: "weapon", Open: true}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Character{UserID: userID, Name: "测试角色"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: merchant.ID, Reputation: 0, Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	for level, requirement := range map[int]int{1: 0, 2: 0, 3: 15, 4: 30, 5: 0, 6: 0} {
		ammo := models.AmmoDef{
			ID: fmt.Sprintf("ammo_556x45_n%d", level), Name: fmt.Sprintf("N%d 弹药", level),
			CaliberID: "556x45", Level: level, Price: level, RoundsPerSlot: 999,
			MerchantCategory: merchant.Category, RepRequirement: requirement,
		}
		if err := db.Create(&ammo).Error; err != nil {
			t.Fatal(err)
		}
	}

	userMerchant, err := GetMerchantByIDForUser(db, userID, merchant.ID)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := MerchantCatalog(db, userMerchant)
	if err != nil {
		t.Fatalf("读取弹药目录: %v", err)
	}
	if len(catalog) != 6 {
		t.Fatalf("商人目录数量 = %d，期望包含 N1-N6 以提供出售价格", len(catalog))
	}
	buyableCount := 0
	for _, item := range catalog {
		if item.Buyable {
			buyableCount++
		}
		if item.ID == "ammo_556x45_n5" || item.ID == "ammo_556x45_n6" {
			if item.Buyable {
				t.Fatalf("高等级弹药不应允许购买: %+v", item)
			}
			if item.SellPrice <= 0 {
				t.Fatalf("高等级弹药应返回后端计算的出售价格: %+v", item)
			}
		}
	}
	if buyableCount != 4 {
		t.Fatalf("可购买弹药数量 = %d，期望 N1-N4", buyableCount)
	}

	n5 := catalogItem{ID: "ammo_556x45_n5", Kind: "ammo", AmmoLevel: 5, Price: 5, MerchantCategory: merchant.Category}
	if err := applyMerchantPriceForUser(db, userID, &n5); !errors.Is(err, ErrMerchantUnavailable) {
		t.Fatalf("N5 购买错误 = %v，期望 ErrMerchantUnavailable", err)
	}
	n3 := catalogItem{ID: "ammo_556x45_n3", Kind: "ammo", AmmoLevel: 3, Price: 4, MerchantCategory: merchant.Category, RepRequirement: 15}
	if err := applyMerchantPriceForUser(db, userID, &n3); !errors.Is(err, ErrMerchantUnavailable) {
		t.Fatalf("初始购买 N3 错误 = %v，期望 ErrMerchantUnavailable", err)
	}
	n2 := catalogItem{ID: "ammo_556x45_n2", Kind: "ammo", AmmoLevel: 2, Price: 2, MerchantCategory: merchant.Category}
	if err := applyMerchantPriceForUser(db, userID, &n2); err != nil {
		t.Fatalf("初始购买 N2: %v", err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 5000}).Error; err != nil {
		t.Fatal(err)
	}
	if err := PurchaseFromMerchantForUser(db, userID, merchant.ID, n2.ID, 999); err != nil {
		t.Fatalf("购买 999 发 N2 弹药: %v", err)
	}
	assertAmmoQuantity(t, db, userID, n2.ID, 999)
	assertAmmoQuantity(t, db, userID, "cash", 3002)
	if err := PurchaseFromMerchantForUser(db, userID, merchant.ID, n5.ID, 1); !errors.Is(err, ErrMerchantUnavailable) {
		t.Fatalf("直接购买 N5 错误 = %v，期望 ErrMerchantUnavailable", err)
	}
	if err := PurchaseFromMerchantForUser(db, userID, merchant.ID, n2.ID, 1000); err == nil {
		t.Fatal("单次购买 1000 发应被拒绝")
	}
}
