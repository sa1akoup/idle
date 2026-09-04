// 探索生存资源：处理行动时的 Energy/Hydration 消耗和非战斗自动医疗。
package engine

import (
	"math"
	"sort"
)

// applyNeedDrain 按行动时长折算小时数扣减能量与饮水；生存属性降低消耗速率，钳在上下限内。
func applyNeedDrain(t Tuning, character *CharacterState, durationSec int64) {
	if durationSec <= 0 {
		return
	}
	hours := float64(durationSec) / 3600
	energyRate := t.Survival.EnergyDrainPerHour - float64(character.Survival)*t.Survival.EnergySurvivalCoef
	hydrationRate := t.Survival.HydrationDrainPerHour - float64(character.Survival)*t.Survival.HydrationSurvivalCoef
	energyRate = math.Max(energyRate, t.Survival.DrainMin)
	hydrationRate = math.Max(hydrationRate, t.Survival.DrainMin)
	character.Energy = clamp(character.Energy-hours*energyRate, 0, 100)
	character.Hydration = clamp(character.Hydration-hours*hydrationRate, 0, 100)
}

// maybeAutoHeal 血量过低时按使用优先级自动消耗医疗品，恢复到目标血量即停。
// 医疗属性降低启动阈值（越高越晚动用医疗资源）。
func maybeAutoHeal(state *eventRunState) {
	survival := state.Tuning.Survival
	medical := 0
	if state.Character != nil {
		medical = state.Character.Medical
	}
	trigger := survival.AutoHealTrigger - float64(medical)*survival.AutoHealMedicalCoef
	trigger = math.Max(0, math.Min(trigger, survival.AutoHealTrigger))
	if state.Player == nil || state.Player.HP <= 0 || state.Player.HP >= state.Player.MaxHP*trigger {
		return
	}
	target := state.Player.MaxHP * survival.AutoHealTarget
	indices := make([]int, 0, len(state.CarriedItems))
	for index, item := range state.CarriedItems {
		definition, ok := state.ItemUseDefs[item.ItemID]
		if !ok || !definition.UsableInSession || definition.HPRecovery <= 0 {
			continue
		}
		if item.InstanceID > 0 && item.CurrentDurability <= 0 {
			continue
		}
		if item.InstanceID == 0 && item.Quantity <= 0 {
			continue
		}
		indices = append(indices, index)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := state.CarriedItems[indices[i]], state.CarriedItems[indices[j]]
		leftDef, rightDef := state.ItemUseDefs[left.ItemID], state.ItemUseDefs[right.ItemID]
		if leftDef.UsePriority != rightDef.UsePriority {
			return leftDef.UsePriority < rightDef.UsePriority
		}
		if left.InstanceID > 0 && right.InstanceID > 0 && left.CurrentDurability != right.CurrentDurability {
			return left.CurrentDurability < right.CurrentDurability
		}
		return left.ItemID < right.ItemID
	})
	for _, index := range indices {
		for state.Player.HP < target {
			if index < 0 || index >= len(state.CarriedItems) {
				break
			}
			before := state.CarriedItems[index]
			definition := state.ItemUseDefs[before.ItemID]
			if !state.consumeCarriedItem(index) {
				break
			}
			after := state.CarriedItems[index]
			healRatio := 1.0
			if before.InstanceID > 0 {
				usedDurability := before.CurrentDurability - after.CurrentDurability
				baseDurability := definition.UseDurability
				if baseDurability <= 0 {
					baseDurability = before.CurrentDurability
				}
				if baseDurability > 0 {
					healRatio = usedDurability / baseDurability
				}
			}
			if healRatio <= 0 {
				continue
			}
			state.Player.HP = clamp(state.Player.HP+definition.HPRecovery*healRatio, 0, state.Player.MaxHP)
			state.creditSkill("medical")
		}
	}
	filtered := state.CarriedItems[:0]
	for _, item := range state.CarriedItems {
		if item.InstanceID == 0 && item.Quantity <= 0 {
			continue
		}
		filtered = append(filtered, item)
	}
	state.CarriedItems = filtered
	if state.Character != nil {
		state.Character.HP = state.Player.HP
	}
}

// maybeAutoRecoverNeeds 在非战斗流程中自动使用携行食物和饮水，维持挂机行动的连续性。
func maybeAutoRecoverNeeds(state *eventRunState) {
	threshold := state.Tuning.Survival.AutoRecoverThreshold
	if state.Character == nil || (state.Character.Energy >= threshold*100 && state.Character.Hydration >= threshold*100) {
		return
	}
	indices := make([]int, 0, len(state.CarriedItems))
	for index, item := range state.CarriedItems {
		definition, ok := state.ItemUseDefs[item.ItemID]
		if !ok || !definition.UsableInSession || (definition.EnergyRecovery <= 0 && definition.HydrationRecovery <= 0) {
			continue
		}
		if item.InstanceID > 0 && item.CurrentDurability <= 0 || item.InstanceID == 0 && item.Quantity <= 0 {
			continue
		}
		needsEnergy := float64(state.Character.Energy) < threshold*100 && definition.EnergyRecovery > 0
		needsHydration := float64(state.Character.Hydration) < threshold*100 && definition.HydrationRecovery > 0
		if (!needsEnergy && !needsHydration) || !canUseNeedRecovery(threshold, *state.Character, definition) {
			continue
		}
		indices = append(indices, index)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := state.CarriedItems[indices[i]], state.CarriedItems[indices[j]]
		leftDef, rightDef := state.ItemUseDefs[left.ItemID], state.ItemUseDefs[right.ItemID]
		if leftDef.UsePriority != rightDef.UsePriority {
			return leftDef.UsePriority < rightDef.UsePriority
		}
		return left.ItemID < right.ItemID
	})
	for _, index := range indices {
		for {
			if float64(state.Character.Energy) >= threshold*100 && float64(state.Character.Hydration) >= threshold*100 {
				break
			}
			if index < 0 || index >= len(state.CarriedItems) {
				break
			}
			item := state.CarriedItems[index]
			definition := state.ItemUseDefs[item.ItemID]
			needsEnergy := float64(state.Character.Energy) < threshold*100 && definition.EnergyRecovery > 0
			needsHydration := float64(state.Character.Hydration) < threshold*100 && definition.HydrationRecovery > 0
			if (!needsEnergy && !needsHydration) || !canUseNeedRecovery(threshold, *state.Character, definition) {
				break
			}
			if !state.consumeCarriedItem(index) {
				break
			}
			state.Character.Energy = clamp(state.Character.Energy+definition.EnergyRecovery, 0, 100)
			state.Character.Hydration = clamp(state.Character.Hydration+definition.HydrationRecovery, 0, 100)
		}
	}
}

// canUseNeedRecovery 排除恢复值为负（反而扣减）且会把资源拖回阈值之下的物品。
func canUseNeedRecovery(threshold float64, character CharacterState, definition ItemUseDefinition) bool {
	if definition.EnergyRecovery < 0 && float64(character.Energy)+definition.EnergyRecovery < threshold*100 {
		return false
	}
	if definition.HydrationRecovery < 0 && float64(character.Hydration)+definition.HydrationRecovery < threshold*100 {
		return false
	}
	return true
}
