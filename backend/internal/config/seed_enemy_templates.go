// 敌人模板种子：定义 grunt/guard/elite/boss/sniper 各类敌人的生成规则。
// 分类参考逃离塔科夫敌人体系：Scav(拾荒者)/Raider(突袭者)/Rogue(据点守卫)/Boss(首领)/Cultist(邪教徒)。
package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

func seedEnemyTemplates(db *gorm.DB) error {
	templates := []models.EnemyTemplateDef{
		// ---- 杂鱼 (grunt) ----
		{
			ID: "template_scav_grunt", Name: "拾荒者", Kind: "grunt", SpawnTags: []string{"outdoor", "residential", "industrial"},
			Tier: 0, HPBase: 60, HPFlux: 10, HPFloor: 40, HPCap: 90, StressBase: 60, StressFlux: 8, StressFloor: 40, StressCap: 80,
			PerceptionBase: 30, PerceptionFlux: 5, StealthBase: 35, StealthFlux: 5, AgilityBase: 35, AgilityFlux: 5,
			EvasionBase: 10, EvasionFlux: 3, MobilityBase: 6, MobilityFlux: 2, SuppressBase: 10, SuppressFlux: 3,
			WeaponPool: []models.WeightedRef{{Ref: "pistol_pm", Weight: 40}, {Ref: "shotgun_m870", Weight: 35}, {Ref: "rifle_sks", Weight: 25}},
			ArmorPool:  []models.WeightedRef{{Ref: "armor_paca", Weight: 60}, {Ref: "light_01", Weight: 40}},
			AmmoLevelMin: 1, AmmoLevelMax: 3, AmmoRoundsBase: 20, AmmoRoundsMult: 1.0,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_basic", Weight: 100}},
			SortOrder: 10,
		},
		{
			ID: "template_patrol", Name: "巡逻小队", Kind: "grunt", SpawnTags: []string{"outdoor", "underground", "residential"},
			Tier: 0, HPBase: 80, HPFlux: 10, HPFloor: 60, HPCap: 100, StressBase: 65, StressFlux: 8, StressFloor: 45, StressCap: 85,
			PerceptionBase: 40, PerceptionFlux: 5, StealthBase: 30, StealthFlux: 5, AgilityBase: 45, AgilityFlux: 4,
			EvasionBase: 15, EvasionFlux: 3, MobilityBase: 5, MobilityFlux: 2, SuppressBase: 20, SuppressFlux: 4,
			WeaponPool: []models.WeightedRef{{Ref: "pistol_glock", Weight: 60}, {Ref: "smg_pp1901", Weight: 40}},
			ArmorPool:  []models.WeightedRef{{Ref: "light_01", Weight: 100}},
			AmmoLevelMin: 1, AmmoLevelMax: 3, AmmoRoundsBase: 24, AmmoRoundsMult: 1.0,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_basic", Weight: 100}},
			SortOrder: 11,
		},

		// ---- 守卫 (guard) ----
		{
			ID: "template_guard", Name: "据点守卫", Kind: "guard", SpawnTags: []string{"indoor", "industrial", "secured"},
			Tier: 1, HPBase: 100, HPFlux: 12, HPFloor: 80, HPCap: 130, StressBase: 80, StressFlux: 10, StressFloor: 60, StressCap: 100,
			PerceptionBase: 50, PerceptionFlux: 5, StealthBase: 25, StealthFlux: 4, AgilityBase: 40, AgilityFlux: 4,
			EvasionBase: 12, EvasionFlux: 3, MobilityBase: 0, MobilityFlux: 2, SuppressBase: 35, SuppressFlux: 5,
			WeaponPool: []models.WeightedRef{{Ref: "smg_mp5", Weight: 50}, {Ref: "shotgun_mp133", Weight: 30}, {Ref: "pistol_m9a3", Weight: 20}},
			ArmorPool:  []models.WeightedRef{{Ref: "light_01", Weight: 70}, {Ref: "armor_6b2", Weight: 30}},
			AmmoLevelMin: 2, AmmoLevelMax: 4, AmmoRoundsBase: 30, AmmoRoundsMult: 1.0,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_guard", Weight: 100}},
			SortOrder: 20,
		},
		{
			ID: "template_rogue", Name: "据点暴徒", Kind: "guard", SpawnTags: []string{"industrial", "secured", "high_value"},
			Tier: 2, HPBase: 110, HPFlux: 12, HPFloor: 90, HPCap: 150, StressBase: 85, StressFlux: 10, StressFloor: 65, StressCap: 110,
			PerceptionBase: 55, PerceptionFlux: 6, StealthBase: 20, StealthFlux: 4, AgilityBase: 42, AgilityFlux: 4,
			EvasionBase: 13, EvasionFlux: 3, MobilityBase: -2, MobilityFlux: 2, SuppressBase: 40, SuppressFlux: 6,
			WeaponPool: []models.WeightedRef{{Ref: "rifle_ak", Weight: 50}, {Ref: "rifle_ak74n", Weight: 30}, {Ref: "smg_ump45", Weight: 20}},
			ArmorPool:  []models.WeightedRef{{Ref: "armor_6b13", Weight: 60}, {Ref: "heavy_01", Weight: 40}},
			AmmoLevelMin: 3, AmmoLevelMax: 4, AmmoRoundsBase: 60, AmmoRoundsMult: 1.2,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_guard", Weight: 100}},
			SortOrder: 21,
		},

		// ---- 精锐 (elite) ----
		{
			ID: "template_raider", Name: "突袭者", Kind: "elite", SpawnTags: []string{"secured", "high_value", "industrial"},
			Tier: 2, HPBase: 115, HPFlux: 15, HPFloor: 95, HPCap: 150, StressBase: 90, StressFlux: 12, StressFloor: 70, StressCap: 115,
			PerceptionBase: 60, PerceptionFlux: 6, StealthBase: 30, StealthFlux: 5, AgilityBase: 48, AgilityFlux: 4,
			EvasionBase: 17, EvasionFlux: 3, MobilityBase: -3, MobilityFlux: 2, SuppressBase: 38, SuppressFlux: 5,
			WeaponPool: []models.WeightedRef{{Ref: "rifle_ak", Weight: 40}, {Ref: "rifle_ak74n", Weight: 30}, {Ref: "smg_mp5", Weight: 30}},
			ArmorPool:  []models.WeightedRef{{Ref: "heavy_01", Weight: 60}, {Ref: "armor_6b13", Weight: 40}},
			AmmoLevelMin: 3, AmmoLevelMax: 5, AmmoRoundsBase: 60, AmmoRoundsMult: 1.3,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_elite", Weight: 100}},
			SortOrder: 30,
		},
		{
			ID: "template_elite", Name: "精锐小队", Kind: "elite", SpawnTags: []string{"secured", "high_value", "indoor"},
			Tier: 3, HPBase: 120, HPFlux: 15, HPFloor: 100, HPCap: 160, StressBase: 95, StressFlux: 12, StressFloor: 70, StressCap: 120,
			PerceptionBase: 60, PerceptionFlux: 6, StealthBase: 35, StealthFlux: 5, AgilityBase: 50, AgilityFlux: 4,
			EvasionBase: 18, EvasionFlux: 4, MobilityBase: -5, MobilityFlux: 2, SuppressBase: 40, SuppressFlux: 6,
			WeaponPool: []models.WeightedRef{{Ref: "rifle_ak", Weight: 70}, {Ref: "rifle_akm", Weight: 30}},
			ArmorPool:  []models.WeightedRef{{Ref: "heavy_01", Weight: 100}},
			AmmoLevelMin: 4, AmmoLevelMax: 4, AmmoRoundsBase: 30, AmmoRoundsMult: 1.0,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_elite", Weight: 100}},
			SortOrder: 31,
		},

		// ---- 狙击 (sniper) ----
		{
			ID: "template_sniper", Name: "狙击哨位", Kind: "sniper", SpawnTags: []string{"far", "high_value", "outdoor"},
			Tier: 2, HPBase: 70, HPFlux: 10, HPFloor: 55, HPCap: 90, StressBase: 60, StressFlux: 8, StressFloor: 45, StressCap: 75,
			PerceptionBase: 70, PerceptionFlux: 5, StealthBase: 50, StealthFlux: 6, AgilityBase: 35, AgilityFlux: 4,
			EvasionBase: 20, EvasionFlux: 4, MobilityBase: 8, MobilityFlux: 2, SuppressBase: 25, SuppressFlux: 4,
			WeaponPool: []models.WeightedRef{{Ref: "sniper_m24", Weight: 60}, {Ref: "sniper_mosin", Weight: 40}},
			ArmorPool:  []models.WeightedRef{{Ref: "light_02", Weight: 100}},
			AmmoLevelMin: 3, AmmoLevelMax: 5, AmmoRoundsBase: 10, AmmoRoundsMult: 1.0,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_basic", Weight: 100}},
			SortOrder: 40,
		},

		// ---- 邪教徒 (elite/近战) ----
		{
			ID: "template_cultist", Name: "邪教徒", Kind: "elite", SpawnTags: []string{"underground", "indoor", "medical"},
			Tier: 2, HPBase: 100, HPFlux: 12, HPFloor: 80, HPCap: 130, StressBase: 100, StressFlux: 10, StressFloor: 80, StressCap: 130,
			PerceptionBase: 65, PerceptionFlux: 6, StealthBase: 70, StealthFlux: 8, AgilityBase: 55, AgilityFlux: 6,
			EvasionBase: 22, EvasionFlux: 4, MobilityBase: 5, MobilityFlux: 3, SuppressBase: 15, SuppressFlux: 3,
			WeaponPool: []models.WeightedRef{{Ref: "melee_knife", Weight: 50}, {Ref: "smg_pp1901", Weight: 30}, {Ref: "pistol_pm", Weight: 20}},
			ArmorPool:  []models.WeightedRef{{Ref: "light_02", Weight: 70}, {Ref: "armor_paca", Weight: 30}},
			AmmoLevelMin: 2, AmmoLevelMax: 4, AmmoRoundsBase: 30, AmmoRoundsMult: 1.0,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_guard", Weight: 100}},
			SortOrder: 35,
		},

		// ---- BOSS ----
		{
			ID: "template_boss_killa", Name: "基拉", Kind: "boss", SpawnTags: []string{"high_value", "indoor"},
			Tier: 3, HPBase: 180, HPFlux: 20, HPFloor: 150, HPCap: 240, StressBase: 120, StressFlux: 15, StressFloor: 95, StressCap: 150,
			PerceptionBase: 65, PerceptionFlux: 5, StealthBase: 25, StealthFlux: 3, AgilityBase: 50, AgilityFlux: 3,
			EvasionBase: 15, EvasionFlux: 2, MobilityBase: -8, MobilityFlux: 1, SuppressBase: 55, SuppressFlux: 5,
			BossWeaponID: "rifle_rpk16", BossArmorID: "heavy_01", BossName: "【BOSS】铁腕·基拉",
			AmmoLevelMin: 4, AmmoLevelMax: 4, AmmoRoundsBase: 120, AmmoRoundsMult: 1.5,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_elite", Weight: 100}},
			BossAmmoDrop: true,
			SortOrder:   50,
		},
		{
			ID: "template_boss_shturman", Name: "施图尔曼", Kind: "boss", SpawnTags: []string{"high_value", "far", "outdoor"},
			Tier: 3, HPBase: 160, HPFlux: 20, HPFloor: 140, HPCap: 210, StressBase: 110, StressFlux: 15, StressFloor: 85, StressCap: 140,
			PerceptionBase: 80, PerceptionFlux: 5, StealthBase: 45, StealthFlux: 4, AgilityBase: 45, AgilityFlux: 3,
			EvasionBase: 20, EvasionFlux: 3, MobilityBase: -5, MobilityFlux: 2, SuppressBase: 30, SuppressFlux: 4,
			BossWeaponID: "sniper_svds", BossArmorID: "heavy_01", BossName: "【BOSS】林驻·施图尔曼",
			AmmoLevelMin: 4, AmmoLevelMax: 5, AmmoRoundsBase: 40, AmmoRoundsMult: 1.2,
			BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_elite", Weight: 100}},
			BossAmmoDrop: true,
			SortOrder:   51,
		},
	}
	for _, t := range templates {
		if err := upsertSeedDef(db, &t, t.ID); err != nil {
			return err
		}
	}
	return nil
}
