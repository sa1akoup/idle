// 携带弹药池回归测试：主弹选择优先级与各栈发数统计。
package engine

import "testing"

func TestSelectUsableAmmoStackPrefersHighestLevel(t *testing.T) {
	snapshot := ScenarioSnapshot{Ammos: map[string]Ammo{
		"n1": {ID: "n1", CaliberID: "c", Level: 1},
		"n2": {ID: "n2", CaliberID: "c", Level: 2},
		"n4": {ID: "n4", CaliberID: "c", Level: 4},
	}}
	weapon := Weapon{CaliberID: "c", AmmoPerRound: 3}
	stacks := []CarriedAmmo{
		{ID: "n1", CaliberID: "c", Level: 1, Rounds: 60},
		{ID: "n4", CaliberID: "c", Level: 4, Rounds: 60},
		{ID: "n2", CaliberID: "c", Level: 2, Rounds: 2}, // 发数不足一轮，跳过
	}

	profile, rounds, index, ok := selectUsableAmmoStack(snapshot, weapon, stacks)
	if !ok || profile.ID != "n4" || rounds != 60 || index != 1 {
		t.Fatalf("应优先选中 N4（发数足够且等级最高）：ok=%v profile=%+v rounds=%d index=%d", ok, profile, rounds, index)
	}

	// 耗尽或不足一轮的池应判定无可用弹药。
	depleted := []CarriedAmmo{{ID: "n4", CaliberID: "c", Level: 4, Rounds: 2}}
	if _, _, _, ok := selectUsableAmmoStack(snapshot, weapon, depleted); ok {
		t.Fatalf("全部栈发数不足一轮时应判定无可用弹药")
	}
}

func TestAmmoStacksRoundsAndSummary(t *testing.T) {
	stacks := []CarriedAmmo{
		{ID: "n4", Level: 4, Rounds: 30},
		{ID: "n2", Level: 2, Rounds: 0},
		{ID: "n2", Level: 2, Rounds: 20},
	}
	if got := ammoStacksRounds(stacks); got != 50 {
		t.Fatalf("池总发数 = %d，期望 50", got)
	}
	best := BestCarriedAmmoSummary(stacks)
	if best.ID != "n4" || best.Rounds != 30 {
		t.Fatalf("主弹摘要应为 N4/30，实际 %+v", best)
	}
	empty := BestCarriedAmmoSummary(nil)
	if empty.ID != "" {
		t.Fatalf("空池摘要应为空，实际 %+v", empty)
	}
}

func TestBestCarriedAmmoSummaryRetainsDepletedPreference(t *testing.T) {
	stack := CarriedAmmo{
		ID: "n4", CaliberID: "c", Level: 4, Rounds: 0,
		PreferredID: "n4", PreferredLevel: 4, TargetRounds: 60,
	}
	best := BestCarriedAmmoSummary([]CarriedAmmo{stack})
	if best.ID != "n4" || best.Rounds != 0 || best.PreferredID != "n4" || best.TargetRounds != 60 {
		t.Fatalf("耗尽栈仍应保留补弹偏好，实际 %+v", best)
	}
}

func TestHasUsableCarriedAmmoStackRejectsOnlyFragments(t *testing.T) {
	snapshot := ScenarioSnapshot{Ammos: map[string]Ammo{
		"n1": {ID: "n1", CaliberID: "c", Level: 1},
	}}
	weapon := Weapon{CaliberID: "c", AmmoPerRound: 3}
	if HasUsableCarriedAmmoStack(snapshot, weapon, []CarriedAmmo{
		{ID: "n1", CaliberID: "c", Level: 1, Rounds: 2},
		{ID: "n1", CaliberID: "c", Level: 1, Rounds: 2},
	}) {
		t.Fatal("多个不足一轮的弹药栈不能判定为可开火")
	}
}
