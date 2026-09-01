// 敌人属性硬编码回归测试：智力/抗性来自模板生成，武器操控由真实属性推导。
package engine

import "testing"

func TestBuildEnemyActorUsesTemplateAttributes(t *testing.T) {
	tests := []struct {
		name                string
		enemy               Enemy
		wantResist          float64
		wantIntellect       float64
		wantWeaponControlRange [2]float64 // 由属性推导，落在区间内
	}{
		{name: "属性超纲敌人", enemy: Enemy{Agility: 45, Perception: 40, Suppress: 20, Intellect: 35, Resist: 40}, wantResist: 40, wantIntellect: 35, wantWeaponControlRange: [2]float64{45, 50}},
		{name: "精锐高感知", enemy: Enemy{Agility: 50, Perception: 65, Suppress: 40, Intellect: 50, Resist: 55}, wantResist: 55, wantIntellect: 50, wantWeaponControlRange: [2]float64{58, 62}},
		{name: "未配置回落", enemy: Enemy{Agility: 45, Perception: 40, Suppress: 20}, wantResist: 0, wantIntellect: 0, wantWeaponControlRange: [2]float64{45, 50}},
	}
	weapon := Weapon{CloseMod: 10, MidMod: 5, FarMod: 0}
	armor := Armor{MaxDurability: 100}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := buildEnemyActor(tt.enemy, weapon, armor, Ammo{})
			if actor.ResistEff != tt.wantResist {
				t.Fatalf("敌人抗性 = %.0f，期望 %.0f", actor.ResistEff, tt.wantResist)
			}
			if actor.Intellect != tt.wantIntellect {
				t.Fatalf("敌人智力 = %.0f，期望 %.0f", actor.Intellect, tt.wantIntellect)
			}
			if actor.WeaponControl < tt.wantWeaponControlRange[0] || actor.WeaponControl > tt.wantWeaponControlRange[1] {
				t.Fatalf("敌人武器操控 = %.2f，期望落在 [%.1f, %.1f]", actor.WeaponControl, tt.wantWeaponControlRange[0], tt.wantWeaponControlRange[1])
			}
		})
	}
}

func TestEnemyWeaponControlFollowsAttributes(t *testing.T) {
	weak := enemyWeaponControl(Enemy{Agility: 35, Perception: 30, Suppress: 10})
	strong := enemyWeaponControl(Enemy{Agility: 50, Perception: 65, Suppress: 40})
	if strong <= weak {
		t.Fatalf("高属性敌人的武器操控应高于低属性敌人: weak=%.2f strong=%.2f", weak, strong)
	}
}

func TestPlayerReconUsesIntellect(t *testing.T) {
	low := calcRecon(DefaultTuning(), 60, 0, 0)
	high := calcRecon(DefaultTuning(), 60, 80, 0)
	// 智力权重 0.1：80 智力比 0 智力多贡献 8 点侦察。
	if high-low < 7 || high-low > 9 {
		t.Fatalf("侦察值未按智力差异变化: low=%.2f high=%.2f", low, high)
	}
}