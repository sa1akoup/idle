// 纯掉落模块：按稳定排序和权重完成节点容器、容器规则与物品抽取。
package engine

import (
	"fmt"
	"math/rand"
	"sort"
)

const NodeContainerPoolSearch = "search"

// materializeNodeContainers 按节点抽定本局要搜索的容器列表；有容器槽位的按槽位加权抽取。
func materializeNodeContainers(nodes []Node, assignments []NodeContainerAssignment, rng *rand.Rand) map[string][]NodeContainerAssignment {
	nodes = sortedNodes(nodes)
	byNode := make(map[string][]NodeContainerAssignment)
	for _, assignment := range assignments {
		if assignment.Pool == "" {
			assignment.Pool = NodeContainerPoolSearch
		}
		byNode[assignment.NodeID] = append(byNode[assignment.NodeID], assignment)
	}
	result := make(map[string][]NodeContainerAssignment, len(nodes))
	for _, node := range nodes {
		entries := filterNodeContainerPool(byNode[node.ID], NodeContainerPoolSearch)
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
					result[node.ID] = append(result[node.ID], NodeContainerAssignment{NodeID: node.ID, ContainerID: entry.ContainerID, Count: 1})
				}
			}
			continue
		}
		for i := 0; i < slots; i++ {
			entry, ok := chooseNodeContainer(entries, rng)
			if !ok {
				break
			}
			result[node.ID] = append(result[node.ID], NodeContainerAssignment{NodeID: node.ID, ContainerID: entry.ContainerID, Count: 1})
		}
	}
	return result
}

// chooseNodeContainer 按权重从容器分配中抽一个，无正权重时返回 false。
func chooseNodeContainer(entries []NodeContainerAssignment, rng *rand.Rand) (NodeContainerAssignment, bool) {
	entries = sortedNodeAssignments(entries)
	totalWeight := 0
	for _, entry := range entries {
		if entry.Weight > 0 {
			totalWeight += entry.Weight
		}
	}
	if totalWeight <= 0 {
		return NodeContainerAssignment{}, false
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
	return NodeContainerAssignment{}, false
}

// filterNodeContainerPool 过滤出指定容器池的分配，空池名按默认搜索池兼容。
func filterNodeContainerPool(entries []NodeContainerAssignment, pool string) []NodeContainerAssignment {
	// 空池名按默认搜索池处理，兼容旧数据缺省配置。
	filtered := make([]NodeContainerAssignment, 0, len(entries))
	for _, entry := range entries {
		entryPool := entry.Pool
		if entryPool == "" {
			entryPool = NodeContainerPoolSearch
		}
		if entryPool == pool {
			filtered = append(filtered, entry)
		}
	}
	return sortedNodeAssignments(filtered)
}

// sortedNodeAssignments 返回按容器 ID 排序的分配副本，保证抽取顺序确定。
func sortedNodeAssignments(entries []NodeContainerAssignment) []NodeContainerAssignment {
	result := append([]NodeContainerAssignment(nil), entries...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ContainerID == result[j].ContainerID {
			return result[i].ID < result[j].ID
		}
		return result[i].ContainerID < result[j].ContainerID
	})
	return result
}

// hasNodeContainerPool 判断节点上是否存在给定池的正权重分配（供事件条件查询）。
func hasNodeContainerPool(entries []NodeContainerAssignment, pool string) bool {
	for _, entry := range filterNodeContainerPool(entries, pool) {
		if entry.Weight > 0 {
			return true
		}
	}
	return false
}

// chooseNodeContainerPool 在指定池内按权重抽一个容器分配。
func chooseNodeContainerPool(entries []NodeContainerAssignment, pool string, rng *rand.Rand) (NodeContainerAssignment, bool) {
	return chooseNodeContainer(filterNodeContainerPool(entries, pool), rng)
}

// chooseContainerRule 按权重从容器规则中抽一条掉落规则，无正权重返回 false。
func chooseContainerRule(container Container, rng *rand.Rand) (ContainerRule, bool) {
	rules := append([]ContainerRule(nil), container.Rules...)
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	totalWeight := 0
	for _, rule := range rules {
		if rule.Weight > 0 {
			totalWeight += rule.Weight
		}
	}
	if totalWeight <= 0 {
		return ContainerRule{}, false
	}
	roll := rng.Intn(totalWeight)
	for _, rule := range rules {
		if rule.Weight <= 0 {
			continue
		}
		if roll < rule.Weight {
			return rule, true
		}
		roll -= rule.Weight
	}
	return ContainerRule{}, false
}

// chooseLootItem 从战利品目录中按类别与掉落权重抽一件物品，未配置权重按 1 参与。
func chooseLootItem(catalog map[string]LootItem, category string, rng *rand.Rand) (LootItem, bool) {
	ids := make([]string, 0, len(catalog))
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
	sort.Strings(ids)
	if len(ids) == 0 || totalWeight <= 0 {
		return LootItem{}, false
	}
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
	return LootItem{}, false
}

// ammoUsage 按快照中的每格发数计算携带弹药的格数与重量。
func ammoUsage(snapshot ScenarioSnapshot, ammoID string, rounds int) (int, float64, error) {
	if rounds <= 0 {
		return 0, 0, nil
	}
	ammo, ok := snapshot.Ammos[ammoID]
	if !ok {
		return 0, 0, fmt.Errorf("弹药 %s 不在场景快照目录中", ammoID)
	}
	if ammo.RoundsPerSlot <= 0 {
		return 0, 0, fmt.Errorf("弹药 %s 的每格容量无效", ammoID)
	}
	groups := ceilDiv(rounds, ammo.RoundsPerSlot)
	return groups, float64(groups) * snapshot.Tuning.AmmoDrop.WeightPerGroup, nil
}

// lootUsageForDrop 计算单个掉落的携行占用；弹药按发数聚合前的单项结果计算。
func lootUsageForDrop(snapshot ScenarioSnapshot, drop LootDrop) (int, float64, error) {
	if drop.Quantity <= 0 {
		return 0, 0, nil
	}
	if _, ok := snapshot.Ammos[drop.ItemID]; ok {
		return ammoUsage(snapshot, drop.ItemID, drop.Quantity)
	}
	item, ok := snapshot.LootItems[drop.ItemID]
	if !ok {
		return 0, 0, fmt.Errorf("掉落物品 %s 不在场景快照目录中", drop.ItemID)
	}
	if item.Slots < 0 || item.Weight < 0 {
		return 0, 0, fmt.Errorf("掉落物品 %s 的容量配置无效", drop.ItemID)
	}
	return item.Slots * drop.Quantity, float64(item.Weight * drop.Quantity), nil
}

// lootUsageForDrops 计算整组掉落的携行占用；同种弹药先合并发数，再统一向上取整。
func lootUsageForDrops(snapshot ScenarioSnapshot, drops []LootDrop) (int, float64, error) {
	slots, weight := 0, 0.0
	ammoRounds := make(map[string]int)
	for _, drop := range drops {
		if drop.Quantity <= 0 {
			continue
		}
		if _, ok := snapshot.Ammos[drop.ItemID]; ok {
			ammoRounds[drop.ItemID] += drop.Quantity
			continue
		}
		itemSlots, itemWeight, err := lootUsageForDrop(snapshot, drop)
		if err != nil {
			return 0, 0, err
		}
		slots += itemSlots
		weight += itemWeight
	}
	for ammoID, rounds := range ammoRounds {
		ammoSlots, ammoWeight, err := ammoUsage(snapshot, ammoID, rounds)
		if err != nil {
			return 0, 0, err
		}
		slots += ammoSlots
		weight += ammoWeight
	}
	return slots, weight, nil
}

// ceilDiv 向上取整除法，供弹药发数折算格数使用。
func ceilDiv(value, divisor int) int {
	if value <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}
