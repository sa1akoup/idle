// 单局探索模拟：推进节点、事件、战斗、搜索和撤离，生成可重放结果。
package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

type simulatedRun struct {
	result                  string
	durationSec             int64
	heat                    int
	ammoUsed                int
	ammoRounds              int
	playerHP                float64
	energy                  float64
	hydration               float64
	playerStress            float64
	carriedItems            []CarriedItem
	finished                bool
	skipResourceConsumption bool
	armorDurability         int
	loot                    []LootDrop
	extractedLoot           []LootDrop
	consumedItems           map[string]int
	report                  []string
	trace                   []TraceEvent
}

// SimulateRun 是探索引擎唯一的运行入口。它只读取快照和输入，不执行数据库读写。
func SimulateRun(snapshot ScenarioSnapshot, input RunInput) (RunResult, error) {
	if err := ValidateSnapshot(snapshot); err != nil {
		return RunResult{}, fmt.Errorf("校验场景快照: %w", err)
	}
	style, err := resolveStyle(snapshot.Styles, input.Style)
	if err != nil {
		return RunResult{}, err
	}
	weapon, ok := snapshot.Weapons[input.State.Loadout.WeaponID]
	if !ok {
		return RunResult{}, fmt.Errorf("武器 %s 不存在", input.State.Loadout.WeaponID)
	}
	armor, ok := snapshot.Armors[input.State.Loadout.ArmorID]
	if !ok {
		return RunResult{}, fmt.Errorf("护甲 %s 不存在", input.State.Loadout.ArmorID)
	}
	var ammo Ammo
	if weapon.AmmoPerRound > 0 {
		ammo, ok = snapshot.Ammos[input.State.Ammo.ID]
		if !ok {
			return RunResult{}, fmt.Errorf("弹药 %s 不存在", input.State.Ammo.ID)
		}
		if ammo.CaliberID != weapon.CaliberID || input.State.Ammo.CaliberID != ammo.CaliberID || input.State.Ammo.Level != ammo.Level {
			return RunResult{}, fmt.Errorf("携带弹药与武器口径或快照配置不匹配")
		}
		if input.State.Ammo.Rounds < weapon.AmmoPerRound {
			return RunResult{}, fmt.Errorf("携带弹药不足以完成一次攻击")
		}
		preferred, ok := snapshot.Ammos[input.State.Ammo.PreferredID]
		if !ok || preferred.CaliberID != weapon.CaliberID || preferred.Level != input.State.Ammo.PreferredLevel {
			return RunResult{}, fmt.Errorf("首选弹药与武器口径或快照配置不匹配")
		}
		if ammo.Level > preferred.Level || input.State.Ammo.TargetRounds < weapon.AmmoPerRound {
			return RunResult{}, fmt.Errorf("当前弹药等级或自动补给目标无效")
		}
	} else if input.State.Ammo.ID != "" || input.State.Ammo.Rounds != 0 || input.State.Ammo.PreferredID != "" || input.State.Ammo.PreferredLevel != 0 || input.State.Ammo.TargetRounds != 0 {
		return RunResult{}, fmt.Errorf("近战武器不能携带弹药")
	}
	usedSlots, usedWeight, err := LoadoutUsage(snapshot, input.State.Loadout, input.State.Consumables, input.State.Ammo)
	if err != nil {
		return RunResult{}, fmt.Errorf("计算当前配装容量: %w", err)
	}
	if usedSlots > input.State.Carry.TotalSlots || usedWeight > input.State.Carry.TotalWeight+1e-9 {
		return RunResult{}, fmt.Errorf("当前配装超过携行容量：%d/%d 格，%.1f/%.1fkg", usedSlots, input.State.Carry.TotalSlots, usedWeight, input.State.Carry.TotalWeight)
	}
	freeSlots := input.State.Carry.TotalSlots - usedSlots
	freeWeight := input.State.Carry.TotalWeight - usedWeight
	// 耳机听力等级：来自快照内的耳机目录（装备丢失/未佩戴时为 0）。
	hearing := 0
	if input.State.Loadout.HeadsetID != "" {
		if headset, ok := snapshot.Headsets[input.State.Loadout.HeadsetID]; ok {
			hearing = headset.HearingLevel
		}
	}
	rng := rand.New(rand.NewSource(sessionRunSeed(input.SessionSeed, input.RunIndex)))
	outcome, err := simulateSingleRun(snapshot, input.State.Character, weapon, armor, input.State.ArmorDurability, ammo, input.State.Ammo.Rounds, input.State.Consumables, input.State.CarriedItems, snapshot.ItemUseDefs, snapshot.Nodes, rng, style, freeSlots, freeWeight, input.RunIndex, hearing)
	if err != nil {
		return RunResult{}, err
	}

	nextState := cloneEngineState(input.State)
	nextState.ArmorDurability = outcome.armorDurability
	nextState.Character.HP = outcome.playerHP
	nextState.Character.Energy = outcome.energy
	nextState.Character.Hydration = outcome.hydration
	nextState.Character.Stress = int(math.Round(outcome.playerStress))
	nextState.CarriedItems = CloneCarriedItems(outcome.carriedItems)
	if outcome.result == "success" {
		increaseWeaponProf(&nextState.Character, weapon.Category)
	}
	remaining := subtractItemStacks(nextState.Consumables, outcome.consumedItems)
	nextState.Consumables = remaining
	nextState.Ammo.Rounds = outcome.ammoRounds
	nextState.Carry.UsedSlots, nextState.Carry.UsedWeight, err = LoadoutUsage(snapshot, nextState.Loadout, nextState.Consumables, nextState.Ammo)
	if err != nil {
		return RunResult{}, fmt.Errorf("计算单局后配装容量: %w", err)
	}

	return RunResult{
		Result:                  outcome.result,
		DurationSec:             outcome.durationSec,
		Heat:                    outcome.heat,
		AmmoUsed:                outcome.ammoUsed,
		StartHP:                 input.State.Character.HP,
		EndHP:                   outcome.playerHP,
		StartEnergy:             input.State.Character.Energy,
		EndEnergy:               outcome.energy,
		StartHydration:          input.State.Character.Hydration,
		EndHydration:            outcome.hydration,
		Loot:                    cloneLoot(outcome.loot),
		ExtractedLoot:           cloneLoot(outcome.extractedLoot),
		ConsumedItems:           itemCountsToStacks(outcome.consumedItems),
		Report:                  append([]string(nil), outcome.report...),
		Trace:                   append([]TraceEvent(nil), outcome.trace...),
		NextState:               nextState,
		Finished:                outcome.finished,
		SkipResourceConsumption: outcome.skipResourceConsumption || outcome.result == "incapacitated",
	}, nil
}

// SimulateRunVersion 按持久化的语义版本选择引擎实现，避免未来算法升级后误用当前版本。
func SimulateRunVersion(version string, snapshot ScenarioSnapshot, input RunInput) (RunResult, error) {
	switch version {
	case EngineVersion:
		return SimulateRun(snapshot, input)
	default:
		return RunResult{}, fmt.Errorf("不支持的探索引擎版本 %s", version)
	}
}

// cloneEngineState 深拷贝输入状态，后续模拟对副本修改而不污染调用方数据。
func cloneEngineState(state EngineState) EngineState {
	state.Consumables = cloneItemStacks(state.Consumables)
	state.CarriedItems = CloneCarriedItems(state.CarriedItems)
	return state
}

// sessionRunSeed 由会话种子按局次序派生独立随机种子，保证每局可单独重放。
func sessionRunSeed(seed int64, runIndex int) int64 {
	// 7919 为质数步进，不同局次的种子互不重叠。
	return seed + int64(runIndex)*7919
}

// increaseWeaponProf 撤离成功时给对应武器类别熟练度 +1，封顶 100。
func increaseWeaponProf(character *CharacterState, category string) {
	value := func(current int) int {
		if current >= 100 {
			return 100
		}
		return current + 1
	}
	switch category {
	case "melee":
		character.MeleeProf = value(character.MeleeProf)
	case "pistol":
		character.PistolProf = value(character.PistolProf)
	case "smg":
		character.SMGProf = value(character.SMGProf)
	case "shotgun":
		character.ShotgunProf = value(character.ShotgunProf)
	case "rifle":
		character.RifleProf = value(character.RifleProf)
	case "sniper":
		character.SniperProf = value(character.SniperProf)
	}
}

// itemCountsToStacks 把消耗计数表转换成按 ID 排序的物品堆叠列表，保证输出顺序确定。
func itemCountsToStacks(counts map[string]int) []ItemStack {
	ids := make([]string, 0, len(counts))
	for id, quantity := range counts {
		if quantity > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	result := make([]ItemStack, 0, len(ids))
	for _, id := range ids {
		result = append(result, ItemStack{ItemID: id, Quantity: counts[id]})
	}
	return result
}

// subtractItemStacks 按消耗数量扣除物品堆叠，扣完的堆叠从结果中移除。
func subtractItemStacks(stacks []ItemStack, consumed map[string]int) []ItemStack {
	result := make([]ItemStack, 0, len(stacks))
	for _, stack := range stacks {
		stack.Quantity -= consumed[stack.ItemID]
		if stack.Quantity > 0 {
			result = append(result, stack)
		}
	}
	return result
}

// LoadoutUsage 统计当前配装、消耗品和自带弹药占用的格子与负重。
func LoadoutUsage(snapshot ScenarioSnapshot, loadout LoadoutState, consumables []ItemStack, ammo CarriedAmmo) (int, float64, error) {
	ids := []string{loadout.WeaponID, loadout.ArmorID, loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID}
	slots, weight := 0, 0.0
	for _, itemID := range ids {
		if itemID == "" {
			continue
		}
		item, ok := snapshot.Items[itemID]
		if !ok {
			return 0, 0, fmt.Errorf("配装物品 %s 不在场景快照目录中", itemID)
		}
		slots += item.Slots
		weight += float64(item.Weight)
	}
	for _, stack := range consumables {
		if stack.Quantity <= 0 {
			continue
		}
		item, ok := snapshot.Items[stack.ItemID]
		if !ok {
			return 0, 0, fmt.Errorf("补给物品 %s 不在场景快照目录中", stack.ItemID)
		}
		if item.Kind == "ammo" {
			return 0, 0, fmt.Errorf("弹药 %s 不能作为普通补给携带", stack.ItemID)
		}
		slots += item.Slots * stack.Quantity
		weight += float64(item.Weight * stack.Quantity)
	}
	ammoSlots, ammoWeight, err := ammoUsage(snapshot, ammo.ID, ammo.Rounds)
	if err != nil {
		return 0, 0, err
	}
	return slots + ammoSlots, weight + ammoWeight, nil
}
