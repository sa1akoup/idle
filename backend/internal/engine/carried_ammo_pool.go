// 携带弹药池结算：多栈弹药的选择、汇总与发数统计。
// 池中每一栈都带自身偏好的自动补给目标，主弹按"等级降序且发数足以开火"确定性选取。
package engine

// selectUsableAmmoStack 从携带弹药池中选中下一次攻击使用的弹药：
// 依次取发数足够一轮（≥ AmmoPerRound）且等级最高的栈；无可用栈时返回 false。
func selectUsableAmmoStack(snapshot ScenarioSnapshot, weapon Weapon, stacks []CarriedAmmo) (Ammo, int, int, bool) {
	bestIndex := -1
	var bestProfile Ammo
	for index, stack := range stacks {
		if stack.ID == "" || stack.Rounds < weapon.AmmoPerRound {
			continue
		}
		profile, ok := snapshot.Ammos[stack.ID]
		if !ok || profile.CaliberID != weapon.CaliberID {
			continue
		}
		if bestIndex == -1 || profile.Level > bestProfile.Level {
			bestIndex = index
			bestProfile = profile
		}
	}
	if bestIndex == -1 {
		return Ammo{}, 0, -1, false
	}
	return bestProfile, stacks[bestIndex].Rounds, bestIndex, true
}

// HasUsableCarriedAmmoStack 判断弹药池中是否存在足够发动一次攻击的有效栈。
// 服务层与引擎层共用该口径，避免“总发数足够但每栈都不足一轮”的状态被误判为可开火。
func HasUsableCarriedAmmoStack(snapshot ScenarioSnapshot, weapon Weapon, stacks []CarriedAmmo) bool {
	_, _, _, ok := selectUsableAmmoStack(snapshot, weapon, stacks)
	return ok
}

// selectAmmoStackForEffect 为事件弹药效果选择一个可修改的同口径弹药栈。
// 优先选择仍有余量的高等级栈；全部耗尽时保留其栈位，允许正向事件效果恢复该栈。
func selectAmmoStackForEffect(snapshot ScenarioSnapshot, weapon Weapon, stacks []CarriedAmmo) (Ammo, int, bool) {
	bestIndex := -1
	var bestProfile Ammo
	for index, stack := range stacks {
		if stack.ID == "" {
			continue
		}
		profile, ok := snapshot.Ammos[stack.ID]
		if !ok || profile.CaliberID != weapon.CaliberID {
			continue
		}
		if bestIndex == -1 || (stack.Rounds > 0 && stacks[bestIndex].Rounds <= 0) ||
			((stack.Rounds > 0) == (stacks[bestIndex].Rounds > 0) && profile.Level > bestProfile.Level) {
			bestIndex = index
			bestProfile = profile
		}
	}
	if bestIndex == -1 {
		return Ammo{}, -1, false
	}
	return bestProfile, bestIndex, true
}

// ammoStacksRounds 统计携带弹药池总发数。
func ammoStacksRounds(stacks []CarriedAmmo) int {
	total := 0
	for _, stack := range stacks {
		total += stack.Rounds
	}
	return total
}

// BestCarriedAmmoSummary 取池中最高等级主弹摘要（含其偏好）。
// 有余量栈优先；全部耗尽时仍保留一个栈的偏好，供下一局自动补弹使用。
func BestCarriedAmmoSummary(stacks []CarriedAmmo) CarriedAmmo {
	best := CarriedAmmo{}
	for _, stack := range stacks {
		if stack.ID == "" {
			continue
		}
		if best.ID == "" || (stack.Rounds > 0 && best.Rounds <= 0) ||
			((stack.Rounds > 0) == (best.Rounds > 0) && stack.Level > best.Level) {
			best = stack
		}
	}
	return best
}
