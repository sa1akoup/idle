package config

import (
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

// seedContainers 初始化节点容器、容器分类权重与敌人背包使用的容器模板。
func seedContainers(db *gorm.DB) error {
	containers := []models.LootContainerDef{
		{ID: "apt_drawer", Name: "居民抽屉", Tags: []string{"food", "residential", "low_value"}, ValueTier: 1, SearchRisk: 1, SearchTime: 1, RollMin: 0, RollMax: 2},
		{ID: "food_shelf", Name: "居民食品柜", Tags: []string{"food", "residential"}, ValueTier: 2, SearchRisk: 1, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "tool_crate", Name: "工具箱", Tags: []string{"tool", "industrial"}, ValueTier: 2, SearchRisk: 2, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "filing_cabinet", Name: "文件柜", Tags: []string{"info", "document", "office"}, ValueTier: 3, SearchRisk: 1, SearchTime: 2, RollMin: 0, RollMax: 2},
		{ID: "computer_case", Name: "电脑机箱", Tags: []string{"electronics", "computer", "low_value"}, ValueTier: 2, SearchRisk: 1, SearchTime: 1, RollMin: 1, RollMax: 2},
		{ID: "server_room", Name: "服务器机房", Tags: []string{"electronics", "server", "intel", "high_value"}, ValueTier: 5, SearchRisk: 4, SearchTime: 4, RollMin: 2, RollMax: 4},
		{ID: "utility_box", Name: "设备箱", Tags: []string{"electronics", "equipment"}, ValueTier: 2, SearchRisk: 2, SearchTime: 2, RollMin: 0, RollMax: 2},
		{ID: "cargo_crate", Name: "货运箱", Tags: []string{"material", "cargo", "industrial"}, ValueTier: 2, SearchRisk: 3, SearchTime: 3, RollMin: 0, RollMax: 3},
		{ID: "medical_bag", Name: "路边医疗袋", Tags: []string{"medical", "portable", "low_value"}, ValueTier: 1, SearchRisk: 1, SearchTime: 1, RollMin: 0, RollMax: 2},
		{ID: "medical_cache", Name: "大型医疗箱", Tags: []string{"medical", "supply"}, ValueTier: 3, SearchRisk: 1, SearchTime: 2, RollMin: 0, RollMax: 2},
		{ID: "medical_room", Name: "大型医疗室", Tags: []string{"medical", "facility", "high_value"}, ValueTier: 5, SearchRisk: 2, SearchTime: 4, RollMin: 2, RollMax: 4},
		{ID: "supply_crate", Name: "补给箱", Tags: []string{"medical", "food", "supply"}, ValueTier: 2, SearchRisk: 2, SearchTime: 2, RollMin: 0, RollMax: 3},
		{ID: "enemy_backpack_basic", Name: "巡逻背包", Tags: []string{"enemy_loot", "low_value"}, ValueTier: 1, SearchRisk: 1, SearchTime: 1, RollMin: 0, RollMax: 2},
		{ID: "enemy_backpack_guard", Name: "守卫背包", Tags: []string{"enemy_loot", "combat"}, ValueTier: 3, SearchRisk: 2, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "enemy_backpack_elite", Name: "精锐背包", Tags: []string{"enemy_loot", "high_value"}, ValueTier: 5, SearchRisk: 3, SearchTime: 2, RollMin: 1, RollMax: 4},
		{ID: "intel_room", Name: "高价值情报室", Tags: []string{"intel", "electronics", "facility", "high_value"}, ValueTier: 5, SearchRisk: 3, SearchTime: 4, RollMin: 2, RollMax: 4},
		{ID: "material_room", Name: "材料存储间", Tags: []string{"material", "industrial", "storage", "high_value"}, ValueTier: 4, SearchRisk: 3, SearchTime: 3, RollMin: 2, RollMax: 5},
		{ID: "safehouse_cache", Name: "安全屋储备", Tags: []string{"medical", "food", "safehouse"}, ValueTier: 3, SearchRisk: 1, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "weapon_cache", Name: "武器零件柜", Tags: []string{"weaponpart", "valuable", "combat"}, ValueTier: 4, SearchRisk: 3, SearchTime: 3, RollMin: 1, RollMax: 3},
	}
	for _, container := range containers {
		if err := upsertSeedDef(db, &container, container.ID); err != nil {
			return err
		}
	}

	rules := []models.LootContainerRule{
		{ID: 1, ContainerID: "apt_drawer", ItemCategory: "food", Weight: 40, MinQuantity: 1, MaxQuantity: 1},
		{ID: 2, ContainerID: "apt_drawer", ItemCategory: "medical", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 3, ContainerID: "apt_drawer", ItemCategory: "info", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 4, ContainerID: "apt_drawer", ItemCategory: "tool", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 5, ContainerID: "apt_drawer", ItemCategory: "valuable", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 64, ContainerID: "food_shelf", ItemCategory: "food", Weight: 55, MinQuantity: 1, MaxQuantity: 1},
		{ID: 65, ContainerID: "food_shelf", ItemCategory: "medical", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 66, ContainerID: "food_shelf", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 67, ContainerID: "food_shelf", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 6, ContainerID: "tool_crate", ItemCategory: "tool", Weight: 55, MinQuantity: 1, MaxQuantity: 1},
		{ID: 7, ContainerID: "tool_crate", ItemCategory: "electronics", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 8, ContainerID: "tool_crate", ItemCategory: "fuel", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 42, ContainerID: "tool_crate", ItemCategory: "material", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 9, ContainerID: "filing_cabinet", ItemCategory: "info", Weight: 65, MinQuantity: 1, MaxQuantity: 1},
		{ID: 10, ContainerID: "filing_cabinet", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 11, ContainerID: "filing_cabinet", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 12, ContainerID: "utility_box", ItemCategory: "electronics", Weight: 50, MinQuantity: 1, MaxQuantity: 1},
		{ID: 13, ContainerID: "utility_box", ItemCategory: "tool", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 14, ContainerID: "utility_box", ItemCategory: "fuel", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 68, ContainerID: "computer_case", ItemCategory: "electronics", Weight: 55, MinQuantity: 1, MaxQuantity: 1},
		{ID: 69, ContainerID: "computer_case", ItemCategory: "tool", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 70, ContainerID: "computer_case", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 71, ContainerID: "server_room", ItemCategory: "info", Weight: 50, MinQuantity: 1, MaxQuantity: 2},
		{ID: 72, ContainerID: "server_room", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 73, ContainerID: "server_room", ItemCategory: "valuable", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 74, ContainerID: "server_room", ItemCategory: "tool", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 15, ContainerID: "cargo_crate", ItemCategory: "tool", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 16, ContainerID: "cargo_crate", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 17, ContainerID: "cargo_crate", ItemCategory: "fuel", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 18, ContainerID: "cargo_crate", ItemCategory: "valuable", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 43, ContainerID: "utility_box", ItemCategory: "material", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 44, ContainerID: "cargo_crate", ItemCategory: "material", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 45, ContainerID: "cargo_crate", ItemCategory: "weaponpart", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 19, ContainerID: "medical_cache", ItemCategory: "medical", Weight: 70, MinQuantity: 1, MaxQuantity: 1},
		{ID: 20, ContainerID: "medical_cache", ItemCategory: "food", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 21, ContainerID: "medical_cache", ItemCategory: "tool", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 75, ContainerID: "medical_bag", ItemCategory: "medical", Weight: 70, MinQuantity: 1, MaxQuantity: 1},
		{ID: 76, ContainerID: "medical_bag", ItemCategory: "food", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 77, ContainerID: "medical_bag", ItemCategory: "tool", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 78, ContainerID: "medical_room", ItemCategory: "medical", Weight: 65, MinQuantity: 1, MaxQuantity: 2},
		{ID: 79, ContainerID: "medical_room", ItemCategory: "tool", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 80, ContainerID: "medical_room", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 81, ContainerID: "medical_room", ItemCategory: "info", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 22, ContainerID: "supply_crate", ItemCategory: "medical", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 23, ContainerID: "supply_crate", ItemCategory: "food", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 24, ContainerID: "supply_crate", ItemCategory: "tool", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 25, ContainerID: "supply_crate", ItemCategory: "electronics", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 26, ContainerID: "supply_crate", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 27, ContainerID: "enemy_backpack_basic", ItemCategory: "food", Weight: 35, MinQuantity: 1, MaxQuantity: 1},
		{ID: 28, ContainerID: "enemy_backpack_basic", ItemCategory: "medical", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 29, ContainerID: "enemy_backpack_basic", ItemCategory: "tool", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 30, ContainerID: "enemy_backpack_basic", ItemCategory: "electronics", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 31, ContainerID: "enemy_backpack_basic", ItemCategory: "valuable", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 32, ContainerID: "enemy_backpack_guard", ItemCategory: "tool", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 33, ContainerID: "enemy_backpack_guard", ItemCategory: "electronics", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 34, ContainerID: "enemy_backpack_guard", ItemCategory: "medical", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 35, ContainerID: "enemy_backpack_guard", ItemCategory: "food", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 36, ContainerID: "enemy_backpack_guard", ItemCategory: "valuable", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 46, ContainerID: "enemy_backpack_guard", ItemCategory: "weaponpart", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 37, ContainerID: "enemy_backpack_elite", ItemCategory: "info", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 38, ContainerID: "enemy_backpack_elite", ItemCategory: "electronics", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 39, ContainerID: "enemy_backpack_elite", ItemCategory: "valuable", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 40, ContainerID: "enemy_backpack_elite", ItemCategory: "medical", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 41, ContainerID: "enemy_backpack_elite", ItemCategory: "tool", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 47, ContainerID: "enemy_backpack_elite", ItemCategory: "weaponpart", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 48, ContainerID: "intel_room", ItemCategory: "info", Weight: 55, MinQuantity: 1, MaxQuantity: 1},
		{ID: 49, ContainerID: "intel_room", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 50, ContainerID: "intel_room", ItemCategory: "valuable", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 51, ContainerID: "material_room", ItemCategory: "material", Weight: 35, MinQuantity: 1, MaxQuantity: 2},
		{ID: 52, ContainerID: "material_room", ItemCategory: "tool", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 53, ContainerID: "material_room", ItemCategory: "electronics", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 54, ContainerID: "material_room", ItemCategory: "fuel", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 55, ContainerID: "material_room", ItemCategory: "weaponpart", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 56, ContainerID: "safehouse_cache", ItemCategory: "medical", Weight: 40, MinQuantity: 1, MaxQuantity: 1},
		{ID: 57, ContainerID: "safehouse_cache", ItemCategory: "food", Weight: 35, MinQuantity: 1, MaxQuantity: 1},
		{ID: 58, ContainerID: "safehouse_cache", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 59, ContainerID: "safehouse_cache", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 60, ContainerID: "weapon_cache", ItemCategory: "weaponpart", Weight: 45, MinQuantity: 1, MaxQuantity: 1},
		{ID: 61, ContainerID: "weapon_cache", ItemCategory: "tool", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 62, ContainerID: "weapon_cache", ItemCategory: "electronics", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 63, ContainerID: "weapon_cache", ItemCategory: "valuable", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
	}
	for _, rule := range rules {
		if err := upsertSeedDef(db, &rule, fmt.Sprint(rule.ID)); err != nil {
			return err
		}
	}

	// 每个节点的容器槽位固定，具体容器类型按节点池权重逐槽抽取。
	assignments := []models.NodeContainerDef{
		{ID: 1, NodeID: "node_apt", ContainerID: "apt_drawer", Weight: 60},
		{ID: 2, NodeID: "node_apt", ContainerID: "food_shelf", Weight: 25},
		{ID: 3, NodeID: "node_apt", ContainerID: "medical_bag", Weight: 10},
		{ID: 4, NodeID: "node_apt", ContainerID: "computer_case", Weight: 5},
		{ID: 5, NodeID: "node_warehouse", ContainerID: "tool_crate", Weight: 45},
		{ID: 6, NodeID: "node_warehouse", ContainerID: "cargo_crate", Weight: 25},
		{ID: 7, NodeID: "node_warehouse", ContainerID: "utility_box", Weight: 20},
		{ID: 8, NodeID: "node_warehouse", ContainerID: "material_room", Weight: 10},
		{ID: 9, NodeID: "node_customs", ContainerID: "filing_cabinet", Weight: 30},
		{ID: 10, NodeID: "node_customs", ContainerID: "computer_case", Weight: 15},
		{ID: 11, NodeID: "node_customs", ContainerID: "server_room", Weight: 25},
		{ID: 12, NodeID: "node_customs", ContainerID: "intel_room", Weight: 20},
		{ID: 13, NodeID: "node_customs", ContainerID: "medical_cache", Weight: 10},
		{ID: 14, NodeID: "node_tunnel", ContainerID: "medical_bag", Weight: 30},
		{ID: 15, NodeID: "node_tunnel", ContainerID: "utility_box", Weight: 30},
		{ID: 16, NodeID: "node_tunnel", ContainerID: "supply_crate", Weight: 25},
		{ID: 17, NodeID: "node_tunnel", ContainerID: "tool_crate", Weight: 15},
		{ID: 18, NodeID: "node_container", ContainerID: "cargo_crate", Weight: 45},
		{ID: 19, NodeID: "node_container", ContainerID: "material_room", Weight: 25},
		{ID: 20, NodeID: "node_container", ContainerID: "tool_crate", Weight: 15},
		{ID: 21, NodeID: "node_container", ContainerID: "computer_case", Weight: 10},
		{ID: 22, NodeID: "node_container", ContainerID: "medical_cache", Weight: 5},
		{ID: 23, NodeID: "node_pier", ContainerID: "medical_bag", Weight: 45},
		{ID: 24, NodeID: "node_pier", ContainerID: "supply_crate", Weight: 35},
		{ID: 25, NodeID: "node_pier", ContainerID: "medical_cache", Weight: 20},

		// 事件奖励池不占用普通搜索槽位；事件成功后才按当前节点的池权重抽取。
		{ID: 26, NodeID: "node_apt", Pool: "safehouse_reward", ContainerID: "safehouse_cache", Weight: 70},
		{ID: 27, NodeID: "node_apt", Pool: "safehouse_reward", ContainerID: "medical_cache", Weight: 30},
		{ID: 28, NodeID: "node_apt", Pool: "supply_reward", ContainerID: "supply_crate", Weight: 70},
		{ID: 29, NodeID: "node_apt", Pool: "supply_reward", ContainerID: "safehouse_cache", Weight: 30},
		{ID: 30, NodeID: "node_warehouse", Pool: "material_reward", ContainerID: "material_room", Weight: 75},
		{ID: 31, NodeID: "node_warehouse", Pool: "material_reward", ContainerID: "cargo_crate", Weight: 25},
		{ID: 32, NodeID: "node_warehouse", Pool: "workshop_reward", ContainerID: "weapon_cache", Weight: 70},
		{ID: 33, NodeID: "node_warehouse", Pool: "workshop_reward", ContainerID: "tool_crate", Weight: 30},
		{ID: 34, NodeID: "node_customs", Pool: "intel_reward", ContainerID: "intel_room", Weight: 70},
		{ID: 35, NodeID: "node_customs", Pool: "intel_reward", ContainerID: "server_room", Weight: 30},
		{ID: 36, NodeID: "node_customs", Pool: "intel_fallback", ContainerID: "filing_cabinet", Weight: 100},
		{ID: 37, NodeID: "node_customs", Pool: "sealed_reward", ContainerID: "weapon_cache", Weight: 55},
		{ID: 38, NodeID: "node_customs", Pool: "sealed_reward", ContainerID: "material_room", Weight: 45},
		{ID: 39, NodeID: "node_tunnel", Pool: "medical_reward", ContainerID: "medical_room", Weight: 100},
		{ID: 40, NodeID: "node_tunnel", Pool: "medical_fallback", ContainerID: "medical_cache", Weight: 100},
		{ID: 41, NodeID: "node_tunnel", Pool: "supply_reward", ContainerID: "supply_crate", Weight: 70},
		{ID: 42, NodeID: "node_tunnel", Pool: "supply_reward", ContainerID: "medical_cache", Weight: 30},
		{ID: 43, NodeID: "node_container", Pool: "material_reward", ContainerID: "material_room", Weight: 60},
		{ID: 44, NodeID: "node_container", Pool: "material_reward", ContainerID: "cargo_crate", Weight: 40},
		{ID: 45, NodeID: "node_container", Pool: "workshop_reward", ContainerID: "weapon_cache", Weight: 60},
		{ID: 46, NodeID: "node_container", Pool: "workshop_reward", ContainerID: "tool_crate", Weight: 40},
		{ID: 47, NodeID: "node_customs", Pool: "server_reward", ContainerID: "server_room", Weight: 100},
		{ID: 48, NodeID: "node_warehouse", Pool: "workshop_fallback", ContainerID: "tool_crate", Weight: 100},
		{ID: 49, NodeID: "node_container", Pool: "workshop_fallback", ContainerID: "tool_crate", Weight: 100},
	}
	// 旧内容器配置复用到九宫格语义节点，新增三个节点的基础搜索池。
	nodeAlias := map[string]string{
		"node_apt": "city_ruins_node_5", "node_warehouse": "city_ruins_node_2", "node_customs": "city_ruins_node_8",
		"node_tunnel": "city_ruins_node_4", "node_container": "city_ruins_node_6", "node_pier": "city_ruins_node_1",
	}
	for index := range assignments {
		if replacement, ok := nodeAlias[assignments[index].NodeID]; ok {
			assignments[index].NodeID = replacement
		}
	}
	assignments = append(assignments,
		models.NodeContainerDef{NodeID: "city_ruins_node_3", ContainerID: "food_shelf", Weight: 45},
		models.NodeContainerDef{NodeID: "city_ruins_node_3", ContainerID: "cargo_crate", Weight: 30},
		models.NodeContainerDef{NodeID: "city_ruins_node_3", ContainerID: "computer_case", Weight: 25},
		models.NodeContainerDef{NodeID: "city_ruins_node_7", ContainerID: "medical_cache", Weight: 55},
		models.NodeContainerDef{NodeID: "city_ruins_node_7", ContainerID: "medical_bag", Weight: 30},
		models.NodeContainerDef{NodeID: "city_ruins_node_7", ContainerID: "safehouse_cache", Weight: 15},
		models.NodeContainerDef{NodeID: "city_ruins_node_9", ContainerID: "utility_box", Weight: 45},
		models.NodeContainerDef{NodeID: "city_ruins_node_9", ContainerID: "utility_box", Weight: 35},
		models.NodeContainerDef{NodeID: "city_ruins_node_9", ContainerID: "cargo_crate", Weight: 20},
	)
	nodeIDs := make([]string, 0, len(assignments))
	seenNodes := make(map[string]bool)
	for _, assignment := range assignments {
		if !seenNodes[assignment.NodeID] {
			nodeIDs = append(nodeIDs, assignment.NodeID)
			seenNodes[assignment.NodeID] = true
		}
	}
	if err := db.Where("node_id IN ?", nodeIDs).Delete(&models.NodeContainerDef{}).Error; err != nil {
		return err
	}
	for _, assignment := range assignments {
		if assignment.Pool == "" {
			assignment.Pool = models.NodeContainerPoolSearch
		}
		assignment.ID = 0
		if err := db.Create(&assignment).Error; err != nil {
			return err
		}
	}
	return nil
}
