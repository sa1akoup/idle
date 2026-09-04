package service

import (
	"fmt"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

func equippedSecureInnerSlotsTx(tx *gorm.DB, userID uint) (int, error) {
	var loadout models.PlayerLoadout
	if err := tx.Where("user_id = ?", userID).First(&loadout).Error; err != nil {
		return 0, fmt.Errorf("读取安全箱配装: %w", err)
	}
	if loadout.SecureContainerID == "" {
		return 0, nil
	}
	var def models.SecureContainerDef
	if err := tx.Where("id = ?", loadout.SecureContainerID).First(&def).Error; err != nil {
		return 0, fmt.Errorf("读取安全箱 %s: %w", loadout.SecureContainerID, err)
	}
	if def.InnerSlots < 0 {
		return 0, nil
	}
	return def.InnerSlots, nil
}

func (s *SessionService) storeSecureLootTx(tx *gorm.DB, snapshot engine.ScenarioSnapshot, loot []engine.LootDrop) ([]engine.LootDrop, []engine.LootDrop, error) {
	slots, err := equippedSecureInnerSlotsTx(tx, s.userID)
	if err != nil {
		return nil, nil, err
	}
	kept := engine.SelectSecureLoot(snapshot, loot, slots)
	if len(kept) == 0 {
		return nil, nil, nil
	}
	stored, overflow, err := fitEngineLootToStorage(tx, s.userID, snapshot, kept)
	if err != nil {
		return nil, nil, err
	}
	for _, drop := range stored {
		item, err := snapshotCatalogItem(snapshot, drop.ItemID)
		if err != nil {
			return nil, nil, err
		}
		if err := addInventoryItem(tx, s.userID, item, drop.Quantity, false); err != nil {
			return nil, nil, err
		}
	}
	return stored, overflow, nil
}

func appendSecureLootReport(report []string, snapshot engine.ScenarioSnapshot, stored []engine.LootDrop) []string {
	if len(stored) == 0 {
		return append(report, ">> 安全箱未能保住搜刮")
	}
	for _, drop := range stored {
		name := drop.ItemID
		if item, ok := snapshot.LootItems[drop.ItemID]; ok && item.Name != "" {
			name = item.Name
		} else if ammo, ok := snapshot.Ammos[drop.ItemID]; ok && ammo.Name != "" {
			name = ammo.Name
		} else if item, ok := snapshot.Items[drop.ItemID]; ok && item.Name != "" {
			name = item.Name
		}
		report = append(report, fmt.Sprintf(">> 安全箱保住 %s x%d（非局内带出）", name, drop.Quantity))
	}
	return report
}
