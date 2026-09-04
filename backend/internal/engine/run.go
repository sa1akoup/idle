// 纯探索运行入口：使用固定快照和输入状态执行一局可重放模拟。
package engine

import (
	"fmt"
	"math/rand"
)

// assignLootDropIDs 用局次序和生成顺序固定掉落 ID，拆分带回与溢出时沿用同一 ID。
func assignLootDropIDs(loot []LootDrop, runIndex int) {
	for index := range loot {
		loot[index].ID = fmt.Sprintf("run-%d-drop-%d", runIndex, index+1)
	}
}

// LootItemsByCategory 从快照战利品表里按类别挑一件（供节点与事件共用）。
// valueTier 限制稀有度：LEDX/比特币等传奇件不会从低价值容器抽出。
func (snapshot ScenarioSnapshot) LootItemsByCategory(category string, valueTier int, rng *rand.Rand) (LootItem, bool) {
	return chooseLootItem(snapshot.LootItems, category, valueTier, rng)
}

// NodeContainerAssignmentsForNode 返回某节点配置的容器分配列表。
func (snapshot ScenarioSnapshot) NodeContainerAssignmentsForNode(nodeID string) []NodeContainerAssignment {
	result := make([]NodeContainerAssignment, 0)
	for _, assignment := range snapshot.NodeContainerAssignments {
		if assignment.NodeID == nodeID {
			result = append(result, assignment)
		}
	}
	return result
}

// resolveNodeEncounter 判定节点是否触发遭遇并执行战斗，返回敌人、是否击倒、是否清场及是否交战过。
func resolveNodeEncounter(snapshot ScenarioSnapshot, state *eventRunState, events *eventManager, node Node, rng *rand.Rand) (Enemy, bool, bool, bool, error) {
	if state.SkipDefaultCombat {
		*state.Lines = append(*state.Lines, "  已根据事件结果避开本节点交战")
		return Enemy{}, false, false, false, nil
	}
	enemyID := node.EnemyID
	forced := state.EncounterRole != ""
	encounterRole := node.EncounterRole
	if forced {
		var err error
		enemyID, err = events.resolveEnemyID(state.EncounterRole, rng)
		if err != nil {
			return Enemy{}, false, false, false, err
		}
		encounterRole = state.EncounterRole
	} else if enemyID == "" && encounterRole != "" {
		var err error
		enemyID, err = events.resolveEnemyID(encounterRole, rng)
		if err != nil {
			return Enemy{}, false, false, false, err
		}
	}
	if enemyID == "" {
		*state.Lines = append(*state.Lines, "  当前节点没有配置敌人，安全通过")
		return Enemy{}, false, false, false, nil
	}
	// 遭遇概率：节点基础遇敌概率 + 本局热度；撤离阶段整体下调（默认节点 60 在撤离时为 35，维持旧手感）。
	en := snapshot.Tuning.Encounter
	probability := nodeEncounterBase(node, en.NodeDefaultChance)
	probability += state.Heat
	if state.Mode == runModeEvacuating {
		probability -= en.NodeEvacModifier
	}
	probability = maxInt(probability, en.NodeChanceMin)
	probability = minInt(probability, en.NodeChanceMax)
	if !forced && rng.Intn(100) >= probability {
		*state.Lines = append(*state.Lines, "  未遭遇敌人，安全通过")
		return Enemy{}, false, false, false, nil
	}
	enemy, ok := snapshot.Enemies[enemyID]
	if !ok {
		return Enemy{}, false, false, false, fmt.Errorf("读取敌人 %s", enemyID)
	}
	enemyWeapon, ok := snapshot.Weapons[enemy.WeaponID]
	if !ok {
		return Enemy{}, false, false, false, fmt.Errorf("读取敌方武器 %s", enemy.WeaponID)
	}
	enemyArmor, ok := snapshot.Armors[enemy.ArmorID]
	if !ok {
		return Enemy{}, false, false, false, fmt.Errorf("读取敌方护甲 %s", enemy.ArmorID)
	}
	var enemyAmmo Ammo
	if enemyWeapon.AmmoPerRound > 0 {
		enemyAmmo, ok = snapshot.Ammos[enemy.AmmoID]
		if !ok {
			return Enemy{}, false, false, false, fmt.Errorf("读取敌方弹药 %s", enemy.AmmoID)
		}
		if enemyAmmo.CaliberID != enemyWeapon.CaliberID {
			return Enemy{}, false, false, false, fmt.Errorf("敌方弹药 %s 与武器口径不匹配", enemy.AmmoID)
		}
	}
	enemyActor := buildEnemyActor(enemy, enemyWeapon, enemyArmor, enemyAmmo)
	policy := stylePolicy(state.Styles, state.Style)
	enemyPolicy := stylePolicy(state.Styles, ActionStyleBalanced)
	approach := policy.PatrolApproach
	if encounterRole != "" {
		approach = encounterApproach(policy, encounterRole)
	}
	*state.Lines = append(*state.Lines, fmt.Sprintf("  %s风格对%s选择%s", policy.Label, enemy.Name, approach))
	forceEscape := state.Mode == runModeEvacuating && state.EvacuationReason == "carry_full"
	if forceEscape {
		*state.Lines = append(*state.Lines, "  当前因负重撤离，战斗内优先尝试脱离")
	}
	// 交战前从携带弹药池中选中本次主弹（等级最高者），战后写回消耗。
	state.syncActiveAmmo(snapshot)
	result := simulateEncounter(snapshot.Tuning, state.Player, &enemyActor, node.Distance, state.Heat, state.hasItem("smoke"), approach, policy, enemyPolicy, forceEscape, rng)
	if result.BypassedWithoutFight {
		state.creditSkill("stealth")
	}
	if result.PlayerSpottedFirst {
		state.creditSkill("perception")
	}
	state.writeBackActiveAmmo()
	*state.Lines = append(*state.Lines, result.Lines[1:]...)
	battleStartedAt := state.DurationSec
	for _, battleEvent := range result.Trace {
		state.addTrace(battleEvent.Type, battleStartedAt+battleEvent.OffsetSec, node.ID, enemy.ID, map[string]interface{}{
			"round": battleEvent.Round, "actor": battleEvent.Actor, "target": battleEvent.Target,
			"success": battleEvent.Success, "message": battleEvent.Message, "winner": battleEvent.Winner,
			"playerHp": battleEvent.PlayerHP, "playerMaxHp": state.Player.MaxHP, "playerStress": battleEvent.PlayerStress,
			"enemyHp": battleEvent.EnemyHP, "enemyMaxHp": enemy.HP, "enemyStress": battleEvent.EnemyStress,
			"playerAmmo": battleEvent.PlayerAmmo, "enemyAmmo": battleEvent.EnemyAmmo,
			"playerArmorDurability": battleEvent.PlayerArmorDurability, "enemyArmorDurability": battleEvent.EnemyArmorDurability,
			"hit": battleEvent.Hit, "hitRate": battleEvent.HitRate, "hitRoll": battleEvent.HitRoll,
			"hitLocation": battleEvent.HitLocation, "covered": battleEvent.Covered, "penetrated": battleEvent.Penetrated,
			"ammoId": battleEvent.AmmoID, "ammoLevel": battleEvent.AmmoLevel,
			"armorLevel": battleEvent.ArmorLevel, "effectiveArmorLevel": battleEvent.EffectiveArmorLevel,
			"fleshDamage": battleEvent.FleshDamage, "healthDamage": battleEvent.HealthDamage,
			"armorDamage": battleEvent.ArmorDamage, "targetHp": battleEvent.TargetHP,
			"targetArmorDurability": battleEvent.TargetArmorDurability,
		})
	}
	state.DurationSec += result.DurationSec
	state.Heat += result.HeatAdd
	state.AmmoUsed += result.AmmoUsed
	if result.SmokeUsed {
		state.consumeItem("smoke")
	}
	state.Player.HP = result.PlayerHP
	state.Player.Stress = result.PlayerStress
	enemyDefeated := result.EnemyHP <= 0
	// 全敌对单位弹药掉落：击倒且武器有弹药时搜缴战后剩余弹药（探索模式、弹药足够一梭）。
	if enemyDefeated && state.Mode == runModeExploring && enemyWeapon.AmmoPerRound > 0 &&
		enemyActor.AmmoRounds >= enemyWeapon.AmmoPerRound && state.CollectAmmoDrop != nil {
		*state.Lines = append(*state.Lines, fmt.Sprintf("  %s 被击倒，弹药剩余 %d 发可以搜缴", enemy.Name, enemyActor.AmmoRounds))
		if err := state.CollectAmmoDrop(enemyAmmo.ID, enemyActor.AmmoRounds, "敌人弹药"); err != nil {
			return Enemy{}, false, false, false, err
		}
	}
	encounterCleared := enemyDefeated || result.Winner == "player_suppress" || (result.Winner == "escape" && !result.EscapeSuccess)
	return enemy, enemyDefeated, encounterCleared, true, nil
}

// encounterApproach 按敌人角色决定交战方式：巡逻用风格默认，守卫/精英直接接战。
func encounterApproach(policy StylePolicy, role string) string {
	if role == "patrol" || role == "" {
		return policy.PatrolApproach
	}
	if role == "guard" || role == "elite" {
		return EncounterApproachEngage
	}
	return EncounterApproachEngage
}

// evaluateAutomaticEvacuation 检查血量/压力/弹药/护甲/负重，任一触线即自动进入撤离。
func evaluateAutomaticEvacuation(snapshot ScenarioSnapshot, state *eventRunState, weapon Weapon) {
	if state.Player.HP <= 0 {
		return
	}
	policy := stylePolicy(state.Styles, state.Style)
	if state.Player.HP < state.Player.MaxHP*policy.HealthEvacRatio {
		state.beginEvacuation("health", false)
	}
	if state.Player.Stress >= state.Player.StressThreshold*policy.StressEvacRatio {
		state.beginEvacuation("stress", false)
	}
	if weapon.AmmoPerRound > 0 && !HasUsableCarriedAmmoStack(snapshot, weapon, state.AmmoStacks) {
		state.beginEvacuation("ammo", true)
	}
	if state.Player.ArmorDurability > 0 {
		state.Player.IgnoreBrokenArmorEscape = false
	}
	if state.Player.Armor.ProtectionLevel > 0 && state.Player.ArmorMaxDur > 0 && state.Player.ArmorDurability <= 0 && !state.Player.IgnoreBrokenArmorEscape {
		state.ArmorBrokenDuringRun = true
		state.beginEvacuation("armor", true)
	}
	if state.CarryBlocked || state.carryRatio() >= policy.CarryEvacRatio {
		state.beginEvacuation("carry_full", false)
	}
}

// startEvacuationEvents 首次触发撤离阶段的入场事件，幂等（只执行一次）。
func startEvacuationEvents(events *eventManager, state *eventRunState, rng *rand.Rand) error {
	if !state.EvacuationPending || state.EvacuationStarted {
		return nil
	}
	state.EvacuationPending = false
	state.EvacuationStarted = true
	return events.Trigger(state, eventPhaseEvacStart, rng)
}

// lootQuantity 累加战利品掉落的总数量。
func lootQuantity(loot []LootDrop) int {
	total := 0
	for _, drop := range loot {
		total += drop.Quantity
	}
	return total
}

// selectExtractedLoot 只有撤离成功才带回战利品，返回副本避免共享底层切片。
func selectExtractedLoot(result string, loot []LootDrop) []LootDrop {
	if result != "success" {
		return nil
	}
	return cloneLoot(loot)
}

// cloneLoot 浅拷贝战利品切片，防止后续修改影响原数据。
func cloneLoot(loot []LootDrop) []LootDrop {
	return append([]LootDrop(nil), loot...)
}

// nodeEncounterBase 返回节点的基础遇敌概率，未配置（<=0）时回落引擎默认值，用于概率计算与节点进入播报。
func nodeEncounterBase(node Node, defaultChance int) int {
	if node.EncounterChance <= 0 {
		return defaultChance
	}
	return node.EncounterChance
}

// extractionLabel 单局终态只有成功/失能，不再有紧急或部分撤离分支。
func extractionLabel(result string) string {
	return "撤离"
}

// minInt 返回 value 与 ceiling 中的较小值（最大值上限约束）。
func minInt(value, ceiling int) int {
	if value > ceiling {
		return ceiling
	}
	return value
}

// cloneItemStacks 浅拷贝物品堆叠切片。
func cloneItemStacks(stacks []ItemStack) []ItemStack {
	return append([]ItemStack(nil), stacks...)
}

// joinStrings 用分隔符拼接字符串切片，空切片返回空串。
func joinStrings(values []string, separator string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		result += separator + value
	}
	return result
}
