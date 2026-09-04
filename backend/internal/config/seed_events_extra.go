package config

import "idle/internal/models"

func extraEventDefinitions() []models.EventDef {
	return []models.EventDef{
		event("evt_loose_body", "倒地搜刮者", "一名搜刮者倒在掩体后，口袋还鼓着。", "exploration", "", "once_per_run",
			annotated(checked("search", []string{modeExploring}, "perception", 48, "", 0, "确认没有陷阱后搜出随身物资。", "翻动时发出声响，只能带走一点。", effects(fx("container_pool", "supply_reward", 0)), effects(fx("heat", "", 6), fx("time", "", 2))), "search", 2, 3)),
		event("evt_radio_chatter", "敌方电台", "附近频道里有人在通报你们的大致方向。", "exploration", "", "once_per_run",
			annotated(checked("jam", []string{modeExploring}, "engineering", 52, "toolkit", 15, "短暂干扰对方频道，降低暴露。", "没能压住呼叫，热度上升。", effects(fx("heat", "", -6)), effects(fx("heat", "", 10))), "secure", 2, 2)),
		event("evt_tripwire", "绊线", "走廊低处横着一根不太明显的细线。", "hazard", "hazard", "once_per_run",
			annotated(checked("spot", nil, "perception", 54, "", 0, "绕开绊线继续前进。", "绊线被带倒，附近响起动静。", effects(fx("time", "", 1)), effects(fx("heat", "", 10), fx("hp", "", -6))), "secure", 3, 1)),
		event("evt_unexploded", "未爆弹药", "墙角堆着一箱被挪动过的弹药。", "hazard", "hazard", "once_per_run",
			annotated(checked("bypass", nil, "survival", 52, "", 0, "沿稳妥路线绕过危险品。", "靠近时箱体倾倒，破片划伤护甲。", effects(fx("time", "", 2)), effects(fx("armor", "", -8), fx("stress", "", 6))), "secure", 4, 1)),
		event("evt_night_glint", "镜面反光", "远处有人用光学瞄具扫过通道。", "combat", "encounter", "once_per_run",
			styled(checked("drop", nil, "stealth", 52, "", 0, "贴墙降下身形，反光移开。", "被提前锁定，只能接战。", effects(fx("skip_combat", "", 0), fx("time", "", 1)), effects(fx("encounter", "sniper", 0))), []string{"balanced", "stealth"}, "bypass", 4, 2, 30),
			styled(automatic("engage", nil, "对着反光点位主动压制。", effects(fx("encounter", "sniper", 0))), []string{"aggressive", "greedy"}, "engage", 5, 2, 30)),
		event("evt_stray_dog", "野狗群", "几条瘦狗围着垃圾袋转，开始朝这边叫。", "combat", "encounter", "once_per_run",
			styled(checked("lure", nil, "survival", 48, "", 0, "甩开食物残渣引开它们。", "叫声引来附近巡逻。", effects(fx("skip_combat", "", 0), fx("time", "", 2)), effects(fx("encounter", "patrol", 0))), []string{"balanced", "stealth"}, "bypass", 3, 2, 30),
			styled(automatic("engage", nil, "清掉叫声源头，避免继续暴露。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "engage", 5, 2, 30)),

		event("evt_clinic_records", "诊所病历柜", "护士站抽屉里还夹着未归档的记录。", "node", "", "once_per_run",
			annotated(checked("skim", []string{modeExploring}, "intellect", 48, "", 0, "抽出有用的夹页和药品清单。", "字迹模糊，只摸到散页。", effects(fx("container_pool", "medical_reward", 0)), effects(fx("time", "", 2))), "intel", 1, 3)),
		event("evt_market_dog", "市场看门狗", "摊位后的铁笼里有一条被拴住的狗在狂吠。", "node", "encounter", "once_per_run",
			styled(checked("quiet", nil, "stealth", 50, "", 0, "绕开铁笼，吠声很快落下。", "吠声引来附近搜刮者。", effects(fx("skip_combat", "", 0)), effects(fx("encounter", "patrol", 0))), []string{"balanced", "stealth"}, "bypass", 3, 2, 30),
			styled(automatic("engage", nil, "先解决吠叫源头再继续搜摊。", effects(fx("encounter", "patrol", 0))), []string{"aggressive", "greedy"}, "engage", 5, 2, 30)),
		event("evt_gas_spill", "油料泄漏", "加油区地面有一层打滑的油膜。", "node", "hazard", "once_per_run",
			annotated(checked("cross", nil, "agility", 50, "", 0, "踩着干区过去。", "滑倒撞上油桶，护甲磕出凹痕。", effects(fx("time", "", 1)), effects(fx("armor", "", -6), fx("time", "", 2))), "secure", 3, 1)),
		event("evt_customs_camera", "海关摄像头", "走廊尽头的摄像头还在缓慢转动。", "node", "", "once_per_run",
			annotated(checked("blind", []string{modeExploring}, "engineering", 52, "toolkit", 15, "切断供电，警戒没有响起。", "镜头捕捉到移动，热度上升。", effects(fx("heat", "", -4)), effects(fx("heat", "", 8))), "unlock", 2, 2)),
		event("evt_warehouse_forklift", "失控叉车", "一辆没刹住的叉车顺着坡道滑来。", "node", "hazard", "once_per_run",
			annotated(checked("dodge", nil, "agility", 54, "", 0, "侧身让过叉车。", "擦到货叉，护甲被刮开。", nil, effects(fx("hp", "", -8), fx("armor", "", -6))), "secure", 4, 1)),

		event("evt_factory_press", "冲压机余震", "停机的冲压位突然落下一次空行程。", "node", "hazard", "once_per_run",
			annotated(checked("step", nil, "perception", 50, "", 0, "提前离开危险区。", "被气流和碎屑打到。", effects(fx("time", "", 1)), effects(fx("hp", "", -8), fx("stress", "", 6))), "secure", 4, 1)),
		event("evt_factory_saw", "伐木场链锯", "有人把链锯丢在木堆上，电机还在空转。", "node", "", "once_per_run",
			annotated(checked("kill", []string{modeExploring}, "engineering", 48, "", 0, "切断电源并翻到工具箱。", "噪声持续，只能草草搜一下。", effects(fx("container_pool", "workshop_reward", 0), fx("heat", "", -3)), effects(fx("heat", "", 7), fx("time", "", 2))), "search", 2, 3)),
		event("evt_factory_fumes", "油料棚汽雾", "棚内汽油味浓到发呛。", "node", "hazard", "once_per_run",
			annotated(checked("hold", nil, "resist", 52, "", 0, "屏息快速搜完油桶区。", "吸入汽雾，状态下滑。", effects(fx("container_pool", "fuel_reward", 0)), effects(fx("hp", "", -8), fx("stress", "", 8))), "search", 3, 2)),
		event("evt_factory_office_safe", "车间保险柜", "办公室墙柜没有完全锁死。", "node", "", "once_per_run",
			annotated(checked("crack", []string{modeExploring}, "engineering", 56, "toolkit", 15, "撬开柜门，摸到高价值零件。", "柜舌卡住，只带出散件。", effects(fx("container_pool", "sealed_reward", 0)), effects(fx("time", "", 3))), "unlock", 2, 4)),
	}
}

func extraEventBindings() []models.EventBinding {
	bindings := make([]models.EventBinding, 0, 16)
	add := func(eventID, scopeType, scopeID, phase string, triggerBP, weight, priority, maxPerRun, cooldown int) {
		bindings = append(bindings, models.EventBinding{
			ID: "bind_" + eventID, EventID: eventID, ScopeType: scopeType, ScopeID: scopeID,
			Phase: phase, TriggerBP: triggerBP, Weight: weight, Priority: priority,
			MaxPerRun: maxPerRun, CooldownNodes: cooldown, Enabled: true,
		})
	}
	add("evt_loose_body", "global", "", "post_search", 400, 45, 10, 1, 0)
	add("evt_radio_chatter", "global", "", "enter_node", 400, 40, 12, 1, 0)
	add("evt_tripwire", "global", "", "enter_node", 350, 40, 12, 1, 0)
	add("evt_unexploded", "global", "", "enter_node", 300, 30, 12, 1, 0)
	add("evt_night_glint", "global", "", "pre_encounter", 320, 30, 18, 1, 0)
	add("evt_stray_dog", "global", "", "pre_encounter", 380, 35, 12, 1, 0)
	add("evt_clinic_records", "node", "city_ruins_node_7", "post_search", 700, 90, 75, 1, 0)
	add("evt_market_dog", "node", "city_ruins_node_3", "pre_encounter", 700, 90, 75, 1, 0)
	add("evt_gas_spill", "node", "city_ruins_node_9", "enter_node", 800, 90, 75, 1, 0)
	add("evt_customs_camera", "node", "city_ruins_node_8", "enter_node", 700, 90, 75, 1, 0)
	add("evt_warehouse_forklift", "node", "city_ruins_node_2", "enter_node", 650, 90, 75, 1, 0)
	add("evt_factory_press", "node", "factory_woods_shop", "enter_node", 800, 90, 75, 1, 0)
	add("evt_factory_saw", "node", "factory_woods_lumber", "post_search", 750, 90, 75, 1, 0)
	add("evt_factory_fumes", "node", "factory_woods_fuel", "enter_node", 800, 90, 75, 1, 0)
	add("evt_factory_office_safe", "node", "factory_woods_office", "post_search", 600, 90, 75, 1, 0)
	return bindings
}
