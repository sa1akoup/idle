// 纯探索引擎回放测试：验证相同快照、输入和 seed 始终得到相同结果。
package engine

import (
	"reflect"
	"testing"
)

func TestSimulateRunIsReplayable(t *testing.T) {
	snapshot := replayTestSnapshot()
	if err := ValidateSnapshot(snapshot); err != nil {
		t.Fatalf("测试快照无效: %v", err)
	}
	input := RunInput{
		SessionSeed: 20260824,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "回放角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, Stress: 10},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	}

	first, err := SimulateRun(snapshot, input)
	if err != nil {
		t.Fatalf("首次运行失败: %v", err)
	}
	second, err := SimulateRun(snapshot, input)
	if err != nil {
		t.Fatalf("回放运行失败: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("相同输入的运行结果不一致:\n首次=%+v\n回放=%+v", first, second)
	}
}

func TestSimulateRunGolden(t *testing.T) {
	result, err := SimulateRun(replayTestSnapshot(), RunInput{
		SessionSeed: 20260824,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "回放角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, Stress: 10},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	})
	if err != nil {
		t.Fatalf("golden 运行失败: %v", err)
	}
	want := struct {
		Result          string
		DurationSec     int64
		Heat            int
		AmmoUsed        int
		Injury          string
		Stress          int
		PistolProf      int
		ArmorDurability int
		CarryUsedSlots  int
		CarryUsedWeight float64
	}{
		Result: "success", DurationSec: 240, Heat: 0, AmmoUsed: 0, Injury: "none",
		Stress: 0, PistolProf: 1, ArmorDurability: 100, CarryUsedSlots: 2, CarryUsedWeight: 10,
	}
	got := struct {
		Result          string
		DurationSec     int64
		Heat            int
		AmmoUsed        int
		Injury          string
		Stress          int
		PistolProf      int
		ArmorDurability int
		CarryUsedSlots  int
		CarryUsedWeight float64
	}{
		Result: result.Result, DurationSec: result.DurationSec, Heat: result.Heat, AmmoUsed: result.AmmoUsed,
		Injury: result.Injury, Stress: result.NextState.Character.Stress, PistolProf: result.NextState.Character.PistolProf,
		ArmorDurability: result.NextState.ArmorDurability, CarryUsedSlots: result.NextState.Carry.UsedSlots,
		CarryUsedWeight: result.NextState.Carry.UsedWeight,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("golden 结果变化:\n期望=%+v\n实际=%+v", want, got)
	}
}

func TestNodeTravelReducesStressIncludingExtraction(t *testing.T) {
	result, err := SimulateRun(replayTestSnapshot(), RunInput{
		SessionSeed: 20260824,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, Stress: 10},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	})
	if err != nil {
		t.Fatalf("模拟节点移动失败: %v", err)
	}
	if result.DurationSec != 240 {
		t.Fatalf("节点移动时间 = %d，期望 240 秒", result.DurationSec)
	}
	if result.NextState.Character.Stress != 0 {
		t.Fatalf("包含撤离点在内的节点移动后压力 = %d，期望 0", result.NextState.Character.Stress)
	}
}

func TestShortcutReducesStressRecoveryByActualDuration(t *testing.T) {
	result, err := SimulateRun(shortcutStressSnapshot(1), shortcutStressInput(40))
	if err != nil {
		t.Fatalf("带捷径的压力恢复模拟失败: %v", err)
	}
	if result.DurationSec != 720 {
		t.Fatalf("捷径后的实际耗时 = %d 秒，期望 720 秒", result.DurationSec)
	}
	if result.NextState.Character.Stress != 0 {
		t.Fatalf("按实际耗时恢复后的压力 = %d，期望 0", result.NextState.Character.Stress)
	}
}

func TestZeroDurationShortcutDoesNotRecoverStress(t *testing.T) {
	result, err := SimulateRun(shortcutStressSnapshot(2), shortcutStressInput(40))
	if err != nil {
		t.Fatalf("零耗时捷径模拟失败: %v", err)
	}
	if result.DurationSec != 660 {
		t.Fatalf("零耗时捷径后的实际耗时 = %d 秒，期望 660 秒", result.DurationSec)
	}
	if result.NextState.Character.Stress != 0 {
		t.Fatalf("零耗时节点后的压力 = %d，期望 0", result.NextState.Character.Stress)
	}
}

func TestShortcutOnlyAffectsNextNodeDurationAndStress(t *testing.T) {
	result, err := SimulateRun(shortcutStressSnapshot(1), shortcutStressInput(40))
	if err != nil {
		t.Fatalf("捷径作用范围模拟失败: %v", err)
	}
	if result.DurationSec != 720 {
		t.Fatalf("捷径影响了后续节点耗时，实际耗时 = %d 秒，期望 720 秒", result.DurationSec)
	}
	if result.NextState.Character.Stress != 0 {
		t.Fatalf("捷径后的压力 = %d，期望 0", result.NextState.Character.Stress)
	}
}

func TestSnapshotHashIsOrderIndependent(t *testing.T) {
	left := replayTestSnapshot()
	right := replayTestSnapshot()
	right.Nodes[0], right.Nodes[1] = right.Nodes[1], right.Nodes[0]
	right.Styles[0], right.Styles[1] = right.Styles[1], right.Styles[0]
	leftHash, err := SnapshotHash(left)
	if err != nil {
		t.Fatalf("计算基准快照 hash 失败: %v", err)
	}
	rightHash, err := SnapshotHash(right)
	if err != nil {
		t.Fatalf("计算重排快照 hash 失败: %v", err)
	}
	if leftHash != rightHash {
		t.Fatalf("等价快照 hash 不一致: %s != %s", leftHash, rightHash)
	}
}

func replayTestSnapshot() ScenarioSnapshot {
	return ScenarioSnapshot{
		SchemaVersion: SchemaVersion,
		Map:           Map{ID: "map_test", Name: "回放地图", StartNodeID: "node_start", LayoutColumns: 2, LayoutRows: 1},
		Nodes: []Node{
			{ID: "node_start", MapID: "map_test", Name: "起点", PositionX: 0, PositionY: 0, ExploreTime: 1},
			{ID: "node_extract", MapID: "map_test", Name: "撤离锚点", PositionX: 1, PositionY: 0, ExploreTime: 1},
		},
		Edges:            []MapEdge{{ID: 1, MapID: "map_test", FromNodeID: "node_start", ToNodeID: "node_extract", MoveTime: 1, Bidirectional: true}},
		ExtractionPoints: []ExtractionPoint{{ID: "extract_test", MapID: "map_test", Name: "测试撤离点", Kind: "normal", AnchorNodeID: "node_extract", TravelTime: 1, Enabled: true}},
		Items: map[string]ItemDefinition{
			"weapon_test": {ID: "weapon_test", Kind: "weapon", Name: "测试武器", Weight: 5, Slots: 1},
			"armor_test":  {ID: "armor_test", Kind: "armor", Name: "测试护甲", Weight: 5, Slots: 1},
		},
		Weapons: map[string]Weapon{
			"weapon_test": {ID: "weapon_test", Name: "测试武器", Category: "pistol", Hit: 50, Damage: 10, AmmoPerRound: 0},
		},
		Armors: map[string]Armor{
			"armor_test": {ID: "armor_test", Name: "测试护甲", ProtectionLevel: 2, MaxDurability: 100},
		},
		Events: EventCatalog{Definitions: map[string]EventDefinition{}, EncounterPools: map[string][]EncounterPoolEntry{}},
		Styles: DefaultStylePolicies(),
	}
}

func shortcutStressSnapshot(shortcutMinutes int) ScenarioSnapshot {
	snapshot := replayTestSnapshot()
	snapshot.Nodes = []Node{
		{ID: "node_start", MapID: "map_test", Name: "起点", PositionX: 0, PositionY: 0, ExploreTime: 2},
		{ID: "node_middle", MapID: "map_test", Name: "中段", PositionX: 1, PositionY: 0, ExploreTime: 2},
		{ID: "node_extract", MapID: "map_test", Name: "撤离锚点", PositionX: 2, PositionY: 0, ExploreTime: 2},
	}
	snapshot.Map.LayoutColumns = 3
	snapshot.Map.LayoutRows = 1
	snapshot.Edges = []MapEdge{
		{ID: 1, MapID: "map_test", FromNodeID: "node_start", ToNodeID: "node_middle", MoveTime: 3, Bidirectional: true},
		{ID: 2, MapID: "map_test", FromNodeID: "node_middle", ToNodeID: "node_extract", MoveTime: 3, Bidirectional: true},
	}
	snapshot.ExtractionPoints = []ExtractionPoint{{ID: "extract_test", MapID: "map_test", Name: "测试撤离点", Kind: "normal", AnchorNodeID: "node_extract", TravelTime: 1, Enabled: true}}
	snapshot.Events = EventCatalog{
		Definitions: map[string]EventDefinition{
			"shortcut_test": {
				ID: "shortcut_test", Name: "测试捷径", RepeatPolicy: "once_per_run",
				Options: []EventOption{{
					ID: "take_shortcut", Modes: []string{"exploring"}, Check: EventCheck{Type: "none"},
					SuccessEffects: []EventEffect{{Type: "evac_shortcut", Value: shortcutMinutes}},
				}},
			},
		},
		Bindings: []EventBinding{{
			ID: "shortcut_test_binding", EventID: "shortcut_test", ScopeType: "node", ScopeID: "node_start",
			Phase: "enter_node", TriggerBP: 10000, Weight: 1, Enabled: true,
		}},
		EncounterPools: map[string][]EncounterPoolEntry{},
	}
	return snapshot
}

func shortcutStressInput(stress int) RunInput {
	return RunInput{
		SessionSeed: 20260825,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "捷径测试角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, Stress: stress},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	}
}
