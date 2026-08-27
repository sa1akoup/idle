// 可用物品目录：合并消耗品、医疗品、食品、饮水、燃料和维修包的使用效果。
package service

import (
	"fmt"
	"sort"

	"idle/internal/models"

	"gorm.io/gorm"
)

// UsableItemCatalogItem 是前端和商人共用的可使用物品目录项。
type UsableItemCatalogItem struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Kind              string  `json:"kind"`
	Category          string  `json:"category"`
	Desc              string  `json:"desc"`
	Price             int     `json:"price"`
	Weight            int     `json:"weight"`
	Slots             int     `json:"slots"`
	MerchantCategory  string  `json:"merchantCategory"`
	RepRequirement    int     `json:"repRequirement"`
	HPRecovery        float64 `json:"hpRecovery"`
	EnergyRecovery    float64 `json:"energyRecovery"`
	HydrationRecovery float64 `json:"hydrationRecovery"`
	RepairValue       float64 `json:"repairValue"`
	FuelSeconds       int64   `json:"fuelSeconds"`
	MaxDurability     float64 `json:"maxDurability"`
	UseDurability     float64 `json:"useDurability"`
	InstanceRequired  bool    `json:"instanceRequired"`
	UsableInSession   bool    `json:"usableInSession"`
	UsableInHideout   bool    `json:"usableInHideout"`
}

// ListUsableItems 返回所有可以携带或在藏身处使用的补给类物品。
func ListUsableItems(db *gorm.DB) ([]UsableItemCatalogItem, error) {
	var uses []models.ItemUseDef
	if err := db.Order("item_id asc").Find(&uses).Error; err != nil {
		return nil, fmt.Errorf("读取可用物品效果: %w", err)
	}
	useByID := make(map[string]models.ItemUseDef, len(uses))
	for _, use := range uses {
		useByID[use.ItemID] = use
	}

	items := make(map[string]UsableItemCatalogItem)
	var consumables []models.ConsumableDef
	if err := db.Order("id asc").Find(&consumables).Error; err != nil {
		return nil, fmt.Errorf("读取消耗品目录: %w", err)
	}
	for _, item := range consumables {
		use, ok := useByID[item.ID]
		if !ok || (!use.UsableInSession && !use.UsableInHideout) {
			continue
		}
		items[item.ID] = usableItemFromConsumable(item, use)
	}

	var lootItems []models.LootItemDef
	if err := db.Order("id asc").Find(&lootItems).Error; err != nil {
		return nil, fmt.Errorf("读取战利品目录: %w", err)
	}
	for _, item := range lootItems {
		use, ok := useByID[item.ID]
		if !ok || (!use.UsableInSession && !use.UsableInHideout && use.RepairValue <= 0 && use.FuelSeconds <= 0) {
			continue
		}
		items[item.ID] = usableItemFromLoot(item, use)
	}

	result := make([]UsableItemCatalogItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func usableItemFromConsumable(item models.ConsumableDef, use models.ItemUseDef) UsableItemCatalogItem {
	return UsableItemCatalogItem{
		ID: item.ID, Name: item.Name, Kind: "consumable", Desc: item.Desc, Price: item.Price, Weight: item.Weight, Slots: item.Slots,
		MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement,
		HPRecovery: use.HPRecovery, EnergyRecovery: use.EnergyRecovery, HydrationRecovery: use.HydrationRecovery,
		RepairValue: use.RepairValue, FuelSeconds: use.FuelSeconds, MaxDurability: use.MaxDurability, UseDurability: use.UseDurability,
		InstanceRequired: use.InstanceRequired, UsableInSession: use.UsableInSession, UsableInHideout: use.UsableInHideout,
	}
}

func usableItemFromLoot(item models.LootItemDef, use models.ItemUseDef) UsableItemCatalogItem {
	return UsableItemCatalogItem{
		ID: item.ID, Name: item.Name, Kind: "loot", Category: item.Category, Desc: item.Desc, Price: item.Price, Weight: item.Weight, Slots: item.Slots,
		MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement,
		HPRecovery: use.HPRecovery, EnergyRecovery: use.EnergyRecovery, HydrationRecovery: use.HydrationRecovery,
		RepairValue: use.RepairValue, FuelSeconds: use.FuelSeconds, MaxDurability: use.MaxDurability, UseDurability: use.UseDurability,
		InstanceRequired: use.InstanceRequired, UsableInSession: use.UsableInSession, UsableInHideout: use.UsableInHideout,
	}
}
