package engine

import "testing"

func TestSelectSecureLootFitsBySlotsNotCount(t *testing.T) {
	snapshot := ScenarioSnapshot{
		LootItems: map[string]LootItem{
			"bitcoin": {ID: "bitcoin", Name: "比特币", Price: 10000, Weight: 1, Slots: 1, DropWeight: 1},
			"lion":    {ID: "lion", Name: "青铜狮", Price: 2700, Weight: 3, Slots: 2, DropWeight: 2},
			"bolt":    {ID: "bolt", Name: "螺栓", Price: 35, Weight: 1, Slots: 1, DropWeight: 40},
		},
	}
	loot := []LootDrop{
		{ItemID: "lion", Quantity: 1},
		{ItemID: "bitcoin", Quantity: 1},
		{ItemID: "bolt", Quantity: 2},
	}

	one := SelectSecureLoot(snapshot, loot, 1)
	if len(one) != 1 || one[0].ItemID != "bitcoin" || one[0].Quantity != 1 {
		t.Fatalf("1 格口袋应保住比特币，实际 %+v", one)
	}

	two := SelectSecureLoot(snapshot, loot, 2)
	gotTwo := map[string]int{}
	for _, drop := range two {
		gotTwo[drop.ItemID] += drop.Quantity
	}
	if gotTwo["bitcoin"] != 1 || gotTwo["bolt"] != 1 {
		t.Fatalf("2 格口袋应保住比特币+螺栓，实际 %+v", two)
	}

	four := SelectSecureLoot(snapshot, loot, 4)
	got := map[string]int{}
	for _, drop := range four {
		got[drop.ItemID] += drop.Quantity
	}
	if got["bitcoin"] != 1 || got["lion"] != 1 || got["bolt"] != 1 {
		t.Fatalf("4 格口袋应保住比特币+青铜狮+螺栓，实际 %+v", four)
	}
}

func TestSelectSecureLootRejectsOversizedItem(t *testing.T) {
	snapshot := ScenarioSnapshot{
		LootItems: map[string]LootItem{
			"lion": {ID: "lion", Name: "青铜狮", Price: 2700, Weight: 3, Slots: 2, DropWeight: 2},
		},
	}
	kept := SelectSecureLoot(snapshot, []LootDrop{{ItemID: "lion", Quantity: 1}}, 1)
	if len(kept) != 0 {
		t.Fatalf("1 格口袋装不下 2 格物品，实际 %+v", kept)
	}
}
