// 探索会话服务：负责创建、查询和后台推进 Session，具体模拟由纯引擎 worker 执行。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

type SessionService struct {
	db         *gorm.DB
	userID     uint
	leaseOwner string
	scheduler  *SessionScheduler
}

// NewSessionService 创建基础会话服务；未配置调度器与租约时仅支持查询类操作。
func NewSessionService(db *gorm.DB, userID uint) *SessionService {
	return &SessionService{db: db, userID: userID}
}

// NewSessionServiceWithScheduler 创建带调度器的会话服务，Start 后可派发后台 worker 推进模拟。
func NewSessionServiceWithScheduler(db *gorm.DB, userID uint, scheduler *SessionScheduler) *SessionService {
	return &SessionService{db: db, userID: userID, scheduler: scheduler}
}

// NewSessionServiceWithLease 创建带租约归属的会话服务，写状态时携带租约条件防重入。
func NewSessionServiceWithLease(db *gorm.DB, userID uint, leaseOwner string) *SessionService {
	return &SessionService{db: db, userID: userID, leaseOwner: leaseOwner}
}

// StartReq 启动挂机会话的请求：地图、行动风格、恢复策略与失能后的预设装备序号。
// 携带弹药不在此配置：弹药槽在角色页设置，开局按随身弹药槽从仓库预留。
type StartReq struct {
	MapID          string          `json:"mapId"`
	Style          string          `json:"style"`
	RecoveryPreset int             `json:"recoveryPreset"`
	RecoveryPolicy *RecoveryPolicy `json:"recoveryPolicy"`
	WeaponID       string          `json:"-"`
	ArmorID        string          `json:"-"`
	ChestRigID     string          `json:"-"`
	BackpackID     string          `json:"-"`
	HelmetID       string          `json:"-"`
	HeadsetID      string          `json:"-"`
	Consumables    []string        `json:"-"`
}

const defaultOfflineLimitMin = 1440

// Start 在创建时固定场景快照和完整连续状态，worker 只消费这些持久化输入。
func (s *SessionService) Start(req StartReq) (*models.Session, error) {
	if s.scheduler == nil {
		return nil, ErrSessionSchedulerNotConfigured
	}
	style, err := engine.ResolveStyle(req.Style)
	if err != nil {
		return nil, err
	}
	req.Style = string(style)
	if req.RecoveryPreset < 1 || req.RecoveryPreset > 3 {
		return nil, fmt.Errorf("失败预设装备序号需为 1-3")
	}
	recoveryPolicyJSON, err := recoveryPolicyJSONForStart(req.RecoveryPolicy)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var sess models.Session
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		if err := settleRecoveryForUserTx(tx, s.userID); err != nil {
			return err
		}
		if err := ensureRecoveryReadyForStartTx(tx, s.userID); err != nil {
			return err
		}
		// 事务内重新检查活跃 Session，确保启动与出售/换装使用同一把用户资源锁。
		var running int64
		if err := tx.Model(&models.Session{}).Where("user_id = ? AND status = ?", s.userID, "running").Count(&running).Error; err != nil {
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
		if txCharacter.HP <= 0 || txCharacter.Energy <= 0 || txCharacter.Hydration <= 0 {
			return fmt.Errorf("角色正在恢复中，请等待生命、能量和饮水恢复")
		}
		txLoadout, err := GetPlayerLoadoutForUser(tx, s.userID)
		if err != nil {
			return err
		}
		if isEmptyCurrentLoadout(txLoadout) {
			if err := restoreLostLoadoutForStartTx(tx, s.userID, txLoadout, req.RecoveryPreset); err != nil {
				return err
			}
			txLoadout, err = GetPlayerLoadoutForUser(tx, s.userID)
			if err != nil {
				return err
			}
			// 没有历史失能 Session 时，按当前保存的预设价格快照补购，避免部署页可启动但实际没有装备。
			if isEmptyCurrentLoadout(txLoadout) {
				fallbackSnapshot, _, _, snapshotErr := buildScenarioSnapshotTx(tx, s.userID, req.MapID)
				if snapshotErr != nil {
					return snapshotErr
				}
				if err := purchaseRecoveryPresetTx(tx, s.userID, req.RecoveryPreset, fallbackSnapshot, txLoadout.ID); err != nil {
					return fmt.Errorf("按恢复预设补购当前装备: %w", err)
				}
				txLoadout, err = GetPlayerLoadoutForUser(tx, s.userID)
				if err != nil {
					return err
				}
			}
		}
		if err := validateOwnedLoadoutForUser(tx, s.userID, txLoadout.WeaponID, txLoadout.ArmorID, txLoadout.Consumables, txLoadout.ChestRigID, txLoadout.BackpackID, txLoadout.HelmetID, txLoadout.HeadsetID, txLoadout.KeyCaseID, txLoadout.SecureContainerID); err != nil {
			return err
		}
		if txLoadout.WeaponID == "" {
			return fmt.Errorf("当前装备缺少武器，请先装备武器或清空整套装备使用恢复预设")
		}
		presetWeaponID, presetArmorID, _ := PresetOf(txLoadout, req.RecoveryPreset)
		if presetWeaponID == "" || presetArmorID == "" {
			return fmt.Errorf("预设装备 %d 未配置，请先在角色页面配置", req.RecoveryPreset)
		}
		snapshot, snapshotJSON, snapshotHash, err := buildScenarioSnapshotTx(tx, s.userID, req.MapID)
		if err != nil {
			return err
		}
		// 携带弹药槽以角色页为准；槽位为空时回退本套预设的默认弹药（失能恢复/新账号首战仍可启动）。
		ammoCells := compactAmmoCells(txLoadout.CarriedAmmo)
		if len(ammoCells) == 0 {
			if weapon, ok := snapshot.Weapons[txLoadout.WeaponID]; ok && weapon.AmmoPerRound > 0 {
				presetAmmoID, presetAmmoRounds := PresetAmmoOf(txLoadout, req.RecoveryPreset)
				if presetAmmoID != "" && presetAmmoRounds > 0 {
					ammoCells = []models.AmmoCell{{AmmoID: presetAmmoID, Rounds: presetAmmoRounds}}
				}
			}
		}
		carriedStacks, err := reserveCarriedAmmoStacks(tx, s.userID, snapshot, txLoadout.WeaponID, ammoCells)
		if err != nil {
			return fmt.Errorf("配置探索弹药: %w", err)
		}
		seed := now.UnixNano()
		if err := refillKeyCaseTx(tx, s.userID, txLoadout.KeyCaseID); err != nil {
			return err
		}
		state, err := buildEngineState(tx, s.userID, txCharacter, txLoadout, carriedStacks)
		if err != nil {
			return err
		}
		if err := reserveCarriedItemsTx(tx, s.userID, state.CarriedItems, fmt.Sprintf("session-seed:%d", seed)); err != nil {
			return err
		}
		stateJSON, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("序列化探索初始状态: %w", err)
		}
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
			WeaponID: txLoadout.WeaponID, ArmorID: txLoadout.ArmorID, ArmorInstanceID: state.Loadout.ArmorInstanceID, AmmoID: state.Ammo.ID, AmmoRounds: state.Ammo.Rounds,
			Consumables: stringsFromStacks(state.Consumables), Status: "running", RecoveryPolicyJSON: recoveryPolicyJSON,
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
	s.scheduler.dispatchSessionWorker(s.userID, sess.ID)
	return &sess, nil
}

// isEmptyCurrentLoadout 判断当前配装是否为空（无任何装备与补给），空配装无法直接开局。
func isEmptyCurrentLoadout(loadout *models.PlayerLoadout) bool {
	return loadout.WeaponID == "" && loadout.ArmorID == "" && loadout.ChestRigID == "" && loadout.BackpackID == "" &&
		loadout.HelmetID == "" && loadout.HeadsetID == "" && len(loadout.Consumables) == 0 && len(loadout.ConsumableRefs) == 0
}

// restoreLostLoadoutForStartTx 开局配装为空时，从最近一次失能会话按预设自动补购装备；无失能记录则直接忽略。
func restoreLostLoadoutForStartTx(tx *gorm.DB, userID uint, loadout *models.PlayerLoadout, presetIndex int) error {
	var previous models.Session
	if err := tx.Where("user_id = ? AND status = ?", userID, "incapacitated").Order("id desc").First(&previous).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("读取失能行动: %w", err)
	}
	var snapshot engine.ScenarioSnapshot
	if err := json.Unmarshal([]byte(previous.ScenarioSnapshot), &snapshot); err != nil {
		return fmt.Errorf("读取失能行动补购快照: %w", err)
	}
	if err := purchaseRecoveryPresetTx(tx, userID, presetIndex, snapshot, loadout.ID); err != nil {
		return fmt.Errorf("自动补购失能预设: %w", err)
	}
	return nil
}

// runSession 执行一次会话推进；租约被抢占时静默退出，其余错误写入失败状态并完成异常结算。
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

// failSession 将运行中会话标记为失败并异常结算：只归还最后一次持久化状态并清空实例，游戏内失能才执行丢装。
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
		if sess.Status != "running" {
			return nil
		}
		query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status = ?", s.userID, id, "running")
		if s.leaseOwner != "" {
			query = query.Where("lease_owner = ? AND lease_until > ?", s.leaseOwner, now)
		}
		result := query.Updates(map[string]interface{}{
			"status": "failed", "terminal_reason": "internal_error", "end_time": now, "current_run_started_at": nil, "next_run_at": nil,
			"pending_run_index": 0, "pending_run_result": "{}", "pending_run_hash": "", "heartbeat_at": now,
		})
		if result.Error != nil {
			return fmt.Errorf("保存行动失败状态: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return nil
		}
		snapshot, state, snapErr := decodeSessionAmmoState(sess)
		if snapErr != nil {
			// 旧版本快照解码失败时必须继续完成失败结算：若在此回滚，
			// 会话将永远停留在 running 并被调度器死循环重派，用户也无法开新局。
			log.Printf("session %d 快照解码失败，按无资产异常结束处理: %v", id, snapErr)
		}
		if snapErr == nil {
			// 技术异常只归还最后一次已持久化状态，游戏内失能才执行丢装。
			if err := settleSessionArmorTx(tx, s.userID, state, false); err != nil {
				return err
			}
			if err := syncLoadoutConsumablesFromCarriedItemsTx(tx, s.userID, sess.CharacterID, state.CarriedItems); err != nil {
				return err
			}
			if err := returnCarriedAmmoTx(tx, s.userID, snapshot, &state); err != nil {
				return err
			}
			if err := returnCarriedItemsTx(tx, s.userID, snapshot, state.CarriedItems); err != nil {
				return err
			}
			if err := settleSecureKeysTx(tx, s.userID, state.CarriedItems); err != nil {
				return err
			}
			state.Ammo = engine.CarriedAmmo{}
			state.CarriedItems = nil
			state.Consumables = nil
			state.Carry.UsedSlots = 0
			state.Carry.UsedWeight = 0
			if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, state.Character); err != nil {
				return err
			}
			if err := createRecoveryPlanTx(tx, s.userID, sess.ID, state.Character, sess.RecoveryPolicyJSON); err != nil {
				return err
			}
			encodedState, err := json.Marshal(state)
			if err != nil {
				return fmt.Errorf("序列化失败 Session 状态: %w", err)
			}
			if err := tx.Model(&models.Session{}).Where("user_id = ? AND id = ?", s.userID, id).
				Updates(map[string]interface{}{"state_json": string(encodedState), "weapon_id": state.Loadout.WeaponID, "armor_id": state.Loadout.ArmorID, "armor_instance_id": state.Loadout.ArmorInstanceID, "ammo_id": "", "ammo_rounds": 0, "consumables": ""}).Error; err != nil {
				return fmt.Errorf("保存失败 Session 弹药状态: %w", err)
			}
		} else {
			// 无快照可还原：不动装备/角色字段，仅清空弹药与补给标记，保持会话行语义干净。
			if err := tx.Model(&models.Session{}).Where("user_id = ? AND id = ?", s.userID, id).
				Updates(map[string]interface{}{"ammo_id": "", "ammo_rounds": 0, "consumables": ""}).Error; err != nil {
				return fmt.Errorf("保存失败 Session 弹药状态: %w", err)
			}
		}
		if err := discardSessionItemInstancesTx(tx, s.userID); err != nil {
			return fmt.Errorf("清理异常 Session 实例: %w", err)
		}
		runIndex := sessionEventRunIndex(sess)
		payload := map[string]interface{}{
			"status": "failed", "reason": cause.Error(),
		}
		if snapErr != nil {
			payload["snapshotDecodeFailed"] = true
		}
		return appendSessionEventTx(tx, s.userID, id, runIndex, sessionEventSessionFailed, sess.ElapsedSec, now, "", "", payload)
	})
}

// GetSession 查询单个会话及其全部局记录，供详情与回放使用。
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

// ListSessions 查询该用户最近 20 条会话（倒序），用于会话列表展示。
func (s *SessionService) ListSessions() ([]models.Session, error) {
	var list []models.Session
	err := s.db.Where("user_id = ?", s.userID).Order("id desc").Limit(20).Find(&list).Error
	return list, err
}

// maxInt 返回 value 与 floor 中的较大者，用于数值下限约束。
func maxInt(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

// getAttrValue 按属性名读取角色属性值，未知属性名时返回默认值 50。
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
