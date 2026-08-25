// 快照校验辅助：验证事件、容器、遭遇和规范化集合的引用完整性。
package engine

import "sort"

func hasStyle(styles []StylePolicy, style string) bool {
	for _, policy := range styles {
		if policy.ID == style {
			return true
		}
	}
	return false
}

func validCondition(condition EventCondition) bool {
	validType := map[string]bool{"hp_ratio": true, "stress_ratio": true, "ammo": true, "heat": true, "carry_ratio": true, "has_item": true, "flag": true}
	validOperator := map[string]bool{"eq": true, "ne": true, "lt": true, "lte": true, "gt": true, "gte": true}
	return validType[condition.Type] && validOperator[condition.Operator]
}

func validEffect(effect EventEffect) bool {
	switch effect.Type {
	case "hp", "stress", "heat", "time", "armor", "ammo", "container", "container_pool", "encounter", "skip_combat", "skip_search", "start_evacuation", "set_flag", "consume_item", "discard_loot", "evac_shortcut":
		return true
	default:
		return false
	}
}

func hasAnyContainerPool(assignments []NodeContainerAssignment, pool string) bool {
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

func normalizeSnapshot(snapshot ScenarioSnapshot) ScenarioSnapshot {
	snapshot.SchemaVersion = valueOr(snapshot.SchemaVersion, SchemaVersion)
	snapshot.Map.Tags = sortedStrings(snapshot.Map.Tags)
	sort.SliceStable(snapshot.Nodes, func(i, j int) bool {
		if snapshot.Nodes[i].RouteOrder == snapshot.Nodes[j].RouteOrder {
			return snapshot.Nodes[i].ID < snapshot.Nodes[j].ID
		}
		return snapshot.Nodes[i].RouteOrder < snapshot.Nodes[j].RouteOrder
	})
	for i := range snapshot.Nodes {
		snapshot.Nodes[i].Tags = sortedStrings(snapshot.Nodes[i].Tags)
	}
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
	return snapshot
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
