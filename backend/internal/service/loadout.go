package service

// 角色装备配置服务：校验当前库存、保存携行方案，并处理失能后的丢失与预设补购。

import (
	"errors"
	"fmt"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// GetPlayerLoadoutForUser 读取指定用户的装备配置。
func GetPlayerLoadoutForUser(db *gorm.DB, userID uint) (*models.PlayerLoadout, error) {
	var loadout models.PlayerLoadout
	if err := db.Where("user_id = ?", userID).First(&loadout).Error; err != nil {
		return nil, fmt.Errorf("读取角色装备配置: %w", err)
	}
	return &loadout, nil
}

// SavePlayerLoadoutForUser 保存指定用户的装备配置。
func SavePlayerLoadoutForUser(db *gorm.DB, userID uint, req SaveLoadoutReq) (*models.PlayerLoadout, error) {
	req.Consumables = uniqueItemIDs(req.Consumables)
	req.PresetConsumables = uniqueItemIDs(req.PresetConsumables)
	req.Preset2Consumables = uniqueItemIDs(req.Preset2Consumables)
	req.Preset3Consumables = uniqueItemIDs(req.Preset3Consumables)
	for i, consumables := range [][]string{req.Consumables, req.PresetConsumables, req.Preset2Consumables, req.Preset3Consumables} {
		if len(consumables) > 4 {
			return nil, fmt.Errorf("第 %d 套装备补给最多选择 4 种", i+1)
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		if err := ensureLoadoutMutationAllowed(tx, userID); err != nil {
			return err
		}
		if err := validateLoadoutCatalog(tx, req.WeaponID, req.ArmorID, req.Consumables,
			req.ChestRigID, req.BackpackID, req.HelmetID, req.HeadsetID); err != nil {
			return fmt.Errorf("当前装备无效: %w", err)
		}
		if err := validateOwnedLoadoutForUser(tx, userID, req.WeaponID, req.ArmorID, req.Consumables,
			req.ChestRigID, req.BackpackID, req.HelmetID, req.HeadsetID); err != nil {
			return err
		}
		for i := 1; i <= 3; i++ {
			weaponID, armorID, consumables, equip := presetOfReq(req, i)
			ammoID, ammoRounds := presetAmmoOfReq(req, i)
			if weaponID == "" && armorID == "" && len(consumables) == 0 && allEmpty(equip) && ammoID == "" && ammoRounds == 0 {
				continue // 空预设允许，视为未配置
			}
			if weaponID == "" || armorID == "" {
				return fmt.Errorf("预设装备 %d 需同时配置武器与护甲", i)
			}
			if err := validateLoadoutCatalog(tx, weaponID, armorID, consumables,
				equip[0], equip[1], equip[2], equip[3]); err != nil {
				return fmt.Errorf("预设装备 %d 无效: %w", i, err)
			}
			if err := validatePresetAmmoCatalog(tx, userID, weaponID, ammoID, ammoRounds); err != nil {
				return fmt.Errorf("预设装备 %d 的弹药无效: %w", i, err)
			}
		}

		updates := models.PlayerLoadout{
			WeaponID: req.WeaponID, ArmorID: req.ArmorID,
			ChestRigID: req.ChestRigID, BackpackID: req.BackpackID, HelmetID: req.HelmetID, HeadsetID: req.HeadsetID,
			Consumables: req.Consumables, ConsumableRefs: []models.LoadoutItemRef{},
			PresetWeaponID: req.PresetWeaponID, PresetArmorID: req.PresetArmorID,
			PresetChestRigID: req.PresetChestRigID, PresetBackpackID: req.PresetBackpackID,
			PresetHelmetID: req.PresetHelmetID, PresetHeadsetID: req.PresetHeadsetID,
			PresetName: req.PresetName, PresetConsumables: req.PresetConsumables, PresetConsumableRefs: []models.LoadoutItemRef{}, PresetAmmoID: req.PresetAmmoID, PresetAmmoRounds: req.PresetAmmoRounds,
			Preset2WeaponID: req.Preset2WeaponID, Preset2ArmorID: req.Preset2ArmorID,
			Preset2ChestRigID: req.Preset2ChestRigID, Preset2BackpackID: req.Preset2BackpackID,
			Preset2HelmetID: req.Preset2HelmetID, Preset2HeadsetID: req.Preset2HeadsetID,
			Preset2Name: req.Preset2Name, Preset2Consumables: req.Preset2Consumables, Preset2ConsumableRefs: []models.LoadoutItemRef{}, Preset2AmmoID: req.Preset2AmmoID, Preset2AmmoRounds: req.Preset2AmmoRounds,
			Preset3WeaponID: req.Preset3WeaponID, Preset3ArmorID: req.Preset3ArmorID,
			Preset3ChestRigID: req.Preset3ChestRigID, Preset3BackpackID: req.Preset3BackpackID,
			Preset3HelmetID: req.Preset3HelmetID, Preset3HeadsetID: req.Preset3HeadsetID,
			Preset3Name: req.Preset3Name, Preset3Consumables: req.Preset3Consumables, Preset3ConsumableRefs: []models.LoadoutItemRef{}, Preset3AmmoID: req.Preset3AmmoID, Preset3AmmoRounds: req.Preset3AmmoRounds,
		}
		var player models.Character
		if err := tx.Where("user_id = ?", userID).First(&player).Error; err != nil {
			return fmt.Errorf("读取角色: %w", err)
		}
		if err := tx.Model(&models.PlayerLoadout{}).
			Where("user_id = ? AND character_id = ?", userID, player.ID).
			Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables", "ConsumableRefs",
				"PresetWeaponID", "PresetArmorID", "PresetChestRigID", "PresetBackpackID", "PresetHelmetID", "PresetHeadsetID", "PresetName", "PresetConsumables", "PresetConsumableRefs", "PresetAmmoID", "PresetAmmoRounds",
				"Preset2WeaponID", "Preset2ArmorID", "Preset2ChestRigID", "Preset2BackpackID", "Preset2HelmetID", "Preset2HeadsetID", "Preset2Name", "Preset2Consumables", "Preset2ConsumableRefs", "Preset2AmmoID", "Preset2AmmoRounds",
				"Preset3WeaponID", "Preset3ArmorID", "Preset3ChestRigID", "Preset3BackpackID", "Preset3HelmetID", "Preset3HeadsetID", "Preset3Name", "Preset3Consumables", "Preset3ConsumableRefs", "Preset3AmmoID", "Preset3AmmoRounds").
			Updates(&updates).Error; err != nil {
			return fmt.Errorf("保存角色装备配置: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return GetPlayerLoadoutForUser(db, userID)
}

// validatePresetAmmoCatalog 校验预设弹药的携弹量与口径，并核对商人好感度解锁。
func validatePresetAmmoCatalog(db *gorm.DB, userID uint, weaponID, ammoID string, rounds int) error {
	var weapon models.WeaponDef
	if err := db.First(&weapon, "id = ?", weaponID).Error; err != nil {
		return fmt.Errorf("读取武器: %w", err)
	}
	if weapon.AmmoPerRound <= 0 {
		if ammoID != "" || rounds != 0 {
			return fmt.Errorf("近战武器不能配置弹药")
		}
		return nil
	}
	if rounds < weapon.AmmoPerRound || rounds > 9999 {
		return fmt.Errorf("携弹量需为 %d-9999 发", weapon.AmmoPerRound)
	}
	var ammo models.AmmoDef
	if err := db.First(&ammo, "id = ?", ammoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("弹药不存在")
		}
		return fmt.Errorf("读取弹药: %w", err)
	}
	if ammo.CaliberID != weapon.CaliberID {
		return fmt.Errorf("弹药口径 %s 与武器口径 %s 不匹配", ammo.CaliberID, weapon.CaliberID)
	}
	if ammo.Level > 4 {
		return fmt.Errorf("武器商人最高只出售 N4 弹药（%s 为 N%d）", ammo.Name, ammo.Level)
	}
	merchant, err := GetMerchantByIDForUser(db, userID, ammo.MerchantCategory)
	if err != nil {
		return err
	}
	if merchant.Open && ammo.RepRequirement > merchant.Reputation {
		return fmt.Errorf("好感度不足，无法使用 %s（需好感度 %d，当前 %d）", ammo.Name, ammo.RepRequirement, merchant.Reputation)
	}
	return nil
}

// validateLoadoutCatalog 校验装备与补给条目在目录中存在且可用，保证装备方案合法。
func validateLoadoutCatalog(db *gorm.DB, weaponID, armorID string, consumables []string, chestRigID, backpackID, helmetID, headsetID string) error {
	var count int64
	if err := db.Model(&models.WeaponDef{}).Where("id = ?", weaponID).Count(&count).Error; err != nil {
		return fmt.Errorf("读取武器: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("武器不存在")
	}
	if err := db.Model(&models.ArmorDef{}).Where("id = ?", armorID).Count(&count).Error; err != nil {
		return fmt.Errorf("读取护甲: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("护甲不存在")
	}
	equipChecks := []struct {
		id    string
		name  string
		model interface{}
	}{
		{chestRigID, "胸挂", &models.ChestRigDef{}},
		{backpackID, "背包", &models.BackpackDef{}},
		{helmetID, "头盔", &models.HelmetDef{}},
		{headsetID, "耳机", &models.HeadsetDef{}},
	}
	for _, ec := range equipChecks {
		if ec.id == "" {
			continue
		}
		if err := db.Model(ec.model).Where("id = ?", ec.id).Count(&count).Error; err != nil {
			return fmt.Errorf("读取%s: %w", ec.name, err)
		}
		if count != 1 {
			return fmt.Errorf("%s不存在", ec.name)
		}
	}
	for _, itemID := range consumables {
		if err := db.Model(&models.ConsumableDef{}).Where("id = ?", itemID).Count(&count).Error; err != nil {
			return fmt.Errorf("读取补给: %w", err)
		}
		if count == 1 {
			continue
		}
		var loot models.LootItemDef
		if err := db.Where("id = ?", itemID).First(&loot).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("补给 %s 不存在", itemID)
			}
			return fmt.Errorf("读取补给 %s: %w", itemID, err)
		}
		var use models.ItemUseDef
		if err := db.Where("item_id = ?", itemID).First(&use).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("补给 %s 未配置使用效果", itemID)
			}
			return fmt.Errorf("读取补给效果 %s: %w", itemID, err)
		}
		if !use.UsableInSession {
			return fmt.Errorf("补给 %s 不能带入行动", loot.Name)
		}
	}
	return nil
}

// validateOwnedLoadoutForUser 校验用户仓库确实拥有所选装备、补给与可用护甲实例。
func validateOwnedLoadoutForUser(db *gorm.DB, userID uint, weaponID, armorID string, consumables []string, chestRigID, backpackID, helmetID, headsetID string) error {
	ids := []string{weaponID, armorID}
	for _, id := range []string{chestRigID, backpackID, helmetID, headsetID} {
		if id != "" {
			ids = append(ids, id)
		}
	}
	for _, itemID := range ids {
		var inventory models.Inventory
		if err := db.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).First(&inventory).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("仓库中缺少装备 %s", itemID)
			}
			return fmt.Errorf("读取装备库存 %s: %w", itemID, err)
		}
	}
	for _, itemID := range consumables {
		var use models.ItemUseDef
		if err := db.Where("item_id = ?", itemID).First(&use).Error; err == nil && use.InstanceRequired {
			var instanceCount int64
			// 需实例化的补给须在仓库存在未损坏实例，凭实例数而非库存量校验
			if err := db.Model(&models.ItemInstance{}).
				Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, itemID, "inventory", "normal").
				Count(&instanceCount).Error; err != nil {
				return fmt.Errorf("读取补给实例 %s: %w", itemID, err)
			}
			if instanceCount == 0 {
				return fmt.Errorf("仓库中缺少可用补给 %s", itemID)
			}
			continue
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("读取补给效果 %s: %w", itemID, err)
		}
		var inventory models.Inventory
		if err := db.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).First(&inventory).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("仓库中缺少补给 %s", itemID)
			}
			return fmt.Errorf("读取补给库存 %s: %w", itemID, err)
		}
	}
	var armorInstance models.ArmorInstance
	if err := db.Where("user_id = ? AND armor_id = ? AND status = ?", userID, armorID, "normal").First(&armorInstance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("护甲 %s 已损坏或没有可用实例", armorID)
		}
		return fmt.Errorf("读取护甲耐久 %s: %w", armorID, err)
	}
	return nil
}

// ReplaceLostLoadoutForUser 处理指定用户失能后的丢失装备与预设补购。
func ReplaceLostLoadoutForUser(db *gorm.DB, userID uint, presetIndex int) (int, error) {
	var lostLoadout models.PlayerLoadout
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		loadout, err := GetPlayerLoadoutForUser(tx, userID)
		if err != nil {
			return err
		}
		lostLoadout = *loadout

		lostIDs := append([]string{loadout.WeaponID, loadout.ArmorID}, loadout.Consumables...)
		for _, id := range []string{loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID} {
			if id != "" {
				lostIDs = append(lostIDs, id)
			}
		}
		for _, itemID := range uniqueItemIDs(lostIDs) {
			if err := removeInventoryItem(tx, userID, itemID, 1); err != nil {
				return err
			}
		}
		var armorInstance models.ArmorInstance
		if err := tx.Where("user_id = ? AND armor_id = ?", userID, loadout.ArmorID).Order("id asc").First(&armorInstance).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("读取丢失护甲: %w", err)
			}
		} else if err := tx.Delete(&armorInstance).Error; err != nil {
			return fmt.Errorf("移除丢失护甲: %w", err)
		}

		// 丢失后清空当前装备方案（护甲实例已删，保留预设待补购）

		cleared := models.PlayerLoadout{Consumables: []string{}, ConsumableRefs: []models.LoadoutItemRef{}}
		if err := tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND id = ?", userID, loadout.ID).
			Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables", "ConsumableRefs").
			Updates(&cleared).Error; err != nil {
			return fmt.Errorf("清空丢失装备: %w", err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	presetWeaponID, presetArmorID, presetConsumables := PresetOf(&lostLoadout, presetIndex)
	presetEquip := presetEquipOf(&lostLoadout, presetIndex)
	if presetWeaponID == "" || presetArmorID == "" {
		return 0, fmt.Errorf("预设装备 %d 未配置，无法恢复", presetIndex)
	}

	paid := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		presetIDs := append([]string{presetWeaponID, presetArmorID}, presetConsumables...)
		for _, id := range presetEquip {
			if id != "" {
				presetIDs = append(presetIDs, id)
			}
		}
		catalogRepo := catalog.New(tx)
		uniquePresetIDs := uniqueItemIDs(presetIDs)
		catalogItems, err := catalogRepo.FindByIDs(uniquePresetIDs)
		if err != nil {
			return err
		}
		items := make([]catalogItem, 0, len(uniquePresetIDs))
		for _, itemID := range uniquePresetIDs {
			item, ok := catalogItems[itemID]
			if !ok {
				return fmt.Errorf("读取预设商品 %s: %w", itemID, catalog.ErrItemNotFound)
			}
			// 与商人联动：按归属商人的好感度折算价格，并校验解锁
			if err := applyMerchantPriceForUser(tx, userID, &item); err != nil {
				return fmt.Errorf("%w：预设装备补购失败（%v）", ErrPurchaseUnavailable, err)
			}
			items = append(items, item)
		}
		paid, err = purchaseCatalogItems(tx, userID, items)
		if err != nil {
			return err
		}

		updates := models.PlayerLoadout{
			WeaponID: presetWeaponID, ArmorID: presetArmorID,
			ChestRigID: presetEquip[0], BackpackID: presetEquip[1], HelmetID: presetEquip[2], HeadsetID: presetEquip[3],
			Consumables: presetConsumables, ConsumableRefs: []models.LoadoutItemRef{},
		}
		if err := tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND id = ?", userID, lostLoadout.ID).
			Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables", "ConsumableRefs").
			Updates(&updates).Error; err != nil {
			return fmt.Errorf("启用补购装备: %w", err)
		}
		return nil
	})
	return paid, err
}
