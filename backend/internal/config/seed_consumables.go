package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedConsumables 消耗品 及其初始库存。
func seedConsumables(db *gorm.DB) error {
	consumables := []models.ConsumableDef{
		{ID: "smoke", Name: "烟雾弹", Desc: "首次脱离失败自动使用", Price: 200, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "flash", Name: "闪光弹", Desc: "近距首轮+先手", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "bandage", Name: "止血带", Desc: "止血防恶化", Price: 100, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "medkit", Name: "医疗包", Desc: "战后回血", Price: 250, Weight: 2, Slots: 2, MerchantCategory: "medical"},
		{ID: "toolkit", Name: "工具包", Desc: "工程事件必需", Price: 300, Weight: 3, Slots: 2, MerchantCategory: "mechanical"},
	}
	for _, c := range consumables {
		if err := upsertSeedDef(db, &c, c.ID); err != nil {
			return err
		}
	}

	for _, inv := range initialConsumableInventory() {
		if err := upsertInventory(db, inv); err != nil {
			return err
		}
	}
	return nil
}

func initialConsumableInventory() []models.Inventory {
	return []models.Inventory{
		{ItemID: "smoke", Name: "烟雾弹", Kind: "consumable", Quantity: 1, Price: 200, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ItemID: "toolkit", Name: "工具包", Kind: "consumable", Quantity: 1, Price: 300, Weight: 3, Slots: 2, MerchantCategory: "mechanical"},
	}
}
