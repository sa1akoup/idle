// 工作台制造配方定义：输入/输出与等级门槛，由种子数据维护。
package models

import "time"

type RecipeDef struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	Name           string    `json:"name"`
	FacilityID     string    `gorm:"index;not null" json:"facilityId"`
	RequiredLevel  int       `json:"requiredLevel"`
	InputsJSON     string    `json:"inputsJson"`
	OutputItemID   string    `gorm:"index;not null" json:"outputItemId"`
	OutputQuantity int       `json:"outputQuantity"`
	CraftSeconds   int64     `json:"craftSeconds"`
	SortOrder      int       `json:"sortOrder"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// RecipeInput 是配方输入的 JSON 载荷单元。
type RecipeInput struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}
