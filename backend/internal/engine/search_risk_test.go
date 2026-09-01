// 搜索风险测试：暴露率公式、expose 惩罚与运气增产。
package engine

import (
	"math/rand"
	"testing"
)

func TestSearchExposeProbability(t *testing.T) {
	tuning := DefaultTuning()
	tests := []struct {
		name       string
		risk       int
		luck       int
		perception int
		want       float64
	}{
		{name: "零属性裸风险", risk: 3, luck: 0, perception: 0, want: 24}, // 3×8
		{name: "满运气压低", risk: 3, luck: 100, perception: 0, want: 9},  // 24-15
		{name: "感知叠加压低", risk: 3, luck: 100, perception: 100, want: 2}, // 触底
		{name: "高风险高暴露", risk: 5, luck: 0, perception: 0, want: 40},  // 触顶
		{name: "低风险下限", risk: 1, luck: 100, perception: 100, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchExposeProbability(tuning, tt.risk, tt.luck, tt.perception)
			if got != tt.want {
				t.Fatalf("暴露率 = %.2f，期望 %.2f", got, tt.want)
			}
		})
	}
}

func TestApplySearchExposePenalty(t *testing.T) {
	lines := []string{}
	state := &eventRunState{DurationSec: 60, Heat: 5, Lines: &lines, Trace: &[]TraceEvent{}}
	err := applySearchFailPenalty(DefaultTuning(), state, Container{SearchTime: 2}, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("expose 惩罚执行失败: %v", err)
	}
	// 耗时 60 + 2分钟×60×1（翻倍=多加一份），热度 5+3
	if state.DurationSec != 180 {
		t.Fatalf("expose 后耗时 = %d，期望 180", state.DurationSec)
	}
	if state.Heat != 8 {
		t.Fatalf("expose 后热度 = %d，期望 8", state.Heat)
	}
	if len(*state.Lines) == 0 || (*state.Lines)[0] == "" {
		t.Fatal("expose 应有播报文案")
	}
}

func TestUnknownSearchPenaltyKindRejectedByValidation(t *testing.T) {
	tuning := DefaultTuning()
	tuning.Search.FailPenalty.Kind = "lose_item" // 尚未实现
	if err := ValidateTuning(tuning); err == nil {
		t.Fatal("未实现的惩罚形态应被快照校验拒绝")
	}
}

func TestLuckBonusRolls(t *testing.T) {
	tuning := DefaultTuning() // LuckBonusCoef=0.3，满运气阈值 30
	lines := []string{}
	// 找 seed：首掷 <=30 触发增产，且 rolls<=0 时永不触发。
	bonusSeed := int64(-1)
	for s := int64(1); s < 1000 && bonusSeed < 0; s++ {
		if rand.New(rand.NewSource(s)).Intn(100)+1 <= 30 {
			bonusSeed = s
		}
	}
	if bonusSeed < 0 {
		t.Fatal("未找到增产 seed")
	}
	if got := luckBonusRolls(tuning, 0, 100, rand.New(rand.NewSource(bonusSeed)), &lines); got != 0 {
		t.Fatalf("空容器不应因运气增产: %d", got)
	}
	if got := luckBonusRolls(tuning, 3, 0, rand.New(rand.NewSource(bonusSeed)), &lines); got != 3 {
		t.Fatalf("零运气不应增产: %d", got)
	}
	if got := luckBonusRolls(tuning, 3, 100, rand.New(rand.NewSource(bonusSeed)), &lines); got != 4 {
		t.Fatalf("满运气应额外多搜一件: %d", got)
	}
}