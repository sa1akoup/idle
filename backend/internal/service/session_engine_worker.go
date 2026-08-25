// 探索引擎适配层：负责现实时间调度、纯引擎调用和连续状态恢复。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	maxCatchUpRunsPerWorker = 100
	lightInjuryWaitSec      = 10
	heavyInjuryWaitSec      = 30
	lethalInjuryWaitSec     = 60
)

func planEngineRun(engineVersion string, snapshot engine.ScenarioSnapshot, seed int64, runIndex int, style string, state engine.EngineState, runStartAt time.Time) (RunPlan, time.Time, error) {
	input := engine.RunInput{SessionSeed: seed, RunIndex: runIndex, Style: style, State: state}
	result, err := engine.SimulateRunVersion(engineVersion, snapshot, input)
	if err != nil {
		return RunPlan{}, time.Time{}, fmt.Errorf("执行第%d局探索: %w", runIndex, err)
	}
	if result.DurationSec <= 0 {
		return RunPlan{}, time.Time{}, fmt.Errorf("第%d局探索耗时无效", runIndex)
	}
	if err := engine.ValidateTrace(result.Trace, result.DurationSec); err != nil {
		return RunPlan{}, time.Time{}, fmt.Errorf("校验第%d局探索事件: %w", runIndex, err)
	}
	_, hash, err := marshalPendingRun(runIndex, state, result)
	if err != nil {
		return RunPlan{}, time.Time{}, err
	}
	return RunPlan{RunIndex: runIndex, Input: state, Result: result, Events: result.Trace, Hash: hash}, runStartAt.Add(time.Duration(result.DurationSec) * time.Second), nil
}

func (s *SessionService) simulateSession(id uint) error {
	var sess models.Session
	if err := s.db.Where("user_id = ? AND id = ?", s.userID, id).First(&sess).Error; err != nil {
		return fmt.Errorf("读取行动会话: %w", err)
	}
	if sess.Status == "aborted" || sess.Status == "finished" || sess.Status == "failed" {
		return nil
	}
	if err := s.ensureSessionLease(sess.ID); err != nil {
		return err
	}
	if sess.ScenarioSnapshot == "" || sess.StateJSON == "" || sess.NextRunAt == nil {
		return fmt.Errorf("行动会话缺少引擎快照或调度状态")
	}
	if sess.Status == "running" && sess.CurrentRunStartedAt == nil {
		return fmt.Errorf("行动会话缺少当前局开始时间")
	}
	if sess.Status == "running" && sess.PendingRunIndex <= 0 {
		return fmt.Errorf("行动会话缺少待结算局计划")
	}
	if strings.TrimSpace(sess.EngineVersion) == "" {
		return fmt.Errorf("行动会话缺少引擎版本")
	}
	var snapshot engine.ScenarioSnapshot
	if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &snapshot); err != nil {
		return fmt.Errorf("读取行动场景快照: %w", err)
	}
	if err := engine.ValidateSnapshot(snapshot); err != nil {
		return fmt.Errorf("校验行动场景快照: %w", err)
	}
	snapshotHash, err := engine.SnapshotHash(snapshot)
	if err != nil {
		return fmt.Errorf("计算行动场景快照 hash: %w", err)
	}
	if snapshotHash != sess.ScenarioHash {
		return fmt.Errorf("行动场景快照 hash 不匹配")
	}
	var state engine.EngineState
	if err := json.Unmarshal([]byte(sess.StateJSON), &state); err != nil {
		return fmt.Errorf("读取行动连续状态: %w", err)
	}
	if sess.OfflineLimitSec <= 0 {
		return fmt.Errorf("行动离线时限无效")
	}

	for runCount := 0; runCount < maxCatchUpRunsPerWorker; runCount++ {
		if err := s.refreshEngineSession(&sess); err != nil {
			return err
		}
		if sess.Status == "aborted" || sess.Status == "finished" || sess.Status == "failed" {
			return nil
		}
		now := time.Now()
		if sess.Status == "waiting_injury" {
			if sess.NextRunAt == nil || sess.NextRunAt.After(now) {
				return nil
			}
			if err := s.resumeAfterInjury(&sess, &state, snapshot, now); err != nil {
				return err
			}
			continue
		}
		if sess.Status != "running" || sess.CurrentRunStartedAt == nil || sess.NextRunAt == nil || sess.NextRunAt.After(now) {
			return nil
		}
		deadline := sess.StartTime.Add(time.Duration(sess.OfflineLimitSec) * time.Second)
		if !sess.CurrentRunStartedAt.Before(deadline) {
			return s.finishEngineSession(&sess, &state, snapshot, now, "offline_limit", "")
		}
		runIndex := sess.PendingRunIndex
		if runIndex != sess.TotalRuns+1 {
			return fmt.Errorf("待结算局序号无效：期望%d，实际%d", sess.TotalRuns+1, runIndex)
		}
		result, err := decodePendingRun(sess, state)
		if err != nil {
			return fmt.Errorf("读取第%d局探索计划: %w", runIndex, err)
		}
		// current_run_started_at 是本局起点，next_run_at 是持久化的预计结算时间。
		runStartAt := *sess.CurrentRunStartedAt
		runEndAt := *sess.NextRunAt
		if result.DurationSec <= 0 {
			return fmt.Errorf("第%d局探索耗时无效", runIndex)
		}
		if runEndAt.Before(runStartAt) {
			return fmt.Errorf("第%d局探索时间游标无效", runIndex)
		}
		if runEndAt.Sub(runStartAt) != time.Duration(result.DurationSec)*time.Second {
			return fmt.Errorf("第%d局探索计划与调度时间不一致", runIndex)
		}
		if err := s.settleEngineRun(&sess, &state, snapshot, result, runIndex, runEndAt, deadline); err != nil {
			return err
		}
		if sess.Status == "finished" || sess.Status == "aborted" {
			return nil
		}
	}
	return nil
}

func (s *SessionService) refreshEngineSession(sess *models.Session) error {
	var current models.Session
	if err := s.db.Where("user_id = ? AND id = ?", s.userID, sess.ID).First(&current).Error; err != nil {
		return fmt.Errorf("刷新行动调度状态: %w", err)
	}
	*sess = current
	return s.ensureSessionLease(sess.ID)
}

func (s *SessionService) ensureSessionLease(sessionID uint) error {
	if s.leaseOwner == "" {
		return nil
	}
	now := time.Now()
	leaseUntil := now.Add(30 * time.Second)
	result := s.db.Model(&models.Session{}).
		Where("user_id = ? AND id = ? AND status IN ? AND lease_owner = ? AND lease_until > ?", s.userID, sessionID, []string{"running", "waiting_injury"}, s.leaseOwner, now).
		Updates(map[string]interface{}{"lease_until": leaseUntil, "heartbeat_at": now})
	if result.Error != nil {
		return fmt.Errorf("续租行动调度: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrSessionLeaseLost
	}
	return nil
}

func (s *SessionService) releaseSessionLease(sessionID uint) {
	if s.leaseOwner == "" {
		return
	}
	s.db.Model(&models.Session{}).Where("user_id = ? AND id = ? AND lease_owner = ?", s.userID, sessionID, s.leaseOwner).Updates(map[string]interface{}{"lease_owner": "", "lease_until": nil})
}

func (s *SessionService) resumeAfterInjury(sess *models.Session, state *engine.EngineState, snapshot engine.ScenarioSnapshot, now time.Time) error {
	if sess.NextRunAt == nil {
		return fmt.Errorf("伤势恢复行动缺少下一局开始时间")
	}
	runStartAt := *sess.NextRunAt
	var plan RunPlan
	var nextRunAt time.Time
	var pendingRunResult string
	var pendingRunHash string
	var refill *ammoRefillResult
	var finalStateJSON string
	finished := false
	finishDetail := ""
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		// 先在副本中清除伤势，再准备下一局资源；系统错误会回滚这次事务。
		candidate := *state
		candidate.Character.Injury = "none"
		candidateRefill, resourceErr := ensureSessionAmmoTx(tx, s.userID, snapshot, &candidate)
		if resourceErr != nil {
			if !errors.Is(resourceErr, ErrPurchaseUnavailable) {
				return fmt.Errorf("准备伤势恢复后的探索资源: %w", resourceErr)
			}
			finishDetail = classifyResourceUnavailable(resourceErr, "ammo_unavailable")
			if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, candidate.Character, nil); err != nil {
				return err
			}
			finalStateJSON, resourceErr = s.finishEngineSessionTx(tx, sess, &candidate, snapshot, now, "resource_unavailable", finishDetail)
			if resourceErr != nil {
				return resourceErr
			}
			*state = candidate
			finished = true
			return nil
		}
		*state = candidate
		refill = candidateRefill
		if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, state.Character, nil); err != nil {
			return err
		}
		var err error
		plan, nextRunAt, err = planEngineRun(sess.EngineVersion, snapshot, sess.Seed, sess.TotalRuns+1, sess.Style, *state, runStartAt)
		if err != nil {
			return fmt.Errorf("规划伤势恢复后的探索: %w", err)
		}
		pendingRunResult, pendingRunHash, err = marshalPendingRun(plan.RunIndex, plan.Input, plan.Result)
		if err != nil {
			return fmt.Errorf("序列化伤势恢复后的探索计划: %w", err)
		}
		encoded, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("序列化恢复状态: %w", err)
		}
		query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status = ? AND next_run_at <= ?", s.userID, sess.ID, "waiting_injury", now)
		if s.leaseOwner != "" {
			query = query.Where("lease_owner = ? AND lease_until > ?", s.leaseOwner, now)
		}
		result := query.Updates(map[string]interface{}{
			"status": "running", "state_json": string(encoded), "current_run_started_at": runStartAt, "next_run_at": nextRunAt, "heartbeat_at": now,
			"pending_run_index": plan.RunIndex, "pending_run_result": pendingRunResult, "pending_run_hash": pendingRunHash,
		})
		if result.Error != nil {
			return fmt.Errorf("恢复行动运行状态: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("行动伤势恢复状态已被其他 worker 修改")
		}
		if refill != nil {
			if err := appendSessionEventTx(tx, s.userID, sess.ID, plan.RunIndex, sessionEventAmmoRefilled, 0, runStartAt, "", refill.ToAmmoID, map[string]interface{}{
				"fromAmmoId": refill.FromAmmoID, "toAmmoId": refill.ToAmmoID,
				"fromLevel": refill.FromLevel, "toLevel": refill.ToLevel,
				"rounds": refill.Rounds, "unitPrice": refill.UnitPrice,
				"totalPrice": refill.TotalPrice, "source": refill.Source,
			}); err != nil {
				return err
			}
		}
		if err := appendPlannedSessionEvents(tx, s.userID, sess.ID, plan, runStartAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if finished {
		sess.Status = "finished"
		sess.EndTime = &now
		sess.CurrentRunStartedAt = nil
		sess.NextRunAt = nil
		sess.PendingRunIndex = 0
		sess.PendingRunResult = "{}"
		sess.PendingRunHash = ""
		sess.StateJSON = finalStateJSON
		sess.AmmoID = ""
		sess.AmmoRounds = 0
		return nil
	}
	sess.Status = "running"
	sess.CurrentRunStartedAt = &runStartAt
	sess.NextRunAt = &nextRunAt
	sess.HeartbeatAt = &now
	sess.PendingRunIndex = plan.RunIndex
	sess.PendingRunResult = pendingRunResult
	sess.PendingRunHash = pendingRunHash
	return nil
}

func (s *SessionService) finishEngineSessionTx(tx *gorm.DB, sess *models.Session, state *engine.EngineState, snapshot engine.ScenarioSnapshot, now time.Time, reason, detail string) (string, error) {
	query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status IN ?", s.userID, sess.ID, []string{"running", "waiting_injury"})
	if s.leaseOwner != "" {
		query = query.Where("lease_owner = ? AND lease_until > ?", s.leaseOwner, now)
	}
	result := query.Updates(map[string]interface{}{
		"status": "finished", "end_time": now, "current_run_started_at": nil,
		"next_run_at": nil, "pending_run_index": 0, "pending_run_result": "{}", "pending_run_hash": "", "heartbeat_at": now,
	})
	if result.Error != nil {
		return "", fmt.Errorf("保存行动完成状态: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return "", ErrSessionLeaseLost
	}
	if err := returnCarriedAmmoTx(tx, s.userID, snapshot, state); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("序列化行动最终状态: %w", err)
	}
	finalStateJSON := string(encoded)
	if err := tx.Model(&models.Session{}).Where("user_id = ? AND id = ?", s.userID, sess.ID).
		Updates(map[string]interface{}{"state_json": finalStateJSON, "ammo_id": "", "ammo_rounds": 0}).Error; err != nil {
		return "", fmt.Errorf("保存行动最终弹药状态: %w", err)
	}
	runIndex := sess.PendingRunIndex
	if runIndex <= 0 {
		runIndex = sess.TotalRuns
	}
	if runIndex <= 0 {
		runIndex = 1
	}
	payload := map[string]interface{}{"status": "finished", "reason": reason}
	if detail != "" {
		payload["detail"] = detail
	}
	if err := appendSessionEventTx(tx, s.userID, sess.ID, runIndex, sessionEventSessionFinished, sess.ElapsedSec, now, "", "", payload); err != nil {
		return "", err
	}
	return finalStateJSON, nil
}

func (s *SessionService) finishEngineSession(sess *models.Session, state *engine.EngineState, snapshot engine.ScenarioSnapshot, now time.Time, reason, detail string) error {
	var finalStateJSON string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		var err error
		finalStateJSON, err = s.finishEngineSessionTx(tx, sess, state, snapshot, now, reason, detail)
		return err
	})
	if err != nil {
		return err
	}
	sess.Status = "finished"
	sess.EndTime = &now
	sess.CurrentRunStartedAt = nil
	sess.NextRunAt = nil
	sess.PendingRunIndex = 0
	sess.PendingRunResult = "{}"
	sess.PendingRunHash = ""
	sess.StateJSON = finalStateJSON
	sess.AmmoID = ""
	sess.AmmoRounds = 0
	return nil
}
