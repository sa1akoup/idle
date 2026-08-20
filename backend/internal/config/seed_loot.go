package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedLoot 初始化可搜集战利品目录；物品名称与分类参考 Escape from Tarkov Wiki 的 Loot 页面。
// 目录只维护物品定义，实际数量由探索成功后的库存结算逻辑产生。
func seedLoot(db *gorm.DB) error {
	items := []models.LootItemDef{
		// 工具与建材
		{ID: "screwdriver", Name: "螺丝刀", Category: "tool", Desc: "常见的手持螺丝刀。", Price: 60, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "construction_tape", Name: "建筑测量卷尺", Category: "tool", Desc: "用于测量建筑结构的卷尺。", Price: 80, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "wrench", Name: "扳手", Category: "tool", Desc: "维护机械设备的通用扳手。", Price: 90, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "set_of_tools", Name: "工具套装", Category: "tool", Desc: "装在箱中的成套维修工具。", Price: 350, Weight: 2, Slots: 2, MerchantCategory: "mechanical"},
		{ID: "awl", Name: "锥子", Category: "tool", Desc: "用于皮革与软质材料加工的手工具。", Price: 50, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "pliers", Name: "钳子", Category: "tool", Desc: "普通钢制钳子。", Price: 70, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "bolts", Name: "螺栓", Category: "tool", Desc: "一袋工业用螺栓。", Price: 35, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "screws", Name: "螺丝", Category: "tool", Desc: "一盒各种规格的螺丝。", Price: 35, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "kektape", Name: "KEKTAPE 胶带", Category: "tool", Desc: "结实耐用的多用途胶带。", Price: 90, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "pack_of_nails", Name: "一包钉子", Category: "tool", Desc: "建筑施工用的钢钉。", Price: 65, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "electric_drill", Name: "电钻", Category: "tool", Desc: "用于快速打孔的电动工具。", Price: 420, Weight: 2, Slots: 2, MerchantCategory: "mechanical"},
		{ID: "propane_tank", Name: "丙烷罐", Category: "fuel", Desc: "用于焊接和加热设备的丙烷罐。", Price: 900, Weight: 5, Slots: 2, MerchantCategory: "mechanical"},
		{ID: "metal_fuel_tank", Name: "金属燃油罐", Category: "fuel", Desc: "可储存燃料的军用金属罐。", Price: 1100, Weight: 4, Slots: 2, MerchantCategory: "mechanical"},

		// 电子与军用技术物资
		{ID: "power_cord", Name: "电源线", Category: "electronics", Desc: "带插头的通用电源线。", Price: 110, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "printed_circuit_board", Name: "电路板", Category: "electronics", Desc: "从家用设备中拆出的印刷电路板。", Price: 240, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "capacitors", Name: "电容", Category: "electronics", Desc: "电子设备维修用电容元件。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "cpu_fan", Name: "CPU 风扇", Category: "electronics", Desc: "电脑处理器散热风扇。", Price: 260, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "ram", Name: "内存条", Category: "electronics", Desc: "拆机获得的电脑内存模块。", Price: 350, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "broken_gphone", Name: "损坏的 GPhone 手机", Category: "electronics", Desc: "屏幕破裂但仍有回收价值的智能手机。", Price: 280, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "ssd", Name: "SSD 固态硬盘", Category: "electronics", Desc: "可读取的固态存储设备。", Price: 750, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "secure_flash_drive", Name: "加密闪存盘", Category: "electronics", Desc: "带有加密分区的便携式存储设备。", Price: 950, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "military_circuit_board", Name: "军用电路板", Category: "electronics", Desc: "军用设备拆出的高规格电路板。", Price: 1400, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "military_cable", Name: "军用电缆", Category: "electronics", Desc: "带屏蔽层的军用通信电缆。", Price: 950, Weight: 2, Slots: 1, MerchantCategory: "union"},
		{ID: "graphics_card", Name: "显卡", Category: "electronics", Desc: "高性能计算机显卡。", Price: 2200, Weight: 1, Slots: 2, MerchantCategory: "black"},

		// 情报与文件
		{ID: "diary", Name: "日记本", Category: "info", Desc: "记录着封锁区居民生活线索的日记。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "intelligence_folder", Name: "情报文件夹", Category: "info", Desc: "整理过的行动情报和人员档案。", Price: 500, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "tech_manual", Name: "技术手册", Category: "info", Desc: "工业设备的技术说明书。", Price: 380, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "topographic_maps", Name: "地形调查地图", Category: "info", Desc: "标注城区地形与旧设施的地图。", Price: 450, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "sas_drive", Name: "SAS 硬盘", Category: "info", Desc: "服务器拆下的高速硬盘。", Price: 700, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "terragroup_blue_folder", Name: "TerraGroup 蓝色文件夹", Category: "info", Desc: "与 TerraGroup 有关的机密资料。", Price: 1200, Weight: 1, Slots: 1, MerchantCategory: "union"},

		// 医疗与食品
		{ID: "ai2_medkit", Name: "AI-2 急救包", Category: "medical", Desc: "苏制单兵应急医疗包。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "salewa", Name: "Salewa 急救包", Category: "medical", Desc: "民用便携式急救包。", Price: 400, Weight: 2, Slots: 2, MerchantCategory: "medical"},
		{ID: "ifak", Name: "IFAK 急救包", Category: "medical", Desc: "单兵个人急救包。", Price: 350, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "calok", Name: "CALOK-B 止血剂", Category: "medical", Desc: "用于快速止血的注射器。", Price: 260, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "esmarch", Name: "Esmarch 止血带", Category: "medical", Desc: "橡胶止血带。", Price: 160, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "vaseline", Name: "凡士林", Category: "medical", Desc: "可用于医疗和基础护理的凡士林。", Price: 240, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "cms", Name: "CMS 手术包", Category: "medical", Desc: "便携式创伤处理手术包。", Price: 500, Weight: 2, Slots: 2, MerchantCategory: "medical"},
		{ID: "iskra", Name: "Iskra 口粮", Category: "food", Desc: "军用单兵口粮。", Price: 220, Weight: 2, Slots: 1, MerchantCategory: "medical"},
		{ID: "alyonka", Name: "Alyonka 巧克力", Category: "food", Desc: "高热量巧克力。", Price: 90, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "slickers", Name: "Slickers 巧克力", Category: "food", Desc: "便携式巧克力能量棒。", Price: 110, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "aquamari", Name: "Aquamari 水", Category: "food", Desc: "便携式饮用水。", Price: 160, Weight: 2, Slots: 1, MerchantCategory: "medical"},
		{ID: "tarcola", Name: "TarCola", Category: "food", Desc: "含咖啡因的碳酸饮料。", Price: 80, Weight: 1, Slots: 1, MerchantCategory: "medical"},

		// 贵重物
		{ID: "gold_chain", Name: "金链", Category: "valuable", Desc: "可快速变现的金饰。", Price: 1200, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "golden_rooster", Name: "金色公鸡", Category: "valuable", Desc: "镀金的收藏摆件。", Price: 3000, Weight: 2, Slots: 2, MerchantCategory: "black"},
		{ID: "bronze_lion", Name: "青铜狮", Category: "valuable", Desc: "沉重的青铜狮雕像。", Price: 2700, Weight: 3, Slots: 2, MerchantCategory: "black"},
		{ID: "horse_figurine", Name: "马雕像", Category: "valuable", Desc: "小型马匹收藏雕像。", Price: 850, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "cat_figurine", Name: "猫雕像", Category: "valuable", Desc: "常见的猫形收藏摆件。", Price: 900, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "gp_coin", Name: "GP 币", Category: "valuable", Desc: "带有特殊标记的贵金属纪念币。", Price: 1000, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "physical_bitcoin", Name: "实物比特币", Category: "valuable", Desc: "刻有比特币标识的贵金属收藏品。", Price: 10000, Weight: 1, Slots: 1, MerchantCategory: "black"},
	}

	if err := upsertLootItems(db, items); err != nil {
		return err
	}
	return seedLootExpansion(db)
}

func upsertLootItems(db *gorm.DB, items []models.LootItemDef) error {
	for _, item := range items {
		if err := db.FirstOrCreate(&item, models.LootItemDef{ID: item.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.LootItemDef{}).Where("id = ?", item.ID).Updates(item).Error; err != nil {
			return err
		}
	}
	return nil
}
