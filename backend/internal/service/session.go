// 探索会话服务：负责创建、查询和中止 Session，具体模拟由纯引擎 worker 执行。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

type SessionService struct {
	db         *gorm.DB
	userID     uint
	leaseOwner string
}

func NewSessionService(db *gorm.DB, userID uint) *SessionService {
	return &SessionService{db: db, userID: userID}
}

func NewSessionServiceWithLease(db *gorm.DB, userID uint, leaseOwner string) *SessionService {
	return &SessionService{db: db, userID: userID, leaseOwner: leaseOwner}
}

// StartReq 启动挂机会话的请求：地图、行动风格与失能后的预设装备序号。
type StartReq struct {
	MapID          string   `json:"mapId"`
	Style          string   `json:"style"`
	RecoveryPreset int      `json:"recoveryPreset"`
	AmmoID         string   `json:"ammoId"`
	AmmoRounds     int      `json:"ammoRounds"`
	WeaponID       string   `json:"-"`
	ArmorID        string   `json:"-"`
	ChestRigID     string   `json:"-"`
	BackpackID     string   `json:"-"`
	HelmetID       string   `json:"-"`
	HeadsetID      string   `json:"-"`
	Consumables    []string `json:"-"`
}

const defaultOfflineLimitMin = 1440

// Start 在创建时固定场景快照和完整连续状态，worker 只消费这些持久化输入。
func (s *SessionService) Start(req StartReq) (*models.Session, error) {
	style, err := engine.ResolveStyle(req.Style)
	if err != nil {
		return nil, err
	}
	req.Style = string(style)
	if req.RecoveryPreset < 1 || req.RecoveryPreset > 3 {
		return nil, fmt.Errorf("失败预设装备序号需为 1-3")
	}
	now := time.Now()
	var sess models.Session
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		// 事务内重新检查活跃 Session，确保启动与出售/换装使用同一把用户资源锁。
		var running int64
		if err := tx.Model(&models.Session{}).Where("user_id = ? AND status IN ?", s.userID, []string{"running", "waiting_injury"}).Count(&running).Error; err != nil {
			return fmt.Errorf("读取行动状态: %w", err)
		}
		if running > 0 {
			return fmt.Errorf("已有行动正在进行")
		}
		// 快照、连续状态和 Session 必须来自同一个事务，确保启动时三者严格对应。
		var txCharacter models.Character
		if err := tx.Where("user_id = ?", s.userID).First(&txCharacter).Error; err != nil {
			return fmt.Errorf("读取玩家角色: %w", err)
		}
		if txCharacter.Injury != "" && txCharacter.Injury != "none" && txCharacter.InjuryUntil != nil && now.Before(*txCharacter.InjuryUntil) {
			return fmt.Errorf("角色伤势恢复中，剩余 %v", time.Until(*txCharacter.InjuryUntil).Round(time.Second))
		}
		txLoadout, err := GetPlayerLoadoutForUser(tx, s.userID)
		if err != nil {
			return err
		}
		if err := validateOwnedLoadoutForUser(tx, s.userID, txLoadout.WeaponID, txLoadout.ArmorID, txLoadout.Consumables, txLoadout.ChestRigID, txLoadout.BackpackID, txLoadout.HelmetID, txLoadout.HeadsetID); err != nil {
			return err
		}
		presetWeaponID, presetArmorID, _ := PresetOf(txLoadout, req.RecoveryPreset)
		if presetWeaponID == "" || presetArmorID == "" {
			return fmt.Errorf("预设装备 %d 未配置，请先在角色页面配置", req.RecoveryPreset)
		}
		snapshot, snapshotJSON, snapshotHash, err := buildScenarioSnapshotTx(tx, s.userID, req.MapID)
		if err != nil {
			return err
		}
		carriedAmmo, err := reserveCarriedAmmo(tx, s.userID, snapshot, txLoadout.WeaponID, req.AmmoID, req.AmmoRounds)
		if err != nil {
			return fmt.Errorf("配置探索弹药: %w", err)
		}
		state, err := buildEngineState(tx, s.userID, txCharacter, txLoadout, carriedAmmo)
		if err != nil {
			return err
		}
		stateJSON, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("序列化探索初始状态: %w", err)
		}
		seed := now.UnixNano()
		plan, nextRunAt, err := planEngineRun(engine.EngineVersion, snapshot, seed, 1, req.Style, state, now)
		if err != nil {
			return fmt.Errorf("规划首局探索: %w", err)
		}
		pendingRunResult, pendingRunHash, err := marshalPendingRun(plan.RunIndex, plan.Input, plan.Result)
		if err != nil {
			return fmt.Errorf("序列化首局探索计划: %w", err)
		}
		sess = models.Session{
			UserID: s.userID, CharacterID: txCharacter.ID, MapID: req.MapID, Style: req.Style, RecoveryPreset: req.RecoveryPreset,
			WeaponID: txLoadout.WeaponID, ArmorID: txLoadout.ArmorID, AmmoID: carriedAmmo.ID, AmmoRounds: carriedAmmo.Rounds,
			Consumables: strings.Join(txLoadout.Consumables, ","), Status: "running",
			Seed: seed, StartTime: now, OfflineLimitMin: defaultOfflineLimitMin, OfflineLimitSec: int64(defaultOfflineLimitMin) * 60,
			ElapsedMin: 0, ElapsedSec: 0, CurrentRunStartedAt: &now, NextRunAt: &nextRunAt, LastProcessedAt: nil, EngineVersion: engine.EngineVersion,
			ScenarioSnapshot: snapshotJSON, ScenarioHash: snapshotHash, InitialStateJSON: string(stateJSON), StateJSON: string(stateJSON),
			PendingRunIndex: plan.RunIndex, PendingRunResult: pendingRunResult, PendingRunHash: pendingRunHash,
		}
		if err := tx.Create(&sess).Error; err != nil {
			return fmt.Errorf("创建行动会话: %w", err)
		}
		if err := appendPlannedSessionEvents(tx, s.userID, sess.ID, plan, now); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	dispatchSessionWorker(s.db, s.userID, sess.ID)
	return &sess, nil
}

func (s *SessionService) runSession(id uint) {
	if err := s.simulateSession(id); err != nil {
		if errors.Is(err, ErrSessionLeaseLost) {
			log.Printf("session %d lease lost: %v", id, err)
			return
		}
		if failErr := s.failSession(id, err); failErr != nil {
			log.Printf("session %d failed: %v; status update failed: %v", id, err, failErr)
			return
		}
		log.Printf("session %d failed: %v", id, err)
	}
}

func (s *SessionService) failSession(id uint, cause error) error {
	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		var sess models.Session
		if err := tx.Where("user_id = ? AND id = ?", s.userID, id).First(&sess).Error; err != nil {
			return err
		}
		if sess.Status != "running" && sess.Status != "waiting_injury" {
			return nil
		}
		query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status IN ?", s.userID, id, []string{"running", "waiting_injury"})
		if s.leaseOwner != "" {
			query = query.Where("lease_owner = ? AND lease_until > ?", s.leaseOwner, now)
		}
		result := query.Updates(map[string]interface{}{
			"status": "failed", "end_time": now, "current_run_started_at": nil, "next_run_at": nil,
			"pending_run_index": 0, "pending_run_result": "{}", "pending_run_hash": "", "heartbeat_at": now,
		})
		if result.Error != nil {
			return fmt.Errorf("保存行动失败状态: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return nil
		}
		snapshot, state, err := decodeSessionAmmoState(sess)
		if err != nil {
			return err
		}
		if err := returnCarriedAmmoTx(tx, s.userID, snapshot, &state); err != nil {
			return err
		}
		encodedState, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("序列化失败 Session 状态: %w", err)
		}
		if err := tx.Model(&models.Session{}).Where("user_id = ? AND id = ?", s.userID, id).
			Updates(map[string]interface{}{"state_json": string(encodedState), "ammo_id": "", "ammo_rounds": 0}).Error; err != nil {
			return fmt.Errorf("保存失败 Session 弹药状态: %w", err)
		}
		runIndex := sess.PendingRunIndex
		if runIndex <= 0 {
			runIndex = sess.TotalRuns
		}
		if runIndex <= 0 {
			runIndex = 1
		}
		return appendSessionEventTx(tx, s.userID, id, runIndex, sessionEventSessionFailed, sess.ElapsedSec, now, "", "", map[string]interface{}{
			"status": "failed", "reason": cause.Error(),
		})
	})
}

func (s *SessionService) GetSession(id uint) (*models.Session, []models.SessionRun, error) {
	var sess models.Session
	if err := s.db.Where("user_id = ? AND id = ?", s.userID, id).First(&sess).Error; err != nil {
		return nil, nil, err
	}
	var runs []models.SessionRun
	if err := s.db.Where("user_id = ? AND session_id = ?", s.userID, id).Order("run_index asc").Find(&runs).Error; err != nil {
		return nil, nil, fmt.Errorf("读取行动记录: %w", err)
	}
	return &sess, runs, nil
}

func (s *SessionService) ListSessions() ([]models.Session, error) {
	var list []models.Session
	err := s.db.Where("user_id = ?", s.userID).Order("id desc").Limit(20).Find(&list).Error
	return list, err
}

func (s *SessionService) Abort(id uint) error {
	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		var sess models.Session
		if err := tx.Where("user_id = ? AND id = ?", s.userID, id).First(&sess).Error; err != nil {
			return err
		}
		if sess.Status != "running" && sess.Status != "waiting_injury" {
			return nil
		}
		result := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status IN ?", s.userID, id, []string{"running", "waiting_injury"}).Updates(map[string]interface{}{
			"status": "aborted", "end_time": now, "current_run_started_at": nil, "next_run_at": nil,
			"pending_run_index": 0, "pending_run_result": "{}", "pending_run_hash": "",
		})
		if result.Error != nil {
			return fmt.Errorf("中止行动: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return nil
		}
		snapshot, state, err := decodeSessionAmmoState(sess)
		if err != nil {
			return err
		}
		if err := returnCarriedAmmoTx(tx, s.userID, snapshot, &state); err != nil {
			return err
		}
		encodedState, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("序列化中止 Session 状态: %w", err)
		}
		if err := tx.Model(&models.Session{}).Where("user_id = ? AND id = ?", s.userID, id).
			Updates(map[string]interface{}{"state_json": string(encodedState), "ammo_id": "", "ammo_rounds": 0}).Error; err != nil {
			return fmt.Errorf("保存中止 Session 弹药状态: %w", err)
		}
		runIndex := sess.PendingRunIndex
		if runIndex <= 0 {
			runIndex = sess.TotalRuns
		}
		if runIndex <= 0 {
			runIndex = 1
		}
		return appendSessionEventTx(tx, s.userID, id, runIndex, sessionEventSessionAborted, sess.ElapsedSec, now, "", "", map[string]interface{}{
			"status": "aborted", "reason": "user_abort",
		})
	})
}

func splitIDs(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func maxInt(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func getAttrValue(character *models.Character, attribute string) int {
	switch attribute {
	case "strength":
		return character.Strength
	case "agility":
		return character.Agility
	case "intellect":
		return character.Intellect
	case "charisma":
		return character.Charisma
	case "perception":
		return character.Perception
	case "stealth":
		return character.Stealth
	case "negotiation":
		return character.Negotiation
	case "engineering":
		return character.Engineering
	case "medical":
		return character.Medical
	case "luck":
		return character.Luck
	case "survival":
		return character.Survival
	case "resist":
		return character.Resist
	default:
		return 50
	}
}
