// 通用物品实例：负责补给携带、耐久变化、成功归还和失能丢失。
package service

import (
	"fmt"
	"sort"

	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

func itemRefsFromLoadout(loadout *models.PlayerLoadout) []models.LoadoutItemRef {
	if len(loadout.ConsumableRefs) > 0 {
		return append([]models.LoadoutItemRef(nil), loadout.ConsumableRefs...)
	}
	refs := make([]models.LoadoutItemRef, 0, len(loadout.Consumables))
	for _, itemID := range loadout.Consumables {
		if itemID != "" {
			refs = append(refs, models.LoadoutItemRef{ItemID: itemID, Quantity: 1})
		}
	}
	return refs
}

func carriedItemsForLoadout(db *gorm.DB, userID uint, loadout *models.PlayerLoadout) ([]engine.CarriedItem, error) {
	refs := itemRefsFromLoadout(loadout)
	items := make([]engine.CarriedItem, 0, len(refs))
	for _, ref := range refs {
		quantity := ref.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		var useDef models.ItemUseDef
		if err := db.Where("item_id = ?", ref.ItemID).First(&useDef).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				items = append(items, engine.CarriedItem{ItemID: ref.ItemID, Quantity: quantity})
				continue
			}
			return nil, fmt.Errorf("读取物品效果 %s: %w", ref.ItemID, err)
		}
		if !useDef.InstanceRequired {
			var inventory models.Inventory
			if err := db.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, ref.ItemID, quantity).
				Order("raid_extract desc, id asc").First(&inventory).Error; err != nil {
				return nil, fmt.Errorf("物品 %s 聚合库存不足: %w", ref.ItemID, err)
			}
			items = append(items, engine.CarriedItem{ItemID: ref.ItemID, Quantity: quantity, RaidExtract: inventory.RaidExtract})
			continue
		}
		var instance models.ItemInstance
		query := db.Where("user_id = ? AND item_id = ? AND status = ? AND location_type = ?", userID, ref.ItemID, "normal", "inventory")
		if ref.InstanceID > 0 {
			query = db.Where("user_id = ? AND id = ? AND item_id = ? AND status = ? AND location_type = ?", userID, ref.InstanceID, ref.ItemID, "normal", "inventory")
		}
		if err := query.Order("current_durability asc, id asc").First(&instance).Error; err != nil {
			return nil, fmt.Errorf("物品 %s 没有可用耐久实例: %w", ref.ItemID, err)
		}
		items = append(items, engine.CarriedItem{InstanceID: instance.ID, ItemID: instance.ItemID, Quantity: 1, CurrentDurability: instance.CurrentDurability, MaxDurability: instance.MaxDurability, RaidExtract: instance.RaidExtract})
	}
	return items, nil
}

func itemStacksFromCarriedItems(items []engine.CarriedItem) []engine.ItemStack {
	quantities := make(map[string]int)
	for _, item := range items {
		if item.InstanceID > 0 && item.CurrentDurability <= 0 {
			continue
		}
		quantity := item.Quantity
		if item.InstanceID > 0 {
			quantity = 1
		}
		if quantity > 0 {
			quantities[item.ItemID] += quantity
		}
	}
	ids := make([]string, 0, len(quantities))
	for itemID := range quantities {
		ids = append(ids, itemID)
	}
	sort.Strings(ids)
	result := make([]engine.ItemStack, 0, len(ids))
	for _, itemID := range ids {
		result = append(result, engine.ItemStack{ItemID: itemID, Quantity: quantities[itemID]})
	}
	return result
}

func reserveCarriedItemsTx(tx *gorm.DB, userID uint, items []engine.CarriedItem, locationRef string) error {
	type itemSource struct {
		itemID      string
		raidExtract bool
	}
	quantities := make(map[itemSource]int)
	for _, item := range items {
		if item.InstanceID > 0 {
			result := tx.Model(&models.ItemInstance{}).
				Where("user_id = ? AND id = ? AND status = ? AND location_type = ?", userID, item.InstanceID, "normal", "inventory").
				Updates(map[string]interface{}{"status": "locked", "location_type": "session", "location_ref": locationRef})
			if result.Error != nil {
				return fmt.Errorf("锁定物品实例 %d: %w", item.InstanceID, result.Error)
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("物品实例 %d 不可用", item.InstanceID)
			}
			continue
		}
		quantities[itemSource{itemID: item.ItemID, raidExtract: item.RaidExtract}] += item.Quantity
	}
	for source, quantity := range quantities {
		raidExtract := source.raidExtract
		if err := removeInventoryItemFromSource(tx, userID, source.itemID, quantity, &raidExtract); err != nil {
			return fmt.Errorf("携带补给 %s: %w", source.itemID, err)
		}
	}
	return nil
}

func returnCarriedItemsTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, items []engine.CarriedItem) error {
	for _, item := range items {
		if item.InstanceID > 0 {
			status := "normal"
			if item.CurrentDurability <= 0 {
				status = "depleted"
			}
			if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ? AND location_type = ?", userID, item.InstanceID, "session").Updates(map[string]interface{}{
				"current_durability": item.CurrentDurability, "status": status, "location_type": "inventory", "location_ref": "",
				"raid_extract": item.RaidExtract,
			}).Error; err != nil {
				return fmt.Errorf("归还物品实例 %d: %w", item.InstanceID, err)
			}
			continue
		}
		if item.Quantity <= 0 {
			continue
		}
		catalog, err := snapshotCatalogItem(snapshot, item.ItemID)
		if err != nil {
			return err
		}
		if err := addInventoryItem(tx, userID, catalog, item.Quantity, false); err != nil {
			return fmt.Errorf("归还补给 %s: %w", item.ItemID, err)
		}
	}
	return nil
}

type ItemInstanceView struct {
	models.ItemInstance
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Category         string `json:"category"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

func ListItemInstancesForUser(db *gorm.DB, userID uint) ([]ItemInstanceView, error) {
	var items []models.ItemInstance
	if err := db.Where("user_id = ? AND location_type = ?", userID, "inventory").Order("item_id asc, id asc").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("读取物品实例: %w", err)
	}
	itemIDs := make([]string, 0, len(items))
	for _, instance := range items {
		itemIDs = append(itemIDs, instance.ItemID)
	}
	catalogItems, err := catalog.New(db).FindByIDs(itemIDs)
	if err != nil {
		return nil, fmt.Errorf("读取物品实例目录: %w", err)
	}
	result := make([]ItemInstanceView, 0, len(items))
	for _, instance := range items {
		item, ok := catalogItems[instance.ItemID]
		if !ok {
			return nil, fmt.Errorf("读取物品实例目录 %s: %w", instance.ItemID, catalog.ErrItemNotFound)
		}
		result = append(result, ItemInstanceView{
			ItemInstance:     instance,
			Name:             item.Name,
			Kind:             item.Kind,
			Category:         item.Category,
			Price:            item.Price,
			Weight:           item.Weight,
			Slots:            item.Slots,
			MerchantCategory: item.MerchantCategory,
			RepRequirement:   item.RepRequirement,
		})
	}
	return result, nil
}

func discardCarriedItemsTx(tx *gorm.DB, userID uint, items []engine.CarriedItem) error {
	for _, item := range items {
		if item.InstanceID == 0 {
			continue
		}
		if err := tx.Where("user_id = ? AND id = ?", userID, item.InstanceID).Delete(&models.ItemInstance{}).Error; err != nil {
			return fmt.Errorf("丢失物品实例 %d: %w", item.InstanceID, err)
		}
	}
	return nil
}

// discardSessionItemInstancesTx 清理行动中已消耗、因此不再出现在 EngineState 的实例物品。
func discardSessionItemInstancesTx(tx *gorm.DB, userID uint) error {
	return tx.Where("user_id = ? AND location_type = ?", userID, "session").Delete(&models.ItemInstance{}).Error
}
