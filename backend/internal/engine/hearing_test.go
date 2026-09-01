// 耳机听力接线测试：听力等级加成玩家发现率（+听力×3）并降低被敌人发现率（-听力×2）。
package engine

import (
	"math/rand"
	"strings"
	"testing"
)

// hearingTestActors 构造发现率落在听力可翻转区间的双方：
// 无听力时 eFindProb=44.5（roll2=41 会判敌人发现），带听力3时 eFindProb=38.5（roll2=41 不发现）。
func hearingTestActors() (*BattleActor, *BattleActor) {
	player := &BattleActor{
		Name: "玩家", MaxHP: 500, HP: 500, StressThreshold: 500,
		PerceptionEff: 50, StealthEff: 40, Agility: 40, Intellect: 50, ResistEff: 50,
		Weapon: Weapon{ID: "player_gun", Hit: 60, Damage: 2, AmmoPerRound: 0, CloseMod: 10, MidMod: 0, FarMod: -10},
		Armor: Armor{Conceal: 4}, ArmorDurability: 100, ArmorMaxDur: 100,
	}
	enemy := &BattleActor{
		Name: "敌人", MaxHP: 500, HP: 500, StressThreshold: 500,
		PerceptionEff: 30, StealthEff: 20, Agility: 20, Intellect: 40, ResistEff: 40,
		Weapon: Weapon{ID: "enemy_gun", Hit: 60, Damage: 2, AmmoPerRound: 0, CloseMod: -30, MidMod: 0, FarMod: 12},
		Armor: Armor{Conceal: 0}, ArmorDurability: 100, ArmorMaxDur: 100,
	}
	return player, enemy
}

func TestHearingReducesEnemyFindRate(t *testing.T) {
	policy := DefaultStylePolicies()[0]
	// seed 145：roll1=74（玩家未发现，两种听力下都是），roll2=41 位于 38.5/44.5 之间。
	player, enemy := hearingTestActors()
	player.Hearing = 3 // 佩戴三级听力耳机（如 ComTac IV / XCEL）
	withHearing := simulateEncounter(DefaultTuning(), player, enemy, "mid", 0, false, EncounterApproachEngage, policy, policy, false, rand.New(rand.NewSource(145)))
	joined := strings.Join(withHearing.Lines, "\n")
	if !strings.Contains(joined, "双方错过，巡逻擦肩") {
		t.Fatalf("带听力时玩家未发现、敌人也未发现，应显示双方错过:\n%s", joined)
	}
	if strings.Contains(joined, "被敌人伏击") {
		t.Fatalf("带听力时不应被敌人伏击:\n%s", joined)
	}
}

func TestNoHearingGetsAmbushedOnSameRoll(t *testing.T) {
	policy := DefaultStylePolicies()[0]
	// 同一 seed 145、同一对掷骰：无听力时 roll2=41 <= 44.5，敌人发现玩家并伏击。
	player, enemy := hearingTestActors()
	withoutHearing := simulateEncounter(DefaultTuning(), player, enemy, "mid", 0, false, EncounterApproachEngage, policy, policy, false, rand.New(rand.NewSource(145)))
	joined := strings.Join(withoutHearing.Lines, "\n")
	if !strings.Contains(joined, "被敌人伏击") {
		t.Fatalf("无听力时同掷骰应被敌人伏击:\n%s", joined)
	}
}

func TestHearingReconBonusScaling(t *testing.T) {
	// pRecon = 感知×0.7 + 智力×0.1 + 听力×3：基线 60×0.7+50×0.1=47，等级1/2/3 分别 +3/+6/+9。
	if got := calcRecon(DefaultTuning(), 60, 50, 3); got != 50 {
		t.Fatalf("听力1级侦察 = %.2f，期望 50", got)
	}
	if got := calcRecon(DefaultTuning(), 60, 50, 9); got != 56 {
		t.Fatalf("听力3级侦察 = %.2f，期望 56", got)
	}
}