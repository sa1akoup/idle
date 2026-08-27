// 撤离阶段测试：验证撤离点作用域事件能够在抵达后进入真实战斗流程。
package engine

import "testing"

func TestExtractionEncounterIsResolvedAfterArrival(t *testing.T) {
	snapshot := replayTestSnapshot()
	snapshot.Weapons["weapon_test"] = Weapon{
		ID: "weapon_test", Name: "测试近战武器", Category: "melee", Hit: 100, Damage: 100,
	}
	snapshot.Armors["armor_test"] = Armor{ID: "armor_test", Name: "测试护甲", ProtectionLevel: 1, MaxDurability: 100}
	snapshot.Enemies = map[string]Enemy{
		"enemy_extract": {
			ID: "enemy_extract", Name: "撤离点伏兵", HP: 1, StressThreshold: 100,
			WeaponID: "weapon_test", ArmorID: "armor_test", Evasion: 0, Mobility: 0,
		},
	}
	snapshot.Events = EventCatalog{
		Definitions: map[string]EventDefinition{
			"extract_ambush_test": {
				ID: "extract_ambush_test", Name: "撤离点伏兵", Category: "evacuation",
				ExclusiveGroup: "encounter", RepeatPolicy: "once_per_run",
				Options: []EventOption{{
					ID: "engage", Modes: []string{runModeEvacuating}, Check: EventCheck{Type: "none"},
					SuccessEffects: []EventEffect{{Type: "encounter", Ref: "extraction"}},
				}},
			},
		},
		Bindings: []EventBinding{{
			ID: "extract_ambush_test_binding", EventID: "extract_ambush_test", ScopeType: "extraction",
			ScopeID: "extract_test", Phase: eventPhaseExtractionPointReached, TriggerBP: 10000, Weight: 1, Enabled: true,
		}},
		EncounterPools: map[string][]EncounterPoolEntry{
			"extraction": {{ID: "extract_pool_test", MapID: "map_test", Role: "extraction", EnemyID: "enemy_extract", Weight: 1}},
		},
	}

	result, err := SimulateRun(snapshot, RunInput{
		SessionSeed: 20260825,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "撤离测试角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	})
	if err != nil {
		t.Fatalf("撤离点遭遇模拟失败: %v", err)
	}
	if result.Result != "success" {
		t.Fatalf("撤离点遭遇后的结果 = %s，期望 success", result.Result)
	}

	battleStarted := false
	extractionCompleted := false
	for _, event := range result.Trace {
		if event.Type == TraceBattleStarted && event.NodeID == "node_extract" {
			battleStarted = true
		}
		if event.Type == TraceExtractionCompleted {
			extractionCompleted = true
		}
	}
	if !battleStarted {
		t.Fatal("撤离点遭遇事件未产生真实战斗事件")
	}
	if !extractionCompleted {
		t.Fatal("撤离点战斗后未产生撤离完成事件")
	}
}
