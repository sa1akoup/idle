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
	Intellect       float64
	Hearing         float64
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
func EffectiveSkill(t Tuning, train, mainAttr int) float64 {
	return float64(train)*t.Combat.SkillCoef + float64(mainAttr)*t.Combat.AttrCoef
}

// CalcMaxHP 根据力量计算角色最大生命值，并限制在 90-110。
func CalcMaxHP(t Tuning, strength int) float64 {
	return clamp(t.Combat.MaxHPBase+float64(strength-50)*t.Combat.MaxHPStrengthCoef, t.Combat.MaxHPMin, t.Combat.MaxHPMax)
}

// CalcStressThreshold 根据有效抗压计算压力阈值。
func CalcStressThreshold(t Tuning, resistEff float64) float64 {
	return t.Combat.StressThresholdBase + resistEff*t.Combat.StressThresholdResistCoef
}

// calcEvasion 计算闪避率：敏捷与护甲机动性加权。
func calcEvasion(t Tuning, agility float64, armorMobility float64) float64 {
	return clamp(agility*t.Combat.EvasionAgilityCoef+armorMobility, 0, t.Combat.EvasionMax)
}

// calcInitiative 计算先手值：敏捷/感知/武器就绪/护甲加成，压力会降低先手。
func calcInitiative(t Tuning, agility, percepEff, weaponReady, armorInit, stress float64, ambush int) float64 {
	return agility*t.Combat.InitiativeAgilityCoef + percepEff*t.Combat.InitiativePerceptionCoef + weaponReady + armorInit - stress*t.Combat.InitiativeStressCoef + float64(ambush)
}

// calcRecon 计算侦察值：感知占主导，叠加智力与装备加成。
func calcRecon(t Tuning, percepEff float64, intellect int, equipBonus float64) float64 {
	return percepEff*t.Encounter.ReconPerceptionCoef + float64(intellect)*t.Encounter.ReconIntellectCoef + equipBonus
}

// calcConceal 计算隐蔽值：潜行占主导，叠加敏捷与护甲隐蔽加成。
func calcConceal(t Tuning, stealthEff float64, agility int, armorConceal float64) float64 {
	return stealthEff*t.Encounter.ConcealStealthCoef + float64(agility)*t.Encounter.ConcealAgilityCoef + armorConceal
}

// calcWeaponAttrControl 按武器类别用不同属性权重计算属性操控值，未知类别取固定 50。
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

// getProf 按武器类别取出对应熟练度，未知类别返回 0。
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

// finalWeaponControl 综合属性操控与熟练度得出最终武器操控值。
func finalWeaponControl(t Tuning, attrControl float64, prof int) float64 {
	return attrControl*t.Combat.ControlAttrCoef + float64(prof)*t.Combat.ControlProfCoef
}

// buildPlayerActor 由角色状态与装备构造玩家战斗实体，纯 DTO 转换不产生副作用。
func buildPlayerActor(t Tuning, character CharacterState, weapon Weapon, armor Armor, armorDurability int, ammo Ammo, ammoRounds int, hearing int) BattleActor {
	percepEff := EffectiveSkill(t, character.Perception, character.Agility)
	stealthEff := EffectiveSkill(t, character.Stealth, character.Agility)
	resistEff := EffectiveSkill(t, character.Resist, character.Strength)
	attrControl := calcWeaponAttrControl(weapon.Category, character, percepEff, resistEff)
	maxHP := CalcMaxHP(t, character.Strength)
	hp := clamp(character.HP, 0, maxHP)
	return BattleActor{
		Name: character.Name, MaxHP: maxHP, HP: hp,
		Stress: float64(character.Stress), StressThreshold: CalcStressThreshold(t, resistEff), Weapon: weapon, Armor: armor,
		ArmorDurability: float64(armorDurability), ArmorMaxDur: float64(armor.MaxDurability),
		Evasion: calcEvasion(t, float64(character.Agility), float64(armor.Mobility)), Mobility: float64(armor.Mobility),
		PerceptionEff: percepEff, StealthEff: stealthEff, Agility: float64(character.Agility), Intellect: float64(character.Intellect),
		Hearing: float64(hearing), ResistEff: resistEff,
		WeaponControl: finalWeaponControl(t, attrControl, getProf(character, weapon.Category)), Ammo: ammo, AmmoRounds: ammoRounds,
	}
}

// enemyWeaponControl 由敌人真实属性推导武器操控：以模板典型属性（敏捷45/感知45/压制25）为基准做偏离，
// 替代原先的固定值 55，让属性差异在命中与先手上可见。
func enemyWeaponControl(enemy Enemy) float64 {
	return 50 + float64(enemy.Agility-45)*0.3 + float64(enemy.Perception-45)*0.3 + float64(enemy.Suppress-25)*0.2
}

// buildEnemyActor 由敌人数据构造敌方战斗实体，压力阈值缺失时默认 60。
func buildEnemyActor(enemy Enemy, weapon Weapon, armor Armor, ammo Ammo) BattleActor {
	threshold := float64(enemy.StressThreshold)
	if threshold == 0 {
		threshold = 60
	}
	return BattleActor{
		Name: enemy.Name, MaxHP: float64(enemy.HP), HP: float64(enemy.HP), StressThreshold: threshold,
		Weapon: weapon, Armor: armor, ArmorDurability: float64(armor.MaxDurability), ArmorMaxDur: float64(armor.MaxDurability),
		Evasion: float64(enemy.Evasion), Mobility: float64(enemy.Mobility), PerceptionEff: float64(enemy.Perception),
		StealthEff: float64(enemy.Stealth), Agility: float64(enemy.Agility), Intellect: float64(enemy.Intellect),
		ResistEff: float64(enemy.Resist), WeaponControl: enemyWeaponControl(enemy), Ammo: ammo, AmmoRounds: enemy.AmmoRounds,
	}
}

// getDistMod 取武器在指定距离段的命中修正，未知距离段返回 0。
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

// bestDistanceForWeapon 返回武器距离修正最优的距离段，平手时按近-中-远优先；
// 任何距离段都无法攻击（最优修正 ≤ -100）时返回空串，表示不调整距离。
func bestDistanceForWeapon(weapon Weapon) string {
	best, bestMod := "close", weapon.CloseMod
	if weapon.MidMod > bestMod {
		best, bestMod = "mid", weapon.MidMod
	}
	if weapon.FarMod > bestMod {
		best, bestMod = "far", weapon.FarMod
	}
	if bestMod <= -100 {
		return ""
	}
	return best
}

// distanceLabel 把距离段翻译成中文播报文案，未知值原样返回。
func distanceLabel(distance string) string {
	switch distance {
	case "close":
		return "近距离"
	case "mid":
		return "中距离"
	case "far":
		return "远距离"
	default:
		return distance
	}
}

// shouldEscape 判定是否满足脱离条件：低血量、高压力、弹药或护甲耗尽之一。
func shouldEscape(actor BattleActor, policy StylePolicy) bool {
	if actor.HP <= 0 {
		return false
	}
	return actor.HP < actor.MaxHP*policy.HealthEvacRatio ||
		actor.Stress >= actor.StressThreshold*policy.StressEvacRatio ||
		(actor.Weapon.AmmoPerRound > 0 && actor.AmmoRounds < actor.Weapon.AmmoPerRound) ||
		(actor.Armor.ProtectionLevel > 0 && actor.ArmorMaxDur > 0 && actor.ArmorDurability <= 0)
}

// canAttack 判定弹药是否足够发动一次攻击，无弹药消耗的武器恒可攻击。
func canAttack(actor BattleActor) bool {
	return actor.Weapon.AmmoPerRound <= 0 || actor.AmmoRounds >= actor.Weapon.AmmoPerRound
}

type escapeResult struct {
	success   bool
	msg       string
	usedSmoke bool
}

// tryEscape 掷随机数判定脱离成败；判定失败但携带烟雾弹时自动消耗，并给予一次额外的脱离判定（不保证成功）。
func tryEscape(t Tuning, actor, chaser *BattleActor, hasSmoke bool, rng *rand.Rand) escapeResult {
	escapeVal := actor.StealthEff*t.Encounter.BypassStealthCoef + actor.Agility*t.Encounter.BypassAgilityCoef + float64(actor.Armor.Escape)
	chaseVal := chaser.PerceptionEff*t.Encounter.ApproachBlockPerceptionCoef + float64(chaser.Weapon.Suppress)*t.Encounter.ApproachBlockSuppressCoef + chaser.Mobility
	probability := clamp(t.Combat.EscapeBase+(escapeVal-chaseVal)*t.Combat.EscapeCoef, t.Combat.EscapeMin, t.Combat.EscapeMax)
	roll := rng.Intn(100) + 1
	success := float64(roll) <= probability
	status := "失败"
	if success {
		status = "成功"
	}
	message := fmt.Sprintf("脱离判定 %.0f%% 掷%d -> %s", probability, roll, status)
	if !success && hasSmoke {
		secondRoll := rng.Intn(100) + 1
		success = float64(secondRoll) <= probability
		status = "失败"
		if success {
			status = "成功"
		}
		return escapeResult{success: success, msg: fmt.Sprintf("%s；烟雾弹自动消耗，二次判定 %.0f%% 掷%d -> %s", message, probability, secondRoll, status), usedSmoke: true}
	}
	return escapeResult{success: success, msg: message}
}

// attack 执行一次攻击结算：命中判定、护甲穿透、伤害与压力累积，结果写入 AttackResult。
func attack(t Tuning, attacker, defender *BattleActor, distanceModifier, ambush int, rng *rand.Rand, lines *[]string) AttackResult {
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
	hitRate := clamp(float64(attacker.Weapon.Hit)+(attacker.WeaponControl-50)*t.Combat.HitRateCoef+float64(distanceModifier+ambush)-defender.Evasion-attacker.Stress*t.Combat.HitRateStressCoef, t.Combat.HitRateMin, t.Combat.HitRateMax)
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
	hitCoeff := t.Combat.StressSuppressMissCoef
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
			result.EffectiveArmorLevel = effectiveArmorLevel(t, defender.Armor, defender.ArmorDurability, defender.ArmorMaxDur)
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
	antiSuppress := float64(defender.Armor.AntiSuppress)
	stressAdd := math.Max(t.Combat.StressMin, float64(attacker.Weapon.Suppress)*hitCoeff*t.Combat.StressSuppressCoef*(1-defender.ResistEff*t.Combat.StressResistCoef)+damageForStress*t.Combat.StressDamageCoef-antiSuppress)
	defender.Stress += stressAdd
	if defender.Stress > defender.StressThreshold {
		defender.Stress = defender.StressThreshold
	}
	*lines = append(*lines, fmt.Sprintf("  压力+%.1f (当前%.1f/%.0f)", stressAdd, defender.Stress, defender.StressThreshold))
	result.TargetHP = defender.HP
	result.TargetArmorDurability = defender.ArmorDurability
	return result
}

// attackProfile 返回弹药档案：近战按穿透值折算弹药等级，枪械直接用弹药配置。
func attackProfile(actor BattleActor) (ammoID string, level int, fleshMultiplier, armorMultiplier float64) {
	if actor.Weapon.AmmoPerRound <= 0 {
		level = maxInt(minInt(actor.Weapon.Penetration/10+1, 6), 1)
		return "melee", level, 1, 1
	}
	return actor.Ammo.ID, actor.Ammo.Level, actor.Ammo.FleshDamageMultiplier, actor.Ammo.ArmorDamageMultiplier
}

// effectiveArmorLevel 按耐久剩余比例折算有效防护等级：损耗越多防护越弱，直至 0。
func effectiveArmorLevel(t Tuning, armor Armor, durability, maxDurability float64) int {
	if armor.ProtectionLevel <= 0 || durability <= 0 || maxDurability <= 0 {
		return 0
	}
	ratio := durability / maxDurability
	switch {
	case ratio > t.Combat.ArmorDuraHighBand:
		return armor.ProtectionLevel
	case ratio > t.Combat.ArmorDuraLowBand:
		return maxInt(armor.ProtectionLevel-1, 0)
	default:
		return maxInt(armor.ProtectionLevel-2, 0)
	}
}

// penetrationHealthRetention 由弹药与有效护甲等级差决定穿透后保留的伤害比例，等级差越小保留越低。
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
