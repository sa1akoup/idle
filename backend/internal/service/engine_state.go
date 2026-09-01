// 引擎状态适配层：将数据库角色、装备、容量和资源转换为纯 DTO。
package service

import (
	"errors"
	"fmt"
	"sort"

	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// characterToEngineState 将数据库角色字段映射为引擎所需的无状态 DTO。
func characterToEngineState(character models.Character) engine.CharacterState {
	return engine.CharacterState{
		Name: character.Name, Strength: character.Strength, Agility: character.Agility, Intellect: character.Intellect,
		Charisma: character.Charisma, Stealth: character.Stealth, Perception: character.Perception, Negotiation: character.Negotiation,
		Luck: character.Luck, Survival: character.Survival, Resist: character.Resist, Engineering: character.Engineering, Medical: character.Medical,
		MeleeProf: character.MeleeProf, PistolProf: character.PistolProf, SMGProf: character.SMGProf, ShotgunProf: character.ShotgunProf,
		RifleProf: character.RifleProf, SniperProf: character.SniperProf,
		HP: character.HP, Energy: character.Energy, Hydration: character.Hydration, Stress: character.Stress,
	}
}

// loadoutToEngineState 提取装备配置中的当前装备位字段构成加载 DTO。
func loadoutToEngineState(loadout *models.PlayerLoadout) engine.LoadoutState {
	return engine.LoadoutState{
		WeaponID: loadout.WeaponID, ArmorID: loadout.ArmorID, ChestRigID: loadout.ChestRigID, BackpackID: loadout.BackpackID,
		ArmorInstanceID: loadout.ArmorInstanceID, HelmetID: loadout.HelmetID, HeadsetID: loadout.HeadsetID,
	}
}

// itemIDsToStacks 把物品 ID 列表转为每件数量为 1 的堆栈，并跳过空 ID。
func itemIDsToStacks(ids []string) []engine.ItemStack {
	stacks := make([]engine.ItemStack, 0, len(ids))
	for _, itemID := range ids {
		if itemID != "" {
			stacks = append(stacks, engine.ItemStack{ItemID: itemID, Quantity: 1})
		}
	}
	return stacks
}

// buildEngineState 汇总角色、装备、护甲耐久、携行物品与弹药，组装出完整引擎状态。
func buildEngineState(db *gorm.DB, userID uint, character models.Character, loadout *models.PlayerLoadout, ammo engine.CarriedAmmo) (engine.EngineState, error) {
	carry, err := carryCapacityCore(db, userID, loadout, ammo.ID, ammo.Rounds)
	if err != nil {
		return engine.EngineState{}, fmt.Errorf("读取探索携行状态: %w", err)
	}
	if carry.UsedSlots > carry.TotalSlots || carry.UsedWeight > carry.TotalWeight+1e-9 {
		return engine.EngineState{}, fmt.Errorf("当前配装超过携行容量：%d/%d 格，%.1f/%.1fkg", carry.UsedSlots, carry.TotalSlots, carry.UsedWeight, carry.TotalWeight)
	}
	armorInstance, err := findCurrentArmorInstance(db, userID, loadout.ArmorID)
	if err != nil {
		return engine.EngineState{}, err
	}
	loadoutState := loadoutToEngineState(loadout)
	loadoutState.ArmorInstanceID = armorInstance.ID
	carriedItems, err := carriedItemsForLoadout(db, userID, loadout)
	if err != nil {
		return engine.EngineState{}, err
	}
	return engine.EngineState{
		Character: characterToEngineState(character), Loadout: loadoutState,
		ArmorDurability: armorInstance.CurDurability, Consumables: itemStacksFromCarriedItems(carriedItems), CarriedItems: carriedItems,
		Ammo:  ammo,
		Carry: engine.CarryState{TotalSlots: carry.TotalSlots, UsedSlots: carry.UsedSlots, TotalWeight: carry.TotalWeight, UsedWeight: carry.UsedWeight},
	}, nil
}

// findCurrentArmorInstance 取该用户当前护甲的第一个正常实例，用于读取出征耐久。
func findCurrentArmorInstance(db *gorm.DB, userID uint, armorID string) (*models.ArmorInstance, error) {
	var instance models.ArmorInstance
	if err := db.Where("user_id = ? AND armor_id = ? AND status = ?", userID, armorID, "normal").Order("id asc").First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("护甲 %s 已损坏或没有可用实例", armorID)
		}
		return nil, fmt.Errorf("读取护甲耐久: %w", err)
	}
	return &instance, nil
}

// attachRecoveryPresets 把三套补购预设（装备、弹药、补给及其商人价格）挂到场景快照上。
func attachRecoveryPresets(db *gorm.DB, userID uint, loadout *models.PlayerLoadout, snapshot *engine.ScenarioSnapshot) error {
	if loadout == nil {
		var err error
		loadout, err = GetPlayerLoadoutForUser(db, userID)
		if err != nil {
			return err
		}
	}
	snapshot.RecoveryPresets = make(map[int]engine.RecoveryPreset, 3)
	catalogRepo := catalog.New(db)
	for index := 1; index <= 3; index++ {
		weaponID, armorID, consumables := PresetOf(loadout, index)
		ammoID, ammoRounds := PresetAmmoOf(loadout, index)
		equip := presetEquipOf(loadout, index)
		ids := []string{weaponID, armorID, equip[0], equip[1], equip[2], equip[3]}
		ids = append(ids, consumables...)
		quantities := make(map[string]int, len(ids))
		for _, itemID := range ids {
			if itemID != "" {
				quantities[itemID]++
			}
		}
		itemIDs := make([]string, 0, len(quantities))
		for itemID := range quantities {
			itemIDs = append(itemIDs, itemID)
		}
		sort.Strings(itemIDs)
		catalogItems, err := catalogRepo.FindByIDs(itemIDs)
		if err != nil {
			return fmt.Errorf("读取补购商品目录: %w", err)
		}
		preset := engine.RecoveryPreset{
			Index: index, AmmoID: ammoID, AmmoRounds: ammoRounds,
			Loadout: engine.LoadoutState{
				WeaponID: weaponID, ArmorID: armorID, ChestRigID: equip[0], BackpackID: equip[1], HelmetID: equip[2], HeadsetID: equip[3],
			},
			Consumables: itemIDsToStacks(consumables),
		}
		engine.SortItemStacks(preset.Consumables)
		for _, itemID := range itemIDs {
			item, ok := catalogItems[itemID]
			if !ok {
				return fmt.Errorf("读取补购商品 %s: %w", itemID, catalog.ErrItemNotFound)
			}
			available := true
			if err := applyMerchantPriceForUser(db, userID, &item); err != nil {
				if !errors.Is(err, ErrMerchantUnavailable) {
					return fmt.Errorf("计算补购商品 %s 价格: %w", itemID, err)
				}
				available = false
			}
			unitPrice := item.Price
			// 商人未解锁时商品仍计入预设但标记不可购买，供界面显示缺省回退价
			if item.PaidPrice > 0 {
				unitPrice = item.PaidPrice
			}
			preset.Items = append(preset.Items, engine.RecoveryItem{ItemID: itemID, Quantity: quantities[itemID], UnitPrice: unitPrice, Available: available})
		}
		snapshot.RecoveryPresets[index] = preset
	}
	return nil
}

// finalizeScenarioSnapshot 对场景快照做校验后输出规范 JSON 与 hash，供持久化与校验使用。
func finalizeScenarioSnapshot(snapshot engine.ScenarioSnapshot) (string, string, error) {
	if err := engine.ValidateSnapshot(snapshot); err != nil {
		return "", "", fmt.Errorf("校验场景快照: %w", err)
	}
	encoded, err := engine.CanonicalSnapshotJSON(snapshot)
	if err != nil {
		return "", "", fmt.Errorf("序列化场景快照: %w", err)
	}
	hash, err := engine.SnapshotHash(snapshot)
	if err != nil {
		return "", "", fmt.Errorf("计算场景快照 hash: %w", err)
	}
	return string(encoded), hash, nil
}
