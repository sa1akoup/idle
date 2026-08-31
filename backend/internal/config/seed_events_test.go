// 事件种子测试：确保每个事件方案都持久化显式决策意图、风险等级和收益等级。
package config

import "testing"

func TestSeedEventsPopulateExplicitOptionSemantics(t *testing.T) {
	for _, definition := range eventDefinitions() {
		for _, option := range definition.Options {
			if option.Intent == "" || option.RiskTier < 1 || option.RiskTier > 5 || option.ValueTier < 1 || option.ValueTier > 5 {
				t.Fatalf("事件 %s 方案 %s 的显式语义不完整: intent=%q risk=%d value=%d", definition.ID, option.ID, option.Intent, option.RiskTier, option.ValueTier)
			}
		}
	}
}
