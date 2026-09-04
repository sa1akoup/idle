package engine

import (
	"math/rand"
	"testing"
)

func TestPickWeightedRefSkipsZeroWeight(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	id, ok := pickWeightedRef(rng, []WeightedRef{
		{Ref: "ledx", Weight: 0},
		{Ref: "bolts", Weight: 10},
	})
	if !ok || id != "bolts" {
		t.Fatalf("got %s %v", id, ok)
	}
}

func TestCollectBossSpecialLootAbandonsWhenFull(t *testing.T) {
	snapshot := ScenarioSnapshot{
		LootItems: map[string]LootItem{
			"ledx": {ID: "ledx", Name: "LEDX", Category: "medical", Weight: 1, Slots: 2},
		},
		Items: map[string]ItemDefinition{
			"ledx": {ID: "ledx", Name: "LEDX", Kind: "loot", Weight: 1, Slots: 2},
		},
	}
	lines := []string{}
	state := &eventRunState{
		Lines:       &lines,
		CarrySlots:  1,
		CarryWeight: 0.5,
		Snapshot:    &snapshot,
	}
	enemy := Enemy{Kind: "boss", BossLootItems: []WeightedRef{{Ref: "ledx", Weight: 1}}}
	loot := []LootDrop{}
	if err := collectBossSpecialLoot(snapshot, state, enemy, rand.New(rand.NewSource(1)), &loot); err != nil {
		t.Fatal(err)
	}
	if len(loot) != 0 {
		t.Fatalf("容量不足仍收下 %v", loot)
	}
	if !state.CarryBlocked {
		t.Fatal("容量不足应标记携行阻断")
	}
}

func TestNodeRiskBossHigherThanElite(t *testing.T) {
	if nodeRisk(Node{EncounterRole: "boss"}) <= nodeRisk(Node{EncounterRole: "elite"}) {
		t.Fatal("Boss 节点风险应高于精英")
	}
}
