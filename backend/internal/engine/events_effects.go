// 事件效果与判定：集中处理随机判定、效果应用和敌人解析。
package engine

import (
	"fmt"
	"math/rand"
)

// resolveEventCheck 掷骰判定事件成败：固定概率、属性差值或直接通过，结果钳在 5-95%。
func resolveEventCheck(check EventCheck, option EventOption, state *eventRunState, rng *rand.Rand) (bool, string) {
	t := state.Tuning.Events
	switch check.Type {
	case "", "none":
		return true, ""
	case "fixed":
		probability := int(clamp(float64(check.Target), t.CheckMin, t.CheckMax))
		roll := rng.Intn(100) + 1
		return roll <= probability, fmt.Sprintf("固定判定 %d%%，掷 %d", probability, roll)
	case "attribute":
		value := getAttrValue(state.Character, check.Attribute)
		probability := int(t.CheckBase) + int(float64(value-check.Target)*t.CheckCoef)
		if state.hasItem(check.ItemBonusRef) {
			probability += check.ItemBonus
		}
		probability += stylePolicy(state.Styles, state.Style).checkBonus(option)
		probability = int(clamp(float64(probability), t.CheckMin, t.CheckMax))
		roll := rng.Intn(100) + 1
		return roll <= probability, fmt.Sprintf("%s=%d，成功率 %d%%，掷 %d", check.Attribute, value, probability, roll)
	default:
		return false, fmt.Sprintf("未知判定类型 %s", check.Type)
	}
}

// applyEventEffect 按效果类型修改运行状态并返回可读摘要，未知效果类型报错。
func applyEventEffect(effect EventEffect, state *eventRunState) (string, error) {
	switch effect.Type {
	case "hp":
		state.Player.HP = clamp(state.Player.HP+float64(effect.Value), 0, state.Player.MaxHP)
		return fmt.Sprintf("生命 %+d，当前 %.0f/%.0f", effect.Value, state.Player.HP, state.Player.MaxHP), nil
	case "stress":
		state.Player.Stress = clamp(state.Player.Stress+float64(effect.Value), 0, state.Player.StressThreshold)
		return fmt.Sprintf("压力 %+d，当前 %.0f/%.0f", effect.Value, state.Player.Stress, state.Player.StressThreshold), nil
	case "heat":
		state.Heat = maxInt(state.Heat+effect.Value, 0)
		return fmt.Sprintf("热度 %+d，当前 %d", effect.Value, state.Heat), nil
	case "time":
		previousDuration := state.DurationSec
		state.DurationSec += int64(effect.Value) * 60
		if state.DurationSec < 0 {
			state.DurationSec = 0
		}
		// 负时间只能抵扣尚未产生事件的时段，避免实时事件时间轴回退。
		if state.Trace != nil && len(*state.Trace) > 0 {
			latestOffset := (*state.Trace)[len(*state.Trace)-1].OffsetSec
			if state.DurationSec < latestOffset {
				state.DurationSec = latestOffset
			}
		}
		appliedSec := state.DurationSec - previousDuration
		if appliedSec == int64(effect.Value)*60 {
			return fmt.Sprintf("行动时间 %+d 分钟", effect.Value), nil
		}
		return fmt.Sprintf("行动时间调整 %+d 分钟，当前进度实际调整 %+d 秒", effect.Value, appliedSec), nil
	case "armor":
		state.Player.ArmorDurability = clamp(state.Player.ArmorDurability+float64(effect.Value), 0, state.Player.ArmorMaxDur)
		return fmt.Sprintf("护甲耐久 %+d，当前 %.0f", effect.Value, state.Player.ArmorDurability), nil
	case "ammo":
		previousRounds, nextRounds := state.adjustAmmo(effect.Value)
		// 弹药负数即消耗，需要计入本局总弹药用量。
		if effect.Value < 0 {
			state.AmmoUsed += previousRounds - nextRounds
		}
		return fmt.Sprintf("弹药 %+d，当前 %d", effect.Value, nextRounds), nil
	case "container":
		if state.CollectContainer == nil {
			return "", fmt.Errorf("容器收集器未初始化")
		}
		if err := state.CollectContainer(effect.Ref, "事件"); err != nil {
			return "", err
		}
		return "搜索容器 " + effect.Ref, nil
	case "container_pool":
		if state.CollectContainerPool == nil {
			return "", fmt.Errorf("事件奖励容器收集器未初始化")
		}
		count := effect.Value
		if count <= 0 {
			count = 1
		}
		if err := state.CollectContainerPool(effect.Ref, "事件奖励", count); err != nil {
			return "", err
		}
		return fmt.Sprintf("按权重搜索事件奖励池 %s x%d", effect.Ref, count), nil
	case "encounter":
		state.EncounterRole = effect.Ref
		return "遭遇池切换为 " + effect.Ref, nil
	case "skip_combat":
		state.SkipDefaultCombat = true
		state.EncounterRole = ""
		return "避开本节点交战", nil
	case "skip_search":
		state.SkipSearch = true
		return "跳过本节点搜索", nil
	case "start_evacuation":
		reason := effect.Ref
		if reason == "" {
			reason = "event"
		}
		state.beginEvacuation(reason, effect.Value > 0)
		return "进入撤离模式", nil
	case "set_flag":
		state.Flags[effect.Ref] = effect.Value >= 0
		return "记录局内标记 " + effect.Ref, nil
	case "consume_item":
		if state.consumeItem(effect.Ref) {
			return "消耗 " + effect.Ref, nil
		}
		return "未携带可消耗的 " + effect.Ref, nil
	case "discard_loot":
		if state.DiscardLoot == nil {
			return "", fmt.Errorf("物资丢弃器未初始化")
		}
		discarded, err := state.DiscardLoot(maxInt(effect.Value, 1))
		if err != nil {
			return "", fmt.Errorf("重算丢弃物资容量: %w", err)
		}
		return fmt.Sprintf("丢弃 %d 件物资", discarded), nil
	case "evac_shortcut":
		if effect.Value <= 0 {
			return "", fmt.Errorf("撤离捷径缩短时间必须为正数")
		}
		state.NextMoveReductionSec += int64(effect.Value) * 60
		return fmt.Sprintf("发现通往撤离点的临时捷径，下一段移动缩短 %d 分钟", effect.Value), nil
	default:
		return "", fmt.Errorf("未知事件效果 %s", effect.Type)
	}
}

// resolveEnemyID 按遭遇角色池加权随机选出一个敌人 ID，权重非正值按 1 参与。
func (manager *eventManager) resolveEnemyID(role string, rng *rand.Rand) (string, error) {
	entries := manager.encounterPool[role]
	if len(entries) == 0 {
		return "", fmt.Errorf("地图 %s 未配置遭遇角色 %s", manager.gameMap.ID, role)
	}
	totalWeight := 0
	for _, entry := range entries {
		totalWeight += maxInt(entry.Weight, 1)
	}
	roll := rng.Intn(totalWeight)
	for _, entry := range entries {
		weight := maxInt(entry.Weight, 1)
		if roll < weight {
			return entry.EnemyID, nil
		}
		roll -= weight
	}
	return entries[len(entries)-1].EnemyID, nil
}

// getAttrValue 取角色属性值，未知属性名退回默认 50。
func getAttrValue(character *CharacterState, attribute string) int {
	switch attribute {
	case "strength":
		return character.Strength
	case "agility":
		return character.Agility
	case "intellect":
		return character.Intellect
	case "charisma":
		return character.Charisma
	case "perception":
		return character.Perception
	case "stealth":
		return character.Stealth
	case "negotiation":
		return character.Negotiation
	case "engineering":
		return character.Engineering
	case "medical":
		return character.Medical
	case "luck":
		return character.Luck
	case "survival":
		return character.Survival
	case "resist":
		return character.Resist
	default:
		return 50
	}
}
