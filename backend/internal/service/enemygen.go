// 敌人生成器：按模板 + 生成上下文产出合法敌人变体（engine.Enemy）。
// 设计见 docs/敌人生成器设计方案-V1.0.md；分类参考逃离塔科夫敌人体系。
package service

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"idle/internal/engine"
	"idle/internal/models"
)

// GenerateContext 生成上下文：难度缩放输入。
type GenerateContext struct {
	PlayerLevel   int
	Heat          int
	NodeValueTier int
}

// EnemyGenerator 所需目录数据，由快照构建器注入（便于纯函数测试）。
type EnemyGenerator struct {
	Weapons   map[string]engine.Weapon
	Armors    map[string]engine.Armor
	Ammos     map[string]engine.Ammo
	Containers map[string]engine.Container
}

// GenerateEnemy 根据模板生成一个合法敌人变体。rng 注入保证确定性可复现。
func GenerateEnemy(rng *rand.Rand, template models.EnemyTemplateDef, ctx GenerateContext, catalog EnemyGenerator) (engine.Enemy, error) {
	scale := difficultyScale(ctx)

	// 1. 属性：区间浮动 + 温和缩放
	hp := iFlux(rng, template.HPBase, template.HPFlux, template.HPFloor, template.HPCap, scale)
	stress := iFlux(rng, template.StressBase, template.StressFlux, template.StressFloor, template.StressCap, scale)
	percep := iFlux(rng, template.PerceptionBase, template.PerceptionFlux, 5, 100, scale)
	stealth := iFlux(rng, template.StealthBase, template.StealthFlux, 5, 100, scale)
	agility := iFlux(rng, template.AgilityBase, template.AgilityFlux, 5, 100, scale)
	evasion := iFlux(rng, template.EvasionBase, template.EvasionFlux, 0, 60, scale)
	mobility := iFlux(rng, template.MobilityBase, template.MobilityFlux, -20, 30, scale)
	suppress := iFlux(rng, template.SuppressBase, template.SuppressFlux, 0, 100, scale)
	// 智力/抗性：模板未配置上下限时回落默认 20-100，保证旧模板行仍生成合法值。
	intellect := iFlux(rng, template.IntellectBase, template.IntellectFlux,
		attrFloor(template.IntellectFloor, 20), attrCap(template.IntellectCap, 100), scale)
	resist := iFlux(rng, template.ResistBase, template.ResistFlux,
		attrFloor(template.ResistFloor, 20), attrCap(template.ResistCap, 100), scale)

	// 2. 装备：Boss 固定，其余走池
	var weaponID, armorID string
	if template.Kind == "boss" && template.BossWeaponID != "" {
		weaponID = template.BossWeaponID
	} else {
		weaponID, _ = pickWeighted(rng, template.WeaponPool)
	}
	if template.Kind == "boss" && template.BossArmorID != "" {
		armorID = template.BossArmorID
	} else {
		armorID, _ = pickWeighted(rng, template.ArmorPool)
	}
	weapon, ok := catalog.Weapons[weaponID]
	if !ok {
		return engine.Enemy{}, fmt.Errorf("敌人 %s 抽取的武器 %s 不在目录中", template.ID, weaponID)
	}
	if _, ok := catalog.Armors[armorID]; !ok {
		return engine.Enemy{}, fmt.Errorf("敌人 %s 抽取的护甲 %s 不在目录中", template.ID, armorID)
	}

	// 3. 弹药：口径匹配 + 等级区间内选最高可用
	ammoID := ""
	ammoRounds := 0
	if weapon.AmmoPerRound > 0 {
		ammo, err := pickMatchingAmmo(rng, weapon.CaliberID, template.AmmoLevelMin, template.AmmoLevelMax, catalog.Ammos)
		if err != nil {
			return engine.Enemy{}, fmt.Errorf("敌人 %s 弹药生成失败: %w", template.ID, err)
		}
		ammoID = ammo.ID
		ammoRounds = int(math.Round(float64(template.AmmoRoundsBase) * template.AmmoRoundsMult))
		if ammoRounds < weapon.AmmoPerRound {
			ammoRounds = weapon.AmmoPerRound
		}
	}

	// 4. 背包：按池权重抽一个容器
	backpackID, _ := pickWeighted(rng, template.BackpackPool)

	// 5. 命名：Boss 用专属名
	name := template.Name
	if template.Kind == "boss" && template.BossName != "" {
		name = template.BossName
	} else if ctx.NodeValueTier >= 3 && template.Kind != "boss" {
		// 高价值节点的高阶敌人加双名变体
		name = template.Name
	}

	return engine.Enemy{
		ID:                  template.ID + "_" + randomSuffix(rng),
		Name:                name,
		Kind:                template.Kind,
		VariedFrom:          template.ID,
		HP:                  hp,
		StressThreshold:     stress,
		Perception:          percep,
		Stealth:             stealth,
		Agility:             agility,
		WeaponID:            weaponID,
		ArmorID:             armorID,
		AmmoID:              ammoID,
		AmmoRounds:          ammoRounds,
		Evasion:             evasion,
		Mobility:            mobility,
		Suppress:            suppress,
		Intellect:           intellect,
		Resist:              resist,
		BackpackContainerID: backpackID,
	}, nil
}

// attrFloor / attrCap 属性区间缺省回落：未配置（<=0）时使用默认下限/上限。
func attrFloor(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func attrCap(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

// difficultyScale 温和缩放：HP/压力 ±20% 随 Heat 每 10 点 +2%，节点价值档少量上浮。
func difficultyScale(ctx GenerateContext) float64 {
	heatBonus := float64(ctx.Heat) / 10 * 0.02
	valueBonus := float64(max(ctx.NodeValueTier-1, 0)) * 0.02
	return 1 + heatBonus + valueBonus
}

// iFlux 在基础值上下按 flux 随机浮动，再乘难度缩放并钳制到 [floor, cap]。
func iFlux(rng *rand.Rand, base, flux, floor, cap int, scale float64) int {
	v := float64(base) + randRange(rng, -flux, flux)
	scaled := int(math.Round(v * scale))
	return clampInt(scaled, floor, cap)
}

// randRange 返回 [min, max) 区间的随机浮点数，全部走注入的 rng 以保证确定性。
func randRange(rng *rand.Rand, min, max int) float64 {
	if max <= min {
		return float64(min)
	}
	return float64(min) + rng.Float64()*float64(max-min)
}

// pickWeighted 按权重抽取一个引用；池为空时返回 false，调用方按缺省回退处理。
func pickWeighted(rng *rand.Rand, pool []models.WeightedRef) (string, bool) {
	total := 0
	for _, p := range pool {
		total += max(p.Weight, 1)
	}
	if total <= 0 {
		return "", false
	}
	roll := rng.Intn(total)
	for _, p := range pool {
		w := max(p.Weight, 1)
		if roll < w {
			return p.Ref, true
		}
		roll -= w
	}
	return "", false
}

// pickMatchingAmmo 在该武器口径与模板等级区间内随机选一种弹药，候选为空则报错。
func pickMatchingAmmo(rng *rand.Rand, caliberID string, levelMin, levelMax int, ammos map[string]engine.Ammo) (engine.Ammo, error) {
	var candidates []engine.Ammo
	for _, ammo := range ammos {
		if ammo.CaliberID != caliberID {
			continue
		}
		if ammo.Level < levelMin || ammo.Level > levelMax {
			continue
		}
		candidates = append(candidates, ammo)
	}
	if len(candidates) == 0 {
		return engine.Ammo{}, fmt.Errorf("口径 %s 在等级 %d-%d 无可用弹药", caliberID, levelMin, levelMax)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return candidates[rng.Intn(len(candidates))], nil
}

// randomSuffix 生成 4 位随机后缀，拼在模板 ID 后形成每个敌人实例的唯一 ID。
func randomSuffix(rng *rand.Rand) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

// clampInt 把 v 钳制到 [lo, hi] 区间，避免属性越界。
func clampInt(v, lo, hi int) int { return min(max(v, lo), hi) }
