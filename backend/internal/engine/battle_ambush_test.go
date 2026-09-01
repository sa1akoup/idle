// 伏击距离接管测试：单方先发现时，伏击方按自己武器的优势距离接管接敌距离。
package engine

import (
	"math/rand"
	"strings"
	"testing"
)

// ambushTestActors 构造发现率被钳到极值的双方：
// 玩家感知/潜行/敏捷拉满 -> 玩家发现率 90%；敌人感知/潜行极低 -> 敌人发现率 10%。
func ambushTestActors() (*BattleActor, *BattleActor) {
	playerWeapon := Weapon{
		ID: "player_rifle", Name: "玩家步枪", Hit: 60, Damage: 5, AmmoPerRound: 0,
		CloseMod: 15, MidMod: 8, FarMod: 0, // 最佳距离：近距离
	}
	enemyWeapon := Weapon{
		ID: "enemy_sniper", Name: "敌方狙击", Hit: 60, Damage: 5, AmmoPerRound: 0,
		CloseMod: -30, MidMod: 0, FarMod: 18, // 最佳距离：远距离
	}
	player := &BattleActor{
		Name: "玩家", MaxHP: 1000, HP: 1000, StressThreshold: 1000,
		PerceptionEff: 100, StealthEff: 100, Agility: 100, ResistEff: 50,
		Weapon: playerWeapon, Armor: Armor{Conceal: 12}, ArmorDurability: 100, ArmorMaxDur: 100,
	}
	enemy := &BattleActor{
		Name: "敌人", MaxHP: 1000, HP: 1000, StressThreshold: 1000,
		PerceptionEff: 5, StealthEff: 5, Agility: 5, ResistEff: 40,
		Weapon: enemyWeapon, Armor: Armor{Conceal: 0}, ArmorDurability: 100, ArmorMaxDur: 100,
	}
	return player, enemy
}

func TestPlayerAmbushTakesOwnWeaponBestDistance(t *testing.T) {
	player, enemy := ambushTestActors()
	policy := DefaultStylePolicies()[0]
	// seed 1：玩家发现(82<=90)、敌人未发现(88>10) -> 玩家伏击；节点距离 mid，玩家最佳 close。
	result := simulateEncounter(DefaultTuning(), player, enemy, "mid", 0, false, EncounterApproachEngage, policy, policy, false, rand.New(rand.NewSource(1)))
	joined := strings.Join(result.Lines, "\n")
	if !strings.Contains(joined, "伏击成功，把接敌距离拉入 玩家步枪 的优势射程（近距离）") {
		t.Fatalf("玩家伏击未按自己武器最佳距离接管接敌距离:\n%s", joined)
	}
}

func TestEnemyAmbushForcesPlayerIntoEnemyBestDistance(t *testing.T) {
	player, enemy := ambushTestActors()
	policy := DefaultStylePolicies()[0]
	// seed 28：玩家未发现(93>90)、敌人发现(6<=10) -> 敌人伏击；节点距离 mid，敌人最佳 far。
	result := simulateEncounter(DefaultTuning(), player, enemy, "mid", 0, false, EncounterApproachEngage, policy, policy, false, rand.New(rand.NewSource(28)))
	joined := strings.Join(result.Lines, "\n")
	if !strings.Contains(joined, "被敌人伏击，被迫在 敌方狙击 的优势射程交战（远距离）") {
		t.Fatalf("敌人伏击未按自己武器最佳距离接管接敌距离:\n%s", joined)
	}
}

func TestMutualDetectionKeepsNodeDistance(t *testing.T) {
	player, enemy := ambushTestActors()
	policy := DefaultStylePolicies()[0]
	// seed 6：双方同时发现(49<=90、4<=10) -> 不调整距离。
	result := simulateEncounter(DefaultTuning(), player, enemy, "mid", 0, false, EncounterApproachEngage, policy, policy, false, rand.New(rand.NewSource(6)))
	joined := strings.Join(result.Lines, "\n")
	if strings.Contains(joined, "伏击成功") || strings.Contains(joined, "被敌人伏击") {
		t.Fatalf("双方同时发现不应调整接敌距离:\n%s", joined)
	}
}

func TestBestDistanceForWeapon(t *testing.T) {
	tests := []struct {
		name   string
		weapon Weapon
		want   string
	}{
		{name: "近战武器", weapon: Weapon{CloseMod: 15, MidMod: -100, FarMod: -100}, want: "close"},
		{name: "步枪中距离最优", weapon: Weapon{CloseMod: -5, MidMod: 10, FarMod: -8}, want: "mid"},
		{name: "狙击远距离最优", weapon: Weapon{CloseMod: -30, MidMod: 0, FarMod: 18}, want: "far"},
		{name: "处处无法攻击", weapon: Weapon{CloseMod: -100, MidMod: -100, FarMod: -100}, want: ""},
		{name: "同修正就近", weapon: Weapon{CloseMod: 5, MidMod: 5, FarMod: 5}, want: "close"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bestDistanceForWeapon(tt.weapon); got != tt.want {
				t.Fatalf("bestDistanceForWeapon = %q，期望 %q", got, tt.want)
			}
		})
	}
}