// 弹药持久化测试：验证按发预留、终态返还、重复返还幂等和弹药仓位折算。
package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAmmunitionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:ammunition-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("读取测试连接: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.Inventory{}, &models.AmmoDef{}, &models.PlayerLoadout{}); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	return db
}

func TestReserveAndReturnCarriedAmmo(t *testing.T) {
	db := newAmmunitionTestDB(t)
	const userID uint = 7
	ammo := engine.Ammo{
		ID: "ammo_556x45_n4", Name: "5.56×45mm N4级弹", CaliberID: "556x45", Level: 4,
		FleshDamageMultiplier: 1, ArmorDamageMultiplier: 1, Price: 8, RoundsPerSlot: 999, MerchantCategory: "weapon",
	}
	snapshot := engine.ScenarioSnapshot{
		Weapons: map[string]engine.Weapon{"rifle": {ID: "rifle", CaliberID: "556x45", AmmoPerRound: 3}},
		Ammos:   map[string]engine.Ammo{ammo.ID: ammo},
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: ammo.ID, Name: ammo.Name, Kind: "ammo", Quantity: 90}).Error; err != nil {
		t.Fatal(err)
	}

	carried, err := reserveCarriedAmmo(db, userID, snapshot, "rifle", ammo.ID, 60)
	if err != nil {
		t.Fatalf("预留弹药: %v", err)
	}
	if carried.Rounds != 60 || carried.PreferredID != ammo.ID || carried.PreferredLevel != 4 || carried.TargetRounds != 60 {
		t.Fatalf("Session 携弹目标异常: %+v", carried)
	}
	assertAmmoQuantity(t, db, userID, ammo.ID, 30)

	state := engine.EngineState{Ammo: carried}
	state.Ammo.Rounds = 42
	if err := returnCarriedAmmoTx(db, userID, snapshot, &state); err != nil {
		t.Fatalf("返还剩余弹药: %v", err)
	}
	assertAmmoQuantity(t, db, userID, ammo.ID, 72)
	if state.Ammo.ID != "" || state.Ammo.Rounds != 0 {
		t.Fatalf("返还后 Session 弹药未清空: %+v", state.Ammo)
	}
	if err := returnCarriedAmmoTx(db, userID, snapshot, &state); err != nil {
		t.Fatalf("重复返还应幂等: %v", err)
	}
	assertAmmoQuantity(t, db, userID, ammo.ID, 72)
}

func TestAmmoInventoryUsesPackedSlots(t *testing.T) {
	db := newAmmunitionTestDB(t)
	const userID uint = 9
	definition := models.AmmoDef{ID: "ammo_9x19_n3", Name: "9×19mm N3级弹", CaliberID: "9x19", Level: 3, FleshDamageMultiplier: 1.05, ArmorDamageMultiplier: 0.8, RoundsPerSlot: 999}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: definition.ID, Name: definition.Name, Kind: "ammo", Quantity: 1000}).Error; err != nil {
		t.Fatal(err)
	}
	second := models.AmmoDef{ID: "ammo_556x45_n3", Name: "5.56×45mm N3级弹", CaliberID: "556x45", Level: 3, FleshDamageMultiplier: 1.05, ArmorDamageMultiplier: 0.8, RoundsPerSlot: 999}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: second.ID, Name: second.Name, Kind: "ammo", Quantity: 1}).Error; err != nil {
		t.Fatal(err)
	}
	used, err := inventoryUsage(db, userID)
	if err != nil {
		t.Fatalf("计算弹药仓位: %v", err)
	}
	if used != 3 {
		t.Fatalf("1000 发第一种弹药加 1 发第二种弹药占用 = %d，期望 3 格", used)
	}
}

func TestRefillSessionAmmoDowngradesToHighestAvailableLevel(t *testing.T) {
	db := newAmmunitionTestDB(t)
	const userID uint = 11
	n4 := engine.Ammo{ID: "ammo_556x45_n4", Name: "5.56×45mm N4级弹", CaliberID: "556x45", Level: 4, Price: 8, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	n2 := engine.Ammo{ID: "ammo_556x45_n2", Name: "5.56×45mm N2级弹", CaliberID: "556x45", Level: 2, Price: 2, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	n1 := engine.Ammo{ID: "ammo_556x45_n1", Name: "5.56×45mm N1级弹", CaliberID: "556x45", Level: 1, Price: 1, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	snapshot := engine.ScenarioSnapshot{
		Weapons: map[string]engine.Weapon{"rifle": {ID: "rifle", CaliberID: "556x45", AmmoPerRound: 3}},
		Ammos:   map[string]engine.Ammo{n1.ID: n1, n2.ID: n2, n4.ID: n4},
		AmmoSupplies: map[string]engine.AmmoSupply{
			n1.ID: {AmmoID: n1.ID, CaliberID: n1.CaliberID, Level: 1, UnitPrice: 1, Available: true},
			n2.ID: {AmmoID: n2.ID, CaliberID: n2.CaliberID, Level: 2, UnitPrice: 2, Available: true},
			n4.ID: {AmmoID: n4.ID, CaliberID: n4.CaliberID, Level: 4, UnitPrice: 8, Available: false},
		},
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 1000}).Error; err != nil {
		t.Fatal(err)
	}
	state := engine.EngineState{Ammo: carriedAmmoWithPreference(n4, 2, n4, 120)}
	refill, err := refillSessionAmmoTx(db, userID, snapshot, "rifle", &state)
	if err != nil {
		t.Fatalf("自动降级补给: %v", err)
	}
	if state.Ammo.ID != n2.ID || state.Ammo.Level != 2 || state.Ammo.Rounds != 120 {
		t.Fatalf("自动补给弹药异常: %+v", state.Ammo)
	}
	if state.Ammo.PreferredID != n4.ID || state.Ammo.PreferredLevel != 4 || state.Ammo.TargetRounds != 120 {
		t.Fatalf("自动补给后丢失原始偏好: %+v", state.Ammo)
	}
	if refill.FromLevel != 4 || refill.ToLevel != 2 || refill.TotalPrice != 240 {
		t.Fatalf("自动补给结果异常: %+v", refill)
	}
	assertAmmoQuantity(t, db, userID, n4.ID, 2)
	assertAmmoQuantity(t, db, userID, "cash", 760)
}

func TestRefillSessionAmmoReturnsLeftoverWhenCashIsInsufficient(t *testing.T) {
	db := newAmmunitionTestDB(t)
	const userID uint = 12
	n4 := engine.Ammo{ID: "ammo_762x39_n4", Name: "7.62×39mm N4级弹", CaliberID: "762x39", Level: 4, Price: 8, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	n2 := engine.Ammo{ID: "ammo_762x39_n2", Name: "7.62×39mm N2级弹", CaliberID: "762x39", Level: 2, Price: 2, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	snapshot := engine.ScenarioSnapshot{
		Weapons: map[string]engine.Weapon{"rifle": {ID: "rifle", CaliberID: "762x39", AmmoPerRound: 3}},
		Ammos:   map[string]engine.Ammo{n2.ID: n2, n4.ID: n4},
		AmmoSupplies: map[string]engine.AmmoSupply{
			n2.ID: {AmmoID: n2.ID, CaliberID: n2.CaliberID, Level: 2, UnitPrice: 2, Available: true},
		},
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 100}).Error; err != nil {
		t.Fatal(err)
	}
	state := engine.EngineState{Ammo: carriedAmmoWithPreference(n4, 2, n4, 120)}
	_, err := refillSessionAmmoTx(db, userID, snapshot, "rifle", &state)
	if !errors.Is(err, ErrPurchaseUnavailable) {
		t.Fatalf("错误 = %v，期望 ErrPurchaseUnavailable", err)
	}
	if leftover, quantityErr := ammoInventoryQuantity(db, userID, n4.ID); quantityErr != nil || leftover != 0 {
		t.Fatalf("补给业务失败后剩余弹药数量 = %d，期望保持 0，错误=%v", leftover, quantityErr)
	}
	assertAmmoQuantity(t, db, userID, "cash", 100)
}

func TestReservePresetAmmoUsesPartialPreferredWarehouseStock(t *testing.T) {
	db := newAmmunitionTestDB(t)
	const userID uint = 14
	n4 := engine.Ammo{ID: "ammo_556x45_n4", Name: "5.56×45mm N4级弹", CaliberID: "556x45", Level: 4, Price: 8, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	n2 := engine.Ammo{ID: "ammo_556x45_n2", Name: "5.56×45mm N2级弹", CaliberID: "556x45", Level: 2, Price: 2, RoundsPerSlot: 999, MerchantCategory: "weapon"}
	snapshot := engine.ScenarioSnapshot{
		Weapons: map[string]engine.Weapon{"rifle": {ID: "rifle", CaliberID: "556x45", AmmoPerRound: 3}},
		Ammos:   map[string]engine.Ammo{n2.ID: n2, n4.ID: n4},
		AmmoSupplies: map[string]engine.AmmoSupply{
			n2.ID: {AmmoID: n2.ID, CaliberID: n2.CaliberID, Level: 2, UnitPrice: 2, Available: true},
		},
	}
	if err := db.Create(&models.Inventory{UserID: userID, ItemID: n4.ID, Name: n4.Name, Kind: "ammo", Quantity: 7}).Error; err != nil {
		t.Fatal(err)
	}
	carried, refill, err := reservePresetAmmoTx(db, userID, snapshot, engine.RecoveryPreset{
		Loadout: engine.LoadoutState{WeaponID: "rifle"}, AmmoID: n4.ID, AmmoRounds: 60,
	})
	if err != nil {
		t.Fatalf("预留恢复预设弹药: %v", err)
	}
	if carried.ID != n4.ID || carried.Rounds != 7 || carried.TargetRounds != 60 || refill == nil || refill.Source != "preset_warehouse" {
		t.Fatalf("恢复预设应先使用可完成攻击的高级弹药余量: carried=%+v refill=%+v", carried, refill)
	}
	assertAmmoQuantity(t, db, userID, n4.ID, 0)
}

func assertAmmoQuantity(t *testing.T, db *gorm.DB, userID uint, ammoID string, want int) {
	t.Helper()
	quantity, err := ammoInventoryQuantity(db, userID, ammoID)
	if err != nil {
		t.Fatal(err)
	}
	if quantity != want {
		t.Fatalf("弹药 %s 数量 = %d，期望 %d", ammoID, quantity, want)
	}
}
