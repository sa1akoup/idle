// 事件配置校验：只负责启动时检查数据库配置，不参与正式探索运行时。
package service

import (
	"fmt"

	"idle/internal/engine"
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
				if _, err := engine.ResolveStyle(style); err != nil {
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
						return fmt.Errorf("事件 %s 引用不存在消耗品 %s", definition.ID, effect.Ref)
					}
				case "set_flag":
					if effect.Ref == "" {
						return fmt.Errorf("事件 %s 的标记效果缺少名称", definition.ID)
					}
				case "start_evacuation":
					if effect.Ref != "" && !supportedEvacuationReasons[effect.Ref] {
						return fmt.Errorf("事件 %s 使用未知撤离原因 %s", definition.ID, effect.Ref)
					}
				case "evac_shortcut":
					if effect.Value <= 0 {
						return fmt.Errorf("事件 %s 的撤离捷径缩短时间必须为正数", definition.ID)
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
