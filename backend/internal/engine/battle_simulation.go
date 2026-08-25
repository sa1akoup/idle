// 遭遇模拟：保留战斗轮次、脱离和报告生成的完整结算语义。
package engine

import (
	"fmt"
	"math/rand"
)

// simulateEncounter 保留原有五轮战斗语义，所有随机性只来自调用方传入的 RNG。
func simulateEncounter(player, enemy *BattleActor, distance string, heat int, hasSmoke bool, approach string, policy, enemyPolicy StylePolicy, forceEscape bool, rng *rand.Rand) BattleResult {
	lines := []string{fmt.Sprintf("遭遇 %s，距离:%s", enemy.Name, distance)}
	timelineSec := int64(0)
	newTrace := func(eventType TraceEventType, round int) BattleTrace {
		return BattleTrace{
			Type: eventType, OffsetSec: timelineSec, Round: round,
			PlayerHP: player.HP, PlayerStress: player.Stress, EnemyHP: enemy.HP, EnemyStress: enemy.Stress,
			PlayerAmmo: player.AmmoRounds, EnemyAmmo: enemy.AmmoRounds,
			PlayerArmorDurability: player.ArmorDurability, EnemyArmorDurability: enemy.ArmorDurability,
		}
	}
	started := newTrace(TraceBattleStarted, 0)
	started.Actor = player.Name
	started.Target = enemy.Name
	trace := []BattleTrace{started}
	finish := func(result BattleResult) BattleResult {
		timelineSec++
		finished := newTrace(TraceBattleFinished, result.Rounds)
		finished.Winner = result.Winner
		trace = append(trace, finished)
		result.PlayerHP = player.HP
		result.PlayerStress = player.Stress
		result.EnemyHP = enemy.HP
		result.EnemyStress = enemy.Stress
		result.DurationSec = timelineSec
		result.Trace = trace
		return result
	}
	appendEscape := func(round int, actor, target string, escape escapeResult) {
		timelineSec += 2
		event := newTrace(TraceBattleEscape, round)
		event.Actor = actor
		event.Target = target
		event.Success = escape.success
		event.Message = escape.msg
		trace = append(trace, event)
	}
	appendRound := func(round int) {
		timelineSec++
		trace = append(trace, newTrace(TraceBattleRound, round))
	}
	pRecon := calcRecon(player.PerceptionEff, 50, 0)
	eRecon := calcRecon(enemy.PerceptionEff, 40, 0)
	pConceal := calcConceal(player.StealthEff, int(player.Agility), float64(player.Armor.Conceal))
	eConceal := calcConceal(enemy.StealthEff, int(enemy.Agility), float64(enemy.Armor.Conceal))
	pFindProb := clamp(50+(pRecon-eConceal)*0.8, 10, 90)
	eFindProb := clamp(50+(eRecon-pConceal)*0.8, 10, 90)
	pFound := float64(rng.Intn(100)+1) <= pFindProb
	eFound := float64(rng.Intn(100)+1) <= eFindProb
	ambushPlayer, ambushEnemy := 0, 0
	if pFound && !eFound {
		lines = append(lines, fmt.Sprintf("感知判定成功，提前发现敌人 (%.0f%%)", pFindProb))
		ambushPlayer = 15
	} else if !pFound && eFound {
		lines = append(lines, "被敌人伏击，敌方首轮命中+15")
		ambushEnemy = 15
	} else if pFound && eFound {
		lines = append(lines, "双方同时发现，正常接敌")
	} else {
		lines = append(lines, "双方错过，巡逻擦肩")
	}

	if approach == EncounterApproachBypass {
		bypassVal := player.StealthEff*0.5 + player.Agility*0.15 + float64(player.Armor.Conceal)
		alertVal := enemy.PerceptionEff * 0.45
		bypassProb := clamp(50+(bypassVal-alertVal)*0.8, 10, 95)
		if float64(rng.Intn(100)+1) <= bypassProb {
			lines = append(lines, fmt.Sprintf("潜行绕行成功 (%.0f%%)，未触发交战", bypassProb))
			return finish(BattleResult{Lines: lines, Winner: "escape"})
		}
		lines = append(lines, fmt.Sprintf("绕行失败 (%.0f%%)，进入交战", bypassProb))
	} else if approach == EncounterApproachAmbush {
		if pFound {
			ambushPlayer += 10
			lines = append(lines, "行动风格选择伏击，取得主动接敌修正")
		} else {
			lines = append(lines, "行动风格选择伏击，但未能提前发现敌人，强制接战")
		}
	}

	distModPlayer := getDistMod(player.Weapon, distance)
	distModEnemy := getDistMod(enemy.Weapon, distance)
	if distModPlayer <= -100 {
		lines = append(lines, fmt.Sprintf("武器 %s 无法在 %s 距离攻击，尝试接近", player.Weapon.Name, distance))
		approachVal := player.Agility*0.35 + player.StealthEff*0.25 + 6 - player.Stress*0.15
		blockVal := enemy.PerceptionEff*0.3 + float64(enemy.Weapon.Suppress)*0.3
		approachProb := clamp(50+(approachVal-blockVal)*0.8, 10, 90)
		if float64(rng.Intn(100)+1) <= approachProb {
			lines = append(lines, "接近成功，距离缩短一级")
			distModPlayer = 0
		} else {
			lines = append(lines, "接近失败，敌方拦截攻击+10命中")
			ambushEnemy += 10
		}
	}

	pInit := calcInitiative(player.Agility, player.PerceptionEff, float64(player.Weapon.Ready), float64(player.Armor.Initiative), player.Stress, ambushPlayer)
	eInit := calcInitiative(enemy.Agility, enemy.PerceptionEff, float64(enemy.Weapon.Ready), float64(enemy.Armor.Initiative), enemy.Stress, ambushEnemy)
	playerFirst := pInit >= eInit
	if pInit == eInit {
		playerFirst = rng.Intn(2) == 0
	}
	first := "敌人"
	if playerFirst {
		first = "玩家"
	}
	lines = append(lines, fmt.Sprintf("先手判定 玩家%.1f vs 敌人%.1f -> %s先手", pInit, eInit, first))

	heatAdd, ammoUsed := 0, 0
	damageDealt, armorDamage := 0.0, 0.0
	performAttack := func(round int, attacker, defender *BattleActor, distanceModifier int, isPlayer bool) {
		result := attack(attacker, defender, distanceModifier, 0, rng, &lines)
		timelineSec += 2
		event := newTrace(TraceBattleAttack, round)
		event.Actor = attacker.Name
		event.Target = defender.Name
		event.Success = result.Hit
		event.Hit = result.Hit
		event.HitRate = result.HitRate
		event.HitRoll = result.HitRoll
		event.HitLocation = result.HitLocation
		event.Covered = result.Covered
		event.Penetrated = result.Penetrated
		event.AmmoID = result.AmmoID
		event.AmmoLevel = result.AmmoLevel
		event.ArmorLevel = result.ArmorLevel
		event.EffectiveArmorLevel = result.EffectiveArmorLevel
		event.FleshDamage = result.FleshDamage
		event.HealthDamage = result.HealthDamage
		event.ArmorDamage = result.ArmorDamage
		event.TargetHP = result.TargetHP
		event.TargetArmorDurability = result.TargetArmorDurability
		trace = append(trace, event)
		if isPlayer {
			heatAdd += attacker.Weapon.Noise / 10
			ammoUsed += result.AmmoSpent
			damageDealt += result.HealthDamage
			armorDamage += result.ArmorDamage
		}
	}
	for round := 1; round <= 5; round++ {
		if player.HP <= 0 || enemy.HP <= 0 {
			break
		}
		if forceEscape || shouldEscape(*player, policy) {
			escape := tryEscape(player, enemy, hasSmoke, rng)
			lines = append(lines, fmt.Sprintf("第%d轮 玩家尝试脱离 -> %s", round, escape.msg))
			appendEscape(round, player.Name, enemy.Name, escape)
			if escape.success {
				return finish(BattleResult{Lines: lines, Winner: "escape", HeatAdd: heatAdd, AmmoUsed: ammoUsed, DamageDealt: damageDealt, ArmorDamage: armorDamage, Rounds: round, EscapeSuccess: true, SmokeUsed: escape.usedSmoke})
			}
			if enemy.HP > 0 && canAttack(*enemy) {
				performAttack(round, enemy, player, distModEnemy+10, false)
				heatAdd += enemy.Weapon.Noise / 10
			}
			appendRound(round)
			continue
		}
		if shouldEscape(*enemy, enemyPolicy) {
			escape := tryEscape(enemy, player, false, rng)
			lines = append(lines, fmt.Sprintf("第%d轮 敌人尝试脱离 -> %s", round, escape.msg))
			appendEscape(round, enemy.Name, player.Name, escape)
			if escape.success {
				lines = append(lines, "敌人压退，可通过节点")
				return finish(BattleResult{Lines: lines, Winner: "player_suppress", HeatAdd: heatAdd, AmmoUsed: ammoUsed, DamageDealt: damageDealt, ArmorDamage: armorDamage, Rounds: round})
			}
		}

		type attacker struct {
			atk      *BattleActor
			def      *BattleActor
			distMod  int
			isPlayer bool
		}
		attackers := []attacker{{player, enemy, distModPlayer, true}, {enemy, player, distModEnemy, false}}
		if !playerFirst {
			attackers = []attacker{{enemy, player, distModEnemy, false}, {player, enemy, distModPlayer, true}}
		}
		for _, current := range attackers {
			if current.atk.HP <= 0 || current.def.HP <= 0 {
				break
			}
			if current.atk.Stress >= current.atk.StressThreshold {
				continue
			}
			if !canAttack(*current.atk) {
				lines = append(lines, fmt.Sprintf("%s 弹药耗尽", current.atk.Name))
				continue
			}
			performAttack(round, current.atk, current.def, current.distMod, current.isPlayer)
			if current.def.HP <= 0 {
				lines = append(lines, fmt.Sprintf("%s 被击倒", current.def.Name))
				break
			}
		}
		if round == 5 {
			lines = append(lines, "5轮结束，进入强制脱离判定")
			escape := tryEscape(player, enemy, hasSmoke, rng)
			lines = append(lines, escape.msg)
			appendEscape(round, player.Name, enemy.Name, escape)
			if escape.success {
				return finish(BattleResult{Lines: lines, Winner: "escape", HeatAdd: heatAdd, AmmoUsed: ammoUsed, DamageDealt: damageDealt, ArmorDamage: armorDamage, Rounds: 5, EscapeSuccess: true, SmokeUsed: escape.usedSmoke})
			}
		}
		appendRound(round)
	}

	winner := "draw"
	if player.HP <= 0 {
		winner = "enemy"
	} else if enemy.HP <= 0 || player.HP > enemy.HP {
		winner = "player"
	}
	return finish(BattleResult{Lines: lines, Winner: winner, HeatAdd: heatAdd, AmmoUsed: ammoUsed, DamageDealt: damageDealt, ArmorDamage: armorDamage, Rounds: 5})
}
