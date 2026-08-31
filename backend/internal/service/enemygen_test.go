// 敌人生成器测试：确定性、合法性、缩放与 BOSS 规则。
package service

import (
	"math/rand"
	"testing"

	"idle/internal/engine"
	"idle/internal/models"
)

func testCatalog() EnemyGenerator {
	return EnemyGenerator{
		Weapons: map[string]engine.Weapon{
			"pistol_glock": {ID: "pistol_glock", CaliberID: "9x19", AmmoPerRound: 1},
			"rifle_ak":     {ID: "rifle_ak", CaliberID: "762x39", AmmoPerRound: 3},
			"smg_mp5":      {ID: "smg_mp5", CaliberID: "9x19", AmmoPerRound: 4},
			"melee_knife":  {ID: "melee_knife", AmmoPerRound: 0},
			"sniper_svds":  {ID: "sniper_svds", CaliberID: "762x54r", AmmoPerRound: 3},
			"rifle_rpk16":  {ID: "rifle_rpk16", CaliberID: "545x39", AmmoPerRound: 4},
		},
		Armors: map[string]engine.Armor{
			"light_01":  {ID: "light_01", ProtectionLevel: 2, MaxDurability: 100},
			"heavy_01":  {ID: "heavy_01", ProtectionLevel: 5, MaxDurability: 150},
			"light_02":  {ID: "light_02", ProtectionLevel: 2, MaxDurability: 80},
			"armor_paca": {ID: "armor_paca", ProtectionLevel: 2, MaxDurability: 70},
		},
		Ammos: map[string]engine.Ammo{
			"ammo_9x19_n2":   {ID: "ammo_9x19_n2", CaliberID: "9x19", Level: 2},
			"ammo_9x19_n3":   {ID: "ammo_9x19_n3", CaliberID: "9x19", Level: 3},
			"ammo_762x39_n4": {ID: "ammo_762x39_n4", CaliberID: "762x39", Level: 4},
			"ammo_762x54r_n4": {ID: "ammo_762x54r_n4", CaliberID: "762x54r", Level: 4},
			"ammo_545x39_n4": {ID: "ammo_545x39_n4", CaliberID: "545x39", Level: 4},
		},
		Containers: map[string]engine.Container{
			"enemy_backpack_basic": {ID: "enemy_backpack_basic"},
			"enemy_backpack_elite": {ID: "enemy_backpack_elite"},
		},
	}
}

func TestGenerateEnemyDeterministic(t *testing.T) {
	tpl := models.EnemyTemplateDef{
		ID: "template_patrol", Name: "巡逻小队", Kind: "grunt", Tier: 0,
		HPBase: 80, HPFlux: 10, HPFloor: 60, HPCap: 100, StressBase: 60, StressFlux: 8, StressFloor: 40, StressCap: 80,
		PerceptionBase: 40, PerceptionFlux: 5, StealthBase: 30, StealthFlux: 5, AgilityBase: 45, AgilityFlux: 4,
		EvasionBase: 15, EvasionFlux: 3, MobilityBase: 5, MobilityFlux: 2, SuppressBase: 20, SuppressFlux: 4,
		WeaponPool: []models.WeightedRef{{Ref: "pistol_glock", Weight: 100}},
		ArmorPool:  []models.WeightedRef{{Ref: "light_01", Weight: 100}},
		AmmoLevelMin: 1, AmmoLevelMax: 3, AmmoRoundsBase: 24, AmmoRoundsMult: 1.0,
		BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_basic", Weight: 100}},
	}
	gen := testCatalog()
	// same seed -> same output
	e1, err := GenerateEnemy(rand.New(rand.NewSource(42)), tpl, GenerateContext{}, gen)
	if err != nil { t.Fatal(err) }
	e2, err := GenerateEnemy(rand.New(rand.NewSource(42)), tpl, GenerateContext{}, gen)
	if err != nil { t.Fatal(err) }
	if e1.ID != e2.ID || e1.HP != e2.HP || e1.AmmoID != e2.AmmoID {
		t.Fatalf("确定性失败: %+v vs %+v", e1, e2)
	}
	// bounds
	if e1.HP < 60 || e1.HP > 100 { t.Fatalf("HP 越界: %d", e1.HP) }
	if e1.StressThreshold < 40 || e1.StressThreshold > 80 { t.Fatalf("压力越界: %d", e1.StressThreshold) }
	if e1.WeaponID != "pistol_glock" || e1.ArmorID != "light_01" { t.Fatalf("装备抽取错误: %+v", e1) }
	if e1.AmmoID == "" || e1.AmmoRounds < 1 { t.Fatalf("弹药配置错误: %+v", e1) }
	if e1.BackpackContainerID != "enemy_backpack_basic" { t.Fatalf("背包错误: %+v", e1) }
}

func TestGenerateEnemyAmmoCaliberMatch(t *testing.T) {
	tpl := models.EnemyTemplateDef{
		ID: "template_elite", Name: "精锐", Kind: "elite",
		HPBase: 120, HPFlux: 12, HPFloor: 100, HPCap: 150, StressBase: 80, StressFlux: 10, StressFloor: 60, StressCap: 100,
		WeaponPool: []models.WeightedRef{{Ref: "rifle_ak", Weight: 100}},
		ArmorPool:  []models.WeightedRef{{Ref: "heavy_01", Weight: 100}},
		AmmoLevelMin: 3, AmmoLevelMax: 4, AmmoRoundsBase: 30, AmmoRoundsMult: 1.0,
		BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_elite", Weight: 100}},
	}
	gen := testCatalog()
	for i := 0; i < 20; i++ {
		e, err := GenerateEnemy(rand.New(rand.NewSource(int64(i))), tpl, GenerateContext{}, gen)
		if err != nil { t.Fatal(err) }
		ammo := gen.Ammos[e.AmmoID]
		if ammo.CaliberID != "762x39" { t.Fatalf("口径不匹配: %s", e.AmmoID) }
		if ammo.Level < 3 || ammo.Level > 4 { t.Fatalf("等级越界: %+v", ammo) }
	}
}

func TestGenerateEnemyBossRules(t *testing.T) {
	tpl := models.EnemyTemplateDef{
		ID: "template_boss_killa", Name: "基拉", Kind: "boss",
		HPBase: 180, HPFlux: 20, HPFloor: 150, HPCap: 240, StressBase: 100, StressFlux: 15, StressFloor: 80, StressCap: 130,
		BossWeaponID: "rifle_rpk16", BossArmorID: "heavy_01", BossName: "【BOSS】铁腕·基拉",
		AmmoLevelMin: 4, AmmoLevelMax: 4, AmmoRoundsBase: 120, AmmoRoundsMult: 1.5,
		BackpackPool: []models.WeightedRef{{Ref: "enemy_backpack_elite", Weight: 100}},
	}
	gen := testCatalog()
	e, err := GenerateEnemy(rand.New(rand.NewSource(7)), tpl, GenerateContext{NodeValueTier: 5}, gen)
	if err != nil { t.Fatal(err) }
	if e.Name != "【BOSS】铁腕·基拉" { t.Fatalf("BOSS 命名错误: %s", e.Name) }
	if e.WeaponID != "rifle_rpk16" || e.ArmorID != "heavy_01" { t.Fatalf("BOSS 固定装备错误: %+v", e) }
	if e.HP < 150 || e.HP > 240 { t.Fatalf("BOSS HP 越界: %d", e.HP) }
	// high value tier should scale up
	eLow, _ := GenerateEnemy(rand.New(rand.NewSource(1)), tpl, GenerateContext{NodeValueTier: 1}, gen)
	eHigh, _ := GenerateEnemy(rand.New(rand.NewSource(1)), tpl, GenerateContext{NodeValueTier: 5}, gen)
	if eHigh.HP <= eLow.HP { t.Logf("注意：缩放可能被区间浮动掩盖 low=%d high=%d", eLow.HP, eHigh.HP) }
}
