// 探索引擎适配层：负责现实时间调度、纯引擎调用和连续状态恢复。
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

const maxCatchUpRunsPerWorker = 100

// planEngineRun 调用纯引擎模拟一局并生成 RunPlan：附带覆盖局序号、输入状态与结果的完整性 hash，并返回本局结束的实际时间。
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

// validateSessionEngineVersion 校验会话引擎版本是否在当前发布白名单内；旧版本由上层迁移收尾。
func validateSessionEngineVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("行动会话缺少引擎版本")
	}
	if !engine.IsSupportedEngineVersion(version) {
		return fmt.Errorf("行动会话使用不受支持的引擎版本 %s，请重新开始", version)
	}
	return nil
}

// normalizeLegacySnapshot 为缺少 v17 调参与耳机目录字段的旧快照补齐仅收尾所需默认值。
func normalizeLegacySnapshot(snapshot *engine.ScenarioSnapshot) {
	if snapshot.Tuning == (engine.Tuning{}) {
		snapshot.Tuning = engine.DefaultTuning()
	}
	if snapshot.Headsets == nil {
		snapshot.Headsets = make(map[string]engine.Headset)
	}
}

// validateLegacySnapshotHash 同时兼容旧快照原始 canonical JSON hash 与当前 DTO hash。
func validateLegacySnapshotHash(snapshotJSON string, snapshot engine.ScenarioSnapshot, expectedHash string) error {
	if expectedHash == "" {
		return fmt.Errorf("旧版本行动场景快照缺少 hash")
	}
	currentHash, err := engine.SnapshotHash(snapshot)
	if err != nil {
		return fmt.Errorf("计算旧版本行动场景快照 hash: %w", err)
	}
	if currentHash == expectedHash {
		return nil
	}
	rawDigest := sha256.Sum256([]byte(snapshotJSON))
	if hex.EncodeToString(rawDigest[:]) != expectedHash {
		return fmt.Errorf("旧版本行动场景快照 hash 不匹配")
	}
	return nil
}

// simulateSession 推进一个会话：校验快照与租约后循环结算所有已到期的局；worker 可随时重启，靠幂等结算与游标校验续跑。
func (s *SessionService) simulateSession(id uint) error {
	var sess models.Session
	if err := s.db.Where("user_id = ? AND id = ?", s.userID, id).First(&sess).Error; err != nil {
		return fmt.Errorf("读取行动会话: %w", err)
	}
	if sess.Status == "success" || sess.Status == "incapacitated" {
		return nil
	}
	if err := s.ensureSessionLease(sess.ID); err != nil {
		return err
	}
	if sess.ScenarioSnapshot == "" || sess.StateJSON == "" {
		return fmt.Errorf("行动会话缺少引擎快照或调度状态")
	}
	if strings.TrimSpace(sess.EngineVersion) == engine.LegacyEngineVersionV16 {
		var snapshot engine.ScenarioSnapshot
		if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &snapshot); err != nil {
			return fmt.Errorf("读取旧版本行动场景快照: %w", err)
		}
		normalizeLegacySnapshot(&snapshot)
		if err := engine.ValidateSnapshot(snapshot); err != nil {
			return fmt.Errorf("校验旧版本行动场景快照: %w", err)
		}
		if err := validateLegacySnapshotHash(sess.ScenarioSnapshot, snapshot, sess.ScenarioHash); err != nil {
			return err
		}
		var state engine.EngineState
		if err := json.Unmarshal([]byte(sess.StateJSON), &state); err != nil {
			return fmt.Errorf("读取旧版本行动连续状态: %w", err)
		}
		return s.finishUnscheduledSession(&sess, &state, snapshot, time.Now(), "engine_migrated")
	}
	if sess.NextRunAt == nil {
		return fmt.Errorf("行动会话缺少调度状态")
	}
	if sess.CurrentRunStartedAt == nil || sess.PendingRunIndex <= 0 {
		return fmt.Errorf("行动会话缺少当前局调度状态")
	}
	if err := validateSessionEngineVersion(sess.EngineVersion); err != nil {
		return err
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
		if sess.Status == "success" || sess.Status == "incapacitated" {
			return nil
		}
		now := time.Now()
		if sess.Status != "running" || sess.CurrentRunStartedAt == nil || sess.NextRunAt == nil || sess.NextRunAt.After(now) {
			return nil
		}
		// 离线时限 = 会话开始时间 + 离线秒数；当前局启动时间不早于该时限则不再开新局，按离线超时正常收尾。
		deadline := sess.StartTime.Add(time.Duration(sess.OfflineLimitSec) * time.Second)
		if !sess.CurrentRunStartedAt.Before(deadline) {
			return s.finishUnscheduledSession(&sess, &state, snapshot, now, "offline_limit")
		}
		// 待结算局序号必须严格等于总局数+1，防止跳局、漏局或重复结算。
		runIndex := sess.PendingRunIndex
		if runIndex != sess.TotalRuns+1 {
			return fmt.Errorf("待结算局序号无效：期望%d，实际%d", sess.TotalRuns+1, runIndex)
		}
		result, err := decodePendingRun(sess, state)
		if err != nil {
			return fmt.Errorf("读取第%d局探索计划: %w", runIndex, err)
		}
		runStartAt := *sess.CurrentRunStartedAt
		runEndAt := *sess.NextRunAt
		// 游标校验：结束时间必须严格等于开始时间加本局耗时，防止计划被篡改或错位。
		if result.DurationSec <= 0 || runEndAt.Before(runStartAt) || runEndAt.Sub(runStartAt) != time.Duration(result.DurationSec)*time.Second {
			return fmt.Errorf("第%d局探索时间游标无效", runIndex)
		}
		if err := s.settleEngineRun(&sess, &state, snapshot, result, runIndex, runEndAt, deadline); err != nil {
			return err
		}
		if sess.Status == "success" || sess.Status == "incapacitated" {
			return nil
		}
	}
	return nil
}

// refreshEngineSession 重新读取会话最新一行并续租，供 catch-up 循环在各局之间感知其他 worker 或租约的变更。
func (s *SessionService) refreshEngineSession(sess *models.Session) error {
	var current models.Session
	if err := s.db.Where("user_id = ? AND id = ?", s.userID, sess.ID).First(&current).Error; err != nil {
		return fmt.Errorf("刷新行动调度状态: %w", err)
	}
	*sess = current
	return s.ensureSessionLease(sess.ID)
}

// ensureSessionLease 将租约续期 30 秒；租约过期或归属已转交其他 worker 时返回 ErrSessionLeaseLost，防两个 worker 双写。
func (s *SessionService) ensureSessionLease(sessionID uint) error {
	if s.leaseOwner == "" {
		return nil
	}
	now := time.Now()
	leaseUntil := now.Add(30 * time.Second)
	result := s.db.Model(&models.Session{}).
		Where("user_id = ? AND id = ? AND status = ? AND lease_owner = ? AND lease_until > ?", s.userID, sessionID, "running", s.leaseOwner, now).
		Updates(map[string]interface{}{"lease_until": leaseUntil, "heartbeat_at": now})
	if result.Error != nil {
		return fmt.Errorf("续租行动调度: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrSessionLeaseLost
	}
	return nil
}

// releaseSessionLease 主动清空租约归属与到期时间，让其他 worker 可以立即接管该会话。
func (s *SessionService) releaseSessionLease(sessionID uint) {
	if s.leaseOwner == "" {
		return
	}
	s.db.Model(&models.Session{}).Where("user_id = ? AND id = ? AND lease_owner = ?", s.userID, sessionID, s.leaseOwner).Updates(map[string]interface{}{"lease_owner": "", "lease_until": nil})
}

// finishUnscheduledSession 对没有待办局的会话做正常收尾：归还携带物与弹药、写恢复计划，并标记为成功结束。
func (s *SessionService) finishUnscheduledSession(sess *models.Session, state *engine.EngineState, snapshot engine.ScenarioSnapshot, now time.Time, reason string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		if err := syncLoadoutConsumablesFromCarriedItemsTx(tx, s.userID, sess.CharacterID, state.CarriedItems); err != nil {
			return err
		}
		if err := returnCarriedAmmoTx(tx, s.userID, snapshot, state); err != nil {
			return err
		}
		if err := returnCarriedItemsTx(tx, s.userID, snapshot, state.CarriedItems); err != nil {
			return err
		}
		if err := settleSecureKeysTx(tx, s.userID, state.CarriedItems); err != nil {
			return err
		}
		state.CarriedItems = nil
		state.Consumables = nil
		state.Carry.UsedSlots = 0
		state.Carry.UsedWeight = 0
		if err := ensureInventoryWithinCapacityTx(tx, s.userID); err != nil {
			return err
		}
		if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, state.Character); err != nil {
			return err
		}
		if err := createRecoveryPlanTx(tx, s.userID, sess.ID, state.Character, sess.RecoveryPolicyJSON); err != nil {
			return err
		}
		if err := grantSessionSuccessReputationTx(tx, s.userID); err != nil {
			return err
		}
		encoded, err := json.Marshal(state)
		if err != nil {
			return err
		}
		query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status = ?", s.userID, sess.ID, "running")
		if s.leaseOwner != "" {
			query = query.Where("lease_owner = ? AND lease_until > ?", s.leaseOwner, now)
		}
		if result := query.Updates(map[string]interface{}{
			"status": "success", "terminal_reason": reason, "end_time": now, "current_run_started_at": nil, "next_run_at": nil,
			"pending_run_index": 0, "pending_run_result": "{}", "pending_run_hash": "", "state_json": string(encoded), "weapon_id": state.Loadout.WeaponID, "armor_id": state.Loadout.ArmorID,
			"armor_instance_id": state.Loadout.ArmorInstanceID, "ammo_id": "", "ammo_rounds": 0, "consumables": "",
		}); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return ErrSessionLeaseLost
		}
		return appendSessionEventTx(tx, s.userID, sess.ID, sessionEventRunIndex(*sess), sessionEventSessionFinished, sess.ElapsedSec, now, "", "", map[string]interface{}{"status": "success", "reason": reason})
	})
}
