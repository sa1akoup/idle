// 通用事件管理器：负责多层作用域筛选、概率触发、自动选项、判定与强类型效果执行。
package service

import (
	"fmt"
	"math/rand"
	"sort"

	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	runModeExploring  = "exploring"
	runModeEvacuating = "evacuating"

	eventPhaseEnterNode     = "enter_node"
	eventPhasePreEncounter  = "pre_encounter"
	eventPhasePostEncounter = "post_encounter"
	eventPhasePreSearch     = "pre_search"
	eventPhasePostSearch    = "post_search"
	eventPhaseEvacStart     = "evac_start"
	eventPhaseEvacStep      = "evac_step"
	eventPhaseAtExtraction  = "at_extraction"
)

var supportedEventPhases = map[string]bool{
	eventPhaseEnterNode: true, eventPhasePreEncounter: true, eventPhasePostEncounter: true,
	eventPhasePreSearch: true, eventPhasePostSearch: true, eventPhaseEvacStart: true,
	eventPhaseEvacStep: true, eventPhaseAtExtraction: true,
}

var supportedEventEffects = map[string]bool{
	"hp": true, "stress": true, "heat": true, "time": true, "armor": true, "ammo": true,
	"container": true, "container_pool": true, "encounter": true, "skip_combat": true, "skip_search": true,
	"start_evacuation": true, "set_flag": true, "consume_item": true,
	"discard_loot": true, "evac_shortcut": true,
}

var supportedEventAttributes = map[string]bool{
	"strength": true, "agility": true, "intellect": true, "charisma": true,
	"stealth": true, "perception": true, "negotiation": true, "luck": true,
	"survival": true, "resist": true, "engineering": true, "medical": true,
}

var supportedEventConditions = map[string]bool{
	"hp_ratio": true, "stress_ratio": true, "ammo": true, "heat": true,
	"carry_ratio": true, "has_item": true, "flag": true,
}

var supportedEventIntents = map[string]bool{
	"bypass": true, "ambush": true, "engage": true, "force": true,
	"conceal": true, "secure": true, "search": true, "loot": true,
	"intel": true, "unlock": true, "rush": true, "withdraw": true,
	"treat": true, "drop": true, "reroute": true, "wait": true,
}

var supportedConditionOperators = map[string]bool{
	"eq": true, "ne": true, "lt": true, "lte": true, "gt": true, "gte": true,
}

var supportedEvacuationReasons = map[string]bool{
	"health": true, "stress": true, "ammo": true, "armor": true,
	"carry_full": true, "target_acquired": true, "event": true,
}

type eventManager struct {
	gameMap       models.MapDef
	definitions   map[string]models.EventDef
	bindings      []models.EventBinding
	encounterPool map[string][]models.EncounterPoolEntry
}

// eventRunState 是事件、战斗、搜索与撤离共用的单局状态。
type eventRunState struct {
	Character *models.Character
	Player    *BattleActor
	Node      models.NodeDef
	Mode      string
	Style     ActionStyle

	EvacuationReason    string
	EvacuationEmergency bool
	EvacuationPending   bool
	EvacuationStarted   bool

	Duration      int
	Heat          int
	AmmoUsed      int
	CarrySlots    int
	CarryWeight   float64
	LootSlots     int
	LootWeight    float64
	CarryBlocked  bool
	VisitSequence int

	SkipDefaultCombat bool
	SkipSearch        bool
	EncounterRole     string
	EvacShortcut      bool

	AvailableItems map[string]bool
	ConsumedItems  map[string]int
	Flags          map[string]bool
	EventCounts    map[string]int
	LastEventVisit map[string]int
	Lines          *[]string

	CollectContainer     func(containerID, source string) error
	CollectContainerPool func(poolID, source string, count int) error
	HasContainerPool     func(poolID string) bool
	DiscardLoot          func(quantity int) int
}

func (state *eventRunState) resetNodeActions() {
	state.SkipDefaultCombat = false
	state.SkipSearch = false
	state.EncounterRole = ""
	state.EvacShortcut = false
}

func (state *eventRunState) hasItem(itemID string) bool {
	return itemID != "" && state.AvailableItems[itemID]
}

func (state *eventRunState) consumeItem(itemID string) bool {
	if !state.hasItem(itemID) {
		return false
	}
	state.AvailableItems[itemID] = false
	state.ConsumedItems[itemID]++
	return true
}

func (state *eventRunState) beginEvacuation(reason string, emergency bool) bool {
	if state.Mode == runModeEvacuating {
		if emergency && !state.EvacuationEmergency {
			state.EvacuationEmergency = true
			state.EvacuationReason = reason
			*state.Lines = append(*state.Lines, fmt.Sprintf(">> 撤离状态升级为紧急：%s", evacuationReasonName(reason)))
		}
		return false
	}
	state.Mode = runModeEvacuating
	state.EvacuationReason = reason
	state.EvacuationEmergency = emergency
	state.EvacuationPending = true
	level := "常规"
	if emergency {
		level = "紧急"
	}
	*state.Lines = append(*state.Lines, fmt.Sprintf(">> 产生%s撤离意图：%s，开始规划撤离路线", level, evacuationReasonName(reason)))
	return true
}

func evacuationReasonName(reason string) string {
	switch reason {
	case "health":
		return "生命过低"
	case "stress":
		return "压力过高"
	case "ammo":
		return "弹药耗尽"
	case "armor":
		return "护甲损坏"
	case "carry_full":
		return "携行容量接近上限"
	case "target_acquired":
		return "已获得高价值目标"
	case "event":
		return "事件要求撤离"
	default:
		return reason
	}
}

func loadEventManager(db *gorm.DB, gameMap models.MapDef) (*eventManager, error) {
	var definitions []models.EventDef
	if err := db.Find(&definitions).Error; err != nil {
		return nil, fmt.Errorf("读取事件定义: %w", err)
	}
	var bindings []models.EventBinding
	if err := db.Where("enabled = ?", true).Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("读取事件绑定: %w", err)
	}
	var poolEntries []models.EncounterPoolEntry
	if err := db.Where("map_id = ?", gameMap.ID).Find(&poolEntries).Error; err != nil {
		return nil, fmt.Errorf("读取遭遇池: %w", err)
	}

	manager := &eventManager{
		gameMap: gameMap, definitions: make(map[string]models.EventDef, len(definitions)),
		bindings: bindings, encounterPool: make(map[string][]models.EncounterPoolEntry),
	}
	for _, definition := range definitions {
		manager.definitions[definition.ID] = definition
	}
	for _, entry := range poolEntries {
		manager.encounterPool[entry.Role] = append(manager.encounterPool[entry.Role], entry)
	}
	return manager, nil
}

type eventCandidate struct {
	binding models.EventBinding
	def     models.EventDef
	roll    int
	option  models.EventOption
}

func (manager *eventManager) Trigger(state *eventRunState, phase string, rng *rand.Rand) error {
	if !supportedEventPhases[phase] {
		return fmt.Errorf("未知事件阶段 %s", phase)
	}

	// 同一事件存在多层绑定时使用最具体的一层，方便节点覆盖通用概率。
	matched := make(map[string]models.EventBinding)
	for _, binding := range manager.bindings {
		if binding.Phase != phase || !manager.matchScope(binding, state.Node) {
			continue
		}
		stored, ok := matched[binding.EventID]
		if !ok || scopeSpecificity(binding.ScopeType) > scopeSpecificity(stored.ScopeType) ||
			(scopeSpecificity(binding.ScopeType) == scopeSpecificity(stored.ScopeType) && binding.Priority > stored.Priority) {
			matched[binding.EventID] = binding
		}
	}

	general := make([]eventCandidate, 0)
	specific := make([]eventCandidate, 0)
	for _, binding := range matched {
		definition, ok := manager.definitions[binding.EventID]
		if !ok || !eventRepeatAllowed(definition, binding, state) {
			continue
		}
		option, ok := selectEventOption(definition, state)
		if !ok {
			continue
		}
		roll := rng.Intn(10000) + 1
		if roll > binding.TriggerBP {
			continue
		}
		candidate := eventCandidate{binding: binding, def: definition, roll: roll, option: option}
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

func (manager *eventManager) matchScope(binding models.EventBinding, node models.NodeDef) bool {
	switch binding.ScopeType {
	case "global":
		return true
	case "map":
		return binding.ScopeID == manager.gameMap.ID
	case "map_tag":
		return containsString(manager.gameMap.Tags, binding.ScopeID)
	case "node":
		return binding.ScopeID == node.ID
	case "node_tag":
		return containsString(node.Tags, binding.ScopeID)
	default:
		return false
	}
}

func scopeSpecificity(scopeType string) int {
	switch scopeType {
	case "node":
		return 5
	case "node_tag":
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

func eventRepeatAllowed(definition models.EventDef, binding models.EventBinding, state *eventRunState) bool {
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

func selectEventOption(definition models.EventDef, state *eventRunState) (models.EventOption, bool) {
	options := make([]models.EventOption, 0, len(definition.Options))
	for _, option := range definition.Options {
		option = normalizeEventOption(definition, option)
		if len(option.Modes) > 0 && !containsString(option.Modes, state.Mode) {
			continue
		}
		if len(option.Styles) > 0 && !containsString(option.Styles, string(state.Style)) {
			continue
		}
		eligible := true
		for _, condition := range option.Conditions {
			if !eventConditionMatches(condition, state) {
				eligible = false
				break
			}
		}
		if eligible {
			if !eventOptionContainerPoolsAvailable(option, state) {
				continue
			}
			options = append(options, option)
		}
	}
	if len(options) == 0 {
		return models.EventOption{}, false
	}
	policy := actionStylePolicy(state.Style)
	sort.SliceStable(options, func(i, j int) bool {
		return policy.optionScore(options[i]) > policy.optionScore(options[j])
	})
	return options[0], true
}

func eventOptionContainerPoolsAvailable(option models.EventOption, state *eventRunState) bool {
	for _, effects := range [][]models.EventEffect{option.SuccessEffects, option.FailureEffects} {
		for _, effect := range effects {
			if effect.Type != "container_pool" {
				continue
			}
			if effect.Ref == "" || state.HasContainerPool == nil || !state.HasContainerPool(effect.Ref) {
				return false
			}
		}
	}
	return true
}

func eventConditionMatches(condition models.EventCondition, state *eventRunState) bool {
	var actual float64
	switch condition.Type {
	case "hp_ratio":
		actual = state.Player.HP / state.Player.MaxHP
	case "stress_ratio":
		actual = state.Player.Stress / state.Player.StressThreshold
	case "ammo":
		actual = float64(state.Player.Ammo)
	case "heat":
		actual = float64(state.Heat)
	case "carry_ratio":
		actual = state.carryRatio()
	case "has_item":
		actual = 0
		if state.hasItem(condition.Ref) {
			actual = 1
		}
	case "flag":
		actual = 0
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

func (state *eventRunState) carryRatio() float64 {
	ratio := 0.0
	if state.CarrySlots > 0 {
		ratio = float64(state.LootSlots) / float64(state.CarrySlots)
	}
	if state.CarryWeight > 0 {
		weightRatio := state.LootWeight / state.CarryWeight
		if weightRatio > ratio {
			ratio = weightRatio
		}
	}
	return ratio
}

func chooseEventCandidate(candidates []eventCandidate, rng *rand.Rand) (eventCandidate, bool) {
	if len(candidates) == 0 {
		return eventCandidate{}, false
	}
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

func (manager *eventManager) resolveEvent(candidate eventCandidate, state *eventRunState, phase string, rng *rand.Rand) error {
	state.EventCounts[candidate.def.ID]++
	state.LastEventVisit[candidate.def.ID] = state.VisitSequence
	intent := candidate.option.Intent
	if intent == "" {
		intent = inferEventIntent(candidate.def, candidate.option.ID)
	}
	*state.Lines = append(*state.Lines, fmt.Sprintf(
		"  [事件/%s/%s] %s，风格%s，触发 %d/10000，掷 %d，采用方案 %s(%s，风险%d/收益%d)",
		phase, candidate.binding.ScopeType, candidate.def.Name, state.Style, candidate.binding.TriggerBP, candidate.roll, candidate.option.ID, intent, candidate.option.RiskTier, candidate.option.ValueTier,
	))

	success, checkLine := resolveEventCheck(candidate.option.Check, candidate.option, state, rng)
	if checkLine != "" {
		*state.Lines = append(*state.Lines, "    "+checkLine)
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
	return nil
}

func resolveEventCheck(check models.EventCheck, option models.EventOption, state *eventRunState, rng *rand.Rand) (bool, string) {
	switch check.Type {
	case "", "none":
		return true, ""
	case "fixed":
		probability := int(clamp(float64(check.Target), 5, 95))
		roll := rng.Intn(100) + 1
		return roll <= probability, fmt.Sprintf("固定判定 %d%%，掷 %d", probability, roll)
	case "attribute":
		value := getAttrValue(state.Character, check.Attribute)
		probability := 55 + int(float64(value-check.Target)*1.1)
		if state.hasItem(check.ItemBonusRef) {
			probability += check.ItemBonus
		}
		probability += actionStylePolicy(state.Style).checkBonus(option)
		probability = int(clamp(float64(probability), 5, 95))
		roll := rng.Intn(100) + 1
		return roll <= probability, fmt.Sprintf("%s=%d，成功率 %d%%，掷 %d", check.Attribute, value, probability, roll)
	default:
		return false, fmt.Sprintf("未知判定类型 %s", check.Type)
	}
}

func applyEventEffect(effect models.EventEffect, state *eventRunState) (string, error) {
	switch effect.Type {
	case "hp":
		state.Player.HP = clamp(state.Player.HP+float64(effect.Value), 0, state.Player.MaxHP)
		return fmt.Sprintf("生命 %+d，当前 %.0f/%.0f", effect.Value, state.Player.HP, state.Player.MaxHP), nil
	case "stress":
		state.Player.Stress = clamp(state.Player.Stress+float64(effect.Value), 0, state.Player.StressThreshold)
		return fmt.Sprintf("压力 %+d，当前 %.0f/%.0f", effect.Value, state.Player.Stress, state.Player.StressThreshold), nil
	case "heat":
		state.Heat = maxInt(state.Heat+effect.Value, 0)
		return fmt.Sprintf("热度 %+d，当前 %d", effect.Value, state.Heat), nil
	case "time":
		state.Duration = maxInt(state.Duration+effect.Value, 0)
		return fmt.Sprintf("行动时间 %+d 分钟", effect.Value), nil
	case "armor":
		state.Player.ArmorDurability = clamp(state.Player.ArmorDurability+float64(effect.Value), 0, state.Player.ArmorMaxDur)
		return fmt.Sprintf("护甲耐久 %+d，当前 %.0f", effect.Value, state.Player.ArmorDurability), nil
	case "ammo":
		state.Player.Ammo = maxInt(state.Player.Ammo+effect.Value, 0)
		if effect.Value < 0 {
			state.AmmoUsed += -effect.Value
		}
		return fmt.Sprintf("弹药 %+d，当前 %d", effect.Value, state.Player.Ammo), nil
	case "container":
		if state.CollectContainer == nil {
			return "", fmt.Errorf("容器收集器未初始化")
		}
		if err := state.CollectContainer(effect.Ref, "事件"); err != nil {
			return "", err
		}
		return "搜索容器 " + effect.Ref, nil
	case "container_pool":
		if state.CollectContainerPool == nil {
			return "", fmt.Errorf("事件奖励容器收集器未初始化")
		}
		count := effect.Value
		if count <= 0 {
			count = 1
		}
		if err := state.CollectContainerPool(effect.Ref, "事件奖励", count); err != nil {
			return "", err
		}
		return fmt.Sprintf("按权重搜索事件奖励池 %s x%d", effect.Ref, count), nil
	case "encounter":
		state.EncounterRole = effect.Ref
		return "遭遇池切换为 " + effect.Ref, nil
	case "skip_combat":
		state.SkipDefaultCombat = true
		state.EncounterRole = ""
		return "避开本节点交战", nil
	case "skip_search":
		state.SkipSearch = true
		return "跳过本节点搜索", nil
	case "start_evacuation":
		reason := effect.Ref
		if reason == "" {
			reason = "event"
		}
		state.beginEvacuation(reason, effect.Value > 0)
		return "进入撤离模式", nil
	case "set_flag":
		state.Flags[effect.Ref] = effect.Value >= 0
		return "记录局内标记 " + effect.Ref, nil
	case "consume_item":
		if state.consumeItem(effect.Ref) {
			return "消耗 " + effect.Ref, nil
		}
		return "未携带可消耗的 " + effect.Ref, nil
	case "discard_loot":
		if state.DiscardLoot == nil {
			return "", fmt.Errorf("物资丢弃器未初始化")
		}
		discarded := state.DiscardLoot(maxInt(effect.Value, 1))
		return fmt.Sprintf("丢弃 %d 件物资", discarded), nil
	case "evac_shortcut":
		state.EvacShortcut = true
		return "发现通往撤离点的临时捷径", nil
	default:
		return "", fmt.Errorf("未知事件效果 %s", effect.Type)
	}
}

func (manager *eventManager) ResolveEnemyID(role string, rng *rand.Rand) (string, error) {
	entries := manager.encounterPool[role]
	if len(entries) == 0 {
		return "", fmt.Errorf("地图 %s 未配置遭遇角色 %s", manager.gameMap.ID, role)
	}
	totalWeight := 0
	for _, entry := range entries {
		totalWeight += maxInt(entry.Weight, 1)
	}
	roll := rng.Intn(totalWeight)
	for _, entry := range entries {
		weight := maxInt(entry.Weight, 1)
		if roll < weight {
			return entry.EnemyID, nil
		}
		roll -= weight
	}
	return entries[len(entries)-1].EnemyID, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// ValidateEventConfig 在服务启动时检查事件引用、概率、作用域与地图撤离可达性。
func ValidateEventConfig(db *gorm.DB) error {
	var definitions []models.EventDef
	if err := db.Find(&definitions).Error; err != nil {
		return fmt.Errorf("校验事件定义: %w", err)
	}
	definitionByID := make(map[string]models.EventDef, len(definitions))
	for _, definition := range definitions {
		if len(definition.Options) == 0 {
			return fmt.Errorf("事件 %s 没有处理方案", definition.ID)
		}
		for _, option := range definition.Options {
			for _, effect := range append(append([]models.EventEffect{}, option.SuccessEffects...), option.FailureEffects...) {
				if !supportedEventEffects[effect.Type] {
					return fmt.Errorf("事件 %s 使用未知效果 %s", definition.ID, effect.Type)
				}
			}
			for _, style := range option.Styles {
				if _, ok := stylePolicies[ActionStyle(style)]; !ok {
					return fmt.Errorf("事件 %s 使用未知行动风格 %s", definition.ID, style)
				}
			}
			if option.Intent != "" && !supportedEventIntents[option.Intent] {
				return fmt.Errorf("事件 %s 使用未知决策意图 %s", definition.ID, option.Intent)
			}
			if option.RiskTier < 0 || option.RiskTier > 5 || option.ValueTier < 0 || option.ValueTier > 5 {
				return fmt.Errorf("事件 %s 的风险或收益等级无效", definition.ID)
			}
		}
		definitionByID[definition.ID] = definition
	}

	var maps []models.MapDef
	var nodes []models.NodeDef
	var bindings []models.EventBinding
	var pools []models.EncounterPoolEntry
	var enemies []models.EnemyDef
	var containers []models.LootContainerDef
	var nodeContainers []models.NodeContainerDef
	var consumables []models.ConsumableDef
	if err := db.Find(&maps).Error; err != nil {
		return fmt.Errorf("校验事件地图: %w", err)
	}
	if err := db.Find(&nodes).Error; err != nil {
		return fmt.Errorf("校验事件节点: %w", err)
	}
	if err := db.Find(&bindings).Error; err != nil {
		return fmt.Errorf("校验事件绑定: %w", err)
	}
	if err := db.Find(&pools).Error; err != nil {
		return fmt.Errorf("校验遭遇池: %w", err)
	}
	if err := db.Find(&enemies).Error; err != nil {
		return fmt.Errorf("校验敌人引用: %w", err)
	}
	if err := db.Find(&containers).Error; err != nil {
		return fmt.Errorf("校验容器引用: %w", err)
	}
	if err := db.Find(&nodeContainers).Error; err != nil {
		return fmt.Errorf("校验节点容器池: %w", err)
	}
	if err := db.Find(&consumables).Error; err != nil {
		return fmt.Errorf("校验消耗品引用: %w", err)
	}

	mapIDs, nodeIDs, enemyIDs, containerIDs, consumableIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	mapTags, nodeTags := map[string]bool{}, map[string]bool{}
	for _, gameMap := range maps {
		mapIDs[gameMap.ID] = true
		for _, tag := range gameMap.Tags {
			mapTags[tag] = true
		}
		mapNodes := make([]models.NodeDef, 0)
		for _, node := range nodes {
			if node.MapID == gameMap.ID {
				mapNodes = append(mapNodes, node)
			}
		}
		if len(mapNodes) == 0 {
			return fmt.Errorf("地图 %s 没有节点", gameMap.ID)
		}
		if err := validateDirectedRoute(mapNodes, gameMap); err != nil {
			return fmt.Errorf("地图 %s 撤离路线无效: %w", gameMap.ID, err)
		}
	}
	for _, node := range nodes {
		if node.ValueTier < 1 || node.ValueTier > 5 || node.ContainerSlots < 0 || node.ExploreTime < 0 {
			return fmt.Errorf("节点 %s 的价值、容器槽位或耗时配置无效", node.ID)
		}
		nodeIDs[node.ID] = true
		for _, tag := range node.Tags {
			nodeTags[tag] = true
		}
	}
	for _, enemy := range enemies {
		enemyIDs[enemy.ID] = true
	}
	for _, node := range nodes {
		if node.EnemyID != "" && !enemyIDs[node.EnemyID] {
			return fmt.Errorf("节点 %s 引用不存在的敌人 %s", node.ID, node.EnemyID)
		}
	}
	for _, container := range containers {
		if container.ValueTier < 1 || container.ValueTier > 5 || container.SearchRisk < 0 || container.SearchRisk > 5 || container.SearchTime < 0 || container.RollMin < 0 || container.RollMax < container.RollMin {
			return fmt.Errorf("容器 %s 的价值、风险、耗时或抽取范围无效", container.ID)
		}
		containerIDs[container.ID] = true
	}
	for _, consumable := range consumables {
		consumableIDs[consumable.ID] = true
	}
	nodeContainerWeights := make(map[string]int)
	nodeContainerCounts := make(map[string]int)
	containerPoolIDs := make(map[string]bool)
	containerPoolWeights := make(map[string]int)
	for _, assignment := range nodeContainers {
		if !nodeIDs[assignment.NodeID] || !containerIDs[assignment.ContainerID] || assignment.Weight < 0 || assignment.Count < 0 {
			return fmt.Errorf("节点容器挂载 %d 引用无效", assignment.ID)
		}
		pool := assignment.Pool
		if pool == "" {
			pool = models.NodeContainerPoolSearch
		}
		containerPoolIDs[pool] = true
		containerPoolWeights[pool] += assignment.Weight
		if pool == models.NodeContainerPoolSearch {
			nodeContainerWeights[assignment.NodeID] += assignment.Weight
			nodeContainerCounts[assignment.NodeID]++
		}
	}
	for _, node := range nodes {
		if node.ContainerSlots > 0 && (nodeContainerCounts[node.ID] == 0 || nodeContainerWeights[node.ID] <= 0) {
			return fmt.Errorf("节点 %s 配置了%d个容器槽位，但没有可抽取的容器池", node.ID, node.ContainerSlots)
		}
	}

	for _, definition := range definitions {
		for _, option := range definition.Options {
			if option.Check.Type != "" && option.Check.Type != "none" && option.Check.Type != "fixed" && option.Check.Type != "attribute" {
				return fmt.Errorf("事件 %s 使用未知判定类型 %s", definition.ID, option.Check.Type)
			}
			if option.Check.Type == "attribute" && !supportedEventAttributes[option.Check.Attribute] {
				return fmt.Errorf("事件 %s 使用未知判定属性 %s", definition.ID, option.Check.Attribute)
			}
			if option.Check.ItemBonusRef != "" && !consumableIDs[option.Check.ItemBonusRef] {
				return fmt.Errorf("事件 %s 的判定加成引用不存在的消耗品 %s", definition.ID, option.Check.ItemBonusRef)
			}
			for _, mode := range option.Modes {
				if mode != runModeExploring && mode != runModeEvacuating {
					return fmt.Errorf("事件 %s 使用未知模式 %s", definition.ID, mode)
				}
			}
			for _, condition := range option.Conditions {
				if !supportedEventConditions[condition.Type] || !supportedConditionOperators[condition.Operator] {
					return fmt.Errorf("事件 %s 使用未知条件 %s/%s", definition.ID, condition.Type, condition.Operator)
				}
				if condition.Type == "has_item" && (condition.Ref == "" || !consumableIDs[condition.Ref]) {
					return fmt.Errorf("事件 %s 的物品条件引用无效 %s", definition.ID, condition.Ref)
				}
				if condition.Type == "flag" && condition.Ref == "" {
					return fmt.Errorf("事件 %s 的条件 %s 缺少引用", definition.ID, condition.Type)
				}
			}
		}
	}

	for _, binding := range bindings {
		if _, ok := definitionByID[binding.EventID]; !ok {
			return fmt.Errorf("事件绑定 %s 引用不存在的事件 %s", binding.ID, binding.EventID)
		}
		if !supportedEventPhases[binding.Phase] || binding.TriggerBP < 0 || binding.TriggerBP > 10000 || binding.Weight < 0 || binding.MaxPerRun < 0 || binding.CooldownNodes < 0 {
			return fmt.Errorf("事件绑定 %s 的阶段、概率或限制无效", binding.ID)
		}
		scopeValid := binding.ScopeType == "global" ||
			(binding.ScopeType == "map" && mapIDs[binding.ScopeID]) ||
			(binding.ScopeType == "node" && nodeIDs[binding.ScopeID]) ||
			(binding.ScopeType == "map_tag" && mapTags[binding.ScopeID]) ||
			(binding.ScopeType == "node_tag" && nodeTags[binding.ScopeID])
		if !scopeValid {
			return fmt.Errorf("事件绑定 %s 的作用域无效", binding.ID)
		}
	}

	rolesByMap := make(map[string]map[string]bool)
	for _, pool := range pools {
		if !mapIDs[pool.MapID] || !enemyIDs[pool.EnemyID] || pool.Role == "" || pool.Weight <= 0 {
			return fmt.Errorf("遭遇池条目 %s 引用无效", pool.ID)
		}
		if rolesByMap[pool.MapID] == nil {
			rolesByMap[pool.MapID] = make(map[string]bool)
		}
		rolesByMap[pool.MapID][pool.Role] = true
	}

	rolesByEvent := make(map[string]map[string]bool)
	for _, definition := range definitions {
		roles := make(map[string]bool)
		for _, option := range definition.Options {
			for _, effect := range append(append([]models.EventEffect{}, option.SuccessEffects...), option.FailureEffects...) {
				switch effect.Type {
				case "container":
					if effect.Ref == "" || !containerIDs[effect.Ref] {
						return fmt.Errorf("事件 %s 引用不存在的容器 %s", definition.ID, effect.Ref)
					}
				case "container_pool":
					if effect.Ref == "" || !containerPoolIDs[effect.Ref] || containerPoolWeights[effect.Ref] <= 0 {
						return fmt.Errorf("事件 %s 引用不存在的节点事件奖励池 %s", definition.ID, effect.Ref)
					}
				case "encounter":
					if effect.Ref == "" {
						return fmt.Errorf("事件 %s 的遭遇效果缺少角色", definition.ID)
					}
					roles[effect.Ref] = true
				case "consume_item":
					if effect.Ref == "" || !consumableIDs[effect.Ref] {
						return fmt.Errorf("事件 %s 引用不存在的消耗品 %s", definition.ID, effect.Ref)
					}
				case "set_flag":
					if effect.Ref == "" {
						return fmt.Errorf("事件 %s 的标记效果缺少名称", definition.ID)
					}
				case "start_evacuation":
					if effect.Ref != "" && !supportedEvacuationReasons[effect.Ref] {
						return fmt.Errorf("事件 %s 使用未知撤离原因 %s", definition.ID, effect.Ref)
					}
				}
			}
		}
		rolesByEvent[definition.ID] = roles
	}

	// 只对实际绑定到当前地图的事件检查遭遇角色，允许不同地图拥有不同敌人池。
	for _, gameMap := range maps {
		mapNodes := make([]models.NodeDef, 0)
		for _, node := range nodes {
			if node.MapID == gameMap.ID {
				mapNodes = append(mapNodes, node)
			}
		}
		for _, binding := range bindings {
			if !bindingAppliesToMap(binding, gameMap, mapNodes) {
				continue
			}
			for role := range rolesByEvent[binding.EventID] {
				if !rolesByMap[gameMap.ID][role] {
					return fmt.Errorf("地图 %s 的事件 %s 缺少遭遇角色 %s", gameMap.ID, binding.EventID, role)
				}
			}
		}
	}
	return nil
}

func bindingAppliesToMap(binding models.EventBinding, gameMap models.MapDef, nodes []models.NodeDef) bool {
	switch binding.ScopeType {
	case "global":
		return true
	case "map":
		return binding.ScopeID == gameMap.ID
	case "map_tag":
		return containsString(gameMap.Tags, binding.ScopeID)
	case "node":
		for _, node := range nodes {
			if node.ID == binding.ScopeID {
				return true
			}
		}
	case "node_tag":
		for _, node := range nodes {
			if containsString(node.Tags, binding.ScopeID) {
				return true
			}
		}
	}
	return false
}
