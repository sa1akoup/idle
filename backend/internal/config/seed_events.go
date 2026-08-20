// 通用事件配置：集中定义事件、作用域绑定与地图遭遇角色池，供多地图复用。
package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	modeExploring  = "exploring"
	modeEvacuating = "evacuating"
)

// seedEvents 全量刷新静态事件配置，避免旧结构残留的数据在启动校验时继续生效。
func seedEvents(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, value := range []interface{}{&models.EventBinding{}, &models.EncounterPoolEntry{}, &models.EventDef{}} {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(value).Error; err != nil {
				return err
			}
		}

		definitions := eventDefinitions()
		if err := tx.Create(&definitions).Error; err != nil {
			return err
		}
		bindings := eventBindings()
		if err := tx.Create(&bindings).Error; err != nil {
			return err
		}
		pools := encounterPools()
		return tx.Create(&pools).Error
	})
}

func eventDefinitions() []models.EventDef {
	return []models.EventDef{
		// 战斗事件：只声明通用遭遇角色，具体敌人由地图遭遇池解析。
		event("evt_common_patrol", "临时巡逻队", "一支巡逻队正沿节点推进。", "combat", "encounter", "repeat",
			styled(checked("evade", nil, "stealth", 45, "", 0, "提前隐蔽，避开巡逻。", "巡逻队封住去路。", effects(fx("skip_combat", "", 0)), effects(fx("encounter", "patrol", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("ambush", nil, "利用巡逻队的行进间隙主动伏击。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "ambush", 30)),
		event("evt_elite_patrol", "精英巡逻队", "装备精良的小队正在执行清扫。", "combat", "encounter", "once_per_run",
			styled(checked("observe", nil, "perception", 58, "", 0, "及时发现其搜索队形并绕开。", "精英小队抢先占据交火位置。", effects(fx("skip_combat", "", 0), fx("time", "", 2)), effects(fx("stress", "", 8), fx("encounter", "elite", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("force", nil, "抓住精英小队换位的空当，主动接战。", effects(fx("encounter", "elite", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_enemy_ambush", "近距离伏击", "隐蔽火力突然从侧面出现。", "combat", "encounter", "once_per_run",
			styled(automatic("withdraw", nil, "不与伏击纠缠，立即脱离火力范围。", effects(fx("skip_combat", "", 0), fx("time", "", 2))), []string{"stealth"}, "withdraw", 35),
			styled(checked("react", nil, "agility", 52, "", 0, "及时转移，迫使伏击者暴露。", "反应稍慢，被迫仓促接战。", effects(fx("encounter", "patrol", 0)), effects(fx("hp", "", -8), fx("encounter", "guard", 0))), []string{"balanced"}, "ambush", 30),
			styled(automatic("counter", nil, "抓住伏击者暴露的瞬间，立即反击。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "ambush", 35)),
		event("evt_sniper_fire", "远程狙击", "远处制高点出现疑似狙击火力。", "combat", "encounter", "once_per_run",
			styled(checked("locate", nil, "perception", 58, "", 0, "确认射界后从掩体间穿过。", "未能及时定位射手。", effects(fx("skip_combat", "", 0), fx("time", "", 2)), effects(fx("hp", "", -12), fx("encounter", "sniper", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "锁定制高点，快速压制狙击火力。", effects(fx("encounter", "sniper", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_checkpoint", "临时盘查哨", "武装人员正在盘查所有通行者。", "combat", "encounter", "once_per_run",
			styled(checked("negotiate", nil, "negotiation", 50, "", 0, "话术奏效，顺利通过哨卡。", "身份说辞被识破。", effects(fx("skip_combat", "", 0), fx("time", "", 1)), effects(fx("heat", "", 10), fx("encounter", "guard", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "拒绝接受盘查，趁哨兵分散时强行突破。", effects(fx("encounter", "guard", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_reinforcements", "增援抵达", "无线电里传来增援接近的短促呼叫。", "combat", "encounter", "once_per_run",
			styled(checked("reroute", nil, "survival", 55, "", 0, "从维护通道抢先离开封锁区。", "增援完成合围。", effects(fx("skip_combat", "", 0), fx("time", "", 3)), effects(fx("encounter", "elite", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "抢在增援完成展开前，主动迎击先头队。", effects(fx("encounter", "elite", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_rival_scavengers", "竞争者小队", "另一支搜索小队盯上了同一片区域。", "combat", "encounter", "once_per_run",
			styled(checked("deal", nil, "charisma", 52, "", 0, "双方交换情报后各自离开。", "谈判破裂，对方抢先举枪。", effects(fx("skip_combat", "", 0), fx("container_pool", "supply_reward", 0)), effects(fx("encounter", "guard", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "先下手夺取搜索区域的控制权。", effects(fx("encounter", "guard", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_cornered_guard", "困兽守卫", "一名落单守卫守住了唯一通道。", "combat", "encounter", "once_per_run",
			styled(checked("slip", nil, "stealth", 50, "", 0, "利用死角无声穿过。", "脚步声暴露了位置。", effects(fx("skip_combat", "", 0)), effects(fx("stress", "", 6), fx("encounter", "patrol", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "压制守卫，清理唯一通道。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "engage", 30)),

		// 探索事件：可产出容器、恢复状态或直接形成返航目标。
		event("evt_high_value_intel", "高价值情报室", "一扇加密门后仍有设备在运行。", "exploration", "", "once_per_run",
			checked("decrypt", []string{modeExploring}, "intellect", 58, "toolkit", 15, "取得完整情报，继续滞留风险过高。", "数据大部分损坏，只能结束读取。", effects(fx("container_pool", "intel_reward", 0), fx("set_flag", "high_value_intel", 1), fx("start_evacuation", "target_acquired", 0)), effects(fx("time", "", 4), fx("heat", "", 5)))),
		event("evt_safehouse", "废弃安全屋", "隐蔽房间里还留有少量储备。", "exploration", "", "once_per_run",
			checked("secure", []string{modeExploring}, "survival", 45, "", 0, "确认房间安全，获得短暂休整。", "清理房间耗费了额外时间。", effects(fx("hp", "", 12), fx("stress", "", -15), fx("container_pool", "safehouse_reward", 0)), effects(fx("time", "", 3)))),
		event("evt_material_storage", "材料存储间", "封存货架上堆着可回收材料。", "exploration", "", "once_per_run",
			automatic("search", []string{modeExploring}, "快速分类后装入背包。", effects(fx("container_pool", "material_reward", 0)))),
		event("evt_medical_room", "临时医疗室", "废弃救护点还留有基础药品。", "exploration", "", "once_per_run",
			checked("treat", []string{modeExploring}, "medical", 45, "bandage", 15, "处理伤口并清点药柜。", "仅找到零散耗材。", effects(fx("hp", "", 15), fx("stress", "", -8), fx("container_pool", "medical_reward", 0)), effects(fx("container_pool", "medical_fallback", 0), fx("time", "", 2)))),
		event("evt_maintenance_workshop", "维护工坊", "工作台仍可用于简单维修。", "exploration", "", "once_per_run",
			checked("repair", []string{modeExploring}, "engineering", 48, "toolkit", 20, "完成临时修补并搜出零件。", "修补失败，只能带走散件。", effects(fx("armor", "", 12), fx("container_pool", "workshop_reward", 0)), effects(fx("time", "", 4), fx("container_pool", "workshop_fallback", 0)))),
		event("evt_sealed_cache", "密封储藏柜", "储藏柜的电子锁仍在工作。", "exploration", "", "once_per_run",
			checked("unlock", []string{modeExploring}, "engineering", 58, "toolkit", 20, "绕过锁控，完整打开储藏柜。", "强行处理制造了明显噪声。", effects(fx("container_pool", "workshop_reward", 0)), effects(fx("time", "", 3), fx("heat", "", 8)))),
		event("evt_abandoned_camp", "遗弃营地", "简易营地似乎刚被匆忙放弃。", "exploration", "", "once_per_run",
			checked("inspect", []string{modeExploring}, "perception", 48, "", 0, "排除诱饵后找到隐藏储备。", "搜查没有结果，紧张感持续上升。", effects(fx("container_pool", "safehouse_reward", 0)), effects(fx("stress", "", 5), fx("time", "", 2)))),
		event("evt_hidden_stash", "隐蔽夹层", "墙板后的空间似乎被反复开启过。", "exploration", "", "once_per_run",
			checked("search", []string{modeExploring}, "luck", 50, "", 0, "夹层中仍有一批补给。", "夹层早已被清空。", effects(fx("container_pool", "supply_reward", 0)), effects(fx("time", "", 2)))),
		event("evt_server_room", "残存服务器机房", "部分阵列仍保持低功耗运行。", "exploration", "", "once_per_run",
			checked("extract", []string{modeExploring}, "intellect", 56, "toolkit", 15, "导出一批可交易的数据。", "读取触发了设备报警。", effects(fx("container_pool", "server_reward", 0), fx("heat", "", 4)), effects(fx("time", "", 5), fx("heat", "", 9)))),
		event("evt_survivor_cache", "幸存者物资点", "有人正守着一处共享物资点。", "exploration", "", "once_per_run",
			checked("request", []string{modeExploring}, "charisma", 50, "", 0, "对方同意分出一部分物资。", "沟通没有进展，双方保持戒备。", effects(fx("container_pool", "supply_reward", 0), fx("stress", "", -5)), effects(fx("stress", "", 5)))),

		// 环境与行动事件：改变当前状态，并继续参与自动撤离阈值判断。
		event("evt_accidental_injury", "意外受伤", "碎裂地面和外露钢筋阻断了直线路径。", "hazard", "hazard", "once_per_run",
			checked("cross", nil, "agility", 50, "", 0, "稳住重心安全穿过。", "踩空后被锐物划伤。", effects(fx("time", "", 1)), effects(fx("hp", "", -12), fx("time", "", 2)))),
		event("evt_structural_collapse", "结构坍塌", "上层结构突然松动并开始坠落。", "hazard", "hazard", "once_per_run",
			checked("brace", nil, "strength", 55, "", 0, "推开障碍后继续行动。", "被碎块击中，护甲也受到损伤。", effects(fx("time", "", 2)), effects(fx("hp", "", -15), fx("armor", "", -8)))),
		event("evt_toxic_leak", "有毒泄漏", "空气中出现刺激性气味。", "hazard", "hazard", "once_per_run",
			checked("endure", nil, "resist", 55, "", 0, "屏息快速通过污染区。", "吸入污染物，身体状况恶化。", effects(fx("time", "", 2)), effects(fx("hp", "", -10), fx("stress", "", 8)))),
		event("evt_power_outage", "照明中断", "区域照明和指示标记同时熄灭。", "hazard", "hazard", "once_per_run",
			checked("navigate", nil, "perception", 50, "", 0, "借助环境轮廓找到出口。", "在黑暗中走了弯路并制造声响。", effects(fx("time", "", 2)), effects(fx("time", "", 5), fx("heat", "", 5)))),
		event("evt_lost_route", "路线迷失", "原定路径与现场结构无法对应。", "hazard", "hazard", "once_per_run",
			checked("orient", nil, "survival", 50, "", 0, "根据地标快速修正方向。", "反复折返浪费了时间。", effects(fx("time", "", 1)), effects(fx("time", "", 6)))),
		event("evt_noise_exposure", "噪声暴露", "松动物件倒地，在空旷区域形成回声。", "hazard", "hazard", "once_per_run",
			checked("silence", nil, "stealth", 50, "", 0, "及时接住物件，动静被控制住。", "噪声向周边暴露了位置。", nil, effects(fx("heat", "", 12)))),
		event("evt_flooded_path", "积水路段", "深水盖住了路面缺口。", "hazard", "hazard", "once_per_run",
			checked("wade", nil, "agility", 50, "", 0, "沿稳固边缘缓慢通过。", "跌入积水，装备受到撞击。", effects(fx("time", "", 3)), effects(fx("armor", "", -10), fx("time", "", 5)))),
		event("evt_local_fire", "局部火灾", "燃烧物正在向搜索区域蔓延。", "hazard", "hazard", "once_per_run",
			checked("contain", nil, "survival", 55, "", 0, "找到安全间隙穿过火场。", "高温迫使行动员放弃本节点搜索。", effects(fx("time", "", 3)), effects(fx("hp", "", -12), fx("skip_search", "", 0)))),

		// 撤离事件：仅撤离模式可选，沿途依然执行遭遇、判定和资源损耗。
		event("evt_enemy_pursuit", "敌人追击", "后方脚步与无线电呼叫持续接近。", "evacuation", "encounter", "once_per_run",
			styled(checked("break_contact", []string{modeEvacuating}, "stealth", 55, "smoke", 20, "成功拉开距离并降低暴露。", "追击者逼近，必须交战。", effects(fx("heat", "", -5)), effects(fx("encounter", "patrol", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", []string{modeEvacuating}, "停止撤退，回身压制追击者。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_route_blockade", "撤离路线封锁", "临时障碍和火力点截断了短路。", "evacuation", "encounter", "once_per_run",
			styled(checked("detour", []string{modeEvacuating}, "survival", 55, "", 0, "找到未被监视的穿行缺口。", "绕行失败，封锁守卫发现了目标。", effects(fx("evac_shortcut", "", 0), fx("time", "", 2)), effects(fx("encounter", "guard", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", []string{modeEvacuating}, "直接突破封锁火力，抢出撤离窗口。", effects(fx("encounter", "guard", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_injury_worsens", "伤势恶化", "持续移动令现有伤口重新出血。", "evacuation", "", "once_per_run",
			conditioned("stabilize", []string{modeEvacuating}, condition("hp_ratio", "lt", "", 0.6), "medical", 50, "bandage", 20, "完成紧急止血。", "伤口继续恶化。", effects(fx("hp", "", 5)), effects(fx("hp", "", -10)))),
		event("evt_forced_discard", "负重拖慢撤离", "携带物资正在明显降低移动效率。", "evacuation", "", "once_per_run",
			conditioned("carry", []string{modeEvacuating}, condition("carry_ratio", "gte", "", 0.5), "strength", 55, "", 0, "重新固定背包后保持速度。", "只能丢弃部分物资恢复机动。", effects(fx("time", "", 2)), effects(fx("discard_loot", "", 2)))),
		event("evt_hidden_shortcut", "撤离捷径", "一条维护通路可能直达外围。", "evacuation", "", "once_per_run",
			checked("identify", []string{modeEvacuating}, "perception", 50, "", 0, "确认通路方向，缩短一段路线。", "入口已经堵死。", effects(fx("evac_shortcut", "", 0)), effects(fx("time", "", 2)))),
		event("evt_signal_disrupted", "导航信号干扰", "定位信号出现持续漂移。", "evacuation", "", "once_per_run",
			checked("restore", []string{modeEvacuating}, "engineering", 50, "toolkit", 15, "恢复定位并校正路线。", "定位失败，只能依赖地标移动。", effects(fx("time", "", 2)), effects(fx("time", "", 5), fx("heat", "", 5)))),
		event("evt_extraction_ambush", "撤离点外围伏兵", "撤离点附近出现有组织的拦截力量。", "evacuation", "encounter", "once_per_run",
			styled(checked("spot", []string{modeEvacuating}, "perception", 58, "", 0, "提前发现伏兵并避开交叉火力。", "进入伏击圈后被迫接战。", effects(fx("skip_combat", "", 0)), effects(fx("encounter", "extraction", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", []string{modeEvacuating}, "抢在伏兵合围前，主动清理撤离点外围。", effects(fx("encounter", "extraction", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_smoke_corridor", "烟幕通道", "开阔地带缺少足够掩护。", "evacuation", "encounter", "once_per_run",
			conditioned("deploy", []string{modeEvacuating}, condition("has_item", "eq", "smoke", 1), "", 0, "", 0, "投放烟幕并快速脱离危险区。", "", effects(fx("consume_item", "smoke", 0), fx("skip_combat", "", 0)), nil)),
		event("evt_field_treatment", "撤离途中急救", "短暂掩体提供了一次处理伤势的机会。", "evacuation", "", "once_per_run",
			checked("treat", []string{modeEvacuating}, "medical", 48, "medkit", 20, "稳定生命状态后继续撤离。", "处理效果有限。", effects(fx("hp", "", 10), fx("stress", "", -5)), effects(fx("time", "", 3)))),
		event("evt_extraction_window", "撤离窗口变化", "接应窗口受到现场态势影响。", "evacuation", "", "once_per_run",
			fixed("window", []string{modeEvacuating}, 70, "接应提前完成准备。", "接应延迟，需要继续隐蔽等待。", effects(fx("time", "", -2), fx("heat", "", -5)), effects(fx("time", "", 4)))),

		// 废弃城区节点事件：与通用事件同阶段竞争，节点绑定优先级更高。
		event("evt_apt_safehouse", "居民楼隐蔽室", "住户改造的夹层保留着应急储备。", "node", "", "once_per_run",
			automatic("open", []string{modeExploring}, "打开夹层并清点物资。", effects(fx("container_pool", "safehouse_reward", 0), fx("stress", "", -6)))),
		event("evt_apt_survivor_ambush", "居民楼误判", "楼内幸存者将行动员当成掠夺者。", "node", "encounter", "once_per_run",
			styled(checked("explain", nil, "negotiation", 52, "", 0, "解除误会并交换少量补给。", "对方拒绝沟通并发起攻击。", effects(fx("skip_combat", "", 0), fx("container_pool", "supply_reward", 0)), effects(fx("encounter", "patrol", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "不再解释，先控制住误判现场。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_warehouse_material", "仓库完整货架", "一排未被登记的货架仍然完好。", "node", "", "once_per_run",
			automatic("collect", []string{modeExploring}, "找到成批可用材料。", effects(fx("container_pool", "material_reward", 0)))),
		event("evt_warehouse_shelf", "倾倒货架", "锈蚀货架在经过时突然倾斜。", "node", "hazard", "once_per_run",
			checked("dodge", nil, "agility", 52, "", 0, "及时侧身避开倒塌范围。", "被货架边缘撞伤。", nil, effects(fx("hp", "", -8), fx("time", "", 2)))),
		event("evt_customs_intel_vault", "海关情报库", "封存终端保存着完整货运记录。", "node", "", "once_per_run",
			checked("recover", []string{modeExploring}, "intellect", 60, "toolkit", 15, "导出高价值货运情报，应立即返航。", "只恢复出少量破损记录。", effects(fx("container_pool", "intel_reward", 0), fx("set_flag", "customs_intel", 1), fx("start_evacuation", "target_acquired", 0)), effects(fx("container_pool", "intel_fallback", 0), fx("heat", "", 8)))),
		event("evt_customs_lockdown", "海关门禁封锁", "自动门禁将楼层切成多个隔离区。", "node", "encounter", "once_per_run",
			styled(checked("override", nil, "engineering", 58, "toolkit", 20, "解除门禁并避开警戒力量。", "警报引来精英守卫。", effects(fx("skip_combat", "", 0), fx("time", "", 2)), effects(fx("heat", "", 10), fx("encounter", "elite", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "强行闯入封锁区，正面压制精英守卫。", effects(fx("encounter", "elite", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_tunnel_drain", "地下排水支路", "旧排水支路朝码头方向延伸。", "node", "", "once_per_run",
			checked("trace", []string{modeEvacuating}, "survival", 50, "", 0, "确认支路可用，缩短一段撤离路线。", "支路尽头已经坍塌。", effects(fx("evac_shortcut", "", 0)), effects(fx("time", "", 3)))),
		event("evt_tunnel_gas", "地下通道积气", "低洼处积聚着成分不明的气体。", "node", "hazard", "once_per_run",
			checked("hold_breath", nil, "resist", 55, "", 0, "快速穿过积气区域。", "呼吸受到刺激，状态下降。", effects(fx("time", "", 2)), effects(fx("hp", "", -10), fx("stress", "", 10)))),
		event("evt_container_sealed_cargo", "密封高价值货箱", "一个带独立锁控的货箱没有开启痕迹。", "node", "", "once_per_run",
			checked("unlock", []string{modeExploring}, "engineering", 58, "toolkit", 20, "完整解除货箱锁控。", "锁控触发警报，只能迅速离开。", effects(fx("container_pool", "sealed_reward", 2)), effects(fx("heat", "", 12), fx("time", "", 3)))),
		event("evt_container_elite_convoy", "精英车队经过", "一支车队正在穿越集装箱通道。", "node", "encounter", "once_per_run",
			styled(checked("conceal", nil, "stealth", 58, "", 0, "借助箱体阴影避开车队。", "掩体选择失误，被车队警戒发现。", effects(fx("skip_combat", "", 0)), effects(fx("encounter", "elite", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", nil, "利用集装箱通道卡住车队，主动发起攻击。", effects(fx("encounter", "elite", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
		event("evt_pier_signal_failure", "码头信标故障", "撤离信标无法完成自动握手。", "node", "", "once_per_run",
			checked("repair", []string{modeEvacuating}, "engineering", 52, "toolkit", 20, "恢复信标，接应按时抵达。", "只能手动发送坐标并等待确认。", effects(fx("time", "", 1)), effects(fx("time", "", 5), fx("heat", "", 5)))),
		event("evt_pier_final_ambush", "码头最后拦截", "接近撤离点时发现埋伏痕迹。", "node", "encounter", "once_per_run",
			styled(checked("counter", []string{modeEvacuating}, "perception", 60, "smoke", 15, "识破埋伏路线，从侧面进入码头。", "拦截者封锁了最后入口。", effects(fx("skip_combat", "", 0)), effects(fx("encounter", "extraction", 0))), []string{"balanced", "stealth"}, "bypass", 30),
			styled(automatic("engage", []string{modeEvacuating}, "不再寻找侧路，直接清除撤离点最后的拦截者。", effects(fx("encounter", "extraction", 0))), []string{"aggressive", "greedy"}, "engage", 30)),
	}
}

func eventBindings() []models.EventBinding {
	bindings := make([]models.EventBinding, 0, 48)
	add := func(eventID, scopeType, scopeID, phase string, triggerBP, weight, priority, maxPerRun, cooldown int) {
		bindings = append(bindings, models.EventBinding{
			ID: "bind_" + eventID, EventID: eventID, ScopeType: scopeType, ScopeID: scopeID,
			Phase: phase, TriggerBP: triggerBP, Weight: weight, Priority: priority,
			MaxPerRun: maxPerRun, CooldownNodes: cooldown, Enabled: true,
		})
	}

	add("evt_common_patrol", "global", "", "pre_encounter", 800, 100, 10, 2, 2)
	add("evt_elite_patrol", "global", "", "pre_encounter", 450, 35, 20, 1, 0)
	add("evt_enemy_ambush", "global", "", "pre_encounter", 550, 50, 15, 1, 0)
	add("evt_sniper_fire", "global", "", "pre_encounter", 350, 30, 20, 1, 0)
	add("evt_checkpoint", "global", "", "pre_encounter", 500, 45, 15, 1, 0)
	add("evt_reinforcements", "global", "", "pre_encounter", 350, 25, 20, 1, 0)
	add("evt_rival_scavengers", "global", "", "pre_encounter", 450, 40, 15, 1, 0)
	add("evt_cornered_guard", "global", "", "pre_encounter", 450, 40, 15, 1, 0)

	for _, entry := range []struct {
		id   string
		rate int
	}{
		{"evt_high_value_intel", 250}, {"evt_safehouse", 500}, {"evt_material_storage", 650},
		{"evt_medical_room", 450}, {"evt_maintenance_workshop", 450}, {"evt_sealed_cache", 400},
		{"evt_abandoned_camp", 450}, {"evt_hidden_stash", 500}, {"evt_server_room", 250},
		{"evt_survivor_cache", 400},
	} {
		add(entry.id, "global", "", "post_search", entry.rate, 50, 10, 1, 0)
	}

	for _, entry := range []struct {
		id   string
		rate int
	}{
		{"evt_accidental_injury", 550}, {"evt_structural_collapse", 400}, {"evt_toxic_leak", 400},
		{"evt_power_outage", 500}, {"evt_lost_route", 500}, {"evt_noise_exposure", 550},
		{"evt_flooded_path", 400}, {"evt_local_fire", 350},
	} {
		add(entry.id, "global", "", "enter_node", entry.rate, 50, 10, 1, 0)
	}

	add("evt_enemy_pursuit", "global", "", "evac_step", 900, 70, 20, 1, 0)
	add("evt_route_blockade", "global", "", "evac_step", 700, 55, 20, 1, 0)
	add("evt_injury_worsens", "global", "", "evac_step", 1600, 40, 20, 1, 0)
	add("evt_forced_discard", "global", "", "evac_step", 1200, 35, 20, 1, 0)
	add("evt_hidden_shortcut", "global", "", "evac_step", 700, 45, 15, 1, 0)
	add("evt_signal_disrupted", "global", "", "evac_step", 650, 45, 15, 1, 0)
	add("evt_extraction_ambush", "node_tag", "extraction", "pre_encounter", 1500, 70, 60, 1, 0)
	add("evt_smoke_corridor", "node_tag", "outdoor", "pre_encounter", 700, 60, 50, 1, 0)
	add("evt_field_treatment", "global", "", "evac_step", 750, 45, 15, 1, 0)
	add("evt_extraction_window", "node_tag", "extraction", "at_extraction", 1800, 50, 40, 1, 0)

	add("evt_apt_safehouse", "node", "node_apt", "post_search", 800, 100, 80, 1, 0)
	add("evt_apt_survivor_ambush", "node", "node_apt", "pre_encounter", 900, 100, 80, 1, 0)
	add("evt_warehouse_material", "node", "node_warehouse", "post_search", 800, 100, 80, 1, 0)
	add("evt_warehouse_shelf", "node", "node_warehouse", "enter_node", 1000, 100, 80, 1, 0)
	add("evt_customs_intel_vault", "node", "node_customs", "post_search", 500, 100, 90, 1, 0)
	add("evt_customs_lockdown", "node", "node_customs", "pre_encounter", 1200, 100, 90, 1, 0)
	add("evt_tunnel_drain", "node", "node_tunnel", "evac_step", 1600, 100, 85, 1, 0)
	add("evt_tunnel_gas", "node", "node_tunnel", "enter_node", 1400, 100, 85, 1, 0)
	add("evt_container_sealed_cargo", "node", "node_container", "post_search", 500, 100, 85, 1, 0)
	add("evt_container_elite_convoy", "node", "node_container", "pre_encounter", 1000, 100, 85, 1, 0)
	add("evt_pier_signal_failure", "node", "node_pier", "at_extraction", 2200, 100, 90, 1, 0)
	add("evt_pier_final_ambush", "node", "node_pier", "pre_encounter", 1800, 100, 100, 1, 0)
	return bindings
}

func encounterPools() []models.EncounterPoolEntry {
	return []models.EncounterPoolEntry{
		{ID: "city_patrol_1", MapID: "city_ruins", Role: "patrol", EnemyID: "enemy_patrol", Weight: 100},
		{ID: "city_guard_1", MapID: "city_ruins", Role: "guard", EnemyID: "enemy_guard", Weight: 80},
		{ID: "city_guard_2", MapID: "city_ruins", Role: "guard", EnemyID: "enemy_patrol", Weight: 20},
		{ID: "city_elite_1", MapID: "city_ruins", Role: "elite", EnemyID: "enemy_elite", Weight: 80},
		{ID: "city_elite_2", MapID: "city_ruins", Role: "elite", EnemyID: "enemy_guard", Weight: 20},
		{ID: "city_sniper_1", MapID: "city_ruins", Role: "sniper", EnemyID: "enemy_sniper", Weight: 100},
		{ID: "city_extract_1", MapID: "city_ruins", Role: "extraction", EnemyID: "enemy_guard", Weight: 60},
		{ID: "city_extract_2", MapID: "city_ruins", Role: "extraction", EnemyID: "enemy_elite", Weight: 40},
	}
}

func event(id, name, desc, category, group, repeat string, options ...models.EventOption) models.EventDef {
	return models.EventDef{ID: id, Name: name, Desc: desc, Category: category, Tags: []string{category}, ExclusiveGroup: group, RepeatPolicy: repeat, Options: options}
}

func automatic(id string, modes []string, text string, success []models.EventEffect) models.EventOption {
	return models.EventOption{ID: id, Modes: modes, Check: models.EventCheck{Type: "none"}, SuccessText: text, SuccessEffects: success}
}

func styled(option models.EventOption, styles []string, intent string, priority int) models.EventOption {
	option.Styles = styles
	option.Intent = intent
	option.Priority = priority
	return option
}

func fixed(id string, modes []string, target int, successText, failureText string, success, failure []models.EventEffect) models.EventOption {
	return models.EventOption{ID: id, Modes: modes, Check: models.EventCheck{Type: "fixed", Target: target}, SuccessText: successText, FailureText: failureText, SuccessEffects: success, FailureEffects: failure}
}

func checked(id string, modes []string, attribute string, target int, bonusRef string, bonus int, successText, failureText string, success, failure []models.EventEffect) models.EventOption {
	checkType := "attribute"
	if attribute == "" {
		checkType = "none"
	}
	return models.EventOption{ID: id, Modes: modes, Check: models.EventCheck{Type: checkType, Attribute: attribute, Target: target, ItemBonusRef: bonusRef, ItemBonus: bonus}, SuccessText: successText, FailureText: failureText, SuccessEffects: success, FailureEffects: failure}
}

func conditioned(id string, modes []string, condition models.EventCondition, attribute string, target int, bonusRef string, bonus int, successText, failureText string, success, failure []models.EventEffect) models.EventOption {
	option := checked(id, modes, attribute, target, bonusRef, bonus, successText, failureText, success, failure)
	option.Conditions = []models.EventCondition{condition}
	option.Priority = 10
	return option
}

func condition(kind, operator, ref string, value float64) models.EventCondition {
	return models.EventCondition{Type: kind, Operator: operator, Ref: ref, Value: value}
}

func effects(values ...models.EventEffect) []models.EventEffect {
	return values
}

func fx(kind, ref string, value int) models.EventEffect {
	return models.EventEffect{Type: kind, Ref: ref, Value: value}
}
