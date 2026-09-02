// 跨局运行状态 DTO：描述连续状态、单局输入和纯引擎结果。
package engine

import "sort"

// EngineState 是跨局连续状态，重启时直接从 Session 恢复。
type EngineState struct {
	Character       CharacterState `json:"character"`
	Loadout         LoadoutState   `json:"loadout"`
	ArmorDurability int            `json:"armorDurability"`
	Ammo            CarriedAmmo    `json:"ammo"`       // 当前主弹药摘要（UI/会话行展示与旧状态兼容）
	AmmoStacks      []CarriedAmmo  `json:"ammoStacks"` // 携带弹药池：最多 4 栈，逐栈扣减与返还
	Consumables     []ItemStack    `json:"consumables"`
	CarriedItems    []CarriedItem  `json:"carriedItems"`
	Carry           CarryState     `json:"carry"`
}

// CloneCarriedAmmoStacks 拷贝携带弹药池切片，避免跨局修改共享底层数组。
func CloneCarriedAmmoStacks(stacks []CarriedAmmo) []CarriedAmmo {
	return append([]CarriedAmmo(nil), stacks...)
}

// CarriedAmmoStacks 归一化状态内的携带弹药池：新状态的 AmmoStacks 为空但 Ammo 有值时，
// 视为旧版单栈状态并包装成单栈池，保证存量会话与单栈路径继续可用。
func CarriedAmmoStacks(state *EngineState) []CarriedAmmo {
	if len(state.AmmoStacks) > 0 {
		return state.AmmoStacks
	}
	if state.Ammo.ID != "" && state.Ammo.Rounds > 0 {
		return []CarriedAmmo{state.Ammo}
	}
	return nil
}

// CarriedItem 是行动内携带的聚合补给或耐久实例。
type CarriedItem struct {
	InstanceID        uint    `json:"instanceId"`
	ItemID            string  `json:"itemId"`
	Quantity          int     `json:"quantity"`
	CurrentDurability float64 `json:"currentDurability"`
	MaxDurability     float64 `json:"maxDurability"`
	RaidExtract       bool    `json:"raidExtract"`
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
	StartHP                 float64      `json:"startHp"`
	EndHP                   float64      `json:"endHp"`
	StartEnergy             float64      `json:"startEnergy"`
	EndEnergy               float64      `json:"endEnergy"`
	StartHydration          float64      `json:"startHydration"`
	EndHydration            float64      `json:"endHydration"`
	Loot                    []LootDrop   `json:"loot"`
	ExtractedLoot           []LootDrop   `json:"extractedLoot"`
	ConsumedItems           []ItemStack  `json:"consumedItems"`
	Report                  []string     `json:"report"`
	Trace                   []TraceEvent `json:"trace"`
	NextState               EngineState  `json:"nextState"`
	Finished                bool         `json:"finished"`
	ArmorBrokenDuringRun    bool         `json:"armorBrokenDuringRun,omitempty"`
	SkipResourceConsumption bool         `json:"skipResourceConsumption"`
}

// CloneCarriedItems 拷贝携带物品切片，避免跨局修改共享底层数组。
func CloneCarriedItems(items []CarriedItem) []CarriedItem {
	return append([]CarriedItem(nil), items...)
}

// SortItemStacks 按物品 ID 稳定排序堆叠列表，保证跨局输出顺序确定。
func SortItemStacks(stacks []ItemStack) {
	sort.SliceStable(stacks, func(i, j int) bool {
		return stacks[i].ItemID < stacks[j].ItemID
	})
}
