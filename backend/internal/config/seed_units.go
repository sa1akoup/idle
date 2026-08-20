package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedUnits 单位数据：敌人。
func seedUnits(db *gorm.DB) error {
	enemies := []models.EnemyDef{
		{ID: "enemy_patrol", Name: "巡逻小队", HP: 80, StressThreshold: 60, Perception: 40, Stealth: 30, Agility: 45, WeaponID: "pistol_glock", ArmorID: "light_01", Evasion: 15, Mobility: 5, Suppress: 20, BackpackContainerID: "enemy_backpack_basic"},
		{ID: "enemy_guard", Name: "据点守卫", HP: 100, StressThreshold: 70, Perception: 50, Stealth: 25, Agility: 40, WeaponID: "smg_mp5", ArmorID: "light_01", Evasion: 12, Mobility: 0, Suppress: 35, BackpackContainerID: "enemy_backpack_guard"},
		{ID: "enemy_elite", Name: "精锐小队", HP: 120, StressThreshold: 80, Perception: 60, Stealth: 35, Agility: 50, WeaponID: "rifle_ak", ArmorID: "heavy_01", Evasion: 18, Mobility: -5, Suppress: 40, BackpackContainerID: "enemy_backpack_elite"},
		{ID: "enemy_sniper", Name: "狙击哨位", HP: 70, StressThreshold: 50, Perception: 70, Stealth: 50, Agility: 35, WeaponID: "sniper_m24", ArmorID: "light_02", Evasion: 20, Mobility: 8, Suppress: 25, BackpackContainerID: "enemy_backpack_basic"},
	}
	for _, e := range enemies {
		if err := db.FirstOrCreate(&e, models.EnemyDef{ID: e.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.EnemyDef{}).Where("id=?", e.ID).Updates(e).Error; err != nil {
			return err
		}
	}
	return nil
}
