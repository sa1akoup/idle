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

func (snapshot ScenarioSnapshot) LootItemsByCategory(category string, rng *rand.Rand) (LootItem, bool) {
	return chooseLootItem(snapshot.LootItems, category, rng)
}

func (snapshot ScenarioSnapshot) NodeContainerAssignmentsForNode(nodeID string) []NodeContainerAssignment {
	result := make([]NodeContainerAssignment, 0)
	for _, assignment := range snapshot.NodeContainerAssignments {
		if assignment.NodeID == nodeID {
			result = append(result, assignment)
		}
	}
	return result
}

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
	probability := 60 + state.Heat
	if state.Mode == runModeEvacuating {
		probability = 35 + state.Heat
	}
	probability = minInt(probability, 90)
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
	result := simulateEncounter(state.Player, &enemyActor, node.Distance, state.Heat, state.hasItem("smoke"), approach, policy, enemyPolicy, forceEscape, rng)
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
	encounterCleared := enemyDefeated || result.Winner == "player_suppress" || (result.Winner == "escape" && !result.EscapeSuccess)
	return enemy, enemyDefeated, encounterCleared, true, nil
}

func encounterApproach(policy StylePolicy, role string) string {
	if role == "patrol" || role == "" {
		return policy.PatrolApproach
	}
	if role == "guard" || role == "elite" {
		return EncounterApproachEngage
	}
	return EncounterApproachEngage
}

func evaluateAutomaticEvacuation(state *eventRunState, weapon Weapon) {
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
	if weapon.AmmoPerRound > 0 && state.Player.AmmoRounds < weapon.AmmoPerRound {
		state.beginEvacuation("ammo", true)
	}
	if state.Player.ArmorDurability <= 0 {
		state.beginEvacuation("armor", true)
	}
	if state.CarryBlocked || state.carryRatio() >= policy.CarryEvacRatio {
		state.beginEvacuation("carry_full", false)
	}
}

func startEvacuationEvents(events *eventManager, state *eventRunState, rng *rand.Rand) error {
	if !state.EvacuationPending || state.EvacuationStarted {
		return nil
	}
	state.EvacuationPending = false
	state.EvacuationStarted = true
	return events.Trigger(state, eventPhaseEvacStart, rng)
}

func lootQuantity(loot []LootDrop) int {
	total := 0
	for _, drop := range loot {
		total += drop.Quantity
	}
	return total
}

func selectExtractedLoot(result string, loot []LootDrop) []LootDrop {
	if result != "success" {
		return nil
	}
	return cloneLoot(loot)
}

func cloneLoot(loot []LootDrop) []LootDrop {
	return append([]LootDrop(nil), loot...)
}

// extractionLabel 单局终态只有成功/失能，不再有紧急或部分撤离分支。
func extractionLabel(result string) string {
	return "撤离"
}

func minInt(value, ceiling int) int {
	if value > ceiling {
		return ceiling
	}
	return value
}

func cloneItemStacks(stacks []ItemStack) []ItemStack {
	return append([]ItemStack(nil), stacks...)
}

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
