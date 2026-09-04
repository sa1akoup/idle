package engine

import (
	"fmt"
	"math/rand"
)

func pickWeightedRef(rng *rand.Rand, items []WeightedRef) (string, bool) {
	total := 0
	for _, item := range items {
		if item.Ref == "" || item.Weight <= 0 {
			continue
		}
		total += item.Weight
	}
	if total <= 0 {
		return "", false
	}
	roll := rng.Intn(total)
	for _, item := range items {
		if item.Ref == "" || item.Weight <= 0 {
			continue
		}
		if roll < item.Weight {
			return item.Ref, true
		}
		roll -= item.Weight
	}
	return "", false
}

func collectBossSpecialLoot(snapshot ScenarioSnapshot, state *eventRunState, enemy Enemy, rng *rand.Rand, loot *[]LootDrop) error {
	if enemy.Kind != "boss" || len(enemy.BossLootItems) == 0 {
		return nil
	}
	itemID, ok := pickWeightedRef(rng, enemy.BossLootItems)
	if !ok {
		return nil
	}
	item, ok := snapshot.LootItems[itemID]
	if !ok {
		itemDef, found := snapshot.Items[itemID]
		if !found {
			*state.Lines = append(*state.Lines, fmt.Sprintf("    Boss 掉落 %s 不在目录中，跳过", itemID))
			return nil
		}
		item = LootItem{ID: itemDef.ID, Name: itemDef.Name, Category: itemDef.Category, Weight: itemDef.Weight, Slots: itemDef.Slots}
	}
	candidate := append(append([]LootDrop(nil), *loot...), LootDrop{ItemID: itemID, Quantity: 1, Source: "boss"})
	needSlots, needWeight, err := lootUsageForDrops(snapshot, candidate)
	if err != nil {
		return err
	}
	if needSlots > state.CarrySlots || needWeight > state.CarryWeight+1e-9 {
		state.CarryBlocked = true
		*state.Lines = append(*state.Lines, fmt.Sprintf("    容量不足，放弃 Boss 掉落 %s", item.Name))
		return nil
	}
	*loot = append(*loot, LootDrop{ItemID: itemID, Quantity: 1, Source: "boss"})
	state.noteCollectedKey(itemID)
	state.LootSlots, state.LootWeight = needSlots, needWeight
	*state.Lines = append(*state.Lines, fmt.Sprintf("    搜出 Boss 掉落 %s", item.Name))
	return nil
}
