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
			Character:       CharacterState{Name: "回放角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100, Stress: 10},
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
			Character:       CharacterState{Name: "回放角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100, Stress: 10},
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
		Stress          int
		PistolProf      int
		ArmorDurability int
		CarryUsedSlots  int
		CarryUsedWeight float64
	}{
		Result: "success", DurationSec: 240, Heat: 0, AmmoUsed: 0,
		Stress: 0, PistolProf: 1, ArmorDurability: 100, CarryUsedSlots: 2, CarryUsedWeight: 10,
	}
	got := struct {
		Result          string
		DurationSec     int64
		Heat            int
		AmmoUsed        int
		Stress          int
		PistolProf      int
		ArmorDurability int
		CarryUsedSlots  int
		CarryUsedWeight float64
	}{
		Result: result.Result, DurationSec: result.DurationSec, Heat: result.Heat, AmmoUsed: result.AmmoUsed,
		Stress: result.NextState.Character.Stress, PistolProf: result.NextState.Character.PistolProf,
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
			Character:       CharacterState{Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100, Stress: 10},
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

// v6 出生点随机化后，捷径机制测试改用"单出生候选"夹具（出生节点是唯一非锚点），
// 保证捷径必定在出生节点触发；断言聚焦在下一段移动的缩短量与 0 耗时下限。
func TestShortcutReducesNextMoveByValue(t *testing.T) {
	result, err := SimulateRun(shortcutStressSnapshot(1), shortcutStressInput(10))
	if err != nil {
		t.Fatalf("带捷径的模拟失败: %v", err)
	}
	// 节点探索 120s + 移动 180s-60s + 锚点探索 120s + 撤离读取 60s
	if result.DurationSec != 420 {
		t.Fatalf("捷径(1分钟)后的实际耗时 = %d 秒，期望 420 秒", result.DurationSec)
	}
}

func TestShortcutClampsNextMoveToZero(t *testing.T) {
	result, err := SimulateRun(shortcutStressSnapshot(40), shortcutStressInput(10))
	if err != nil {
		t.Fatalf("大步长捷径模拟失败: %v", err)
	}
	// 缩短量超过下一段移动 180s，被钳制到 0：120 + 0 + 120 + 60
	if result.DurationSec != 300 {
		t.Fatalf("大步长捷径后的实际耗时 = %d 秒，期望 300 秒", result.DurationSec)
	}
}

func TestShortcutOnlyAffectsNextMove(t *testing.T) {
	short, err := SimulateRun(shortcutStressSnapshot(40), shortcutStressInput(10))
	if err != nil {
		t.Fatalf("大步长捷径模拟失败: %v", err)
	}
	full, err := SimulateRun(shortcutStressSnapshot(1), shortcutStressInput(10))
	if err != nil {
		t.Fatalf("小步长捷径模拟失败: %v", err)
	}
	// 捷径只作用于下一段移动：两个不同步长之间的差值不得超过单段移动 180s。
	if full.DurationSec-short.DurationSec > 180 {
		t.Fatalf("捷径影响了后续节点耗时: %d vs %d", full.DurationSec, short.DurationSec)
	}
	if full.DurationSec <= short.DurationSec {
		t.Fatalf("更长短步长应耗时更长: %d vs %d", full.DurationSec, short.DurationSec)
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
		{ID: "node_extract", MapID: "map_test", Name: "撤离锚点", PositionX: 1, PositionY: 0, ExploreTime: 2},
	}
	snapshot.Map.LayoutColumns = 2
	snapshot.Map.LayoutRows = 1
	snapshot.Edges = []MapEdge{
		{ID: 1, MapID: "map_test", FromNodeID: "node_start", ToNodeID: "node_extract", MoveTime: 3, Bidirectional: true},
	}
	snapshot.ExtractionPoints = []ExtractionPoint{{ID: "extract_test", MapID: "map_test", Name: "测试撤离点", Kind: "normal", AnchorNodeID: "node_extract", TravelTime: 1, Enabled: true}}
	snapshot.Events = EventCatalog{
		Definitions: map[string]EventDefinition{
			"shortcut_test": {
				ID: "shortcut_test", Name: "测试捷径", RepeatPolicy: "once_per_run",
				Options: []EventOption{{
					ID: "take_shortcut", Modes: []string{"exploring"}, Intent: "reroute", RiskTier: 1, ValueTier: 2, Check: EventCheck{Type: "none"},
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
			Character:       CharacterState{Name: "捷径测试角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100, Stress: stress},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	}
}
