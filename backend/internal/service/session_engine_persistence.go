// Session 结算持久化辅助：转换快照目录、库存掉落、角色状态和连续补给。
package service

import (
	"fmt"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

func snapshotCatalogItem(snapshot engine.ScenarioSnapshot, itemID string) (catalogItem, error) {
	definition, ok := snapshot.Items[itemID]
	if !ok {
		return catalogItem{}, fmt.Errorf("商品 %s 不在场景快照中", itemID)
	}
	item := catalogItem{ID: definition.ID, Name: definition.Name, Kind: definition.Kind, Category: definition.Category, Price: definition.Price, Weight: definition.Weight, Slots: definition.Slots, MerchantCategory: definition.MerchantCategory, RepRequirement: definition.RepRequirement, ArmorMax: definition.ArmorMax}
	if definition.Kind == "ammo" {
		ammo, ok := snapshot.Ammos[itemID]
		if !ok {
			return catalogItem{}, fmt.Errorf("弹药商品 %s 不在场景快照中", itemID)
		}
		item.RoundsPerSlot = ammo.RoundsPerSlot
		item.AmmoLevel = ammo.Level
	}
	return item, nil
}

func fitEngineLootToStorage(tx *gorm.DB, userID uint, loot []engine.LootDrop) ([]engine.LootDrop, []engine.LootDrop, error) {
	if len(loot) == 0 {
		return nil, nil, nil
	}
	used, err := inventoryUsage(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	space := models.InventoryCapacity - used
	if space < 0 {
		space = 0
	}
	stored := make([]engine.LootDrop, 0, len(loot))
	overflow := make([]engine.LootDrop, 0, len(loot))
	for _, drop := range loot {
		kept := drop.Quantity
		if kept > space {
			kept = space
		}
		if kept > 0 {
			copyDrop := drop
			copyDrop.Quantity = kept
			stored = append(stored, copyDrop)
			space -= kept
		}
		if drop.Quantity > kept {
			copyDrop := drop
			copyDrop.Quantity = drop.Quantity - kept
			overflow = append(overflow, copyDrop)
		}
	}
	return stored, overflow, nil
}

func consumeEngineResources(tx *gorm.DB, userID uint, result engine.RunResult) error {
	if result.SkipResourceConsumption {
		return nil
	}
	for _, item := range result.ConsumedItems {
		if err := removeInventoryItem(tx, userID, item.ItemID, item.Quantity); err != nil {
			return fmt.Errorf("扣除%s: %w", item.ItemID, err)
		}
	}
	return nil
}

func updateCharacterFromEngine(tx *gorm.DB, userID uint, characterID uint, state engine.CharacterState, injuryUntil *time.Time) error {
	updates := map[string]interface{}{
		"stress": state.Stress, "melee_prof": state.MeleeProf, "pistol_prof": state.PistolProf, "smg_prof": state.SMGProf,
		"shotgun_prof": state.ShotgunProf, "rifle_prof": state.RifleProf, "sniper_prof": state.SniperProf,
		"injury": state.Injury, "injury_until": injuryUntil,
	}
	if err := tx.Model(&models.Character{}).Where("user_id = ? AND id = ?", userID, characterID).Updates(updates).Error; err != nil {
		return fmt.Errorf("保存角色连续状态: %w", err)
	}
	return nil
}

func loadoutConsumableIDs(stacks []engine.ItemStack) []string {
	ids := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		for i := 0; i < stack.Quantity; i++ {
			ids = append(ids, stack.ItemID)
		}
	}
	return ids
}

func syncLoadoutConsumables(tx *gorm.DB, userID, characterID uint, stacks []engine.ItemStack) error {
	ids := loadoutConsumableIDs(stacks)
	if err := tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND character_id = ?", userID, characterID).
		Select("Consumables").Updates(&models.PlayerLoadout{Consumables: ids}).Error; err != nil {
		return fmt.Errorf("保存连续补给: %w", err)
	}
	return nil
}
