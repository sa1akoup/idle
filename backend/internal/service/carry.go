package service

// 携行容量服务：计算角色可携带的物品格数与负重，供角色/探索界面展示。
// 负重基础 = 50kg + (力量-50)*0.3，胸挂/背包可额外增加格数与负重。

import (
	"fmt"

	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// CarryCapacity 角色当前携行容量快照。
type CarryCapacity struct {
	BaseSlots   int     `json:"baseSlots"`   // 基础可携带格数
	BonusSlots  int     `json:"bonusSlots"`  // 胸挂/背包增加的格数
	SecureSlots int     `json:"secureSlots"` // 安全箱口袋格数
	TotalSlots  int     `json:"totalSlots"`  // 总可携带格数
	BaseWeight  float64 `json:"baseWeight"`  // 基础可携带负重 kg
	BonusWeight float64 `json:"bonusWeight"` // 胸挂/背包增加的负重 kg
	TotalWeight float64 `json:"totalWeight"` // 总可携带负重 kg
	UsedSlots   int     `json:"usedSlots"`   // 当前携行已占用格数
	UsedWeight  float64 `json:"usedWeight"`  // 当前携行已占用负重 kg
}

// GetCarryCapacityForUser 计算指定用户当前携行装备的容量与占用；展示侧按随身携带弹药槽位计入。
func GetCarryCapacityForUser(db *gorm.DB, userID uint) (*CarryCapacity, error) {
	loadout, err := GetPlayerLoadoutForUser(db, userID)
	if err != nil {
		return nil, err
	}
	return carryCapacityCore(db, userID, loadout, ammoCellsToStacks(loadout.CarriedAmmo))
}

// ammoCellsToStacks 把携带弹药槽位映射为引擎携带栈（容量计算只用到 ID 与发数）。
func ammoCellsToStacks(cells []models.AmmoCell) []engine.CarriedAmmo {
	stacks := make([]engine.CarriedAmmo, 0, len(cells))
	for _, cell := range cells {
		if cell.AmmoID == "" || cell.Rounds <= 0 {
			continue
		}
		stacks = append(stacks, engine.CarriedAmmo{ID: cell.AmmoID, Rounds: cell.Rounds})
	}
	return stacks
}

// carryCapacityCore 按弹药栈计算容量；开局状态传入实际预留弹药，角色页传入随身弹药槽位。
func carryCapacityCore(db *gorm.DB, userID uint, loadout *models.PlayerLoadout, ammoStacks []engine.CarriedAmmo) (*CarryCapacity, error) {
	var c models.Character
	if err := db.Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, fmt.Errorf("读取角色: %w", err)
	}

	baseWeight := models.BaseCarryWeight(c.Strength)
	baseSlots := models.BaseCarrySlots

	ids := []string{loadout.WeaponID, loadout.ArmorID,
		loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID}
	ids = append(ids, loadout.Consumables...)
	if loadout.SecureContainerID != "" {
		ids = append(ids, loadout.SecureContainerID)
	}
	catalogRepo := catalog.New(db)
	items, err := catalogRepo.FindByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("读取携行物品目录: %w", err)
	}

	bonusSlots, bonusWeight, secureSlots := 0, 0.0, 0
	usedSlots, usedWeight := 0, 0.0
	for _, id := range ids {
		if id == "" {
			continue
		}
		item, ok := items[id]
		if !ok {
			return nil, fmt.Errorf("读取携行物品 %s: %w", id, catalog.ErrItemNotFound)
		}
		usedWeight += float64(item.Weight)
		if item.Kind == "secure" {
			secureSlots += item.AddSlots
			continue
		}
		usedSlots += item.Slots
		// 只有胸挂与背包提供额外格数与负重加成。
		switch item.Kind {
		case "chestrig", "backpack":
			bonusSlots += item.AddSlots
			bonusWeight += float64(item.AddWeight)
		}
	}
	for _, stack := range ammoStacks {
		if stack.ID == "" || stack.Rounds <= 0 {
			continue
		}
		ammo, err := catalogRepo.FindByID(stack.ID)
		if err != nil {
			return nil, fmt.Errorf("读取携行弹药目录: %w", err)
		}
		if ammo.Kind != "ammo" || ammo.RoundsPerSlot <= 0 {
			return nil, fmt.Errorf("弹药 %s 的每格容量无效", stack.ID)
		}
		groups := (stack.Rounds + ammo.RoundsPerSlot - 1) / ammo.RoundsPerSlot
		usedSlots += groups
		usedWeight += float64(groups) * engine.DefaultTuning().AmmoDrop.WeightPerGroup
	}

	return &CarryCapacity{
		BaseSlots:   baseSlots,
		BonusSlots:  bonusSlots,
		SecureSlots: secureSlots,
		TotalSlots:  baseSlots + bonusSlots + secureSlots,
		BaseWeight:  baseWeight,
		BonusWeight: bonusWeight,
		TotalWeight: baseWeight + bonusWeight,
		UsedSlots:   usedSlots,
		UsedWeight:  usedWeight,
	}, nil
}
