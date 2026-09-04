package engine

import "testing"

func TestApplyRunProgressionAwardsSkillsAndAttributes(t *testing.T) {
	character := CharacterState{Strength: 25, Agility: 25, Survival: 19, Stealth: 19}
	credits := map[string]struct{}{"survival": {}, "stealth": {}}
	applyRunProgression(&character, credits, 0)
	if character.Survival != 20 || character.Stealth != 20 {
		t.Fatalf("技能 = survival %d stealth %d，期望 20", character.Survival, character.Stealth)
	}
	if character.Strength != 26 || character.Agility != 26 {
		t.Fatalf("主属性 = str %d agi %d，期望 26", character.Strength, character.Agility)
	}
}

func TestApplyRunProgressionPhysicalBonus(t *testing.T) {
	character := CharacterState{Stealth: 10}
	credits := map[string]struct{}{"stealth": {}}
	applyRunProgression(&character, credits, 40)
	if character.Stealth != 12 {
		t.Fatalf("身体技能成长后潜行 = %d，期望 12", character.Stealth)
	}
}

func TestSimulateRunVersionV17SkipsSkillGrowth(t *testing.T) {
	snapshot := replayTestSnapshot()
	input := RunInput{
		SessionSeed: 20260824, RunIndex: 1, Style: ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "回放角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	}
	legacy, err := SimulateRunVersion(LegacyEngineVersionV17, snapshot, input)
	if err != nil {
		t.Fatalf("v17 运行失败: %v", err)
	}
	current, err := SimulateRunVersion(EngineVersion, snapshot, input)
	if err != nil {
		t.Fatalf("v18 运行失败: %v", err)
	}
	if legacy.Result != "success" || current.Result != "success" {
		t.Fatalf("结果 = v17 %s v18 %s", legacy.Result, current.Result)
	}
	if legacy.NextState.Character.Survival != 0 {
		t.Fatalf("v17 生存技能 = %d，期望保持 0", legacy.NextState.Character.Survival)
	}
	if current.NextState.Character.Survival != 1 {
		t.Fatalf("v18 生存技能 = %d，期望 1", current.NextState.Character.Survival)
	}
	if legacy.NextState.Character.PistolProf != 1 || current.NextState.Character.PistolProf != 1 {
		t.Fatalf("武器熟练度应仍在撤离成功时 +1")
	}
}
