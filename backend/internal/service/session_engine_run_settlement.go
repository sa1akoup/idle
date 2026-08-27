// Session 单局结算：保存资源快照，并在终局时按结果归还或丢失携行物品。
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

func (s *SessionService) settleEngineRun(sess *models.Session, state *engine.EngineState, snapshot engine.ScenarioSnapshot, result engine.RunResult, runIndex int, runEndAt, deadline time.Time) error {
	inputStateJSON, err := json.Marshal(*state)
	if err != nil {
		return fmt.Errorf("序列化单局输入状态: %w", err)
	}
	var storedLoot, overflowLoot []engine.LootDrop
	var nextPlan RunPlan
	var ammoRefill *ammoRefillResult
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		if err := settleDueHideoutJobsTx(tx, s.userID, time.Now()); err != nil {
			return err
		}
		var existing models.SessionRun
		if err := tx.Where("user_id = ? AND session_id = ? AND run_index = ?", s.userID, sess.ID, runIndex).First(&existing).Error; err == nil {
			if existing.NextState == "" {
				return fmt.Errorf("第%d局已结算但缺少后续状态", runIndex)
			}
			return json.Unmarshal([]byte(existing.NextState), state)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("检查单局幂等状态: %w", err)
		}

		stateAfter := result.NextState
		if result.Result != "incapacitated" {
			result.Result = "success"
		}
		stateAfter.Consumables = itemStacksFromCarriedItems(stateAfter.CarriedItems)
		storedLoot, overflowLoot, err = s.storeSuccessfulLootTx(tx, snapshot, result)
		if err != nil {
			return err
		}
		if err := settleSessionArmorTx(tx, s.userID, stateAfter, result.Result == "incapacitated"); err != nil {
			return err
		}
		if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, stateAfter.Character); err != nil {
			return err
		}

		status := "running"
		terminalReason := ""
		if result.Result == "incapacitated" {
			status = "incapacitated"
			terminalReason = "hp_zero"
		} else if !runEndAt.Before(deadline) || stateAfter.Character.Energy <= 0 || stateAfter.Character.Hydration <= 0 || stateAfter.ArmorDurability <= 0 || result.Finished {
			status = "success"
			switch {
			case !runEndAt.Before(deadline):
				terminalReason = "offline_limit"
			case stateAfter.Character.Energy <= 0 || stateAfter.Character.Hydration <= 0:
				terminalReason = "resource_depleted"
			case stateAfter.ArmorDurability <= 0:
				terminalReason = "armor_broken"
			default:
				terminalReason = "run_complete"
			}
		}

		if status == "running" {
			candidate := stateAfter
			ammoRefill, err = ensureSessionAmmoBeforeNextRun(tx, s.userID, snapshot, &candidate)
			if err != nil {
				if !errors.Is(err, ErrPurchaseUnavailable) {
					return err
				}
				status = "success"
				terminalReason = "ammo_unavailable"
			} else {
				stateAfter = candidate
				nextRunStartAt := runEndAt
				var nextRunAt time.Time
				nextPlan, nextRunAt, err = planEngineRun(sess.EngineVersion, snapshot, sess.Seed, runIndex+1, sess.Style, stateAfter, nextRunStartAt)
				if err != nil {
					return fmt.Errorf("规划第%d局探索: %w", runIndex+1, err)
				}
				sess.CurrentRunStartedAt = &nextRunStartAt
				sess.NextRunAt = &nextRunAt
			}
		}

		if status != "running" {
			if result.Result == "incapacitated" {
				if err := discardSessionLoadoutTx(tx, s.userID, state.Loadout, stateAfter.CarriedItems); err != nil {
					return err
				}
				stateAfter.Ammo = engine.CarriedAmmo{}
				stateAfter.Loadout = engine.LoadoutState{}
				stateAfter.CarriedItems = nil
				stateAfter.Consumables = nil
				stateAfter.Carry.UsedSlots = 0
				stateAfter.Carry.UsedWeight = 0
			} else {
				if err := syncLoadoutConsumablesFromCarriedItemsTx(tx, s.userID, sess.CharacterID, stateAfter.CarriedItems); err != nil {
					return err
				}
				if err := returnCarriedAmmoTx(tx, s.userID, snapshot, &stateAfter); err != nil {
					return err
				}
				if err := returnCarriedItemsTx(tx, s.userID, snapshot, stateAfter.CarriedItems); err != nil {
					return err
				}
				stateAfter.CarriedItems = nil
				stateAfter.Consumables = nil
			}
			if err := createRecoveryPlanTx(tx, s.userID, sess.ID, stateAfter.Character, sess.RecoveryPolicyJSON); err != nil {
				return err
			}
		}

		encodedState, err := json.Marshal(stateAfter)
		if err != nil {
			return fmt.Errorf("序列化单局后状态: %w", err)
		}
		lootJSON, err := encodeEngineLoot(result.ExtractedLoot, snapshot)
		if err != nil {
			return err
		}
		storedLootJSON, err := encodeEngineLoot(storedLoot, snapshot)
		if err != nil {
			return err
		}
		overflowJSON, err := encodeEngineLoot(overflowLoot, snapshot)
		if err != nil {
			return err
		}
		reportJSON, err := encodeEngineReport(result.Report)
		if err != nil {
			return err
		}
		consumedJSON, err := json.Marshal(result.ConsumedItems)
		if err != nil {
			return fmt.Errorf("序列化单局消耗: %w", err)
		}
		itemChangesJSON, err := json.Marshal(result.NextState.CarriedItems)
		if err != nil {
			return fmt.Errorf("序列化物品实例变化: %w", err)
		}
		run := models.SessionRun{
			UserID: s.userID, SessionID: sess.ID, RunIndex: runIndex, Result: result.Result,
			DurationMin: int((result.DurationSec + 59) / 60), DurationSec: result.DurationSec,
			Heat: result.Heat, AmmoUsed: result.AmmoUsed,
			StartHP: result.StartHP, EndHP: result.EndHP, StartEnergy: result.StartEnergy, EndEnergy: result.EndEnergy,
			StartHydration: result.StartHydration, EndHydration: result.EndHydration,
			Loot: lootJSON, StoredLoot: storedLootJSON, OverflowLoot: overflowJSON,
			Consumed: string(consumedJSON), ItemInstanceChanges: string(itemChangesJSON),
			InputState: string(inputStateJSON), NextState: string(encodedState), Report: reportJSON,
		}
		if err := tx.Create(&run).Error; err != nil {
			return fmt.Errorf("保存单局记录: %w", err)
		}

		sess.ElapsedSec += result.DurationSec
		sess.ElapsedMin = int(sess.ElapsedSec / 60)
		sess.TotalRuns = runIndex
		sess.StateJSON = string(encodedState)
		sess.WeaponID = stateAfter.Loadout.WeaponID
		sess.ArmorID = stateAfter.Loadout.ArmorID
		sess.ArmorInstanceID = stateAfter.Loadout.ArmorInstanceID
		sess.AmmoID = stateAfter.Ammo.ID
		sess.AmmoRounds = stateAfter.Ammo.Rounds
		sess.Consumables = stringsFromStacks(stateAfter.Consumables)
		sess.LastProcessedAt = &runEndAt
		now := time.Now()
		updates := map[string]interface{}{
			"status": status, "terminal_reason": terminalReason, "elapsed_sec": sess.ElapsedSec, "elapsed_min": sess.ElapsedMin,
			"total_runs": sess.TotalRuns, "weapon_id": sess.WeaponID, "armor_id": sess.ArmorID,
			"armor_instance_id": sess.ArmorInstanceID,
			"ammo_id":           sess.AmmoID, "ammo_rounds": sess.AmmoRounds, "consumables": sess.Consumables,
			"state_json": sess.StateJSON, "last_processed_at": runEndAt, "heartbeat_at": now,
		}
		if status == "running" {
			pendingResult, pendingHash, err := marshalPendingRun(nextPlan.RunIndex, nextPlan.Input, nextPlan.Result)
			if err != nil {
				return err
			}
			sess.Status = "running"
			sess.PendingRunIndex = nextPlan.RunIndex
			sess.PendingRunResult = pendingResult
			sess.PendingRunHash = pendingHash
			updates["current_run_started_at"] = sess.CurrentRunStartedAt
			updates["next_run_at"] = sess.NextRunAt
			updates["pending_run_index"] = sess.PendingRunIndex
			updates["pending_run_result"] = pendingResult
			updates["pending_run_hash"] = pendingHash
		} else {
			sess.Status = status
			sess.TerminalReason = terminalReason
			sess.EndTime = &now
			sess.CurrentRunStartedAt = nil
			sess.NextRunAt = nil
			sess.PendingRunIndex = 0
			sess.PendingRunResult = "{}"
			sess.PendingRunHash = ""
			updates["end_time"] = now
			updates["current_run_started_at"] = nil
			updates["next_run_at"] = nil
			updates["pending_run_index"] = 0
			updates["pending_run_result"] = "{}"
			updates["pending_run_hash"] = ""
		}
		query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status = ?", s.userID, sess.ID, "running")
		if s.leaseOwner != "" {
			query = query.Where("lease_owner = ? AND lease_until > ?", s.leaseOwner, now)
		}
		updateResult := query.Updates(updates)
		if updateResult.Error != nil {
			return fmt.Errorf("保存行动调度进度: %w", updateResult.Error)
		}
		if updateResult.RowsAffected != 1 {
			return ErrSessionLeaseLost
		}
		if status == "running" {
			if err := appendPlannedSessionEvents(tx, s.userID, sess.ID, nextPlan, *sess.CurrentRunStartedAt); err != nil {
				return err
			}
		}
		if ammoRefill != nil {
			if err := appendSessionEventTx(tx, s.userID, sess.ID, runIndex, sessionEventAmmoRefilled, result.DurationSec, now, "", ammoRefill.ToAmmoID, map[string]interface{}{
				"fromAmmoId": ammoRefill.FromAmmoID, "toAmmoId": ammoRefill.ToAmmoID,
				"fromLevel": ammoRefill.FromLevel, "toLevel": ammoRefill.ToLevel,
				"rounds": ammoRefill.Rounds, "unitPrice": ammoRefill.UnitPrice,
				"totalPrice": ammoRefill.TotalPrice, "source": ammoRefill.Source,
			}); err != nil {
				return err
			}
		}
		if err := appendLootSettlementEvents(tx, s.userID, sess.ID, runIndex, snapshot, result.ExtractedLoot, sessionEventLootExtracted, result.DurationSec, now); err != nil {
			return err
		}
		if err := appendLootSettlementEvents(tx, s.userID, sess.ID, runIndex, snapshot, storedLoot, sessionEventLootStored, result.DurationSec, now); err != nil {
			return err
		}
		if err := appendLootSettlementEvents(tx, s.userID, sess.ID, runIndex, snapshot, overflowLoot, sessionEventLootOverflow, result.DurationSec, now); err != nil {
			return err
		}
		if err := appendSessionEventTx(tx, s.userID, sess.ID, runIndex, sessionEventRunSettled, result.DurationSec, now, "", "", map[string]interface{}{
			"result": result.Result, "status": status, "durationSec": result.DurationSec, "heat": result.Heat, "ammoUsed": result.AmmoUsed,
		}); err != nil {
			return err
		}
		if status != "running" {
			if err := appendSessionEventTx(tx, s.userID, sess.ID, runIndex, sessionEventSessionFinished, result.DurationSec, now, "", "", map[string]interface{}{
				"status": status, "result": result.Result, "reason": terminalReason,
			}); err != nil {
				return err
			}
		}
		*state = stateAfter
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *SessionService) storeSuccessfulLootTx(tx *gorm.DB, snapshot engine.ScenarioSnapshot, result engine.RunResult) ([]engine.LootDrop, []engine.LootDrop, error) {
	if result.Result == "incapacitated" || len(result.ExtractedLoot) == 0 {
		return nil, nil, nil
	}
	stored, overflow, err := fitEngineLootToStorage(tx, s.userID, result.ExtractedLoot)
	if err != nil {
		return nil, nil, err
	}
	for _, drop := range stored {
		item, err := snapshotCatalogItem(snapshot, drop.ItemID)
		if err != nil {
			return nil, nil, err
		}
		if err := addInventoryItem(tx, s.userID, item, drop.Quantity, true); err != nil {
			return nil, nil, err
		}
	}
	return stored, overflow, nil
}

func settleSessionArmorTx(tx *gorm.DB, userID uint, state engine.EngineState, incapacitated bool) error {
	if state.Loadout.ArmorID == "" && state.Loadout.ArmorInstanceID == 0 {
		return nil
	}
	var armor models.ArmorInstance
	query := tx.Where("user_id = ? AND status IN ?", userID, []string{"normal", "broken"})
	if state.Loadout.ArmorInstanceID > 0 {
		query = query.Where("id = ?", state.Loadout.ArmorInstanceID)
	} else {
		query = query.Where("armor_id = ?", state.Loadout.ArmorID).Order("id asc")
	}
	err := query.First(&armor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取结算护甲: %w", err)
	}
	if incapacitated {
		return tx.Delete(&armor).Error
	}
	status := "normal"
	if state.ArmorDurability <= 0 {
		status = "broken"
	}
	return tx.Model(&models.ArmorInstance{}).Where("user_id = ? AND id = ?", userID, armor.ID).Updates(map[string]interface{}{
		"cur_durability": maxInt(state.ArmorDurability, 0), "status": status,
	}).Error
}

func discardSessionLoadoutTx(tx *gorm.DB, userID uint, loadout engine.LoadoutState, carriedItems []engine.CarriedItem) error {
	ids := []string{loadout.WeaponID, loadout.ArmorID, loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID}
	for _, itemID := range ids {
		if itemID == "" {
			continue
		}
		if err := removeInventoryItem(tx, userID, itemID, 1); err != nil {
			return err
		}
	}
	if err := discardCarriedItemsTx(tx, userID, carriedItems); err != nil {
		return err
	}
	if err := discardSessionItemInstancesTx(tx, userID); err != nil {
		return fmt.Errorf("清理失能携行实例: %w", err)
	}
	return tx.Model(&models.PlayerLoadout{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"weapon_id": "", "armor_id": "", "chest_rig_id": "", "backpack_id": "", "helmet_id": "", "headset_id": "", "consumables": "[]", "consumable_refs": "[]",
	}).Error
}

func ensureSessionAmmoBeforeNextRun(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, state *engine.EngineState) (*ammoRefillResult, error) {
	return ensureSessionAmmoTx(tx, userID, snapshot, state)
}

func lootQuantityEngine(loot []engine.LootDrop) int {
	total := 0
	for _, drop := range loot {
		total += drop.Quantity
	}
	return total
}

func stringsFromStacks(stacks []engine.ItemStack) string {
	ids := make([]string, 0)
	for _, stack := range stacks {
		for i := 0; i < stack.Quantity; i++ {
			ids = append(ids, stack.ItemID)
		}
	}
	return strings.Join(ids, ",")
}
