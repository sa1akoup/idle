package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedEquipment 装备：武器、护甲、胸挂、背包、头盔、耳机 及其初始库存。
// 装备名称参考 Tarkov Wiki 的武器、护甲、胸挂、背包、头盔与耳机目录，数值映射到简化战斗和携行模型。
func seedEquipment(db *gorm.DB) error {
	weapons := []models.WeaponDef{
		// 近战与手枪
		{ID: "melee_knife", Name: "战术匕首", Category: "melee", Hit: 72, Damage: 32, Penetration: 8, Suppress: 8, Ready: 10, AmmoPerRound: 0, Noise: 0, Reliability: 100, CloseMod: 15, MidMod: -100, FarMod: -100, Price: 300, Weight: 2, Slots: 1, MerchantCategory: "weapon"},
		{ID: "melee_hatchet", Name: "战术手斧", Category: "melee", Hit: 68, Damage: 38, Penetration: 10, Suppress: 10, Ready: 8, Reliability: 95, CloseMod: 20, MidMod: -100, FarMod: -100, Price: 600, Weight: 3, Slots: 2, MerchantCategory: "weapon"},
		{ID: "melee_crowbar", Name: "撬棍", Category: "melee", Hit: 64, Damage: 42, Penetration: 12, Suppress: 12, Ready: 4, Reliability: 100, CloseMod: 16, MidMod: -100, FarMod: -100, Price: 500, Weight: 3, Slots: 2, MerchantCategory: "weapon"},
		{ID: "pistol_glock", Name: "G17手枪", Category: "pistol", CaliberID: "9x19", Hit: 68, Damage: 22, Penetration: 18, Suppress: 12, Ready: 12, AmmoPerRound: 1, Noise: 25, Reliability: 95, CloseMod: 5, MidMod: 0, FarMod: -25, Price: 800, Weight: 3, Slots: 1, MerchantCategory: "weapon"},
		{ID: "pistol_pm", Name: "PM 手枪", Category: "pistol", CaliberID: "9x18", Hit: 64, Damage: 20, Penetration: 12, Suppress: 12, Ready: 14, AmmoPerRound: 1, Noise: 20, Reliability: 96, CloseMod: 5, MidMod: 0, FarMod: -30, Price: 400, Weight: 2, Slots: 1, MerchantCategory: "weapon"},
		{ID: "pistol_m9a3", Name: "M9A3 手枪", Category: "pistol", CaliberID: "9x19", Hit: 70, Damage: 24, Penetration: 20, Suppress: 14, Ready: 13, AmmoPerRound: 1, Noise: 28, Reliability: 96, CloseMod: 6, MidMod: 1, FarMod: -20, Price: 1000, Weight: 3, Slots: 1, MerchantCategory: "weapon", RepRequirement: 5},
		{ID: "pistol_five7", Name: "Five-seveN 手枪", Category: "pistol", CaliberID: "57x28", Hit: 72, Damage: 26, Penetration: 36, Suppress: 15, Ready: 12, AmmoPerRound: 1, Noise: 30, Reliability: 95, CloseMod: 4, MidMod: 4, FarMod: -10, Price: 1800, Weight: 3, Slots: 1, MerchantCategory: "weapon", RepRequirement: 15},
		{ID: "pistol_usp45", Name: "USP .45 手枪", Category: "pistol", CaliberID: "45acp", Hit: 68, Damage: 32, Penetration: 25, Suppress: 15, Ready: 10, AmmoPerRound: 1, Noise: 34, Reliability: 94, CloseMod: 5, MidMod: 0, FarMod: -20, Price: 1400, Weight: 3, Slots: 1, MerchantCategory: "weapon", RepRequirement: 10},
		{ID: "pistol_mp443", Name: "MP-443 Grach 手枪", Category: "pistol", CaliberID: "9x19", Hit: 67, Damage: 24, Penetration: 19, Suppress: 13, Ready: 12, AmmoPerRound: 1, Noise: 24, Reliability: 95, CloseMod: 5, MidMod: 0, FarMod: -25, Price: 650, Weight: 2, Slots: 1, MerchantCategory: "weapon"},

		// 冲锋枪
		{ID: "smg_mp5", Name: "MP5冲锋枪", Category: "smg", CaliberID: "9x19", Hit: 62, Damage: 28, Penetration: 20, Suppress: 38, Ready: 8, AmmoPerRound: 4, Noise: 60, Reliability: 90, CloseMod: 12, MidMod: 0, FarMod: -25, Price: 2200, Weight: 6, Slots: 2, MerchantCategory: "weapon", RepRequirement: 10},
		{ID: "smg_pp1901", Name: "PP-19-01 Vityaz 冲锋枪", Category: "smg", CaliberID: "9x19", Hit: 64, Damage: 26, Penetration: 18, Suppress: 35, Ready: 9, AmmoPerRound: 4, Noise: 50, Reliability: 90, CloseMod: 12, MidMod: 0, FarMod: -25, Price: 1500, Weight: 5, Slots: 2, MerchantCategory: "weapon"},
		{ID: "smg_mp7", Name: "MP7A1 冲锋枪", Category: "smg", CaliberID: "46x30", Hit: 72, Damage: 34, Penetration: 42, Suppress: 40, Ready: 8, AmmoPerRound: 5, Noise: 55, Reliability: 93, CloseMod: 12, MidMod: 4, FarMod: -15, Price: 4200, Weight: 3, Slots: 2, MerchantCategory: "weapon", RepRequirement: 25},
		{ID: "smg_p90", Name: "P90 冲锋枪", Category: "smg", CaliberID: "57x28", Hit: 70, Damage: 32, Penetration: 38, Suppress: 42, Ready: 8, AmmoPerRound: 5, Noise: 58, Reliability: 92, CloseMod: 14, MidMod: 4, FarMod: -10, Price: 4800, Weight: 6, Slots: 2, MerchantCategory: "weapon", RepRequirement: 30},
		{ID: "smg_ump45", Name: "UMP .45 冲锋枪", Category: "smg", CaliberID: "45acp", Hit: 68, Damage: 35, Penetration: 28, Suppress: 39, Ready: 9, AmmoPerRound: 4, Noise: 48, Reliability: 94, CloseMod: 10, MidMod: 0, FarMod: -20, Price: 2600, Weight: 5, Slots: 2, MerchantCategory: "weapon", RepRequirement: 15},
		{ID: "smg_mpx", Name: "MPX 冲锋枪", Category: "smg", CaliberID: "9x19", Hit: 66, Damage: 29, Penetration: 22, Suppress: 36, Ready: 9, AmmoPerRound: 4, Noise: 52, Reliability: 92, CloseMod: 12, MidMod: 2, FarMod: -20, Price: 2800, Weight: 5, Slots: 2, MerchantCategory: "weapon", RepRequirement: 15},

		// 霰弹枪
		{ID: "shotgun_m870", Name: "M870霰弹枪", Category: "shotgun", CaliberID: "12g", Hit: 66, Damage: 40, Penetration: 24, Suppress: 42, Ready: 0, AmmoPerRound: 1, Noise: 75, Reliability: 92, CloseMod: 18, MidMod: -15, FarMod: -100, Price: 1800, Weight: 7, Slots: 2, MerchantCategory: "weapon"},
		{ID: "shotgun_mp133", Name: "MP-133 霰弹枪", Category: "shotgun", CaliberID: "12g", Hit: 68, Damage: 38, Penetration: 22, Suppress: 40, Ready: 0, AmmoPerRound: 1, Noise: 76, Reliability: 95, CloseMod: 20, MidMod: -20, FarMod: -100, Price: 1300, Weight: 7, Slots: 2, MerchantCategory: "weapon"},
		{ID: "shotgun_mp153", Name: "MP-153 霰弹枪", Category: "shotgun", CaliberID: "12g", Hit: 64, Damage: 43, Penetration: 28, Suppress: 38, Ready: 0, AmmoPerRound: 1, Noise: 78, Reliability: 90, CloseMod: 16, MidMod: -10, FarMod: -100, Price: 2500, Weight: 8, Slots: 3, MerchantCategory: "weapon", RepRequirement: 10},
		{ID: "shotgun_saiga12", Name: "Saiga-12 霰弹枪", Category: "shotgun", CaliberID: "12g", Hit: 62, Damage: 42, Penetration: 26, Suppress: 35, Ready: 0, AmmoPerRound: 2, Noise: 78, Reliability: 88, CloseMod: 12, MidMod: -10, FarMod: -100, Price: 3000, Weight: 9, Slots: 3, MerchantCategory: "weapon", RepRequirement: 20},

		// 突击步枪与卡宾枪
		{ID: "rifle_ak", Name: "AK突击步枪", Category: "rifle", CaliberID: "762x39", Hit: 68, Damage: 32, Penetration: 40, Suppress: 34, Ready: 0, AmmoPerRound: 3, Noise: 70, Reliability: 88, CloseMod: -5, MidMod: 10, FarMod: -8, Price: 3000, Weight: 8, Slots: 2, MerchantCategory: "weapon"},
		{ID: "rifle_ak74n", Name: "AK-74N 突击步枪", Category: "rifle", CaliberID: "545x39", Hit: 69, Damage: 30, Penetration: 35, Suppress: 32, Ready: 0, AmmoPerRound: 3, Noise: 68, Reliability: 91, CloseMod: -2, MidMod: 12, FarMod: -6, Price: 2500, Weight: 8, Slots: 2, MerchantCategory: "weapon"},
		{ID: "rifle_akm", Name: "AKM 突击步枪", Category: "rifle", CaliberID: "762x39", Hit: 66, Damage: 36, Penetration: 45, Suppress: 34, Ready: 0, AmmoPerRound: 3, Noise: 72, Reliability: 89, CloseMod: -4, MidMod: 10, FarMod: -6, Price: 3200, Weight: 8, Slots: 2, MerchantCategory: "weapon", RepRequirement: 10},
		{ID: "rifle_m4a1", Name: "M4A1 突击步枪", Category: "rifle", CaliberID: "556x45", Hit: 72, Damage: 31, Penetration: 38, Suppress: 36, Ready: 0, AmmoPerRound: 3, Noise: 68, Reliability: 94, CloseMod: 2, MidMod: 12, FarMod: 0, Price: 4300, Weight: 8, Slots: 2, MerchantCategory: "weapon", RepRequirement: 20},
		{ID: "rifle_hk416", Name: "HK 416A5 突击步枪", Category: "rifle", CaliberID: "556x45", Hit: 74, Damage: 33, Penetration: 40, Suppress: 40, Ready: 0, AmmoPerRound: 4, Noise: 70, Reliability: 95, CloseMod: 4, MidMod: 14, FarMod: 2, Price: 5200, Weight: 8, Slots: 2, MerchantCategory: "weapon", RepRequirement: 30},
		{ID: "rifle_scar_l", Name: "SCAR-L 突击步枪", Category: "rifle", CaliberID: "556x45", Hit: 70, Damage: 34, Penetration: 43, Suppress: 37, Ready: 0, AmmoPerRound: 3, Noise: 72, Reliability: 93, CloseMod: 0, MidMod: 10, FarMod: 2, Price: 5000, Weight: 9, Slots: 2, MerchantCategory: "weapon", RepRequirement: 25},
		{ID: "rifle_asval", Name: "AS VAL 突击步枪", Category: "rifle", CaliberID: "9x39", Hit: 68, Damage: 38, Penetration: 45, Suppress: 30, Ready: 0, AmmoPerRound: 4, Noise: 60, Reliability: 87, CloseMod: 8, MidMod: 8, FarMod: -5, Price: 5200, Weight: 7, Slots: 2, MerchantCategory: "weapon", RepRequirement: 30},
		{ID: "rifle_rpk16", Name: "RPK-16 轻机枪", Category: "rifle", CaliberID: "545x39", Hit: 65, Damage: 34, Penetration: 41, Suppress: 35, Ready: 0, AmmoPerRound: 4, Noise: 70, Reliability: 90, CloseMod: -5, MidMod: 10, FarMod: -8, Price: 3800, Weight: 9, Slots: 3, MerchantCategory: "weapon", RepRequirement: 20},
		{ID: "rifle_sks", Name: "SKS 卡宾枪", Category: "rifle", CaliberID: "762x39", Hit: 68, Damage: 34, Penetration: 36, Suppress: 27, Ready: 0, AmmoPerRound: 1, Noise: 58, Reliability: 93, CloseMod: -2, MidMod: 8, FarMod: 5, Price: 1700, Weight: 5, Slots: 2, MerchantCategory: "weapon"},

		// 栓动与精确射手武器
		{ID: "sniper_m24", Name: "M24狙击枪", Category: "sniper", CaliberID: "762x51", Hit: 75, Damage: 50, Penetration: 58, Suppress: 25, Ready: -12, AmmoPerRound: 1, Noise: 85, Reliability: 85, CloseMod: -30, MidMod: 0, FarMod: 18, Price: 4500, Weight: 10, Slots: 3, MerchantCategory: "weapon", RepRequirement: 30},
		{ID: "sniper_mosin", Name: "莫辛纳甘步枪", Category: "sniper", CaliberID: "762x54r", Hit: 71, Damage: 55, Penetration: 60, Suppress: 25, Ready: -10, AmmoPerRound: 1, Noise: 82, Reliability: 90, CloseMod: -30, MidMod: 0, FarMod: 15, Price: 2400, Weight: 5, Slots: 2, MerchantCategory: "weapon"},
		{ID: "sniper_sv98", Name: "SV-98 狙击步枪", Category: "sniper", CaliberID: "762x54r", Hit: 76, Damage: 54, Penetration: 62, Suppress: 26, Ready: -12, AmmoPerRound: 1, Noise: 80, Reliability: 90, CloseMod: -28, MidMod: 4, FarMod: 18, Price: 4200, Weight: 6, Slots: 2, MerchantCategory: "weapon", RepRequirement: 20},
		{ID: "sniper_svds", Name: "SVDS 精确射手步枪", Category: "sniper", CaliberID: "762x54r", Hit: 73, Damage: 52, Penetration: 60, Suppress: 28, Ready: -8, AmmoPerRound: 3, Noise: 84, Reliability: 88, CloseMod: -25, MidMod: 5, FarMod: 14, Price: 5500, Weight: 7, Slots: 3, MerchantCategory: "weapon", RepRequirement: 25},
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
		{ID: "armor_paca", Name: "PACA 软质护甲", Type: "light", Protect: 12, ProtectionLevel: 2, Coverage: 40, Mobility: 10, Initiative: 7, Conceal: 10, AntiSuppress: 1, Escape: 12, MaxDurability: 70, Price: 900, Weight: 4, Slots: 2, MerchantCategory: "clothing"},
		{ID: "armor_6b2", Name: "6B2 防弹衣", Type: "light", Protect: 20, ProtectionLevel: 3, Coverage: 55, Mobility: 3, Initiative: 2, Conceal: 3, AntiSuppress: 3, Escape: 4, MaxDurability: 90, Price: 1400, Weight: 6, Slots: 2, MerchantCategory: "clothing"},
		{ID: "armor_6b13", Name: "6B13 突击护甲", Type: "heavy", Protect: 32, ProtectionLevel: 4, Coverage: 65, Mobility: -2, Initiative: -2, Conceal: -4, AntiSuppress: 5, Escape: -4, MaxDurability: 120, Price: 2300, Weight: 10, Slots: 3, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "armor_korund", Name: "Korund-VM 防弹衣", Type: "heavy", Protect: 40, ProtectionLevel: 5, Coverage: 65, Mobility: -8, Initiative: -5, Conceal: -8, AntiSuppress: 10, Escape: -12, MaxDurability: 140, Price: 3200, Weight: 12, Slots: 3, MerchantCategory: "clothing", RepRequirement: 20},
		{ID: "armor_trooper", Name: "Trooper 防弹衣", Type: "light", Protect: 28, ProtectionLevel: 3, Coverage: 55, Mobility: 2, Initiative: 2, Conceal: 4, AntiSuppress: 4, Escape: 4, MaxDurability: 105, Price: 2600, Weight: 8, Slots: 3, MerchantCategory: "clothing", RepRequirement: 20},
		{ID: "armor_slick", Name: "Slick 防弹板载具", Type: "heavy", Protect: 48, ProtectionLevel: 5, Coverage: 75, Mobility: -4, Initiative: -4, Conceal: -7, AntiSuppress: 8, Escape: -8, MaxDurability: 160, Price: 5200, Weight: 8, Slots: 3, MerchantCategory: "clothing", RepRequirement: 35},
		{ID: "armor_redut", Name: "Redut-M 防弹衣", Type: "heavy", Protect: 44, ProtectionLevel: 5, Coverage: 80, Mobility: -12, Initiative: -8, Conceal: -12, AntiSuppress: 12, Escape: -16, MaxDurability: 170, Price: 4600, Weight: 15, Slots: 4, MerchantCategory: "clothing", RepRequirement: 30},
		{ID: "armor_defender2", Name: "Defender-2 防弹衣", Type: "heavy", Protect: 50, ProtectionLevel: 6, Coverage: 85, Mobility: -14, Initiative: -9, Conceal: -15, AntiSuppress: 14, Escape: -18, MaxDurability: 190, Price: 6000, Weight: 18, Slots: 4, MerchantCategory: "clothing", RepRequirement: 40},
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
		{ID: "chestrig_bank_robber", Name: "Bank Robber 轻型胸挂", AddSlots: 6, AddWeight: 8, Price: 300, Weight: 1, Slots: 1, MerchantCategory: "clothing"},
		{ID: "chestrig_alpha", Name: "Alpha 战术胸挂", AddSlots: 12, AddWeight: 14, Price: 900, Weight: 1, Slots: 2, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "chestrig_blackrock", Name: "BlackRock 胸挂", AddSlots: 14, AddWeight: 18, Price: 1200, Weight: 2, Slots: 2, MerchantCategory: "clothing", RepRequirement: 15},
		{ID: "chestrig_mk3", Name: "MK3 胸挂", AddSlots: 14, AddWeight: 16, Price: 1000, Weight: 2, Slots: 2, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "chestrig_avs", Name: "AVS 胸挂", AddSlots: 16, AddWeight: 20, Price: 1500, Weight: 2, Slots: 2, MerchantCategory: "clothing", RepRequirement: 20},
		{ID: "chestrig_tv110", Name: "TV-110 胸挂", AddSlots: 18, AddWeight: 22, Price: 1800, Weight: 3, Slots: 3, MerchantCategory: "clothing", RepRequirement: 25},
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
		{ID: "backpack_mbss", Name: "MBSS 背包", AddSlots: 10, AddWeight: 12, Price: 400, Weight: 1, Slots: 2, MerchantCategory: "clothing"},
		{ID: "backpack_scav", Name: "Scav 背包", AddSlots: 18, AddWeight: 25, Price: 900, Weight: 2, Slots: 2, MerchantCategory: "clothing"},
		{ID: "backpack_daypack", Name: "Day Pack 背包", AddSlots: 20, AddWeight: 25, Price: 1000, Weight: 2, Slots: 3, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "backpack_attack2", Name: "Attack 2 背包", AddSlots: 24, AddWeight: 35, Price: 1800, Weight: 3, Slots: 3, MerchantCategory: "clothing", RepRequirement: 20},
		{ID: "backpack_pilgrim", Name: "Pilgrim 背包", AddSlots: 30, AddWeight: 40, Price: 2600, Weight: 4, Slots: 4, MerchantCategory: "clothing", RepRequirement: 30},
		{ID: "backpack_blackjack", Name: "Blackjack 背包", AddSlots: 35, AddWeight: 45, Price: 3600, Weight: 5, Slots: 5, MerchantCategory: "clothing", RepRequirement: 35},
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
		{ID: "helmet_ssh68", Name: "SSh-68 钢盔", Protect: 8, Coverage: 40, Mobility: 5, Initiative: 2, Conceal: 5, AntiSuppress: 1, Escape: 4, MaxDurability: 70, Price: 450, Weight: 2, Slots: 1, MerchantCategory: "clothing"},
		{ID: "helmet_6b47", Name: "6B47 战斗头盔", Protect: 14, Coverage: 45, Mobility: 1, Initiative: 1, Conceal: 2, AntiSuppress: 3, Escape: 1, MaxDurability: 90, Price: 900, Weight: 3, Slots: 1, MerchantCategory: "clothing"},
		{ID: "helmet_untar", Name: "UNTAR 防弹头盔", Protect: 12, Coverage: 50, Mobility: -1, Initiative: 0, Conceal: -2, AntiSuppress: 2, Escape: 0, MaxDurability: 85, Price: 1000, Weight: 3, Slots: 1, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "helmet_fast_mt", Name: "FAST MT 战术头盔", Protect: 18, Coverage: 55, Mobility: -1, Initiative: 1, Conceal: -2, AntiSuppress: 3, Escape: 0, MaxDurability: 100, Price: 1800, Weight: 2, Slots: 2, MerchantCategory: "clothing", RepRequirement: 20},
		{ID: "helmet_zsh1", Name: "ZSh-1-2M 重型头盔", Protect: 28, Coverage: 65, Mobility: -5, Initiative: -3, Conceal: -6, AntiSuppress: 6, Escape: -6, MaxDurability: 135, Price: 2500, Weight: 5, Slots: 2, MerchantCategory: "clothing", RepRequirement: 25},
		{ID: "helmet_airframe", Name: "Airframe 战术头盔", Protect: 24, Coverage: 60, Mobility: -3, Initiative: -1, Conceal: -5, AntiSuppress: 5, Escape: -4, MaxDurability: 120, Price: 2800, Weight: 3, Slots: 2, MerchantCategory: "clothing", RepRequirement: 30},
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
		{ID: "headset_gssh", Name: "GSSh-01 耳机", HearingLevel: 1, Price: 650, Weight: 1, Slots: 1, MerchantCategory: "clothing"},
		{ID: "headset_comtac2", Name: "ComTac II 耳机", HearingLevel: 2, Price: 900, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "headset_m32", Name: "M32 耳机", HearingLevel: 2, Price: 1000, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 10},
		{ID: "headset_peltor", Name: "Peltor Tactical Sport 耳机", HearingLevel: 2, Price: 1100, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 15},
		{ID: "headset_comtac4", Name: "ComTac IV 耳机", HearingLevel: 3, Price: 1600, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 25},
		{ID: "headset_xcel", Name: "XCEL 500BT 耳机", HearingLevel: 3, Price: 1800, Weight: 1, Slots: 1, MerchantCategory: "clothing", RepRequirement: 30},
	}
	for _, hs := range headsets {
		if err := db.FirstOrCreate(&hs, models.HeadsetDef{ID: hs.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.HeadsetDef{}).Where("id=?", hs.ID).Updates(hs).Error; err != nil {
			return err
		}
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
