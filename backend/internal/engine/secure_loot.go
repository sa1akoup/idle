package engine

// SelectSecureLoot 按物品占格从本局搜刮中选出能放进口袋的最高价值子集。
// 容量为口袋内部格数；失能保物用，成功撤离不走这里。
func SelectSecureLoot(snapshot ScenarioSnapshot, loot []LootDrop, innerSlots int) []LootDrop {
	if innerSlots <= 0 || len(loot) == 0 {
		return nil
	}
	units := flattenSecureUnits(snapshot, loot)
	if len(units) == 0 {
		return nil
	}
	chosen := knapsackSecureUnits(units, innerSlots)
	if len(chosen) == 0 {
		return nil
	}
	quantities := make([]int, len(loot))
	for _, unit := range chosen {
		quantities[unit.dropIndex] += unit.quantity
	}
	kept := make([]LootDrop, 0, len(loot))
	for index, drop := range loot {
		if quantities[index] <= 0 {
			continue
		}
		copyDrop := drop
		copyDrop.Quantity = quantities[index]
		kept = append(kept, copyDrop)
	}
	return kept
}

type secureUnit struct {
	dropIndex int
	quantity  int
	slots     int
	value     int
	rarity    int
}

func flattenSecureUnits(snapshot ScenarioSnapshot, loot []LootDrop) []secureUnit {
	units := make([]secureUnit, 0, len(loot))
	for index, drop := range loot {
		if drop.Quantity <= 0 {
			continue
		}
		if ammo, ok := snapshot.Ammos[drop.ItemID]; ok {
			remaining := drop.Quantity
			for remaining > 0 {
				take := ammo.RoundsPerSlot
				if take <= 0 || take > remaining {
					take = remaining
				}
				slots, _, err := ammoUsage(snapshot, drop.ItemID, take)
				if err != nil || slots <= 0 {
					break
				}
				units = append(units, secureUnit{
					dropIndex: index, quantity: take, slots: slots,
					value: ammo.Price * take, rarity: 40,
				})
				remaining -= take
			}
			continue
		}
		item, ok := snapshot.LootItems[drop.ItemID]
		if !ok || item.Slots <= 0 {
			continue
		}
		rarity := item.DropWeight
		if rarity <= 0 {
			rarity = 40
		}
		for i := 0; i < drop.Quantity; i++ {
			units = append(units, secureUnit{
				dropIndex: index, quantity: 1, slots: item.Slots,
				value: item.Price, rarity: rarity,
			})
		}
	}
	return units
}

type secureMemo struct {
	value  int
	rarity int
	count  int
	take   bool
	seen   bool
}

func knapsackSecureUnits(units []secureUnit, capacity int) []secureUnit {
	memo := make([][]secureMemo, len(units)+1)
	for i := range memo {
		memo[i] = make([]secureMemo, capacity+1)
	}
	var search func(index, cap int) secureMemo
	search = func(index, cap int) secureMemo {
		if index >= len(units) || cap <= 0 {
			return secureMemo{}
		}
		if memo[index][cap].seen {
			return memo[index][cap]
		}
		skip := search(index+1, cap)
		best := secureMemo{value: skip.value, rarity: skip.rarity, count: skip.count}
		unit := units[index]
		if unit.slots > 0 && unit.slots <= cap {
			takeRest := search(index+1, cap-unit.slots)
			take := secureMemo{
				value:  takeRest.value + unit.value,
				rarity: takeRest.rarity + unit.rarity,
				count:  takeRest.count + 1,
				take:   true,
			}
			if take.value > best.value || (take.value == best.value && (take.rarity < best.rarity || (take.rarity == best.rarity && take.count > best.count))) {
				best = take
			}
		}
		best.seen = true
		memo[index][cap] = best
		return best
	}
	search(0, capacity)
	picked := make([]secureUnit, 0)
	for index, cap := 0, capacity; index < len(units) && cap > 0; index++ {
		cell := memo[index][cap]
		if !cell.take {
			continue
		}
		picked = append(picked, units[index])
		cap -= units[index].slots
	}
	return picked
}
