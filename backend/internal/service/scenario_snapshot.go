// 场景快照构建器：在 Session 创建时把探索运行所需的配置固定为纯引擎 DTO。
package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

// buildScenarioSnapshot 在事务内构建场景快照，并返回其规范 JSON 与 hash。
func buildScenarioSnapshot(db *gorm.DB, userID uint, mapID string) (engine.ScenarioSnapshot, string, string, error) {
	var snapshot engine.ScenarioSnapshot
	var encoded, hash string
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		snapshot, encoded, hash, err = buildScenarioSnapshotTx(tx, userID, mapID)
		return err
	})
	if err != nil {
		return engine.ScenarioSnapshot{}, "", "", err
	}
	return snapshot, encoded, hash, nil
}

// buildScenarioSnapshotTx 读取地图、节点、容器与各类目录定义，物化敌人并组装出纯引擎快照。
func buildScenarioSnapshotTx(db *gorm.DB, userID uint, mapID string) (engine.ScenarioSnapshot, string, string, error) {
	var gameMap models.MapDef
	if err := db.First(&gameMap, "id = ?", mapID).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取场景地图: %w", err)
	}
	var nodes []models.NodeDef
	if err := db.Where("map_id = ?", mapID).Order("position_y asc, position_x asc, id asc").Find(&nodes).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取场景节点: %w", err)
	}
	if len(nodes) == 0 {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("地图没有可探索节点")
	}
	var edges []models.MapEdgeDef
	if err := db.Where("map_id = ?", mapID).Order("from_node_id asc, to_node_id asc, id asc").Find(&edges).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取地图边: %w", err)
	}
	var extractionPoints []models.ExtractionPointDef
	if err := db.Where("map_id = ?", mapID).Order("id asc").Find(&extractionPoints).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取撤离点: %w", err)
	}
	nodeIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	var assignments []models.NodeContainerDef
	if err := db.Where("node_id IN ?", nodeIDs).Order("node_id asc, pool asc, container_id asc, id asc").Find(&assignments).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取节点容器池: %w", err)
	}
	var containerDefs []models.LootContainerDef
	if err := db.Order("id asc").Find(&containerDefs).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取容器目录: %w", err)
	}
	var containerRules []models.LootContainerRule
	if err := db.Order("container_id asc, id asc").Find(&containerRules).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取容器规则: %w", err)
	}
	var lootDefs []models.LootItemDef
	if err := db.Order("id asc").Find(&lootDefs).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取 loot 目录: %w", err)
	}
	var weapons []models.WeaponDef
	if err := db.Order("id asc").Find(&weapons).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取武器目录: %w", err)
	}
	var ammos []models.AmmoDef
	if err := db.Order("caliber_id asc, level asc, id asc").Find(&ammos).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取弹药目录: %w", err)
	}
	weaponMerchant, err := GetMerchantByIDForUser(db, userID, "weapon")
	if err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取武器商人弹药补给状态: %w", err)
	}
	var armors []models.ArmorDef
	if err := db.Order("id asc").Find(&armors).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取护甲目录: %w", err)
	}
	var consumables []models.ConsumableDef
	if err := db.Order("id asc").Find(&consumables).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取消耗品目录: %w", err)
	}
	var itemUseDefs []models.ItemUseDef
	if err := db.Order("item_id asc").Find(&itemUseDefs).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取物品效果目录: %w", err)
	}
	var chestRigs []models.ChestRigDef
	if err := db.Order("id asc").Find(&chestRigs).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取胸挂目录: %w", err)
	}
	var backpacks []models.BackpackDef
	if err := db.Order("id asc").Find(&backpacks).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取背包目录: %w", err)
	}
	var helmets []models.HelmetDef
	if err := db.Order("id asc").Find(&helmets).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取头盔目录: %w", err)
	}
	var headsets []models.HeadsetDef
	if err := db.Order("id asc").Find(&headsets).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取耳机目录: %w", err)
	}
	var secureContainers []models.SecureContainerDef
	if db.Migrator().HasTable(&models.SecureContainerDef{}) {
		if err := db.Order("id asc").Find(&secureContainers).Error; err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取安全箱目录: %w", err)
		}
	}
	var enemyTemplates []models.EnemyTemplateDef
	if err := db.Order("sort_order asc, id asc").Find(&enemyTemplates).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取敌人模板: %w", err)
	}
	var eventDefs []models.EventDef
	if err := db.Order("id asc").Find(&eventDefs).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取事件定义: %w", err)
	}
	var bindings []models.EventBinding
	if err := db.Order("event_id asc, scope_type asc, scope_id asc, phase asc, priority asc, id asc").Find(&bindings).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取事件绑定: %w", err)
	}
	var encounterPools []models.EncounterPoolEntry
	if err := db.Where("map_id = ?", mapID).Order("role asc, enemy_id asc, id asc").Find(&encounterPools).Error; err != nil {
		return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("读取遭遇池: %w", err)
	}

	snapshot := engine.ScenarioSnapshot{
		SchemaVersion:            engine.SchemaVersion,
		Map:                      convertMap(gameMap),
		Nodes:                    make([]engine.Node, 0, len(nodes)),
		Edges:                    make([]engine.MapEdge, 0, len(edges)),
		ExtractionPoints:         make([]engine.ExtractionPoint, 0, len(extractionPoints)),
		NodeContainerAssignments: make([]engine.NodeContainerAssignment, 0, len(assignments)),
		Containers:               make(map[string]engine.Container, len(containerDefs)),
		LootItems:                make(map[string]engine.LootItem, len(lootDefs)),
		Items:                    make(map[string]engine.ItemDefinition),
		ItemUseDefs:              make(map[string]engine.ItemUseDefinition, len(itemUseDefs)),
		Weapons:                  make(map[string]engine.Weapon, len(weapons)),
		Ammos:                    make(map[string]engine.Ammo, len(ammos)),
		AmmoSupplies:             make(map[string]engine.AmmoSupply, len(ammos)),
		Armors:                   make(map[string]engine.Armor, len(armors)),
		Headsets:                 make(map[string]engine.Headset, len(headsets)),
		Enemies:                  make(map[string]engine.Enemy, len(enemyTemplates)*2),
		Events: engine.EventCatalog{
			Definitions:    make(map[string]engine.EventDefinition, len(eventDefs)),
			Bindings:       make([]engine.EventBinding, 0, len(bindings)),
			EncounterPools: make(map[string][]engine.EncounterPoolEntry),
		},
		Styles: engine.DefaultStylePolicies(),
		Tuning: engine.DefaultTuning(),
	}

	// 敌人生成：模板 → 变体（确定性 RNG），重写节点与遭遇池引用。
	// Heat/PlayerLevel 在快照构建期不可得，缩放仅用节点价值档（温和缩放）。
	generatedEnemies, nodeEnemyIDs, poolEnemyIDs, err := materializeEnemies(
		enemyTemplates, nodes, encounterPools, weapons, armors, ammos, containerDefs,
	)
	if err != nil {
		return engine.ScenarioSnapshot{}, "", "", err
	}
	for i, node := range nodes {
		if id, ok := nodeEnemyIDs[node.ID]; ok {
			node.EnemyID = id
			nodes[i] = node
		}
	}
	for i, entry := range encounterPools {
		if id, ok := poolEnemyIDs[entry.ID]; ok {
			entry.EnemyID = id
			encounterPools[i] = entry
		}
	}

	for _, node := range nodes {
		snapshot.Nodes = append(snapshot.Nodes, convertNode(node))
	}
	for _, edge := range edges {
		snapshot.Edges = append(snapshot.Edges, engine.MapEdge{
			ID: edge.ID, MapID: edge.MapID, FromNodeID: edge.FromNodeID, ToNodeID: edge.ToNodeID,
			MoveTime: edge.MoveTime, Bidirectional: edge.Bidirectional,
		})
	}
	for _, point := range extractionPoints {
		snapshot.ExtractionPoints = append(snapshot.ExtractionPoints, engine.ExtractionPoint{
			ID: point.ID, MapID: point.MapID, Name: point.Name, Kind: point.Kind, AnchorNodeID: point.AnchorNodeID,
			TravelTime: point.TravelTime, Enabled: point.Enabled, IconKey: point.IconKey, Tags: append([]string(nil), point.Tags...),
		})
	}
	for _, assignment := range assignments {
		snapshot.NodeContainerAssignments = append(snapshot.NodeContainerAssignments, engine.NodeContainerAssignment{
			ID: assignment.ID, NodeID: assignment.NodeID, ContainerID: assignment.ContainerID, Pool: assignment.Pool,
			Count: assignment.Count, Weight: assignment.Weight,
		})
	}
	for _, definition := range containerDefs {
		snapshot.Containers[definition.ID] = engine.Container{
			ID: definition.ID, Name: definition.Name, Tags: append([]string(nil), definition.Tags...), ValueTier: definition.ValueTier,
			SearchRisk: definition.SearchRisk, SearchTime: definition.SearchTime, RollMin: definition.RollMin, RollMax: definition.RollMax,
		}
	}
	for _, rule := range containerRules {
		container, ok := snapshot.Containers[rule.ContainerID]
		if !ok {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("容器规则 %d 引用不存在容器 %s", rule.ID, rule.ContainerID)
		}
		container.Rules = append(container.Rules, engine.ContainerRule{ID: rule.ID, ItemCategory: rule.ItemCategory, Weight: rule.Weight, MinQuantity: rule.MinQuantity, MaxQuantity: rule.MaxQuantity})
		snapshot.Containers[rule.ContainerID] = container
	}
	for _, definition := range lootDefs {
		item := engine.LootItem{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换 loot %s: %w", definition.ID, err)
		}
		snapshot.LootItems[item.ID] = item
		snapshot.Items[item.ID] = engine.ItemDefinition{ID: item.ID, Kind: "loot", Name: item.Name, Category: item.Category, Price: item.Price, Weight: item.Weight, Slots: item.Slots, MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement}
	}
	for _, definition := range weapons {
		item := engine.Weapon{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换武器 %s: %w", definition.ID, err)
		}
		snapshot.Weapons[item.ID] = item
		snapshot.Items[item.ID] = engine.ItemDefinition{ID: item.ID, Kind: "weapon", Name: item.Name, Price: item.Price, Weight: item.Weight, Slots: item.Slots, MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement}
	}
	for _, definition := range ammos {
		item := engine.Ammo{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换弹药 %s: %w", definition.ID, err)
		}
		snapshot.Ammos[item.ID] = item
		snapshot.AmmoSupplies[item.ID] = engine.AmmoSupply{
			AmmoID: item.ID, CaliberID: item.CaliberID, Level: item.Level,
			UnitPrice: roundPrice(item.Price, buyMultiplier(weaponMerchant.Reputation)),
			Available: weaponMerchant.Open && item.MerchantCategory == weaponMerchant.Category &&
				item.Level <= 4 && item.RepRequirement <= weaponMerchant.Reputation,
		}
		snapshot.Items[item.ID] = engine.ItemDefinition{
			ID: item.ID, Kind: "ammo", Name: item.Name, Price: item.Price, Slots: 1,
			MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement,
		}
	}
	for _, definition := range armors {
		item := engine.Armor{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换护甲 %s: %w", definition.ID, err)
		}
		snapshot.Armors[item.ID] = item
		snapshot.Items[item.ID] = engine.ItemDefinition{ID: item.ID, Kind: "armor", Name: item.Name, Price: item.Price, Weight: item.Weight, Slots: item.Slots, MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement, ArmorMax: item.MaxDurability}
	}
	for _, definition := range consumables {
		item := engine.ItemDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换消耗品 %s: %w", definition.ID, err)
		}
		item.Kind = "consumable"
		snapshot.Items[item.ID] = item
	}
	for _, definition := range itemUseDefs {
		item := engine.ItemUseDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换物品效果 %s: %w", definition.ItemID, err)
		}
		snapshot.ItemUseDefs[item.ItemID] = item
	}
	for _, definition := range chestRigs {
		item := engine.ItemDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换胸挂 %s: %w", definition.ID, err)
		}
		item.Kind = "chestrig"
		snapshot.Items[item.ID] = item
	}
	for _, definition := range backpacks {
		item := engine.ItemDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换背包 %s: %w", definition.ID, err)
		}
		item.Kind = "backpack"
		snapshot.Items[item.ID] = item
	}
	for _, definition := range helmets {
		item := engine.ItemDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换头盔 %s: %w", definition.ID, err)
		}
		item.Kind = "helmet"
		snapshot.Items[item.ID] = item
	}
	for _, definition := range headsets {
		item := engine.ItemDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换耳机 %s: %w", definition.ID, err)
		}
		item.Kind = "headset"
		snapshot.Items[item.ID] = item
		snapshot.Headsets[item.ID] = engine.Headset{
			ID: definition.ID, Name: definition.Name, HearingLevel: definition.HearingLevel,
			Price: definition.Price, MerchantCategory: definition.MerchantCategory, RepRequirement: definition.RepRequirement,
		}
	}
	for _, definition := range secureContainers {
		snapshot.Items[definition.ID] = engine.ItemDefinition{
			ID: definition.ID, Kind: "secure", Name: definition.Name, Price: definition.Price,
			Weight: definition.Weight, Slots: definition.Slots, AddSlots: definition.InnerSlots,
			MerchantCategory: definition.MerchantCategory, RepRequirement: definition.RepRequirement,
		}
	}
	for _, definition := range generatedEnemies {
		snapshot.Enemies[definition.ID] = definition
	}
	for _, definition := range eventDefs {
		item := engine.EventDefinition{}
		if err := convertJSON(definition, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换事件 %s: %w", definition.ID, err)
		}
		snapshot.Events.Definitions[item.ID] = item
	}
	for _, binding := range bindings {
		item := engine.EventBinding{}
		if err := convertJSON(binding, &item); err != nil {
			return engine.ScenarioSnapshot{}, "", "", fmt.Errorf("转换事件绑定 %s: %w", binding.ID, err)
		}
		snapshot.Events.Bindings = append(snapshot.Events.Bindings, item)
	}
	for _, entry := range encounterPools {
		item := engine.EncounterPoolEntry{ID: entry.ID, MapID: entry.MapID, Role: entry.Role, EnemyID: entry.EnemyID, Weight: entry.Weight}
		snapshot.Events.EncounterPools[item.Role] = append(snapshot.Events.EncounterPools[item.Role], item)
	}
	if err := attachRecoveryPresets(db, userID, nil, &snapshot); err != nil {
		return engine.ScenarioSnapshot{}, "", "", err
	}
	intelBonus, physicalGrowth, err := hideoutProgressionBonusesTx(db, userID)
	if err != nil {
		return engine.ScenarioSnapshot{}, "", "", err
	}
	snapshot.Tuning.Encounter.IntelReconBonus = float64(intelBonus)
	snapshot.Tuning.Encounter.IntelEventBPBonus = intelBonus * 20
	snapshot.Tuning.Encounter.PhysicalSkillGrowthPercent = physicalGrowth
	contracts, err := activeQuestContractsTx(db, userID)
	if err != nil {
		return engine.ScenarioSnapshot{}, "", "", err
	}
	snapshot.Contracts = contracts

	encoded, hash, err := finalizeScenarioSnapshot(snapshot)
	if err != nil {
		return engine.ScenarioSnapshot{}, "", "", err
	}
	return snapshot, encoded, hash, nil
}

// raidExtractStashKeysTx 读取仓库中局内带出的钥匙，供本局 has_item 判定使用，不占用携带格。
func raidExtractStashKeysTx(db *gorm.DB, userID uint, lootDefs []models.LootItemDef) ([]string, error) {
	keyIDs := make([]string, 0)
	for _, definition := range lootDefs {
		if definition.Category == "key" {
			keyIDs = append(keyIDs, definition.ID)
		}
	}
	if len(keyIDs) == 0 {
		return nil, nil
	}
	var inventories []models.Inventory
	if err := db.Where("user_id = ? AND raid_extract = ? AND quantity > 0 AND item_id IN ?", userID, true, keyIDs).
		Order("item_id asc").Find(&inventories).Error; err != nil {
		return nil, fmt.Errorf("读取仓库钥匙: %w", err)
	}
	keys := make([]string, 0, len(inventories))
	seen := make(map[string]bool, len(inventories))
	for _, inventory := range inventories {
		if seen[inventory.ItemID] {
			continue
		}
		seen[inventory.ItemID] = true
		keys = append(keys, inventory.ItemID)
	}
	return keys, nil
}

// convertMap 把数据库地图定义转为引擎 DTO，标签做一次切片拷贝避免共享底层数组。
func convertMap(definition models.MapDef) engine.Map {
	return engine.Map{ID: definition.ID, Name: definition.Name, Desc: definition.Desc, StartNodeID: definition.StartNodeID, LayoutColumns: definition.LayoutColumns, LayoutRows: definition.LayoutRows, Tags: append([]string(nil), definition.Tags...)}
}

// convertNode 把数据库节点定义转为引擎 DTO，保留敌人生成后回填的 EnemyID。
func convertNode(definition models.NodeDef) engine.Node {
	return engine.Node{ID: definition.ID, MapID: definition.MapID, Name: definition.Name, PositionX: definition.PositionX, PositionY: definition.PositionY, ExploreTime: definition.ExploreTime, Distance: definition.Distance, EnemyID: definition.EnemyID, EncounterRole: definition.EncounterRole, ContainerSlots: definition.ContainerSlots, ValueTier: definition.ValueTier, EncounterChance: definition.EncounterChance, Tags: append([]string(nil), definition.Tags...)}
}

// materializeEnemies 把模板引用物化为敌人变体：为每个引用（节点/遭遇池）生成独立变体，
// 并返回 变体列表 + 节点ID→变体ID + 遭遇池ID→变体ID 映射。
// 目录来自快照构建主流程已读取的 models 切片。
func materializeEnemies(
	templates []models.EnemyTemplateDef,
	nodes []models.NodeDef,
	pools []models.EncounterPoolEntry,
	weapons []models.WeaponDef,
	armors []models.ArmorDef,
	ammos []models.AmmoDef,
	containers []models.LootContainerDef,
) ([]engine.Enemy, map[string]string, map[string]string, error) {
	templatesByID := make(map[string]models.EnemyTemplateDef, len(templates))
	for _, t := range templates {
		templatesByID[t.ID] = t
	}
	catalog := EnemyGenerator{
		Weapons:    make(map[string]engine.Weapon, len(weapons)),
		Armors:     make(map[string]engine.Armor, len(armors)),
		Ammos:      make(map[string]engine.Ammo, len(ammos)),
		Containers: make(map[string]engine.Container, len(containers)),
	}
	for i := range weapons {
		item := engine.Weapon{}
		if err := convertJSON(&weapons[i], &item); err != nil {
			return nil, nil, nil, fmt.Errorf("转换武器 %s: %w", weapons[i].ID, err)
		}
		catalog.Weapons[item.ID] = item
	}
	for i := range armors {
		item := engine.Armor{}
		if err := convertJSON(&armors[i], &item); err != nil {
			return nil, nil, nil, fmt.Errorf("转换护甲 %s: %w", armors[i].ID, err)
		}
		catalog.Armors[item.ID] = item
	}
	for i := range ammos {
		item := engine.Ammo{}
		if err := convertJSON(&ammos[i], &item); err != nil {
			return nil, nil, nil, fmt.Errorf("转换弹药 %s: %w", ammos[i].ID, err)
		}
		catalog.Ammos[item.ID] = item
	}
	for i := range containers {
		item := engine.Container{}
		if err := convertJSON(&containers[i], &item); err != nil {
			return nil, nil, nil, fmt.Errorf("转换容器 %s: %w", containers[i].ID, err)
		}
		catalog.Containers[item.ID] = item
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var generated []engine.Enemy
	nodeEnemyIDs := make(map[string]string)
	poolEnemyIDs := make(map[string]string)
	ctx := GenerateContext{}

	// 节点引用：每个节点生成独立变体，ValueTier 作为缩放输入
	for _, node := range nodes {
		template, ok := templatesByID[node.EnemyID]
		if !ok || node.EnemyID == "" {
			continue
		}
		ctx.NodeValueTier = node.ValueTier
		enemy, err := GenerateEnemy(rng, template, ctx, catalog)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("节点 %s 生成敌人失败: %w", node.ID, err)
		}
		generated = append(generated, enemy)
		nodeEnemyIDs[node.ID] = enemy.ID
	}

	// 遭遇池引用：每个池条目生成独立变体
	ctx.NodeValueTier = 0
	for _, entry := range pools {
		template, ok := templatesByID[entry.EnemyID]
		if !ok || entry.EnemyID == "" {
			continue
		}
		enemy, err := GenerateEnemy(rng, template, ctx, catalog)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("遭遇池 %s 生成敌人失败: %w", entry.ID, err)
		}
		generated = append(generated, enemy)
		poolEnemyIDs[entry.ID] = enemy.ID
	}

	return generated, nodeEnemyIDs, poolEnemyIDs, nil
}

// convertJSON 通过 JSON 往返把含命名习惯的 models 定义复制到同名引擎结构。
func convertJSON(source interface{}, target interface{}) error {
	encoded, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}
