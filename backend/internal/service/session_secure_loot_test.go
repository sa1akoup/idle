package service

import (
	"testing"

	"idle/internal/engine"
	"idle/internal/models"
)

func TestStoreSecureLootKeepsBestFitWithoutFIR(t *testing.T) {
	db := newSessionEventsTestDB(t, "secure-loot-incap")
	const userID uint = 41
	if err := db.Create(&models.Character{UserID: userID, Name: "安全箱角色"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SecureContainerDef{ID: "secure_01", Name: "简易安全袋", InnerSlots: 1, Weight: 1, Slots: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PlayerLoadout{UserID: userID, CharacterID: 1, SecureContainerID: "secure_01"}).Error; err != nil {
		t.Fatal(err)
	}

	snapshot := engine.ScenarioSnapshot{
		LootItems: map[string]engine.LootItem{
			"bitcoin": {ID: "bitcoin", Name: "比特币", Price: 10000, Weight: 1, Slots: 1, DropWeight: 1},
			"lion":    {ID: "lion", Name: "青铜狮", Price: 2700, Weight: 3, Slots: 2, DropWeight: 2},
		},
		Items: map[string]engine.ItemDefinition{
			"bitcoin": {ID: "bitcoin", Kind: "loot", Name: "比特币", Price: 10000, Weight: 1, Slots: 1},
			"lion":    {ID: "lion", Kind: "loot", Name: "青铜狮", Price: 2700, Weight: 3, Slots: 2},
		},
	}
	svc := &SessionService{db: db, userID: userID}
	stored, overflow, err := svc.storeSecureLootTx(db, snapshot, []engine.LootDrop{
		{ItemID: "lion", Quantity: 1},
		{ItemID: "bitcoin", Quantity: 1},
	})
	if err != nil {
		t.Fatalf("失能安全箱保物失败: %v", err)
	}
	if len(stored) != 1 || stored[0].ItemID != "bitcoin" || stored[0].Quantity != 1 || len(overflow) != 0 {
		t.Fatalf("1 格口袋应保住比特币，实际 stored=%+v overflow=%+v", stored, overflow)
	}
	var bitcoin models.Inventory
	if err := db.Where("user_id = ? AND item_id = ?", userID, "bitcoin").First(&bitcoin).Error; err != nil {
		t.Fatalf("比特币应入库: %v", err)
	}
	if bitcoin.RaidExtract {
		t.Fatal("失能保住的搜刮不应带出局内带出标志")
	}
	var lion models.Inventory
	if err := db.Where("user_id = ? AND item_id = ?", userID, "lion").First(&lion).Error; err == nil {
		t.Fatal("2 格青铜狮不应被 1 格口袋保住")
	}
}
