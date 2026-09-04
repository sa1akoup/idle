package service

import (
	"math/rand"
	"testing"
	"time"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBlackMarketTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:black-market-" + t.Name() + "?mode=memory&cache=shared"
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
		&models.UserBlackMarketOffer{}, &models.ItemInstance{},
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

func TestBlackMarketOfferWeightFollowsDropWeight(t *testing.T) {
	common := catalog.Item{ID: "bolts", Kind: "loot", Price: 35, DropWeight: 40}
	legend := catalog.Item{ID: "ledx", Kind: "loot", Price: 85000, DropWeight: 1}
	if blackMarketOfferWeight(common) != 40 || blackMarketOfferWeight(legend) != 1 {
		t.Fatalf("战利品权重应等于 DropWeight")
	}
	counts := map[string]int{}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		picks := pickBlackMarketOffers(rng, []catalog.Item{common, legend}, 1)
		if len(picks) != 1 {
			t.Fatalf("抽选数量 = %d", len(picks))
		}
		counts[picks[0].Item.ID]++
	}
	if counts["ledx"] >= counts["bolts"] {
		t.Fatalf("传奇不应比常见件更容易上架: %+v", counts)
	}
}

func TestBlackMarketCycleStartAlignsToSixHours(t *testing.T) {
	now := time.Date(2026, 9, 4, 14, 22, 0, 0, time.UTC)
	got := blackMarketCycleStart(now)
	want := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("周期起点 = %s，期望 %s", got, want)
	}
}

func TestBlackMarketSellsAnyItemCheaperThanSpecialist(t *testing.T) {
	db := newBlackMarketTestDB(t)
	const userID uint = 31
	if err := db.Create(&models.Character{UserID: userID, Name: "黑市角色", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MerchantDef{ID: "weapon", Name: "武器商人", Category: "weapon", Open: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MerchantDef{ID: "black", Name: "黑市商人", Category: "black", Open: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: "weapon", Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: "black", Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{
		ID: "screwdriver", Name: "螺丝刀", Category: "tool", Price: 60, Weight: 1, Slots: 1,
		MerchantCategory: "mechanical", DropWeight: 40,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{
		ID: "ledx", Name: "LEDX", Category: "medical", Price: 85000, Weight: 1, Slots: 1,
		MerchantCategory: "medical", DropWeight: 1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 0, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "screwdriver", Name: "螺丝刀", Kind: "loot", Quantity: 2, Price: 60, MerchantCategory: "mechanical"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "ledx", Name: "LEDX", Kind: "loot", Quantity: 1, Price: 85000, MerchantCategory: "medical", RaidExtract: true}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := SellItemForUser(db, userID, "weapon", "screwdriver", 1); err == nil {
		t.Fatal("武器商人不应收购螺丝")
	}
	blackTotal, err := SellItemForUser(db, userID, "black", "screwdriver", 1)
	if err != nil {
		t.Fatalf("黑市收购螺丝: %v", err)
	}
	specialistPrice := roundPrice(60, sellMultiplier(0))
	if blackTotal >= specialistPrice {
		t.Fatalf("黑市收购价 %d 应低于专精 %d", blackTotal, specialistPrice)
	}
	ledxTotal, err := SellItemForUser(db, userID, "black", "ledx", 1)
	if err != nil {
		t.Fatalf("黑市收购 LEDX: %v", err)
	}
	if ledxTotal != roundPrice(85000, playerSellMultiplier(&models.MerchantDef{ID: "black"})) {
		t.Fatalf("LEDX 黑市价 = %d", ledxTotal)
	}
}

func TestBlackMarketCatalogRefreshAndPurchase(t *testing.T) {
	db := newBlackMarketTestDB(t)
	const userID uint = 32
	if err := db.Create(&models.Character{UserID: userID, Name: "货架角色", ResourceVersion: 1, NeedsUpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MerchantDef{ID: "black", Name: "黑市商人", Category: "black", Open: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserMerchantState{UserID: userID, MerchantID: "black", Unlocked: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{
		ID: "bolts", Name: "螺栓", Category: "tool", Price: 35, Weight: 1, Slots: 1,
		MerchantCategory: "mechanical", DropWeight: 40,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.LootItemDef{
		ID: "ledx", Name: "LEDX", Category: "medical", Price: 85000, Weight: 1, Slots: 1,
		MerchantCategory: "medical", DropWeight: 1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 200000, Price: 1}).Error; err != nil {
		t.Fatal(err)
	}

	merchant, err := GetMerchantByIDForUser(db, userID, "black")
	if err != nil {
		t.Fatal(err)
	}
	first, err := MerchantCatalog(db, userID, merchant)
	if err != nil {
		t.Fatalf("读取黑市货架: %v", err)
	}
	if !first.AcceptsAny || first.NextRefreshAt == nil || len(first.Items) == 0 {
		t.Fatalf("黑市货架不完整: %+v", first)
	}
	for _, item := range first.Items {
		if item.Price != roundPrice(item.BasePrice, 2*buyMultiplier(0)) {
			t.Fatalf("%s 售价 %d 应为翻倍后的好感价", item.ID, item.Price)
		}
		if item.Stock <= 0 || !item.Buyable {
			t.Fatalf("%s 应可购买且有库存", item.ID)
		}
	}

	target := first.Items[0]
	if err := PurchaseFromMerchantForUser(db, userID, "black", target.ID, 1); err != nil {
		t.Fatalf("购买黑市商品: %v", err)
	}
	second, err := MerchantCatalog(db, userID, merchant)
	if err != nil {
		t.Fatal(err)
	}
	if second.NextRefreshAt == nil || !second.NextRefreshAt.Equal(*first.NextRefreshAt) {
		t.Fatal("同一周期不应刷新货架时间")
	}
	found := false
	for _, item := range second.Items {
		if item.ID != target.ID {
			continue
		}
		found = true
		if item.Stock != target.Stock-1 && !(target.Stock == 1 && item.Stock == 0) {
			if target.Stock == 1 {
				t.Fatalf("买完 1 件后仍留在货架: %+v", item)
			}
			t.Fatalf("库存未扣减: 买前 %d 买后 %d", target.Stock, item.Stock)
		}
	}
	if target.Stock > 1 && !found {
		t.Fatalf("未买完的商品不应从货架消失")
	}

	stale := blackMarketCycleStart(time.Now()).Add(-blackMarketRefreshInterval)
	if err := db.Model(&models.UserBlackMarketOffer{}).Where("user_id = ?", userID).Update("cycle_start", stale).Error; err != nil {
		t.Fatal(err)
	}
	third, err := MerchantCatalog(db, userID, merchant)
	if err != nil {
		t.Fatal(err)
	}
	if third.NextRefreshAt == nil || !third.NextRefreshAt.Equal(*first.NextRefreshAt) {
		t.Fatal("仍在同一自然周期时，下次刷新时间应保持不变")
	}
	restored := 0
	for _, item := range third.Items {
		if item.ID == target.ID {
			restored = item.Stock
		}
	}
	if restored != target.Stock {
		t.Fatalf("过期周期应重抽货架，%s 库存 %d 期望回到 %d", target.ID, restored, target.Stock)
	}
}
