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
		{ID: "computer_case", Name: "电脑机箱", Tags: []string{"electronics", "computer"}, ValueTier: 3, SearchRisk: 1, SearchTime: 1, RollMin: 1, RollMax: 2},
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
		{ID: "material_room", Name: "材料存储间", Tags: []string{"material", "industrial", "storage", "high_value"}, ValueTier: 4, SearchRisk: 3, SearchTime: 3, RollMin: 1, RollMax: 3},
		{ID: "safehouse_cache", Name: "安全屋储备", Tags: []string{"medical", "food", "safehouse"}, ValueTier: 3, SearchRisk: 1, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "weapon_cache", Name: "武器零件柜", Tags: []string{"weaponpart", "valuable", "combat"}, ValueTier: 4, SearchRisk: 3, SearchTime: 3, RollMin: 1, RollMax: 3},
		{ID: "locked_room", Name: "上锁储藏室", Tags: []string{"intel", "valuable", "electronics", "high_value"}, ValueTier: 5, SearchRisk: 2, SearchTime: 3, RollMin: 2, RollMax: 4},
		{ID: "jacket", Name: "遗弃夹克", Tags: []string{"key", "residential", "low_value"}, ValueTier: 2, SearchRisk: 1, SearchTime: 1, RollMin: 0, RollMax: 2},
		{ID: "cash_register", Name: "收银机", Tags: []string{"valuable", "market"}, ValueTier: 3, SearchRisk: 2, SearchTime: 2, RollMin: 0, RollMax: 2},
		{ID: "fuel_stash", Name: "油料堆", Tags: []string{"fuel", "industrial"}, ValueTier: 3, SearchRisk: 2, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "wooden_crate", Name: "木箱", Tags: []string{"material", "industrial", "cargo"}, ValueTier: 2, SearchRisk: 2, SearchTime: 2, RollMin: 1, RollMax: 3},
		{ID: "sport_bag", Name: "运动包", Tags: []string{"mixed", "portable"}, ValueTier: 2, SearchRisk: 1, SearchTime: 1, RollMin: 0, RollMax: 2},
		{ID: "safe", Name: "保险箱", Tags: []string{"valuable", "secured", "high_value"}, ValueTier: 4, SearchRisk: 2, SearchTime: 3, RollMin: 1, RollMax: 2},
		{ID: "medcase", Name: "塑料医疗箱", Tags: []string{"medical", "portable"}, ValueTier: 2, SearchRisk: 1, SearchTime: 1, RollMin: 0, RollMax: 2},
		{ID: "suitcase", Name: "塑料手提箱", Tags: []string{"food", "valuable", "residential"}, ValueTier: 2, SearchRisk: 1, SearchTime: 2, RollMin: 0, RollMax: 2},
		{ID: "ground_cache", Name: "埋藏点", Tags: []string{"mixed", "outdoor", "stash"}, ValueTier: 3, SearchRisk: 2, SearchTime: 3, RollMin: 1, RollMax: 3},
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
		{ID: 5, ContainerID: "apt_drawer", ItemCategory: "valuable", Weight: 0, MinQuantity: 1, MaxQuantity: 1},
		{ID: 82, ContainerID: "apt_drawer", ItemCategory: "key", Weight: 8, MinQuantity: 1, MaxQuantity: 1},
		{ID: 64, ContainerID: "food_shelf", ItemCategory: "food", Weight: 55, MinQuantity: 1, MaxQuantity: 1},
		{ID: 65, ContainerID: "food_shelf", ItemCategory: "medical", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 66, ContainerID: "food_shelf", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 67, ContainerID: "food_shelf", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 6, ContainerID: "tool_crate", ItemCategory: "tool", Weight: 40, MinQuantity: 1, MaxQuantity: 1},
		{ID: 7, ContainerID: "tool_crate", ItemCategory: "electronics", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 8, ContainerID: "tool_crate", ItemCategory: "fuel", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 42, ContainerID: "tool_crate", ItemCategory: "material", Weight: 45, MinQuantity: 1, MaxQuantity: 1},
		{ID: 9, ContainerID: "filing_cabinet", ItemCategory: "info", Weight: 65, MinQuantity: 1, MaxQuantity: 1},
		{ID: 10, ContainerID: "filing_cabinet", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 11, ContainerID: "filing_cabinet", ItemCategory: "valuable", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 83, ContainerID: "filing_cabinet", ItemCategory: "key", Weight: 12, MinQuantity: 1, MaxQuantity: 1},
		{ID: 12, ContainerID: "utility_box", ItemCategory: "electronics", Weight: 50, MinQuantity: 1, MaxQuantity: 1},
		{ID: 13, ContainerID: "utility_box", ItemCategory: "tool", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 14, ContainerID: "utility_box", ItemCategory: "fuel", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 68, ContainerID: "computer_case", ItemCategory: "electronics", Weight: 75, MinQuantity: 1, MaxQuantity: 1},
		{ID: 69, ContainerID: "computer_case", ItemCategory: "tool", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 70, ContainerID: "computer_case", ItemCategory: "info", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 84, ContainerID: "computer_case", ItemCategory: "key", Weight: 8, MinQuantity: 1, MaxQuantity: 1},
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
		{ID: 31, ContainerID: "enemy_backpack_basic", ItemCategory: "valuable", Weight: 0, MinQuantity: 1, MaxQuantity: 1},
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
		{ID: 85, ContainerID: "locked_room", ItemCategory: "info", Weight: 40, MinQuantity: 1, MaxQuantity: 2},
		{ID: 86, ContainerID: "locked_room", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 87, ContainerID: "locked_room", ItemCategory: "valuable", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 88, ContainerID: "locked_room", ItemCategory: "medical", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 89, ContainerID: "enemy_backpack_elite", ItemCategory: "key", Weight: 8, MinQuantity: 1, MaxQuantity: 1},
		{ID: 90, ContainerID: "jacket", ItemCategory: "key", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 91, ContainerID: "jacket", ItemCategory: "valuable", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 92, ContainerID: "jacket", ItemCategory: "info", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 93, ContainerID: "jacket", ItemCategory: "food", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 94, ContainerID: "jacket", ItemCategory: "medical", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 95, ContainerID: "jacket", ItemCategory: "tool", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 96, ContainerID: "cash_register", ItemCategory: "valuable", Weight: 45, MinQuantity: 1, MaxQuantity: 1},
		{ID: 97, ContainerID: "cash_register", ItemCategory: "food", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 98, ContainerID: "cash_register", ItemCategory: "electronics", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 99, ContainerID: "cash_register", ItemCategory: "info", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 100, ContainerID: "cash_register", ItemCategory: "key", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 101, ContainerID: "fuel_stash", ItemCategory: "fuel", Weight: 40, MinQuantity: 1, MaxQuantity: 1},
		{ID: 102, ContainerID: "fuel_stash", ItemCategory: "electronics", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 103, ContainerID: "fuel_stash", ItemCategory: "tool", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 104, ContainerID: "fuel_stash", ItemCategory: "material", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 105, ContainerID: "wooden_crate", ItemCategory: "material", Weight: 35, MinQuantity: 1, MaxQuantity: 2},
		{ID: 106, ContainerID: "wooden_crate", ItemCategory: "tool", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 107, ContainerID: "wooden_crate", ItemCategory: "food", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 108, ContainerID: "wooden_crate", ItemCategory: "electronics", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 109, ContainerID: "sport_bag", ItemCategory: "food", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 110, ContainerID: "sport_bag", ItemCategory: "medical", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 111, ContainerID: "sport_bag", ItemCategory: "tool", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 112, ContainerID: "sport_bag", ItemCategory: "electronics", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 113, ContainerID: "sport_bag", ItemCategory: "valuable", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 114, ContainerID: "sport_bag", ItemCategory: "info", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 115, ContainerID: "safe", ItemCategory: "valuable", Weight: 80, MinQuantity: 1, MaxQuantity: 1},
		{ID: 116, ContainerID: "safe", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 117, ContainerID: "safe", ItemCategory: "key", Weight: 5, MinQuantity: 1, MaxQuantity: 1},
		{ID: 118, ContainerID: "medcase", ItemCategory: "medical", Weight: 90, MinQuantity: 1, MaxQuantity: 1},
		{ID: 119, ContainerID: "medcase", ItemCategory: "food", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 120, ContainerID: "suitcase", ItemCategory: "food", Weight: 30, MinQuantity: 1, MaxQuantity: 1},
		{ID: 121, ContainerID: "suitcase", ItemCategory: "valuable", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 122, ContainerID: "suitcase", ItemCategory: "tool", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 123, ContainerID: "suitcase", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 124, ContainerID: "suitcase", ItemCategory: "medical", Weight: 10, MinQuantity: 1, MaxQuantity: 1},
		{ID: 125, ContainerID: "ground_cache", ItemCategory: "material", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 126, ContainerID: "ground_cache", ItemCategory: "valuable", Weight: 25, MinQuantity: 1, MaxQuantity: 1},
		{ID: 127, ContainerID: "ground_cache", ItemCategory: "tool", Weight: 20, MinQuantity: 1, MaxQuantity: 1},
		{ID: 128, ContainerID: "ground_cache", ItemCategory: "electronics", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
		{ID: 129, ContainerID: "ground_cache", ItemCategory: "info", Weight: 15, MinQuantity: 1, MaxQuantity: 1},
	}
	for _, rule := range rules {
		if err := upsertSeedDef(db, &rule, fmt.Sprint(rule.ID)); err != nil {
			return err
		}
	}

	assignments := nodeContainerAssignments()
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
