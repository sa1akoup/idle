package service

// loot 服务：将数据库中的可搜集物品定义转换为探索结算与库存共用的商品结构。

import (
	"fmt"
	"math/rand"
	"sort"

	"idle/internal/models"

	"gorm.io/gorm"
)

func loadLootCatalog(db *gorm.DB) (map[string]catalogItem, error) {
	var definitions []models.LootItemDef
	if err := db.Order("id asc").Find(&definitions).Error; err != nil {
		return nil, fmt.Errorf("读取 loot 目录: %w", err)
	}

	catalog := make(map[string]catalogItem, len(definitions))
	for _, item := range definitions {
		catalog[item.ID] = catalogItem{
			ID: item.ID, Name: item.Name, Kind: "loot", Category: item.Category,
			Price: item.Price, Weight: item.Weight, Slots: item.Slots,
			DropWeight:       item.DropWeight,
			MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement,
		}
	}
	return catalog, nil
}

type lootContainer struct {
	Def   models.LootContainerDef
	Rules []models.LootContainerRule
}

func loadLootContainers(db *gorm.DB) (map[string]lootContainer, error) {
	var definitions []models.LootContainerDef
	if err := db.Order("id asc").Find(&definitions).Error; err != nil {
		return nil, fmt.Errorf("读取容器目录: %w", err)
	}
	var rules []models.LootContainerRule
	if err := db.Order("id asc").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("读取容器规则: %w", err)
	}

	containers := make(map[string]lootContainer, len(definitions))
	for _, definition := range definitions {
		containers[definition.ID] = lootContainer{Def: definition}
	}
	for _, rule := range rules {
		container := containers[rule.ContainerID]
		container.Rules = append(container.Rules, rule)
		containers[rule.ContainerID] = container
	}
	return containers, nil
}

func loadNodeContainers(db *gorm.DB, nodeIDs []string) (map[string][]models.NodeContainerDef, error) {
	var assignments []models.NodeContainerDef
	if err := db.Where("node_id IN ?", nodeIDs).Order("id asc").Find(&assignments).Error; err != nil {
		return nil, fmt.Errorf("读取节点容器池: %w", err)
	}
	result := make(map[string][]models.NodeContainerDef)
	for _, assignment := range assignments {
		if assignment.Pool == "" {
			assignment.Pool = models.NodeContainerPoolSearch
		}
		result[assignment.NodeID] = append(result[assignment.NodeID], assignment)
	}
	return result, nil
}

// materializeNodeContainers 按节点槽位和容器池权重生成本局实际容器。
// 这里只生成普通 search 池；事件奖励池在事件成功时即时抽取，不占用普通搜索槽位。
func materializeNodeContainers(nodes []models.NodeDef, pools map[string][]models.NodeContainerDef, rng *rand.Rand) map[string][]models.NodeContainerDef {
	result := make(map[string][]models.NodeContainerDef, len(nodes))
	for _, node := range nodes {
		entries := filterNodeContainerPool(pools[node.ID], models.NodeContainerPoolSearch)
		if len(entries) == 0 {
			continue
		}
		slots := node.ContainerSlots
		if slots <= 0 {
			for _, entry := range entries {
				count := entry.Count
				if count <= 0 {
					count = 1
				}
				for i := 0; i < count; i++ {
					result[node.ID] = append(result[node.ID], models.NodeContainerDef{NodeID: node.ID, ContainerID: entry.ContainerID, Count: 1})
				}
			}
			continue
		}
		for i := 0; i < slots; i++ {
			entry, ok := chooseNodeContainer(entries, rng)
			if !ok {
				break
			}
			result[node.ID] = append(result[node.ID], models.NodeContainerDef{NodeID: node.ID, ContainerID: entry.ContainerID, Count: 1})
		}
	}
	return result
}

func chooseNodeContainer(entries []models.NodeContainerDef, rng *rand.Rand) (models.NodeContainerDef, bool) {
	totalWeight := 0
	for _, entry := range entries {
		if entry.Weight > 0 {
			totalWeight += entry.Weight
		}
	}
	if totalWeight <= 0 {
		return models.NodeContainerDef{}, false
	}
	roll := rng.Intn(totalWeight)
	for _, entry := range entries {
		if entry.Weight <= 0 {
			continue
		}
		if roll < entry.Weight {
			return entry, true
		}
		roll -= entry.Weight
	}
	return models.NodeContainerDef{}, false
}

func filterNodeContainerPool(entries []models.NodeContainerDef, pool string) []models.NodeContainerDef {
	filtered := make([]models.NodeContainerDef, 0, len(entries))
	for _, entry := range entries {
		entryPool := entry.Pool
		if entryPool == "" {
			entryPool = models.NodeContainerPoolSearch
		}
		if entryPool == pool {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func hasNodeContainerPool(entries []models.NodeContainerDef, pool string) bool {
	for _, entry := range filterNodeContainerPool(entries, pool) {
		if entry.Weight > 0 {
			return true
		}
	}
	return false
}

func chooseNodeContainerPool(entries []models.NodeContainerDef, pool string, rng *rand.Rand) (models.NodeContainerDef, bool) {
	return chooseNodeContainer(filterNodeContainerPool(entries, pool), rng)
}

func chooseContainerRule(container lootContainer, rng *rand.Rand) (models.LootContainerRule, bool) {
	totalWeight := 0
	for _, rule := range container.Rules {
		if rule.Weight > 0 {
			totalWeight += rule.Weight
		}
	}
	if totalWeight <= 0 {
		return models.LootContainerRule{}, false
	}

	roll := rng.Intn(totalWeight)
	for _, rule := range container.Rules {
		if rule.Weight <= 0 {
			continue
		}
		if roll < rule.Weight {
			return rule, true
		}
		roll -= rule.Weight
	}
	return models.LootContainerRule{}, false
}

func chooseLootItem(catalog map[string]catalogItem, category string, rng *rand.Rand) (catalogItem, bool) {
	ids := make([]string, 0)
	totalWeight := 0
	for id, item := range catalog {
		if item.Category != category {
			continue
		}
		weight := item.DropWeight
		if weight <= 0 {
			weight = 1
		}
		ids = append(ids, id)
		totalWeight += weight
	}
	if len(ids) == 0 || totalWeight <= 0 {
		return catalogItem{}, false
	}
	sort.Strings(ids)

	roll := rng.Intn(totalWeight)
	for _, id := range ids {
		item := catalog[id]
		weight := item.DropWeight
		if weight <= 0 {
			weight = 1
		}
		if roll < weight {
			return item, true
		}
		roll -= weight
	}
	return catalogItem{}, false
}
