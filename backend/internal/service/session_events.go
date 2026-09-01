// 探索事件服务：保存 RunPlan 事件、提供历史查询，并区分发现与结算结果。
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	sessionEventRunSettled      = "run_settled"
	sessionEventSessionFinished = "session_finished"
	sessionEventSessionFailed   = "session_failed"
	sessionEventLootExtracted   = "loot_extracted"
	sessionEventLootStored      = "loot_stored"
	sessionEventLootOverflow    = "loot_overflow"
	sessionEventAmmoRefilled    = "ammo_refilled"
)

var ErrPendingRunIntegrity = errors.New("待结算探索计划完整性校验失败")

// RunPlan 是一局尚未结算的完整纯引擎结果。
type RunPlan struct {
	RunIndex int
	Input    engine.EngineState
	Result   engine.RunResult
	Events   []engine.TraceEvent
	Hash     string
}

// sessionEventRunIndex 返回终局事件应使用的有效局序号，避免异常会话把事件写入 run_index=0。
func sessionEventRunIndex(sess models.Session) int {
	runIndex := sess.PendingRunIndex
	if runIndex <= 0 {
		runIndex = sess.TotalRuns
	}
	if runIndex <= 0 {
		return 1
	}
	return runIndex
}

// SessionEventView 是前端实时播放和历史回放使用的事件格式。
type SessionEventView struct {
	ID          uint                   `json:"id"`
	SessionID   uint                   `json:"sessionId"`
	RunIndex    int                    `json:"runIndex"`
	Sequence    int                    `json:"sequence"`
	EventType   string                 `json:"eventType"`
	OffsetSec   int64                  `json:"offsetSec"`
	AvailableAt time.Time              `json:"availableAt"`
	NodeID      string                 `json:"nodeId"`
	SubjectID   string                 `json:"subjectId"`
	Payload     map[string]interface{} `json:"payload"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// marshalPendingRun 将局序号、计划输入和纯引擎结果共同纳入 hash，防止 StateJSON 被替换后继续结算旧计划。
func marshalPendingRun(runIndex int, input engine.EngineState, result engine.RunResult) (string, string, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", "", fmt.Errorf("序列化待结算探索结果: %w", err)
	}
	integrityPayload := struct {
		RunIndex int                `json:"runIndex"`
		Input    engine.EngineState `json:"input"`
		Result   engine.RunResult   `json:"result"`
	}{RunIndex: runIndex, Input: input, Result: result}
	integrityJSON, err := json.Marshal(integrityPayload)
	if err != nil {
		return "", "", fmt.Errorf("序列化待结算探索完整性数据: %w", err)
	}
	hash := sha256.Sum256(integrityJSON)
	return string(resultJSON), hex.EncodeToString(hash[:]), nil
}

// decodePendingRun 从 Session 行解码待结算结果并做完整性校验：hash 覆盖局序号、当前状态与结果，防止 StateJSON 被替换后继续结算旧计划。
func decodePendingRun(sess models.Session, currentState engine.EngineState) (engine.RunResult, error) {
	if sess.PendingRunIndex <= 0 || sess.PendingRunResult == "" || sess.PendingRunResult == "{}" {
		return engine.RunResult{}, fmt.Errorf("%w：行动会话缺少待结算 RunPlan", ErrPendingRunIntegrity)
	}
	var result engine.RunResult
	if err := json.Unmarshal([]byte(sess.PendingRunResult), &result); err != nil {
		return engine.RunResult{}, fmt.Errorf("%w：读取待结算探索结果: %w", ErrPendingRunIntegrity, err)
	}
	_, hash, err := marshalPendingRun(sess.PendingRunIndex, currentState, result)
	if err != nil {
		return engine.RunResult{}, err
	}
	if hash != sess.PendingRunHash {
		return engine.RunResult{}, fmt.Errorf("%w：runIndex=%d 或 StateJSON 与计划输入不一致", ErrPendingRunIntegrity, sess.PendingRunIndex)
	}
	if err := engine.ValidateTrace(result.Trace, result.DurationSec); err != nil {
		return engine.RunResult{}, fmt.Errorf("%w：校验待结算探索事件: %w", ErrPendingRunIntegrity, err)
	}
	return result, nil
}

// appendPlannedSessionEvents 将一局预计算的纯引擎事件批量落库，按事件偏移量预排 AvailableAt 供前端按时播放。
func appendPlannedSessionEvents(tx *gorm.DB, userID, sessionID uint, plan RunPlan, runStartAt time.Time) error {
	if err := engine.ValidateTrace(plan.Events, plan.Result.DurationSec); err != nil {
		return fmt.Errorf("校验第%d局探索事件: %w", plan.RunIndex, err)
	}
	for _, event := range plan.Events {
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("序列化第%d局事件%d: %w", plan.RunIndex, event.Sequence, err)
		}
		row := models.SessionEvent{
			UserID: userID, SessionID: sessionID, RunIndex: plan.RunIndex, Sequence: event.Sequence,
			EventType: string(event.Type), OffsetSec: event.OffsetSec,
			AvailableAt: runStartAt.Add(time.Duration(event.OffsetSec) * time.Second),
			NodeID:      event.NodeID, SubjectID: event.SubjectID, PayloadJSON: string(payload), CreatedAt: time.Now(),
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("保存第%d局事件%d: %w", plan.RunIndex, event.Sequence, err)
		}
	}
	return nil
}

// nextSessionEventSequence 查询指定局内当前最大事件序号并加 1，保证同局追加的事件顺序连续。
func nextSessionEventSequence(tx *gorm.DB, sessionID uint, runIndex int) (int, error) {
	var row struct {
		MaxSequence int `gorm:"column:max_sequence"`
	}
	if err := tx.Model(&models.SessionEvent{}).
		Select("COALESCE(MAX(sequence), 0) AS max_sequence").
		Where("session_id = ? AND run_index = ?", sessionID, runIndex).
		Scan(&row).Error; err != nil {
		return 0, fmt.Errorf("读取事件序号: %w", err)
	}
	return row.MaxSequence + 1, nil
}

// appendSessionEventTx 在事务内追加单条会话事件，自动分配序号并序列化 payload。
func appendSessionEventTx(tx *gorm.DB, userID, sessionID uint, runIndex int, eventType string, offsetSec int64, availableAt time.Time, nodeID, subjectID string, payload interface{}) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化事件 %s: %w", eventType, err)
	}
	sequence, err := nextSessionEventSequence(tx, sessionID, runIndex)
	if err != nil {
		return err
	}
	row := models.SessionEvent{
		UserID: userID, SessionID: sessionID, RunIndex: runIndex, Sequence: sequence,
		EventType: eventType, OffsetSec: offsetSec, AvailableAt: availableAt,
		NodeID: nodeID, SubjectID: subjectID, PayloadJSON: string(encoded), CreatedAt: time.Now(),
	}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("保存事件 %s: %w", eventType, err)
	}
	return nil
}

// sessionEventView 将数据库事件行转换为前端视图结构，并解析 payload JSON。
func sessionEventView(row models.SessionEvent) (SessionEventView, error) {
	payload := make(map[string]interface{})
	if row.PayloadJSON != "" {
		if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
			return SessionEventView{}, fmt.Errorf("解析事件%d payload: %w", row.ID, err)
		}
	}
	return SessionEventView{
		ID: row.ID, SessionID: row.SessionID, RunIndex: row.RunIndex, Sequence: row.Sequence,
		EventType: row.EventType, OffsetSec: row.OffsetSec, AvailableAt: row.AvailableAt,
		NodeID: row.NodeID, SubjectID: row.SubjectID, Payload: payload, CreatedAt: row.CreatedAt,
	}, nil
}

// ListSessionEvents 只返回当前时间线已经可见的事件；已结束 Session 以 end_time 截断预计算事件。
func ListSessionEvents(db *gorm.DB, userID, sessionID uint, afterID uint) ([]SessionEventView, error) {
	var sess models.Session
	if err := db.Where("user_id = ? AND id = ?", userID, sessionID).First(&sess).Error; err != nil {
		return nil, err
	}
	cutoff := time.Now()
	if sess.EndTime != nil && sess.EndTime.Before(cutoff) {
		cutoff = *sess.EndTime
	}
	query := db.Where("user_id = ? AND session_id = ? AND available_at <= ?", userID, sessionID, cutoff)
	if afterID > 0 {
		query = query.Where("id > ?", afterID)
	}
	var rows []models.SessionEvent
	if err := query.Order("id asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取行动事件: %w", err)
	}
	result := make([]SessionEventView, 0, len(rows))
	for _, row := range rows {
		view, err := sessionEventView(row)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

// appendLootSettlementEvents 为一批战利品掉落生成结算事件（拾取/入库/溢出等），物品名与类目取自场景快照目录。
func appendLootSettlementEvents(tx *gorm.DB, userID, sessionID uint, runIndex int, snapshot engine.ScenarioSnapshot, loot []engine.LootDrop, eventType string, offsetSec int64, now time.Time) error {
	for _, drop := range loot {
		name, category, err := resolveLootSummary(snapshot, drop)
		if err != nil {
			return err
		}
		if err := appendSessionEventTx(tx, userID, sessionID, runIndex, eventType, offsetSec, now, "", drop.ItemID, map[string]interface{}{
			"dropId": drop.ID, "itemId": drop.ItemID, "name": name, "category": category,
			"quantity": drop.Quantity, "containerId": drop.ContainerID, "source": drop.Source,
		}); err != nil {
			return err
		}
	}
	return nil
}
