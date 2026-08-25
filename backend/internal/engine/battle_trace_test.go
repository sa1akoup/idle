// 战斗时间线回归测试：验证攻击、回合、脱离和结束事件的 OffsetSec 不会倒退。
package engine

import (
	"math/rand"
	"testing"
)

func TestBattleTraceOffsetsIncrease(t *testing.T) {
	weapon := Weapon{ID: "weapon_test", Name: "测试武器", Hit: 80, Damage: 12, Suppress: 10, AmmoPerRound: 1, CloseMod: 10, MidMod: 5}
	armor := Armor{ID: "armor_test", MaxDurability: 100}
	player := &BattleActor{
		Name: "玩家", MaxHP: 100, HP: 100, StressThreshold: 100, Weapon: weapon, Armor: armor,
		ArmorDurability: 100, ArmorMaxDur: 100, Evasion: 20, Mobility: 50, PerceptionEff: 60,
		StealthEff: 50, Agility: 60, ResistEff: 50, WeaponControl: 60, AmmoRounds: 10,
	}
	enemy := &BattleActor{
		Name: "敌人", MaxHP: 100, HP: 100, StressThreshold: 100, Weapon: weapon, Armor: armor,
		ArmorDurability: 100, ArmorMaxDur: 100, Evasion: 20, Mobility: 50, PerceptionEff: 50,
		StealthEff: 40, Agility: 50, ResistEff: 40, WeaponControl: 50, AmmoRounds: 10,
	}
	policy := DefaultStylePolicies()[0]
	result := simulateEncounter(player, enemy, "mid", 0, false, EncounterApproachEngage, policy, policy, false, rand.New(rand.NewSource(20260825)))
	if len(result.Trace) < 2 {
		t.Fatalf("战斗事件数量 = %d，期望至少 2 个事件", len(result.Trace))
	}
	for i := 1; i < len(result.Trace); i++ {
		if result.Trace[i].OffsetSec < result.Trace[i-1].OffsetSec {
			t.Fatalf("战斗事件时间倒退：第%d个=%d，第%d个=%d", i, result.Trace[i-1].OffsetSec, i+1, result.Trace[i].OffsetSec)
		}
	}
	if result.DurationSec != result.Trace[len(result.Trace)-1].OffsetSec {
		t.Fatalf("战斗总时长=%d，与最后事件偏移=%d不一致", result.DurationSec, result.Trace[len(result.Trace)-1].OffsetSec)
	}
}
