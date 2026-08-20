package service

import (
	"fmt"
	"math"
	"math/rand"

	"idle/internal/models"
)

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// BattleActor 战斗单位
type BattleActor struct {
	Name            string
	MaxHP           float64
	HP              float64
	Stress          float64
	StressThreshold float64
	Weapon          models.WeaponDef
	Armor           models.ArmorDef
	ArmorDurability float64
	ArmorMaxDur     float64
	Evasion         float64
	Mobility        float64
	PerceptionEff   float64
	StealthEff      float64
	Agility         float64
	ResistEff       float64
	WeaponControl   float64
	Ammo            int
}

// BattleResult 单次遭遇结果
type BattleResult struct {
	Lines         []string
	Winner        string // player/enemy/draw/escape
	HeatAdd       int
	AmmoUsed      int
	DamageDealt   float64
	ArmorDamage   float64
	PlayerHP      float64
	PlayerStress  float64
	EnemyHP       float64
	EnemyStress   float64
	Rounds        int
	EscapeSuccess bool
	SmokeUsed     bool
}

// Calc helpers per spec
func CalcMaxHP(strength int) float64 {
	return clamp(100+float64(strength-50)*0.2, 90, 110)
}
func CalcStressThreshold(resistEff float64) float64 {
	return 70 + resistEff*0.2
}
func CalcEvasion(agility float64, armorMobility float64) float64 {
	return clamp(agility*0.12+armorMobility, 0, 35)
}
func CalcInitiative(agility float64, percepEff float64, weaponReady float64, armorInit float64, stress float64, ambush int) float64 {
	return agility*0.35 + percepEff*0.35 + weaponReady + armorInit - stress*0.2 + float64(ambush)
}
func CalcRecon(percepEff float64, intellect int, equipBonus float64) float64 {
	return percepEff*0.7 + float64(intellect)*0.1 + equipBonus
}
func CalcConceal(stealthEff float64, agility int, armorConceal float64) float64 {
	return stealthEff*0.7 + float64(agility)*0.1 + armorConceal
}

// WeaponControl attribute part
func CalcWeaponAttrControl(cat string, c models.Character, percepEff, resistEff float64) float64 {
	switch cat {
	case "melee":
		return float64(c.Strength)*0.45 + float64(c.Agility)*0.30 + resistEff*0.25
	case "pistol":
		return float64(c.Agility)*0.45 + percepEff*0.35 + resistEff*0.20
	case "smg":
		return float64(c.Agility)*0.35 + float64(c.Strength)*0.20 + percepEff*0.20 + resistEff*0.25
	case "shotgun":
		return float64(c.Strength)*0.40 + percepEff*0.25 + resistEff*0.35
	case "rifle":
		return float64(c.Strength)*0.20 + float64(c.Agility)*0.25 + percepEff*0.30 + resistEff*0.25
	case "sniper":
		return float64(c.Intellect)*0.15 + float64(c.Agility)*0.10 + percepEff*0.45 + resistEff*0.30
	}
	return 50
}
func GetProf(c models.Character, cat string) int {
	switch cat {
	case "melee":
		return c.MeleeProf
	case "pistol":
		return c.PistolProf
	case "smg":
		return c.SMGProf
	case "shotgun":
		return c.ShotgunProf
	case "rifle":
		return c.RifleProf
	case "sniper":
		return c.SniperProf
	}
	return 0
}
func FinalWeaponControl(attrControl float64, prof int) float64 {
	return attrControl*0.7 + float64(prof)*0.3
}

func BuildPlayerActor(c models.Character, w models.WeaponDef, a models.ArmorDef) BattleActor {
	percepEff := models.EffectiveSkill(c.Perception, c.Agility)
	stealthEff := models.EffectiveSkill(c.Stealth, c.Agility)
	resistEff := models.EffectiveSkill(c.Resist, c.Strength)
	attrCtrl := CalcWeaponAttrControl(w.Category, c, percepEff, resistEff)
	prof := GetProf(c, w.Category)
	finalCtrl := FinalWeaponControl(attrCtrl, prof)
	return BattleActor{
		Name:            c.Name,
		MaxHP:           CalcMaxHP(c.Strength),
		HP:              CalcMaxHP(c.Strength),
		Stress:          float64(c.Stress),
		StressThreshold: CalcStressThreshold(resistEff),
		Weapon:          w,
		Armor:           a,
		ArmorDurability: float64(a.MaxDurability),
		ArmorMaxDur:     float64(a.MaxDurability),
		Evasion:         CalcEvasion(float64(c.Agility), float64(a.Mobility)),
		Mobility:        float64(a.Mobility),
		PerceptionEff:   percepEff,
		StealthEff:      stealthEff,
		Agility:         float64(c.Agility),
		ResistEff:       resistEff,
		WeaponControl:   finalCtrl,
		Ammo:            30,
	}
}

func BuildEnemyActor(e models.EnemyDef, w models.WeaponDef, a models.ArmorDef) BattleActor {
	th := float64(e.StressThreshold)
	if th == 0 {
		th = 60
	}
	return BattleActor{
		Name:            e.Name,
		MaxHP:           float64(e.HP),
		HP:              float64(e.HP),
		Stress:          0,
		StressThreshold: th,
		Weapon:          w,
		Armor:           a,
		ArmorDurability: float64(a.MaxDurability),
		ArmorMaxDur:     float64(a.MaxDurability),
		Evasion:         float64(e.Evasion),
		Mobility:        float64(e.Mobility),
		PerceptionEff:   float64(e.Perception),
		StealthEff:      float64(e.Stealth),
		Agility:         float64(e.Agility),
		ResistEff:       40,
		WeaponControl:   55,
		Ammo:            30,
	}
}

// SimulateEncounter 完整8阶段；战术选择由上层策略决定，战斗函数只处理判定和结算。
func SimulateEncounter(player *BattleActor, enemy *BattleActor, distance string, heat int, hasSmoke bool, approach string, policy StylePolicy, forceEscape bool, rng *rand.Rand) BattleResult {
	var lines []string
	lines = append(lines, fmt.Sprintf("遭遇 %s，距离:%s", enemy.Name, distance))

	// 阶段二 发现
	pRecon := CalcRecon(player.PerceptionEff, 50, 0)
	eRecon := CalcRecon(enemy.PerceptionEff, 40, 0)
	pConceal := CalcConceal(player.StealthEff, int(player.Agility), float64(player.Armor.Conceal))
	eConceal := CalcConceal(enemy.StealthEff, int(enemy.Agility), float64(enemy.Armor.Conceal))
	pFindProb := clamp(50+(pRecon-eConceal)*0.8, 10, 90)
	eFindProb := clamp(50+(eRecon-pConceal)*0.8, 10, 90)
	pFound := float64(rng.Intn(100)+1) <= pFindProb
	eFound := float64(rng.Intn(100)+1) <= eFindProb
	ambushPlayer := 0
	ambushEnemy := 0
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
		// 守卫仍会接敌，这里简化为继续
	}

	// 阶段三 战术选择：只有上层明确选择绕行时才进行绕行判定。
	canBypass := approach == EncounterApproachBypass
	if canBypass {
		bypassVal := player.StealthEff*0.5 + player.Agility*0.15 + float64(player.Armor.Conceal)
		alertVal := enemy.PerceptionEff * 0.45
		bypassProb := clamp(50+(bypassVal-alertVal)*0.8, 10, 95)
		if float64(rng.Intn(100)+1) <= bypassProb {
			lines = append(lines, fmt.Sprintf("潜行绕行成功 (%.0f%%)，未触发交战", bypassProb))
			return BattleResult{Lines: lines, Winner: "escape", HeatAdd: 0, AmmoUsed: 0, PlayerHP: player.HP, PlayerStress: player.Stress, EnemyHP: enemy.HP, EnemyStress: enemy.Stress}
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

	// 距离修正查表
	distModPlayer := getDistMod(player.Weapon, distance)
	distModEnemy := getDistMod(enemy.Weapon, distance)
	if distModPlayer <= -100 {
		lines = append(lines, fmt.Sprintf("武器 %s 无法在 %s 距离攻击，尝试接近", player.Weapon.Name, distance))
		// 简化接近判定
		approachVal := player.Agility*0.35 + player.StealthEff*0.25 + float64(60)*0.1 - player.Stress*0.15
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

	// 阶段四 先手
	pInit := CalcInitiative(player.Agility, player.PerceptionEff, float64(player.Weapon.Ready), float64(player.Armor.Initiative), player.Stress, ambushPlayer)
	eInit := CalcInitiative(enemy.Agility, enemy.PerceptionEff, float64(enemy.Weapon.Ready), float64(enemy.Armor.Initiative), enemy.Stress, ambushEnemy)
	playerFirst := pInit >= eInit
	if pInit == eInit {
		playerFirst = rng.Intn(2) == 0
	}
	lines = append(lines, fmt.Sprintf("先手判定 玩家%.1f vs 敌人%.1f -> %s先手", pInit, eInit, map[bool]string{true: "玩家", false: "敌人"}[playerFirst]))

	heatAdd := 0
	ammoUsed := 0
	// 5轮
	for round := 1; round <= 5; round++ {
		if player.HP <= 0 || enemy.HP <= 0 {
			break
		}
		// 检查脱离
		if forceEscape || shouldEscape(player, policy) {
			esc := tryEscape(player, enemy, hasSmoke, rng)
			lines = append(lines, fmt.Sprintf("第%d轮 玩家尝试脱离 -> %s", round, esc.msg))
			if esc.success {
				return BattleResult{Lines: lines, Winner: "escape", HeatAdd: heatAdd, AmmoUsed: ammoUsed, PlayerHP: player.HP, PlayerStress: player.Stress, EnemyHP: enemy.HP, EnemyStress: enemy.Stress, Rounds: round, EscapeSuccess: true, SmokeUsed: esc.usedSmoke}
			}
			// 失败追击
			if enemy.HP > 0 {
				hr := attack(enemy, player, distModEnemy+10, 0, rng, &lines, false)
				heatAdd += enemy.Weapon.Noise / 10
				_ = hr
			}
			continue
		}
		if shouldEscape(enemy, actionStylePolicy(ActionStyleBalanced)) {
			esc := tryEscape(enemy, player, false, rng)
			lines = append(lines, fmt.Sprintf("第%d轮 敌人尝试脱离 -> %s", round, esc.msg))
			if esc.success {
				lines = append(lines, "敌人压退，可通过节点")
				return BattleResult{Lines: lines, Winner: "player_suppress", HeatAdd: heatAdd, AmmoUsed: ammoUsed, PlayerHP: player.HP, PlayerStress: player.Stress, EnemyHP: enemy.HP, EnemyStress: enemy.Stress, Rounds: round}
			}
		}

		// 一轮攻击顺序
		attackers := []struct {
			atk      *BattleActor
			def      *BattleActor
			distMod  int
			isPlayer bool
		}{}
		if playerFirst {
			attackers = append(attackers, struct {
				atk      *BattleActor
				def      *BattleActor
				distMod  int
				isPlayer bool
			}{player, enemy, distModPlayer, true})
			attackers = append(attackers, struct {
				atk      *BattleActor
				def      *BattleActor
				distMod  int
				isPlayer bool
			}{enemy, player, distModEnemy, false})
		} else {
			attackers = append(attackers, struct {
				atk      *BattleActor
				def      *BattleActor
				distMod  int
				isPlayer bool
			}{enemy, player, distModEnemy, false})
			attackers = append(attackers, struct {
				atk      *BattleActor
				def      *BattleActor
				distMod  int
				isPlayer bool
			}{player, enemy, distModPlayer, true})
		}
		for _, at := range attackers {
			if at.atk.HP <= 0 || at.def.HP <= 0 {
				break
			}
			if at.atk.Stress >= at.atk.StressThreshold {
				continue // 崩溃不攻击
			}
			if at.atk.Ammo <= 0 && at.atk.Weapon.AmmoPerRound > 0 {
				lines = append(lines, fmt.Sprintf("%s 弹药耗尽", at.atk.Name))
				continue
			}
			// 闪光弹首轮加成
			ambush := 0
			if round == 1 && at.isPlayer {
				// 简化：若携带闪光视为已用，这里不追踪
			}
			hitRes := attack(at.atk, at.def, at.distMod, 0, rng, &lines, at.isPlayer)
			_ = ambush
			_ = hitRes
			if at.isPlayer {
				heatAdd += at.atk.Weapon.Noise / 10
				ammoUsed += at.atk.Weapon.AmmoPerRound
				if at.atk.Weapon.AmmoPerRound > 0 {
					at.atk.Ammo -= at.atk.Weapon.AmmoPerRound
					if at.atk.Ammo < 0 {
						at.atk.Ammo = 0
					}
				}
			} else {
				// 敌人也耗弹但不计入玩家heat/ ammo
			}
			if at.def.HP <= 0 {
				lines = append(lines, fmt.Sprintf("%s 被击倒", at.def.Name))
				break
			}
		}
		if round == 5 {
			lines = append(lines, "5轮结束，进入强制脱离判定")
			esc := tryEscape(player, enemy, hasSmoke, rng)
			lines = append(lines, esc.msg)
			if esc.success {
				return BattleResult{Lines: lines, Winner: "escape", HeatAdd: heatAdd, AmmoUsed: ammoUsed, PlayerHP: player.HP, PlayerStress: player.Stress, EnemyHP: enemy.HP, EnemyStress: enemy.Stress, Rounds: 5, EscapeSuccess: true, SmokeUsed: esc.usedSmoke}
			}
		}
	}

	winner := "draw"
	if player.HP <= 0 {
		winner = "enemy"
	} else if enemy.HP <= 0 {
		winner = "player"
	} else if player.HP > enemy.HP {
		winner = "player"
	}
	return BattleResult{Lines: lines, Winner: winner, HeatAdd: heatAdd, AmmoUsed: ammoUsed, PlayerHP: player.HP, PlayerStress: player.Stress, EnemyHP: enemy.HP, EnemyStress: enemy.Stress, Rounds: 5}
}

func getDistMod(w models.WeaponDef, dist string) int {
	switch dist {
	case "close":
		return w.CloseMod
	case "mid":
		return w.MidMod
	case "far":
		return w.FarMod
	}
	return 0
}

func shouldEscape(a *BattleActor, policy StylePolicy) bool {
	if a.HP <= 0 {
		return false
	}
	if a.HP < a.MaxHP*policy.HealthEvacRatio {
		return true
	}
	if a.Stress >= a.StressThreshold*policy.StressEvacRatio {
		return true
	}
	if a.Ammo <= 0 && a.Weapon.AmmoPerRound > 0 {
		return true
	}
	if a.ArmorDurability <= 0 {
		return true
	}
	return false
}

type escapeRes struct {
	success   bool
	msg       string
	usedSmoke bool
}

func tryEscape(actor, chaser *BattleActor, hasSmoke bool, rng *rand.Rand) escapeRes {
	escapeVal := actor.StealthEff*0.35 + actor.Agility*0.25 + float64(actor.Armor.Escape) - 0 // 伤势省略
	chaseVal := chaser.PerceptionEff*0.3 + float64(chaser.Weapon.Suppress)*0.3 + chaser.Mobility
	prob := clamp(50+(escapeVal-chaseVal)*0.8, 10, 95)
	success := float64(rng.Intn(100)+1) <= prob
	msg := fmt.Sprintf("脱离判定 %.0f%% -> %s", prob, map[bool]string{true: "成功", false: "失败"}[success])
	if !success && hasSmoke {
		msg += " 烟雾弹自动消耗，重判成功"
		return escapeRes{success: true, msg: msg, usedSmoke: true}
	}
	return escapeRes{success: success, msg: msg}
}

func attack(atk, def *BattleActor, distMod int, ambush int, rng *rand.Rand, lines *[]string, isPlayer bool) bool {
	hitRate := clamp(float64(atk.Weapon.Hit)+(atk.WeaponControl-50)*0.4+float64(distMod+ambush)-def.Evasion-atk.Stress*0.25, 5, 95)
	roll := rng.Intn(100) + 1
	hit := float64(roll) <= hitRate
	*lines = append(*lines, fmt.Sprintf("%s 攻击 %s 命中率%.1f%% 掷%d -> %s", atk.Name, def.Name, hitRate, roll, map[bool]string{true: "命中", false: "未命中"}[hit]))
	// 压力
	hitCoeff := 0.5
	dmgForStress := 0.0
	if hit {
		hitCoeff = 1.0
		// 护甲
		coverRoll := rng.Intn(100) + 1
		covered := float64(coverRoll) <= float64(def.Armor.Coverage)
		var finalDmg float64
		if covered {
			durRate := def.ArmorDurability / def.ArmorMaxDur
			if def.ArmorMaxDur == 0 {
				durRate = 1
			}
			actualProtect := float64(def.Armor.Protect) * (0.5 + durRate*0.5)
			reduceRate := actualProtect / (actualProtect + float64(atk.Weapon.Penetration))
			if reduceRate > 0.8 {
				reduceRate = 0.8
			}
			randFloat := 0.9 + rng.Float64()*0.2
			finalDmg = float64(atk.Weapon.Damage) * randFloat * (1 - reduceRate)
			// 耐久损失
			penRatio := float64(atk.Weapon.Penetration) / (float64(atk.Weapon.Penetration) + actualProtect)
			durLoss := float64(atk.Weapon.Damage) * (0.15 + penRatio*0.35)
			def.ArmorDurability -= durLoss
			if def.ArmorDurability < 0 {
				def.ArmorDurability = 0
			}
			*lines = append(*lines, fmt.Sprintf("  命中护甲 覆盖%d≤%d 减伤%.0f%% 伤害%.1f 耐久-%.1f", coverRoll, def.Armor.Coverage, reduceRate*100, finalDmg, durLoss))
		} else {
			randFloat := 0.9 + rng.Float64()*0.2
			finalDmg = float64(atk.Weapon.Damage) * randFloat
			*lines = append(*lines, fmt.Sprintf("  命中未防护区 伤害%.1f", finalDmg))
		}
		def.HP -= finalDmg
		if def.HP < 0 {
			def.HP = 0
		}
		dmgForStress = finalDmg
		// 幸运保护省略
	}
	// 压力增加
	suppressCoeff := 1.0
	if atk.Weapon.Category == "melee" && hit {
		suppressCoeff = 1.0
	}
	antiSuppress := float64(def.Armor.AntiSuppress)
	stressAdd := math.Max(1, float64(atk.Weapon.Suppress)*suppressCoeff*hitCoeff*(1-def.ResistEff*0.005)+dmgForStress*0.25-antiSuppress)
	def.Stress += stressAdd
	if def.Stress > def.StressThreshold {
		def.Stress = def.StressThreshold
	}
	*lines = append(*lines, fmt.Sprintf("  压力+%.1f (当前%.1f/%.0f)", stressAdd, def.Stress, def.StressThreshold))
	return hit
}
