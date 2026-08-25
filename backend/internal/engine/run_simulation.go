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
	playerStress            float64
	injury                  string
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
	style, err := resolveStyle(snapshot.Styles, input.Style)
	if err != nil {
		return RunResult{}, err
	}
	if err := validateDirectedRoute(snapshot.Nodes, snapshot.Map); err != nil {
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
	freeSlots := input.State.Carry.TotalSlots - input.State.Carry.UsedSlots
	freeWeight := input.State.Carry.TotalWeight - input.State.Carry.UsedWeight
	rng := rand.New(rand.NewSource(sessionRunSeed(input.SessionSeed, input.RunIndex)))
	outcome, err := simulateSingleRun(snapshot, input.State.Character, weapon, armor, input.State.ArmorDurability, ammo, input.State.Ammo.Rounds, input.State.Consumables, snapshot.Nodes, rng, style, freeSlots, freeWeight, input.RunIndex)
	if err != nil {
		return RunResult{}, err
	}

	nextState := cloneEngineState(input.State)
	nextState.ArmorDurability = outcome.armorDurability
	nextState.Character.Stress = int(math.Round(outcome.playerStress))
	nextState.Character.Injury = outcome.injury
	if outcome.result == "success" || outcome.result == "partial" {
		increaseWeaponProf(&nextState.Character, weapon.Category)
	}
	remaining := subtractItemStacks(nextState.Consumables, outcome.consumedItems)
	nextState.Consumables = remaining
	nextState.Ammo.Rounds = outcome.ammoRounds
	nextState.Carry.UsedSlots, nextState.Carry.UsedWeight = loadoutUsage(snapshot, nextState.Loadout, nextState.Consumables)

	return RunResult{
		Result:                  outcome.result,
		DurationSec:             outcome.durationSec,
		Heat:                    outcome.heat,
		AmmoUsed:                outcome.ammoUsed,
		Injury:                  outcome.injury,
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

func cloneEngineState(state EngineState) EngineState {
	state.Consumables = cloneItemStacks(state.Consumables)
	return state
}

func sessionRunSeed(seed int64, runIndex int) int64 {
	return seed + int64(runIndex)*7919
}

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

func loadoutUsage(snapshot ScenarioSnapshot, loadout LoadoutState, consumables []ItemStack) (int, float64) {
	ids := []string{loadout.WeaponID, loadout.ArmorID, loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID}
	for _, stack := range consumables {
		for i := 0; i < stack.Quantity; i++ {
			ids = append(ids, stack.ItemID)
		}
	}
	slots, weight := 0, 0.0
	for _, itemID := range ids {
		item, ok := snapshot.Items[itemID]
		if !ok {
			continue
		}
		slots += item.Slots
		weight += float64(item.Weight)
	}
	return slots, weight
}
