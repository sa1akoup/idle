// 跨局运行状态 DTO：描述连续状态、单局输入和纯引擎结果。
package engine

import "sort"

// EngineState 是跨局连续状态，重启时直接从 Session 恢复。
type EngineState struct {
	Character       CharacterState `json:"character"`
	Loadout         LoadoutState   `json:"loadout"`
	ArmorDurability int            `json:"armorDurability"`
	Ammo            CarriedAmmo    `json:"ammo"`
	Consumables     []ItemStack    `json:"consumables"`
	Carry           CarryState     `json:"carry"`
}

type RunInput struct {
	SessionSeed int64       `json:"sessionSeed"`
	RunIndex    int         `json:"runIndex"`
	Style       string      `json:"style"`
	State       EngineState `json:"state"`
}

type LootDrop struct {
	ID          string `json:"id"`
	ItemID      string `json:"itemId"`
	Quantity    int    `json:"quantity"`
	ContainerID string `json:"containerId"`
	Source      string `json:"source"`
}

type RunResult struct {
	Result                  string       `json:"result"`
	DurationSec             int64        `json:"durationSec"` // 探索时间轴秒；配置中的1分钟按60秒换算
	Heat                    int          `json:"heat"`
	AmmoUsed                int          `json:"ammoUsed"`
	Injury                  string       `json:"injury"`
	Loot                    []LootDrop   `json:"loot"`
	ExtractedLoot           []LootDrop   `json:"extractedLoot"`
	ConsumedItems           []ItemStack  `json:"consumedItems"`
	Report                  []string     `json:"report"`
	Trace                   []TraceEvent `json:"trace"`
	NextState               EngineState  `json:"nextState"`
	Finished                bool         `json:"finished"`
	SkipResourceConsumption bool         `json:"skipResourceConsumption"`
}

func CloneItemStacks(stacks []ItemStack) []ItemStack {
	return append([]ItemStack(nil), stacks...)
}

func SortItemStacks(stacks []ItemStack) {
	sort.SliceStable(stacks, func(i, j int) bool {
		return stacks[i].ItemID < stacks[j].ItemID
	})
}
