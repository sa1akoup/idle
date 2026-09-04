// 藏身处种子：初始化设施定义、原版风格的等级收益、升级条件和玩家基础模块状态。
package config

import (
	"errors"

	"idle/internal/models"

	"gorm.io/gorm"
)

func seedHideout(db *gorm.DB) error {
	if err := seedHideoutDefinitions(db); err != nil {
		return err
	}

	var users []models.User
	if err := db.Select("id").Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if err := seedHideoutForUser(db, user.ID); err != nil {
			return err
		}
	}
	return nil
}

func seedHideoutDefinitions(db *gorm.DB) error {
	facilities := []models.FacilityDef{
		{ID: "storage", Name: "储物间", Category: "storage", Description: "扩大仓库容量。", IconKey: "storage", MaxLevel: 3, SortOrder: 1},
		{ID: "security", Name: "安保", Category: "security", Description: "提高藏身处安全等级，作为高级设施的前置模块。", IconKey: "security", MaxLevel: 3, SortOrder: 2},
		{ID: "ventilation", Name: "通风管道", Category: "ventilation", Description: "改善藏身处环境，解锁高级生产和成长模块。", IconKey: "ventilation", MaxLevel: 3, SortOrder: 3},
		{ID: "generator", Name: "发电机", Category: "power", Description: "为需要供电的设施提供持续电力。", IconKey: "generator", MaxLevel: 3, SortOrder: 4},
		{ID: "heating", Name: "供暖", Category: "survival", Description: "持续恢复 Energy。", IconKey: "heating", MaxLevel: 3, SortOrder: 5},
		{ID: "lighting", Name: "照明", Category: "power", Description: "改善工作环境并作为发电设施的前置模块。", IconKey: "lighting", MaxLevel: 3, SortOrder: 6},
		{ID: "rest_area", Name: "休息处", Category: "survival", Description: "持续恢复 Stress。", IconKey: "rest_area", MaxLevel: 3, SortOrder: 7},
		{ID: "medstation", Name: "医疗站", Category: "medical", Description: "持续恢复 HP。", IconKey: "medical", MaxLevel: 3, SortOrder: 8},
		{ID: "nutrition_unit", Name: "饮食单元", Category: "production", Description: "为食品和饮水生产模块提供基础设施。", IconKey: "nutrition_unit", MaxLevel: 3, SortOrder: 9},
		{ID: "lavatory", Name: "洗手间", Category: "production", Description: "为基础消耗品生产提供条件。", IconKey: "lavatory", MaxLevel: 3, SortOrder: 10},
		{ID: "water_collector", Name: "饮水收集器", Category: "survival", Description: "持续恢复 Hydration。", IconKey: "water_collector", MaxLevel: 3, SortOrder: 11},
		{ID: "workbench", Name: "工作台", Category: "workbench", Description: "维修护甲并降低维修包消耗。", IconKey: "workbench", MaxLevel: 3, SortOrder: 12},
		{ID: "air_filter", Name: "空气过滤装置", Category: "training", Description: "提高身体技能成长速度。", IconKey: "air_filter", MaxLevel: 1, SortOrder: 13},
		{ID: "solar_panel", Name: "太阳能板", Category: "power", Description: "降低发电机燃料消耗。", IconKey: "solar_panel", MaxLevel: 1, SortOrder: 14},
		{ID: "intel", Name: "情报中心", Category: "intel", Description: "提高行动情报质量。", IconKey: "intel", MaxLevel: 2, SortOrder: 15},
		{ID: "booze_generator", Name: "酒精发生器", Category: "production", Description: "生产酒精和高价值补给。", IconKey: "booze_generator", MaxLevel: 1, SortOrder: 16},
		{ID: "bitcoin_farm", Name: "比特币农场", Category: "production", Description: "使用显卡生产实物比特币。", IconKey: "bitcoin_farm", MaxLevel: 3, SortOrder: 17},
		{ID: "scav_case", Name: "Scav 箱", Category: "production", Description: "派遣 Scav 搜集额外物资。", IconKey: "scav_case", MaxLevel: 1, SortOrder: 18},
		{ID: "shooting_range", Name: "射击场", Category: "training", Description: "训练武器熟练度并测试装备。", IconKey: "shooting_range", MaxLevel: 3, SortOrder: 19},
		{ID: "gym", Name: "健身房", Category: "training", Description: "训练身体技能。", IconKey: "gym", MaxLevel: 1, SortOrder: 20},
		{ID: "library", Name: "图书馆", Category: "training", Description: "提高行动经验和技能成长。", IconKey: "library", MaxLevel: 1, SortOrder: 21},
		{ID: "hall_of_fame", Name: "荣誉大厅", Category: "collection", Description: "陈列战利品并提高技能成长。", IconKey: "hall_of_fame", MaxLevel: 3, SortOrder: 22},
	}

	levels := []models.FacilityLevelDef{
		{FacilityID: "storage", Level: 0}, {FacilityID: "storage", Level: 1},
		{FacilityID: "storage", Level: 2, UpgradeCost: 1200, UpgradeSeconds: 900, MaterialID: "construction_tape", MaterialName: "建筑胶带", MaterialQuantity: 1, StorageBonus: 120},
		{FacilityID: "storage", Level: 3, UpgradeCost: 3000, UpgradeSeconds: 1800, MaterialID: "metal_spare_parts", MaterialName: "金属备件", MaterialQuantity: 1, StorageBonus: 240},

		{FacilityID: "security", Level: 0}, {FacilityID: "security", Level: 1, UpgradeCost: 800, UpgradeSeconds: 900, MaterialID: "metal_spare_parts", MaterialName: "金属备件", MaterialQuantity: 1},
		{FacilityID: "security", Level: 2, UpgradeCost: 2200, UpgradeSeconds: 1800, MaterialID: "military_cable", MaterialName: "军用电缆", MaterialQuantity: 1},
		{FacilityID: "security", Level: 3, UpgradeCost: 4800, UpgradeSeconds: 3600, MaterialID: "military_battery", MaterialName: "军用电池", MaterialQuantity: 1},

		{FacilityID: "ventilation", Level: 0}, {FacilityID: "ventilation", Level: 1, UpgradeCost: 900, UpgradeSeconds: 900, MaterialID: "duct_tape", MaterialName: "布基胶带", MaterialQuantity: 1},
		{FacilityID: "ventilation", Level: 2, UpgradeCost: 2400, UpgradeSeconds: 1800, MaterialID: "corrugated_hose", MaterialName: "波纹管", MaterialQuantity: 1},
		{FacilityID: "ventilation", Level: 3, UpgradeCost: 5200, UpgradeSeconds: 3600, MaterialID: "military_corrugated_tube", MaterialName: "军用波纹管", MaterialQuantity: 1},

		{FacilityID: "generator", Level: 0}, {FacilityID: "generator", Level: 1, UpgradeCost: 1200, UpgradeSeconds: 900, MaterialID: "propane_tank", MaterialName: "丙烷罐", MaterialQuantity: 1, FuelSlotCount: 1},
		{FacilityID: "generator", Level: 2, UpgradeCost: 3200, UpgradeSeconds: 1800, MaterialID: "car_battery", MaterialName: "汽车电瓶", MaterialQuantity: 1, FuelSlotCount: 2},
		{FacilityID: "generator", Level: 3, UpgradeCost: 7000, UpgradeSeconds: 3600, MaterialID: "military_battery", MaterialName: "军用电池", MaterialQuantity: 1, FuelSlotCount: 3},

		{FacilityID: "heating", Level: 0, EnergyRecoveryPerHour: 25}, {FacilityID: "heating", Level: 1, UpgradeCost: 700, UpgradeSeconds: 900, EnergyRecoveryPerHour: 40, MaterialID: "corrugated_hose", MaterialName: "波纹管", MaterialQuantity: 1},
		{FacilityID: "heating", Level: 2, UpgradeCost: 1800, UpgradeSeconds: 1800, EnergyRecoveryPerHour: 60, MaterialID: "analog_thermometer", MaterialName: "模拟温度计", MaterialQuantity: 1},
		{FacilityID: "heating", Level: 3, UpgradeCost: 4200, UpgradeSeconds: 3600, EnergyRecoveryPerHour: 80, MaterialID: "propane_tank", MaterialName: "丙烷罐", MaterialQuantity: 1},

		{FacilityID: "lighting", Level: 0}, {FacilityID: "lighting", Level: 1, UpgradeCost: 600, UpgradeSeconds: 900, MaterialID: "power_cord", MaterialName: "电源线", MaterialQuantity: 1},
		{FacilityID: "lighting", Level: 2, UpgradeCost: 1600, UpgradeSeconds: 1800, MaterialID: "printed_circuit_board", MaterialName: "电路板", MaterialQuantity: 1},
		{FacilityID: "lighting", Level: 3, UpgradeCost: 3600, UpgradeSeconds: 3600, MaterialID: "rechargeable_battery", MaterialName: "可充电电池", MaterialQuantity: 1},

		{FacilityID: "rest_area", Level: 0, StressRecoveryPerHour: 20}, {FacilityID: "rest_area", Level: 1, UpgradeCost: 600, UpgradeSeconds: 900, StressRecoveryPerHour: 30, MaterialID: "construction_tape", MaterialName: "建筑胶带", MaterialQuantity: 1},
		{FacilityID: "rest_area", Level: 2, UpgradeCost: 1600, UpgradeSeconds: 1800, StressRecoveryPerHour: 45, MaterialID: "set_of_tools", MaterialName: "工具套装", MaterialQuantity: 1},
		{FacilityID: "rest_area", Level: 3, UpgradeCost: 3600, UpgradeSeconds: 3600, StressRecoveryPerHour: 60, MaterialID: "electric_drill", MaterialName: "电钻", MaterialQuantity: 1},

		{FacilityID: "medstation", Level: 0, HPRecoveryPerHour: 25}, {FacilityID: "medstation", Level: 1, HPRecoveryPerHour: 50},
		{FacilityID: "medstation", Level: 2, UpgradeCost: 1000, UpgradeSeconds: 900, HPRecoveryPerHour: 75, MaterialID: "salewa", MaterialName: "Salewa 急救包", MaterialQuantity: 1},
		{FacilityID: "medstation", Level: 3, UpgradeCost: 2800, UpgradeSeconds: 1800, HPRecoveryPerHour: 100, MaterialID: "medical_tools", MaterialName: "医疗工具", MaterialQuantity: 1},

		{FacilityID: "nutrition_unit", Level: 0}, {FacilityID: "nutrition_unit", Level: 1, UpgradeCost: 700, UpgradeSeconds: 900, MaterialID: "iskra", MaterialName: "Iskra 口粮", MaterialQuantity: 1, RequiresPower: true},
		{FacilityID: "nutrition_unit", Level: 2, UpgradeCost: 1800, UpgradeSeconds: 1800, MaterialID: "condensed_milk", MaterialName: "炼乳罐头", MaterialQuantity: 1, RequiresPower: true},
		{FacilityID: "nutrition_unit", Level: 3, UpgradeCost: 4200, UpgradeSeconds: 3600, MaterialID: "emergency_water_ration", MaterialName: "紧急饮水口粮", MaterialQuantity: 1, RequiresPower: true},

		{FacilityID: "lavatory", Level: 0}, {FacilityID: "lavatory", Level: 1, UpgradeCost: 700, UpgradeSeconds: 900, MaterialID: "corrugated_hose", MaterialName: "波纹管", MaterialQuantity: 1, RequiresPower: true},
		{FacilityID: "lavatory", Level: 2, UpgradeCost: 1800, UpgradeSeconds: 1800, MaterialID: "silicone_tube", MaterialName: "硅胶管", MaterialQuantity: 1, RequiresPower: true},
		{FacilityID: "lavatory", Level: 3, UpgradeCost: 4200, UpgradeSeconds: 3600, MaterialID: "pressure_gauge", MaterialName: "压力表", MaterialQuantity: 1, RequiresPower: true},

		{FacilityID: "water_collector", Level: 0, HydrationRecoveryPerHour: 25}, {FacilityID: "water_collector", Level: 1, HydrationRecoveryPerHour: 40, MaterialID: "corrugated_hose", MaterialName: "波纹管", MaterialQuantity: 1},
		{FacilityID: "water_collector", Level: 2, UpgradeCost: 1800, UpgradeSeconds: 1800, HydrationRecoveryPerHour: 60, MaterialID: "silicone_tube", MaterialName: "硅胶管", MaterialQuantity: 1},
		{FacilityID: "water_collector", Level: 3, UpgradeCost: 4200, UpgradeSeconds: 3600, HydrationRecoveryPerHour: 80, MaterialID: "gas_analyzer", MaterialName: "气体分析仪", MaterialQuantity: 1},

		{FacilityID: "workbench", Level: 0}, {FacilityID: "workbench", Level: 1, RepairSpeedPercent: 0, RepairKitDiscountPercent: 0},
		{FacilityID: "workbench", Level: 2, UpgradeCost: 1400, UpgradeSeconds: 900, RepairSpeedPercent: 25, RepairKitDiscountPercent: 3, MaterialID: "set_of_tools", MaterialName: "工具套装", MaterialQuantity: 1},
		{FacilityID: "workbench", Level: 3, UpgradeCost: 3200, UpgradeSeconds: 1800, RepairSpeedPercent: 50, RepairKitDiscountPercent: 9, MaterialID: "electric_drill", MaterialName: "电钻", MaterialQuantity: 1},

		{FacilityID: "air_filter", Level: 0}, {FacilityID: "air_filter", Level: 1, PhysicalSkillGrowthPercent: 40, RequiresPower: true, UpgradeCost: 3600, UpgradeSeconds: 1800, MaterialID: "gas_analyzer", MaterialName: "气体分析仪", MaterialQuantity: 1},
		{FacilityID: "solar_panel", Level: 0}, {FacilityID: "solar_panel", Level: 1, FuelConsumptionReductionPercent: 50, UpgradeCost: 5200, UpgradeSeconds: 3600, MaterialID: "military_battery", MaterialName: "军用电池", MaterialQuantity: 1},

		{FacilityID: "intel", Level: 0}, {FacilityID: "intel", Level: 1, UpgradeCost: 1600, UpgradeSeconds: 900, IntelBonusPercent: 10, MaterialID: "printed_circuit_board", MaterialName: "电路板", MaterialQuantity: 1},
		{FacilityID: "intel", Level: 2, UpgradeCost: 3600, UpgradeSeconds: 1800, IntelBonusPercent: 20, MaterialID: "secure_flash_drive", MaterialName: "加密闪存盘", MaterialQuantity: 1},

		{FacilityID: "booze_generator", Level: 0},
		{FacilityID: "booze_generator", Level: 1, OriginalCost: 1780270, OriginalCurrency: "RUB", OriginalSeconds: 172800, UpgradeCost: 17803, UpgradeSeconds: 2880, EffectsJSON: `{"production":"moonshine"}`, RequiresPower: true},

		{FacilityID: "bitcoin_farm", Level: 0},
		{FacilityID: "bitcoin_farm", Level: 1, OriginalCost: 3295461, OriginalCurrency: "RUB", OriginalSeconds: 122400, UpgradeCost: 32955, UpgradeSeconds: 2040, EffectsJSON: `{"production":"physical_bitcoin","additionalSlots":10}`, RequiresPower: true},
		{FacilityID: "bitcoin_farm", Level: 2, OriginalCost: 2361237, OriginalCurrency: "RUB", OriginalSeconds: 180000, UpgradeCost: 23613, UpgradeSeconds: 3000, EffectsJSON: `{"production":"physical_bitcoin","additionalSlots":15}`, RequiresPower: true},
		{FacilityID: "bitcoin_farm", Level: 3, OriginalCost: 2669405, OriginalCurrency: "RUB", OriginalSeconds: 381600, UpgradeCost: 26695, UpgradeSeconds: 6360, EffectsJSON: `{"production":"physical_bitcoin","additionalSlots":25}`, RequiresPower: true},

		{FacilityID: "scav_case", Level: 0},
		{FacilityID: "scav_case", Level: 1, OriginalCost: 1667103, OriginalCurrency: "RUB", OriginalSeconds: 288000, UpgradeCost: 16672, UpgradeSeconds: 4800, EffectsJSON: `{"jobType":"scav_case"}`},

		{FacilityID: "shooting_range", Level: 0},
		{FacilityID: "shooting_range", Level: 1, OriginalCost: 127322, OriginalCurrency: "RUB", OriginalSeconds: 3600, UpgradeCost: 1274, UpgradeSeconds: 60, EffectsJSON: `{"jobType":"training","level":1}`},
		{FacilityID: "shooting_range", Level: 2, OriginalCost: 1346884, OriginalCurrency: "RUB", OriginalSeconds: 86400, UpgradeCost: 13469, UpgradeSeconds: 1440, EffectsJSON: `{"jobType":"training","level":2}`},
		{FacilityID: "shooting_range", Level: 3, OriginalCost: 1089795, OriginalCurrency: "RUB", OriginalSeconds: 86400, UpgradeCost: 10898, UpgradeSeconds: 1440, EffectsJSON: `{"jobType":"training","level":3}`},

		{FacilityID: "gym", Level: 0},
		{FacilityID: "gym", Level: 1, OriginalCost: 559720, OriginalCurrency: "RUB", OriginalSeconds: 14400, UpgradeCost: 5598, UpgradeSeconds: 240, EffectsJSON: `{"training":"physical"}`},

		{FacilityID: "library", Level: 0},
		{FacilityID: "library", Level: 1, OriginalCost: 1064436, OriginalCurrency: "RUB", OriginalSeconds: 194400, UpgradeCost: 10645, UpgradeSeconds: 3240, EffectsJSON: `{"raidExperienceBonusPercent":15,"skillGroupGrowthPercent":30}`},

		{FacilityID: "hall_of_fame", Level: 0},
		{FacilityID: "hall_of_fame", Level: 1, OriginalCost: 366382, OriginalCurrency: "RUB", OriginalSeconds: 43200, UpgradeCost: 3664, UpgradeSeconds: 720, EffectsJSON: `{"skillGroupGrowthPercent":0}`},
		{FacilityID: "hall_of_fame", Level: 2, OriginalCost: 952144, OriginalCurrency: "RUB", OriginalSeconds: 64800, UpgradeCost: 9522, UpgradeSeconds: 1080, EffectsJSON: `{"skillGroupGrowthPercent":0}`},
		{FacilityID: "hall_of_fame", Level: 3, OriginalCost: 1822206, OriginalCurrency: "RUB", OriginalSeconds: 86400, UpgradeCost: 18223, UpgradeSeconds: 1440, EffectsJSON: `{"skillGroupGrowthPercent":0}`},
	}

	requirements := []models.FacilityRequirement{
		{FacilityID: "storage", Level: 2, RequirementType: "item", ReferenceID: "pack_of_nails", Quantity: 4}, {FacilityID: "storage", Level: 2, RequirementType: "item", ReferenceID: "duct_tape", Quantity: 2}, {FacilityID: "storage", Level: 2, RequirementType: "item", ReferenceID: "bolts", Quantity: 4},
		{FacilityID: "storage", Level: 3, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 3}, {FacilityID: "storage", Level: 3, RequirementType: "item", ReferenceID: "pack_of_screws", Quantity: 5}, {FacilityID: "storage", Level: 3, RequirementType: "item", ReferenceID: "hand_drill", Quantity: 1}, {FacilityID: "storage", Level: 3, RequirementType: "facility", ReferenceID: "security", RequiredValue: 1},
		{FacilityID: "security", Level: 1, RequirementType: "item", ReferenceID: "construction_tape", Quantity: 1}, {FacilityID: "security", Level: 1, RequirementType: "item", ReferenceID: "bolts", Quantity: 2}, {FacilityID: "security", Level: 1, RequirementType: "item", ReferenceID: "screw_nuts", Quantity: 2},
		{FacilityID: "security", Level: 2, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 2}, {FacilityID: "security", Level: 2, RequirementType: "item", ReferenceID: "wd40", Quantity: 1}, {FacilityID: "security", Level: 2, RequirementType: "item", ReferenceID: "military_cable", Quantity: 1},
		{FacilityID: "security", Level: 3, RequirementType: "item", ReferenceID: "military_battery", Quantity: 1}, {FacilityID: "security", Level: 3, RequirementType: "item", ReferenceID: "military_cable", Quantity: 2}, {FacilityID: "security", Level: 3, RequirementType: "trader", ReferenceID: "mechanical", RequiredValue: 2},
		{FacilityID: "ventilation", Level: 1, RequirementType: "item", ReferenceID: "duct_tape", Quantity: 2}, {FacilityID: "ventilation", Level: 1, RequirementType: "item", ReferenceID: "corrugated_hose", Quantity: 1},
		{FacilityID: "ventilation", Level: 2, RequirementType: "item", ReferenceID: "corrugated_hose", Quantity: 3}, {FacilityID: "ventilation", Level: 2, RequirementType: "item", ReferenceID: "insulating_tape", Quantity: 2},
		{FacilityID: "ventilation", Level: 3, RequirementType: "item", ReferenceID: "military_corrugated_tube", Quantity: 1}, {FacilityID: "ventilation", Level: 3, RequirementType: "item", ReferenceID: "silicone_tube", Quantity: 2}, {FacilityID: "ventilation", Level: 3, RequirementType: "facility", ReferenceID: "security", RequiredValue: 1},
		{FacilityID: "generator", Level: 1, RequirementType: "item", ReferenceID: "spark_plug", Quantity: 1}, {FacilityID: "generator", Level: 1, RequirementType: "item", ReferenceID: "bundle_of_wires", Quantity: 2}, {FacilityID: "generator", Level: 1, RequirementType: "item", ReferenceID: "power_cord", Quantity: 1},
		{FacilityID: "generator", Level: 2, RequirementType: "item", ReferenceID: "car_battery", Quantity: 1}, {FacilityID: "generator", Level: 2, RequirementType: "item", ReferenceID: "phase_control_relay", Quantity: 2}, {FacilityID: "generator", Level: 2, RequirementType: "item", ReferenceID: "capacitors", Quantity: 3}, {FacilityID: "generator", Level: 2, RequirementType: "facility", ReferenceID: "lighting", RequiredValue: 1},
		{FacilityID: "generator", Level: 3, RequirementType: "item", ReferenceID: "military_battery", Quantity: 1}, {FacilityID: "generator", Level: 3, RequirementType: "item", ReferenceID: "electric_motor", Quantity: 1}, {FacilityID: "generator", Level: 3, RequirementType: "item", ReferenceID: "power_supply_unit", Quantity: 1},
		{FacilityID: "heating", Level: 1, RequirementType: "item", ReferenceID: "corrugated_hose", Quantity: 2}, {FacilityID: "heating", Level: 1, RequirementType: "item", ReferenceID: "duct_tape", Quantity: 1},
		{FacilityID: "heating", Level: 2, RequirementType: "item", ReferenceID: "analog_thermometer", Quantity: 1}, {FacilityID: "heating", Level: 2, RequirementType: "item", ReferenceID: "radiator_helix", Quantity: 2},
		{FacilityID: "heating", Level: 3, RequirementType: "item", ReferenceID: "propane_tank", Quantity: 1}, {FacilityID: "heating", Level: 3, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 2}, {FacilityID: "heating", Level: 3, RequirementType: "facility", ReferenceID: "generator", RequiredValue: 1},
		{FacilityID: "lighting", Level: 1, RequirementType: "item", ReferenceID: "light_bulb", Quantity: 5}, {FacilityID: "lighting", Level: 1, RequirementType: "item", ReferenceID: "energy_saving_lamp", Quantity: 2}, {FacilityID: "lighting", Level: 1, RequirementType: "item", ReferenceID: "power_cord", Quantity: 1},
		{FacilityID: "lighting", Level: 2, RequirementType: "item", ReferenceID: "printed_circuit_board", Quantity: 2}, {FacilityID: "lighting", Level: 2, RequirementType: "item", ReferenceID: "capacitors", Quantity: 3}, {FacilityID: "lighting", Level: 2, RequirementType: "item", ReferenceID: "bundle_of_wires", Quantity: 3},
		{FacilityID: "lighting", Level: 3, RequirementType: "item", ReferenceID: "rechargeable_battery", Quantity: 2}, {FacilityID: "lighting", Level: 3, RequirementType: "item", ReferenceID: "electric_motor", Quantity: 1}, {FacilityID: "lighting", Level: 3, RequirementType: "facility", ReferenceID: "generator", RequiredValue: 1},
		{FacilityID: "rest_area", Level: 1, RequirementType: "item", ReferenceID: "construction_tape", Quantity: 2}, {FacilityID: "rest_area", Level: 1, RequirementType: "item", ReferenceID: "pack_of_nails", Quantity: 2},
		{FacilityID: "rest_area", Level: 2, RequirementType: "item", ReferenceID: "set_of_tools", Quantity: 1}, {FacilityID: "rest_area", Level: 2, RequirementType: "item", ReferenceID: "duct_tape", Quantity: 2}, {FacilityID: "rest_area", Level: 2, RequirementType: "item", ReferenceID: "insulating_tape", Quantity: 2},
		{FacilityID: "rest_area", Level: 3, RequirementType: "item", ReferenceID: "electric_drill", Quantity: 1}, {FacilityID: "rest_area", Level: 3, RequirementType: "item", ReferenceID: "fleece_fabric", Quantity: 3},
		{FacilityID: "medstation", Level: 2, RequirementType: "item", ReferenceID: "salewa", Quantity: 1}, {FacilityID: "medstation", Level: 2, RequirementType: "item", ReferenceID: "saline_solution", Quantity: 3}, {FacilityID: "medstation", Level: 2, RequirementType: "item", ReferenceID: "esmarch", Quantity: 3}, {FacilityID: "medstation", Level: 2, RequirementType: "item", ReferenceID: "medical_bloodset", Quantity: 1},
		{FacilityID: "medstation", Level: 3, RequirementType: "item", ReferenceID: "medical_tools", Quantity: 2}, {FacilityID: "medstation", Level: 3, RequirementType: "item", ReferenceID: "pile_of_meds", Quantity: 2}, {FacilityID: "medstation", Level: 3, RequirementType: "facility", ReferenceID: "ventilation", RequiredValue: 2},
		{FacilityID: "nutrition_unit", Level: 1, RequirementType: "item", ReferenceID: "iskra", Quantity: 2}, {FacilityID: "nutrition_unit", Level: 1, RequirementType: "item", ReferenceID: "army_crackers", Quantity: 2},
		{FacilityID: "nutrition_unit", Level: 2, RequirementType: "item", ReferenceID: "condensed_milk", Quantity: 2}, {FacilityID: "nutrition_unit", Level: 2, RequirementType: "item", ReferenceID: "squash_spread", Quantity: 2},
		{FacilityID: "nutrition_unit", Level: 3, RequirementType: "item", ReferenceID: "emergency_water_ration", Quantity: 1}, {FacilityID: "nutrition_unit", Level: 3, RequirementType: "facility", ReferenceID: "generator", RequiredValue: 1},
		{FacilityID: "lavatory", Level: 1, RequirementType: "item", ReferenceID: "corrugated_hose", Quantity: 2}, {FacilityID: "lavatory", Level: 1, RequirementType: "item", ReferenceID: "duct_tape", Quantity: 1},
		{FacilityID: "lavatory", Level: 2, RequirementType: "item", ReferenceID: "silicone_tube", Quantity: 2}, {FacilityID: "lavatory", Level: 2, RequirementType: "item", ReferenceID: "pressure_gauge", Quantity: 1},
		{FacilityID: "lavatory", Level: 3, RequirementType: "item", ReferenceID: "pressure_gauge", Quantity: 1}, {FacilityID: "lavatory", Level: 3, RequirementType: "item", ReferenceID: "xenomorph_foam", Quantity: 2}, {FacilityID: "lavatory", Level: 3, RequirementType: "facility", ReferenceID: "water_collector", RequiredValue: 1},
		{FacilityID: "water_collector", Level: 1, RequirementType: "item", ReferenceID: "corrugated_hose", Quantity: 3}, {FacilityID: "water_collector", Level: 1, RequirementType: "item", ReferenceID: "bolts", Quantity: 2},
		{FacilityID: "water_collector", Level: 2, RequirementType: "item", ReferenceID: "silicone_tube", Quantity: 2}, {FacilityID: "water_collector", Level: 2, RequirementType: "item", ReferenceID: "analog_thermometer", Quantity: 1},
		{FacilityID: "water_collector", Level: 3, RequirementType: "item", ReferenceID: "gas_analyzer", Quantity: 1}, {FacilityID: "water_collector", Level: 3, RequirementType: "item", ReferenceID: "military_corrugated_tube", Quantity: 1},
		{FacilityID: "workbench", Level: 2, RequirementType: "item", ReferenceID: "set_of_tools", Quantity: 1}, {FacilityID: "workbench", Level: 2, RequirementType: "item", ReferenceID: "pack_of_screws", Quantity: 4}, {FacilityID: "workbench", Level: 2, RequirementType: "item", ReferenceID: "screw_nuts", Quantity: 4}, {FacilityID: "workbench", Level: 2, RequirementType: "item", ReferenceID: "bolts", Quantity: 4}, {FacilityID: "workbench", Level: 2, RequirementType: "item", ReferenceID: "wd40", Quantity: 1}, {FacilityID: "workbench", Level: 2, RequirementType: "item", ReferenceID: "pack_of_nails", Quantity: 3},
		{FacilityID: "workbench", Level: 3, RequirementType: "item", ReferenceID: "electric_drill", Quantity: 1}, {FacilityID: "workbench", Level: 3, RequirementType: "item", ReferenceID: "printed_circuit_board", Quantity: 3}, {FacilityID: "workbench", Level: 3, RequirementType: "item", ReferenceID: "capacitors", Quantity: 4}, {FacilityID: "workbench", Level: 3, RequirementType: "skill", ReferenceID: "engineering", RequiredValue: 40},
		{FacilityID: "air_filter", Level: 1, RequirementType: "item", ReferenceID: "gas_analyzer", Quantity: 3}, {FacilityID: "air_filter", Level: 1, RequirementType: "item", ReferenceID: "military_corrugated_tube", Quantity: 2}, {FacilityID: "air_filter", Level: 1, RequirementType: "facility", ReferenceID: "ventilation", RequiredValue: 3},
		{FacilityID: "solar_panel", Level: 1, RequirementType: "item", ReferenceID: "military_battery", Quantity: 1}, {FacilityID: "solar_panel", Level: 1, RequirementType: "item", ReferenceID: "printed_circuit_board", Quantity: 3}, {FacilityID: "solar_panel", Level: 1, RequirementType: "facility", ReferenceID: "generator", RequiredValue: 2},
		{FacilityID: "intel", Level: 1, RequirementType: "item", ReferenceID: "intelligence_folder", Quantity: 1}, {FacilityID: "intel", Level: 1, RequirementType: "item", ReferenceID: "topographic_maps", Quantity: 1}, {FacilityID: "intel", Level: 1, RequirementType: "item", ReferenceID: "factory_plan_map", Quantity: 1}, {FacilityID: "intel", Level: 1, RequirementType: "item", ReferenceID: "working_lcd", Quantity: 1}, {FacilityID: "intel", Level: 1, RequirementType: "item", ReferenceID: "power_cord", Quantity: 1},
		{FacilityID: "intel", Level: 2, RequirementType: "item", ReferenceID: "intelligence_folder", Quantity: 2}, {FacilityID: "intel", Level: 2, RequirementType: "item", ReferenceID: "secure_flash_drive", Quantity: 2}, {FacilityID: "intel", Level: 2, RequirementType: "item", ReferenceID: "power_cord", Quantity: 4}, {FacilityID: "intel", Level: 2, RequirementType: "item", ReferenceID: "damaged_hard_drive", Quantity: 3}, {FacilityID: "intel", Level: 2, RequirementType: "facility", ReferenceID: "ventilation", RequiredValue: 2},
		{FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "silicone_tube", Quantity: 10}, {FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "analog_thermometer", Quantity: 2}, {FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "pressure_gauge", Quantity: 2}, {FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "corrugated_hose", Quantity: 10}, {FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "pipe_grip_wrench", Quantity: 1}, {FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "radiator_helix", Quantity: 5}, {FacilityID: "booze_generator", Level: 1, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 5}, {FacilityID: "booze_generator", Level: 1, RequirementType: "facility", ReferenceID: "water_collector", RequiredValue: 3}, {FacilityID: "booze_generator", Level: 1, RequirementType: "facility", ReferenceID: "nutrition_unit", RequiredValue: 3},
		{FacilityID: "bitcoin_farm", Level: 1, RequirementType: "item", ReferenceID: "cpu_fan", Quantity: 15}, {FacilityID: "bitcoin_farm", Level: 1, RequirementType: "item", ReferenceID: "power_supply_unit", Quantity: 10}, {FacilityID: "bitcoin_farm", Level: 1, RequirementType: "item", ReferenceID: "power_cord", Quantity: 15}, {FacilityID: "bitcoin_farm", Level: 1, RequirementType: "item", ReferenceID: "vpx_flash_storage_module", Quantity: 2}, {FacilityID: "bitcoin_farm", Level: 1, RequirementType: "item", ReferenceID: "t_shaped_plug", Quantity: 10}, {FacilityID: "bitcoin_farm", Level: 1, RequirementType: "facility", ReferenceID: "intel", RequiredValue: 2},
		{FacilityID: "bitcoin_farm", Level: 2, RequirementType: "item", ReferenceID: "cpu_fan", Quantity: 15}, {FacilityID: "bitcoin_farm", Level: 2, RequirementType: "item", ReferenceID: "power_supply_unit", Quantity: 10}, {FacilityID: "bitcoin_farm", Level: 2, RequirementType: "item", ReferenceID: "printed_circuit_board", Quantity: 15}, {FacilityID: "bitcoin_farm", Level: 2, RequirementType: "item", ReferenceID: "phase_control_relay", Quantity: 10}, {FacilityID: "bitcoin_farm", Level: 2, RequirementType: "item", ReferenceID: "military_power_filter", Quantity: 5}, {FacilityID: "bitcoin_farm", Level: 2, RequirementType: "facility", ReferenceID: "bitcoin_farm", RequiredValue: 1}, {FacilityID: "bitcoin_farm", Level: 2, RequirementType: "facility", ReferenceID: "generator", RequiredValue: 3},
		{FacilityID: "bitcoin_farm", Level: 3, RequirementType: "item", ReferenceID: "cpu_fan", Quantity: 25}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "item", ReferenceID: "silicone_tube", Quantity: 15}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "item", ReferenceID: "electric_motor", Quantity: 10}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "item", ReferenceID: "pressure_gauge", Quantity: 10}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "item", ReferenceID: "military_battery", Quantity: 2}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "facility", ReferenceID: "bitcoin_farm", RequiredValue: 2}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "facility", ReferenceID: "solar_panel", RequiredValue: 1}, {FacilityID: "bitcoin_farm", Level: 3, RequirementType: "facility", ReferenceID: "water_collector", RequiredValue: 3},
		{FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "bronze_lion", Quantity: 3}, {FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "gold_skull_ring", Quantity: 6}, {FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "golden_neck_chain", Quantity: 8}, {FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "roler_watch", Quantity: 4}, {FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "moonshine", Quantity: 3}, {FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "lucky_scav_junk_box", Quantity: 1}, {FacilityID: "scav_case", Level: 1, RequirementType: "item", ReferenceID: "golden_rooster", Quantity: 1}, {FacilityID: "scav_case", Level: 1, RequirementType: "facility", ReferenceID: "intel", RequiredValue: 2},
		{FacilityID: "shooting_range", Level: 1, RequirementType: "item", ReferenceID: "screw_nuts", Quantity: 1}, {FacilityID: "shooting_range", Level: 1, RequirementType: "item", ReferenceID: "bolts", Quantity: 1}, {FacilityID: "shooting_range", Level: 1, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 1}, {FacilityID: "shooting_range", Level: 1, RequirementType: "facility", ReferenceID: "lighting", RequiredValue: 1},
		{FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "set_of_tools", Quantity: 1}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 8}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "construction_tape", Quantity: 1}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "bundle_of_wires", Quantity: 6}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "pack_of_screws", Quantity: 3}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "poxeram", Quantity: 3}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "electric_motor", Quantity: 3}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "energy_saving_lamp", Quantity: 6}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "wi_fi_camera", Quantity: 3}, {FacilityID: "shooting_range", Level: 2, RequirementType: "item", ReferenceID: "electric_drill", Quantity: 1}, {FacilityID: "shooting_range", Level: 2, RequirementType: "facility", ReferenceID: "lighting", RequiredValue: 3}, {FacilityID: "shooting_range", Level: 2, RequirementType: "facility", ReferenceID: "workbench", RequiredValue: 2}, {FacilityID: "shooting_range", Level: 2, RequirementType: "facility", ReferenceID: "shooting_range", RequiredValue: 1},
		{FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "set_of_files_master", Quantity: 1}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "bundle_of_wires", Quantity: 10}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "printed_circuit_board", Quantity: 5}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 5}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "capacitors", Quantity: 5}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "phase_control_relay", Quantity: 5}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "power_cord", Quantity: 5}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "leatherman", Quantity: 1}, {FacilityID: "shooting_range", Level: 3, RequirementType: "item", ReferenceID: "tech_manual", Quantity: 1}, {FacilityID: "shooting_range", Level: 3, RequirementType: "facility", ReferenceID: "shooting_range", RequiredValue: 2},
		{FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "set_of_tools", Quantity: 1}, {FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "electric_drill", Quantity: 1}, {FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "metal_cutting_scissors", Quantity: 1}, {FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "screw_nuts", Quantity: 5}, {FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "bolts", Quantity: 5}, {FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "wd40", Quantity: 1}, {FacilityID: "gym", Level: 1, RequirementType: "item", ReferenceID: "insulating_tape", Quantity: 5}, {FacilityID: "gym", Level: 1, RequirementType: "facility", ReferenceID: "lighting", RequiredValue: 2}, {FacilityID: "gym", Level: 1, RequirementType: "facility", ReferenceID: "ventilation", RequiredValue: 2},
		{FacilityID: "library", Level: 1, RequirementType: "item", ReferenceID: "horse_figurine", Quantity: 1}, {FacilityID: "library", Level: 1, RequirementType: "item", ReferenceID: "chainlet", Quantity: 2}, {FacilityID: "library", Level: 1, RequirementType: "item", ReferenceID: "tech_manual", Quantity: 8}, {FacilityID: "library", Level: 1, RequirementType: "item", ReferenceID: "diary", Quantity: 5}, {FacilityID: "library", Level: 1, RequirementType: "item", ReferenceID: "slim_diary", Quantity: 5}, {FacilityID: "library", Level: 1, RequirementType: "item", ReferenceID: "bakeezy_cookbook", Quantity: 1}, {FacilityID: "library", Level: 1, RequirementType: "facility", ReferenceID: "rest_area", RequiredValue: 3}, {FacilityID: "library", Level: 1, RequirementType: "skill", ReferenceID: "engineering", RequiredValue: 5},
		{FacilityID: "hall_of_fame", Level: 1, RequirementType: "item", ReferenceID: "round_pliers", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "item", ReferenceID: "fleece_fabric", Quantity: 5}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "item", ReferenceID: "cat_figurine", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "item", ReferenceID: "light_bulb", Quantity: 5}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "item", ReferenceID: "pack_of_nails", Quantity: 5}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "item", ReferenceID: "insulating_tape", Quantity: 5}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "facility", ReferenceID: "lighting", RequiredValue: 2}, {FacilityID: "hall_of_fame", Level: 1, RequirementType: "trader", ReferenceID: "mechanical", RequiredValue: 2},
		{FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "tech_manual", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "pliers_elite", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "set_of_tools", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "golden_rooster", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "energy_saving_lamp", Quantity: 10}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "pack_of_screws", Quantity: 6}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "duct_tape", Quantity: 3}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "xenomorph_foam", Quantity: 5}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "item", ReferenceID: "poxeram", Quantity: 2}, {FacilityID: "hall_of_fame", Level: 2, RequirementType: "facility", ReferenceID: "hall_of_fame", RequiredValue: 1},
		{FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "set_of_files_master", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "electric_drill", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "bronze_lion", Quantity: 1}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "energy_saving_lamp", Quantity: 15}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "kektape", Quantity: 3}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "metal_spare_parts", Quantity: 15}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "item", ReferenceID: "power_cord", Quantity: 5}, {FacilityID: "hall_of_fame", Level: 3, RequirementType: "facility", ReferenceID: "hall_of_fame", RequiredValue: 2},
	}

	for _, facility := range facilities {
		var stored models.FacilityDef
		err := db.Where("id = ?", facility.ID).First(&stored).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := db.Create(&facility).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if err := db.Model(&stored).Updates(facility).Error; err != nil {
				return err
			}
		}
	}
	for _, level := range levels {
		var stored models.FacilityLevelDef
		err := db.Where("facility_id = ? AND level = ?", level.FacilityID, level.Level).First(&stored).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := db.Create(&level).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if err := db.Model(&stored).Updates(level).Error; err != nil {
				return err
			}
		}
	}
	if err := db.Exec("DELETE FROM facility_requirements").Error; err != nil {
		return err
	}
	if len(requirements) > 0 {
		if err := db.Create(&requirements).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedHideoutForUser(db *gorm.DB, userID uint) error {
	initial := map[string]int{"storage": 1, "medstation": 1, "workbench": 1, "intel": 0}
	var facilities []models.FacilityDef
	if err := db.Order("sort_order asc, id asc").Find(&facilities).Error; err != nil {
		return err
	}
	for _, facility := range facilities {
		state := models.HideoutFacility{UserID: userID, FacilityID: facility.ID, Level: initial[facility.ID], State: "ready"}
		var stored models.HideoutFacility
		err := db.Where("user_id = ? AND facility_id = ?", userID, facility.ID).First(&stored).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := db.Create(&state).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		}
	}
	return nil
}
