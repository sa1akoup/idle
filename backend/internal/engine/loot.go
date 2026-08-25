// 纯掉落模块：按稳定排序和权重完成节点容器、容器规则与物品抽取。
package engine

import (
	"math/rand"
	"sort"
)

const NodeContainerPoolSearch = "search"

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

func filterNodeContainerPool(entries []NodeContainerAssignment, pool string) []NodeContainerAssignment {
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

func hasNodeContainerPool(entries []NodeContainerAssignment, pool string) bool {
	for _, entry := range filterNodeContainerPool(entries, pool) {
		if entry.Weight > 0 {
			return true
		}
	}
	return false
}

func chooseNodeContainerPool(entries []NodeContainerAssignment, pool string, rng *rand.Rand) (NodeContainerAssignment, bool) {
	return chooseNodeContainer(filterNodeContainerPool(entries, pool), rng)
}

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
