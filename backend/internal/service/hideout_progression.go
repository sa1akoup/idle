package service

import (
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

var plannedHideoutFacilities = map[string]struct{}{
	"booze_generator": {},
	"bitcoin_farm":    {},
	"scav_case":       {},
	"shooting_range":  {},
	"gym":             {},
	"library":         {},
	"hall_of_fame":    {},
}

func hideoutFacilityRuntime(facilityID string) string {
	if _, planned := plannedHideoutFacilities[facilityID]; planned {
		return "planned"
	}
	return "active"
}

// hideoutProgressionBonusesTx 读取当前已生效的情报与身体技能成长加成，供场景快照冻结。
func hideoutProgressionBonusesTx(tx *gorm.DB, userID uint) (intelBonus int, physicalGrowth int, err error) {
	var states []models.HideoutFacility
	if err := tx.Where("user_id = ?", userID).Find(&states).Error; err != nil {
		return 0, 0, fmt.Errorf("读取藏身处设施: %w", err)
	}
	var levels []models.FacilityLevelDef
	if err := tx.Order("facility_id asc, level asc").Find(&levels).Error; err != nil {
		return 0, 0, fmt.Errorf("读取设施等级: %w", err)
	}
	levelsByFacility := make(map[string]map[int]models.FacilityLevelDef, len(levels))
	for _, level := range levels {
		if levelsByFacility[level.FacilityID] == nil {
			levelsByFacility[level.FacilityID] = make(map[int]models.FacilityLevelDef)
		}
		levelsByFacility[level.FacilityID][level.Level] = level
	}
	generator, err := generatorViewTx(tx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("读取发电机状态: %w", err)
	}
	bonuses := bonusesFromStates(states, levelsByFacility, generator.Enabled)
	return bonuses.IntelBonusPercent, bonuses.PhysicalSkillGrowthPercent, nil
}
