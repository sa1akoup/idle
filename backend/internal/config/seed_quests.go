package config

import (
	"encoding/json"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

func seedQuests(db *gorm.DB) error {
	defs := []models.QuestDef{
		quest("weapon_debut", "weapon", 1, "首战撤离", "完成一次成功撤离，证明你还能从封锁区活着回来。", "",
			models.QuestObjectiveSurviveRuns, objSurvive(1), reward("weapon", 800, 3), 1),
		quest("weapon_patrol", "weapon", 2, "清扫巡逻", "击倒 3 名巡逻单位后撤离。", "weapon_debut",
			models.QuestObjectiveDefeatKind, objKind("grunt", 3), reward("weapon", 1200, 4), 2),
		quest("weapon_customs", "weapon", 3, "海关调查", "进入海关办公楼并成功撤离。", "weapon_patrol",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_8"), reward("weapon", 1500, 5), 3),
		quest("weapon_guard", "weapon", 4, "拔除据点", "击倒 2 名守卫后撤离。", "weapon_customs",
			models.QuestObjectiveDefeatKind, objKind("guard", 2), reward("weapon", 1800, 6), 4),
		quest("weapon_elite", "weapon", 5, "精锐猎杀", "击倒 1 名精英单位后撤离。", "weapon_guard",
			models.QuestObjectiveDefeatKind, objKind("elite", 1), reward("weapon", 2500, 8), 5),
		quest("weapon_container", "weapon", 6, "货场清扫", "进入集装箱场并成功撤离。", "weapon_elite",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_6"), reward("weapon", 2200, 7), 6),
		quest("weapon_parts", "weapon", 7, "零件回收", "上交 2 份从局内带出的武器零件。", "weapon_container",
			models.QuestObjectiveExtractItem, objItem("weapon_parts", 2), reward("weapon", 2600, 8), 7),
		quest("weapon_aggressive", "weapon", 8, "强攻撤离", "以激进型风格成功撤离 1 局。", "weapon_parts",
			models.QuestObjectiveStyleExtract, objStyle("aggressive", 1), reward("weapon", 3000, 10), 8),
		quest("weapon_key", "weapon", 9, "海关钥匙", "上交 1 把从局内带出的海关内务室钥匙。去居民楼翻夹克，或搜海关文件柜。", "weapon_aggressive",
			models.QuestObjectiveExtractItem, objItem("key_customs_office", 1), reward("weapon", 2800, 9), 9),
		quest("weapon_safe", "weapon", 10, "金柜取样", "上交 1 条从局内带出的金链。海关保险箱和旧市收银机最容易出。", "weapon_key",
			models.QuestObjectiveExtractItem, objItem("gold_chain", 1), reward("weapon", 3200, 10), 10),

		quest("medical_shortage", "medical", 1, "药品短缺", "上交 1 个从局内带出的 Salewa 急救包。商店购买的不算。", "",
			models.QuestObjectiveExtractItem, objItem("salewa", 1), reward("medical", 1000, 4), 10),
		quest("medical_clinic", "medical", 2, "诊所勘查", "进入社区诊所并成功撤离。", "medical_shortage",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_7"), reward("medical", 1200, 4), 11),
		quest("medical_stealth", "medical", 3, "低扰动撤离", "以隐秘型风格成功撤离 1 局。", "medical_clinic",
			models.QuestObjectiveStyleExtract, objStyle("stealth", 1), reward("medical", 1500, 5), 12),
		quest("medical_intel", "medical", 4, "伤员名册", "上交 1 本从局内带出的日记本。", "medical_stealth",
			models.QuestObjectiveExtractItem, objItem("diary", 1), reward("medical", 2000, 6), 13),
		quest("medical_saline", "medical", 5, "输液补给", "上交 2 瓶从局内带出的生理盐水。", "medical_intel",
			models.QuestObjectiveExtractItem, objItem("saline_solution", 2), reward("medical", 2200, 7), 14),
		quest("medical_market", "medical", 6, "市场采买点", "进入旧市场并成功撤离。", "medical_saline",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_3"), reward("medical", 2400, 7), 15),
		quest("medical_ifak", "medical", 7, "单兵急救", "上交 1 个从局内带出的 IFAK 急救包。", "medical_market",
			models.QuestObjectiveExtractItem, objItem("ifak", 1), reward("medical", 2800, 9), 16),
		quest("medical_key", "medical", 8, "药房钥匙", "上交 1 把从局内带出的诊所药房钥匙。夹克和诊所文件最容易摸到。", "medical_ifak",
			models.QuestObjectiveExtractItem, objItem("key_clinic_pharmacy", 1), reward("medical", 2600, 8), 17),
		quest("medical_meds", "medical", 9, "散装药品", "上交 2 份从局内带出的一堆药品。社区诊所的塑料医疗箱里常见。", "medical_key",
			models.QuestObjectiveExtractItem, objItem("pile_of_meds", 2), reward("medical", 3000, 10), 18),

		quest("mechanical_screws", "mechanical", 1, "基础紧固件", "上交 2 包从局内带出的螺丝。商人柜台买的不算。", "",
			models.QuestObjectiveExtractItem, objItem("pack_of_screws", 2), reward("mechanical", 800, 3), 20),
		quest("mechanical_hose", "mechanical", 2, "管路维修", "上交 1 根从局内带出的波纹管。", "mechanical_screws",
			models.QuestObjectiveExtractItem, objItem("corrugated_hose", 1), reward("mechanical", 1200, 5), 21),
		quest("mechanical_warehouse", "mechanical", 3, "仓库勘查", "进入废弃仓库并成功撤离。", "mechanical_hose",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_2"), reward("mechanical", 1400, 5), 22),
		quest("mechanical_board", "mechanical", 4, "电路回收", "上交 1 块从局内带出的电路板。", "mechanical_warehouse",
			models.QuestObjectiveExtractItem, objItem("printed_circuit_board", 1), reward("mechanical", 2000, 8), 23),
		quest("mechanical_bolts", "mechanical", 5, "螺栓清单", "上交 4 袋从局内带出的螺栓。", "mechanical_board",
			models.QuestObjectiveExtractItem, objItem("bolts", 4), reward("mechanical", 1800, 6), 24),
		quest("mechanical_spark", "mechanical", 6, "点火组件", "上交 1 个从局内带出的火花塞。", "mechanical_bolts",
			models.QuestObjectiveExtractItem, objItem("spark_plug", 1), reward("mechanical", 2200, 8), 25),
		quest("mechanical_gas", "mechanical", 7, "油站机房", "抵达加油站并成功撤离。", "mechanical_spark",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_9"), reward("mechanical", 2400, 8), 26),
		quest("mechanical_motor", "mechanical", 8, "动力回收", "上交 1 台从局内带出的电动机。", "mechanical_gas",
			models.QuestObjectiveExtractItem, objItem("electric_motor", 1), reward("mechanical", 3200, 10), 27),
		quest("mechanical_key", "mechanical", 9, "仓库钥匙", "上交 1 把从局内带出的仓库办公室钥匙。去居民楼或仓库翻夹克。", "mechanical_motor",
			models.QuestObjectiveExtractItem, objItem("key_warehouse_office", 1), reward("mechanical", 2800, 9), 28),
		quest("mechanical_battery", "mechanical", 10, "车载电瓶", "上交 1 块从局内带出的汽车电瓶。加油站油料堆和弃置轿车事件会出。", "mechanical_key",
			models.QuestObjectiveExtractItem, objItem("car_battery", 1), reward("mechanical", 3400, 10), 29),

		quest("clothing_survive", "clothing", 1, "着装评估", "成功撤离 2 局。", "",
			models.QuestObjectiveSurviveRuns, objSurvive(2), reward("clothing", 900, 3), 30),
		quest("clothing_fabric", "clothing", 2, "布料征用", "上交 1 块从局内带出的抓绒布。", "clothing_survive",
			models.QuestObjectiveExtractItem, objItem("fleece_fabric", 1), reward("clothing", 1300, 4), 31),
		quest("clothing_gas", "clothing", 3, "加油站锚点", "抵达加油站并成功撤离。", "clothing_fabric",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_9"), reward("clothing", 1800, 6), 32),
		quest("clothing_sewing", "clothing", 4, "缝补套件", "上交 1 套从局内带出的缝纫套件。", "clothing_gas",
			models.QuestObjectiveExtractItem, objItem("sewing_kit", 1), reward("clothing", 2000, 6), 33),
		quest("clothing_market", "clothing", 5, "旧市布摊", "进入旧市场并成功撤离。", "clothing_sewing",
			models.QuestObjectiveVisitNode, objNode("city_ruins_node_3"), reward("clothing", 2200, 7), 34),
		quest("clothing_tape", "clothing", 6, "封箱胶带", "上交 2 卷从局内带出的布基胶带。", "clothing_market",
			models.QuestObjectiveExtractItem, objItem("duct_tape", 2), reward("clothing", 2600, 8), 35),
		quest("clothing_chain", "clothing", 7, "旧市金饰", "上交 1 条从局内带出的金链。旧市收银机、海关保险箱和夹克口袋都会出。", "clothing_tape",
			models.QuestObjectiveExtractItem, objItem("gold_chain", 1), reward("clothing", 3000, 9), 36),
	}
	for _, def := range defs {
		var stored models.QuestDef
		err := db.Where("id = ?", def.ID).First(&stored).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&def).Error; err != nil {
				return fmt.Errorf("创建合同 %s: %w", def.ID, err)
			}
		case err != nil:
			return fmt.Errorf("读取合同 %s: %w", def.ID, err)
		default:
			if err := db.Model(&stored).Updates(map[string]interface{}{
				"merchant_id": def.MerchantID, "chain_index": def.ChainIndex, "name": def.Name,
				"description": def.Description, "prerequisite_id": def.PrerequisiteID,
				"objective_type": def.ObjectiveType, "objective_json": def.ObjectiveJSON,
				"reward_json": def.RewardJSON, "sort_order": def.SortOrder,
			}).Error; err != nil {
				return fmt.Errorf("更新合同 %s: %w", def.ID, err)
			}
		}
	}
	return nil
}

func quest(id, merchant string, chain int, name, desc, prereq, objType string, objective models.QuestObjective, reward models.QuestReward, sort int) models.QuestDef {
	return models.QuestDef{
		ID: id, MerchantID: merchant, ChainIndex: chain, Name: name, Description: desc,
		PrerequisiteID: prereq, ObjectiveType: objType, ObjectiveJSON: mustJSON(objective),
		RewardJSON: mustJSON(reward), SortOrder: sort,
	}
}

func objItem(id string, quantity int) models.QuestObjective {
	return models.QuestObjective{ItemID: id, Quantity: quantity}
}
func objKind(kind string, quantity int) models.QuestObjective {
	return models.QuestObjective{Kind: kind, Quantity: quantity}
}
func objNode(nodeID string) models.QuestObjective {
	return models.QuestObjective{NodeID: nodeID, Quantity: 1}
}
func objStyle(style string, quantity int) models.QuestObjective {
	return models.QuestObjective{Style: style, Quantity: quantity}
}
func objSurvive(quantity int) models.QuestObjective {
	return models.QuestObjective{Quantity: quantity}
}
func reward(merchant string, cash, rep int) models.QuestReward {
	return models.QuestReward{Cash: cash, MerchantID: merchant, Reputation: rep}
}

func mustJSON(value interface{}) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
