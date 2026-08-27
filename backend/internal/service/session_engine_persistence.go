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
	if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
		return nil, nil, err
	}
	used, err := inventoryUsage(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	capacity, err := storageCapacityForUser(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	space := capacity - used
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

func updateCharacterFromEngine(tx *gorm.DB, userID uint, characterID uint, state engine.CharacterState) error {
	updates := map[string]interface{}{
		"stress": state.Stress, "melee_prof": state.MeleeProf, "pistol_prof": state.PistolProf, "smg_prof": state.SMGProf,
		"shotgun_prof": state.ShotgunProf, "rifle_prof": state.RifleProf, "sniper_prof": state.SniperProf,
		"hp": state.HP, "energy": state.Energy, "hydration": state.Hydration, "needs_updated_at": time.Now(),
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

// syncLoadoutConsumablesFromCarriedItemsTx 将终局仍携带的普通补给和耐久实例写回当前装备配置。
func syncLoadoutConsumablesFromCarriedItemsTx(tx *gorm.DB, userID, characterID uint, items []engine.CarriedItem) error {
	ids := make([]string, 0, len(items))
	refs := make([]models.LoadoutItemRef, 0, len(items))
	for _, item := range items {
		if item.ItemID == "" {
			continue
		}
		if item.InstanceID > 0 {
			if item.CurrentDurability <= 0 {
				continue
			}
			ids = append(ids, item.ItemID)
			refs = append(refs, models.LoadoutItemRef{InstanceID: item.InstanceID, ItemID: item.ItemID, Quantity: 1})
			continue
		}
		if item.Quantity <= 0 {
			continue
		}
		for i := 0; i < item.Quantity; i++ {
			ids = append(ids, item.ItemID)
		}
		refs = append(refs, models.LoadoutItemRef{ItemID: item.ItemID, Quantity: item.Quantity})
	}
	if err := tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND character_id = ?", userID, characterID).
		Select("Consumables", "ConsumableRefs").Updates(&models.PlayerLoadout{Consumables: ids, ConsumableRefs: refs}).Error; err != nil {
		return fmt.Errorf("保存终局补给配置: %w", err)
	}
	return nil
}
