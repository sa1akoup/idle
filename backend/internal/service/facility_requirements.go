// 设施升级条件：统一计算物品、设施、商人好感度和角色技能条件。
package service

import (
	"errors"
	"fmt"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
		owned, err := ownedItemQuantityTx(db, userID, requirement.ReferenceID)
		if err != nil {
			return err
		}
		current := float64(owned)
		required := float64(quantity)
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

func currentRequirementValueTx(db *gorm.DB, userID uint, character models.Character, requirement models.FacilityRequirement) (float64, error) {
	switch requirement.RequirementType {
	case "item":
		owned, err := ownedItemQuantityTx(db, userID, requirement.ReferenceID)
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

func ownedItemQuantityTx(db *gorm.DB, userID uint, itemID string) (int, error) {
	var inventoryQuantity int
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&inventoryQuantity).Error; err != nil {
		return 0, fmt.Errorf("统计库存材料 %s: %w", itemID, err)
	}
	var instanceQuantity int64
	if err := db.Model(&models.ItemInstance{}).
		Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, itemID, "inventory", "normal").
		Count(&instanceQuantity).Error; err != nil {
		return 0, fmt.Errorf("统计材料实例 %s: %w", itemID, err)
	}
	return inventoryQuantity + int(instanceQuantity), nil
}

func consumeRequirementItemTx(tx *gorm.DB, userID uint, itemID string, quantity int) error {
	if quantity <= 0 {
		return nil
	}
	var inventoryQuantity int
	if err := tx.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&inventoryQuantity).Error; err != nil {
		return fmt.Errorf("统计材料 %s: %w", itemID, err)
	}
	fromInventory := quantity
	if fromInventory > inventoryQuantity {
		fromInventory = inventoryQuantity
	}
	if fromInventory > 0 {
		if err := removeInventoryItem(tx, userID, itemID, fromInventory); err != nil {
			return err
		}
		quantity -= fromInventory
	}
	if quantity <= 0 {
		return nil
	}
	var instances []models.ItemInstance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, itemID, "inventory", "normal").
		Order("current_durability asc, id asc").Limit(quantity).Find(&instances).Error; err != nil {
		return fmt.Errorf("读取材料实例 %s: %w", itemID, err)
	}
	if len(instances) < quantity {
		return fmt.Errorf("材料 %s 数量不足", itemID)
	}
	for _, instance := range instances {
		if err := tx.Delete(&instance).Error; err != nil {
			return fmt.Errorf("消耗材料实例 %d: %w", instance.ID, err)
		}
	}
	return nil
}
