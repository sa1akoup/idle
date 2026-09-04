package engine

import (
	"math/rand"
	"testing"
)

func TestChooseLootItemBlocksLegendaryInLowTier(t *testing.T) {
	catalog := map[string]LootItem{
		"ai2_medkit": {ID: "ai2_medkit", Category: "medical", Rarity: "common", DropWeight: 40},
		"salewa":     {ID: "salewa", Category: "medical", Rarity: "uncommon", DropWeight: 12},
		"ledx":       {ID: "ledx", Category: "medical", Rarity: "legendary", DropWeight: 1},
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		item, ok := chooseLootItem(catalog, "medical", 1, rng)
		if !ok {
			t.Fatal("低档医疗容器应能抽出普通药品")
		}
		if item.ID == "ledx" {
			t.Fatal("医疗袋不应抽出 LEDX")
		}
	}
	found := false
	for i := 0; i < 400; i++ {
		item, ok := chooseLootItem(catalog, "medical", 5, rng)
		if !ok {
			t.Fatal("医疗室应能抽出药品")
		}
		if item.ID == "ledx" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("医疗室在足够抽样下应能抽出 LEDX")
	}
}

func TestChooseLootItemSkipsZeroDropWeight(t *testing.T) {
	catalog := map[string]LootItem{
		"disabled": {ID: "disabled", Category: "valuable", Rarity: "uncommon", DropWeight: 0},
		"chain":    {ID: "chain", Category: "valuable", Rarity: "rare", DropWeight: 4},
	}
	rng := rand.New(rand.NewSource(2))
	_, ok := chooseLootItem(catalog, "valuable", 1, rng)
	if ok {
		t.Fatal("ValueTier 1 且贵重物均为 rare 或权重 0 时应抽空")
	}
	item, ok := chooseLootItem(catalog, "valuable", 2, rng)
	if !ok || item.ID != "chain" {
		t.Fatalf("ValueTier 2 应抽出金链，实际 ok=%v item=%s", ok, item.ID)
	}
}
