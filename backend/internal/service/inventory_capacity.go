// 仓库容量辅助：计算装备占用、弹药格数和预设资源分配。
package service

import (
	"errors"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

// inventoryUsage 计算仓库已占用容量，扣除当前装备与 3 套预设装备（含补给）的占用。
func inventoryUsage(db *gorm.DB, userIDs ...uint) (int, error) {
	userID := models.DefaultUserID
	if len(userIDs) > 0 {
		userID = userIDs[0]
	}
	var rows []struct {
		ItemID   string
		Kind     string
		Quantity int
	}
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id <> ? AND quantity > 0", userID, "cash").
		Select("item_id, kind, quantity").Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("计算仓库容量: %w", err)
	}
	alloc, err := loadoutAllocatedItems(db, userID)
	if err != nil {
		return 0, err
	}
	allocatedInstances, err := loadoutAllocatedInstanceIDs(db, userID)
	if err != nil {
		return 0, err
	}
	stock := make(map[string]int, len(rows))
	kinds := make(map[string]string, len(rows))
	for _, row := range rows {
		stock[row.ItemID] += row.Quantity
		kinds[row.ItemID] = row.Kind
	}
	used := 0
	for itemID, quantity := range stock {
		if kinds[itemID] == "ammo" {
			var ammo models.AmmoDef
			if err := db.Select("rounds_per_slot").First(&ammo, "id = ?", itemID).Error; err != nil {
				return 0, fmt.Errorf("读取弹药容量 %s: %w", itemID, err)
			}
			if ammo.RoundsPerSlot <= 0 {
				return 0, fmt.Errorf("弹药 %s 的每格容量无效", itemID)
			}
			used += ceilDiv(quantity, ammo.RoundsPerSlot)
			continue
		}
		deduct := alloc[itemID]
		if deduct > quantity {
			deduct = quantity
		}
		used += quantity - deduct
	}
	var instances []models.ItemInstance
	if err := db.Where("user_id = ? AND location_type = ? AND status = ?", userID, "inventory", "normal").Find(&instances).Error; err != nil {
		return 0, fmt.Errorf("计算耐久物品容量: %w", err)
	}
	for _, instance := range instances {
		if _, allocated := allocatedInstances[instance.ID]; !allocated {
			used++
		}
	}
	return used, nil
}

func ceilDiv(value, divisor int) int {
	if value <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}

// loadoutAllocatedItems 统计当前装备与 3 套预设清单中每件物品各占用的仓库单位。
func loadoutAllocatedItems(db *gorm.DB, userID uint) (map[string]int, error) {
	loadout, err := GetPlayerLoadoutForUser(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]int{}, nil
		}
		return nil, err
	}
	alloc := make(map[string]int)
	add := func(ids []string) {
		for _, id := range ids {
			if id != "" {
				alloc[id]++
			}
		}
	}
	add([]string{loadout.WeaponID, loadout.ArmorID, loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID})
	add(loadout.Consumables)
	for i := 1; i <= 3; i++ {
		weaponID, armorID, consumables := PresetOf(loadout, i)
		base := []string{weaponID, armorID}
		base = append(base, presetEquipOf(loadout, i)...)
		add(base)
		add(consumables)
	}
	return alloc, nil
}

func loadoutAllocatedInstanceIDs(db *gorm.DB, userID uint) (map[uint]struct{}, error) {
	loadout, err := GetPlayerLoadoutForUser(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[uint]struct{}{}, nil
		}
		return nil, err
	}
	allocated := make(map[uint]struct{})
	type loadoutItems struct {
		ids  []string
		refs []models.LoadoutItemRef
	}
	sets := []loadoutItems{
		{ids: loadout.Consumables, refs: loadout.ConsumableRefs},
		{ids: loadout.PresetConsumables, refs: loadout.PresetConsumableRefs},
		{ids: loadout.Preset2Consumables, refs: loadout.Preset2ConsumableRefs},
		{ids: loadout.Preset3Consumables, refs: loadout.Preset3ConsumableRefs},
	}
	for _, set := range sets {
		if len(set.refs) > 0 {
			for _, ref := range set.refs {
				if ref.InstanceID > 0 {
					allocated[ref.InstanceID] = struct{}{}
				}
			}
			continue
		}
		for _, itemID := range set.ids {
			var use models.ItemUseDef
			if err := db.Where("item_id = ?", itemID).First(&use).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return nil, fmt.Errorf("读取补给效果 %s: %w", itemID, err)
			}
			if !use.InstanceRequired {
				continue
			}
			var instance models.ItemInstance
			query := db.Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, itemID, "inventory", "normal")
			if ids := allocatedIDs(allocated); len(ids) > 0 {
				query = query.Where("id NOT IN ?", ids)
			}
			if err := query.Order("current_durability asc, id asc").First(&instance).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return nil, fmt.Errorf("读取补给实例 %s: %w", itemID, err)
			}
			allocated[instance.ID] = struct{}{}
		}
	}
	return allocated, nil
}

func allocatedIDs(allocated map[uint]struct{}) []uint {
	ids := make([]uint, 0, len(allocated))
	for id := range allocated {
		ids = append(ids, id)
	}
	return ids
}

// presetEquipOf 返回第 N 套（1-3）预设新增装备（胸挂/背包/头盔/耳机）清单。
func presetEquipOf(loadout *models.PlayerLoadout, index int) []string {
	switch index {
	case 2:
		return []string{loadout.Preset2ChestRigID, loadout.Preset2BackpackID, loadout.Preset2HelmetID, loadout.Preset2HeadsetID}
	case 3:
		return []string{loadout.Preset3ChestRigID, loadout.Preset3BackpackID, loadout.Preset3HelmetID, loadout.Preset3HeadsetID}
	default:
		return []string{loadout.PresetChestRigID, loadout.PresetBackpackID, loadout.PresetHelmetID, loadout.PresetHeadsetID}
	}
}

// PresetOf 返回第 N 套（1-3）预设装备清单，供补购与容量计算使用。
func PresetOf(loadout *models.PlayerLoadout, index int) (weaponID, armorID string, consumables []string) {
	switch index {
	case 2:
		return loadout.Preset2WeaponID, loadout.Preset2ArmorID, loadout.Preset2Consumables
	case 3:
		return loadout.Preset3WeaponID, loadout.Preset3ArmorID, loadout.Preset3Consumables
	default:
		return loadout.PresetWeaponID, loadout.PresetArmorID, loadout.PresetConsumables
	}
}

// PresetAmmoOf 返回第 N 套恢复预设的弹药与携弹发数。
func PresetAmmoOf(loadout *models.PlayerLoadout, index int) (ammoID string, rounds int) {
	switch index {
	case 2:
		return loadout.Preset2AmmoID, loadout.Preset2AmmoRounds
	case 3:
		return loadout.Preset3AmmoID, loadout.Preset3AmmoRounds
	default:
		return loadout.PresetAmmoID, loadout.PresetAmmoRounds
	}
}

func uniqueItemIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
