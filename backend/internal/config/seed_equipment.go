package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedEquipment 装备：武器、护甲、胸挂、背包、头盔、耳机 及其初始库存。
func seedEquipment(db *gorm.DB) error {
	weapons := []models.WeaponDef{
		{ID: "melee_knife", Name: "战术匕首", Category: "melee", Hit: 72, Damage: 32, Penetration: 8, Suppress: 8, Ready: 10, AmmoPerRound: 0, Noise: 0, Reliability: 100, CloseMod: 15, MidMod: -100, FarMod: -100, Price: 300, Weight: 2, Slots: 1, MerchantCategory: "weapon"},
		{ID: "pistol_glock", Name: "G17手枪", Category: "pistol", CaliberID: "9x19", Hit: 68, Damage: 22, Penetration: 18, Suppress: 12, Ready: 12, AmmoPerRound: 1, Noise: 25, Reliability: 95, CloseMod: 5, MidMod: 0, FarMod: -25, Price: 800, Weight: 3, Slots: 1, MerchantCategory: "weapon"},
		{ID: "smg_mp5", Name: "MP5冲锋枪", Category: "smg", CaliberID: "9x19", Hit: 62, Damage: 28, Penetration: 20, Suppress: 38, Ready: 8, AmmoPerRound: 4, Noise: 60, Reliability: 90, CloseMod: 12, MidMod: 0, FarMod: -25, Price: 2200, Weight: 6, Slots: 2, MerchantCategory: "weapon", RepRequirement: 10},
		{ID: "shotgun_m870", Name: "M870霰弹枪", Category: "shotgun", CaliberID: "12g", Hit: 66, Damage: 40, Penetration: 24, Suppress: 42, Ready: 0, AmmoPerRound: 1, Noise: 75, Reliability: 92, CloseMod: 18, MidMod: -15, FarMod: -100, Price: 1800, Weight: 7, Slots: 2, MerchantCategory: "weapon"},
		{ID: "rifle_ak", Name: "AK突击步枪", Category: "rifle", CaliberID: "762x39", Hit: 68, Damage: 32, Penetration: 40, Suppress: 34, Ready: 0, AmmoPerRound: 3, Noise: 70, Reliability: 88, CloseMod: -5, MidMod: 10, FarMod: -8, Price: 3000, Weight: 8, Slots: 2, MerchantCategory: "weapon"},
		{ID: "sniper_m24", Name: "M24狙击枪", Category: "sniper", CaliberID: "762x51", Hit: 75, Damage: 50, Penetration: 58, Suppress: 25, Ready: -12, AmmoPerRound: 1, Noise: 85, Reliability: 85, CloseMod: -30, MidMod: 0, FarMod: 18, Price: 4500, Weight: 10, Slots: 3, MerchantCategory: "weapon", RepRequirement: 30},
	}
	for _, w := range weapons {
		if err := db.FirstOrCreate(&w, models.WeaponDef{ID: w.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.WeaponDef{}).Where("id=?", w.ID).Updates(w).Error; err != nil {
			return err
		}
	}

	armors := []models.ArmorDef{
		{ID: "light_01", Name: "轻型防弹衣", Type: "light", Protect: 18, ProtectionLevel: 2, Coverage: 55, Mobility: 8, Initiative: 5, Conceal: 8, AntiSuppress: 0, Escape: 10, MaxDurability: 100, Price: 1200, Weight: 5, Slots: 2, MerchantCategory: "clothing"},
		{ID: "light_02", Name: "侦察轻甲", Type: "light", Protect: 14, ProtectionLevel: 2, Coverage: 45, Mobility: 12, Initiative: 8, Conceal: 12, AntiSuppress: 2, Escape: 14, MaxDurability: 80, Price: 1000, Weight: 4, Slots: 2, MerchantCategory: "clothing"},
		{ID: "heavy_01", Name: "重型防弹甲", Type: "heavy", Protect: 42, ProtectionLevel: 5, Coverage: 85, Mobility: -10, Initiative: -8, Conceal: -12, AntiSuppress: 10, Escape: -15, MaxDurability: 150, Price: 2800, Weight: 14, Slots: 3, MerchantCategory: "clothing"},
		{ID: "heavy_02", Name: "战术重甲", Type: "heavy", Protect: 38, ProtectionLevel: 4, Coverage: 80, Mobility: -6, Initiative: -5, Conceal: -8, AntiSuppress: 8, Escape: -10, MaxDurability: 130, Price: 2400, Weight: 12, Slots: 3, MerchantCategory: "clothing", RepRequirement: 30},
	}
	for _, a := range armors {
		if err := db.FirstOrCreate(&a, models.ArmorDef{ID: a.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.ArmorDef{}).Where("id=?", a.ID).Updates(a).Error; err != nil {
			return err
		}
	}

	chestRigs := []models.ChestRigDef{
		{ID: "chestrig_01", Name: "轻型胸挂", AddSlots: 8, AddWeight: 10, Price: 400, Weight: 1, Slots: 1, MerchantCategory: "clothing"},
		{ID: "chestrig_02", Name: "战术胸挂", AddSlots: 12, AddWeight: 15, Price: 800, Weight: 2, Slots: 2, MerchantCategory: "clothing", RepRequirement: 15},
	}
	for _, cr := range chestRigs {
		if err := db.FirstOrCreate(&cr, models.ChestRigDef{ID: cr.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.ChestRigDef{}).Where("id=?", cr.ID).Updates(cr).Error; err != nil {
			return err
		}
	}

	backpacks := []models.BackpackDef{
		{ID: "backpack_01", Name: "登山背包", AddSlots: 15, AddWeight: 20, Price: 600, Weight: 2, Slots: 2, MerchantCategory: "clothing"},
		{ID: "backpack_02", Name: "战术背包", AddSlots: 20, AddWeight: 30, Price: 1200, Weight: 3, Slots: 3, MerchantCategory: "clothing", RepRequirement: 15},
	}
	for _, bp := range backpacks {
		if err := db.FirstOrCreate(&bp, models.BackpackDef{ID: bp.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.BackpackDef{}).Where("id=?", bp.ID).Updates(bp).Error; err != nil {
			return err
		}
	}

	helmets := []models.HelmetDef{
		{ID: "helmet_01", Name: "轻型头盔", Protect: 10, Coverage: 35, Mobility: 2, Initiative: 2, Conceal: 4, AntiSuppress: 2, Escape: 2, MaxDurability: 80, Price: 700, Weight: 2, Slots: 1, MerchantCategory: "clothing"},
		{ID: "helmet_02", Name: "重型头盔", Protect: 22, Coverage: 60, Mobility: -3, Initiative: -2, Conceal: -4, AntiSuppress: 5, Escape: -4, MaxDurability: 120, Price: 1600, Weight: 4, Slots: 2, MerchantCategory: "clothing", RepRequirement: 25},
	}
	for _, hl := range helmets {
		if err := db.FirstOrCreate(&hl, models.HelmetDef{ID: hl.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.HelmetDef{}).Where("id=?", hl.ID).Updates(hl).Error; err != nil {
			return err
		}
	}

	headsets := []models.HeadsetDef{
		{ID: "headset_01", Name: "基础战术耳机", HearingLevel: 1, Price: 500, Weight: 1, Slots: 1, MerchantCategory: "clothing"},
		{ID: "headset_02", Name: "降噪指挥耳机", HearingLevel: 2, Price: 1200, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 20},
	}
	for _, hs := range headsets {
		if err := db.FirstOrCreate(&hs, models.HeadsetDef{ID: hs.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.HeadsetDef{}).Where("id=?", hs.ID).Updates(hs).Error; err != nil {
			return err
		}
	}

	if err := seedEquipmentExpansion(db); err != nil {
		return err
	}

	for _, inv := range initialEquipmentInventory() {
		if err := upsertInventory(db, inv); err != nil {
			return err
		}
	}
	return nil
}

func initialEquipmentInventory() []models.Inventory {
	return []models.Inventory{
		{ItemID: "rifle_ak", Name: "AK突击步枪", Kind: "weapon", Quantity: 1, Price: 3000, Weight: 8, Slots: 2, MerchantCategory: "weapon"},
		{ItemID: "light_01", Name: "轻型防弹衣", Kind: "armor", Quantity: 1, Price: 1200, Weight: 5, Slots: 2, MerchantCategory: "clothing"},
		{ItemID: "heavy_01", Name: "重型防弹甲", Kind: "armor", Quantity: 1, Price: 2800, Weight: 14, Slots: 3, MerchantCategory: "clothing"},
		{ItemID: "chestrig_01", Name: "轻型胸挂", Kind: "chestrig", Quantity: 1, Price: 400, Weight: 1, Slots: 1, MerchantCategory: "clothing"},
		{ItemID: "backpack_01", Name: "登山背包", Kind: "backpack", Quantity: 1, Price: 600, Weight: 2, Slots: 2, MerchantCategory: "clothing"},
		{ItemID: "helmet_01", Name: "轻型头盔", Kind: "helmet", Quantity: 1, Price: 700, Weight: 2, Slots: 1, MerchantCategory: "clothing"},
		{ItemID: "headset_01", Name: "基础战术耳机", Kind: "headset", Quantity: 1, Price: 500, Weight: 1, Slots: 1, MerchantCategory: "clothing"},
	}
}
