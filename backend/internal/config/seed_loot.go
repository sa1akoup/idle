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
		{ID: "corrugated_hose", Name: "波纹管", Category: "material", Desc: "用于维修水路和工业设施的波纹软管。", Price: 120, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "duct_tape", Name: "布基胶带", Category: "material", Desc: "比普通胶带更结实的工业胶带。", Price: 100, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "insulating_tape", Name: "绝缘胶带", Category: "material", Desc: "电气维修用绝缘胶带。", Price: 75, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "piece_of_plexiglass", Name: "有机玻璃片", Category: "material", Desc: "透明的工程塑料板材。", Price: 140, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "screw_nuts", Name: "螺母", Category: "material", Desc: "一袋通用六角螺母。", Price: 45, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "silicone_tube", Name: "硅胶管", Category: "material", Desc: "耐热耐腐蚀的硅胶软管。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "pressure_gauge", Name: "压力表", Category: "tool", Desc: "用于检测管路压力的仪表。", Price: 230, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "analog_thermometer", Name: "模拟温度计", Category: "tool", Desc: "工业设备上的机械温度计。", Price: 160, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "metal_spare_parts", Name: "金属备件", Category: "material", Desc: "一批可用于修理设备的金属零件。", Price: 300, Weight: 2, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "military_corrugated_tube", Name: "军用波纹管", Category: "material", Desc: "军用设备使用的耐压波纹管。", Price: 520, Weight: 2, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "nippers", Name: "剥线钳", Category: "tool", Desc: "用于剪切和剥离电线的钳子。", Price: 120, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "hand_drill", Name: "手钻", Category: "tool", Desc: "无需电源即可使用的手动钻具。", Price: 210, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "sewing_kit", Name: "缝纫套件", Category: "tool", Desc: "用于修补衣物和装备的针线套件。", Price: 190, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "bulbex_cable_cutter", Name: "Bulbex 电缆剪", Category: "tool", Desc: "专门剪切粗电缆的工业剪。", Price: 380, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "fierce_blow_sledgehammer", Name: "Fierce Blow 大锤", Category: "tool", Desc: "用于破拆和重型施工的大锤。", Price: 650, Weight: 5, Slots: 2, MerchantCategory: "mechanical"},

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
		{ID: "greenbat", Name: "GreenBat 锂电池", Category: "electronics", Desc: "高容量锂电池组。", Price: 420, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "cyclon", Name: "Cyclon 充电电池", Category: "electronics", Desc: "可重复充电的工业电池。", Price: 360, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "aa_battery", Name: "AA 电池", Category: "electronics", Desc: "常见的五号电池。", Price: 35, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "car_battery", Name: "汽车电瓶", Category: "electronics", Desc: "汽车用铅酸蓄电池。", Price: 950, Weight: 8, Slots: 2, MerchantCategory: "mechanical"},
		{ID: "d_size_battery", Name: "D 型电池", Category: "electronics", Desc: "大号圆柱电池。", Price: 80, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "portable_powerbank", Name: "便携式充电宝", Category: "electronics", Desc: "用于给小型设备供电的移动电源。", Price: 240, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "rechargeable_battery", Name: "可充电电池", Category: "electronics", Desc: "通用规格的充电电池。", Price: 130, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "military_battery", Name: "6-STEN-140-M 军用电池", Category: "electronics", Desc: "军用通信设备的高容量电池。", Price: 1200, Weight: 4, Slots: 2, MerchantCategory: "union"},
		{ID: "damaged_hard_drive", Name: "损坏的硬盘", Category: "electronics", Desc: "外壳损坏但仍可拆解回收的硬盘。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "gas_analyzer", Name: "气体分析仪", Category: "electronics", Desc: "检测空气成分和工业气体的仪器。", Price: 650, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "geiger_counter", Name: "盖革计数器", Category: "electronics", Desc: "检测辐射水平的便携仪器。", Price: 720, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "pc_cpu", Name: "PC 处理器", Category: "electronics", Desc: "从电脑中拆出的处理器。", Price: 480, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "power_supply_unit", Name: "电源供应器", Category: "electronics", Desc: "台式电脑用电源模块。", Price: 300, Weight: 2, Slots: 2, MerchantCategory: "mechanical"},
		{ID: "tetriz", Name: "Tetriz 便携游戏机", Category: "electronics", Desc: "带有收藏价值的便携式游戏机。", Price: 1250, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "usb_adapter", Name: "USB 转接器", Category: "electronics", Desc: "用于连接不同接口设备的小型转接器。", Price: 160, Weight: 1, Slots: 1, MerchantCategory: "mechanical"},
		{ID: "virtex", Name: "Virtex 可编程处理器", Category: "electronics", Desc: "高规格可编程计算模块。", Price: 1800, Weight: 1, Slots: 1, MerchantCategory: "union"},

		// 情报与文件
		{ID: "diary", Name: "日记本", Category: "info", Desc: "记录着封锁区居民生活线索的日记。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "intelligence_folder", Name: "情报文件夹", Category: "info", Desc: "整理过的行动情报和人员档案。", Price: 500, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "tech_manual", Name: "技术手册", Category: "info", Desc: "工业设备的技术说明书。", Price: 380, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "topographic_maps", Name: "地形调查地图", Category: "info", Desc: "标注城区地形与旧设施的地图。", Price: 450, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "sas_drive", Name: "SAS 硬盘", Category: "info", Desc: "服务器拆下的高速硬盘。", Price: 700, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "terragroup_blue_folder", Name: "TerraGroup 蓝色文件夹", Category: "info", Desc: "与 TerraGroup 有关的机密资料。", Price: 1200, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "slim_diary", Name: "薄型日记本", Category: "info", Desc: "记录着封锁区人员和路线的薄型日记。", Price: 260, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "military_flash_drive", Name: "军用闪存盘", Category: "info", Desc: "军用终端使用的加密存储设备。", Price: 1100, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "secure_magnetic_tape", Name: "加密磁带", Category: "info", Desc: "保存旧系统备份的加密磁带。", Price: 620, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "bakeezy_cookbook", Name: "BakeEzy 食谱", Category: "info", Desc: "封锁前流行的烹饪书。", Price: 150, Weight: 1, Slots: 1, MerchantCategory: "union"},
		{ID: "cyborg_killer_cassette", Name: "《赛博杀手》录像带", Category: "info", Desc: "一盘带有收藏价值的电影录像带。", Price: 420, Weight: 1, Slots: 1, MerchantCategory: "black"},

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
		{ID: "hydrogen_peroxide", Name: "过氧化氢", Category: "medical", Desc: "用于清洁伤口的消毒液。", Price: 100, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "ololo_multivitamins", Name: "OLOLO 复合维生素", Category: "medical", Desc: "瓶装复合维生素。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "medical_bloodset", Name: "医疗血液采集套装", Category: "medical", Desc: "用于采集和处理血液的医疗套装。", Price: 280, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "ophthalmoscope", Name: "检眼镜", Category: "medical", Desc: "用于检查眼底的医疗仪器。", Price: 700, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "portable_defibrillator", Name: "便携式除颤器", Category: "medical", Desc: "用于急救的便携式除颤设备。", Price: 1400, Weight: 3, Slots: 2, MerchantCategory: "medical"},
		{ID: "saline_solution", Name: "生理盐水", Category: "medical", Desc: "医疗输液用盐水瓶。", Price: 170, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "pile_of_meds", Name: "一堆药品", Category: "medical", Desc: "从药房搜集的散装药品。", Price: 230, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "disposable_syringe", Name: "一次性注射器", Category: "medical", Desc: "无菌一次性注射器。", Price: 90, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "medical_tools", Name: "医疗工具", Category: "medical", Desc: "用于手术和创伤处理的医疗器械。", Price: 520, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "aquapeps", Name: "Aquapeps 净水片", Category: "medical", Desc: "用于净化饮用水的药片。", Price: 210, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "nooby_shield", Name: "Nooby Shield 碘片", Category: "medical", Desc: "用于防护辐射的碘化钾片。", Price: 260, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "aceso_analyzer", Name: "Aceso 生化分析仪", Category: "medical", Desc: "半自动生化检测设备。", Price: 1600, Weight: 2, Slots: 2, MerchantCategory: "medical"},
		{ID: "golden_star_balm", Name: "Golden Star 药膏", Category: "medical", Desc: "用于缓解伤痛的药膏。", Price: 230, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "morphine_injector", Name: "吗啡注射器", Category: "medical", Desc: "强效镇痛注射器。", Price: 480, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "ledx", Name: "LEDX 皮肤透照仪", Category: "medical", Desc: "高价值的专业医疗扫描设备。", Price: 8500, Weight: 1, Slots: 2, MerchantCategory: "black"},
		{ID: "propital_injector", Name: "Propital 再生兴奋剂", Category: "medical", Desc: "用于短时间恢复的生化注射器。", Price: 850, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "squash_spread", Name: "南瓜泥罐头", Category: "food", Desc: "高热量蔬菜泥罐头。", Price: 130, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "army_crackers", Name: "军用饼干", Category: "food", Desc: "耐储存的军用压缩饼干。", Price: 95, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "green_peas", Name: "青豆罐头", Category: "food", Desc: "罐装青豆。", Price: 85, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "herring", Name: "鲱鱼罐头", Category: "food", Desc: "咸味鱼类罐头。", Price: 100, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "majaica_coffee", Name: "Majaica 咖啡豆", Category: "food", Desc: "一罐烘焙咖啡豆。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "humpback_salmon", Name: "驼背鲑鱼罐头", Category: "food", Desc: "罐装鲑鱼。", Price: 120, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "pacific_saury", Name: "太平洋秋刀鱼罐头", Category: "food", Desc: "罐装秋刀鱼。", Price: 110, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "signature_tea", Name: "Signature Blend 英式茶", Category: "food", Desc: "高档英式混合茶。", Price: 250, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "dr_lupo_coffee", Name: "Dr. Lupo 咖啡豆", Category: "food", Desc: "罐装精品咖啡豆。", Price: 220, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "water_bottle", Name: "0.6L 饮用水", Category: "food", Desc: "瓶装饮用水。", Price: 50, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "emergency_water_ration", Name: "紧急饮水口粮", Category: "food", Desc: "应急情况下使用的饮水包。", Price: 140, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "condensed_milk", Name: "炼乳罐头", Category: "food", Desc: "高热量炼乳罐头。", Price: 115, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "tarkovskaya_vodka", Name: "Tarkovskaya 伏特加", Category: "food", Desc: "封锁区常见的伏特加。", Price: 180, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "devildog_mayo", Name: "DevilDog 蛋黄酱", Category: "food", Desc: "高热量蛋黄酱罐。", Price: 130, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "white_salt", Name: "食盐罐", Category: "food", Desc: "一罐食用盐。", Price: 70, Weight: 1, Slots: 1, MerchantCategory: "medical"},
		{ID: "pickles", Name: "腌黄瓜罐头", Category: "food", Desc: "罐装腌黄瓜。", Price: 85, Weight: 1, Slots: 1, MerchantCategory: "medical"},

		// 贵重物
		{ID: "gold_chain", Name: "金链", Category: "valuable", Desc: "可快速变现的金饰。", Price: 1200, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "golden_rooster", Name: "金色公鸡", Category: "valuable", Desc: "镀金的收藏摆件。", Price: 3000, Weight: 2, Slots: 2, MerchantCategory: "black"},
		{ID: "bronze_lion", Name: "青铜狮", Category: "valuable", Desc: "沉重的青铜狮雕像。", Price: 2700, Weight: 3, Slots: 2, MerchantCategory: "black"},
		{ID: "horse_figurine", Name: "马雕像", Category: "valuable", Desc: "小型马匹收藏雕像。", Price: 850, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "cat_figurine", Name: "猫雕像", Category: "valuable", Desc: "常见的猫形收藏摆件。", Price: 900, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "gp_coin", Name: "GP 币", Category: "valuable", Desc: "带有特殊标记的贵金属纪念币。", Price: 1000, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "physical_bitcoin", Name: "实物比特币", Category: "valuable", Desc: "刻有比特币标识的贵金属收藏品。", Price: 10000, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "antique_teapot", Name: "古董茶壶", Category: "valuable", Desc: "有收藏价值的古董茶壶。", Price: 1800, Weight: 2, Slots: 2, MerchantCategory: "black"},
		{ID: "antique_vase", Name: "古董花瓶", Category: "valuable", Desc: "易碎的古董花瓶。", Price: 2100, Weight: 2, Slots: 2, MerchantCategory: "black"},
		{ID: "raven_figurine", Name: "乌鸦雕像", Category: "valuable", Desc: "黑色乌鸦收藏雕像。", Price: 1350, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "chain_prokill", Name: "Prokill 纪念章项链", Category: "valuable", Desc: "带有 Prokill 纪念章的项链。", Price: 1700, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "golden_neck_chain", Name: "金色项链", Category: "valuable", Desc: "精致的金色项链。", Price: 1600, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "roler_watch", Name: "Roler 潜水金表", Category: "valuable", Desc: "Roler 品牌的金色潜水腕表。", Price: 2600, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "silver_badge", Name: "银质徽章", Category: "valuable", Desc: "带有旧组织标志的银质徽章。", Price: 1900, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "wooden_clock", Name: "木制时钟", Category: "valuable", Desc: "做工精细的木制摆钟。", Price: 1200, Weight: 2, Slots: 2, MerchantCategory: "black"},
		{ID: "gold_skull_ring", Name: "金骷髅戒指", Category: "valuable", Desc: "带有骷髅图案的金戒指。", Price: 2300, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "golden_egg", Name: "金蛋", Category: "valuable", Desc: "金色的收藏摆件。", Price: 2800, Weight: 2, Slots: 2, MerchantCategory: "black"},
		{ID: "axel_parrot", Name: "Axel 鹦鹉雕像", Category: "valuable", Desc: "色彩鲜艳的鹦鹉收藏雕像。", Price: 1500, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "battered_antique_book", Name: "破旧古书", Category: "valuable", Desc: "年代久远且磨损严重的古书。", Price: 900, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "old_firesteel", Name: "老式打火钢", Category: "valuable", Desc: "旧时代的金属打火工具。", Price: 600, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "tarcoin", Name: "TarCoin", Category: "valuable", Desc: "封锁区流通的特殊纪念币。", Price: 1250, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "duck_figurine", Name: "鸭子雕像", Category: "valuable", Desc: "黄色橡皮鸭收藏品。", Price: 520, Weight: 1, Slots: 1, MerchantCategory: "black"},
		{ID: "loot_lord_plushie", Name: "Loot Lord 毛绒玩具", Category: "valuable", Desc: "带有 Loot Lord 标识的毛绒玩具。", Price: 980, Weight: 1, Slots: 2, MerchantCategory: "black"},

		// 武器零件与战场杂项
		{ID: "weapon_parts", Name: "武器零件", Category: "weaponpart", Desc: "可用于维修和改造武器的通用零件。", Price: 260, Weight: 1, Slots: 1, MerchantCategory: "weapon"},
		{ID: "weapon_repair_kit_used", Name: "用过的武器维修包", Category: "weaponpart", Desc: "仍有部分余量的武器维修工具包。", Price: 480, Weight: 2, Slots: 2, MerchantCategory: "weapon"},
		{ID: "fireklean_lube", Name: "FireKlean 枪械润滑油", Category: "weaponpart", Desc: "用于保养枪械的高品质润滑油。", Price: 320, Weight: 1, Slots: 1, MerchantCategory: "weapon"},
		{ID: "ofz_shell", Name: "OFZ 30x165mm 炮弹", Category: "weaponpart", Desc: "重型武器使用的炮弹。", Price: 900, Weight: 3, Slots: 2, MerchantCategory: "weapon"},
		{ID: "dogtag", Name: "身份牌", Category: "weaponpart", Desc: "记录身份信息的金属身份牌。", Price: 150, Weight: 1, Slots: 1, MerchantCategory: "black"},
	}

	return upsertLootItems(db, items)
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
