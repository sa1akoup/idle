// 搜索风险模块：把 SearchRisk 激活为真实判定，运气/感知参与压低暴露率，
// 失败惩罚按 Tuning.Search.FailPenalty.Kind 扩展（当前实现 "expose"：耗时翻倍 + 热度）。
package engine

import (
	"fmt"
	"math/rand"
)

// searchExposeProbability 计算搜索暴露概率：风险等级拉高、运气与感知压低，并钳制到配置区间。
func searchExposeProbability(t Tuning, risk int, luck, perception int) float64 {
	st := t.Search
	return clamp(float64(risk)*st.RiskPerLevel-(float64(luck)*st.LuckCoef+float64(perception)*st.PerceptionCoef), st.ExposeMin, st.ExposeMax)
}

// applySearchFailPenalty 按配置的惩罚形态施加搜索失败代价；未知形态报错（快照校验已拦截）。
func applySearchFailPenalty(t Tuning, state *eventRunState, container Container, rng *rand.Rand) error {
	penalty := t.Search.FailPenalty
	switch penalty.Kind {
	case "expose":
		extra := float64(container.SearchTime) * 60 * (penalty.TimeMultiplier - 1)
		if extra < 0 {
			extra = 0
		}
		state.DurationSec += int64(extra)
		state.Heat += penalty.Heat
		*state.Lines = append(*state.Lines, fmt.Sprintf("    翻找动静过大，暴露了行踪（耗时翻倍、热度+%d）", penalty.Heat))
		return nil
	default:
		return fmt.Errorf("未知搜索失败惩罚形态 %s", penalty.Kind)
	}
}

// luckBonusRolls 掷骰决定幸运是否额外多搜一件；返回合计件数并追加播报。
func luckBonusRolls(t Tuning, rolls int, luck int, rng *rand.Rand, lines *[]string) int {
	st := t.Search
	if rolls <= 0 || luck <= 0 || st.LuckBonusCoef <= 0 {
		return rolls
	}
	if float64(rng.Intn(100)+1) <= float64(luck)*st.LuckBonusCoef {
		*lines = append(*lines, "    运气不错，额外翻出一处夹层")
		return rolls + 1
	}
	return rolls
}