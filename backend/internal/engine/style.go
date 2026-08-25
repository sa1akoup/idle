// 纯风格策略模块：为每次调用创建独立策略配置，不持有可变全局状态。
package engine

import (
	"fmt"
	"strings"
)

const (
	ActionStyleBalanced   = "balanced"
	ActionStyleStealth    = "stealth"
	ActionStyleAggressive = "aggressive"
	ActionStyleGreedy     = "greedy"

	EncounterApproachBypass = "bypass"
	EncounterApproachAmbush = "ambush"
	EncounterApproachEngage = "engage"
)

// DefaultStylePolicies 每次返回独立副本，避免引擎依赖可变包级配置。
func DefaultStylePolicies() []StylePolicy {
	return []StylePolicy{
		{
			ID: ActionStyleBalanced, Label: "均衡型", Description: "在收益与风险之间保持平衡。",
			HealthEvacRatio: 0.30, StressEvacRatio: 0.70, CarryEvacRatio: 0.90,
			PatrolApproach: EncounterApproachBypass, ValueWeight: 2, RiskWeight: 2,
			IntentBias:       map[string]int{"bypass": 20, "secure": 10, "search": 10, "ambush": 0, "engage": 0, "rush": 0, "withdraw": 0},
			CheckIntentBonus: map[string]int{"secure": 2, "search": 2, "bypass": 2},
		},
		{
			ID: ActionStyleStealth, Label: "隐秘型", Description: "优先避战、降低热度并尽早保全成果。",
			HealthEvacRatio: 0.40, StressEvacRatio: 0.60, CarryEvacRatio: 0.80,
			PatrolApproach: EncounterApproachBypass, ValueWeight: 1, RiskWeight: 5,
			IntentBias:       map[string]int{"bypass": 45, "conceal": 35, "secure": 20, "search": 0, "ambush": -45, "engage": -35, "force": -30, "rush": -10, "withdraw": 30},
			CheckIntentBonus: map[string]int{"secure": 5, "bypass": 6, "conceal": 6, "withdraw": 5},
		},
		{
			ID: ActionStyleAggressive, Label: "激进型", Description: "主动接战、快速清除阻碍并接受战斗代价。",
			HealthEvacRatio: 0.20, StressEvacRatio: 0.80, CarryEvacRatio: 0.95,
			PatrolApproach: EncounterApproachAmbush, ValueWeight: 2, RiskWeight: 1,
			IntentBias:       map[string]int{"ambush": 45, "engage": 40, "force": 35, "rush": 20, "search": 12, "bypass": -20, "withdraw": -35, "conceal": -15},
			CheckIntentBonus: map[string]int{"ambush": 6, "engage": 6, "force": 5, "rush": 4},
		},
		{
			ID: ActionStyleGreedy, Label: "贪婪型", Description: "优先高价值物资和情报，愿意为收益承担风险。",
			HealthEvacRatio: 0.25, StressEvacRatio: 0.75, CarryEvacRatio: 0.98,
			PatrolApproach: EncounterApproachAmbush, ValueWeight: 6, RiskWeight: 1,
			IntentBias:       map[string]int{"ambush": 25, "engage": 20, "search": 35, "loot": 40, "unlock": 30, "intel": 30, "bypass": -10, "withdraw": -25, "rush": 8},
			CheckIntentBonus: map[string]int{"ambush": 4, "engage": 4, "search": 7, "unlock": 7, "loot": 6},
		},
	}
}

func resolveStyle(styles []StylePolicy, raw string) (string, error) {
	style := strings.TrimSpace(raw)
	if style == "" {
		return ActionStyleBalanced, nil
	}
	for _, policy := range styles {
		if policy.ID == style {
			return style, nil
		}
	}
	return "", fmt.Errorf("未知行动风格 %s", raw)
}

// ResolveStyle 使用当前引擎策略校验并规范化行动风格。
func ResolveStyle(raw string) (string, error) {
	return resolveStyle(DefaultStylePolicies(), raw)
}

func stylePolicy(styles []StylePolicy, style string) StylePolicy {
	for _, policy := range styles {
		if policy.ID == style {
			return policy
		}
	}
	for _, policy := range DefaultStylePolicies() {
		if policy.ID == ActionStyleBalanced {
			return policy
		}
	}
	return StylePolicy{ID: ActionStyleBalanced}
}

func (policy StylePolicy) optionScore(option EventOption) int {
	score := option.Priority
	if option.Intent != "" {
		score += policy.IntentBias[option.Intent]
	}
	score += option.ValueTier*policy.ValueWeight - option.RiskTier*policy.RiskWeight
	score += option.StyleBias[policy.ID]
	return score
}

func (policy StylePolicy) checkBonus(option EventOption) int {
	return policy.CheckIntentBonus[option.Intent] + option.CheckBonus[policy.ID]
}

func normalizeEventOption(definition EventDefinition, option EventOption) EventOption {
	if option.Intent == "" {
		option.Intent = inferEventIntent(definition, option.ID)
	}
	if option.RiskTier == 0 {
		option.RiskTier = inferEventRisk(option)
	}
	if option.ValueTier == 0 {
		option.ValueTier = inferEventValue(option)
	}
	return option
}

func inferEventRisk(option EventOption) int {
	risk := 0
	for _, effect := range option.FailureEffects {
		switch effect.Type {
		case "encounter":
			risk = maxInt(risk, 4)
		case "hp":
			if effect.Value < 0 {
				risk = maxInt(risk, 4)
			}
		case "heat", "stress":
			if effect.Value > 0 {
				risk = maxInt(risk, 2)
			}
		case "skip_search", "discard_loot":
			risk = maxInt(risk, 3)
		case "time":
			if effect.Value > 0 {
				risk = maxInt(risk, 1)
			}
		}
	}
	return risk
}

func inferEventValue(option EventOption) int {
	value := 0
	for _, effect := range option.SuccessEffects {
		switch effect.Type {
		case "start_evacuation":
			value = maxInt(value, 4)
		case "container":
			switch effect.Ref {
			case "server_room", "intel_room", "medical_room", "weapon_cache", "enemy_backpack_elite":
				value = maxInt(value, 5)
			case "material_room", "safehouse_cache", "medical_cache", "enemy_backpack_guard":
				value = maxInt(value, 3)
			default:
				value = maxInt(value, 2)
			}
		case "hp", "stress", "armor":
			if effect.Value > 0 {
				value = maxInt(value, 2)
			}
		}
	}
	return value
}

func inferEventIntent(definition EventDefinition, optionID string) string {
	switch optionID {
	case "evade", "observe", "locate", "negotiate", "reroute", "deal", "slip", "explain", "conceal", "spot", "break_contact", "detour", "identify", "restore":
		return "bypass"
	case "react", "counter":
		return "ambush"
	case "decrypt", "recover", "extract":
		return "intel"
	case "secure", "treat", "repair", "stabilize", "deploy":
		return "secure"
	case "search", "collect", "open", "inspect", "request":
		return "search"
	case "unlock":
		return "unlock"
	case "cross", "brace", "endure", "navigate", "orient", "silence", "wade", "contain", "hold_breath", "dodge":
		return "secure"
	case "carry":
		return "drop"
	case "trace":
		return "reroute"
	case "window":
		return "wait"
	default:
		if definition.Category == "combat" {
			return "engage"
		}
		if definition.Category == "evacuation" {
			return "withdraw"
		}
		return "search"
	}
}
