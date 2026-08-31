// 事件配置校验测试：锁定显式方案语义和共享枚举校验，防止运行时再次静默补默认值。
package engine

import "testing"

func TestValidateEventDefinitionRequiresExplicitSemantics(t *testing.T) {
	base := EventDefinition{
		ID: "event_test",
		Options: []EventOption{{
			ID: "option_test", Intent: "secure", RiskTier: 1, ValueTier: 1,
			Check: EventCheck{Type: "none"},
		}},
	}
	tests := []struct {
		name    string
		mutate  func(*EventDefinition)
		wantErr bool
	}{
		{name: "missing intent", mutate: func(definition *EventDefinition) { definition.Options[0].Intent = "" }, wantErr: true},
		{name: "missing risk tier", mutate: func(definition *EventDefinition) { definition.Options[0].RiskTier = 0 }, wantErr: true},
		{name: "missing value tier", mutate: func(definition *EventDefinition) { definition.Options[0].ValueTier = 0 }, wantErr: true},
		{name: "valid semantics", mutate: func(*EventDefinition) {}, wantErr: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := base
			definition.Options = append([]EventOption(nil), base.Options...)
			test.mutate(&definition)
			err := ValidateEventDefinition(definition, DefaultStylePolicies())
			if (err != nil) != test.wantErr {
				t.Fatalf("校验结果 error=%v，期望错误=%t", err, test.wantErr)
			}
		})
	}
}

func TestValidateEventCatalogUsesSharedBindingAndEffectRules(t *testing.T) {
	base := EventCatalog{
		Definitions: map[string]EventDefinition{
			"event_test": {
				ID: "event_test",
				Options: []EventOption{{
					ID: "option_test", Intent: "secure", RiskTier: 1, ValueTier: 1,
					Check: EventCheck{Type: "none"},
				}},
			},
		},
		Bindings: []EventBinding{{
			ID: "binding_test", EventID: "event_test", ScopeType: "global",
			Phase: eventPhaseEnterNode, TriggerBP: 10000, Weight: 1, Enabled: true,
		}},
	}

	if err := ValidateEventCatalog(base, DefaultStylePolicies()); err != nil {
		t.Fatalf("有效事件目录校验失败: %v", err)
	}

	invalidPhase := base
	invalidPhase.Bindings = append([]EventBinding(nil), base.Bindings...)
	invalidPhase.Bindings[0].Phase = "unknown_phase"
	if err := ValidateEventCatalog(invalidPhase, DefaultStylePolicies()); err == nil {
		t.Fatal("未知事件阶段应由共享目录校验拒绝")
	}

	invalidEffect := base
	invalidEffect.Definitions = map[string]EventDefinition{
		"event_test": {
			ID: "event_test",
			Options: []EventOption{{
				ID: "option_test", Intent: "secure", RiskTier: 1, ValueTier: 1,
				Check:          EventCheck{Type: "none"},
				SuccessEffects: []EventEffect{{Type: "unknown_effect"}},
			}},
		},
	}
	if err := ValidateEventCatalog(invalidEffect, DefaultStylePolicies()); err == nil {
		t.Fatal("未知事件效果应由共享目录校验拒绝")
	}
}

func TestSimulateRunRejectsMissingEventSemantics(t *testing.T) {
	snapshot := shortcutStressSnapshot(1)
	definition := snapshot.Events.Definitions["shortcut_test"]
	definition.Options[0].Intent = ""
	snapshot.Events.Definitions[definition.ID] = definition

	if _, err := SimulateRun(snapshot, shortcutStressInput(10)); err == nil {
		t.Fatal("模拟入口应拒绝缺少决策意图的事件方案")
	}
}
