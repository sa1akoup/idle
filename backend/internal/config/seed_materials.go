package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedMaterials 真实 loot 的初始库存（探索掉落仍由 session 结算逻辑追加）。
func seedMaterials(db *gorm.DB) error {
	materials := []models.Inventory{
		{ItemID: "construction_tape", Name: "建筑测量卷尺", Kind: "loot", Category: "tool", Quantity: 2, Price: 80, Weight: 1, Slots: 1, RaidExtract: true, MerchantCategory: "mechanical"},
		{ItemID: "set_of_tools", Name: "工具套装", Kind: "loot", Category: "tool", Quantity: 1, Price: 350, Weight: 2, Slots: 2, RaidExtract: true, MerchantCategory: "mechanical"},
		{ItemID: "salewa", Name: "Salewa 急救包", Kind: "loot", Category: "medical", Quantity: 2, Price: 400, Weight: 2, Slots: 2, RaidExtract: true, MerchantCategory: "medical"},
		{ItemID: "iskra", Name: "Iskra 口粮", Kind: "loot", Category: "food", Quantity: 2, Price: 220, Weight: 2, Slots: 1, RaidExtract: true, MerchantCategory: "medical"},
	}
	for _, inv := range materials {
		if err := upsertInventory(db, inv); err != nil {
			return err
		}
	}
	return nil
}
