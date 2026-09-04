package models

import "time"

const (
	QuestStatusLocked    = "locked"
	QuestStatusAvailable = "available"
	QuestStatusActive    = "active"
	QuestStatusCompleted = "completed"

	QuestObjectiveExtractItem  = "extract_item"
	QuestObjectiveDefeatKind   = "defeat_kind"
	QuestObjectiveVisitNode    = "visit_node"
	QuestObjectiveStyleExtract = "style_extract"
	QuestObjectiveSurviveRuns  = "survive_runs"
)

// QuestDef 商人合同静态定义。
type QuestDef struct {
	ID             string `gorm:"primaryKey" json:"id"`
	MerchantID     string `gorm:"index;not null" json:"merchantId"`
	ChainIndex     int    `json:"chainIndex"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	PrerequisiteID string `json:"prerequisiteId"`
	ObjectiveType  string `json:"objectiveType"`
	ObjectiveJSON  string `json:"-"`
	RewardJSON     string `json:"-"`
	SortOrder      int    `json:"sortOrder"`
}

// UserQuest 玩家合同进度。未接取的合同可以没有行，由服务按前置关系推导 available/locked。
type UserQuest struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"uniqueIndex:idx_user_quest,priority:1;not null" json:"userId"`
	QuestID      string     `gorm:"uniqueIndex:idx_user_quest,priority:2;not null" json:"questId"`
	Status       string     `gorm:"index;not null" json:"status"`
	ProgressJSON string     `json:"-"`
	AcceptedAt   *time.Time `json:"acceptedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

type QuestObjective struct {
	ItemID   string `json:"itemId,omitempty"`
	NodeID   string `json:"nodeId,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Style    string `json:"style,omitempty"`
	Quantity int    `json:"quantity"`
}

type QuestReward struct {
	Cash       int    `json:"cash"`
	MerchantID string `json:"merchantId"`
	Reputation int    `json:"reputation"`
}

type QuestProgress struct {
	Count int `json:"count"`
}
