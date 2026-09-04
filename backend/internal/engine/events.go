// 纯事件模块：按快照绑定执行事件筛选、自动决策、判定和状态效果。
package engine

import (
	"fmt"
	"math/rand"
	"sort"
)

const (
	runModeExploring  = "exploring"
	runModeEvacuating = "evacuating"

	eventPhaseEnterNode              = "enter_node"
	eventPhasePreEncounter           = "pre_encounter"
	eventPhasePostEncounter          = "post_encounter"
	eventPhasePreSearch              = "pre_search"
	eventPhasePostSearch             = "post_search"
	eventPhaseEvacStart              = "evac_start"
	eventPhaseEvacStep               = "evac_step"
	eventPhaseExtractionApproach     = "extraction_approach"
	eventPhaseExtractionPointReached = "extraction_point_reached"
	eventPhaseAtExtraction           = "at_extraction"
)

// supportedEventPhase 判断事件阶段名是否属于已注册阶段集合。
func supportedEventPhase(phase string) bool {
	switch phase {
	case eventPhaseEnterNode, eventPhasePreEncounter, eventPhasePostEncounter, eventPhasePreSearch,
		eventPhasePostSearch, eventPhaseEvacStart, eventPhaseEvacStep, eventPhaseExtractionApproach,
		eventPhaseExtractionPointReached, eventPhaseAtExtraction:
		return true
	default:
		return false
	}
}

type eventManager struct {
	gameMap       Map
	definitions   map[string]EventDefinition
	bindings      []EventBinding
	encounterPool map[string][]EncounterPoolEntry
	styles        []StylePolicy
}

type eventCandidate struct {
	binding EventBinding
	def     EventDefinition
	roll    int
	option  EventOption
}

// Trigger 触发某阶段事件：过滤绑定、随机抽选并逐条执行，所有随机仅来自传入 RNG。
func (manager *eventManager) Trigger(state *eventRunState, phase string, rng *rand.Rand) error {
	if !supportedEventPhase(phase) {
		return fmt.Errorf("未知事件阶段 %s", phase)
	}
	matched := make(map[string]EventBinding)
	for _, binding := range manager.bindings {
		if !binding.Enabled || binding.Phase != phase || !manager.matchScope(binding, state) {
			continue
		}
		stored, ok := matched[binding.EventID]
		if !ok || scopeSpecificity(binding.ScopeType) > scopeSpecificity(stored.ScopeType) ||
			(scopeSpecificity(binding.ScopeType) == scopeSpecificity(stored.ScopeType) && binding.Priority > stored.Priority) {
			matched[binding.EventID] = binding
		}
	}

	eventIDs := make([]string, 0, len(matched))
	for eventID := range matched {
		eventIDs = append(eventIDs, eventID)
	}
	sort.Strings(eventIDs)
	general := make([]eventCandidate, 0, len(eventIDs))
	specific := make([]eventCandidate, 0, len(eventIDs))
	for _, eventID := range eventIDs {
		binding := matched[eventID]
		definition, ok := manager.definitions[eventID]
		if !ok || !eventRepeatAllowed(definition, binding, state) {
			continue
		}
		option, ok := selectEventOption(definition, state)
		if !ok {
			continue
		}
		roll := rng.Intn(10000) + 1
		triggerBP := binding.TriggerBP
		if definition.Category == "exploration" && state.Tuning.Encounter.IntelEventBPBonus > 0 {
			triggerBP += state.Tuning.Encounter.IntelEventBPBonus
			if triggerBP > 10000 {
				triggerBP = 10000
			}
		}
		if roll > triggerBP {
			continue
		}
		candidate := eventCandidate{binding: binding, def: definition, roll: roll, option: option}
		// 通用事件与节点级事件分池：各阶段至多各触一个，避免高相关事件被全局事件挤掉。
		if scopeSpecificity(binding.ScopeType) >= scopeSpecificity("node_tag") {
			specific = append(specific, candidate)
		} else {
			general = append(general, candidate)
		}
	}

	selected := make([]eventCandidate, 0, 2)
	if candidate, ok := chooseEventCandidate(general, rng); ok {
		selected = append(selected, candidate)
	}
	if candidate, ok := chooseEventCandidate(specific, rng); ok {
		selected = append(selected, candidate)
	}
	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].binding.Priority == selected[j].binding.Priority {
			return scopeSpecificity(selected[i].binding.ScopeType) > scopeSpecificity(selected[j].binding.ScopeType)
		}
		return selected[i].binding.Priority > selected[j].binding.Priority
	})

	usedGroups := make(map[string]bool)
	for _, candidate := range selected {
		group := candidate.def.ExclusiveGroup
		if group != "" && usedGroups[group] {
			continue
		}
		if group != "" {
			usedGroups[group] = true
		}
		if err := manager.resolveEvent(candidate, state, phase, rng); err != nil {
			return err
		}
	}
	return nil
}

// matchScope 判断事件绑定作用域是否匹配当前地图/节点/撤离点。
func (manager *eventManager) matchScope(binding EventBinding, state *eventRunState) bool {
	switch binding.ScopeType {
	case "global":
		return true
	case "map":
		return binding.ScopeID == manager.gameMap.ID
	case "map_tag":
		return containsString(manager.gameMap.Tags, binding.ScopeID)
	case "node":
		return binding.ScopeID == state.Node.ID
	case "node_tag":
		return containsString(state.Node.Tags, binding.ScopeID)
	case "extraction":
		return state.ExtractionPoint != nil && state.ExtractionPoint.ID == binding.ScopeID
	case "extraction_tag":
		return state.ExtractionPoint != nil && containsString(state.ExtractionPoint.Tags, binding.ScopeID)
	default:
		return false
	}
}

// scopeSpecificity 返回作用域具体程度数值，用于同事件多绑定时优先更具体的绑定。
func scopeSpecificity(scopeType string) int {
	switch scopeType {
	case "extraction":
		return 6
	case "node":
		return 5
	case "node_tag":
		return 4
	case "extraction_tag":
		return 4
	case "map":
		return 3
	case "map_tag":
		return 2
	case "global":
		return 1
	default:
		return 0
	}
}

// eventRepeatAllowed 按重复策略、单局上限与冷却节点数判断事件是否可再次触发。
func eventRepeatAllowed(definition EventDefinition, binding EventBinding, state *eventRunState) bool {
	count := state.EventCounts[definition.ID]
	if binding.MaxPerRun > 0 && count >= binding.MaxPerRun {
		return false
	}
	switch definition.RepeatPolicy {
	case "once_per_run":
		if count > 0 {
			return false
		}
	case "once_per_node":
		if state.LastEventVisit[definition.ID] == state.VisitSequence {
			return false
		}
	}
	lastVisit, triggered := state.LastEventVisit[definition.ID]
	return !triggered || binding.CooldownNodes <= 0 || state.VisitSequence-lastVisit > binding.CooldownNodes
}

// selectEventOption 过滤模式/风格/条件均匹配的事件选项，并按风格偏好取最优方案。
func selectEventOption(definition EventDefinition, state *eventRunState) (EventOption, bool) {
	options := append([]EventOption(nil), definition.Options...)
	sort.SliceStable(options, func(i, j int) bool { return options[i].ID < options[j].ID })
	eligible := make([]EventOption, 0, len(options))
	for _, option := range options {
		if len(option.Modes) > 0 && !containsString(option.Modes, state.Mode) {
			continue
		}
		if len(option.Styles) > 0 && !containsString(option.Styles, state.Style) {
			continue
		}
		valid := true
		for _, condition := range option.Conditions {
			if !eventConditionMatches(condition, state) {
				valid = false
				break
			}
		}
		if valid && eventOptionContainerPoolsAvailable(option, state) {
			eligible = append(eligible, option)
		}
	}
	if len(eligible) == 0 {
		return EventOption{}, false
	}
	policy := stylePolicy(state.Styles, state.Style)
	sort.SliceStable(eligible, func(i, j int) bool { return policy.optionScore(eligible[i]) > policy.optionScore(eligible[j]) })
	return eligible[0], true
}

// eventOptionContainerPoolsAvailable 校验选项引用的容器奖励池在当前节点真实存在。
func eventOptionContainerPoolsAvailable(option EventOption, state *eventRunState) bool {
	for _, effects := range [][]EventEffect{option.SuccessEffects, option.FailureEffects} {
		for _, effect := range effects {
			if effect.Type == "container_pool" && (effect.Ref == "" || state.HasContainerPool == nil || !state.HasContainerPool(effect.Ref)) {
				return false
			}
		}
	}
	return true
}

// eventConditionMatches 把当前状态换算成条件值并与操作符比较，未知条件类型视为不匹配。
func eventConditionMatches(condition EventCondition, state *eventRunState) bool {
	var actual float64
	switch condition.Type {
	case "hp_ratio":
		actual = state.Player.HP / state.Player.MaxHP
	case "stress_ratio":
		actual = state.Player.Stress / state.Player.StressThreshold
	case "ammo":
		if len(state.AmmoStacks) > 0 {
			actual = float64(state.totalAmmoRounds())
		} else {
			actual = float64(state.Player.AmmoRounds)
		}
	case "heat":
		actual = float64(state.Heat)
	case "carry_ratio":
		actual = state.carryRatio()
	case "has_item":
		if state.hasItem(condition.Ref) {
			actual = 1
		}
	case "flag":
		if state.Flags[condition.Ref] {
			actual = 1
		}
	default:
		return false
	}
	switch condition.Operator {
	case "eq":
		return actual == condition.Value
	case "ne":
		return actual != condition.Value
	case "lt":
		return actual < condition.Value
	case "lte":
		return actual <= condition.Value
	case "gt":
		return actual > condition.Value
	case "gte":
		return actual >= condition.Value
	default:
		return false
	}
}

// carryRatio 返回负重占用比例：格子和重量两侧取更满的一侧。
func (state *eventRunState) carryRatio() float64 {
	ratio := 0.0
	if state.CarrySlots > 0 {
		ratio = float64(state.LootSlots) / float64(state.CarrySlots)
	}
	if state.CarryWeight > 0 && state.LootWeight/state.CarryWeight > ratio {
		ratio = state.LootWeight / state.CarryWeight
	}
	return ratio
}

// chooseEventCandidate 按绑定权重随机挑出一个候选事件，权重非正值按 1 参与。
func chooseEventCandidate(candidates []eventCandidate, rng *rand.Rand) (eventCandidate, bool) {
	if len(candidates) == 0 {
		return eventCandidate{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].binding.EventID == candidates[j].binding.EventID {
			return candidates[i].binding.ID < candidates[j].binding.ID
		}
		return candidates[i].binding.EventID < candidates[j].binding.EventID
	})
	totalWeight := 0
	for _, candidate := range candidates {
		weight := candidate.binding.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}
	roll := rng.Intn(totalWeight)
	for _, candidate := range candidates {
		weight := candidate.binding.Weight
		if weight <= 0 {
			weight = 1
		}
		if roll < weight {
			return candidate, true
		}
		roll -= weight
	}
	return candidates[len(candidates)-1], true
}

// resolveEvent 执行选定事件：掷判定骰、写文案、应用成败效果并记录轨迹。
func (manager *eventManager) resolveEvent(candidate eventCandidate, state *eventRunState, phase string, rng *rand.Rand) error {
	state.EventCounts[candidate.def.ID]++
	state.LastEventVisit[candidate.def.ID] = state.VisitSequence
	intent := candidate.option.Intent
	*state.Lines = append(*state.Lines, fmt.Sprintf("  [事件/%s/%s] %s，风格%s，触发 %d/10000，掷 %d，采用方案 %s(%s，风险%d/收益%d)", phase, candidate.binding.ScopeType, candidate.def.Name, state.Style, candidate.binding.TriggerBP, candidate.roll, candidate.option.ID, intent, candidate.option.RiskTier, candidate.option.ValueTier))
	success, checkLine := resolveEventCheck(candidate.option.Check, candidate.option, state, rng)
	if checkLine != "" {
		*state.Lines = append(*state.Lines, "    "+checkLine)
	}
	if success && candidate.option.Check.Type == "attribute" {
		state.creditSkill(candidate.option.Check.Attribute)
	}
	text := candidate.option.FailureText
	effects := candidate.option.FailureEffects
	if success {
		text = candidate.option.SuccessText
		effects = candidate.option.SuccessEffects
	}
	if text != "" {
		*state.Lines = append(*state.Lines, "    "+text)
	}
	for _, effect := range effects {
		summary, err := applyEventEffect(effect, state)
		if err != nil {
			return fmt.Errorf("执行事件 %s: %w", candidate.def.ID, err)
		}
		if summary != "" {
			*state.Lines = append(*state.Lines, "    效果："+summary)
		}
	}
	state.addTrace(TraceEventTriggered, state.DurationSec, state.Node.ID, candidate.def.ID, map[string]interface{}{
		"phase": phase, "name": candidate.def.Name, "scopeType": candidate.binding.ScopeType,
		"optionId": candidate.option.ID, "intent": intent, "success": success,
		"roll": candidate.roll, "triggerBP": candidate.binding.TriggerBP,
	})
	return nil
}
