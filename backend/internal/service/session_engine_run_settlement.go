// Session 单局结算：在一个事务内完成资源消耗、掉落、恢复、下一局计划和事件落库。
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

func (s *SessionService) settleEngineRun(sess *models.Session, state *engine.EngineState, snapshot engine.ScenarioSnapshot, result engine.RunResult, runIndex int, runEndAt, deadline time.Time) error {
	inputStateJSON, err := json.Marshal(*state)
	if err != nil {
		return fmt.Errorf("序列化单局输入状态: %w", err)
	}
	storedLoot, overflowLoot := []engine.LootDrop(nil), []engine.LootDrop(nil)
	var nextPlan RunPlan
	var ammoRefill *ammoRefillResult
	finishReason := ""
	finishDetail := ""
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, s.userID); err != nil {
			return err
		}
		var existing models.SessionRun
		if err := tx.Where("user_id = ? AND session_id = ? AND run_index = ?", s.userID, sess.ID, runIndex).First(&existing).Error; err == nil {
			if existing.NextState == "" {
				return fmt.Errorf("第%d局已结算但缺少后续状态", runIndex)
			}
			if err := json.Unmarshal([]byte(existing.NextState), state); err != nil {
				return fmt.Errorf("恢复第%d局结算状态: %w", runIndex, err)
			}
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("检查单局幂等状态: %w", err)
		}
		stateAfter := result.NextState
		if !result.SkipResourceConsumption {
			if err := consumeEngineResources(tx, s.userID, result); err != nil {
				return err
			}
		}
		// 正常撤离先同步剩余补给，失能局保留整套当前携行，随后由丢装逻辑统一扣除。
		if result.Result != "incapacitated" {
			if err := syncLoadoutConsumables(tx, s.userID, sess.CharacterID, stateAfter.Consumables); err != nil {
				return err
			}
		}
		var err error
		storedLoot, overflowLoot, err = fitEngineLootToStorage(tx, s.userID, result.ExtractedLoot)
		if err != nil {
			return err
		}
		for _, drop := range storedLoot {
			item, err := snapshotCatalogItem(snapshot, drop.ItemID)
			if err != nil {
				return err
			}
			if err := addInventoryItem(tx, s.userID, item, drop.Quantity, true); err != nil {
				return err
			}
		}

		var armorInstance models.ArmorInstance
		if stateAfter.Loadout.ArmorID != "" {
			if err := tx.Where("user_id = ? AND armor_id = ? AND status = ?", s.userID, stateAfter.Loadout.ArmorID, "normal").Order("id asc").First(&armorInstance).Error; err != nil {
				return fmt.Errorf("读取结算护甲: %w", err)
			}
		}
		if result.Result != "incapacitated" && armorInstance.ID != 0 {
			armorStatus := "normal"
			if stateAfter.ArmorDurability <= 0 {
				armorStatus = "broken"
				result.Finished = true
				finishReason = "run_finished"
				finishDetail = "armor_broken"
				result.Report = appendEngineReport(result.Report, ">> 护甲耐久归零，行动结束，需先维修护甲")
			}
			if err := tx.Model(&models.ArmorInstance{}).Where("user_id = ? AND id = ?", s.userID, armorInstance.ID).Updates(map[string]interface{}{"cur_durability": maxInt(stateAfter.ArmorDurability, 0), "status": armorStatus}).Error; err != nil {
				return fmt.Errorf("保存护甲耐久: %w", err)
			}
		}

		// 先落库压力与熟练度，失能补购完成后再统一写入最终伤势。
		baseCharacter := stateAfter.Character
		baseCharacter.Injury = "none"
		if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, baseCharacter, nil); err != nil {
			return err
		}

		if result.Result == "incapacitated" {
			// Session 携弹已在启动时移出仓库，失能时剩余弹药随当前携行一并丢失。
			stateAfter.Ammo = engine.CarriedAmmo{}
			operationKey := fmt.Sprintf("session:%d:run:%d:recovery", sess.ID, runIndex)
			operation, replay, err := claimEconomicOperation(tx, s.userID, operationKey, "session_recovery")
			if err != nil {
				return err
			} else if !replay {
				if err := replaceLostLoadoutTx(tx, s.userID, sess.RecoveryPreset, snapshot, state.Loadout, state.Consumables); err != nil {
					if !errors.Is(err, ErrPurchaseUnavailable) {
						return fmt.Errorf("处理失能丢装: %w", err)
					}
					finishReason = "resource_unavailable"
					finishDetail = classifyResourceUnavailable(err, "loadout_unavailable")
					operation.ResultJSON = `{"ok":false}`
					result.Report = appendEngineReport(result.Report, fmt.Sprintf(">> 携行装备已丢失，按预设 %d 补购失败：%s", sess.RecoveryPreset, err))
					result.Finished = true
					stateAfter.Loadout = engine.LoadoutState{}
					stateAfter.ArmorDurability = 0
					stateAfter.Consumables = nil
					stateAfter.Carry.UsedSlots = 0
					stateAfter.Carry.UsedWeight = 0
				} else {
					result.Finished = false
					var character models.Character
					if err := tx.Where("user_id = ? AND id = ?", s.userID, sess.CharacterID).First(&character).Error; err != nil {
						return fmt.Errorf("读取补购后的角色: %w", err)
					}
					loadout, err := GetPlayerLoadoutForUser(tx, s.userID)
					if err != nil {
						return err
					}
					preset := snapshot.RecoveryPresets[sess.RecoveryPreset]
					recoveredAmmo, recoveryRefill, ammoErr := reservePresetAmmoTx(tx, s.userID, snapshot, preset)
					if ammoErr != nil && !errors.Is(ammoErr, ErrPurchaseUnavailable) {
						return ammoErr
					}
					if ammoErr != nil {
						finishReason = "resource_unavailable"
						finishDetail = classifyResourceUnavailable(ammoErr, "ammo_unavailable")
						result.Finished = true
						operation.ResultJSON = `{"ok":false}`
						result.Report = appendEngineReport(result.Report, fmt.Sprintf(">> 预设装备补购完成，但弹药补给失败，行动结束：%s", ammoErr))
					} else {
						ammoRefill = recoveryRefill
						result.Report = appendEngineReport(result.Report, fmt.Sprintf(">> 携行装备已丢失，按预设 %d 补购完成，准备继续探索", sess.RecoveryPreset))
					}
					stateAfter, err = buildEngineState(tx, s.userID, character, loadout, recoveredAmmo)
					if err != nil {
						return err
					}
					stateAfter.Character.Injury = result.Injury
				}
				if operation.ResultJSON == "{}" {
					operation.ResultJSON = `{"ok":true}`
				}
				if err := tx.Save(operation).Error; err != nil {
					return fmt.Errorf("保存失能补购结果: %w", err)
				}
			}
		}
		if !result.Finished && runEndAt.Before(deadline) {
			candidate := stateAfter
			refill, refillErr := ensureSessionAmmoTx(tx, s.userID, snapshot, &candidate)
			if refillErr != nil && !errors.Is(refillErr, ErrPurchaseUnavailable) {
				return refillErr
			}
			if refillErr != nil {
				finishReason = "resource_unavailable"
				finishDetail = classifyResourceUnavailable(refillErr, "ammo_unavailable")
				result.Finished = true
				result.Report = appendEngineReport(result.Report, fmt.Sprintf(">> 弹药耗尽且自动补给失败，行动结束：%s", refillErr))
			} else {
				stateAfter = candidate
				ammoRefill = refill
				if refill != nil {
					result.Report = appendEngineReport(result.Report, fmt.Sprintf(">> N%d 弹药耗尽，自动补充 %d 发 N%d 弹药，花费 ￥%d", refill.FromLevel, refill.Rounds, refill.ToLevel, refill.TotalPrice))
				}
			}
		}
		injuryUntil := (*time.Time)(nil)
		if result.Injury != "none" && !result.Finished && runEndAt.Before(deadline) {
			until := runEndAt.Add(time.Duration(injuryWaitSeconds(result.Injury)) * time.Second)
			injuryUntil = &until
			stateAfter.Character.Injury = result.Injury
		} else {
			stateAfter.Character.Injury = "none"
		}
		if err := updateCharacterFromEngine(tx, s.userID, sess.CharacterID, stateAfter.Character, injuryUntil); err != nil {
			return err
		}

		stateAfter.Consumables = engine.CloneItemStacks(stateAfter.Consumables)
		if err := syncLoadoutConsumables(tx, s.userID, sess.CharacterID, stateAfter.Consumables); err != nil {
			return err
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
		report := result.Report
		if len(storedLoot) > 0 {
			report = appendEngineReport(report, fmt.Sprintf(">> 实际带回 %d 件物品", lootQuantityEngine(storedLoot)))
		}
		if lootQuantityEngine(overflowLoot) > 0 {
			report = appendEngineReport(report, fmt.Sprintf(">> 基地仓库空间不足，放弃 %d 件物品", lootQuantityEngine(overflowLoot)))
		}
		reportJSON, err := encodeEngineReport(report)
		if err != nil {
			return err
		}
		consumedJSON, err := json.Marshal(result.ConsumedItems)
		if err != nil {
			return fmt.Errorf("序列化单局消耗: %w", err)
		}
		overflowJSON, err := encodeEngineLoot(overflowLoot, snapshot)
		if err != nil {
			return err
		}
		run := models.SessionRun{UserID: s.userID, SessionID: sess.ID, RunIndex: runIndex, Result: result.Result, DurationMin: int((result.DurationSec + 59) / 60), DurationSec: result.DurationSec, Heat: result.Heat, AmmoUsed: result.AmmoUsed, Injury: result.Injury, Loot: lootJSON, StoredLoot: storedLootJSON, OverflowLoot: overflowJSON, Consumed: string(consumedJSON), InputState: string(inputStateJSON), NextState: string(encodedState), Report: reportJSON}
		if err := tx.Create(&run).Error; err != nil {
			return fmt.Errorf("保存单局记录: %w", err)
		}
		sess.ElapsedSec += result.DurationSec
		sess.ElapsedMin = int(sess.ElapsedSec / 60)
		sess.TotalRuns = runIndex
		nextRunAt := runEndAt
		var currentRunStartedAt *time.Time
		status := "running"
		if result.Injury != "none" && !result.Finished && runEndAt.Before(deadline) {
			nextRunAt = runEndAt.Add(time.Duration(injuryWaitSeconds(result.Injury)) * time.Second)
			status = "waiting_injury"
		}
		if result.Finished || !runEndAt.Before(deadline) {
			status = "finished"
		}
		if status != "finished" {
			weapon, ok := snapshot.Weapons[stateAfter.Loadout.WeaponID]
			if !ok {
				return fmt.Errorf("下一局武器 %s 不在场景快照中", stateAfter.Loadout.WeaponID)
			}
			if weapon.AmmoPerRound > 0 && stateAfter.Ammo.Rounds < weapon.AmmoPerRound {
				status = "finished"
				if finishReason == "" {
					finishReason = "resource_unavailable"
					finishDetail = "ammo_unavailable"
				}
			}
		}
		if status == "finished" && finishReason == "" {
			if !runEndAt.Before(deadline) {
				finishReason = "offline_limit"
			} else {
				finishReason = "run_finished"
			}
		}
		if status == "running" {
			nextRunStartAt := runEndAt
			var plannedNextRunAt time.Time
			nextPlan, plannedNextRunAt, err = planEngineRun(sess.EngineVersion, snapshot, sess.Seed, runIndex+1, sess.Style, stateAfter, nextRunStartAt)
			if err != nil {
				return fmt.Errorf("规划第%d局探索: %w", runIndex+1, err)
			}
			currentRunStartedAt = &nextRunStartAt
			nextRunAt = plannedNextRunAt
		}
		if status == "finished" {
			if err := returnCarriedAmmoTx(tx, s.userID, snapshot, &stateAfter); err != nil {
				return err
			}
			encodedState, err = json.Marshal(stateAfter)
			if err != nil {
				return fmt.Errorf("序列化行动终态: %w", err)
			}
			if err := tx.Model(&models.SessionRun{}).Where("user_id = ? AND id = ?", s.userID, run.ID).
				Update("next_state", string(encodedState)).Error; err != nil {
				return fmt.Errorf("保存单局终态弹药: %w", err)
			}
		}
		sess.WeaponID = stateAfter.Loadout.WeaponID
		sess.ArmorID = stateAfter.Loadout.ArmorID
		sess.AmmoID = stateAfter.Ammo.ID
		sess.AmmoRounds = stateAfter.Ammo.Rounds
		sess.Consumables = stringsFromStacks(stateAfter.Consumables)
		sess.StateJSON = string(encodedState)
		sess.CurrentRunStartedAt = currentRunStartedAt
		sess.NextRunAt = &nextRunAt
		sess.Status = status
		now := time.Now()
		sess.LastProcessedAt = &runEndAt
		pendingRunIndex := 0
		pendingRunResult := "{}"
		pendingRunHash := ""
		if status == "running" {
			pendingRunIndex = nextPlan.RunIndex
			pendingRunResult, pendingRunHash, err = marshalPendingRun(nextPlan.RunIndex, nextPlan.Input, nextPlan.Result)
			if err != nil {
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
		if err := appendSessionEventTx(tx, s.userID, sess.ID, runIndex, sessionEventRunSettled, result.DurationSec, now, "", "", map[string]interface{}{
			"result": result.Result, "status": status, "durationSec": result.DurationSec,
			"heat": result.Heat, "ammoUsed": result.AmmoUsed, "injury": result.Injury,
		}); err != nil {
			return err
		}
		if status == "finished" {
			if err := appendSessionEventTx(tx, s.userID, sess.ID, runIndex, sessionEventSessionFinished, result.DurationSec, now, "", "", map[string]interface{}{
				"status": "finished", "result": result.Result, "reason": finishReason, "detail": finishDetail,
			}); err != nil {
				return err
			}
		}
		if status == "running" {
			if err := appendPlannedSessionEvents(tx, s.userID, sess.ID, nextPlan, *currentRunStartedAt); err != nil {
				return err
			}
		}
		var nextRunAtValue interface{}
		if status != "finished" {
			nextRunAtValue = nextRunAt
		}
		updates := map[string]interface{}{
			"status": status, "end_time": nil, "elapsed_sec": sess.ElapsedSec, "elapsed_min": sess.ElapsedMin, "total_runs": sess.TotalRuns,
			"weapon_id": sess.WeaponID, "armor_id": sess.ArmorID, "ammo_id": sess.AmmoID, "ammo_rounds": sess.AmmoRounds,
			"consumables": sess.Consumables, "state_json": sess.StateJSON,
			"current_run_started_at": currentRunStartedAt, "last_processed_at": runEndAt, "next_run_at": nextRunAtValue, "heartbeat_at": now,
			"pending_run_index": pendingRunIndex, "pending_run_result": pendingRunResult, "pending_run_hash": pendingRunHash,
		}
		if status == "finished" {
			updates["end_time"] = now
		}
		query := tx.Model(&models.Session{}).Where("user_id = ? AND id = ? AND status IN ?", s.userID, sess.ID, []string{"running", "waiting_injury"})
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
		sess.PendingRunIndex = pendingRunIndex
		sess.PendingRunResult = pendingRunResult
		sess.PendingRunHash = pendingRunHash
		if status == "finished" {
			sess.NextRunAt = nil
		} else {
			sess.NextRunAt = &nextRunAt
		}
		*state = stateAfter
		return nil
	}); err != nil {
		return err
	}
	if sess.Status == "waiting_injury" {
		log.Printf("session %d waiting injury until %s", sess.ID, sess.NextRunAt.Format(time.RFC3339))
	}
	return nil
}

func lootQuantityEngine(loot []engine.LootDrop) int {
	total := 0
	for _, drop := range loot {
		total += drop.Quantity
	}
	return total
}

func stringsFromStacks(stacks []engine.ItemStack) string {
	ids := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		for i := 0; i < stack.Quantity; i++ {
			ids = append(ids, stack.ItemID)
		}
	}
	return strings.Join(ids, ",")
}
