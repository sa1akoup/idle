package service

// 角色装备配置服务：校验当前库存、保存携行方案，并处理失能后的丢失与预设补购。

import (
	"errors"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

// SaveLoadoutReq 保存当前装备和 3 套失能后补购预设。
type SaveLoadoutReq struct {
	WeaponID           string   `json:"weaponId" binding:"required"`
	ArmorID            string   `json:"armorId" binding:"required"`
	ChestRigID         string   `json:"chestRigId"`
	BackpackID         string   `json:"backpackId"`
	HelmetID           string   `json:"helmetId"`
	HeadsetID          string   `json:"headsetId"`
	Consumables        []string `json:"consumables"`
	PresetWeaponID     string   `json:"presetWeaponId"`
	PresetArmorID      string   `json:"presetArmorId"`
	PresetChestRigID   string   `json:"presetChestRigId"`
	PresetBackpackID   string   `json:"presetBackpackId"`
	PresetHelmetID     string   `json:"presetHelmetId"`
	PresetHeadsetID    string   `json:"presetHeadsetId"`
	PresetConsumables  []string `json:"presetConsumables"`
	PresetName         string   `json:"presetName"`
	Preset2WeaponID    string   `json:"preset2WeaponId"`
	Preset2ArmorID     string   `json:"preset2ArmorId"`
	Preset2ChestRigID  string   `json:"preset2ChestRigId"`
	Preset2BackpackID  string   `json:"preset2BackpackId"`
	Preset2HelmetID    string   `json:"preset2HelmetId"`
	Preset2HeadsetID   string   `json:"preset2HeadsetId"`
	Preset2Consumables []string `json:"preset2Consumables"`
	Preset2Name        string   `json:"preset2Name"`
	Preset3WeaponID    string   `json:"preset3WeaponId"`
	Preset3ArmorID     string   `json:"preset3ArmorId"`
	Preset3ChestRigID  string   `json:"preset3ChestRigId"`
	Preset3BackpackID  string   `json:"preset3BackpackId"`
	Preset3HelmetID    string   `json:"preset3HelmetId"`
	Preset3HeadsetID   string   `json:"preset3HeadsetId"`
	Preset3Consumables []string `json:"preset3Consumables"`
	Preset3Name        string   `json:"preset3Name"`
}

func GetPlayerLoadout(db *gorm.DB) (*models.PlayerLoadout, error) {
	var loadout models.PlayerLoadout
	if err := db.First(&loadout, models.PlayerLoadoutID).Error; err != nil {
		return nil, fmt.Errorf("读取角色装备配置: %w", err)
	}
	return &loadout, nil
}

func SavePlayerLoadout(db *gorm.DB, req SaveLoadoutReq) (*models.PlayerLoadout, error) {
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
		if err := validateLoadoutCatalog(tx, req.WeaponID, req.ArmorID, req.Consumables,
			req.ChestRigID, req.BackpackID, req.HelmetID, req.HeadsetID); err != nil {
			return fmt.Errorf("当前装备无效: %w", err)
		}
		if err := validateOwnedLoadout(tx, req.WeaponID, req.ArmorID, req.Consumables,
			req.ChestRigID, req.BackpackID, req.HelmetID, req.HeadsetID); err != nil {
			return err
		}
		for i := 1; i <= 3; i++ {
			weaponID, armorID, consumables, equip := presetOfReq(req, i)
			if weaponID == "" && armorID == "" && len(consumables) == 0 && len(equip) == 0 {
				continue // 空预设允许，视为未配置
			}
			if weaponID == "" || armorID == "" {
				return fmt.Errorf("预设装备 %d 需同时配置武器与护甲", i)
			}
			if err := validateLoadoutCatalog(tx, weaponID, armorID, consumables,
				equip[0], equip[1], equip[2], equip[3]); err != nil {
				return fmt.Errorf("预设装备 %d 无效: %w", i, err)
			}
		}

		updates := models.PlayerLoadout{
			WeaponID: req.WeaponID, ArmorID: req.ArmorID,
			ChestRigID: req.ChestRigID, BackpackID: req.BackpackID, HelmetID: req.HelmetID, HeadsetID: req.HeadsetID,
			Consumables:    req.Consumables,
			PresetWeaponID: req.PresetWeaponID, PresetArmorID: req.PresetArmorID,
			PresetChestRigID: req.PresetChestRigID, PresetBackpackID: req.PresetBackpackID,
			PresetHelmetID: req.PresetHelmetID, PresetHeadsetID: req.PresetHeadsetID,
			PresetName: req.PresetName, PresetConsumables: req.PresetConsumables,
			Preset2WeaponID: req.Preset2WeaponID, Preset2ArmorID: req.Preset2ArmorID,
			Preset2ChestRigID: req.Preset2ChestRigID, Preset2BackpackID: req.Preset2BackpackID,
			Preset2HelmetID: req.Preset2HelmetID, Preset2HeadsetID: req.Preset2HeadsetID,
			Preset2Name: req.Preset2Name, Preset2Consumables: req.Preset2Consumables,
			Preset3WeaponID: req.Preset3WeaponID, Preset3ArmorID: req.Preset3ArmorID,
			Preset3ChestRigID: req.Preset3ChestRigID, Preset3BackpackID: req.Preset3BackpackID,
			Preset3HelmetID: req.Preset3HelmetID, Preset3HeadsetID: req.Preset3HeadsetID,
			Preset3Name: req.Preset3Name, Preset3Consumables: req.Preset3Consumables,
		}
		if err := tx.Model(&models.PlayerLoadout{}).
			Where("id = ?", models.PlayerLoadoutID).
			Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables",
				"PresetWeaponID", "PresetArmorID", "PresetChestRigID", "PresetBackpackID", "PresetHelmetID", "PresetHeadsetID", "PresetName", "PresetConsumables",
				"Preset2WeaponID", "Preset2ArmorID", "Preset2ChestRigID", "Preset2BackpackID", "Preset2HelmetID", "Preset2HeadsetID", "Preset2Name", "Preset2Consumables",
				"Preset3WeaponID", "Preset3ArmorID", "Preset3ChestRigID", "Preset3BackpackID", "Preset3HelmetID", "Preset3HeadsetID", "Preset3Name", "Preset3Consumables").
			Updates(&updates).Error; err != nil {
			return fmt.Errorf("保存角色装备配置: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return GetPlayerLoadout(db)
}

// PresetNameOf 返回第 N 套（1-3）预设名称。
func PresetNameOf(loadout *models.PlayerLoadout, index int) string {
	switch index {
	case 2:
		return loadout.Preset2Name
	case 3:
		return loadout.Preset3Name
	default:
		return loadout.PresetName
	}
}

func presetOfReq(req SaveLoadoutReq, index int) (weaponID, armorID string, consumables []string, equip []string) {
	switch index {
	case 2:
		return req.Preset2WeaponID, req.Preset2ArmorID, req.Preset2Consumables,
			[]string{req.Preset2ChestRigID, req.Preset2BackpackID, req.Preset2HelmetID, req.Preset2HeadsetID}
	case 3:
		return req.Preset3WeaponID, req.Preset3ArmorID, req.Preset3Consumables,
			[]string{req.Preset3ChestRigID, req.Preset3BackpackID, req.Preset3HelmetID, req.Preset3HeadsetID}
	default:
		return req.PresetWeaponID, req.PresetArmorID, req.PresetConsumables,
			[]string{req.PresetChestRigID, req.PresetBackpackID, req.PresetHelmetID, req.PresetHeadsetID}
	}
}

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
		if count != 1 {
			return fmt.Errorf("补给 %s 不存在", itemID)
		}
	}
	return nil
}

func validateOwnedLoadout(db *gorm.DB, weaponID, armorID string, consumables []string, chestRigID, backpackID, helmetID, headsetID string) error {
	ids := []string{weaponID, armorID}
	ids = append(ids, consumables...)
	for _, id := range []string{chestRigID, backpackID, helmetID, headsetID} {
		if id != "" {
			ids = append(ids, id)
		}
	}
	for _, itemID := range ids {
		var inventory models.Inventory
		if err := db.Where("item_id = ? AND quantity > 0", itemID).First(&inventory).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("仓库中缺少装备 %s", itemID)
			}
			return fmt.Errorf("读取装备库存 %s: %w", itemID, err)
		}
	}
	var armorInstance models.ArmorInstance
	if err := db.Where("armor_id = ? AND status = ?", armorID, "normal").First(&armorInstance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("护甲 %s 已损坏或没有可用实例", armorID)
		}
		return fmt.Errorf("读取护甲耐久 %s: %w", armorID, err)
	}
	return nil
}

// ReplaceLostLoadout 扣除丢失装备并按第 presetIndex 套预设补购，返回补购金额。
func ReplaceLostLoadout(db *gorm.DB, presetIndex int) (int, error) {
	var lostLoadout models.PlayerLoadout
	if err := db.Transaction(func(tx *gorm.DB) error {
		loadout, err := GetPlayerLoadout(tx)
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
			if err := removeInventoryItem(tx, itemID, 1); err != nil {
				return err
			}
		}
		var armorInstance models.ArmorInstance
		if err := tx.Where("armor_id = ?", loadout.ArmorID).Order("id asc").First(&armorInstance).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("读取丢失护甲: %w", err)
			}
		} else if err := tx.Delete(&armorInstance).Error; err != nil {
			return fmt.Errorf("移除丢失护甲: %w", err)
		}

		cleared := models.PlayerLoadout{Consumables: []string{}}
		if err := tx.Model(&models.PlayerLoadout{}).Where("id = ?", models.PlayerLoadoutID).
			Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables").
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
		presetIDs := append([]string{presetWeaponID, presetArmorID}, presetConsumables...)
		for _, id := range presetEquip {
			if id != "" {
				presetIDs = append(presetIDs, id)
			}
		}
		items := make([]catalogItem, 0, len(presetIDs))
		for _, itemID := range uniqueItemIDs(presetIDs) {
			item, err := findCatalogItem(tx, itemID)
			if err != nil {
				return err
			}
			// 与商人联动：按归属商人的好感度折算价格，并校验解锁
			if err := applyMerchantPrice(tx, &item); err != nil {
				return fmt.Errorf("%w：预设装备补购失败（%v）", ErrPurchaseUnavailable, err)
			}
			items = append(items, item)
		}
		var err error
		paid, err = purchaseCatalogItems(tx, items)
		if err != nil {
			return err
		}

		updates := models.PlayerLoadout{
			WeaponID: presetWeaponID, ArmorID: presetArmorID,
			ChestRigID: presetEquip[0], BackpackID: presetEquip[1], HelmetID: presetEquip[2], HeadsetID: presetEquip[3],
			Consumables: presetConsumables,
		}
		if err := tx.Model(&models.PlayerLoadout{}).Where("id = ?", models.PlayerLoadoutID).
			Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables").
			Updates(&updates).Error; err != nil {
			return fmt.Errorf("启用补购装备: %w", err)
		}
		return nil
	})
	return paid, err
}
