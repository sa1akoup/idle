// 设施升级条件：统一计算物品、设施、商人好感度和角色技能条件。
package service

import (
	"errors"
	"fmt"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// facilityRequirementViewsTx 批量读取某设施下一等级的全部升级条件并换算为视图（材料、前置设施、商人好感、技能），返回是否全部满足。
func facilityRequirementViewsTx(db *gorm.DB, userID uint, character models.Character, facilityID string, level int) ([]HideoutRequirementView, bool, error) {
	var levelDef models.FacilityLevelDef
	if err := db.Where("facility_id = ? AND level = ?", facilityID, level).First(&levelDef).Error; err != nil {
		return nil, false, fmt.Errorf("读取设施等级配置: %w", err)
	}
	var requirements []models.FacilityRequirement
	if err := db.Where("facility_id = ? AND level = ?", facilityID, level).
		Order("sort_order asc, id asc").Find(&requirements).Error; err != nil {
		return nil, false, fmt.Errorf("读取设施升级条件: %w", err)
	}
	catalogRepo := catalog.New(db)
	itemIDs := make([]string, 0, len(requirements)+1)
	for _, requirement := range requirements {
		if requirement.RequirementType == "item" {
			itemIDs = append(itemIDs, requirement.ReferenceID)
		}
	}
	if levelDef.MaterialID != "" && levelDef.MaterialQuantity > 0 {
		itemIDs = append(itemIDs, levelDef.MaterialID)
	}
	catalogItems, catalogErr := catalogRepo.FindByIDs(itemIDs)
	if catalogErr != nil && !errors.Is(catalogErr, catalog.ErrItemNotFound) {
		return nil, false, fmt.Errorf("读取设施材料目录: %w", catalogErr)
	}
	views := make([]HideoutRequirementView, 0, len(requirements)+1)
	itemViewIndexes := make(map[string]int)
	appendItemView := func(requirement models.FacilityRequirement, quantity int) error {
		owned, err := ownedRaidExtractQuantityTx(db, userID, requirement.ReferenceID)
		if err != nil {
			return err
		}
		current := float64(owned)
		required := float64(quantity)
		// 同一材料多次出现时合并到同一行视图，避免升级面板重复展示。
		if index, exists := itemViewIndexes[requirement.ReferenceID]; exists {
			views[index].Quantity += quantity
			views[index].RequiredValue = float64(views[index].Quantity)
			views[index].CurrentValue = current
			views[index].Satisfied = current >= views[index].RequiredValue
			return nil
		}
		itemViewIndexes[requirement.ReferenceID] = len(views)
		views = append(views, HideoutRequirementView{
			RequirementType: requirement.RequirementType,
			ReferenceID:     requirement.ReferenceID,
			Label:           facilityRequirementLabel(db, catalogItems, requirement),
			Quantity:        quantity,
			RequiredValue:   required,
			CurrentValue:    current,
			Satisfied:       current >= required,
		})
		return nil
	}
	for _, requirement := range requirements {
		if requirement.RequirementType == "item" && requirement.Quantity > 0 {
			if err := appendItemView(requirement, requirement.Quantity); err != nil {
				return nil, false, err
			}
			continue
		}
		current, err := currentRequirementValueTx(db, userID, character, requirement)
		if err != nil {
			return nil, false, err
		}
		views = append(views, HideoutRequirementView{
			RequirementType: requirement.RequirementType,
			ReferenceID:     requirement.ReferenceID,
			Label:           facilityRequirementLabel(db, catalogItems, requirement),
			Quantity:        requirement.Quantity,
			RequiredValue:   requirement.RequiredValue,
			CurrentValue:    current,
			Satisfied:       current >= requirement.RequiredValue,
		})
	}
	if levelDef.MaterialID != "" && levelDef.MaterialQuantity > 0 {
		if _, exists := itemViewIndexes[levelDef.MaterialID]; !exists {
			if err := appendItemView(models.FacilityRequirement{RequirementType: "item", ReferenceID: levelDef.MaterialID}, levelDef.MaterialQuantity); err != nil {
				return nil, false, err
			}
		}
	}
	allSatisfied := true
	for _, view := range views {
		if !view.Satisfied {
			allSatisfied = false
			break
		}
	}
	return views, allSatisfied, nil
}

// currentRequirementValueTx 按条件类型读取当前值：物品库存量、前置设施等级、商人好感度或角色技能属性。
func currentRequirementValueTx(db *gorm.DB, userID uint, character models.Character, requirement models.FacilityRequirement) (float64, error) {
	// 按条件类型换算当前值：物品/设施/商人/技能四种情况。
	switch requirement.RequirementType {
	case "item":
		owned, err := ownedRaidExtractQuantityTx(db, userID, requirement.ReferenceID)
		if err != nil {
			return 0, err
		}
		return float64(owned), nil
	case "facility":
		var state models.HideoutFacility
		if err := db.Where("user_id = ? AND facility_id = ?", userID, requirement.ReferenceID).First(&state).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return 0, nil
			}
			return 0, fmt.Errorf("读取前置设施 %s: %w", requirement.ReferenceID, err)
		}
		return float64(state.Level), nil
	case "trader":
		var state models.UserMerchantState
		if err := db.Where("user_id = ? AND merchant_id = ?", userID, requirement.ReferenceID).First(&state).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return 0, nil
			}
			return 0, fmt.Errorf("读取商人条件 %s: %w", requirement.ReferenceID, err)
		}
		return float64(state.Reputation), nil
	case "skill":
		return float64(getAttrValue(&character, requirement.ReferenceID)), nil
	default:
		return 0, fmt.Errorf("未知设施升级条件类型 %s", requirement.RequirementType)
	}
}

// facilityRequirementLabel 为条件引用生成可读名称（物品/设施/商人取名称，技能拼后缀）；查不到时退回原始 ID。
func facilityRequirementLabel(db *gorm.DB, catalogItems map[string]catalog.Item, requirement models.FacilityRequirement) string {
	switch requirement.RequirementType {
	case "item":
		if item, ok := catalogItems[requirement.ReferenceID]; ok {
			return item.Name
		}
		return requirement.ReferenceID
	case "facility":
		var facility models.FacilityDef
		if err := db.Where("id = ?", requirement.ReferenceID).First(&facility).Error; err == nil {
			return facility.Name
		}
		return requirement.ReferenceID
	case "trader":
		var merchant models.MerchantDef
		if err := db.Where("id = ?", requirement.ReferenceID).First(&merchant).Error; err == nil {
			return merchant.Name
		}
		return requirement.ReferenceID
	case "skill":
		return requirement.ReferenceID + " 技能"
	default:
		return requirement.ReferenceID
	}
}

// consumeRequirementItemTx 扣除升级/制造材料：只消耗局内带出（raidExtract）库存与实例。
func consumeRequirementItemTx(tx *gorm.DB, userID uint, itemID string, quantity int) error {
	if err := consumeRaidExtractItemsTx(tx, userID, itemID, quantity); err != nil {
		return fmt.Errorf("需要从局内带出 %s: %w", itemID, err)
	}
	return nil
}
