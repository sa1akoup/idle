package config

import "idle/internal/models"

// nodeContainerAssignments 九宫格搜索池与事件奖励池。搜索池按节点槽位逐槽抽取。
func nodeContainerAssignments() []models.NodeContainerDef {
	return []models.NodeContainerDef{
		// 码头北区：岸边货箱与油料，不再是纯医疗点。
		{NodeID: "city_ruins_node_1", ContainerID: "cargo_crate", Weight: 25},
		{NodeID: "city_ruins_node_1", ContainerID: "supply_crate", Weight: 20},
		{NodeID: "city_ruins_node_1", ContainerID: "ground_cache", Weight: 20},
		{NodeID: "city_ruins_node_1", ContainerID: "fuel_stash", Weight: 15},
		{NodeID: "city_ruins_node_1", ContainerID: "jacket", Weight: 12},
		{NodeID: "city_ruins_node_1", ContainerID: "medical_bag", Weight: 8},
		{NodeID: "city_ruins_node_1", Pool: "supply_reward", ContainerID: "supply_crate", Weight: 70},
		{NodeID: "city_ruins_node_1", Pool: "supply_reward", ContainerID: "cargo_crate", Weight: 30},
		{NodeID: "city_ruins_node_1", Pool: "fuel_reward", ContainerID: "fuel_stash", Weight: 100},

		// 废弃仓库：木箱和材料间。
		{NodeID: "city_ruins_node_2", ContainerID: "wooden_crate", Weight: 28},
		{NodeID: "city_ruins_node_2", ContainerID: "tool_crate", Weight: 24},
		{NodeID: "city_ruins_node_2", ContainerID: "cargo_crate", Weight: 18},
		{NodeID: "city_ruins_node_2", ContainerID: "material_room", Weight: 14},
		{NodeID: "city_ruins_node_2", ContainerID: "utility_box", Weight: 8},
		{NodeID: "city_ruins_node_2", ContainerID: "safe", Weight: 8},
		{NodeID: "city_ruins_node_2", Pool: "material_reward", ContainerID: "material_room", Weight: 75},
		{NodeID: "city_ruins_node_2", Pool: "material_reward", ContainerID: "wooden_crate", Weight: 25},
		{NodeID: "city_ruins_node_2", Pool: "workshop_reward", ContainerID: "weapon_cache", Weight: 70},
		{NodeID: "city_ruins_node_2", Pool: "workshop_reward", ContainerID: "tool_crate", Weight: 30},
		{NodeID: "city_ruins_node_2", Pool: "workshop_fallback", ContainerID: "tool_crate", Weight: 100},
		{NodeID: "city_ruins_node_2", Pool: "locked_reward", ContainerID: "locked_room", Weight: 70},
		{NodeID: "city_ruins_node_2", Pool: "locked_reward", ContainerID: "weapon_cache", Weight: 30},

		// 旧市场：收银机、夹克、食品柜。
		{NodeID: "city_ruins_node_3", ContainerID: "food_shelf", Weight: 25},
		{NodeID: "city_ruins_node_3", ContainerID: "cash_register", Weight: 22},
		{NodeID: "city_ruins_node_3", ContainerID: "jacket", Weight: 18},
		{NodeID: "city_ruins_node_3", ContainerID: "suitcase", Weight: 12},
		{NodeID: "city_ruins_node_3", ContainerID: "ground_cache", Weight: 13},
		{NodeID: "city_ruins_node_3", ContainerID: "sport_bag", Weight: 10},
		{NodeID: "city_ruins_node_3", Pool: "supply_reward", ContainerID: "supply_crate", Weight: 70},
		{NodeID: "city_ruins_node_3", Pool: "supply_reward", ContainerID: "safehouse_cache", Weight: 30},
		{NodeID: "city_ruins_node_3", Pool: "market_reward", ContainerID: "cash_register", Weight: 70},
		{NodeID: "city_ruins_node_3", Pool: "market_reward", ContainerID: "jacket", Weight: 30},

		// 地下通道：维修箱和随身包。
		{NodeID: "city_ruins_node_4", ContainerID: "utility_box", Weight: 26},
		{NodeID: "city_ruins_node_4", ContainerID: "medical_bag", Weight: 18},
		{NodeID: "city_ruins_node_4", ContainerID: "medcase", Weight: 16},
		{NodeID: "city_ruins_node_4", ContainerID: "wooden_crate", Weight: 18},
		{NodeID: "city_ruins_node_4", ContainerID: "sport_bag", Weight: 12},
		{NodeID: "city_ruins_node_4", ContainerID: "tool_crate", Weight: 10},
		{NodeID: "city_ruins_node_4", Pool: "medical_reward", ContainerID: "medical_room", Weight: 100},
		{NodeID: "city_ruins_node_4", Pool: "medical_fallback", ContainerID: "medical_cache", Weight: 100},
		{NodeID: "city_ruins_node_4", Pool: "supply_reward", ContainerID: "supply_crate", Weight: 70},
		{NodeID: "city_ruins_node_4", Pool: "supply_reward", ContainerID: "medical_cache", Weight: 30},
		{NodeID: "city_ruins_node_4", Pool: "locker_reward", ContainerID: "utility_box", Weight: 60},
		{NodeID: "city_ruins_node_4", Pool: "locker_reward", ContainerID: "tool_crate", Weight: 40},

		// 居民楼：夹克和抽屉出钥匙。
		{NodeID: "city_ruins_node_5", ContainerID: "jacket", Weight: 30},
		{NodeID: "city_ruins_node_5", ContainerID: "apt_drawer", Weight: 25},
		{NodeID: "city_ruins_node_5", ContainerID: "food_shelf", Weight: 18},
		{NodeID: "city_ruins_node_5", ContainerID: "suitcase", Weight: 15},
		{NodeID: "city_ruins_node_5", ContainerID: "sport_bag", Weight: 7},
		{NodeID: "city_ruins_node_5", ContainerID: "computer_case", Weight: 5},
		{NodeID: "city_ruins_node_5", Pool: "safehouse_reward", ContainerID: "safehouse_cache", Weight: 70},
		{NodeID: "city_ruins_node_5", Pool: "safehouse_reward", ContainerID: "medical_cache", Weight: 30},
		{NodeID: "city_ruins_node_5", Pool: "supply_reward", ContainerID: "supply_crate", Weight: 70},
		{NodeID: "city_ruins_node_5", Pool: "supply_reward", ContainerID: "safehouse_cache", Weight: 30},

		// 集装箱场：货运和木箱。
		{NodeID: "city_ruins_node_6", ContainerID: "cargo_crate", Weight: 28},
		{NodeID: "city_ruins_node_6", ContainerID: "wooden_crate", Weight: 22},
		{NodeID: "city_ruins_node_6", ContainerID: "ground_cache", Weight: 15},
		{NodeID: "city_ruins_node_6", ContainerID: "material_room", Weight: 15},
		{NodeID: "city_ruins_node_6", ContainerID: "tool_crate", Weight: 12},
		{NodeID: "city_ruins_node_6", ContainerID: "computer_case", Weight: 8},
		{NodeID: "city_ruins_node_6", Pool: "material_reward", ContainerID: "material_room", Weight: 60},
		{NodeID: "city_ruins_node_6", Pool: "material_reward", ContainerID: "cargo_crate", Weight: 40},
		{NodeID: "city_ruins_node_6", Pool: "workshop_reward", ContainerID: "weapon_cache", Weight: 60},
		{NodeID: "city_ruins_node_6", Pool: "workshop_reward", ContainerID: "tool_crate", Weight: 40},
		{NodeID: "city_ruins_node_6", Pool: "workshop_fallback", ContainerID: "tool_crate", Weight: 100},

		// 社区诊所：提高医疗室权重。
		{NodeID: "city_ruins_node_7", ContainerID: "medical_cache", Weight: 28},
		{NodeID: "city_ruins_node_7", ContainerID: "medcase", Weight: 22},
		{NodeID: "city_ruins_node_7", ContainerID: "medical_bag", Weight: 18},
		{NodeID: "city_ruins_node_7", ContainerID: "medical_room", Weight: 18},
		{NodeID: "city_ruins_node_7", ContainerID: "safehouse_cache", Weight: 10},
		{NodeID: "city_ruins_node_7", ContainerID: "jacket", Weight: 4},
		{NodeID: "city_ruins_node_7", Pool: "medical_reward", ContainerID: "medical_room", Weight: 100},
		{NodeID: "city_ruins_node_7", Pool: "medical_fallback", ContainerID: "medical_cache", Weight: 100},
		{NodeID: "city_ruins_node_7", Pool: "locked_reward", ContainerID: "locked_room", Weight: 100},

		// 海关办公楼：文件与机房，夹克摸钥匙。
		{NodeID: "city_ruins_node_8", ContainerID: "filing_cabinet", Weight: 26},
		{NodeID: "city_ruins_node_8", ContainerID: "computer_case", Weight: 18},
		{NodeID: "city_ruins_node_8", ContainerID: "server_room", Weight: 18},
		{NodeID: "city_ruins_node_8", ContainerID: "intel_room", Weight: 16},
		{NodeID: "city_ruins_node_8", ContainerID: "safe", Weight: 12},
		{NodeID: "city_ruins_node_8", ContainerID: "jacket", Weight: 10},
		{NodeID: "city_ruins_node_8", Pool: "intel_reward", ContainerID: "intel_room", Weight: 70},
		{NodeID: "city_ruins_node_8", Pool: "intel_reward", ContainerID: "server_room", Weight: 30},
		{NodeID: "city_ruins_node_8", Pool: "intel_fallback", ContainerID: "filing_cabinet", Weight: 100},
		{NodeID: "city_ruins_node_8", Pool: "sealed_reward", ContainerID: "weapon_cache", Weight: 55},
		{NodeID: "city_ruins_node_8", Pool: "sealed_reward", ContainerID: "material_room", Weight: 45},
		{NodeID: "city_ruins_node_8", Pool: "server_reward", ContainerID: "server_room", Weight: 100},
		{NodeID: "city_ruins_node_8", Pool: "locked_reward", ContainerID: "locked_room", Weight: 50},
		{NodeID: "city_ruins_node_8", Pool: "locked_reward", ContainerID: "safe", Weight: 30},
		{NodeID: "city_ruins_node_8", Pool: "locked_reward", ContainerID: "intel_room", Weight: 20},

		// 加油站：油料堆和设备箱。
		{NodeID: "city_ruins_node_9", ContainerID: "fuel_stash", Weight: 35},
		{NodeID: "city_ruins_node_9", ContainerID: "utility_box", Weight: 22},
		{NodeID: "city_ruins_node_9", ContainerID: "tool_crate", Weight: 18},
		{NodeID: "city_ruins_node_9", ContainerID: "ground_cache", Weight: 15},
		{NodeID: "city_ruins_node_9", ContainerID: "cargo_crate", Weight: 10},
		{NodeID: "city_ruins_node_9", Pool: "material_reward", ContainerID: "material_room", Weight: 40},
		{NodeID: "city_ruins_node_9", Pool: "material_reward", ContainerID: "utility_box", Weight: 30},
		{NodeID: "city_ruins_node_9", Pool: "fuel_reward", ContainerID: "fuel_stash", Weight: 100},
	}
}
