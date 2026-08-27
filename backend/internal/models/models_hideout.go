// 藏身处模型：保存设施定义、等级收益、玩家设施状态和正在执行的队列作业。
package models

import "time"

// FacilityDef 藏身处设施静态定义。
type FacilityDef struct {
	ID          string `gorm:"primaryKey" json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IconKey     string `json:"iconKey"`
	MaxLevel    int    `json:"maxLevel"`
	SortOrder   int    `json:"sortOrder"`
}

// FacilityLevelDef 描述某个设施等级的升级消耗和功能收益。
type FacilityLevelDef struct {
	ID                              uint    `gorm:"primaryKey" json:"id"`
	FacilityID                      string  `gorm:"index;not null" json:"facilityId"`
	Level                           int     `gorm:"not null" json:"level"`
	OriginalCost                    int     `json:"originalCost"`
	OriginalCurrency                string  `json:"originalCurrency"`
	OriginalSeconds                 int     `json:"originalSeconds"`
	UpgradeCost                     int     `json:"upgradeCost"`
	UpgradeSeconds                  int     `json:"upgradeSeconds"`
	MaterialID                      string  `json:"materialId"`
	MaterialName                    string  `json:"materialName"`
	MaterialQuantity                int     `json:"materialQuantity"`
	EffectsJSON                     string  `json:"effectsJson"`
	StorageBonus                    int     `json:"storageBonus"`
	RecoverySpeedPercent            int     `json:"recoverySpeedPercent"`
	RepairSpeedPercent              int     `json:"repairSpeedPercent"`
	IntelBonusPercent               int     `json:"intelBonusPercent"`
	HPRecoveryPerHour               float64 `json:"hpRecoveryPerHour"`
	EnergyRecoveryPerHour           float64 `json:"energyRecoveryPerHour"`
	HydrationRecoveryPerHour        float64 `json:"hydrationRecoveryPerHour"`
	RepairKitDiscountPercent        int     `json:"repairKitDiscountPercent"`
	FuelConsumptionReductionPercent int     `json:"fuelConsumptionReductionPercent"`
	PhysicalSkillGrowthPercent      int     `json:"physicalSkillGrowthPercent"`
	StressRecoveryPerHour           float64 `json:"stressRecoveryPerHour"`
	FuelSlotCount                   int     `json:"fuelSlotCount"`
	RequiresPower                   bool    `json:"requiresPower"`
}

// HideoutFacility 玩家拥有的设施状态。
type HideoutFacility struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	FacilityID string    `gorm:"uniqueIndex:idx_hideout_facility_user,priority:2;not null" json:"facilityId"`
	Level      int       `gorm:"not null;default:0" json:"level"`
	State      string    `gorm:"not null;default:ready" json:"state"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// FacilityJob 藏身处离散作业，按服务器时间完成；payload/result 用于生产、训练和搜寻任务。
type FacilityJob struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"userId"`
	FacilityID      string    `gorm:"index;not null" json:"facilityId"`
	JobType         string    `gorm:"not null" json:"jobType"`
	TargetLevel     int       `json:"targetLevel"`
	TargetRef       string    `json:"targetRef"`
	PayloadJSON     string    `json:"-"`
	ResultJSON      string    `json:"-"`
	ArmorInstanceID *uint     `gorm:"index" json:"armorInstanceId"`
	StartedAt       time.Time `json:"startedAt"`
	CompleteAt      time.Time `gorm:"index;not null" json:"completeAt"`
	Status          string    `gorm:"index;not null" json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
}

// FacilityRuntimeState 保存发电机等连续运行设施的时间游标。
type FacilityRuntimeState struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	FacilityID string    `gorm:"uniqueIndex:idx_facility_runtime_user,priority:2;not null" json:"facilityId"`
	Enabled    bool      `json:"enabled"`
	UpdatedAt  time.Time `json:"updatedAt"`
	StateJSON  string    `json:"-"`
}
