package service

// 携行容量服务：计算角色可携带的物品格数与负重，供角色/探索界面展示。
// 负重基础 = 50kg + (力量-50)*0.3，胸挂/背包可额外增加格数与负重。

import (
	"fmt"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// CarryCapacity 角色当前携行容量快照。
type CarryCapacity struct {
	BaseSlots   int     `json:"baseSlots"`   // 基础可携带格数
	BonusSlots  int     `json:"bonusSlots"`  // 胸挂/背包增加的格数
	TotalSlots  int     `json:"totalSlots"`  // 总可携带格数
	BaseWeight  float64 `json:"baseWeight"`  // 基础可携带负重 kg
	BonusWeight float64 `json:"bonusWeight"` // 胸挂/背包增加的负重 kg
	TotalWeight float64 `json:"totalWeight"` // 总可携带负重 kg
	UsedSlots   int     `json:"usedSlots"`   // 当前携行已占用格数
	UsedWeight  float64 `json:"usedWeight"`  // 当前携行已占用负重 kg
}

// GetCarryCapacityForUser 计算指定用户当前携行装备的容量与占用。
func GetCarryCapacityForUser(db *gorm.DB, userID uint) (*CarryCapacity, error) {
	var c models.Character
	if err := db.Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, fmt.Errorf("读取角色: %w", err)
	}
	loadout, err := GetPlayerLoadoutForUser(db, userID)
	if err != nil {
		return nil, err
	}

	baseWeight := models.BaseCarryWeight(c.Strength)
	baseSlots := models.BaseCarrySlots

	ids := []string{loadout.WeaponID, loadout.ArmorID,
		loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID}
	ids = append(ids, loadout.Consumables...)
	catalogRepo := catalog.New(db)
	items, err := catalogRepo.FindByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("读取携行物品目录: %w", err)
	}

	bonusSlots, bonusWeight := 0, 0.0
	usedSlots, usedWeight := 0, 0.0
	for _, id := range ids {
		if id == "" {
			continue
		}
		item, ok := items[id]
		if !ok {
			return nil, fmt.Errorf("读取携行物品 %s: %w", id, catalog.ErrItemNotFound)
		}
		usedSlots += item.Slots
		usedWeight += float64(item.Weight)
		switch item.Kind {
		case "chestrig", "backpack":
			bonusSlots += item.AddSlots
			bonusWeight += float64(item.AddWeight)
		}
	}

	return &CarryCapacity{
		BaseSlots:   baseSlots,
		BonusSlots:  bonusSlots,
		TotalSlots:  baseSlots + bonusSlots,
		BaseWeight:  baseWeight,
		BonusWeight: bonusWeight,
		TotalWeight: baseWeight + bonusWeight,
		UsedSlots:   usedSlots,
		UsedWeight:  usedWeight,
	}, nil
}
