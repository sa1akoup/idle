// 事件运行状态：保存单局内事件、撤离和时间轴的可变状态。
package engine

import (
	"fmt"
	"sort"
)

// eventRunState 是事件、战斗、搜索和撤离共用的单局内存状态。
type eventRunState struct {
	Character       *CharacterState
	Player          *BattleActor
	Node            Node
	ExtractionPoint *ExtractionPoint
	Mode            string
	Style           string
	Styles          []StylePolicy
	Tuning          Tuning

	EvacuationReason    string
	EvacuationEmergency bool
	EvacuationPending   bool
	EvacuationStarted   bool

	DurationSec   int64
	Heat          int
	AmmoUsed      int
	CarrySlots    int
	CarryWeight   float64
	LootSlots     int
	LootWeight    float64
	CarryBlocked  bool
	VisitSequence int

	SkipDefaultCombat    bool
	SkipSearch           bool
	EncounterRole        string
	NextMoveReductionSec int64

	AvailableItems map[string]int
	ConsumedItems  map[string]int
	CarriedItems   []CarriedItem
	ItemUseDefs    map[string]ItemUseDefinition
	Flags          map[string]bool
	EventCounts    map[string]int
	LastEventVisit map[string]int
	Lines          *[]string
	Trace          *[]TraceEvent

	CollectContainer     func(containerID, source string) error
	CollectContainerPool func(poolID, source string, count int) error
	HasContainerPool     func(poolID string) bool
	DiscardLoot          func(quantity int) (int, error)
	CollectAmmoDrop      func(itemID string, quantity int, source string) error
}

// addTrace 向本局轨迹追加一条事件记录。
func (state *eventRunState) addTrace(eventType TraceEventType, offsetSec int64, nodeID, subjectID string, payload map[string]interface{}) {
	appendTraceEvent(state.Trace, eventType, offsetSec, nodeID, subjectID, payload)
}

// resetNodeActions 进入新节点时清空上节点的行动类状态，保证节点互不污染。
func (state *eventRunState) resetNodeActions() {
	state.SkipDefaultCombat = false
	state.SkipSearch = false
	state.EncounterRole = ""
}

// consumeNextMoveDuration 应用撤离捷径的移动缩减并一次性清零该加成。
func (state *eventRunState) consumeNextMoveDuration(baseSec int64) int64 {
	if baseSec <= 0 {
		state.NextMoveReductionSec = 0
		return 0
	}
	reduction := state.NextMoveReductionSec
	state.NextMoveReductionSec = 0
	if reduction <= 0 {
		return baseSec
	}
	if reduction >= baseSec {
		return 0
	}
	return baseSec - reduction
}

// hasItem 判断携带物品中是否还有某物品的可用数量。
func (state *eventRunState) hasItem(itemID string) bool {
	return itemID != "" && state.AvailableItems[itemID] > 0
}

// consumeItem 消耗一个携带物品，找不到可用实例时返回 false。
func (state *eventRunState) consumeItem(itemID string) bool {
	if itemID == "" || state.AvailableItems[itemID] <= 0 {
		return false
	}
	for index := range state.CarriedItems {
		if state.CarriedItems[index].ItemID != itemID {
			continue
		}
		if state.consumeCarriedItem(index) {
			return true
		}
	}
	return false
}

// consumeCarriedItem 扣减单个携带实例：可堆叠品减数量，实例品按使用耐久扣减。
func (state *eventRunState) consumeCarriedItem(index int) bool {
	// 这里会出现降级写入（读-改-写），由单线程模拟保证确定性，不需要锁。
	if index < 0 || index >= len(state.CarriedItems) {
		return false
	}
	item := &state.CarriedItems[index]
	if state.AvailableItems[item.ItemID] <= 0 {
		return false
	}
	if item.InstanceID == 0 {
		if item.Quantity <= 0 {
			return false
		}
		item.Quantity--
	} else {
		if item.CurrentDurability <= 0 {
			return false
		}
		definition := state.ItemUseDefs[item.ItemID]
		use := definition.UseDurability
		if use <= 0 || use > item.CurrentDurability {
			use = item.CurrentDurability
		}
		item.CurrentDurability -= use
	}
	state.AvailableItems[item.ItemID]--
	state.ConsumedItems[item.ItemID]++
	return true
}

// beginEvacuation 置撤离模式并记录原因，已在撤离中时仅允许升级为紧急。
func (state *eventRunState) beginEvacuation(reason string, emergency bool) bool {
	if state.Mode == runModeEvacuating {
		if emergency && !state.EvacuationEmergency {
			state.EvacuationEmergency = true
			state.EvacuationReason = reason
			*state.Lines = append(*state.Lines, fmt.Sprintf(">> 撤离状态升级为紧急：%s", evacuationReasonName(reason)))
		}
		return false
	}
	state.Mode = runModeEvacuating
	state.EvacuationReason = reason
	state.EvacuationEmergency = emergency
	state.EvacuationPending = true
	state.addTrace(TraceEvacuationStarted, state.DurationSec, state.Node.ID, "", map[string]interface{}{
		"reason": reason, "emergency": emergency,
	})
	level := "常规"
	if emergency {
		level = "紧急"
	}
	*state.Lines = append(*state.Lines, fmt.Sprintf(">> 产生%s撤离意图：%s，开始规划撤离路线", level, evacuationReasonName(reason)))
	return true
}

// evacuationReasonName 把撤离原因码翻译成人类可读文案，未知原因原样返回。
func evacuationReasonName(reason string) string {
	switch reason {
	case "health":
		return "生命过低"
	case "stress":
		return "压力过高"
	case "ammo":
		return "弹药耗尽"
	case "armor":
		return "护甲损坏"
	case "carry_full":
		return "携行容量接近上限"
	case "target_acquired":
		return "已获得高价值目标"
	case "event":
		return "事件要求撤离"
	default:
		return reason
	}
}

// newEventManager 由场景快照构造事件管理器：绑定与遭遇池排序保证处理顺序确定。
func newEventManager(snapshot ScenarioSnapshot) *eventManager {
	definitions := make(map[string]EventDefinition, len(snapshot.Events.Definitions))
	for id, definition := range snapshot.Events.Definitions {
		definitions[id] = definition
	}
	bindings := append([]EventBinding(nil), snapshot.Events.Bindings...)
	sort.SliceStable(bindings, func(i, j int) bool {
		left, right := bindings[i], bindings[j]
		if left.EventID != right.EventID {
			return left.EventID < right.EventID
		}
		if left.ScopeType != right.ScopeType {
			return left.ScopeType < right.ScopeType
		}
		if left.ScopeID != right.ScopeID {
			return left.ScopeID < right.ScopeID
		}
		if left.Phase != right.Phase {
			return left.Phase < right.Phase
		}
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		return left.ID < right.ID
	})
	pools := make(map[string][]EncounterPoolEntry, len(snapshot.Events.EncounterPools))
	for role, entries := range snapshot.Events.EncounterPools {
		pools[role] = append([]EncounterPoolEntry(nil), entries...)
		sort.SliceStable(pools[role], func(i, j int) bool {
			if pools[role][i].EnemyID == pools[role][j].EnemyID {
				return pools[role][i].ID < pools[role][j].ID
			}
			return pools[role][i].EnemyID < pools[role][j].EnemyID
		})
	}
	return &eventManager{gameMap: snapshot.Map, definitions: definitions, bindings: bindings, encounterPool: pools, styles: snapshot.Styles}
}
