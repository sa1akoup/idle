// 单局探索细节：处理节点推进、搜索、事件和撤离过程。
package engine

import (
	"fmt"
	"math"
	"math/rand"
)

func simulateSingleRun(snapshot ScenarioSnapshot, character CharacterState, weapon Weapon, armor Armor, armorDurability int, ammo Ammo, ammoRounds int, consumables []ItemStack, carriedItems []CarriedItem, itemUseDefs map[string]ItemUseDefinition, nodes []Node, rng *rand.Rand, style string, carrySlots int, carryWeight float64, runIndex int) (*simulatedRun, error) {
	byID := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	if _, ok := byID[snapshot.Map.StartNodeID]; !ok {
		return nil, fmt.Errorf("起点节点 %s 不存在", snapshot.Map.StartNodeID)
	}
	routePlan, err := PlanRoute(snapshot, style, rng, RoutePlannerOptions{
		MaxRouteNodes: len(nodes), MaxCandidates: 512,
	})
	if err != nil {
		return nil, fmt.Errorf("规划本局探索路线: %w", err)
	}
	if err := ValidateRoutePlan(snapshot, routePlan); err != nil {
		return nil, fmt.Errorf("校验本局探索路线: %w", err)
	}
	adjacency := buildAdjacency(snapshot.Edges)
	events := newEventManager(snapshot)
	materialized := materializeNodeContainers(nodes, snapshot.NodeContainerAssignments, rng)
	lines := []string{fmt.Sprintf("=== 第%d局开始 地图:%s 风格:%s ===", runIndex, snapshot.Map.Name, stylePolicy(snapshot.Styles, style).Label)}
	trace := make([]TraceEvent, 0, len(nodes)*4)
	appendTraceEvent(&trace, TraceRunStarted, 0, "", "", map[string]interface{}{
		"runIndex": runIndex, "mapId": snapshot.Map.ID, "mapName": snapshot.Map.Name, "style": style,
	})
	appendTraceEvent(&trace, TraceRoutePlanned, 0, snapshot.Map.StartNodeID, routePlan.ExtractionID, map[string]interface{}{
		"route": append([]string(nil), routePlan.NodeIDs...), "extractionId": routePlan.ExtractionID,
		"anchorNodeId": routePlan.AnchorNodeID,
	})
	playerActor := buildPlayerActor(character, weapon, armor, armorDurability, ammo, ammoRounds)
	availableItems := make(map[string]int, len(consumables))
	for _, item := range consumables {
		availableItems[item.ItemID] += item.Quantity
	}
	state := &eventRunState{
		Character: &character, Player: &playerActor, Mode: runModeExploring, Style: style, Styles: snapshot.Styles,
		CarrySlots: carrySlots, CarryWeight: carryWeight, AvailableItems: availableItems, CarriedItems: CloneCarriedItems(carriedItems), ItemUseDefs: itemUseDefs,
		ConsumedItems: make(map[string]int), Flags: make(map[string]bool), EventCounts: make(map[string]int),
		LastEventVisit: make(map[string]int), Lines: &lines, Trace: &trace,
	}
	loot := make([]LootDrop, 0)
	state.CollectContainer = func(containerID, source string) error {
		container, ok := snapshot.Containers[containerID]
		if !ok {
			return fmt.Errorf("容器 %s 不存在", containerID)
		}
		searchStart := state.DurationSec
		state.addTrace(TraceContainerSearchStarted, searchStart, state.Node.ID, containerID, map[string]interface{}{
			"containerId": container.ID, "name": container.Name, "source": source,
			"searchTime": container.SearchTime, "valueTier": container.ValueTier,
		})
		state.DurationSec += int64(container.SearchTime) * 60
		lines = append(lines, fmt.Sprintf("  搜索容器 %s（标签:%s，价值%d级，风险%d，耗时%d分钟）", container.Name, joinStrings(container.Tags, "/"), container.ValueTier, container.SearchRisk, container.SearchTime))
		rolls := container.RollMin
		if container.RollMax > container.RollMin {
			rolls += rng.Intn(container.RollMax - container.RollMin + 1)
		}
		if rolls <= 0 {
			lines = append(lines, "    容器为空")
			state.addTrace(TraceContainerSearchFinished, state.DurationSec, state.Node.ID, containerID, map[string]interface{}{
				"containerId": container.ID, "status": "empty", "quantity": 0,
			})
			return nil
		}
		foundQuantity := 0
		for i := 0; i < rolls; i++ {
			rule, ok := chooseContainerRule(container, rng)
			if !ok {
				lines = append(lines, "    容器没有可用掉落规则")
				break
			}
			item, ok := snapshot.LootItemsByCategory(rule.ItemCategory, rng)
			if !ok {
				lines = append(lines, fmt.Sprintf("    %s 分类暂无物品", rule.ItemCategory))
				continue
			}
			quantity := rule.MinQuantity
			if rule.MaxQuantity > rule.MinQuantity {
				quantity += rng.Intn(rule.MaxQuantity - rule.MinQuantity + 1)
			}
			needSlots := item.Slots * quantity
			needWeight := float64(item.Weight * quantity)
			if quantity <= 0 || state.LootSlots+needSlots > state.CarrySlots || state.LootWeight+needWeight > state.CarryWeight {
				if quantity > 0 {
					state.CarryBlocked = true
					lines = append(lines, fmt.Sprintf("    容量不足，放弃 %s x%d", item.Name, quantity))
					state.addTrace(TraceLootFound, state.DurationSec, state.Node.ID, item.ID, map[string]interface{}{
						"itemId": item.ID, "name": item.Name, "category": item.Category, "quantity": quantity,
						"containerId": containerID, "source": source, "collected": false, "reason": "carry_full",
					})
				}
				continue
			}
			loot = append(loot, LootDrop{ItemID: item.ID, Quantity: quantity, ContainerID: containerID, Source: source})
			foundQuantity += quantity
			state.LootSlots += needSlots
			state.LootWeight += needWeight
			lines = append(lines, fmt.Sprintf("    获得 %s x%d", item.Name, quantity))
			state.addTrace(TraceLootFound, state.DurationSec, state.Node.ID, item.ID, map[string]interface{}{
				"itemId": item.ID, "name": item.Name, "category": item.Category, "quantity": quantity,
				"containerId": containerID, "source": source, "collected": true,
			})
			state.addTrace(TraceLootCollected, state.DurationSec, state.Node.ID, item.ID, map[string]interface{}{
				"itemId": item.ID, "name": item.Name, "category": item.Category, "quantity": quantity,
				"containerId": containerID, "source": source,
			})
		}
		state.addTrace(TraceContainerSearchFinished, state.DurationSec, state.Node.ID, containerID, map[string]interface{}{
			"containerId": container.ID, "status": "completed", "quantity": foundQuantity,
		})
		return nil
	}
	state.HasContainerPool = func(poolID string) bool {
		return poolID != "" && hasNodeContainerPool(snapshot.NodeContainerAssignmentsForNode(state.Node.ID), poolID)
	}
	state.CollectContainerPool = func(poolID, source string, count int) error {
		if count <= 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			assignment, ok := chooseNodeContainerPool(snapshot.NodeContainerAssignmentsForNode(state.Node.ID), poolID, rng)
			if !ok {
				return fmt.Errorf("节点 %s 没有可用事件奖励池 %s", state.Node.ID, poolID)
			}
			lines = append(lines, fmt.Sprintf("  事件奖励池 %s 按权重抽到 %s", poolID, assignment.ContainerID))
			if err := state.CollectContainer(assignment.ContainerID, source); err != nil {
				return err
			}
			if state.CarryBlocked {
				break
			}
		}
		return nil
	}
	state.DiscardLoot = func(quantity int) int {
		discarded := 0
		for i := len(loot) - 1; i >= 0 && discarded < quantity; i-- {
			remove := minInt(loot[i].Quantity, quantity-discarded)
			loot[i].Quantity -= remove
			item := snapshot.LootItems[loot[i].ItemID]
			state.LootSlots -= item.Slots * remove
			state.LootWeight -= float64(item.Weight * remove)
			discarded += remove
			if loot[i].Quantity == 0 {
				loot = append(loot[:i], loot[i+1:]...)
			}
		}
		return discarded
	}

	result := ""
	finishedSession := false
	for step, currentNodeID := range routePlan.NodeIDs {
		moveDurationSec := int64(0)
		if step > 0 {
			previousNodeID := routePlan.NodeIDs[step-1]
			moveTime := edgeMoveTime(adjacency[previousNodeID], currentNodeID)
			if moveTime <= 0 {
				return nil, fmt.Errorf("路线缺少移动边 %s -> %s", previousNodeID, currentNodeID)
			}
			actualMoveSec := state.consumeNextMoveDuration(int64(moveTime) * 60)
			moveDurationSec = actualMoveSec
			state.addTrace(TraceNodeMoveStarted, state.DurationSec, previousNodeID, currentNodeID, map[string]interface{}{
				"fromNodeId": previousNodeID, "toNodeId": currentNodeID, "moveTime": moveTime,
				"actualMoveTimeSec": actualMoveSec,
			})
			state.DurationSec += actualMoveSec
		}
		node, ok := byID[currentNodeID]
		if !ok {
			return nil, fmt.Errorf("路线节点 %s 不存在", currentNodeID)
		}
		state.Node = node
		state.ExtractionPoint = nil
		state.VisitSequence++
		state.resetNodeActions()
		modeAtEntry := state.Mode
		minutes := int(state.DurationSec / 60)
		nodeDurationSec := int64(0)
		if modeAtEntry == runModeExploring {
			lines = append(lines, fmt.Sprintf("[%02d:%02d] 进入节点 %s，探索%d分钟，距离%s", minutes/60, minutes%60, node.Name, node.ExploreTime, node.Distance))
			nodeDurationSec = int64(node.ExploreTime) * 60
		} else {
			lines = append(lines, fmt.Sprintf("[%02d:%02d] 撤离途中抵达 %s，距离%s", minutes/60, minutes%60, node.Name, node.Distance))
		}
		state.addTrace(TraceNodeEntered, state.DurationSec, node.ID, node.ID, map[string]interface{}{
			"name": node.Name, "mode": modeAtEntry, "exploreTime": node.ExploreTime, "distance": node.Distance,
			"heat": state.Heat, "playerHp": state.Player.HP, "playerMaxHp": state.Player.MaxHP,
			"playerStress": state.Player.Stress, "playerAmmo": state.Player.AmmoRounds,
			"playerArmorDurability": state.Player.ArmorDurability,
		})
		state.DurationSec += nodeDurationSec
		actualDurationSec := nodeDurationSec + moveDurationSec

		if err := events.Trigger(state, eventPhaseEnterNode, rng); err != nil {
			return nil, err
		}
		evaluateAutomaticEvacuation(state, weapon)
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}
		if state.Mode == runModeEvacuating {
			if err := events.Trigger(state, eventPhaseEvacStep, rng); err != nil {
				return nil, err
			}
			evaluateAutomaticEvacuation(state, weapon)
		}
		if err := events.Trigger(state, eventPhasePreEncounter, rng); err != nil {
			return nil, err
		}
		evaluateAutomaticEvacuation(state, weapon)
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}

		enemy, enemyDefeated, encounterCleared, encountered, err := resolveNodeEncounter(snapshot, state, events, node, rng)
		if err != nil {
			return nil, err
		}
		if state.Player.HP <= 0 {
			lines = append(lines, ">> 玩家失去行动能力")
			result = "incapacitated"
			finishedSession = true
			break
		}
		if err := events.Trigger(state, eventPhasePostEncounter, rng); err != nil {
			return nil, err
		}
		maybeAutoRecoverNeeds(state)
		maybeAutoHeal(state)
		evaluateAutomaticEvacuation(state, weapon)
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}
		if state.Mode == runModeExploring && enemyDefeated && enemy.BackpackContainerID != "" {
			if err := state.CollectContainer(enemy.BackpackContainerID, "敌人背包"); err != nil {
				return nil, err
			}
		}
		canSearch := !encountered || encounterCleared
		if state.Mode == runModeExploring && canSearch && !state.SkipSearch {
			if err := events.Trigger(state, eventPhasePreSearch, rng); err != nil {
				return nil, err
			}
			evaluateAutomaticEvacuation(state, weapon)
			if state.Mode == runModeExploring && !state.SkipSearch {
				for _, assignment := range materialized[node.ID] {
					for i := 0; i < assignment.Count; i++ {
						if err := state.CollectContainer(assignment.ContainerID, "节点:"+node.Name); err != nil {
							return nil, err
						}
						if state.CarryBlocked {
							break
						}
					}
					if state.CarryBlocked {
						break
					}
				}
				if err := events.Trigger(state, eventPhasePostSearch, rng); err != nil {
					return nil, err
				}
			}
			evaluateAutomaticEvacuation(state, weapon)
			if err := startEvacuationEvents(events, state, rng); err != nil {
				return nil, err
			}
		}

		// 每经过一个节点的移动/探索时间都产生减压效果。
		stressRecovery := float64(actualDurationSec) / 60 * 5
		state.Player.Stress = clamp(state.Player.Stress-stressRecovery, 0, state.Player.StressThreshold)
		if node.ID != routePlan.AnchorNodeID {
			continue
		}

		point, ok := extractionPointByID(snapshot.ExtractionPoints, routePlan.ExtractionID)
		if !ok {
			return nil, fmt.Errorf("撤离点 %s 不存在", routePlan.ExtractionID)
		}
		if state.Mode != runModeEvacuating {
			state.beginEvacuation("route_complete", false)
		}
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}
		state.ExtractionPoint = &point
		state.addTrace(TraceExtractionApproach, state.DurationSec, node.ID, point.ID, map[string]interface{}{
			"extractionId": point.ID, "name": point.Name, "anchorNodeId": point.AnchorNodeID, "travelTime": point.TravelTime,
		})
		if err := events.Trigger(state, eventPhaseExtractionApproach, rng); err != nil {
			return nil, err
		}
		state.DurationSec += int64(point.TravelTime) * 60
		state.Player.Stress = clamp(state.Player.Stress-float64(point.TravelTime)*5, 0, state.Player.StressThreshold)
		state.addTrace(TraceExtractionPointReached, state.DurationSec, node.ID, point.ID, map[string]interface{}{
			"extractionId": point.ID, "name": point.Name, "travelTime": point.TravelTime,
		})
		if err := events.Trigger(state, eventPhaseExtractionPointReached, rng); err != nil {
			return nil, err
		}
		// 撤离点事件可以设置 encounter；只有明确设置了撤离遭遇角色时，
		// 才交给统一战斗解析器，避免把锚点节点的普通敌人重复处理一次。
		extractionEncountered := false
		if state.EncounterRole != "" {
			_, _, _, extractionEncountered, err = resolveNodeEncounter(snapshot, state, events, node, rng)
			if err != nil {
				return nil, err
			}
		}
		if extractionEncountered && state.Player.HP <= 0 {
			lines = append(lines, ">> 玩家在撤离点交战中失去行动能力")
			result = "incapacitated"
			finishedSession = true
			break
		}
		if err := events.Trigger(state, eventPhaseAtExtraction, rng); err != nil {
			return nil, err
		}
		maybeAutoRecoverNeeds(state)
		maybeAutoHeal(state)
		if state.Player.HP <= 0 {
			lines = append(lines, ">> 玩家在撤离点失去行动能力")
			result = "incapacitated"
			finishedSession = true
			break
		}
		result = "success"
		state.addTrace(TraceExtractionCompleted, state.DurationSec, node.ID, point.ID, map[string]interface{}{
			"extractionId": point.ID, "name": point.Name, "result": result,
		})
		lines = append(lines, fmt.Sprintf(">> 抵达撤离点 %s，完成%s", point.Name, extractionLabel(result)))
		break
	}
	if result == "" {
		return nil, fmt.Errorf("规划路线未抵达撤离锚点")
	}

	if result != "incapacitated" {
		result = "success"
	}
	character.HP = state.Player.HP
	applyNeedDrain(&character, state.DurationSec)
	maybeAutoRecoverNeeds(state)
	lines = append(lines, fmt.Sprintf("=== 本局结束 结果:%s 耗时:%d分 热度:%d 弹药:%d HP:%.1f 能量:%.1f 饮水:%.1f ===", result, state.DurationSec/60, state.Heat, state.AmmoUsed, character.HP, character.Energy, character.Hydration))
	if len(loot) == 0 {
		lines = append(lines, ">> 本局没有搜集到可带回的物资")
	} else {
		lines = append(lines, fmt.Sprintf(">> 本局搜集到 %d 件物品，占用 %d 格 / %.1fkg", lootQuantity(loot), state.LootSlots, state.LootWeight))
	}
	assignLootDropIDs(loot, runIndex)
	extractedLoot := selectExtractedLoot(result, loot)
	if lootQuantity(extractedLoot) < lootQuantity(loot) {
		lines = append(lines, fmt.Sprintf(">> %s仅保留 %d 件物品", extractionLabel(result), lootQuantity(extractedLoot)))
	}
	return &simulatedRun{
		result: result, durationSec: state.DurationSec, heat: state.Heat, ammoUsed: state.AmmoUsed,
		ammoRounds: state.Player.AmmoRounds,
		playerHP:   state.Player.HP, energy: character.Energy, hydration: character.Hydration,
		playerStress: state.Player.Stress,
		finished:     finishedSession, armorDurability: int(math.Round(state.Player.ArmorDurability)), carriedItems: CloneCarriedItems(state.CarriedItems),
		loot: cloneLoot(loot), extractedLoot: extractedLoot, consumedItems: state.ConsumedItems, report: lines, trace: trace,
	}, nil
}
