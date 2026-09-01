// 生存/医疗属性接线测试：生存降低能量饮水消耗、医疗降低自动医疗触发阈值。
package engine

import (
	"testing"
)

func TestSurvivalAttributeLowersNeedDrain(t *testing.T) {
	tuning := DefaultTuning()
	// 1 小时：生存 0 -> 能量 -8、饮水 -10；生存 100 -> 能量 -6、饮水 -6（10-4=6，触底 5 不下）。
	base := CharacterState{Energy: 100, Hydration: 100, Survival: 0}
	applyNeedDrain(tuning, &base, 3600)
	if base.Energy != 92 || base.Hydration != 90 {
		t.Fatalf("生存0消耗异常: energy=%.1f hydration=%.1f", base.Energy, base.Hydration)
	}
	survive := CharacterState{Energy: 100, Hydration: 100, Survival: 100}
	applyNeedDrain(tuning, &survive, 3600)
	if survive.Energy != 94 || survive.Hydration != 94 {
		t.Fatalf("生存100消耗异常: energy=%.1f hydration=%.1f", survive.Energy, survive.Hydration)
	}
}

func TestMedicalLowersAutoHealTrigger(t *testing.T) {
	tuning := DefaultTuning()
	tuning.Survival.AutoHealTrigger = 0.6
	// 同一血量 50%（0.5）：医疗 0 -> 触发（0.5 < 0.6）；医疗 100 -> 不触发（0.5 >= 0.4）。
	healItem := ItemUseDefinition{HPRecovery: 20, UsableInSession: true, UsePriority: 1}
	makeState := func(medical int) *eventRunState {
		lines := []string{}
		return &eventRunState{
			Tuning:         tuning,
			Character:      &CharacterState{Medical: medical, Name: "测试角色"},
			Player:         &BattleActor{Name: "玩家", MaxHP: 100, HP: 50, StressThreshold: 100},
			CarriedItems:   []CarriedItem{{ItemID: "med_test", Quantity: 2}},
			ItemUseDefs:    map[string]ItemUseDefinition{"med_test": healItem},
			ConsumedItems:  make(map[string]int),
			AvailableItems: map[string]int{"med_test": 2},
			Lines:          &lines,
		}
	}
	triggerLow := makeState(0)
	maybeAutoHeal(triggerLow)
	if triggerLow.Player.HP != 90 {
		t.Fatalf("医疗0应连续自动医疗至目标附近: hp=%.1f", triggerLow.Player.HP)
	}
	if triggerLow.ConsumedItems["med_test"] != 2 || len(triggerLow.CarriedItems) != 0 {
		t.Fatalf("自动医疗消耗结算异常: consumed=%d carried=%+v", triggerLow.ConsumedItems["med_test"], triggerLow.CarriedItems)
	}
	triggerHigh := makeState(100)
	maybeAutoHeal(triggerHigh)
	if triggerHigh.Player.HP != 50 {
		t.Fatalf("医疗100不应在50%%血量触发自动医疗: hp=%.1f", triggerHigh.Player.HP)
	}
}

func TestAutoRecoverNeedsConsumesStackUntilThreshold(t *testing.T) {
	tuning := DefaultTuning()
	lines := []string{}
	state := &eventRunState{
		Tuning:       tuning,
		Character:    &CharacterState{Energy: 30, Hydration: 100},
		CarriedItems: []CarriedItem{{ItemID: "food_test", Quantity: 3}},
		ItemUseDefs: map[string]ItemUseDefinition{
			"food_test": {EnergyRecovery: 25, UsableInSession: true, UsePriority: 1},
		},
		ConsumedItems:  make(map[string]int),
		AvailableItems: map[string]int{"food_test": 3},
		Lines:          &lines,
	}

	maybeAutoRecoverNeeds(state)

	if state.Character.Energy != 80 || state.Character.Hydration != 100 {
		t.Fatalf("自动恢复应补到阈值: energy=%.1f hydration=%.1f", state.Character.Energy, state.Character.Hydration)
	}
	if state.ConsumedItems["food_test"] != 2 || state.AvailableItems["food_test"] != 1 || len(state.CarriedItems) != 1 || state.CarriedItems[0].Quantity != 1 {
		t.Fatalf("自动恢复堆叠消耗异常: consumed=%d available=%d carried=%+v", state.ConsumedItems["food_test"], state.AvailableItems["food_test"], state.CarriedItems)
	}
}

func TestAutoRecoverNeedsSkipsNegativeRecoveryThatBreaksThreshold(t *testing.T) {
	tuning := DefaultTuning()
	state := &eventRunState{
		Tuning:       tuning,
		Character:    &CharacterState{Energy: 70, Hydration: 70},
		CarriedItems: []CarriedItem{{ItemID: "bad_food", Quantity: 1}},
		ItemUseDefs: map[string]ItemUseDefinition{
			"bad_food": {EnergyRecovery: -10, HydrationRecovery: 20, UsableInSession: true, UsePriority: 1},
		},
		ConsumedItems:  make(map[string]int),
		AvailableItems: map[string]int{"bad_food": 1},
	}

	maybeAutoRecoverNeeds(state)

	if state.Character.Energy != 70 || state.Character.Hydration != 70 || state.ConsumedItems["bad_food"] != 0 {
		t.Fatalf("负恢复物品不应破坏阈值: character=%+v consumed=%d", *state.Character, state.ConsumedItems["bad_food"])
	}
}
