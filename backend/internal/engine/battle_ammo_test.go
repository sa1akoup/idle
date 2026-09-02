// 分级弹药战斗测试：覆盖穿透/未穿透与护甲区/无甲四肢区四种核心组合。
package engine

import (
	"math"
	"math/rand"
	"testing"
)

func TestAttackUsesAmmoLevelAndHitLocation(t *testing.T) {
	tests := []struct {
		name            string
		ammoLevel       int
		coverage        int
		wantPenetrated  bool
		wantLocation    string
		wantRetention   float64
		wantArmorDamage bool
	}{
		{name: "高等级弹穿透护甲", ammoLevel: 5, coverage: 100, wantPenetrated: true, wantLocation: "armor", wantRetention: 0.90, wantArmorDamage: true},
		{name: "高等级弹命中无甲四肢", ammoLevel: 5, coverage: 0, wantPenetrated: true, wantLocation: "limb", wantRetention: 1.00},
		{name: "低等级弹未穿透护甲", ammoLevel: 2, coverage: 100, wantPenetrated: false, wantLocation: "armor", wantRetention: 0.08, wantArmorDamage: true},
		{name: "低等级弹命中无甲四肢", ammoLevel: 2, coverage: 0, wantPenetrated: true, wantLocation: "limb", wantRetention: 1.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attacker := BattleActor{
				Name: "攻击方", Weapon: Weapon{ID: "rifle", Hit: 100, Damage: 40, AmmoPerRound: 3}, WeaponControl: 50,
				Ammo: Ammo{ID: "ammo", Level: tt.ammoLevel, FleshDamageMultiplier: 1, ArmorDamageMultiplier: 1}, AmmoRounds: 30,
			}
			defender := BattleActor{
				Name: "目标", MaxHP: 1000, HP: 1000, StressThreshold: 1000, ResistEff: 50,
				Armor:           Armor{ID: "armor", ProtectionLevel: 4, Coverage: tt.coverage, MaxDurability: 1000},
				ArmorDurability: 1000, ArmorMaxDur: 1000,
			}
			lines := []string{}
			result := attack(DefaultTuning(), &attacker, &defender, 0, 0, rand.New(rand.NewSource(1)), &lines)

			if !result.Hit {
				t.Fatal("固定 RNG 下攻击应命中")
			}
			if result.HitLocation != tt.wantLocation {
				t.Fatalf("命中部位 = %s，期望 %s", result.HitLocation, tt.wantLocation)
			}
			if result.Penetrated != tt.wantPenetrated {
				t.Fatalf("穿透结果 = %t，期望 %t", result.Penetrated, tt.wantPenetrated)
			}
			if result.AmmoLevel != tt.ammoLevel || result.AmmoSpent != 3 || attacker.AmmoRounds != 27 {
				t.Fatalf("弹药结算异常: %+v，剩余 %d", result, attacker.AmmoRounds)
			}
			retention := result.HealthDamage / result.FleshDamage
			if math.Abs(retention-tt.wantRetention) > 0.0001 {
				t.Fatalf("生命伤害保留率 = %.4f，期望 %.4f", retention, tt.wantRetention)
			}
			if (result.ArmorDamage > 0) != tt.wantArmorDamage {
				t.Fatalf("护甲伤害 = %.2f，期望存在=%t", result.ArmorDamage, tt.wantArmorDamage)
			}
		})
	}
}

func TestEffectiveArmorLevelFollowsDurabilityBands(t *testing.T) {
	armor := Armor{ProtectionLevel: 5}
	tests := []struct {
		durability float64
		want       int
	}{
		{durability: 100, want: 5},
		{durability: 50, want: 4},
		{durability: 25, want: 3},
		{durability: 0, want: 0},
	}
	for _, tt := range tests {
		if got := effectiveArmorLevel(DefaultTuning(), armor, tt.durability, 100); got != tt.want {
			t.Fatalf("耐久 %.0f 时有效护甲 = A%d，期望 A%d", tt.durability, got, tt.want)
		}
	}
}

func TestAmmoEventUsesActualRoundDelta(t *testing.T) {
	state := eventRunState{Player: &BattleActor{AmmoRounds: 3}}
	if _, err := applyEventEffect(EventEffect{Type: "ammo", Value: -5}, &state); err != nil {
		t.Fatalf("执行弹药消耗事件: %v", err)
	}
	if state.Player.AmmoRounds != 0 || state.AmmoUsed != 3 {
		t.Fatalf("弹药消耗结算异常: rounds=%d used=%d", state.Player.AmmoRounds, state.AmmoUsed)
	}
	if _, err := applyEventEffect(EventEffect{Type: "ammo", Value: 4}, &state); err != nil {
		t.Fatalf("执行弹药补充事件: %v", err)
	}
	if state.Player.AmmoRounds != 4 || state.AmmoUsed != 3 {
		t.Fatalf("弹药补充结算异常: rounds=%d used=%d", state.Player.AmmoRounds, state.AmmoUsed)
	}
}

func TestAmmoEventUpdatesCarriedAmmoStack(t *testing.T) {
	snapshot := ScenarioSnapshot{
		Ammos: map[string]Ammo{
			"ammo": {ID: "ammo", CaliberID: "c", Level: 2, FleshDamageMultiplier: 1, ArmorDamageMultiplier: 1},
		},
	}
	state := eventRunState{
		Snapshot: &snapshot,
		Player: &BattleActor{Weapon: Weapon{CaliberID: "c", AmmoPerRound: 3}},
		AmmoStacks: []CarriedAmmo{{ID: "ammo", CaliberID: "c", Level: 2, Rounds: 10, PreferredID: "ammo", PreferredLevel: 2, TargetRounds: 30}},
	}
	if _, err := applyEventEffect(EventEffect{Type: "ammo", Value: -4}, &state); err != nil {
		t.Fatalf("执行弹药池消耗事件: %v", err)
	}
	if state.AmmoStacks[0].Rounds != 6 || state.Player.AmmoRounds != 6 || state.AmmoUsed != 4 {
		t.Fatalf("弹药池消耗未同步: stacks=%+v player=%d used=%d", state.AmmoStacks, state.Player.AmmoRounds, state.AmmoUsed)
	}
	if _, err := applyEventEffect(EventEffect{Type: "ammo", Value: 5}, &state); err != nil {
		t.Fatalf("执行弹药池补充事件: %v", err)
	}
	if state.AmmoStacks[0].Rounds != 11 || state.Player.AmmoRounds != 11 {
		t.Fatalf("弹药池补充未同步: stacks=%+v player=%d", state.AmmoStacks, state.Player.AmmoRounds)
	}
}

func TestAmmoEventConditionUsesPoolTotal(t *testing.T) {
	state := eventRunState{
		Player:     &BattleActor{AmmoRounds: 1},
		AmmoStacks: []CarriedAmmo{{ID: "ammo-a", Rounds: 2}, {ID: "ammo-b", Rounds: 4}},
	}
	if !eventConditionMatches(EventCondition{Type: "ammo", Operator: "eq", Value: 6}, &state) {
		t.Fatal("弹药条件应按弹药池总发数判定")
	}
}
