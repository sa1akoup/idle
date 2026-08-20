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
		{ID: "ammo_box", Name: "弹药箱", Desc: "补充弹药", Price: 150, Weight: 2, Slots: 2, MerchantCategory: "weapon"},
	}
	for _, c := range consumables {
		if err := db.FirstOrCreate(&c, models.ConsumableDef{ID: c.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.ConsumableDef{}).Where("id=?", c.ID).Updates(c).Error; err != nil {
			return err
		}
	}

	owned := []models.Inventory{
		{ItemID: "smoke", Name: "烟雾弹", Kind: "consumable", Quantity: 1, Price: 200, Weight: 1, Slots: 1},
		{ItemID: "toolkit", Name: "工具包", Kind: "consumable", Quantity: 1, Price: 300, Weight: 3, Slots: 2},
		{ItemID: "ammo_box", Name: "弹药箱", Kind: "consumable", Quantity: 60, Price: 150, Weight: 2, Slots: 2, MerchantCategory: "weapon"},
	}
	for _, inv := range owned {
		if err := upsertInventory(db, inv); err != nil {
			return err
		}
	}
	return nil
}
