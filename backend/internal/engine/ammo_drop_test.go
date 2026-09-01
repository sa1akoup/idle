// 敌人弹药掉落测试：击倒带枪敌人后可搜缴其战后剩余弹药；近战敌人不掉弹药。
package engine

import "testing"

// ammoDropSnapshot 构造“玩家必先手秒杀带枪敌人”的极小快照。
func ammoDropSnapshot(enemy Weapon) ScenarioSnapshot {
	snapshot := replayTestSnapshot()
	snapshot.Weapons["weapon_test"] = Weapon{
		ID: "weapon_test", Name: "测试近战武器", Category: "melee", Hit: 100, Damage: 100, AmmoPerRound: 0,
	}
	snapshot.Weapons["enemy_gun"] = enemy
	snapshot.Ammos = map[string]Ammo{
		"enemy_ammo": {ID: "enemy_ammo", Name: "7.62×39 测试弹", CaliberID: "762x39", Level: 1, RoundsPerSlot: 30, FleshDamageMultiplier: 1.0, ArmorDamageMultiplier: 1.0},
	}
	snapshot.AmmoSupplies = map[string]AmmoSupply{
		"enemy_ammo": {AmmoID: "enemy_ammo", CaliberID: "762x39", Level: 1, UnitPrice: 1, Available: false},
	}
	snapshot.Enemies = map[string]Enemy{
		"enemy_gunner": {
			ID: "enemy_gunner", Name: "带枪敌人", HP: 1, StressThreshold: 100,
			WeaponID: "enemy_gun", ArmorID: "armor_test", AmmoID: "enemy_ammo", AmmoRounds: 30,
		},
	}
	snapshot.Events.EncounterPools = map[string][]EncounterPoolEntry{
		"guard": {{ID: "pool_guard", MapID: "map_test", Role: "guard", EnemyID: "enemy_gunner", Weight: 1}},
	}
	node := snapshot.Nodes[0]
	node.EnemyID = "enemy_gunner"
	node.EncounterRole = "guard" // 守卫强制接战，避免均衡型默认绕行
	node.EncounterChance = 100
	node.Distance = "mid"
	snapshot.Nodes[0] = node
	return snapshot
}

func TestDefeatedGunnerDropsRemainingAmmo(t *testing.T) {
	snapshot := ammoDropSnapshot(Weapon{
		ID: "enemy_gun", Name: "敌方步枪", Category: "rifle", Hit: 0, Damage: 1, AmmoPerRound: 3,
		CaliberID: "762x39", Noise: 60, Suppress: 10,
	})
	result, err := SimulateRun(snapshot, RunInput{
		SessionSeed: 20260825,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "弹药测试角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	})
	if err != nil {
		t.Fatalf("弹药掉落模拟失败: %v", err)
	}
	found := false
	for _, drop := range result.Loot {
		if drop.ItemID == "enemy_ammo" && drop.Source == "敌人弹药" {
			if drop.Quantity != 30 {
				t.Fatalf("弹药数量 = %d，期望战后剩余 30 发", drop.Quantity)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("击倒带枪敌人后未掉落弹药: %+v\n报告: %s", result.Loot, append([]string(nil), result.Report...))
	}
}

func TestMeleeEnemyDropsNoAmmo(t *testing.T) {
	snapshot := ammoDropSnapshot(Weapon{
		ID: "enemy_gun", Name: "敌匕首", Category: "melee", Hit: 0, Damage: 1, AmmoPerRound: 0,
	})
	// 近战敌人没有弹药，击倒后不应出现弹药掉落。
	snapshot.Enemies["enemy_gunner"] = Enemy{
		ID: "enemy_gunner", Name: "近战敌人", HP: 1, StressThreshold: 100,
		WeaponID: "enemy_gun", ArmorID: "armor_test",
	}
	result, err := SimulateRun(snapshot, RunInput{
		SessionSeed: 20260825,
		RunIndex:    1,
		Style:       ActionStyleBalanced,
		State: EngineState{
			Character:       CharacterState{Name: "弹药测试角色", Strength: 50, Agility: 50, Perception: 50, Stealth: 50, Resist: 50, HP: 100, Energy: 100, Hydration: 100},
			Loadout:         LoadoutState{WeaponID: "weapon_test", ArmorID: "armor_test"},
			ArmorDurability: 100,
			Carry:           CarryState{TotalSlots: 20, UsedSlots: 2, TotalWeight: 100, UsedWeight: 10},
		},
	})
	if err != nil {
		t.Fatalf("近战敌人模拟失败: %v", err)
	}
	for _, drop := range result.Loot {
		if drop.Source == "敌人弹药" {
			t.Fatalf("近战敌人不应掉落弹药: %+v", drop)
		}
	}
}

func TestLootUsageMergesSameAmmoBeforePacking(t *testing.T) {
	snapshot := ScenarioSnapshot{
		Ammos: map[string]Ammo{
			"ammo_test": {ID: "ammo_test", Name: "测试弹药", RoundsPerSlot: 30},
		},
		Tuning: DefaultTuning(),
	}

	slots, weight, err := lootUsageForDrops(snapshot, []LootDrop{
		{ItemID: "ammo_test", Quantity: 15},
		{ItemID: "ammo_test", Quantity: 15},
	})
	if err != nil {
		t.Fatalf("合并弹药占用计算失败: %v", err)
	}
	if slots != 1 || weight != 0.5 {
		t.Fatalf("同种弹药合并后占用 = %d 格 %.1fkg，期望 1 格 0.5kg", slots, weight)
	}
}
