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
			InputsJSON: mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 2}, {ItemID: "pack_of_screws", Quantity: 2}, {ItemID: "bundle_of_wires", Quantity: 1}}),
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
		// L2：机电组装（对应 wiki：Electric motor / 拆解分装）
		{ID: "craft_electric_motor", Name: "电动机绕组", FacilityID: "workbench", RequiredLevel: 2,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "bundle_of_wires", Quantity: 2}, {ItemID: "metal_spare_parts", Quantity: 2}}),
			OutputItemID: "electric_motor", OutputQuantity: 1, CraftSeconds: 2400, SortOrder: 5},
		{ID: "craft_screw_split", Name: "五金分装", FacilityID: "workbench", RequiredLevel: 2,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "metal_spare_parts", Quantity: 1}, {ItemID: "construction_tape", Quantity: 1}}),
			OutputItemID: "pack_of_screws", OutputQuantity: 3, CraftSeconds: 1200, SortOrder: 6},
		// L3：耐久成品（对应 wiki：Weapon repair kit 重制；燃料罐为同风格改编）
		{ID: "craft_weapon_repair_kit", Name: "武器维修组具重制", FacilityID: "workbench", RequiredLevel: 3,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "set_of_tools", Quantity: 1}, {ItemID: "metal_spare_parts", Quantity: 2}}),
			OutputItemID: "weapon_repair_kit_used", OutputQuantity: 1, CraftSeconds: 3600, SortOrder: 7},
		{ID: "craft_metal_fuel_tank", Name: "金属燃料罐改装", FacilityID: "workbench", RequiredLevel: 3,
			InputsJSON:   mustInputs([]models.RecipeInput{{ItemID: "poxeram", Quantity: 1}, {ItemID: "metal_spare_parts", Quantity: 2}, {ItemID: "pack_of_screws", Quantity: 2}}),
			OutputItemID: "metal_fuel_tank", OutputQuantity: 1, CraftSeconds: 4500, SortOrder: 8},
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
