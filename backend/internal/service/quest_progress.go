package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ownedRaidExtractQuantityTx(tx *gorm.DB, userID uint, itemID string) (int, error) {
	var inventoryQuantity int
	if err := tx.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND raid_extract = ? AND quantity > 0", userID, itemID, true).
		Select("COALESCE(SUM(quantity), 0)").Scan(&inventoryQuantity).Error; err != nil {
		return 0, fmt.Errorf("统计局内带出库存 %s: %w", itemID, err)
	}
	var instanceQuantity int64
	if err := tx.Model(&models.ItemInstance{}).
		Where("user_id = ? AND item_id = ? AND raid_extract = ? AND location_type = ? AND status = ? AND current_durability > 0",
			userID, itemID, true, "inventory", "normal").
		Count(&instanceQuantity).Error; err != nil {
		return 0, fmt.Errorf("统计局内带出实例 %s: %w", itemID, err)
	}
	return inventoryQuantity + int(instanceQuantity), nil
}

var ErrRaidExtractShortage = errors.New("局内带出数量不足")

func consumeRaidExtractItemsTx(tx *gorm.DB, userID uint, itemID string, quantity int) error {
	if quantity <= 0 {
		return nil
	}
	owned, err := ownedRaidExtractQuantityTx(tx, userID, itemID)
	if err != nil {
		return err
	}
	if owned < quantity {
		return fmt.Errorf("%w：%s", ErrRaidExtractShortage, itemID)
	}
	raid := true
	var inventory models.Inventory
	if err := tx.Where("user_id = ? AND item_id = ? AND raid_extract = ? AND quantity > 0", userID, itemID, true).
		First(&inventory).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("读取局内带出库存 %s: %w", itemID, err)
	}
	fromInventory := 0
	if inventory.ID > 0 {
		fromInventory = inventory.Quantity
		if fromInventory > quantity {
			fromInventory = quantity
		}
		if fromInventory > 0 {
			if err := removeInventoryItemFromSource(tx, userID, itemID, fromInventory, &raid); err != nil {
				return err
			}
			quantity -= fromInventory
		}
	}
	if quantity <= 0 {
		return nil
	}
	var instances []models.ItemInstance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND item_id = ? AND raid_extract = ? AND location_type = ? AND status = ? AND current_durability > 0",
			userID, itemID, true, "inventory", "normal").
		Order("id asc").Limit(quantity).Find(&instances).Error; err != nil {
		return fmt.Errorf("读取局内带出实例 %s: %w", itemID, err)
	}
	if len(instances) < quantity {
		return fmt.Errorf("%w：%s", ErrRaidExtractShortage, itemID)
	}
	for _, instance := range instances {
		if err := tx.Delete(&instance).Error; err != nil {
			return fmt.Errorf("扣除局内带出实例 %d: %w", instance.ID, err)
		}
	}
	return nil
}

func completeQuestTx(tx *gorm.DB, userID uint, def models.QuestDef, state *models.UserQuest) error {
	now := time.Now()
	objective, err := parseQuestObjective(def.ObjectiveJSON)
	if err != nil {
		return err
	}
	required := objective.Quantity
	if required <= 0 {
		required = 1
	}
	progressJSON, err := json.Marshal(models.QuestProgress{Count: required})
	if err != nil {
		return fmt.Errorf("序列化合同进度: %w", err)
	}
	if err := tx.Model(&models.UserQuest{}).Where("id = ? AND user_id = ?", state.ID, userID).
		Updates(map[string]interface{}{
			"status": models.QuestStatusCompleted, "progress_json": string(progressJSON), "completed_at": now,
		}).Error; err != nil {
		return fmt.Errorf("完成合同: %w", err)
	}
	reward, err := parseQuestReward(def.RewardJSON)
	if err != nil {
		return err
	}
	if reward.Cash > 0 {
		if err := addCash(tx, userID, reward.Cash); err != nil {
			return err
		}
	}
	if reward.Reputation > 0 && reward.MerchantID != "" {
		if err := awardReputationTx(tx, userID, reward.MerchantID, reward.Reputation); err != nil {
			return err
		}
	}
	state.Status = models.QuestStatusCompleted
	return nil
}

func activeQuestContractsTx(tx *gorm.DB, userID uint) ([]engine.QuestContract, error) {
	var states []models.UserQuest
	if err := tx.Where("user_id = ? AND status = ?", userID, models.QuestStatusActive).
		Order("quest_id asc").Find(&states).Error; err != nil {
		return nil, fmt.Errorf("读取进行中合同: %w", err)
	}
	if len(states) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(states))
	for _, state := range states {
		ids = append(ids, state.QuestID)
	}
	var defs []models.QuestDef
	if err := tx.Where("id IN ?", ids).Find(&defs).Error; err != nil {
		return nil, fmt.Errorf("读取合同定义: %w", err)
	}
	defByID := make(map[string]models.QuestDef, len(defs))
	for _, def := range defs {
		defByID[def.ID] = def
	}
	contracts := make([]engine.QuestContract, 0, len(states))
	for _, state := range states {
		def := defByID[state.QuestID]
		objective, err := parseQuestObjective(def.ObjectiveJSON)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, engine.QuestContract{
			QuestID: def.ID, Type: def.ObjectiveType, ItemID: objective.ItemID,
			NodeID: objective.NodeID, Kind: objective.Kind, Style: objective.Style, Quantity: objective.Quantity,
		})
	}
	return contracts, nil
}

func consumeUsedStashKeysTx(tx *gorm.DB, userID uint, result *engine.RunResult) error {
	if result == nil || len(result.ConsumedStashItems) == 0 {
		return nil
	}
	remaining := make(map[string]int, len(result.ConsumedStashItems))
	for _, stack := range result.ConsumedStashItems {
		if stack.ItemID == "" || stack.Quantity <= 0 {
			continue
		}
		remaining[stack.ItemID] += stack.Quantity
	}
	peel := func(drops []engine.LootDrop) []engine.LootDrop {
		filtered := make([]engine.LootDrop, 0, len(drops))
		for _, drop := range drops {
			need := remaining[drop.ItemID]
			if need > 0 && drop.Quantity > 0 {
				take := drop.Quantity
				if take > need {
					take = need
				}
				drop.Quantity -= take
				remaining[drop.ItemID] -= take
			}
			if drop.Quantity > 0 {
				filtered = append(filtered, drop)
			}
		}
		return filtered
	}
	result.Loot = peel(result.Loot)
	result.ExtractedLoot = peel(result.ExtractedLoot)
	for itemID, quantity := range remaining {
		if quantity <= 0 {
			continue
		}
		if err := consumeRaidExtractItemsTx(tx, userID, itemID, quantity); err != nil {
			return fmt.Errorf("扣除已使用的仓库钥匙 %s: %w", itemID, err)
		}
	}
	return nil
}

func applyQuestProgressTx(tx *gorm.DB, userID uint, style string, snapshot engine.ScenarioSnapshot, result engine.RunResult) error {
	if result.Result != "success" {
		return nil
	}
	var states []models.UserQuest
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status = ?", userID, models.QuestStatusActive).
		Order("quest_id asc").Find(&states).Error; err != nil {
		return fmt.Errorf("读取待结算合同: %w", err)
	}
	if len(states) == 0 {
		return nil
	}
	contractsByID := make(map[string]engine.QuestContract, len(snapshot.Contracts))
	for _, contract := range snapshot.Contracts {
		contractsByID[contract.QuestID] = contract
	}
	visited := visitedNodesFromTrace(result.Trace)
	kills := defeatedKindsFromTrace(snapshot, result.Trace)
	for i := range states {
		state := &states[i]
		contract, frozen := contractsByID[state.QuestID]
		if !frozen {
			continue
		}
		var def models.QuestDef
		if err := tx.Where("id = ?", state.QuestID).First(&def).Error; err != nil {
			return fmt.Errorf("读取合同 %s: %w", state.QuestID, err)
		}
		objType := contract.Type
		if objType == "" {
			objType = def.ObjectiveType
		}
		if objType == models.QuestObjectiveExtractItem {
			continue
		}
		required := contract.Quantity
		if required <= 0 {
			required = 1
		}
		progress, err := parseQuestProgress(state.ProgressJSON)
		if err != nil {
			return err
		}
		gained := 0
		switch objType {
		case models.QuestObjectiveSurviveRuns:
			gained = 1
		case models.QuestObjectiveStyleExtract:
			if style == contract.Style {
				gained = 1
			}
		case models.QuestObjectiveVisitNode:
			if visited[contract.NodeID] {
				gained = 1
			}
		case models.QuestObjectiveDefeatKind:
			gained = kills[contract.Kind]
		}
		if gained <= 0 {
			continue
		}
		progress.Count += gained
		if progress.Count > required {
			progress.Count = required
		}
		encoded, err := json.Marshal(progress)
		if err != nil {
			return fmt.Errorf("序列化合同进度: %w", err)
		}
		if err := tx.Model(&models.UserQuest{}).Where("id = ? AND user_id = ?", state.ID, userID).
			Update("progress_json", string(encoded)).Error; err != nil {
			return fmt.Errorf("更新合同进度: %w", err)
		}
		state.ProgressJSON = string(encoded)
		if progress.Count >= required {
			if err := completeQuestTx(tx, userID, def, state); err != nil {
				return err
			}
		}
	}
	return nil
}

func visitedNodesFromTrace(events []engine.TraceEvent) map[string]bool {
	visited := make(map[string]bool)
	for _, event := range events {
		if event.Type == engine.TraceNodeEntered && event.NodeID != "" {
			visited[event.NodeID] = true
		}
	}
	return visited
}

func defeatedKindsFromTrace(snapshot engine.ScenarioSnapshot, events []engine.TraceEvent) map[string]int {
	kills := make(map[string]int)
	for _, event := range events {
		if event.Type != engine.TraceBattleFinished {
			continue
		}
		enemy, ok := snapshot.Enemies[event.SubjectID]
		if !ok || enemy.Kind == "" {
			continue
		}
		enemyHP, _ := event.Payload["enemyHp"].(float64)
		if enemyHP > 0 {
			continue
		}
		kills[enemy.Kind]++
	}
	return kills
}
