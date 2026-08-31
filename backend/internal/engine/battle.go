// 纯战斗模块：使用 DTO 和调用方 RNG 完成遭遇、伤害、脱离与弹药计算。
package engine

import (
	"fmt"
	"math"
	"math/rand"
)

type BattleActor struct {
	Name            string
	MaxHP           float64
	HP              float64
	Stress          float64
	StressThreshold float64
	Weapon          Weapon
	Armor           Armor
	ArmorDurability float64
	ArmorMaxDur     float64
	Evasion         float64
	Mobility        float64
	PerceptionEff   float64
	StealthEff      float64
	Agility         float64
	ResistEff       float64
	WeaponControl   float64
	Ammo            Ammo
	AmmoRounds      int
}

type BattleResult struct {
	Lines         []string
	Trace         []BattleTrace
	Winner        string
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
	DurationSec   int64
}

type BattleTrace struct {
	Type                  TraceEventType
	OffsetSec             int64
	Round                 int
	Actor                 string
	Target                string
	Success               bool
	Message               string
	Winner                string
	PlayerHP              float64
	PlayerStress          float64
	EnemyHP               float64
	EnemyStress           float64
	PlayerAmmo            int
	EnemyAmmo             int
	PlayerArmorDurability float64
	EnemyArmorDurability  float64
	Hit                   bool
	HitRate               float64
	HitRoll               int
	HitLocation           string
	Covered               bool
	Penetrated            bool
	AmmoID                string
	AmmoLevel             int
	ArmorLevel            int
	EffectiveArmorLevel   int
	FleshDamage           float64
	HealthDamage          float64
	ArmorDamage           float64
	TargetHP              float64
	TargetArmorDurability float64
}

// AttackResult 是单次攻击的结构化结算，供报告、实时事件与测试共用。
type AttackResult struct {
	Hit                   bool
	HitRate               float64
	HitRoll               int
	HitLocation           string
	Covered               bool
	Penetrated            bool
	AmmoID                string
	AmmoLevel             int
	ArmorLevel            int
	EffectiveArmorLevel   int
	FleshDamage           float64
	HealthDamage          float64
	ArmorDamage           float64
	TargetHP              float64
	TargetArmorDurability float64
	AmmoSpent             int
}

// EffectiveSkill 计算训练技能与主属性的有效技能值。
func EffectiveSkill(train, mainAttr int) float64 {
	return float64(train)*0.75 + float64(mainAttr)*0.25
}

// CalcMaxHP 根据力量计算角色最大生命值，并限制在 90-110。
func CalcMaxHP(strength int) float64 {
	return clamp(100+float64(strength-50)*0.2, 90, 110)
}

// CalcStressThreshold 根据有效抗压计算压力阈值。
func CalcStressThreshold(resistEff float64) float64 {
	return 70 + resistEff*0.2
}

func calcEvasion(agility float64, armorMobility float64) float64 {
	return clamp(agility*0.12+armorMobility, 0, 35)
}

func calcInitiative(agility, percepEff, weaponReady, armorInit, stress float64, ambush int) float64 {
	return agility*0.35 + percepEff*0.35 + weaponReady + armorInit - stress*0.2 + float64(ambush)
}

func calcRecon(percepEff float64, intellect int, equipBonus float64) float64 {
	return percepEff*0.7 + float64(intellect)*0.1 + equipBonus
}

func calcConceal(stealthEff float64, agility int, armorConceal float64) float64 {
	return stealthEff*0.7 + float64(agility)*0.1 + armorConceal
}

func calcWeaponAttrControl(category string, character CharacterState, percepEff, resistEff float64) float64 {
	switch category {
	case "melee":
		return float64(character.Strength)*0.45 + float64(character.Agility)*0.30 + resistEff*0.25
	case "pistol":
		return float64(character.Agility)*0.45 + percepEff*0.35 + resistEff*0.20
	case "smg":
		return float64(character.Agility)*0.35 + float64(character.Strength)*0.20 + percepEff*0.20 + resistEff*0.25
	case "shotgun":
		return float64(character.Strength)*0.40 + percepEff*0.25 + resistEff*0.35
	case "rifle":
		return float64(character.Strength)*0.20 + float64(character.Agility)*0.25 + percepEff*0.30 + resistEff*0.25
	case "sniper":
		return float64(character.Intellect)*0.15 + float64(character.Agility)*0.10 + percepEff*0.45 + resistEff*0.30
	default:
		return 50
	}
}

func getProf(character CharacterState, category string) int {
	switch category {
	case "melee":
		return character.MeleeProf
	case "pistol":
		return character.PistolProf
	case "smg":
		return character.SMGProf
	case "shotgun":
		return character.ShotgunProf
	case "rifle":
		return character.RifleProf
	case "sniper":
		return character.SniperProf
	default:
		return 0
	}
}

func finalWeaponControl(attrControl float64, prof int) float64 {
	return attrControl*0.7 + float64(prof)*0.3
}

func buildPlayerActor(character CharacterState, weapon Weapon, armor Armor, armorDurability int, ammo Ammo, ammoRounds int) BattleActor {
	percepEff := EffectiveSkill(character.Perception, character.Agility)
	stealthEff := EffectiveSkill(character.Stealth, character.Agility)
	resistEff := EffectiveSkill(character.Resist, character.Strength)
	attrControl := calcWeaponAttrControl(weapon.Category, character, percepEff, resistEff)
	maxHP := CalcMaxHP(character.Strength)
	hp := clamp(character.HP, 0, maxHP)
	return BattleActor{
		Name: character.Name, MaxHP: maxHP, HP: hp,
		Stress: float64(character.Stress), StressThreshold: CalcStressThreshold(resistEff), Weapon: weapon, Armor: armor,
		ArmorDurability: float64(armorDurability), ArmorMaxDur: float64(armor.MaxDurability),
		Evasion: calcEvasion(float64(character.Agility), float64(armor.Mobility)), Mobility: float64(armor.Mobility),
		PerceptionEff: percepEff, StealthEff: stealthEff, Agility: float64(character.Agility), ResistEff: resistEff,
		WeaponControl: finalWeaponControl(attrControl, getProf(character, weapon.Category)), Ammo: ammo, AmmoRounds: ammoRounds,
	}
}

func buildEnemyActor(enemy Enemy, weapon Weapon, armor Armor, ammo Ammo) BattleActor {
	threshold := float64(enemy.StressThreshold)
	if threshold == 0 {
		threshold = 60
	}
	return BattleActor{
		Name: enemy.Name, MaxHP: float64(enemy.HP), HP: float64(enemy.HP), StressThreshold: threshold,
		Weapon: weapon, Armor: armor, ArmorDurability: float64(armor.MaxDurability), ArmorMaxDur: float64(armor.MaxDurability),
		Evasion: float64(enemy.Evasion), Mobility: float64(enemy.Mobility), PerceptionEff: float64(enemy.Perception),
		StealthEff: float64(enemy.Stealth), Agility: float64(enemy.Agility), ResistEff: 40, WeaponControl: 55, Ammo: ammo, AmmoRounds: enemy.AmmoRounds,
	}
}

func getDistMod(weapon Weapon, distance string) int {
	switch distance {
	case "close":
		return weapon.CloseMod
	case "mid":
		return weapon.MidMod
	case "far":
		return weapon.FarMod
	default:
		return 0
	}
}

func shouldEscape(actor BattleActor, policy StylePolicy) bool {
	if actor.HP <= 0 {
		return false
	}
	return actor.HP < actor.MaxHP*policy.HealthEvacRatio ||
		actor.Stress >= actor.StressThreshold*policy.StressEvacRatio ||
		(actor.Weapon.AmmoPerRound > 0 && actor.AmmoRounds < actor.Weapon.AmmoPerRound) ||
		(actor.Armor.ProtectionLevel > 0 && actor.ArmorMaxDur > 0 && actor.ArmorDurability <= 0)
}

func canAttack(actor BattleActor) bool {
	return actor.Weapon.AmmoPerRound <= 0 || actor.AmmoRounds >= actor.Weapon.AmmoPerRound
}

type escapeResult struct {
	success   bool
	msg       string
	usedSmoke bool
}

func tryEscape(actor, chaser *BattleActor, hasSmoke bool, rng *rand.Rand) escapeResult {
	escapeVal := actor.StealthEff*0.35 + actor.Agility*0.25 + float64(actor.Armor.Escape)
	chaseVal := chaser.PerceptionEff*0.3 + float64(chaser.Weapon.Suppress)*0.3 + chaser.Mobility
	probability := clamp(50+(escapeVal-chaseVal)*0.8, 10, 95)
	success := float64(rng.Intn(100)+1) <= probability
	status := "失败"
	if success {
		status = "成功"
	}
	message := fmt.Sprintf("脱离判定 %.0f%% -> %s", probability, status)
	if !success && hasSmoke {
		return escapeResult{success: true, msg: message + " 烟雾弹自动消耗，重判成功", usedSmoke: true}
	}
	return escapeResult{success: success, msg: message}
}

func attack(attacker, defender *BattleActor, distanceModifier, ambush int, rng *rand.Rand, lines *[]string) AttackResult {
	result := AttackResult{TargetHP: defender.HP, TargetArmorDurability: defender.ArmorDurability}
	if attacker.Weapon.AmmoPerRound > 0 {
		if attacker.AmmoRounds < attacker.Weapon.AmmoPerRound {
			*lines = append(*lines, fmt.Sprintf("%s 弹药不足，无法攻击", attacker.Name))
			return result
		}
		result.AmmoSpent = attacker.Weapon.AmmoPerRound
		attacker.AmmoRounds -= result.AmmoSpent
	}
	fleshMultiplier, armorMultiplier := 1.0, 1.0
	result.AmmoID, result.AmmoLevel, fleshMultiplier, armorMultiplier = attackProfile(*attacker)
	hitRate := clamp(float64(attacker.Weapon.Hit)+(attacker.WeaponControl-50)*0.4+float64(distanceModifier+ambush)-defender.Evasion-attacker.Stress*0.25, 5, 95)
	roll := rng.Intn(100) + 1
	hit := float64(roll) <= hitRate
	result.Hit = hit
	result.HitRate = hitRate
	result.HitRoll = roll
	status := "未命中"
	if hit {
		status = "命中"
	}
	*lines = append(*lines, fmt.Sprintf("%s 攻击 %s 命中率%.1f%% 掷%d -> %s", attacker.Name, defender.Name, hitRate, roll, status))
	hitCoeff := 0.5
	damageForStress := 0.0
	if hit {
		hitCoeff = 1
		coverRoll := rng.Intn(100) + 1
		covered := float64(coverRoll) <= float64(defender.Armor.Coverage)
		result.Covered = covered
		result.ArmorLevel = defender.Armor.ProtectionLevel
		baseFleshDamage := float64(attacker.Weapon.Damage) * (0.9 + rng.Float64()*0.2) * fleshMultiplier
		result.FleshDamage = baseFleshDamage
		finalDamage := baseFleshDamage
		if covered {
			result.HitLocation = "armor"
			result.EffectiveArmorLevel = effectiveArmorLevel(defender.Armor, defender.ArmorDurability, defender.ArmorMaxDur)
			delta := result.AmmoLevel - result.EffectiveArmorLevel
			result.Penetrated = result.EffectiveArmorLevel == 0 || delta >= 0
			finalDamage *= penetrationHealthRetention(delta, result.EffectiveArmorLevel)
			armorDamageMultiplier := armorMultiplier
			if result.Penetrated {
				armorDamageMultiplier *= 0.6
			}
			durabilityLoss := float64(attacker.Weapon.Damage) * armorDamageMultiplier
			if durabilityLoss > defender.ArmorDurability {
				durabilityLoss = defender.ArmorDurability
			}
			defender.ArmorDurability -= durabilityLoss
			result.ArmorDamage = durabilityLoss
			*lines = append(*lines, fmt.Sprintf("  命中护甲 覆盖%d≤%d N%d 对 A%d(有效A%d) 穿透:%t 肉伤%.1f 实伤%.1f 耐久-%.1f", coverRoll, defender.Armor.Coverage, result.AmmoLevel, result.ArmorLevel, result.EffectiveArmorLevel, result.Penetrated, baseFleshDamage, finalDamage, durabilityLoss))
		} else {
			result.HitLocation = "limb"
			result.Penetrated = true
			result.ArmorDamage = 0
			*lines = append(*lines, fmt.Sprintf("  命中无护甲四肢区 肉伤%.1f", finalDamage))
		}
		oldHP := defender.HP
		defender.HP -= finalDamage
		if defender.HP < 0 {
			defender.HP = 0
		}
		result.HealthDamage = oldHP - defender.HP
		damageForStress = result.HealthDamage
	}
	// 方案B：压力拆分为“交火压力”（未命中也有，来自压制值）与“受创压力”（命中按伤害）两部分。
	// 交火压力系数 0.2（原 1.0 整额，未命中 0.5），受创压力系数 0.15（原 0.25），避免 1-2 轮压力爆炸。
	antiSuppress := float64(defender.Armor.AntiSuppress)
	stressAdd := math.Max(1, float64(attacker.Weapon.Suppress)*hitCoeff*0.2*(1-defender.ResistEff*0.005)+damageForStress*0.15-antiSuppress)
	defender.Stress += stressAdd
	if defender.Stress > defender.StressThreshold {
		defender.Stress = defender.StressThreshold
	}
	*lines = append(*lines, fmt.Sprintf("  压力+%.1f (当前%.1f/%.0f)", stressAdd, defender.Stress, defender.StressThreshold))
	result.TargetHP = defender.HP
	result.TargetArmorDurability = defender.ArmorDurability
	return result
}

func attackProfile(actor BattleActor) (ammoID string, level int, fleshMultiplier, armorMultiplier float64) {
	if actor.Weapon.AmmoPerRound <= 0 {
		level = maxInt(minInt(actor.Weapon.Penetration/10+1, 6), 1)
		return "melee", level, 1, 1
	}
	return actor.Ammo.ID, actor.Ammo.Level, actor.Ammo.FleshDamageMultiplier, actor.Ammo.ArmorDamageMultiplier
}

func effectiveArmorLevel(armor Armor, durability, maxDurability float64) int {
	if armor.ProtectionLevel <= 0 || durability <= 0 || maxDurability <= 0 {
		return 0
	}
	ratio := durability / maxDurability
	switch {
	case ratio > 0.50:
		return armor.ProtectionLevel
	case ratio > 0.25:
		return maxInt(armor.ProtectionLevel-1, 0)
	default:
		return maxInt(armor.ProtectionLevel-2, 0)
	}
}

func penetrationHealthRetention(delta, effectiveLevel int) float64 {
	if effectiveLevel == 0 {
		return 1
	}
	switch {
	case delta >= 2:
		return 1
	case delta == 1:
		return 0.90
	case delta == 0:
		return 0.75
	case delta == -1:
		return 0.15
	case delta == -2:
		return 0.08
	default:
		return 0.03
	}
}
