// 活跃行动资源保护测试：验证当前装备、携带弹药和行动护甲不能被并发交易或维修。
package service

import (
	"errors"
	"sync"
	"testing"

	"idle/internal/config"
	"idle/internal/models"

	"gorm.io/gorm"
)

func TestActiveSessionBlocksSellingCurrentLoadoutItem(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)
	if err := db.Create(&models.Session{
		UserID: models.DefaultUserID, WeaponID: "weapon", ArmorID: "armor", Status: "running",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := SellItem(db, "weapon", "weapon", 1); !errors.Is(err, ErrActiveSessionResourceLocked) {
		t.Fatalf("出售活跃行动武器错误 = %v，期望 ErrActiveSessionResourceLocked", err)
	}
}

func TestActiveSessionBlocksArmorRepair(t *testing.T) {
	db := newInventoryTestDB(t)
	armor := models.ArmorInstance{UserID: models.DefaultUserID, ArmorID: "armor", MaxDurability: 80, CurDurability: 0, Status: "broken"}
	if err := db.Create(&armor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Session{UserID: models.DefaultUserID, ArmorID: "armor", Status: "running"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.HideoutFacility{UserID: models.DefaultUserID, FacilityID: "workbench", Level: 1, State: "ready"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.FacilityLevelDef{FacilityID: "workbench", Level: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := QueueArmorRepairForUser(db, models.DefaultUserID, armor.ID); !errors.Is(err, ErrActiveSessionResourceLocked) {
		t.Fatalf("维修活跃行动护甲错误 = %v，期望 ErrActiveSessionResourceLocked", err)
	}
}

func TestActiveSessionBlocksLoadoutMutation(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)
	if err := db.Create(&models.Session{UserID: models.DefaultUserID, WeaponID: "weapon", ArmorID: "armor", Status: "running"}).Error; err != nil {
		t.Fatal(err)
	}
	loadout, err := GetPlayerLoadout(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = SavePlayerLoadout(db, SaveLoadoutReq{WeaponID: loadout.WeaponID, ArmorID: loadout.ArmorID, Consumables: loadout.Consumables})
	if !errors.Is(err, ErrActiveSessionResourceLocked) {
		t.Fatalf("修改活跃行动装备错误 = %v，期望 ErrActiveSessionResourceLocked", err)
	}
}

func TestConcurrentSellingLastItemOnlyOneSucceeds(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)
	if err := db.Create(&models.Inventory{UserID: models.DefaultUserID, ItemID: "weapon2", Name: "备用武器", Kind: "weapon", Quantity: 1, Price: 150, MerchantCategory: "weapon"}).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, sellErr := SellItem(db, "weapon", "weapon2", 1)
			results <- sellErr
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	var sellErrors []error
	for sellErr := range results {
		if sellErr == nil {
			successes++
		} else {
			sellErrors = append(sellErrors, sellErr)
		}
	}
	if successes != 1 {
		t.Fatalf("并发出售成功数 = %d，期望 1，错误=%v", successes, sellErrors)
	}
	var weaponQuantity int
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, "weapon2").
		Select("COALESCE(SUM(quantity), 0)").Scan(&weaponQuantity).Error; err != nil {
		t.Fatal(err)
	}
	if weaponQuantity != 0 {
		t.Fatalf("并发出售后的武器库存 = %d，期望 0", weaponQuantity)
	}
	assertInventoryQuantity(t, db, "cash", 1045)
}

func TestDifferentUsersCanSellConcurrently(t *testing.T) {
	db := newInventoryTestDB(t)
	seedTestLoadout(t, db, 1000)
	if err := db.Create(&models.Inventory{UserID: models.DefaultUserID, ItemID: "weapon2", Name: "备用武器", Kind: "weapon", Quantity: 1, Price: 150, MerchantCategory: "weapon"}).Error; err != nil {
		t.Fatal(err)
	}
	const secondUserID uint = 2
	if err := db.Create(&models.Character{UserID: secondUserID, Name: "第二用户"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, item := range []models.Inventory{
		{UserID: secondUserID, ItemID: "weapon2", Name: "备用武器", Kind: "weapon", Quantity: 1, Price: 150, MerchantCategory: "weapon"},
		{UserID: secondUserID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 1000, Price: 1},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, userID := range []uint{models.DefaultUserID, secondUserID} {
		wg.Add(1)
		go func(userID uint) {
			defer wg.Done()
			<-start
			_, sellErr := SellItemForUser(db, userID, "weapon", "weapon2", 1)
			results <- sellErr
		}(userID)
	}
	close(start)
	wg.Wait()
	close(results)
	for sellErr := range results {
		if sellErr != nil {
			t.Fatalf("不同用户并发出售失败: %v", sellErr)
		}
	}
	assertInventoryQuantityForUser(t, db, models.DefaultUserID, "cash", 1045)
	assertInventoryQuantityForUser(t, db, secondUserID, "cash", 1045)
}

func TestSessionStartAndSellCurrentWeaponAreSerialized(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-resource-race")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	start := make(chan struct{})
	var wg sync.WaitGroup
	var started *models.Session
	var startErr error
	var sellErr error
	service := NewSessionService(db, models.DefaultUserID)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		started, startErr = service.Start(StartReq{
			MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
			AmmoID: "ammo_762x39_n4", AmmoRounds: 60,
		})
	}()
	go func() {
		defer wg.Done()
		<-start
		_, sellErr = SellItem(db, "weapon", "rifle_ak", 1)
	}()
	close(start)
	wg.Wait()

	if (startErr == nil) == (sellErr == nil) {
		t.Fatalf("Session 启动与出售不应同时成功或同时失败: start=%v sell=%v", startErr, sellErr)
	}
	if startErr == nil {
		if !errors.Is(sellErr, ErrActiveSessionResourceLocked) {
			t.Fatalf("Session 启动成功后出售当前武器错误 = %v，期望资源锁错误", sellErr)
		}
		waitSessionWorker(t, started.ID)
		if err := service.failSession(started.ID, errors.New("test cleanup")); err != nil {
			t.Fatalf("清理并发测试 Session: %v", err)
		}
	} else if sellErr != nil {
		t.Fatalf("出售成功但 Session 启动失败时，出售错误不应存在: %v", sellErr)
	}
}

func TestSessionStartAndRepairCurrentArmorAreSerialized(t *testing.T) {
	db := newSessionEventsTestDB(t, "session-repair-race")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	if err := db.Model(&models.ArmorInstance{}).
		Where("user_id = ? AND armor_id = ?", models.DefaultUserID, "light_01").
		Updates(map[string]interface{}{"cur_durability": 0, "status": "broken"}).Error; err != nil {
		t.Fatal(err)
	}
	var armor models.ArmorInstance
	if err := db.Where("user_id = ? AND armor_id = ?", models.DefaultUserID, "light_01").First(&armor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ItemInstance{
		UserID: models.DefaultUserID, ItemID: "weapon_repair_kit_used",
		MaxDurability: 100, CurrentDurability: 100, Status: "normal", LocationType: "inventory",
	}).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	start := make(chan struct{})
	var wg sync.WaitGroup
	var started *models.Session
	var startErr error
	var repairErr error
	service := NewSessionService(db, models.DefaultUserID)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		started, startErr = service.Start(StartReq{
			MapID: "city_ruins", Style: "balanced", RecoveryPreset: 1,
			AmmoID: "ammo_762x39_n4", AmmoRounds: 60,
		})
	}()
	go func() {
		defer wg.Done()
		<-start
		repairErr = QueueArmorRepairForUser(db, models.DefaultUserID, armor.ID)
	}()
	close(start)
	wg.Wait()

	if repairErr != nil {
		t.Fatalf("并发维修当前护甲失败: %v", repairErr)
	}
	if startErr == nil {
		waitSessionWorker(t, started.ID)
		if err := service.failSession(started.ID, errors.New("test cleanup")); err != nil {
			t.Fatalf("清理并发测试 Session: %v", err)
		}
	}
}

func assertInventoryQuantityForUser(t *testing.T, db *gorm.DB, userID uint, itemID string, want int) {
	t.Helper()
	var quantity int
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&quantity).Error; err != nil {
		t.Fatalf("读取用户 %d 的库存 %s: %v", userID, itemID, err)
	}
	if quantity != want {
		t.Fatalf("用户 %d 的库存 %s = %d，期望 %d", userID, itemID, quantity, want)
	}
}
