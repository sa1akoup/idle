package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrQuestUnavailable = errors.New("合同不可用")

type QuestView struct {
	ID             string `json:"id"`
	MerchantID     string `json:"merchantId"`
	MerchantName   string `json:"merchantName"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	ObjectiveType  string `json:"objectiveType"`
	TargetID       string `json:"targetId"`
	TargetLabel    string `json:"targetLabel"`
	Required       int    `json:"required"`
	Current        int    `json:"current"`
	CanAccept      bool   `json:"canAccept"`
	CanTurnIn      bool   `json:"canTurnIn"`
	RewardCash     int    `json:"rewardCash"`
	RewardRep      int    `json:"rewardRep"`
	PrerequisiteID string `json:"prerequisiteId"`
}

// ListQuestsForUser 返回全部合同及当前进度；上交类进度只统计 raidExtract 库存。
func ListQuestsForUser(db *gorm.DB, userID uint) ([]QuestView, error) {
	var defs []models.QuestDef
	if err := db.Order("sort_order asc, id asc").Find(&defs).Error; err != nil {
		return nil, fmt.Errorf("读取合同定义: %w", err)
	}
	var states []models.UserQuest
	if err := db.Where("user_id = ?", userID).Find(&states).Error; err != nil {
		return nil, fmt.Errorf("读取合同进度: %w", err)
	}
	stateByID := make(map[string]models.UserQuest, len(states))
	completed := make(map[string]bool, len(states))
	for _, state := range states {
		stateByID[state.QuestID] = state
		if state.Status == models.QuestStatusCompleted {
			completed[state.QuestID] = true
		}
	}
	var merchants []models.MerchantDef
	if err := db.Find(&merchants).Error; err != nil {
		return nil, fmt.Errorf("读取商人: %w", err)
	}
	merchantName := make(map[string]string, len(merchants))
	for _, merchant := range merchants {
		merchantName[merchant.ID] = merchant.Name
	}
	views := make([]QuestView, 0, len(defs))
	for _, def := range defs {
		view, err := buildQuestView(db, userID, def, stateByID[def.ID], completed, merchantName[def.MerchantID])
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func buildQuestView(db *gorm.DB, userID uint, def models.QuestDef, state models.UserQuest, completed map[string]bool, merchantName string) (QuestView, error) {
	objective, err := parseQuestObjective(def.ObjectiveJSON)
	if err != nil {
		return QuestView{}, fmt.Errorf("解析合同 %s: %w", def.ID, err)
	}
	reward, err := parseQuestReward(def.RewardJSON)
	if err != nil {
		return QuestView{}, fmt.Errorf("解析合同奖励 %s: %w", def.ID, err)
	}
	status := models.QuestStatusLocked
	switch {
	case state.Status == models.QuestStatusActive || state.Status == models.QuestStatusCompleted:
		status = state.Status
	case def.PrerequisiteID == "" || completed[def.PrerequisiteID]:
		status = models.QuestStatusAvailable
	}
	progress, _ := parseQuestProgress(state.ProgressJSON)
	current := progress.Count
	targetID, label := questTarget(def.ObjectiveType, objective)
	required := objective.Quantity
	if required <= 0 {
		required = 1
	}
	if def.ObjectiveType == models.QuestObjectiveExtractItem && objective.ItemID != "" {
		if item, findErr := catalog.New(db).FindByID(objective.ItemID); findErr == nil {
			label = item.Name
		}
		if status == models.QuestStatusCompleted {
			current = required
		} else {
			owned, err := ownedRaidExtractQuantityTx(db, userID, objective.ItemID)
			if err != nil {
				return QuestView{}, err
			}
			current = owned
		}
	}
	if def.ObjectiveType == models.QuestObjectiveVisitNode {
		if nodeName := nodeNameByID(db, objective.NodeID); nodeName != "" {
			label = nodeName
		}
	}
	return QuestView{
		ID: def.ID, MerchantID: def.MerchantID, MerchantName: merchantName,
		Name: def.Name, Description: def.Description, Status: status,
		ObjectiveType: def.ObjectiveType, TargetID: targetID, TargetLabel: label,
		Required: required, Current: current,
		CanAccept:  status == models.QuestStatusAvailable,
		CanTurnIn:  status == models.QuestStatusActive && def.ObjectiveType == models.QuestObjectiveExtractItem && current >= required,
		RewardCash: reward.Cash, RewardRep: reward.Reputation, PrerequisiteID: def.PrerequisiteID,
	}, nil
}

func questTarget(objType string, objective models.QuestObjective) (string, string) {
	switch objType {
	case models.QuestObjectiveExtractItem:
		return objective.ItemID, objective.ItemID
	case models.QuestObjectiveDefeatKind:
		return objective.Kind, kindLabel(objective.Kind)
	case models.QuestObjectiveVisitNode:
		return objective.NodeID, objective.NodeID
	case models.QuestObjectiveStyleExtract:
		return objective.Style, styleLabel(objective.Style)
	default:
		return "", "成功撤离"
	}
}

func kindLabel(kind string) string {
	switch kind {
	case "grunt":
		return "巡逻单位"
	case "guard":
		return "守卫"
	case "elite":
		return "精英"
	case "sniper":
		return "狙击手"
	default:
		return kind
	}
}

func styleLabel(style string) string {
	switch style {
	case engine.ActionStyleStealth:
		return "隐秘型"
	case engine.ActionStyleAggressive:
		return "激进型"
	case engine.ActionStyleGreedy:
		return "贪婪型"
	default:
		return "均衡型"
	}
}

func nodeNameByID(db *gorm.DB, nodeID string) string {
	var node models.NodeDef
	if err := db.Select("name").Where("id = ?", nodeID).First(&node).Error; err != nil {
		return ""
	}
	return node.Name
}

// AcceptQuestForUser 将可接合同标为进行中。
func AcceptQuestForUser(db *gorm.DB, userID uint, questID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		views, err := ListQuestsForUser(tx, userID)
		if err != nil {
			return err
		}
		var target QuestView
		for _, view := range views {
			if view.ID == questID {
				target = view
				break
			}
		}
		if !target.CanAccept {
			return fmt.Errorf("%w：无法接取该合同", ErrQuestUnavailable)
		}
		now := time.Now()
		var existing models.UserQuest
		err = tx.Where("user_id = ? AND quest_id = ?", userID, questID).First(&existing).Error
		if err == nil {
			return fmt.Errorf("%w：无法接取该合同", ErrQuestUnavailable)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("接取合同: %w", err)
		}
		state := models.UserQuest{UserID: userID, QuestID: questID, Status: models.QuestStatusActive, ProgressJSON: `{"count":0}`, AcceptedAt: &now}
		if err := tx.Create(&state).Error; err != nil {
			return fmt.Errorf("接取合同: %w", err)
		}
		return nil
	})
}

// TurnInQuestForUser 上交局内带出物品并完成合同。商店购买库存不会被扣除。
func TurnInQuestForUser(db *gorm.DB, userID uint, questID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		var def models.QuestDef
		if err := tx.Where("id = ?", questID).First(&def).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w：合同不存在", ErrQuestUnavailable)
			}
			return fmt.Errorf("读取合同: %w", err)
		}
		if def.ObjectiveType != models.QuestObjectiveExtractItem {
			return fmt.Errorf("%w：该合同不需要上交物品", ErrQuestUnavailable)
		}
		var state models.UserQuest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND quest_id = ? AND status = ?", userID, questID, models.QuestStatusActive).
			First(&state).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w：请先接取合同", ErrQuestUnavailable)
			}
			return fmt.Errorf("读取合同进度: %w", err)
		}
		objective, err := parseQuestObjective(def.ObjectiveJSON)
		if err != nil {
			return err
		}
		if objective.ItemID == "" || objective.Quantity <= 0 {
			return fmt.Errorf("合同 %s 上交目标无效", def.ID)
		}
		if err := consumeRaidExtractItemsTx(tx, userID, objective.ItemID, objective.Quantity); err != nil {
			if errors.Is(err, ErrRaidExtractShortage) {
				return fmt.Errorf("%w：需要从局内带出 %s x%d", ErrQuestUnavailable, objective.ItemID, objective.Quantity)
			}
			return err
		}
		return completeQuestTx(tx, userID, def, &state)
	})
}

func parseQuestObjective(raw string) (models.QuestObjective, error) {
	var objective models.QuestObjective
	if raw == "" {
		return objective, nil
	}
	if err := json.Unmarshal([]byte(raw), &objective); err != nil {
		return models.QuestObjective{}, err
	}
	return objective, nil
}

func parseQuestReward(raw string) (models.QuestReward, error) {
	var reward models.QuestReward
	if raw == "" {
		return reward, nil
	}
	if err := json.Unmarshal([]byte(raw), &reward); err != nil {
		return models.QuestReward{}, err
	}
	return reward, nil
}

func parseQuestProgress(raw string) (models.QuestProgress, error) {
	var progress models.QuestProgress
	if raw == "" {
		return progress, nil
	}
	if err := json.Unmarshal([]byte(raw), &progress); err != nil {
		return models.QuestProgress{}, err
	}
	return progress, nil
}
