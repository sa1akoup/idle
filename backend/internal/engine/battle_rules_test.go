// 战斗基础规则测试：锁定有效技能、最大生命和压力阈值的统一公式。
package engine

import (
	"math"
	"strconv"
	"testing"
)

func TestEffectiveSkill(t *testing.T) {
	tests := []struct {
		name     string
		train    int
		mainAttr int
		want     float64
	}{
		{name: "均值属性", train: 50, mainAttr: 50, want: 50},
		{name: "技能偏高", train: 80, mainAttr: 40, want: 70},
		{name: "属性偏高", train: 40, mainAttr: 80, want: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertFloatEqual(t, EffectiveSkill(DefaultTuning(), tt.train, tt.mainAttr), tt.want)
		})
	}
}

func TestCalcMaxHPClampsToRange(t *testing.T) {
	tests := []struct {
		strength int
		want     float64
	}{
		{strength: 0, want: 90},
		{strength: 50, want: 100},
		{strength: 75, want: 105},
		{strength: 100, want: 110},
		{strength: 200, want: 110},
	}
	for _, tt := range tests {
		t.Run("strength_"+strconv.Itoa(tt.strength), func(t *testing.T) {
			assertFloatEqual(t, CalcMaxHP(DefaultTuning(), tt.strength), tt.want)
		})
	}
}

func TestCalcStressThreshold(t *testing.T) {
	assertFloatEqual(t, CalcStressThreshold(DefaultTuning(), 50), 80)
}

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("结果 = %v，期望 %v", got, want)
	}
}
