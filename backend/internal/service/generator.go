// 发电机运行状态：以时间戳懒结算燃料，不启动独立计时器。
package service

import (
	"encoding/json"
	"fmt"
	"time"

	"idle/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	generatorFacilityID  = "generator"
	generatorLocationRef = "generator"
)

type generatorRuntime struct {
	FuelInstanceIDs []uint `json:"fuelInstanceIds"`
}

type GeneratorFuelView struct {
	InstanceID        uint    `json:"instanceId"`
	ItemID            string  `json:"itemId"`
	CurrentDurability float64 `json:"currentDurability"`
	MaxDurability     float64 `json:"maxDurability"`
	FuelSeconds       int64   `json:"fuelSeconds"`
}

type GeneratorView struct {
	Enabled               bool                `json:"enabled"`
	FuelSlots             int                 `json:"fuelSlots"`
	FuelRemainingSeconds  int64               `json:"fuelRemainingSeconds"`
	FuelConsumptionFactor float64             `json:"fuelConsumptionFactor"`
	UpdatedAt             time.Time           `json:"updatedAt"`
	Fuels                 []GeneratorFuelView `json:"fuels"`
}

// settleGeneratorForUser 入口：加资源锁后按当前时间惰性结算发电机燃料（懒结算，无独立计时器）。
func settleGeneratorForUser(db *gorm.DB, userID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		return settleGeneratorTx(tx, userID, time.Now())
	})
}

// getGeneratorRuntimeTx 读取（不存在则创建）发电机运行状态并解析燃料实例 ID 列表，行锁防止并发结算。
func getGeneratorRuntimeTx(tx *gorm.DB, userID uint) (*models.FacilityRuntimeState, *generatorRuntime, error) {
	var runtime models.FacilityRuntimeState
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND facility_id = ?", userID, generatorFacilityID).First(&runtime).Error
	if err == gorm.ErrRecordNotFound {
		runtime = models.FacilityRuntimeState{UserID: userID, FacilityID: generatorFacilityID, UpdatedAt: time.Now()}
		if err := tx.Create(&runtime).Error; err != nil {
			return nil, nil, fmt.Errorf("创建发电机运行状态: %w", err)
		}
	} else if err != nil {
		return nil, nil, fmt.Errorf("读取发电机运行状态: %w", err)
	}
	state := generatorRuntime{}
	if runtime.StateJSON != "" {
		if err := json.Unmarshal([]byte(runtime.StateJSON), &state); err != nil {
			return nil, nil, fmt.Errorf("解析发电机燃料状态: %w", err)
		}
	}
	return &runtime, &state, nil
}

// generatorPowerEnabledTx 只读查询发电机当前是否开启（无记录视为关闭）。
func generatorPowerEnabledTx(tx *gorm.DB, userID uint) (bool, error) {
	var runtime models.FacilityRuntimeState
	if err := tx.Where("user_id = ? AND facility_id = ?", userID, generatorFacilityID).First(&runtime).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return runtime.Enabled, nil
}

// saveGeneratorRuntimeTx 写回发电机开关、结算时间与燃料实例 JSON 状态。
func saveGeneratorRuntimeTx(tx *gorm.DB, runtime *models.FacilityRuntimeState, state generatorRuntime, now time.Time) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化发电机燃料状态: %w", err)
	}
	return tx.Model(&models.FacilityRuntimeState{}).Where("id = ? AND user_id = ?", runtime.ID, runtime.UserID).
		Updates(map[string]interface{}{"enabled": runtime.Enabled, "updated_at": now, "state_json": string(encoded)}).Error
}

// settleGeneratorTx 惰性结算：按距上次结算的耗时扣减燃料，燃料耗尽时自动停机。
func settleGeneratorTx(tx *gorm.DB, userID uint, now time.Time) error {
	runtime, state, err := getGeneratorRuntimeTx(tx, userID)
	if err != nil {
		return err
	}
	if err := pruneGeneratorFuelTx(tx, userID, state); err != nil {
		return err
	}
	// 未开启时仅刷新结算时间戳，不消耗燃料。
	if !runtime.Enabled {
		runtime.UpdatedAt = now
		return saveGeneratorRuntimeTx(tx, runtime, *state, now)
	}
	if len(state.FuelInstanceIDs) == 0 {
		runtime.Enabled = false
		runtime.UpdatedAt = now
		return saveGeneratorRuntimeTx(tx, runtime, *state, now)
	}
	elapsed := now.Sub(runtime.UpdatedAt).Seconds()
	if elapsed <= 0 {
		runtime.UpdatedAt = now
		return saveGeneratorRuntimeTx(tx, runtime, *state, now)
	}
	reduction, err := solarPanelFuelReductionTx(tx, userID)
	if err != nil {
		return err
	}
	requiredSeconds := elapsed * (1 - reduction/100)
	remaining, err := consumeGeneratorFuelTx(tx, userID, state, requiredSeconds)
	if err != nil {
		return err
	}
	// remaining 表示本次结算后仍未满足的消耗秒数；为 0 说明燃料足够。
	runtime.Enabled = remaining <= 0.000001 && len(state.FuelInstanceIDs) > 0
	runtime.UpdatedAt = now
	return saveGeneratorRuntimeTx(tx, runtime, *state, now)
}

// pruneGeneratorFuelTx 清理燃料实例列表中的无效项（重复记录、实例已丢失、非燃料、已耗尽），保证装载列表与实物一致。
func pruneGeneratorFuelTx(tx *gorm.DB, userID uint, state *generatorRuntime) error {
	kept := make([]uint, 0, len(state.FuelInstanceIDs))
	seen := make(map[uint]struct{}, len(state.FuelInstanceIDs))
	for _, instanceID := range state.FuelInstanceIDs {
		if _, exists := seen[instanceID]; exists {
			continue
		}
		seen[instanceID] = struct{}{}
		var instance models.ItemInstance
		if err := tx.Where("user_id = ? AND id = ? AND location_type = ?", userID, instanceID, generatorLocationRef).First(&instance).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return fmt.Errorf("读取发电机燃料实例 %d: %w", instanceID, err)
		}
		var def models.ItemUseDef
		if err := tx.Where("item_id = ?", instance.ItemID).First(&def).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Delete(&instance).Error; err != nil {
					return fmt.Errorf("清理无效燃料实例 %d: %w", instanceID, err)
				}
				continue
			}
			return fmt.Errorf("读取燃料定义 %s: %w", instance.ItemID, err)
		}
		if instance.Status != "normal" || def.FuelSeconds <= 0 || instance.MaxDurability <= 0 || instance.CurrentDurability <= 0 {
			if err := tx.Delete(&instance).Error; err != nil {
				return fmt.Errorf("清理无效燃料实例 %d: %w", instanceID, err)
			}
			continue
		}
		kept = append(kept, instanceID)
	}
	state.FuelInstanceIDs = kept
	return nil
}

// consumeGeneratorFuelTx 按需消耗燃料实例耐久，返回仍未满足的消耗秒数（>0 表示燃料不足）。
func consumeGeneratorFuelTx(tx *gorm.DB, userID uint, state *generatorRuntime, requiredSeconds float64) (float64, error) {
	if requiredSeconds <= 0 {
		return 0, nil
	}
	remaining := requiredSeconds
	kept := make([]uint, 0, len(state.FuelInstanceIDs))
	for index, instanceID := range state.FuelInstanceIDs {
		var instance models.ItemInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND id = ? AND location_type = ?", userID, instanceID, generatorLocationRef).First(&instance).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return 0, fmt.Errorf("读取发电机燃料实例 %d: %w", instanceID, err)
		}
		var def models.ItemUseDef
		if err := tx.Where("item_id = ?", instance.ItemID).First(&def).Error; err != nil {
			return 0, fmt.Errorf("读取燃料定义 %s: %w", instance.ItemID, err)
		}
		if def.FuelSeconds <= 0 || instance.MaxDurability <= 0 || instance.CurrentDurability <= 0 {
			if err := tx.Delete(&instance).Error; err != nil {
				return 0, fmt.Errorf("清理无效燃料实例 %d: %w", instance.ID, err)
			}
			continue
		}
		// 秒数↔耐久换算：剩余可烧秒数 = 耐久比例 × 燃料总秒数，消耗后再按同比例回扣耐久。
		available := instance.CurrentDurability / instance.MaxDurability * float64(def.FuelSeconds)
		consume := available
		if consume > remaining {
			consume = remaining
		}
		instance.CurrentDurability -= consume / float64(def.FuelSeconds) * instance.MaxDurability
		if instance.CurrentDurability <= 0.000001 {
			instance.CurrentDurability = 0
			if err := tx.Delete(&instance).Error; err != nil {
				return 0, fmt.Errorf("清理耗尽燃料实例 %d: %w", instance.ID, err)
			}
		} else {
			kept = append(kept, instance.ID)
		}
		if instance.CurrentDurability > 0 {
			if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ?", userID, instance.ID).Update("current_durability", instance.CurrentDurability).Error; err != nil {
				return 0, fmt.Errorf("保存燃料耐久 %d: %w", instance.ID, err)
			}
		}
		remaining -= consume
		if remaining <= 0.000001 {
			// 当前燃料已经满足本次结算，后续槽位仍保持装载顺序。
			kept = append(kept, state.FuelInstanceIDs[index+1:]...)
			break
		}
	}
	state.FuelInstanceIDs = kept
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// solarPanelFuelReductionTx 返回太阳能板就绪等级提供的燃料减耗百分比，无则按 0 处理。
func solarPanelFuelReductionTx(tx *gorm.DB, userID uint) (float64, error) {
	var state models.HideoutFacility
	if err := tx.Where("user_id = ? AND facility_id = ? AND state = ?", userID, "solar_panel", "ready").First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	var level models.FacilityLevelDef
	if err := tx.Where("facility_id = ? AND level = ?", "solar_panel", state.Level).First(&level).Error; err != nil {
		return 0, err
	}
	return float64(level.FuelConsumptionReductionPercent), nil
}

// generatorViewTx 组装发电机前端视图：开关状态、槽位数、燃料减耗系数、剩余可烧秒数与当前燃料列表。
func generatorViewTx(db *gorm.DB, userID uint) (*GeneratorView, error) {
	runtime, state, err := getGeneratorRuntimeTx(db, userID)
	if err != nil {
		return nil, err
	}
	var facility models.HideoutFacility
	if err := db.Where("user_id = ? AND facility_id = ?", userID, generatorFacilityID).First(&facility).Error; err != nil {
		return nil, err
	}
	var level models.FacilityLevelDef
	if err := db.Where("facility_id = ? AND level = ?", generatorFacilityID, facility.Level).First(&level).Error; err != nil {
		return nil, err
	}
	view := &GeneratorView{Enabled: runtime.Enabled, FuelSlots: level.FuelSlotCount, UpdatedAt: runtime.UpdatedAt, FuelConsumptionFactor: 1}
	reduction, err := solarPanelFuelReductionTx(db, userID)
	if err != nil {
		return nil, fmt.Errorf("读取太阳能板燃料减耗: %w", err)
	}
	view.FuelConsumptionFactor = 1 - reduction/100
	if len(state.FuelInstanceIDs) == 0 {
		view.Fuels = []GeneratorFuelView{}
		return view, nil
	}
	var instances []models.ItemInstance
	if err := db.Where("user_id = ? AND id IN ? AND location_type = ?", userID, state.FuelInstanceIDs, generatorLocationRef).Find(&instances).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint]models.ItemInstance, len(instances))
	for _, instance := range instances {
		byID[instance.ID] = instance
	}
	view.Fuels = make([]GeneratorFuelView, 0, len(state.FuelInstanceIDs))
	for _, instanceID := range state.FuelInstanceIDs {
		instance, ok := byID[instanceID]
		if !ok {
			continue
		}
		var def models.ItemUseDef
		if err := db.Where("item_id = ?", instance.ItemID).First(&def).Error; err != nil {
			return nil, err
		}
		view.Fuels = append(view.Fuels, GeneratorFuelView{InstanceID: instance.ID, ItemID: instance.ItemID, CurrentDurability: instance.CurrentDurability, MaxDurability: instance.MaxDurability, FuelSeconds: def.FuelSeconds})
		view.FuelRemainingSeconds += int64(instance.CurrentDurability / instance.MaxDurability * float64(def.FuelSeconds))
	}
	return view, nil
}

// ToggleGeneratorForUser 开关发电机：先惰性结算，再切换；开启需发电机就绪且已装载燃料。
func ToggleGeneratorForUser(db *gorm.DB, userID uint, enabled bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		now := time.Now()
		if err := settleGeneratorTx(tx, userID, now); err != nil {
			return err
		}
		runtime, state, err := getGeneratorRuntimeTx(tx, userID)
		if err != nil {
			return err
		}
		// 开启需满足：发电机设施就绪、已有等级且已装载燃料实例。
		if enabled {
			var facility models.HideoutFacility
			if err := tx.Where("user_id = ? AND facility_id = ? AND state = ?", userID, generatorFacilityID, "ready").First(&facility).Error; err != nil {
				return fmt.Errorf("发电机不可用")
			}
			if facility.Level <= 0 || len(state.FuelInstanceIDs) == 0 {
				return fmt.Errorf("发电机没有可用燃料")
			}
		}
		runtime.Enabled = enabled
		runtime.UpdatedAt = now
		return saveGeneratorRuntimeTx(tx, runtime, *state, now)
	})
}

// LoadGeneratorFuelForUser 把燃料实例装入发电机：校验槽位空闲、未重复装载与燃料类型，并改实例归属为发电机。
func LoadGeneratorFuelForUser(db *gorm.DB, userID, instanceID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		now := time.Now()
		if err := settleGeneratorTx(tx, userID, now); err != nil {
			return err
		}
		runtime, state, err := getGeneratorRuntimeTx(tx, userID)
		if err != nil {
			return err
		}
		var facility models.HideoutFacility
		if err := tx.Where("user_id = ? AND facility_id = ? AND state = ?", userID, generatorFacilityID, "ready").First(&facility).Error; err != nil || facility.Level <= 0 {
			return fmt.Errorf("发电机不可用")
		}
		var level models.FacilityLevelDef
		if err := tx.Where("facility_id = ? AND level = ?", generatorFacilityID, facility.Level).First(&level).Error; err != nil {
			return err
		}
		if len(state.FuelInstanceIDs) >= level.FuelSlotCount {
			return fmt.Errorf("发电机燃料槽已满")
		}
		if containsUint(state.FuelInstanceIDs, instanceID) {
			return fmt.Errorf("燃料已经装载")
		}
		var instance models.ItemInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND id = ? AND location_type = ? AND status = ?", userID, instanceID, "inventory", "normal").First(&instance).Error; err != nil {
			return fmt.Errorf("燃料实例不可用: %w", err)
		}
		var def models.ItemUseDef
		if err := tx.Where("item_id = ? AND fuel_seconds > 0", instance.ItemID).First(&def).Error; err != nil {
			return fmt.Errorf("物品不是可用燃料")
		}
		if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ?", userID, instanceID).Updates(map[string]interface{}{"location_type": generatorLocationRef, "location_ref": generatorFacilityID}).Error; err != nil {
			return err
		}
		state.FuelInstanceIDs = append(state.FuelInstanceIDs, instanceID)
		runtime.UpdatedAt = now
		return saveGeneratorRuntimeTx(tx, runtime, *state, now)
	})
}

// UnloadGeneratorFuelForUser 把燃料实例卸回仓库并从装载记录中移除。
func UnloadGeneratorFuelForUser(db *gorm.DB, userID, instanceID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		now := time.Now()
		if err := settleGeneratorTx(tx, userID, now); err != nil {
			return err
		}
		runtime, state, err := getGeneratorRuntimeTx(tx, userID)
		if err != nil {
			return err
		}
		index := -1
		for i, id := range state.FuelInstanceIDs {
			if id == instanceID {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("燃料未装载")
		}
		if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ? AND location_type = ?", userID, instanceID, generatorLocationRef).Updates(map[string]interface{}{"location_type": "inventory", "location_ref": "", "status": "normal"}).Error; err != nil {
			return err
		}
		state.FuelInstanceIDs = append(state.FuelInstanceIDs[:index], state.FuelInstanceIDs[index+1:]...)
		runtime.UpdatedAt = now
		return saveGeneratorRuntimeTx(tx, runtime, *state, now)
	})
}

// containsUint 判断切片是否包含目标值，用于燃料装载去重。
func containsUint(values []uint, target uint) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
