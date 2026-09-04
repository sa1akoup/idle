package service

import (
	"errors"
	"fmt"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	keyCaseLocation     = "keycase"
	defaultKeyCaseID    = "keycase_03"
	defaultKeyCaseSlots = 3
)

type KeySlotView struct {
	SlotIndex           int     `json:"slotIndex"`
	InstanceID          uint    `json:"instanceId"`
	ItemID              string  `json:"itemId"`
	Name                string  `json:"name"`
	CurrentDurability   float64 `json:"currentDurability"`
	MaxDurability       float64 `json:"maxDurability"`
}

type LoadoutView struct {
	models.PlayerLoadout
	KeyCaseSlots int           `json:"keyCaseSlots"`
	Keys         []KeySlotView `json:"keys"`
}

func keyCaseSlotsOf(tx *gorm.DB, keyCaseID string) (int, error) {
	if keyCaseID == "" {
		return 0, nil
	}
	var def models.KeyCaseDef
	if err := tx.Where("id = ?", keyCaseID).First(&def).Error; err != nil {
		return 0, fmt.Errorf("读取钥匙包 %s: %w", keyCaseID, err)
	}
	if def.KeySlots < 0 {
		return 0, nil
	}
	return def.KeySlots, nil
}

func LoadoutViewForUser(db *gorm.DB, userID uint) (*LoadoutView, error) {
	loadout, err := GetPlayerLoadoutForUser(db, userID)
	if err != nil {
		return nil, err
	}
	return attachLoadoutView(db, userID, loadout)
}

func attachLoadoutView(db *gorm.DB, userID uint, loadout *models.PlayerLoadout) (*LoadoutView, error) {
	slots, err := keyCaseSlotsOf(db, loadout.KeyCaseID)
	if err != nil {
		return nil, err
	}
	var instances []models.ItemInstance
	if err := db.Where("user_id = ? AND location_type = ? AND status = ?", userID, keyCaseLocation, "normal").
		Order("slot_index asc, id asc").Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("读取钥匙包内容: %w", err)
	}
	keys := make([]KeySlotView, slots)
	for i := range keys {
		keys[i] = KeySlotView{SlotIndex: i}
	}
	for _, instance := range instances {
		if instance.SlotIndex < 0 || instance.SlotIndex >= slots {
			continue
		}
		name := instance.ItemID
		var loot models.LootItemDef
		if err := db.Where("id = ?", instance.ItemID).First(&loot).Error; err == nil {
			name = loot.Name
		}
		keys[instance.SlotIndex] = KeySlotView{
			SlotIndex: instance.SlotIndex, InstanceID: instance.ID, ItemID: instance.ItemID, Name: name,
			CurrentDurability: instance.CurrentDurability, MaxDurability: instance.MaxDurability,
		}
	}
	return &LoadoutView{PlayerLoadout: *loadout, KeyCaseSlots: slots, Keys: keys}, nil
}

func carriedKeysForUser(db *gorm.DB, userID uint) ([]engine.CarriedItem, error) {
	var instances []models.ItemInstance
	if err := db.Where("user_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, keyCaseLocation, "normal").
		Order("slot_index asc, id asc").Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("读取钥匙包钥匙: %w", err)
	}
	items := make([]engine.CarriedItem, 0, len(instances))
	for _, instance := range instances {
		items = append(items, engine.CarriedItem{
			InstanceID: instance.ID, ItemID: instance.ItemID, Quantity: 1,
			CurrentDurability: instance.CurrentDurability, MaxDurability: instance.MaxDurability,
			RaidExtract: instance.RaidExtract, Secure: true,
		})
	}
	return items, nil
}

func applyKeyCaseContentsTx(tx *gorm.DB, userID uint, keyCaseID string, instanceIDs []uint) error {
	slots, err := keyCaseSlotsOf(tx, keyCaseID)
	if err != nil {
		return err
	}
	if len(instanceIDs) > slots {
		return fmt.Errorf("钥匙包最多放入 %d 把钥匙", slots)
	}
	seen := make(map[uint]bool, len(instanceIDs))
	for _, instanceID := range instanceIDs {
		if instanceID == 0 {
			continue
		}
		if seen[instanceID] {
			return fmt.Errorf("同一把钥匙不能重复放入钥匙包")
		}
		seen[instanceID] = true
	}
	var current []models.ItemInstance
	if err := tx.Where("user_id = ? AND location_type = ?", userID, keyCaseLocation).Find(&current).Error; err != nil {
		return fmt.Errorf("读取钥匙包原内容: %w", err)
	}
	keep := make(map[uint]bool, len(seen))
	for instanceID := range seen {
		keep[instanceID] = true
	}
	for _, instance := range current {
		if keep[instance.ID] {
			continue
		}
		if err := tx.Model(&instance).Updates(map[string]interface{}{
			"location_type": "inventory", "location_ref": "", "slot_index": 0,
		}).Error; err != nil {
			return fmt.Errorf("卸下钥匙 %d: %w", instance.ID, err)
		}
	}
	for index, instanceID := range instanceIDs {
		if instanceID == 0 {
			continue
		}
		var instance models.ItemInstance
		if err := tx.Where("user_id = ? AND id = ? AND status = ? AND current_durability > 0", userID, instanceID, "normal").First(&instance).Error; err != nil {
			return fmt.Errorf("钥匙实例 %d 不可用", instanceID)
		}
		var loot models.LootItemDef
		if err := tx.Where("id = ?", instance.ItemID).First(&loot).Error; err != nil || loot.Category != "key" {
			return fmt.Errorf("只能把钥匙放入钥匙包")
		}
		if err := tx.Model(&instance).Updates(map[string]interface{}{
			"location_type": keyCaseLocation, "location_ref": keyCaseID, "slot_index": index,
		}).Error; err != nil {
			return fmt.Errorf("装入钥匙 %d: %w", instanceID, err)
		}
	}
	return nil
}

func writeBackSecureItemsTx(tx *gorm.DB, userID uint, items []engine.CarriedItem) error {
	for _, item := range items {
		if !item.Secure || item.InstanceID == 0 {
			continue
		}
		status := "normal"
		if item.CurrentDurability <= 0 {
			status = "depleted"
		}
		updates := map[string]interface{}{"current_durability": item.CurrentDurability, "status": status}
		if item.CurrentDurability <= 0 {
			updates["location_type"] = "inventory"
			updates["location_ref"] = ""
			updates["slot_index"] = 0
		}
		if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ?", userID, item.InstanceID).Updates(updates).Error; err != nil {
			return fmt.Errorf("写回钥匙 %d: %w", item.InstanceID, err)
		}
		if item.CurrentDurability <= 0 {
			if err := tx.Where("user_id = ? AND id = ?", userID, item.InstanceID).Delete(&models.ItemInstance{}).Error; err != nil {
				return fmt.Errorf("清理耗尽钥匙 %d: %w", item.InstanceID, err)
			}
		}
	}
	return nil
}

func refillKeyCaseTx(tx *gorm.DB, userID uint, keyCaseID string) error {
	slots, err := keyCaseSlotsOf(tx, keyCaseID)
	if err != nil {
		return err
	}
	if slots <= 0 {
		return nil
	}
	var current []models.ItemInstance
	if err := tx.Where("user_id = ? AND location_type = ?", userID, keyCaseLocation).Find(&current).Error; err != nil {
		return fmt.Errorf("读取钥匙包: %w", err)
	}
	occupied := make(map[int]models.ItemInstance, len(current))
	needed := make(map[string]int)
	for _, instance := range current {
		if instance.CurrentDurability <= 0 || instance.Status == "depleted" {
			needed[instance.ItemID]++
			if err := tx.Delete(&instance).Error; err != nil {
				return fmt.Errorf("清理耗尽钥匙: %w", err)
			}
			continue
		}
		occupied[instance.SlotIndex] = instance
	}
	for itemID, count := range needed {
		for i := 0; i < count; i++ {
			slot := firstEmptyKeySlot(occupied, slots)
			if slot < 0 {
				return nil
			}
			var replacement models.ItemInstance
			if err := tx.Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0",
				userID, itemID, "inventory", "normal").Order("current_durability desc, id asc").First(&replacement).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					break
				}
				return fmt.Errorf("补充钥匙 %s: %w", itemID, err)
			}
			if err := tx.Model(&replacement).Updates(map[string]interface{}{
				"location_type": keyCaseLocation, "location_ref": keyCaseID, "slot_index": slot,
			}).Error; err != nil {
				return fmt.Errorf("装入补充钥匙 %s: %w", itemID, err)
			}
			occupied[slot] = replacement
		}
	}
	return nil
}

func firstEmptyKeySlot(occupied map[int]models.ItemInstance, slots int) int {
	for i := 0; i < slots; i++ {
		if _, ok := occupied[i]; !ok {
			return i
		}
	}
	return -1
}

func settleSecureKeysTx(tx *gorm.DB, userID uint, items []engine.CarriedItem) error {
	if err := writeBackSecureItemsTx(tx, userID, items); err != nil {
		return err
	}
	var loadout models.PlayerLoadout
	if err := tx.Where("user_id = ?", userID).First(&loadout).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("读取钥匙包配装: %w", err)
	}
	return refillKeyCaseTx(tx, userID, loadout.KeyCaseID)
}

func ensureStarterKeyCaseTx(tx *gorm.DB, userID uint) error {
	var count int64
	if err := tx.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ? AND quantity > 0", userID, defaultKeyCaseID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		item := models.Inventory{
			UserID: userID, ItemID: defaultKeyCaseID, Name: "简易钥匙包", Kind: "keycase",
			Quantity: 1, Price: 400, Weight: 1, Slots: 1, MerchantCategory: "clothing",
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("发放简易钥匙包: %w", err)
		}
	}
	return tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND (key_case_id = ? OR key_case_id IS NULL)", userID, "").
		Update("key_case_id", defaultKeyCaseID).Error
}
