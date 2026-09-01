// 探索事件轨迹：纯引擎只产生相对时间事件，持久化时间由 service 层补齐。
package engine

import "fmt"

type TraceEventType string

const (
	TraceRunStarted              TraceEventType = "run_started"
	TraceRoutePlanned            TraceEventType = "route_planned"
	TraceNodeEntered             TraceEventType = "node_entered"
	TraceNodeMoveStarted         TraceEventType = "node_move_started"
	TraceEventTriggered          TraceEventType = "event_triggered"
	TraceEvacuationStarted       TraceEventType = "evacuation_started"
	TraceContainerSearchStarted  TraceEventType = "container_search_started"
	TraceLootFound               TraceEventType = "loot_found"
	TraceLootCollected           TraceEventType = "loot_collected"
	TraceContainerSearchFinished TraceEventType = "container_search_finished"
	TraceBattleStarted           TraceEventType = "battle_started"
	TraceBattleAttack            TraceEventType = "battle_attack"
	TraceBattleRound             TraceEventType = "battle_round"
	TraceBattleEscape            TraceEventType = "battle_escape"
	TraceBattleFinished          TraceEventType = "battle_finished"
	TraceExtractionApproach      TraceEventType = "extraction_approach"
	TraceExtractionPointReached  TraceEventType = "extraction_point_reached"
	TraceExtractionCompleted     TraceEventType = "extraction_completed"
)

// TraceEvent 是可重放的局内事件，不包含数据库 ID、用户 ID 或现实时间。
type TraceEvent struct {
	Sequence  int                    `json:"sequence"`
	Type      TraceEventType         `json:"type"`
	OffsetSec int64                  `json:"offsetSec"`
	NodeID    string                 `json:"nodeId,omitempty"`
	SubjectID string                 `json:"subjectId,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// appendTraceEvent 向轨迹追加一条事件并自动分配递增序号，空轨迹指针安全跳过。
func appendTraceEvent(events *[]TraceEvent, eventType TraceEventType, offsetSec int64, nodeID, subjectID string, payload map[string]interface{}) {
	if events == nil {
		return
	}
	*events = append(*events, TraceEvent{
		Sequence:  len(*events) + 1,
		Type:      eventType,
		OffsetSec: offsetSec,
		NodeID:    nodeID,
		SubjectID: subjectID,
		Payload:   payload,
	})
}

// ValidateTrace 检查事件顺序和时间轴，确保事件可安全写入 Session 时间线。
func ValidateTrace(events []TraceEvent, durationSec int64) error {
	previousOffset := int64(0)
	for index, event := range events {
		if event.Sequence != index+1 {
			return fmt.Errorf("事件序号无效：期望%d，实际%d", index+1, event.Sequence)
		}
		if event.Type == "" {
			return fmt.Errorf("第%d个事件缺少类型", event.Sequence)
		}
		if event.OffsetSec < 0 || event.OffsetSec > durationSec {
			return fmt.Errorf("第%d个事件时间偏移无效：%d/%d", event.Sequence, event.OffsetSec, durationSec)
		}
		if index > 0 && event.OffsetSec < previousOffset {
			return fmt.Errorf("第%d个事件时间偏移倒退：%d < %d", event.Sequence, event.OffsetSec, previousOffset)
		}
		previousOffset = event.OffsetSec
	}
	return nil
}
