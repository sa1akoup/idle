// 事件配置校验：只负责启动时检查数据库配置，不参与正式探索运行时。
package service

import (
	"fmt"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

// ValidateEventConfig 在服务启动时检查事件引用、概率、作用域与地图撤离可达性。
func ValidateEventConfig(db *gorm.DB) error {
	var definitions []models.EventDef
	if err := db.Find(&definitions).Error; err != nil {
		return fmt.Errorf("校验事件定义: %w", err)
	}
	definitionByID := make(map[string]models.EventDef, len(definitions))
	for _, definition := range definitions {
		definitionByID[definition.ID] = definition
	}

	var maps []models.MapDef
	var nodes []models.NodeDef
	var bindings []models.EventBinding
	var edges []models.MapEdgeDef
	var extractionPoints []models.ExtractionPointDef
	var pools []models.EncounterPoolEntry
	var enemyTemplates []models.EnemyTemplateDef
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
	eventCatalog := engine.EventCatalog{Definitions: make(map[string]engine.EventDefinition, len(definitions)), Bindings: make([]engine.EventBinding, 0, len(bindings))}
	for _, definition := range definitions {
		converted := engine.EventDefinition{}
		if err := convertJSON(definition, &converted); err != nil {
			return fmt.Errorf("转换事件 %s: %w", definition.ID, err)
		}
		eventCatalog.Definitions[converted.ID] = converted
	}
	for _, binding := range bindings {
		converted := engine.EventBinding{}
		if err := convertJSON(binding, &converted); err != nil {
			return fmt.Errorf("转换事件绑定 %s: %w", binding.ID, err)
		}
		eventCatalog.Bindings = append(eventCatalog.Bindings, converted)
	}
	if err := engine.ValidateEventCatalog(eventCatalog, engine.DefaultStylePolicies()); err != nil {
		return fmt.Errorf("校验事件结构: %w", err)
	}
	if err := db.Find(&edges).Error; err != nil {
		return fmt.Errorf("校验地图边: %w", err)
	}
	if err := db.Find(&extractionPoints).Error; err != nil {
		return fmt.Errorf("校验撤离点: %w", err)
	}
	if err := db.Find(&pools).Error; err != nil {
		return fmt.Errorf("校验遭遇池: %w", err)
	}
	if err := db.Find(&enemyTemplates).Error; err != nil {
		return fmt.Errorf("校验敌人模板: %w", err)
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
	extractionIDs, extractionTags := map[string]bool{}, map[string]bool{}
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
		mapEdges := make([]engine.MapEdge, 0)
		for _, edge := range edges {
			if edge.MapID == gameMap.ID {
				mapEdges = append(mapEdges, engine.MapEdge{ID: edge.ID, MapID: edge.MapID, FromNodeID: edge.FromNodeID, ToNodeID: edge.ToNodeID, MoveTime: edge.MoveTime, Bidirectional: edge.Bidirectional})
			}
		}
		mapPoints := make([]engine.ExtractionPoint, 0)
		for _, point := range extractionPoints {
			if point.MapID == gameMap.ID {
				mapPoints = append(mapPoints, engine.ExtractionPoint{ID: point.ID, MapID: point.MapID, Name: point.Name, Kind: point.Kind, AnchorNodeID: point.AnchorNodeID, TravelTime: point.TravelTime, Enabled: point.Enabled, IconKey: point.IconKey, Tags: point.Tags})
			}
		}
		mapNodesSnapshot := make([]engine.Node, 0, len(mapNodes))
		for _, node := range mapNodes {
			mapNodesSnapshot = append(mapNodesSnapshot, convertNode(node))
		}
		if err := engine.ValidateMapGraph(convertMap(gameMap), mapNodesSnapshot, mapEdges, mapPoints); err != nil {
			return fmt.Errorf("地图 %s 图配置无效: %w", gameMap.ID, err)
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
	for _, point := range extractionPoints {
		if point.ID == "" || point.MapID == "" || point.AnchorNodeID == "" || point.TravelTime <= 0 {
			return fmt.Errorf("撤离点 %s 配置无效", point.ID)
		}
		extractionIDs[point.ID] = true
		for _, tag := range point.Tags {
			extractionTags[tag] = true
		}
	}
	for _, template := range enemyTemplates {
		enemyIDs[template.ID] = true
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
			if option.Check.ItemBonusRef != "" && !consumableIDs[option.Check.ItemBonusRef] {
				return fmt.Errorf("事件 %s 的判定加成引用不存在的消耗品 %s", definition.ID, option.Check.ItemBonusRef)
			}
			for _, condition := range option.Conditions {
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
		scopeValid := binding.ScopeType == "global" ||
			(binding.ScopeType == "map" && mapIDs[binding.ScopeID]) ||
			(binding.ScopeType == "node" && nodeIDs[binding.ScopeID]) ||
			(binding.ScopeType == "map_tag" && mapTags[binding.ScopeID]) ||
			(binding.ScopeType == "node_tag" && nodeTags[binding.ScopeID]) ||
			(binding.ScopeType == "extraction" && extractionIDs[binding.ScopeID]) ||
			(binding.ScopeType == "extraction_tag" && extractionTags[binding.ScopeID])
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
			if !bindingAppliesToMap(binding, gameMap, mapNodes, extractionPoints) {
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

func bindingAppliesToMap(binding models.EventBinding, gameMap models.MapDef, nodes []models.NodeDef, extractionPoints []models.ExtractionPointDef) bool {
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
	case "extraction":
		for _, point := range extractionPoints {
			if point.MapID == gameMap.ID && point.ID == binding.ScopeID && point.Enabled {
				return true
			}
		}
	case "extraction_tag":
		for _, point := range extractionPoints {
			if point.MapID == gameMap.ID && point.Enabled && containsString(point.Tags, binding.ScopeID) {
				return true
			}
		}
	}
	return false
}
