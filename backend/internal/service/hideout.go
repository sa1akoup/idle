// 藏身处服务：处理设施等级、升级作业、护甲维修队列和模块收益。
package service

import (
	"errors"
	"fmt"
	"math"
	"time"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	facilityJobTypeUpgrade        = "upgrade"
	facilityJobTypeRepair         = "repair"
	facilityJobTypeCraft          = "craft"
	facilityJobTypeTraining       = "training"
	facilityJobTypeScavCase       = "scav_case"
	facilityJobRunning            = "running"
	facilityJobCompleted          = "completed"
	facilityJobCompletedUnclaimed = "completed_unclaimed"
	facilityJobFailed             = "failed"

	workbenchFacilityID = "workbench"
)

type HideoutSnapshot struct {
	Facilities      []HideoutFacilityView `json:"facilities"`
	Jobs            []HideoutJobView      `json:"jobs"`
	Bonuses         HideoutBonuses        `json:"bonuses"`
	StorageCapacity StorageCapacity       `json:"storageCapacity"`
	RepairCost      int                   `json:"repairCost"`
	Generator       *GeneratorView        `json:"generator"`
}

type HideoutFacilityView struct {
	ID                              string              `json:"id"`
	Name                            string              `json:"name"`
	Category                        string              `json:"category"`
	Description                     string              `json:"description"`
	IconKey                         string              `json:"iconKey"`
	Level                           int                 `json:"level"`
	MaxLevel                        int                 `json:"maxLevel"`
	State                           string              `json:"state"`
	StorageBonus                    int                 `json:"storageBonus"`
	RecoverySpeedPercent            int                 `json:"recoverySpeedPercent"`
	RepairSpeedPercent              int                 `json:"repairSpeedPercent"`
	IntelBonusPercent               int                 `json:"intelBonusPercent"`
	HPRecoveryPerHour               float64             `json:"hpRecoveryPerHour"`
	EnergyRecoveryPerHour           float64             `json:"energyRecoveryPerHour"`
	HydrationRecoveryPerHour        float64             `json:"hydrationRecoveryPerHour"`
	RepairKitDiscountPercent        int                 `json:"repairKitDiscountPercent"`
	FuelConsumptionReductionPercent int                 `json:"fuelConsumptionReductionPercent"`
	PhysicalSkillGrowthPercent      int                 `json:"physicalSkillGrowthPercent"`
	StressRecoveryPerHour           float64             `json:"stressRecoveryPerHour"`
	FuelSlotCount                   int                 `json:"fuelSlotCount"`
	RequiresPower                   bool                `json:"requiresPower"`
	EffectsJSON                     string              `json:"effectsJson"`
	NextUpgrade                     *HideoutUpgradeView `json:"nextUpgrade"`
}

type HideoutRequirementView struct {
	RequirementType string  `json:"requirementType"`
	ReferenceID     string  `json:"referenceId"`
	Label           string  `json:"label"`
	Quantity        int     `json:"quantity"`
	RequiredValue   float64 `json:"requiredValue"`
	CurrentValue    float64 `json:"currentValue"`
	Satisfied       bool    `json:"satisfied"`
}

type HideoutUpgradeView struct {
	Level            int                      `json:"level"`
	OriginalCost     int                      `json:"originalCost"`
	OriginalCurrency string                   `json:"originalCurrency"`
	OriginalSeconds  int                      `json:"originalSeconds"`
	Cost             int                      `json:"cost"`
	DurationSec      int                      `json:"durationSec"`
	MaterialID       string                   `json:"materialId"`
	MaterialName     string                   `json:"materialName"`
	MaterialQuantity int                      `json:"materialQuantity"`
	EffectsJSON      string                   `json:"effectsJson"`
	Requirements     []HideoutRequirementView `json:"requirements"`
	CanStart         bool                     `json:"canStart"`
}

type HideoutJobView struct {
	ID              uint      `json:"id"`
	FacilityID      string    `json:"facilityId"`
	JobType         string    `json:"jobType"`
	TargetLevel     int       `json:"targetLevel"`
	TargetRef       string    `json:"targetRef"`
	ArmorInstanceID *uint     `json:"armorInstanceId"`
	StartedAt       time.Time `json:"startedAt"`
	CompleteAt      time.Time `json:"completeAt"`
	Status          string    `json:"status"`
}

type HideoutBonuses struct {
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
}

// GetHideoutForUser 返回设施状态、当前作业和已生效的模块收益。
func GetHideoutForUser(db *gorm.DB, userID uint) (*HideoutSnapshot, error) {
	if err := settleRecoveryForUser(db, userID); err != nil {
		return nil, err
	}
	if err := settleDueHideoutJobsForUser(db, userID); err != nil {
		return nil, err
	}
	if err := settleGeneratorForUser(db, userID); err != nil {
		return nil, err
	}
	var facilities []models.FacilityDef
	if err := db.Order("sort_order asc, id asc").Find(&facilities).Error; err != nil {
		return nil, fmt.Errorf("读取藏身处设施: %w", err)
	}
	var levels []models.FacilityLevelDef
	if err := db.Order("facility_id asc, level asc").Find(&levels).Error; err != nil {
		return nil, fmt.Errorf("读取设施等级: %w", err)
	}
	var states []models.HideoutFacility
	if err := db.Where("user_id = ?", userID).Find(&states).Error; err != nil {
		return nil, fmt.Errorf("读取玩家设施状态: %w", err)
	}
	var character models.Character
	if err := db.Where("user_id = ?", userID).First(&character).Error; err != nil {
		return nil, fmt.Errorf("读取角色技能: %w", err)
	}
	var jobs []models.FacilityJob
	if err := db.Where("user_id = ? AND status = ?", userID, facilityJobRunning).
		Order("complete_at asc, id asc").Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("读取藏身处作业: %w", err)
	}
	busyFacilities := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		busyFacilities[job.FacilityID] = true
	}

	stateByFacility := make(map[string]models.HideoutFacility, len(states))
	for _, state := range states {
		stateByFacility[state.FacilityID] = state
	}
	levelsByFacility := make(map[string]map[int]models.FacilityLevelDef, len(levels))
	for _, level := range levels {
		if levelsByFacility[level.FacilityID] == nil {
			levelsByFacility[level.FacilityID] = make(map[int]models.FacilityLevelDef)
		}
		levelsByFacility[level.FacilityID][level.Level] = level
	}
	cash, err := cashQuantity(db, userID)
	if err != nil {
		return nil, err
	}
	generator, err := generatorViewTx(db, userID)
	if err != nil {
		return nil, fmt.Errorf("读取发电机状态: %w", err)
	}
	bonuses := bonusesFromStates(states, levelsByFacility, generator.Enabled)
	used, err := inventoryUsage(db, userID)
	if err != nil {
		return nil, fmt.Errorf("读取仓库占用: %w", err)
	}
	views := make([]HideoutFacilityView, 0, len(facilities))
	for _, facility := range facilities {
		state := stateByFacility[facility.ID]
		if state.State == "" {
			state.State = "ready"
		}
		current := levelsByFacility[facility.ID][state.Level]
		view := HideoutFacilityView{
			ID: facility.ID, Name: facility.Name, Category: facility.Category, Description: facility.Description,
			IconKey: facility.IconKey, Level: state.Level, MaxLevel: facility.MaxLevel, State: state.State,
			StorageBonus: current.StorageBonus, RecoverySpeedPercent: current.RecoverySpeedPercent,
			RepairSpeedPercent: current.RepairSpeedPercent, IntelBonusPercent: current.IntelBonusPercent,
			HPRecoveryPerHour: current.HPRecoveryPerHour, EnergyRecoveryPerHour: current.EnergyRecoveryPerHour,
			HydrationRecoveryPerHour: current.HydrationRecoveryPerHour, RepairKitDiscountPercent: current.RepairKitDiscountPercent,
			FuelConsumptionReductionPercent: current.FuelConsumptionReductionPercent, PhysicalSkillGrowthPercent: current.PhysicalSkillGrowthPercent,
			StressRecoveryPerHour: current.StressRecoveryPerHour, FuelSlotCount: current.FuelSlotCount, RequiresPower: current.RequiresPower,
			EffectsJSON: current.EffectsJSON,
		}
		if next, ok := levelsByFacility[facility.ID][state.Level+1]; ok {
			requirements, requirementsSatisfied, err := facilityRequirementViewsTx(db, userID, character, facility.ID, next.Level)
			if err != nil {
				return nil, err
			}
			view.NextUpgrade = &HideoutUpgradeView{
				Level: next.Level, OriginalCost: next.OriginalCost, OriginalCurrency: next.OriginalCurrency, OriginalSeconds: next.OriginalSeconds,
				Cost: next.UpgradeCost, DurationSec: next.UpgradeSeconds,
				MaterialID: next.MaterialID, MaterialName: next.MaterialName, MaterialQuantity: next.MaterialQuantity,
				EffectsJSON:  next.EffectsJSON,
				Requirements: requirements,
				CanStart:     state.State == "ready" && !busyFacilities[facility.ID] && cash >= next.UpgradeCost && requirementsSatisfied,
			}
		}
		views = append(views, view)
	}
	jobViews := make([]HideoutJobView, 0, len(jobs))
	for _, job := range jobs {
		jobViews = append(jobViews, HideoutJobView{
			ID: job.ID, FacilityID: job.FacilityID, JobType: job.JobType, TargetLevel: job.TargetLevel,
			TargetRef: job.TargetRef, ArmorInstanceID: job.ArmorInstanceID, StartedAt: job.StartedAt, CompleteAt: job.CompleteAt, Status: job.Status,
		})
	}
	workbenchLevel := stateByFacility[workbenchFacilityID].Level
	return &HideoutSnapshot{
		Facilities: views, Jobs: jobViews, Bonuses: bonuses,
		StorageCapacity: StorageCapacity{Capacity: models.InventoryCapacity + bonuses.StorageBonus, Used: used},
		RepairCost:      repairCostForLevel(workbenchLevel),
		Generator:       generator,
	}, nil
}

// StartFacilityUpgradeForUser 创建指定设施的升级作业并扣除升级资源。
func StartFacilityUpgradeForUser(db *gorm.DB, userID uint, facilityID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
			return err
		}
		var facility models.FacilityDef
		if err := tx.Where("id = ?", facilityID).First(&facility).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("设施不存在")
			}
			return fmt.Errorf("读取设施定义: %w", err)
		}
		var state models.HideoutFacility
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND facility_id = ?", userID, facilityID).First(&state).Error; err != nil {
			return fmt.Errorf("读取设施状态: %w", err)
		}
		if state.State != "ready" {
			return fmt.Errorf("%s 正在升级中", facility.Name)
		}
		if state.Level >= facility.MaxLevel {
			return fmt.Errorf("%s 已达到最高等级", facility.Name)
		}
		var activeJobs int64
		if err := tx.Model(&models.FacilityJob{}).
			Where("user_id = ? AND facility_id = ? AND status = ?", userID, facilityID, facilityJobRunning).
			Count(&activeJobs).Error; err != nil {
			return fmt.Errorf("检查设施作业: %w", err)
		}
		if activeJobs > 0 {
			return fmt.Errorf("%s 已有正在执行的作业", facility.Name)
		}
		var next models.FacilityLevelDef
		if err := tx.Where("facility_id = ? AND level = ?", facilityID, state.Level+1).First(&next).Error; err != nil {
			return fmt.Errorf("设施升级配置不存在")
		}
		var character models.Character
		if err := tx.Where("user_id = ?", userID).First(&character).Error; err != nil {
			return fmt.Errorf("读取升级技能条件: %w", err)
		}
		requirements, requirementsSatisfied, err := facilityRequirementViewsTx(tx, userID, character, facilityID, next.Level)
		if err != nil {
			return err
		}
		if !requirementsSatisfied {
			for _, requirement := range requirements {
				if !requirement.Satisfied {
					return fmt.Errorf("升级条件未满足：%s", requirement.Label)
				}
			}
		}
		if err := deductCash(tx, userID, next.UpgradeCost); err != nil {
			return err
		}
		if err := consumeFacilityUpgradeMaterialsTx(tx, userID, next); err != nil {
			return err
		}
		now := time.Now()
		completeAt := now.Add(time.Duration(next.UpgradeSeconds) * time.Second)
		state.State = "upgrading"
		if err := tx.Model(&models.HideoutFacility{}).Where("user_id = ? AND facility_id = ?", userID, facilityID).Updates(map[string]interface{}{
			"state": "upgrading", "updated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("保存设施升级状态: %w", err)
		}
		job := models.FacilityJob{
			UserID: userID, FacilityID: facilityID, JobType: facilityJobTypeUpgrade,
			TargetLevel: next.Level, StartedAt: now, CompleteAt: completeAt, Status: facilityJobRunning,
		}
		if err := tx.Create(&job).Error; err != nil {
			return fmt.Errorf("创建设施升级作业: %w", err)
		}
		return nil
	})
}

// consumeFacilityUpgradeMaterialsTx 在条件校验通过后一次性扣除本次升级的全部物品材料。
func consumeFacilityUpgradeMaterialsTx(tx *gorm.DB, userID uint, level models.FacilityLevelDef) error {
	var requirements []models.FacilityRequirement
	if err := tx.Where("facility_id = ? AND level = ?", level.FacilityID, level.Level).
		Find(&requirements).Error; err != nil {
		return fmt.Errorf("读取升级材料条件: %w", err)
	}
	quantities := make(map[string]int)
	explicitItems := make(map[string]struct{})
	order := make([]string, 0, len(requirements)+1)
	for _, requirement := range requirements {
		if requirement.RequirementType != "item" || requirement.Quantity <= 0 {
			continue
		}
		explicitItems[requirement.ReferenceID] = struct{}{}
		if _, exists := quantities[requirement.ReferenceID]; !exists {
			order = append(order, requirement.ReferenceID)
		}
		quantities[requirement.ReferenceID] += requirement.Quantity
	}
	if level.MaterialID != "" && level.MaterialQuantity > 0 {
		if _, exists := explicitItems[level.MaterialID]; !exists {
			if _, exists := quantities[level.MaterialID]; !exists {
				order = append(order, level.MaterialID)
			}
			quantities[level.MaterialID] += level.MaterialQuantity
		}
	}
	for _, itemID := range order {
		if err := consumeRequirementItemTx(tx, userID, itemID, quantities[itemID]); err != nil {
			return fmt.Errorf("扣除升级材料 %s: %w", itemID, err)
		}
	}
	return nil
}

// QueueArmorRepairForUser 将损坏护甲加入维修台作业队列。
func QueueArmorRepairForUser(db *gorm.DB, userID, armorInstanceID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
			return err
		}
		var workbench models.HideoutFacility
		if err := tx.Where("user_id = ? AND facility_id = ?", userID, workbenchFacilityID).First(&workbench).Error; err != nil {
			return fmt.Errorf("维修台不可用")
		}
		if workbench.State != "ready" {
			return fmt.Errorf("维修台正在升级中")
		}
		var workbenchLevel models.FacilityLevelDef
		if err := tx.Where("facility_id = ? AND level = ?", workbenchFacilityID, workbench.Level).First(&workbenchLevel).Error; err != nil {
			return fmt.Errorf("读取维修台等级收益: %w", err)
		}
		var activeJobs int64
		if err := tx.Model(&models.FacilityJob{}).
			Where("user_id = ? AND facility_id = ? AND status = ?", userID, workbenchFacilityID, facilityJobRunning).
			Count(&activeJobs).Error; err != nil {
			return fmt.Errorf("检查维修队列: %w", err)
		}
		if activeJobs > 0 {
			return fmt.Errorf("维修台已有作业，请等待当前作业完成")
		}
		var armor models.ArmorInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND id = ?", userID, armorInstanceID).First(&armor).Error; err != nil {
			return err
		}
		if err := ensureArmorRepairAllowed(tx, userID, armor.ArmorID); err != nil {
			return err
		}
		if armor.RepairCount >= 1 {
			return fmt.Errorf("已达维修上限，护甲将报废")
		}
		if armor.CurDurability > 0 {
			return fmt.Errorf("仅归零护甲可维修")
		}
		if armor.Status == "repairing" {
			return fmt.Errorf("护甲已经在维修队列中")
		}
		newMaxDurability := armor.MaxDurability / 2
		baseRepairValue := float64(newMaxDurability) - float64(armor.CurDurability)
		if baseRepairValue < 0 {
			baseRepairValue = 0
		}
		repairValue := math.Ceil(baseRepairValue * float64(100-workbenchLevel.RepairKitDiscountPercent) / 100)
		if err := consumeRepairKitsTx(tx, userID, repairValue); err != nil {
			return err
		}
		if err := deductCash(tx, userID, repairCostForLevel(workbench.Level)); err != nil {
			return err
		}
		var armorID = armor.ID
		now := time.Now()
		duration := repairDurationForSpeed(workbenchLevel.RepairSpeedPercent)
		if err := tx.Model(&models.ArmorInstance{}).Where("user_id = ? AND id = ?", userID, armor.ID).Update("status", "repairing").Error; err != nil {
			return fmt.Errorf("标记护甲维修状态: %w", err)
		}
		job := models.FacilityJob{
			UserID: userID, FacilityID: workbenchFacilityID, JobType: facilityJobTypeRepair,
			ArmorInstanceID: &armorID, StartedAt: now, CompleteAt: now.Add(duration), Status: facilityJobRunning,
		}
		if err := tx.Create(&job).Error; err != nil {
			return fmt.Errorf("创建护甲维修作业: %w", err)
		}
		return nil
	})
}

// consumeRepairKitsTx 按当前耐久从低到高消耗武器维修包的可维修值。
func consumeRepairKitsTx(tx *gorm.DB, userID uint, requiredValue float64) error {
	if requiredValue <= 0 {
		return nil
	}
	var definition models.ItemUseDef
	if err := tx.Where("repair_value > 0 AND item_id = ?", "weapon_repair_kit_used").First(&definition).Error; err != nil {
		return fmt.Errorf("读取维修包定义: %w", err)
	}
	var instances []models.ItemInstance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, definition.ItemID, "inventory", "normal").
		Order("current_durability asc, id asc").Find(&instances).Error; err != nil {
		return fmt.Errorf("读取维修包库存: %w", err)
	}
	available := 0.0
	for _, instance := range instances {
		maxDurability := instance.MaxDurability
		if maxDurability <= 0 {
			maxDurability = definition.MaxDurability
		}
		if maxDurability <= 0 {
			maxDurability = 100
		}
		available += instance.CurrentDurability / maxDurability * definition.RepairValue
	}
	if available+0.000001 < requiredValue {
		return fmt.Errorf("维修包余量不足，需要 %.2f 点，当前仅有 %.2f 点", requiredValue, available)
	}
	remaining := requiredValue
	for _, instance := range instances {
		if remaining <= 0.000001 {
			break
		}
		maxDurability := instance.MaxDurability
		if maxDurability <= 0 {
			maxDurability = definition.MaxDurability
		}
		if maxDurability <= 0 {
			maxDurability = 100
		}
		availableValue := instance.CurrentDurability / maxDurability * definition.RepairValue
		consumeValue := math.Min(availableValue, remaining)
		instance.CurrentDurability -= consumeValue / definition.RepairValue * maxDurability
		if instance.CurrentDurability <= 0.000001 {
			instance.CurrentDurability = 0
			instance.Status = "depleted"
		}
		if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ?", userID, instance.ID).Updates(map[string]interface{}{
			"current_durability": instance.CurrentDurability, "status": instance.Status,
		}).Error; err != nil {
			return fmt.Errorf("保存维修包耐久 %d: %w", instance.ID, err)
		}
		remaining -= consumeValue
	}
	return nil
}

func settleDueHideoutJobsForUser(db *gorm.DB, userID uint) error {
	var count int64
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND status IN ? AND complete_at <= ?", userID, []string{facilityJobRunning, facilityJobCompletedUnclaimed}, time.Now()).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查到期藏身处作业: %w", err)
	}
	if count == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		return settleDueHideoutJobsTx(tx, userID, time.Now())
	})
}

func settleDueHideoutJobsTx(tx *gorm.DB, userID uint, now time.Time) error {
	var jobs []models.FacilityJob
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status IN ? AND complete_at <= ?", userID, []string{facilityJobRunning, facilityJobCompletedUnclaimed}, now).
		Order("complete_at asc, id asc").Find(&jobs).Error; err != nil {
		return fmt.Errorf("读取到期藏身处作业: %w", err)
	}
	for _, job := range jobs {
		switch job.JobType {
		case facilityJobTypeUpgrade:
			if err := tx.Model(&models.HideoutFacility{}).
				Where("user_id = ? AND facility_id = ?", userID, job.FacilityID).
				Updates(map[string]interface{}{"level": job.TargetLevel, "state": "ready", "updated_at": now}).Error; err != nil {
				return fmt.Errorf("完成设施升级: %w", err)
			}
		case facilityJobTypeRepair:
			if job.ArmorInstanceID == nil {
				return fmt.Errorf("维修作业缺少护甲实例")
			}
			var armor models.ArmorInstance
			if err := tx.Where("user_id = ? AND id = ?", userID, *job.ArmorInstanceID).First(&armor).Error; err != nil {
				return fmt.Errorf("读取待完成维修护甲: %w", err)
			}
			newMax := armor.MaxDurability / 2
			if err := tx.Model(&models.ArmorInstance{}).Where("user_id = ? AND id = ?", userID, armor.ID).Updates(map[string]interface{}{
				"max_durability": newMax, "cur_durability": newMax, "repair_count": armor.RepairCount + 1, "status": "normal",
			}).Error; err != nil {
				return fmt.Errorf("完成护甲维修: %w", err)
			}
		case facilityJobTypeCraft:
			if job.TargetRef == "" {
				return fmt.Errorf("制造作业缺少配方引用")
			}
			var recipe models.RecipeDef
			if err := tx.Where("id = ?", job.TargetRef).First(&recipe).Error; err != nil {
				return fmt.Errorf("读取制造配方 %s: %w", job.TargetRef, err)
			}
			output, err := catalog.New(tx).FindByID(recipe.OutputItemID)
			if err != nil {
				return err
			}
			outputSlots, err := craftingOutputSlots(tx, userID, output, recipe.OutputQuantity)
			if err != nil {
				return err
			}
			used, err := inventoryUsage(tx, userID)
			if err != nil {
				return err
			}
			capacity, err := storageCapacityForUser(tx, userID)
			if err != nil {
				return err
			}
			if used+outputSlots > capacity {
				// 到期但仓库已被占满：作业保留为未领取，等待下一次结算时空间释放再交付。
				if err := tx.Model(&models.FacilityJob{}).Where("id = ? AND user_id = ?", job.ID, userID).Update("status", facilityJobCompletedUnclaimed).Error; err != nil {
					return fmt.Errorf("标记制造作业待交付: %w", err)
				}
				continue
			}
			// 产物走购买同款入库路径：耐久成品自动建满耐久实例。
			if err := addInventoryItem(tx, userID, output, recipe.OutputQuantity, false); err != nil {
				return fmt.Errorf("发放制造产物 %s: %w", recipe.OutputItemID, err)
			}
		default:
			return fmt.Errorf("未知藏身处作业类型 %s", job.JobType)
		}
		if err := tx.Model(&models.FacilityJob{}).Where("id = ? AND user_id = ?", job.ID, userID).Update("status", facilityJobCompleted).Error; err != nil {
			return fmt.Errorf("更新藏身处作业状态: %w", err)
		}
	}
	return nil
}

func bonusesFromStates(states []models.HideoutFacility, levels map[string]map[int]models.FacilityLevelDef, generatorEnabled bool) HideoutBonuses {
	var bonuses HideoutBonuses
	for _, state := range states {
		level := levels[state.FacilityID][state.Level]
		if level.RequiresPower && !generatorEnabled {
			continue
		}
		bonuses.StorageBonus += level.StorageBonus
		bonuses.RecoverySpeedPercent += level.RecoverySpeedPercent
		bonuses.RepairSpeedPercent += level.RepairSpeedPercent
		bonuses.IntelBonusPercent += level.IntelBonusPercent
		bonuses.HPRecoveryPerHour += level.HPRecoveryPerHour
		bonuses.EnergyRecoveryPerHour += level.EnergyRecoveryPerHour
		bonuses.HydrationRecoveryPerHour += level.HydrationRecoveryPerHour
		bonuses.RepairKitDiscountPercent += level.RepairKitDiscountPercent
		bonuses.FuelConsumptionReductionPercent += level.FuelConsumptionReductionPercent
		bonuses.PhysicalSkillGrowthPercent += level.PhysicalSkillGrowthPercent
		bonuses.StressRecoveryPerHour += level.StressRecoveryPerHour
	}
	return bonuses
}

func storageCapacityForUser(db *gorm.DB, userID uint) (int, error) {
	var states []models.HideoutFacility
	if err := db.Where("user_id = ?", userID).Find(&states).Error; err != nil {
		return 0, fmt.Errorf("读取储物间等级: %w", err)
	}
	var levels []models.FacilityLevelDef
	if err := db.Where("facility_id = ?", "storage").Find(&levels).Error; err != nil {
		return 0, fmt.Errorf("读取储物间收益: %w", err)
	}
	levelMap := make(map[int]models.FacilityLevelDef, len(levels))
	for _, level := range levels {
		levelMap[level.Level] = level
	}
	bonus := 0
	for _, state := range states {
		if state.FacilityID == "storage" {
			bonus = levelMap[state.Level].StorageBonus
			break
		}
	}
	return models.InventoryCapacity + bonus, nil
}

func repairCostForLevel(level int) int {
	switch level {
	case 3:
		return 90
	case 2:
		return 130
	default:
		return 180
	}
}

func repairDurationForSpeed(speedPercent int) time.Duration {
	duration := 30 * time.Second * time.Duration(100-speedPercent) / 100
	if duration < 5*time.Second {
		return 5 * time.Second
	}
	return duration
}

func cashQuantity(db *gorm.DB, userID uint) (int, error) {
	var cash int
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", userID, "cash").
		Select("COALESCE(SUM(quantity), 0)").Scan(&cash).Error; err != nil {
		return 0, fmt.Errorf("读取现金: %w", err)
	}
	return cash, nil
}
