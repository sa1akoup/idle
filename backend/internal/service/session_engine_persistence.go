// Session 结算持久化辅助：转换快照目录、库存掉落、角色状态和连续补给。
package service

import (
	"fmt"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

// snapshotCatalogItem 从场景快照目录取出商品定义转为结算用目录项；弹药类额外附带每格容量与等级。
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
		item.Category = "ammo"
	}
	return item, nil
}

// fitEngineLootToStorage 按当前剩余容量把掉落拆分为入库与溢出两部分，容量不足的部分全部溢出。
func fitEngineLootToStorage(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, loot []engine.LootDrop) ([]engine.LootDrop, []engine.LootDrop, error) {
	if len(loot) == 0 {
		return nil, nil, nil
	}
	if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
		return nil, nil, err
	}
	// 先结算到期藏身处任务再统计占用，保证剩余空间口径一致后拆分入库与溢出。
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
	var ammoRows []struct {
		ItemID   string
		Quantity int
	}
	if err := tx.Model(&models.Inventory{}).
		Where("user_id = ? AND kind = ? AND quantity > 0", userID, "ammo").
		Select("item_id, quantity").Scan(&ammoRows).Error; err != nil {
		return nil, nil, fmt.Errorf("读取仓库弹药库存: %w", err)
	}
	ammoRounds := make(map[string]int, len(ammoRows))
	for _, row := range ammoRows {
		ammoRounds[row.ItemID] += row.Quantity
	}
	stored := make([]engine.LootDrop, 0, len(loot))
	overflow := make([]engine.LootDrop, 0, len(loot))
	for _, drop := range loot {
		if drop.Quantity <= 0 {
			continue
		}
		kept := 0
		if ammo, ok := snapshot.Ammos[drop.ItemID]; ok {
			if ammo.RoundsPerSlot <= 0 {
				return nil, nil, fmt.Errorf("弹药 %s 的每格容量无效", drop.ItemID)
			}
			currentRounds := ammoRounds[drop.ItemID]
			currentSlots := ceilDiv(currentRounds, ammo.RoundsPerSlot)
			availableSlots := capacity - used
			if availableSlots < 0 {
				availableSlots = 0
			}
			// 先填满已有半堆，再使用空余格；同类弹药的容量与 inventoryUsage 保持一致。
			freeRounds := currentSlots*ammo.RoundsPerSlot - currentRounds
			freeRounds += availableSlots * ammo.RoundsPerSlot
			kept = freeRounds
			if kept > drop.Quantity {
				kept = drop.Quantity
			}
			if kept < 0 {
				kept = 0
			}
			if kept > 0 {
				ammoRounds[drop.ItemID] += kept
				used += ceilDiv(ammoRounds[drop.ItemID], ammo.RoundsPerSlot) - currentSlots
			}
		} else {
			if _, ok := snapshot.LootItems[drop.ItemID]; !ok {
				return nil, nil, fmt.Errorf("掉落物品 %s 不在场景快照目录中", drop.ItemID)
			}
			kept = drop.Quantity
			if kept > space {
				kept = space
			}
			if kept > 0 {
				used += kept
			}
		}
		if kept > 0 {
			copyDrop := drop
			copyDrop.Quantity = kept
			stored = append(stored, copyDrop)
			space = capacity - used
			if space < 0 {
				space = 0
			}
		}
		if drop.Quantity > kept {
			copyDrop := drop
			copyDrop.Quantity = drop.Quantity - kept
			overflow = append(overflow, copyDrop)
		}
	}
	return stored, overflow, nil
}

// ensureInventoryWithinCapacityTx 结算完成后复查仓库容量，防止不同入库路径的口径漂移写入超额状态。
func ensureInventoryWithinCapacityTx(tx *gorm.DB, userID uint) error {
	used, err := inventoryUsage(tx, userID)
	if err != nil {
		return err
	}
	capacity, err := storageCapacityForUser(tx, userID)
	if err != nil {
		return err
	}
	if used > capacity {
		return fmt.Errorf("结算后仓库超过容量：%d/%d 格", used, capacity)
	}
	return nil
}

// refreshEngineCarryUsage 用纯引擎公式重算跨局状态中的当前配装占用，覆盖自动补给后的弹药变化。
func refreshEngineCarryUsage(state *engine.EngineState, snapshot engine.ScenarioSnapshot) error {
	usedSlots, usedWeight, err := engine.LoadoutUsage(snapshot, state.Loadout, state.Consumables, engine.CarriedAmmoStacks(state))
	if err != nil {
		return fmt.Errorf("计算 Session 携行容量: %w", err)
	}
	if usedSlots > state.Carry.TotalSlots || usedWeight > state.Carry.TotalWeight+1e-9 {
		return fmt.Errorf("Session 配装超过携行容量：%d/%d 格，%.1f/%.1fkg", usedSlots, state.Carry.TotalSlots, usedWeight, state.Carry.TotalWeight)
	}
	state.Carry.UsedSlots = usedSlots
	state.Carry.UsedWeight = usedWeight
	return nil
}

// updateCharacterFromEngine 把引擎结算后的角色连续状态（属性/生命/体力/饮水）写回角色表。
func updateCharacterFromEngine(tx *gorm.DB, userID uint, characterID uint, state engine.CharacterState) error {
	updates := map[string]interface{}{
		"strength": state.Strength, "agility": state.Agility, "intellect": state.Intellect, "charisma": state.Charisma,
		"stealth": state.Stealth, "perception": state.Perception, "negotiation": state.Negotiation, "luck": state.Luck,
		"survival": state.Survival, "resist": state.Resist, "engineering": state.Engineering, "medical": state.Medical,
		"stress": state.Stress, "melee_prof": state.MeleeProf, "pistol_prof": state.PistolProf, "smg_prof": state.SMGProf,
		"shotgun_prof": state.ShotgunProf, "rifle_prof": state.RifleProf, "sniper_prof": state.SniperProf,
		"hp": state.HP, "energy": state.Energy, "hydration": state.Hydration, "needs_updated_at": time.Now(),
	}
	if err := tx.Model(&models.Character{}).Where("user_id = ? AND id = ?", userID, characterID).Updates(updates).Error; err != nil {
		return fmt.Errorf("保存角色连续状态: %w", err)
	}
	return nil
}

// loadoutConsumableIDs 把补给堆叠按数量展开为物品 ID 列表，供配装持久化使用。
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
