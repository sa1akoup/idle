// 探索生存资源：处理行动时的 Energy/Hydration 消耗和非战斗自动医疗。
package engine

import "sort"

const (
	energyDrainPerHour    = 8.0
	hydrationDrainPerHour = 10.0
)

func applyNeedDrain(character *CharacterState, durationSec int64) {
	if durationSec <= 0 {
		return
	}
	hours := float64(durationSec) / 3600
	character.Energy = clamp(character.Energy-hours*energyDrainPerHour, 0, 100)
	character.Hydration = clamp(character.Hydration-hours*hydrationDrainPerHour, 0, 100)
}

func maybeAutoHeal(state *eventRunState) {
	if state.Player == nil || state.Player.HP <= 0 || state.Player.HP >= state.Player.MaxHP*0.6 {
		return
	}
	target := state.Player.MaxHP * 0.8
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
		if state.Player.HP >= target {
			break
		}
		item := &state.CarriedItems[index]
		definition := state.ItemUseDefs[item.ItemID]
		healRatio := 1.0
		if item.InstanceID > 0 {
			useDurability := definition.UseDurability
			if useDurability <= 0 {
				useDurability = item.CurrentDurability
			}
			actualUse := useDurability
			if actualUse > item.CurrentDurability {
				actualUse = item.CurrentDurability
			}
			if item.CurrentDurability < useDurability {
				healRatio = item.CurrentDurability / useDurability
			}
			item.CurrentDurability -= actualUse
		} else {
			item.Quantity--
		}
		if healRatio <= 0 {
			continue
		}
		state.Player.HP = clamp(state.Player.HP+definition.HPRecovery*healRatio, 0, state.Player.MaxHP)
		state.ConsumedItems[item.ItemID]++
		state.AvailableItems[item.ItemID]--
	}
	filtered := state.CarriedItems[:0]
	for _, item := range state.CarriedItems {
		if item.InstanceID == 0 && item.Quantity <= 0 {
			continue
		}
		filtered = append(filtered, item)
	}
	state.CarriedItems = filtered
	state.Character.HP = state.Player.HP
}

// maybeAutoRecoverNeeds 在非战斗流程中自动使用携行食物和饮水，维持挂机行动的连续性。
func maybeAutoRecoverNeeds(state *eventRunState) {
	if state.Character == nil || (state.Character.Energy >= 80 && state.Character.Hydration >= 80) {
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
		needsEnergy := state.Character.Energy < 80 && definition.EnergyRecovery > 0
		needsHydration := state.Character.Hydration < 80 && definition.HydrationRecovery > 0
		if (!needsEnergy && !needsHydration) || !canUseNeedRecovery(*state.Character, definition) {
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
		if state.Character.Energy >= 80 && state.Character.Hydration >= 80 {
			break
		}
		item := state.CarriedItems[index]
		definition := state.ItemUseDefs[item.ItemID]
		needsEnergy := state.Character.Energy < 80 && definition.EnergyRecovery > 0
		needsHydration := state.Character.Hydration < 80 && definition.HydrationRecovery > 0
		if (!needsEnergy && !needsHydration) || !canUseNeedRecovery(*state.Character, definition) {
			continue
		}
		if !state.consumeCarriedItem(index) {
			continue
		}
		state.Character.Energy = clamp(state.Character.Energy+definition.EnergyRecovery, 0, 100)
		state.Character.Hydration = clamp(state.Character.Hydration+definition.HydrationRecovery, 0, 100)
	}
}

func canUseNeedRecovery(character CharacterState, definition ItemUseDefinition) bool {
	if definition.EnergyRecovery < 0 && character.Energy+definition.EnergyRecovery < 80 {
		return false
	}
	if definition.HydrationRecovery < 0 && character.Hydration+definition.HydrationRecovery < 80 {
		return false
	}
	return true
}
