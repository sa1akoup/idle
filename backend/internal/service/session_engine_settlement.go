// 探索结算补购模块：在单局事务内处理失能丢装、固定价格补购和装备恢复。
package service

import (
	"errors"
	"fmt"
	"sort"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

// replaceLostLoadoutTx 将失能丢装、预设补购和装备实例变更纳入当前结算事务。
func replaceLostLoadoutTx(tx *gorm.DB, userID uint, presetIndex int, snapshot engine.ScenarioSnapshot, lostLoadout engine.LoadoutState, lostConsumables []engine.ItemStack) error {
	loadout, err := GetPlayerLoadoutForUser(tx, userID)
	if err != nil {
		return err
	}
	lostIDs := []string{lostLoadout.WeaponID, lostLoadout.ArmorID, lostLoadout.ChestRigID, lostLoadout.BackpackID, lostLoadout.HelmetID, lostLoadout.HeadsetID}
	for _, stack := range lostConsumables {
		for i := 0; i < stack.Quantity; i++ {
			lostIDs = append(lostIDs, stack.ItemID)
		}
	}
	lostQuantities := make(map[string]int, len(lostIDs))
	for _, itemID := range lostIDs {
		if itemID != "" {
			lostQuantities[itemID]++
		}
	}
	lostItemIDs := make([]string, 0, len(lostQuantities))
	for itemID := range lostQuantities {
		lostItemIDs = append(lostItemIDs, itemID)
	}
	// 按排序后的物品 ID 依次扣减，保证多次扣减的库存操作顺序稳定。
	sort.Strings(lostItemIDs)
	for _, itemID := range lostItemIDs {
		if err := removeInventoryItem(tx, userID, itemID, lostQuantities[itemID]); err != nil {
			return err
		}
	}
	var armorInstance models.ArmorInstance
	if err := tx.Where("user_id = ? AND armor_id = ?", userID, lostLoadout.ArmorID).Order("id asc").First(&armorInstance).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("读取丢失护甲: %w", err)
		}
	} else if err := tx.Delete(&armorInstance).Error; err != nil {
		return fmt.Errorf("移除丢失护甲: %w", err)
	}
	cleared := models.PlayerLoadout{Consumables: []string{}, ConsumableRefs: []models.LoadoutItemRef{}}
	if err := tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND id = ?", userID, loadout.ID).
		Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables", "ConsumableRefs").Updates(&cleared).Error; err != nil {
		return fmt.Errorf("清空丢失装备: %w", err)
	}

	return purchaseRecoveryPresetTx(tx, userID, presetIndex, snapshot, loadout.ID)
}

// purchaseRecoveryPresetTx 只购买并启用已经保存过价格快照的失能预设。
func purchaseRecoveryPresetTx(tx *gorm.DB, userID uint, presetIndex int, snapshot engine.ScenarioSnapshot, loadoutID uint) error {
	preset, ok := snapshot.RecoveryPresets[presetIndex]
	if !ok || len(preset.Items) == 0 {
		return fmt.Errorf("%w：预设装备 %d 没有固定补购配置", ErrPurchaseUnavailable, presetIndex)
	}
	items := make([]catalogItem, 0)
	for _, recoveryItem := range preset.Items {
		if !recoveryItem.Available {
			return fmt.Errorf("%w：补购商品 %s 在行动开始时不可用", ErrPurchaseUnavailable, recoveryItem.ItemID)
		}
		item, err := snapshotCatalogItem(snapshot, recoveryItem.ItemID)
		if err != nil {
			return err
		}
		item.PaidPrice = recoveryItem.UnitPrice
		for i := 0; i < recoveryItem.Quantity; i++ {
			items = append(items, item)
		}
	}
	if _, err := purchaseCatalogItems(tx, userID, items); err != nil {
		return err
	}
	updates := models.PlayerLoadout{
		WeaponID: preset.Loadout.WeaponID, ArmorID: preset.Loadout.ArmorID,
		ChestRigID: preset.Loadout.ChestRigID, BackpackID: preset.Loadout.BackpackID, HelmetID: preset.Loadout.HelmetID, HeadsetID: preset.Loadout.HeadsetID,
		Consumables: loadoutConsumableIDs(preset.Consumables), ConsumableRefs: []models.LoadoutItemRef{}, CarriedAmmo: []models.AmmoCell{},
	}
	if err := tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND id = ?", userID, loadoutID).
		Select("WeaponID", "ArmorID", "ChestRigID", "BackpackID", "HelmetID", "HeadsetID", "Consumables", "ConsumableRefs", "CarriedAmmo").Updates(&updates).Error; err != nil {
		return fmt.Errorf("启用补购装备: %w", err)
	}
	return nil
}
