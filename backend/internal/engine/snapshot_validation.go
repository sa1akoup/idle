// 快照校验辅助：验证事件、容器、遭遇和规范化集合的引用完整性。
package engine

import "sort"

// hasAnyContainerPool 判断是否有某容器池的非零权重分配，空池名按默认搜索池兼容。
func hasAnyContainerPool(assignments []NodeContainerAssignment, pool string) bool {
	// 兼容旧数据：未配置池名的分配归入默认搜索池。
	for _, assignment := range assignments {
		assignmentPool := assignment.Pool
		if assignmentPool == "" {
			assignmentPool = NodeContainerPoolSearch
		}
		if assignmentPool == pool && assignment.Weight > 0 {
			return true
		}
	}
	return false
}

// normalizeSnapshot 规范化所有有序集合与缺省值，保证同一配置的 JSON 输出唯一。
func normalizeSnapshot(snapshot ScenarioSnapshot) ScenarioSnapshot {
	snapshot.SchemaVersion = valueOr(snapshot.SchemaVersion, SchemaVersion)
	snapshot.Map.Tags = sortedStrings(snapshot.Map.Tags)
	sort.SliceStable(snapshot.Nodes, func(i, j int) bool {
		if snapshot.Nodes[i].PositionY == snapshot.Nodes[j].PositionY {
			if snapshot.Nodes[i].PositionX == snapshot.Nodes[j].PositionX {
				return snapshot.Nodes[i].ID < snapshot.Nodes[j].ID
			}
			return snapshot.Nodes[i].PositionX < snapshot.Nodes[j].PositionX
		}
		return snapshot.Nodes[i].PositionY < snapshot.Nodes[j].PositionY
	})
	for i := range snapshot.Nodes {
		snapshot.Nodes[i].Tags = sortedStrings(snapshot.Nodes[i].Tags)
	}
	sort.SliceStable(snapshot.Edges, func(i, j int) bool {
		left, right := snapshot.Edges[i], snapshot.Edges[j]
		if left.MapID != right.MapID {
			return left.MapID < right.MapID
		}
		if left.FromNodeID != right.FromNodeID {
			return left.FromNodeID < right.FromNodeID
		}
		if left.ToNodeID != right.ToNodeID {
			return left.ToNodeID < right.ToNodeID
		}
		return left.ID < right.ID
	})
	for i := range snapshot.ExtractionPoints {
		snapshot.ExtractionPoints[i].Tags = sortedStrings(snapshot.ExtractionPoints[i].Tags)
	}
	sort.SliceStable(snapshot.ExtractionPoints, func(i, j int) bool { return snapshot.ExtractionPoints[i].ID < snapshot.ExtractionPoints[j].ID })
	for i := range snapshot.NodeContainerAssignments {
		if snapshot.NodeContainerAssignments[i].Pool == "" {
			snapshot.NodeContainerAssignments[i].Pool = NodeContainerPoolSearch
		}
	}
	sort.SliceStable(snapshot.NodeContainerAssignments, func(i, j int) bool {
		if snapshot.NodeContainerAssignments[i].NodeID == snapshot.NodeContainerAssignments[j].NodeID {
			if snapshot.NodeContainerAssignments[i].Pool == snapshot.NodeContainerAssignments[j].Pool {
				if snapshot.NodeContainerAssignments[i].ContainerID == snapshot.NodeContainerAssignments[j].ContainerID {
					return snapshot.NodeContainerAssignments[i].ID < snapshot.NodeContainerAssignments[j].ID
				}
				return snapshot.NodeContainerAssignments[i].ContainerID < snapshot.NodeContainerAssignments[j].ContainerID
			}
			return snapshot.NodeContainerAssignments[i].Pool < snapshot.NodeContainerAssignments[j].Pool
		}
		return snapshot.NodeContainerAssignments[i].NodeID < snapshot.NodeContainerAssignments[j].NodeID
	})
	for id, container := range snapshot.Containers {
		container.Tags = sortedStrings(container.Tags)
		sort.SliceStable(container.Rules, func(i, j int) bool { return container.Rules[i].ID < container.Rules[j].ID })
		snapshot.Containers[id] = container
	}
	for id, definition := range snapshot.Events.Definitions {
		definition.Tags = sortedStrings(definition.Tags)
		for i := range definition.Options {
			definition.Options[i].Modes = sortedStrings(definition.Options[i].Modes)
			definition.Options[i].Styles = sortedStrings(definition.Options[i].Styles)
		}
		sort.SliceStable(definition.Options, func(i, j int) bool { return definition.Options[i].ID < definition.Options[j].ID })
		snapshot.Events.Definitions[id] = definition
	}
	sort.SliceStable(snapshot.Events.Bindings, func(i, j int) bool {
		left, right := snapshot.Events.Bindings[i], snapshot.Events.Bindings[j]
		if left.EventID == right.EventID {
			if left.ScopeType == right.ScopeType {
				if left.ScopeID == right.ScopeID {
					if left.Phase == right.Phase {
						if left.Priority == right.Priority {
							return left.ID < right.ID
						}
						return left.Priority < right.Priority
					}
					return left.Phase < right.Phase
				}
				return left.ScopeID < right.ScopeID
			}
			return left.ScopeType < right.ScopeType
		}
		return left.EventID < right.EventID
	})
	for role, entries := range snapshot.Events.EncounterPools {
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].EnemyID == entries[j].EnemyID {
				return entries[i].ID < entries[j].ID
			}
			return entries[i].EnemyID < entries[j].EnemyID
		})
		snapshot.Events.EncounterPools[role] = entries
	}
	for index, preset := range snapshot.RecoveryPresets {
		SortItemStacks(preset.Consumables)
		sort.SliceStable(preset.Items, func(i, j int) bool { return preset.Items[i].ItemID < preset.Items[j].ItemID })
		snapshot.RecoveryPresets[index] = preset
	}
	sort.SliceStable(snapshot.Styles, func(i, j int) bool { return snapshot.Styles[i].ID < snapshot.Styles[j].ID })
	sort.SliceStable(snapshot.Contracts, func(i, j int) bool { return snapshot.Contracts[i].QuestID < snapshot.Contracts[j].QuestID })
	snapshot.StashKeys = sortedStrings(snapshot.StashKeys)
	return snapshot
}

// sortedStrings 返回排序后的字符串副本，不修改原切片。
func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

// valueOr 值为空时返回回退值，否则原样返回。
func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
