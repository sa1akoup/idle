// 工作台制造配方种子：改编自逃离塔科夫 Wiki（Crafts/Workbench）真实配方，
// 输入物品映射到当前目录，制造时长按本作节奏压缩。
package config

import (
	"encoding/json"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

func seedCrafting(db *gorm.DB) error {
	recipes := []models.RecipeDef{
		// L1：装配与分装（对应 wiki：Toolset / Hand drill / Power cord→wires / Light bulbs）
		{ID: "craft_tool_set", Name: "工具组套装", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 2}, {ItemID: "pack_of_screws", Quantity: 2}, {ItemID: "bundle_of_wires", Quantity: 1}}),
			OutputItemID: "set_of_tools", OutputQuantity: 1, CraftSeconds: 1800, SortOrder: 1},
		{ID: "craft_electric_drill", Name: "手电钻改装", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 3}, {ItemID: "pack_of_screws", Quantity: 2}}),
			OutputItemID: "electric_drill", OutputQuantity: 1, CraftSeconds: 2700, SortOrder: 2},
		{ID: "craft_wire_coil", Name: "电线卷绕", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 1}, {ItemID: "energy_saving_lamp", Quantity: 1}}),
			OutputItemID: "bundle_of_wires", OutputQuantity: 3, CraftSeconds: 900, SortOrder: 3},
		{ID: "craft_lamp_assembly", Name: "灯泡装配", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "bundle_of_wires", Quantity: 1}, {ItemID: "pack_of_screws", Quantity: 1}}),
			OutputItemID: "energy_saving_lamp", OutputQuantity: 2, CraftSeconds: 1500, SortOrder: 4},
		{ID: "craft_spark_plug", Name: "火花塞组装", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 1}, {ItemID: "capacitors", Quantity: 1}}),
			OutputItemID: "spark_plug", OutputQuantity: 1, CraftSeconds: 1200, SortOrder: 5},
		{ID: "craft_power_cord", Name: "电源线重接", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "bundle_of_wires", Quantity: 2}}),
			OutputItemID: "power_cord", OutputQuantity: 2, CraftSeconds: 900, SortOrder: 6},
		{ID: "craft_hand_drill", Name: "手钻装配", FacilityID: "workbench", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 2}, {ItemID: "pack_of_screws", Quantity: 1}}),
			OutputItemID: "hand_drill", OutputQuantity: 1, CraftSeconds: 1800, SortOrder: 7},
		// L2：机电组装（对应 wiki：Electric motor / 拆解分装）
		{ID: "craft_electric_motor", Name: "电动机绕组", FacilityID: "workbench", RequiredLevel: 2,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "bundle_of_wires", Quantity: 2}, {ItemID: "metal_spare_parts", Quantity: 2}}),
			OutputItemID: "electric_motor", OutputQuantity: 1, CraftSeconds: 2400, SortOrder: 8},
		{ID: "craft_screw_split", Name: "五金分装", FacilityID: "workbench", RequiredLevel: 2,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 1}, {ItemID: "construction_tape", Quantity: 1}}),
			OutputItemID: "pack_of_screws", OutputQuantity: 3, CraftSeconds: 1200, SortOrder: 9},
		{ID: "craft_circuit_board", Name: "电路板焊接", FacilityID: "workbench", RequiredLevel: 2,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "capacitors", Quantity: 2}, {ItemID: "bundle_of_wires", Quantity: 1}}),
			OutputItemID: "printed_circuit_board", OutputQuantity: 1, CraftSeconds: 2100, SortOrder: 10},
		// L3：耐久成品（对应 wiki：Weapon repair kit 重制；燃料罐为同风格改编）
		{ID: "craft_weapon_repair_kit", Name: "武器维修组具重制", FacilityID: "workbench", RequiredLevel: 3,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "set_of_tools", Quantity: 1}, {ItemID: "metal_spare_parts", Quantity: 2}}),
			OutputItemID: "weapon_repair_kit_used", OutputQuantity: 1, CraftSeconds: 3600, SortOrder: 11},
		{ID: "craft_metal_fuel_tank", Name: "金属燃料罐改装", FacilityID: "workbench", RequiredLevel: 3,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "poxeram", Quantity: 1}, {ItemID: "metal_spare_parts", Quantity: 2}, {ItemID: "pack_of_screws", Quantity: 2}}),
			OutputItemID: "metal_fuel_tank", OutputQuantity: 1, CraftSeconds: 4500, SortOrder: 12},
		{ID: "craft_salewa", Name: "Salewa 组装", FacilityID: "medstation", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "pile_of_meds", Quantity: 1}, {ItemID: "esmarch", Quantity: 1}}),
			OutputItemID: "salewa", OutputQuantity: 1, CraftSeconds: 1200, SortOrder: 13},
		{ID: "craft_ifak", Name: "IFAK 分装", FacilityID: "medstation", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "pile_of_meds", Quantity: 1}, {ItemID: "disposable_syringe", Quantity: 1}}),
			OutputItemID: "ifak", OutputQuantity: 1, CraftSeconds: 900, SortOrder: 14},
		{ID: "craft_cms", Name: "CMS 手术包", FacilityID: "medstation", RequiredLevel: 2,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "medical_tools", Quantity: 1}, {ItemID: "pile_of_meds", Quantity: 2}}),
			OutputItemID: "cms", OutputQuantity: 1, CraftSeconds: 1800, SortOrder: 15},
		{ID: "craft_iskra", Name: "Iskra 口粮组合", FacilityID: "nutrition_unit", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "army_crackers", Quantity: 1}, {ItemID: "squash_spread", Quantity: 1}}),
			OutputItemID: "iskra", OutputQuantity: 1, CraftSeconds: 1200, SortOrder: 16},
		{ID: "craft_water_ration", Name: "紧急饮水分装", FacilityID: "nutrition_unit", RequiredLevel: 1,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "water_bottle", Quantity: 2}}),
			OutputItemID: "emergency_water_ration", OutputQuantity: 1, CraftSeconds: 600, SortOrder: 17},
	}

	for _, recipe := range recipes {
		var stored models.RecipeDef
		err := db.Where("id = ?", recipe.ID).First(&stored).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&recipe).Error; err != nil {
				return fmt.Errorf("创建制造配方 %s: %w", recipe.ID, err)
			}
		case err != nil:
			return fmt.Errorf("读取制造配方 %s: %w", recipe.ID, err)
		default:
			if err := db.Model(&stored).Updates(map[string]interface{}{
				"name": recipe.Name, "facility_id": recipe.FacilityID, "required_level": recipe.RequiredLevel,
				"inputs_json": recipe.InputsJSON, "output_item_id": recipe.OutputItemID,
				"output_quantity": recipe.OutputQuantity, "craft_seconds": recipe.CraftSeconds,
				"sort_order": recipe.SortOrder,
			}).Error; err != nil {
				return fmt.Errorf("更新制造配方 %s: %w", recipe.ID, err)
			}
		}
	}
	return nil
}

func mustInputs(inputs []models.RecipeInput) string {
	encoded, err := json.Marshal(inputs)
	if err != nil {
		// 常量来源，序列化不可能失败；兜底为空数组保持行为可预期。
		return "[]"
	}
	return string(encoded)
}
