// 生存系统种子：统一初始化 HP、Energy、Hydration 物品效果和耐久实例。
package config

import (
	"fmt"
	"time"

	"idle/internal/models"

	"gorm.io/gorm"
)

type survivalEffect struct {
	hp            float64
	energy        float64
	hydration     float64
	maxDurability float64
	useDurability float64
	priority      int
	instance      bool
	session       bool
	hideout       bool
	repairValue   float64
	fuelSeconds   int64
}

// seedSurvivalDefinitions 只为明确可使用的恢复品、维修包和燃料建立效果定义。
func seedSurvivalDefinitions(db *gorm.DB) error {
	effects := map[string]survivalEffect{
		// 自定义消耗品：继续支持事件和失能预设中的基础医疗包。
		"bandage": {hp: 20, maxDurability: 100, useDurability: 100, priority: 1, instance: true, session: true, hideout: true},
		"medkit":  {hp: 45, maxDurability: 100, useDurability: 100, priority: 2, instance: true, session: true, hideout: true},

		// Tarkov Wiki 医疗品：HPRecovery 是单次使用回复，耐久是整包治疗池。
		"ai2_medkit": {hp: 50, maxDurability: 100, useDurability: 50, priority: 3, instance: true, session: true, hideout: true},
		"ifak":       {hp: 50, maxDurability: 300, useDurability: 50, priority: 4, instance: true, session: true, hideout: true},
		"salewa":     {hp: 85, maxDurability: 400, useDurability: 85, priority: 5, instance: true, session: true, hideout: true},

		// Tarkov Wiki 食品和饮料：保留原版正负 Energy/Hydration 效果。
		"iskra":                  {energy: 80, priority: 1, session: true, hideout: true},
		"aquamari":               {energy: 20, hydration: 100, priority: 1, session: true, hideout: true},
		"squash_spread":          {energy: 40, priority: 2, session: true, hideout: true},
		"green_peas":             {energy: 35, hydration: 5, priority: 2, session: true, hideout: true},
		"humpback_salmon":        {energy: 50, hydration: -5, priority: 2, session: true, hideout: true},
		"pacific_saury":          {energy: 48, hydration: -2, priority: 2, session: true, hideout: true},
		"devildog_mayo":          {energy: 100, hydration: -99, priority: 2, session: true, hideout: true},
		"condensed_milk":         {energy: 75, hydration: -65, priority: 3, session: true, hideout: true},
		"alyonka":                {energy: 35, hydration: -15, priority: 3, session: true, hideout: true},
		"slickers":               {energy: 30, hydration: -15, priority: 3, session: true, hideout: true},
		"herring":                {energy: 47, hydration: -3, priority: 3, session: true, hideout: true},
		"tarcola":                {energy: 5, hydration: 15, priority: 3, session: true, hideout: true},
		"army_crackers":          {energy: 10, hydration: -5, priority: 4, session: true, hideout: true},
		"emergency_water_ration": {energy: 5, hydration: 50, priority: 4, session: true, hideout: true},
		"water_bottle":           {hydration: 60, priority: 4, session: true, hideout: true},

		"weapon_repair_kit_used": {repairValue: 100, maxDurability: 100, useDurability: 100, priority: 1, instance: true, hideout: true},
		"propane_tank":           {maxDurability: 100, useDurability: 100, fuelSeconds: 3600, priority: 1, instance: true, hideout: true},
		"metal_fuel_tank":        {maxDurability: 100, useDurability: 100, fuelSeconds: 7200, priority: 2, instance: true, hideout: true},
	}

	itemIDs := make([]string, 0, len(effects))
	for itemID := range effects {
		itemIDs = append(itemIDs, itemID)
	}
	if err := db.Where("item_id NOT IN ?", itemIDs).Delete(&models.ItemUseDef{}).Error; err != nil {
		return fmt.Errorf("清理过期物品效果: %w", err)
	}
	for itemID, effect := range effects {
		def := models.ItemUseDef{
			ItemID: itemID, HPRecovery: effect.hp, EnergyRecovery: effect.energy, HydrationRecovery: effect.hydration,
			RepairValue: effect.repairValue, FuelSeconds: effect.fuelSeconds, MaxDurability: effect.maxDurability,
			UseDurability: effect.useDurability, UsePriority: effect.priority, InstanceRequired: effect.instance,
			UsableInSession: effect.session, UsableInHideout: effect.hideout,
		}
		var stored models.ItemUseDef
		err := db.Where("item_id = ?", itemID).First(&stored).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&def).Error; err != nil {
				return fmt.Errorf("创建物品效果 %s: %w", itemID, err)
			}
		case err != nil:
			return fmt.Errorf("读取物品效果 %s: %w", itemID, err)
		default:
			if err := db.Model(&stored).Updates(def).Error; err != nil {
				return fmt.Errorf("更新物品效果 %s: %w", itemID, err)
			}
		}
	}
	return nil
}

func seedSurvivalForUser(db *gorm.DB, userID uint) error {
	var defs []models.ItemUseDef
	if err := db.Where("instance_required = ?", true).Find(&defs).Error; err != nil {
		return fmt.Errorf("读取实例物品定义: %w", err)
	}
	for _, def := range defs {
		var inventories []models.Inventory
		if err := db.Where("user_id = ? AND item_id = ?", userID, def.ItemID).Find(&inventories).Error; err != nil {
			return fmt.Errorf("读取实例物品库存 %s: %w", def.ItemID, err)
		}
		for _, inventory := range inventories {
			var existing int64
			if err := db.Model(&models.ItemInstance{}).Where("user_id = ? AND item_id = ? AND raid_extract = ?", userID, def.ItemID, inventory.RaidExtract).Count(&existing).Error; err != nil {
				return err
			}
			missing := inventory.Quantity - int(existing)
			for i := 0; i < missing; i++ {
				maxDurability := def.MaxDurability
				if maxDurability <= 0 {
					maxDurability = 100
				}
				if err := db.Create(&models.ItemInstance{UserID: userID, ItemID: def.ItemID, CurrentDurability: maxDurability, MaxDurability: maxDurability, Status: "normal", LocationType: "inventory", RaidExtract: inventory.RaidExtract, CreatedAt: time.Now(), UpdatedAt: time.Now()}).Error; err != nil {
					return fmt.Errorf("创建物品实例 %s: %w", def.ItemID, err)
				}
			}
			if err := db.Delete(&inventory).Error; err != nil {
				return fmt.Errorf("清理实例物品聚合库存 %s: %w", def.ItemID, err)
			}
		}
	}
	return nil
}
