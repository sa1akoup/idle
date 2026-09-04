package engine

var skillAttributeThresholds = []int{20, 40, 60, 80}

// applyRunProgression 在撤离成功后写入本局技能与主属性成长，每项最多跨一档。
func applyRunProgression(character *CharacterState, credits map[string]struct{}, physicalBonusPercent int) {
	if character == nil {
		return
	}
	before := *character
	for name := range credits {
		bumpSkill(character, name)
		if physicalBonusPercent > 0 && (name == "stealth" || name == "survival" || name == "resist") {
			bumpSkill(character, name)
		}
	}
	applyAttributeThresholds(character, before)
}

func bumpSkill(character *CharacterState, name string) {
	ptr := characterSkillPtr(character, name)
	if ptr == nil || *ptr >= 100 {
		return
	}
	*ptr++
}

func applyAttributeThresholds(character *CharacterState, before CharacterState) {
	type crossing struct {
		before int
		after  int
		attr   *int
	}
	crossings := []crossing{
		{before.Stealth, character.Stealth, &character.Agility},
		{before.Perception, character.Perception, &character.Agility},
		{before.Negotiation, character.Negotiation, &character.Charisma},
		{before.Luck, character.Luck, &character.Charisma},
		{before.Survival, character.Survival, &character.Strength},
		{before.Resist, character.Resist, &character.Strength},
		{before.Engineering, character.Engineering, &character.Intellect},
		{before.Medical, character.Medical, &character.Intellect},
	}
	for _, item := range crossings {
		gained := 0
		for _, threshold := range skillAttributeThresholds {
			if item.before < threshold && item.after >= threshold {
				gained++
			}
		}
		if gained == 0 || item.attr == nil {
			continue
		}
		*item.attr += gained
		if *item.attr > 100 {
			*item.attr = 100
		}
	}
}

func characterSkillPtr(character *CharacterState, name string) *int {
	switch name {
	case "stealth":
		return &character.Stealth
	case "perception":
		return &character.Perception
	case "negotiation":
		return &character.Negotiation
	case "luck":
		return &character.Luck
	case "survival":
		return &character.Survival
	case "resist":
		return &character.Resist
	case "engineering":
		return &character.Engineering
	case "medical":
		return &character.Medical
	default:
		return nil
	}
}
