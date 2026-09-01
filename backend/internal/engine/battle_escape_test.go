// 烟雾弹脱离测试：烟雾弹在首次判定失败后提供一次额外判定，而非必定成功。
package engine

import (
	"math/rand"
	"strings"
	"testing"
)

// zeroEscapeActor 构造脱离值与追捕值相等的对抗（概率恰好 50%）。
func zeroEscapeActor() (*BattleActor, *BattleActor) {
	player := &BattleActor{Name: "玩家", StealthEff: 0, Agility: 0, Armor: Armor{Escape: 0}}
	enemy := &BattleActor{Name: "敌人", PerceptionEff: 0, Weapon: Weapon{Suppress: 0}, Mobility: 0}
	return player, enemy
}

func TestSmokeDoesNotConsumeWhenFirstEscapeSucceeds(t *testing.T) {
	player, enemy := zeroEscapeActor()
	// seed 3: 首次掷 9 <= 50，直接成功。
	result := tryEscape(DefaultTuning(), player, enemy, true, rand.New(rand.NewSource(3)))
	if !result.success || result.usedSmoke {
		t.Fatalf("首次判定即成功时不应消耗烟雾弹: success=%t usedSmoke=%t", result.success, result.usedSmoke)
	}
	if strings.Contains(result.msg, "二次判定") {
		t.Fatalf("首次判定成功不应出现二次判定文案: %s", result.msg)
	}
}

func TestSmokeProvidesSecondRollAndSucceeds(t *testing.T) {
	player, enemy := zeroEscapeActor()
	// seed 8: 首次掷 89 失败，二次掷 49 成功。
	result := tryEscape(DefaultTuning(), player, enemy, true, rand.New(rand.NewSource(8)))
	if !result.usedSmoke {
		t.Fatal("首次判定失败后应消耗烟雾弹")
	}
	if !result.success {
		t.Fatalf("烟雾弹二次判定应成功: %s", result.msg)
	}
	if !strings.Contains(result.msg, "二次判定") {
		t.Fatalf("烟雾弹消耗后应标注二次判定: %s", result.msg)
	}
}

func TestSmokeSecondRollCanStillFail(t *testing.T) {
	player, enemy := zeroEscapeActor()
	// seed 1: 首次掷 82 失败，二次掷 88 仍失败：烟雾弹消耗但不保证成功。
	result := tryEscape(DefaultTuning(), player, enemy, true, rand.New(rand.NewSource(1)))
	if !result.usedSmoke {
		t.Fatal("首次判定失败后应消耗烟雾弹")
	}
	if result.success {
		t.Fatalf("烟雾弹不保证成功，二次判定失败应保持失败: %s", result.msg)
	}
}

func TestEscapeWithoutSmokeMatchesSingleRoll(t *testing.T) {
	player, enemy := zeroEscapeActor()
	// 同一 seed 8 且无烟雾弹：仅保留首次判定（失败）。
	withoutSmoke := tryEscape(DefaultTuning(), player, enemy, false, rand.New(rand.NewSource(8)))
	if withoutSmoke.success || withoutSmoke.usedSmoke {
		t.Fatalf("无烟雾弹时判定失败: success=%t usedSmoke=%t", withoutSmoke.success, withoutSmoke.usedSmoke)
	}
}