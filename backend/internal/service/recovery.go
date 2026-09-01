// 自动恢复：Session 终局后统一消费库存物品，并按藏身处设施懒结算恢复时间。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecoveryChoice struct {
	TargetPercent  int    `json:"targetPercent"`
	PrimaryMethod  string `json:"primaryMethod"`
	FallbackMethod string `json:"fallbackMethod"`
}

type RecoveryPolicy struct {
	HP             RecoveryChoice `json:"hp"`
	Energy         RecoveryChoice `json:"energy"`
	Hydration      RecoveryChoice `json:"hydration"`
	MerchantEnable bool           `json:"merchantEnable"`
}

type RecoveryView struct {
	Plan  models.RecoveryPlan   `json:"plan"`
	Tasks []models.RecoveryTask `json:"tasks"`
}

const (
	recoveryMethodInventory = "inventory"
	recoveryMethodHideout   = "hideout"
	recoveryMethodMerchant  = "merchant"
	recoveryMethodNone      = "none"
)

// defaultRecoveryPolicyJSON 返回默认恢复策略的 JSON 字符串，供未提供策略的会话使用。
func defaultRecoveryPolicyJSON() string {
	encoded, _ := json.Marshal(defaultRecoveryPolicy())
	return string(encoded)
}

// defaultRecoveryPolicy 返回默认恢复策略：生命目标 100%，能量/饮水 80%，优先库存、备用藏身处。
func defaultRecoveryPolicy() RecoveryPolicy {
	return RecoveryPolicy{
		HP:             RecoveryChoice{TargetPercent: 100, PrimaryMethod: "inventory", FallbackMethod: "hideout"},
		Energy:         RecoveryChoice{TargetPercent: 80, PrimaryMethod: "inventory", FallbackMethod: "hideout"},
		Hydration:      RecoveryChoice{TargetPercent: 80, PrimaryMethod: "inventory", FallbackMethod: "hideout"},
		MerchantEnable: true,
	}
}

// decodeRecoveryPolicy 解析恢复策略 JSON 并校验各项目标与方式取值。
func decodeRecoveryPolicy(value string) (RecoveryPolicy, error) {
	var policy RecoveryPolicy
	if err := json.Unmarshal([]byte(value), &policy); err != nil {
		return RecoveryPolicy{}, fmt.Errorf("解析恢复策略 JSON: %w", err)
	}
	for resource, choice := range map[string]RecoveryChoice{"hp": policy.HP, "energy": policy.Energy, "hydration": policy.Hydration} {
		if err := validateRecoveryChoice(resource, choice); err != nil {
			return RecoveryPolicy{}, fmt.Errorf("恢复策略字段无效: %w", err)
		}
	}
	return policy, nil
}

// recoveryPolicyJSONForStart 规范化并序列化开局恢复策略；未提供时回退默认策略。
func recoveryPolicyJSONForStart(input *RecoveryPolicy) (string, error) {
	if input == nil {
		return defaultRecoveryPolicyJSON(), nil
	}
	defaults := defaultRecoveryPolicy()
	policy := *input
	policy.HP = normalizeRecoveryChoice(policy.HP, defaults.HP)
	policy.Energy = normalizeRecoveryChoice(policy.Energy, defaults.Energy)
	policy.Hydration = normalizeRecoveryChoice(policy.Hydration, defaults.Hydration)
	for resource, choice := range map[string]RecoveryChoice{"hp": policy.HP, "energy": policy.Energy, "hydration": policy.Hydration} {
		if err := validateRecoveryChoice(resource, choice); err != nil {
			return "", err
		}
	}
	encoded, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("序列化恢复策略: %w", err)
	}
	return string(encoded), nil
}

// normalizeRecoveryChoice 用默认值补齐未填写的目标与方式，并按首选方式推导合理备用方式。
func normalizeRecoveryChoice(choice, defaults RecoveryChoice) RecoveryChoice {
	if choice.TargetPercent <= 0 {
		choice.TargetPercent = defaults.TargetPercent
	}
	if choice.PrimaryMethod == "" {
		choice.PrimaryMethod = defaults.PrimaryMethod
	}
	if choice.FallbackMethod == "" {
		switch choice.PrimaryMethod {
		case recoveryMethodInventory:
			choice.FallbackMethod = defaults.FallbackMethod
		case recoveryMethodMerchant:
			choice.FallbackMethod = recoveryMethodHideout
		default:
			choice.FallbackMethod = recoveryMethodNone
		}
	}
	return choice
}

// validateRecoveryChoice 校验某资源的恢复目标（1-100%）与首选/备用方式是否合法。
func validateRecoveryChoice(resource string, choice RecoveryChoice) error {
	if choice.TargetPercent < 1 || choice.TargetPercent > 100 {
		return fmt.Errorf("%s 恢复目标需为 1-100%%", recoveryResourceName(resource))
	}
	if !validRecoveryMethod(choice.PrimaryMethod) {
		return fmt.Errorf("%s 恢复首选方式无效", recoveryResourceName(resource))
	}
	if !validRecoveryMethod(choice.FallbackMethod) {
		return fmt.Errorf("%s 恢复备用方式无效", recoveryResourceName(resource))
	}
	return nil
}

// validRecoveryMethod 判断恢复方式是否为四种合法取值之一。
func validRecoveryMethod(method string) bool {
	switch method {
	case recoveryMethodInventory, recoveryMethodHideout, recoveryMethodMerchant, recoveryMethodNone:
		return true
	default:
		return false
	}
}

// createRecoveryPlanTx 在终局事务内创建恢复计划：先按策略即时消耗物品/商人购买，再为剩余缺口按藏身处速率生成持续恢复任务。
func createRecoveryPlanTx(tx *gorm.DB, userID, sessionID uint, state engine.CharacterState, policyJSON string) error {
	var policy RecoveryPolicy
	if policyJSON == "" {
		policy = defaultRecoveryPolicy()
		policyJSON = defaultRecoveryPolicyJSON()
	} else {
		var err error
		policy, err = decodeRecoveryPolicy(policyJSON)
		if err != nil {
			return fmt.Errorf("读取恢复策略: %w", err)
		}
	}
	now := time.Now()
	var existing models.RecoveryPlan
	if err := tx.Where("user_id = ? AND source_session_id = ?", userID, sessionID).First(&existing).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("读取恢复计划: %w", err)
	}
	maxHP := engine.CalcMaxHP(engine.DefaultTuning(), state.Strength)
	targets := map[string]float64{
		"hp":        maxHP * float64(policy.HP.TargetPercent) / 100,
		"energy":    100 * float64(policy.Energy.TargetPercent) / 100,
		"hydration": 100 * float64(policy.Hydration.TargetPercent) / 100,
	}
	rates, err := hideoutRecoveryRatesTx(tx, userID)
	if err != nil {
		return err
	}
	actualMethods, err := applyRecoveryPolicyTx(tx, userID, &state, policy, targets, rates)
	if err != nil {
		return err
	}
	ratesByResource := map[string]float64{"hp": rates.HP, "energy": rates.Energy, "hydration": rates.Hydration}
	if err := tx.Model(&models.Character{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"hp": state.HP, "energy": state.Energy, "hydration": state.Hydration, "needs_updated_at": now,
	}).Error; err != nil {
		return fmt.Errorf("保存恢复后的角色资源: %w", err)
	}
	values := map[string]float64{"hp": state.HP, "energy": state.Energy, "hydration": state.Hydration}
	plan := models.RecoveryPlan{UserID: userID, SourceSessionID: sessionID, Status: "running", PolicyJSON: policyJSON, StartedAt: now}
	if err := tx.Create(&plan).Error; err != nil {
		return fmt.Errorf("创建恢复计划: %w", err)
	}
	for _, resource := range []string{"hp", "energy", "hydration"} {
		current := values[resource]
		target := targets[resource]
		choice := recoveryChoiceForResource(policy, resource)
		actualMethod := actualMethods[resource]
		rate := 0.0
		if actualMethod == recoveryMethodHideout {
			rate = ratesByResource[resource]
		}
		status := "failed"
		var completeAt *time.Time
		if current >= target {
			status = "completed"
		} else if rate > 0 {
			status = "running"
			// 预计完成时间 = 当前时间 + 剩余缺口 / 每小时速率（不足 1 小时向上取整）。
			complete := now.Add(time.Duration(math.Ceil((target - current) / rate * float64(time.Hour))))
			completeAt = &complete
		}
		task := models.RecoveryTask{
			RecoveryPlanID: plan.ID, UserID: userID, ResourceType: resource,
			StartValue: current, CurrentValue: current, TargetValue: target, RatePerHour: rate,
			PrimaryMethod: choice.PrimaryMethod, ActualMethod: actualMethod, StartedAt: now, CompleteAt: completeAt, Status: status,
		}
		if err := tx.Create(&task).Error; err != nil {
			return fmt.Errorf("创建%s恢复任务: %w", resource, err)
		}
	}
	return completeRecoveryPlanIfDoneTx(tx, userID, plan.ID, now)
}

// applyRecoveryPolicyTx 按策略逐资源尝试首选/备用/商人三种即时恢复方式，藏身处仅作为持续恢复兜底。
func applyRecoveryPolicyTx(tx *gorm.DB, userID uint, state *engine.CharacterState, policy RecoveryPolicy, targets map[string]float64, rates recoveryRates) (map[string]string, error) {
	actualMethods := make(map[string]string, 3)
	for _, resource := range []string{"hp", "energy", "hydration"} {
		choice := recoveryChoiceForResource(policy, resource)
		actualMethod := recoveryMethodNone
		if resourceValue(*state, resource) >= targets[resource] {
			actualMethods[resource] = actualMethod
			continue
		}
		methods := []string{choice.PrimaryMethod, choice.FallbackMethod}
		if policy.MerchantEnable {
			methods = append(methods, recoveryMethodMerchant)
		}
		attempted := make(map[string]struct{}, len(methods))
		hideoutAvailable := false
		for _, method := range methods {
			if method == recoveryMethodNone {
				continue
			}
			if _, exists := attempted[method]; exists {
				continue
			}
			attempted[method] = struct{}{}
			if method == recoveryMethodHideout {
				hideoutAvailable = recoveryRateForResource(rates, resource) > 0
				continue
			}
			used, err := applyRecoveryMethodTx(tx, userID, state, resource, method, targets)
			if err != nil {
				return nil, err
			}
			if used {
				actualMethod = method
			}
			if resourceValue(*state, resource) >= targets[resource] {
				break
			}
		}
		if resourceValue(*state, resource) < targets[resource] && hideoutAvailable {
			// 藏身处是持续恢复方式，不能被后续失败的即时恢复尝试覆盖。
			actualMethod = recoveryMethodHideout
		}
		actualMethods[resource] = actualMethod
	}
	return actualMethods, nil
}

// applyRecoveryMethodTx 按方式分发即时恢复：库存消耗或商人购买，藏身处/无方式不消耗。
func applyRecoveryMethodTx(tx *gorm.DB, userID uint, state *engine.CharacterState, resource, method string, targets map[string]float64) (bool, error) {
	switch method {
	case recoveryMethodInventory:
		return applyInventoryRecoveryForResourceTx(tx, userID, state, resource, targets)
	case recoveryMethodMerchant:
		return applyMerchantRecoveryTx(tx, userID, state, resource, targets)
	case recoveryMethodHideout, recoveryMethodNone:
		return false, nil
	default:
		return false, fmt.Errorf("未知恢复方式 %s", method)
	}
}

type recoveryRates struct {
	HP        float64
	Energy    float64
	Hydration float64
	Stress    float64
}

// recoveryRateForResource 按资源名取出对应的藏身处每小时恢复速率。
func recoveryRateForResource(rates recoveryRates, resource string) float64 {
	switch resource {
	case "energy":
		return rates.Energy
	case "hydration":
		return rates.Hydration
	default:
		return rates.HP
	}
}

// hideoutRecoveryRatesTx 汇总 ready 状态设施（需供电的先结算发电机）的基础恢复速率，并叠加设施速度加成。
func hideoutRecoveryRatesTx(tx *gorm.DB, userID uint) (recoveryRates, error) {
	if err := settleGeneratorTx(tx, userID, time.Now()); err != nil {
		return recoveryRates{}, err
	}
	generatorEnabled, err := generatorPowerEnabledTx(tx, userID)
	if err != nil {
		return recoveryRates{}, fmt.Errorf("读取发电机供电状态: %w", err)
	}
	var states []models.HideoutFacility
	if err := tx.Where("user_id = ? AND state = ?", userID, "ready").Find(&states).Error; err != nil {
		return recoveryRates{}, fmt.Errorf("读取恢复设施: %w", err)
	}
	rates := recoveryRates{}
	var speed int
	for _, state := range states {
		var level models.FacilityLevelDef
		if err := tx.Where("facility_id = ? AND level = ?", state.FacilityID, state.Level).First(&level).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return recoveryRates{}, fmt.Errorf("恢复设施 %s 等级 %d 定义不存在", state.FacilityID, state.Level)
			}
			return recoveryRates{}, fmt.Errorf("读取恢复设施 %s 等级 %d: %w", state.FacilityID, state.Level, err)
		}
		if level.RequiresPower && !generatorEnabled {
			continue
		}
		rates.HP += level.HPRecoveryPerHour
		rates.Energy += level.EnergyRecoveryPerHour
		rates.Hydration += level.HydrationRecoveryPerHour
		rates.Stress += level.StressRecoveryPerHour
		speed += level.RecoverySpeedPercent
	}
	// 恢复速度加成：各设施速度百分比之和换算为整体倍率。
	multiplier := 1 + float64(speed)/100
	rates.HP *= multiplier
	rates.Energy *= multiplier
	rates.Hydration *= multiplier
	rates.Stress *= multiplier
	return rates, nil
}

// applyInventoryRecoveryForResourceTx 按优先级依次消耗库存恢复物品，直到达到目标或无可消耗物品。
func applyInventoryRecoveryForResourceTx(tx *gorm.DB, userID uint, state *engine.CharacterState, resource string, targets map[string]float64) (bool, error) {
	var defs []models.ItemUseDef
	if err := tx.Where("usable_in_hideout = ? AND (hp_recovery > 0 OR energy_recovery > 0 OR hydration_recovery > 0)", true).
		Order("use_priority asc, item_id asc").Find(&defs).Error; err != nil {
		return false, fmt.Errorf("读取自动恢复物品: %w", err)
	}
	usedAny := false
	for {
		progress := false
		for _, def := range defs {
			if resourceValue(*state, resource) >= targets[resource] {
				return usedAny, nil
			}
			if recoveryEffect(def, resource) <= 0 || !canApplyRecoveryEffects(*state, def, targets) {
				continue
			}
			used, ratio, err := consumeRecoveryItemTx(tx, userID, def)
			if err != nil {
				return false, err
			}
			if !used {
				continue
			}
			applyRecoveryEffects(state, def, ratio)
			progress = true
			usedAny = true
		}
		if !progress {
			return usedAny, nil
		}
	}
}

type merchantRecoveryCandidate struct {
	item       catalogItem
	use        models.ItemUseDef
	paidPrice  int
	efficiency float64
}

// applyMerchantRecoveryTx 按“单价/恢复量”效率排序候选商人医疗商品，并在现金充足时循环购买直到达到目标。
func applyMerchantRecoveryTx(tx *gorm.DB, userID uint, state *engine.CharacterState, resource string, targets map[string]float64) (bool, error) {
	var defs []models.ItemUseDef
	if err := tx.Where("usable_in_hideout = ? AND (hp_recovery > 0 OR energy_recovery > 0 OR hydration_recovery > 0)", true).
		Order("item_id asc").Find(&defs).Error; err != nil {
		return false, fmt.Errorf("读取商人恢复物品: %w", err)
	}
	catalogRepo := catalog.New(tx)
	itemIDs := make([]string, 0, len(defs))
	for _, def := range defs {
		if recoveryEffect(def, resource) > 0 {
			itemIDs = append(itemIDs, def.ItemID)
		}
	}
	catalogItems, err := catalogRepo.FindByIDs(itemIDs)
	if err != nil {
		return false, fmt.Errorf("读取商人恢复商品目录: %w", err)
	}
	candidates := make([]merchantRecoveryCandidate, 0, len(defs))
	for _, def := range defs {
		effect := recoveryEffect(def, resource)
		if effect <= 0 {
			continue
		}
		item, ok := catalogItems[def.ItemID]
		if !ok {
			return false, fmt.Errorf("读取商人恢复商品 %s: %w", def.ItemID, catalog.ErrItemNotFound)
		}
		if item.MerchantCategory != "medical" {
			continue
		}
		if err := applyMerchantPriceForUser(tx, userID, &item); err != nil {
			if errors.Is(err, ErrMerchantUnavailable) || errors.Is(err, ErrPurchaseUnavailable) {
				continue
			}
			return false, err
		}
		paidPrice := item.PaidPrice
		if paidPrice <= 0 {
			paidPrice = item.Price
		}
		candidates = append(candidates, merchantRecoveryCandidate{
			item: item, use: def, paidPrice: paidPrice, efficiency: float64(paidPrice) / effect,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].efficiency != candidates[j].efficiency {
			return candidates[i].efficiency < candidates[j].efficiency
		}
		return candidates[i].item.ID < candidates[j].item.ID
	})

	used := false
	for resourceValue(*state, resource) < targets[resource] {
		purchased := false
		for _, candidate := range candidates {
			if !canApplyRecoveryEffects(*state, candidate.use, targets) {
				continue
			}
			if err := deductCash(tx, userID, candidate.paidPrice); err != nil {
				if errors.Is(err, ErrPurchaseUnavailable) {
					continue
				}
				return used, err
			}
			applyRecoveryEffects(state, candidate.use, 1)
			used = true
			purchased = true
			break
		}
		if !purchased {
			break
		}
	}
	return used, nil
}

// recoveryChoiceForResource 按资源名取出对应的恢复策略选择。
func recoveryChoiceForResource(policy RecoveryPolicy, resource string) RecoveryChoice {
	switch resource {
	case "energy":
		return policy.Energy
	case "hydration":
		return policy.Hydration
	default:
		return policy.HP
	}
}

// recoveryEffect 按资源名读取物品定义中对应的单次恢复量。
func recoveryEffect(def models.ItemUseDef, resource string) float64 {
	switch resource {
	case "energy":
		return def.EnergyRecovery
	case "hydration":
		return def.HydrationRecovery
	default:
		return def.HPRecovery
	}
}

// resourceValue 按资源名读取角色当前资源值。
func resourceValue(state engine.CharacterState, resource string) float64 {
	switch resource {
	case "energy":
		return state.Energy
	case "hydration":
		return state.Hydration
	default:
		return state.HP
	}
}

// applyRecoveryEffects 按使用比例把物品恢复量施加到角色资源，并钳制在合法区间。
func applyRecoveryEffects(state *engine.CharacterState, def models.ItemUseDef, ratio float64) {
	state.HP = clampResource(state.HP+def.HPRecovery*ratio, 0, engine.CalcMaxHP(engine.DefaultTuning(), state.Strength))
	state.Energy = clampResource(state.Energy+def.EnergyRecovery*ratio, 0, 100)
	state.Hydration = clampResource(state.Hydration+def.HydrationRecovery*ratio, 0, 100)
}

// canApplyRecoveryEffects 判断使用该物品后是否会把负向效果（如降饮水）压过目标值，避免过度使用。
func canApplyRecoveryEffects(state engine.CharacterState, def models.ItemUseDef, targets map[string]float64) bool {
	if def.EnergyRecovery < 0 && state.Energy+def.EnergyRecovery < targets["energy"] {
		return false
	}
	if def.HydrationRecovery < 0 && state.Hydration+def.HydrationRecovery < targets["hydration"] {
		return false
	}
	return true
}

// consumeRecoveryItemTx 消耗一个库存恢复物品：实例类扣耐久并处理耗尽，普通类扣数量并删除零库存行。
func consumeRecoveryItemTx(tx *gorm.DB, userID uint, def models.ItemUseDef) (bool, float64, error) {
	if def.InstanceRequired {
		var instance models.ItemInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND item_id = ? AND status = ? AND location_type = ? AND current_durability > 0", userID, def.ItemID, "normal", "inventory").Order("current_durability asc, id asc").First(&instance).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return false, 0, nil
			}
			return false, 0, err
		}
		use := def.UseDurability
		if use <= 0 || use > instance.CurrentDurability {
			use = instance.CurrentDurability
		}
		ratio := 1.0
		if def.UseDurability > 0 && use < def.UseDurability {
			ratio = use / def.UseDurability
		}
		instance.CurrentDurability -= use
		status := "normal"
		if instance.CurrentDurability <= 0 {
			instance.CurrentDurability = 0
			status = "depleted"
		}
		if err := tx.Model(&models.ItemInstance{}).Where("user_id = ? AND id = ?", userID, instance.ID).Updates(map[string]interface{}{
			"current_durability": instance.CurrentDurability, "status": status,
		}).Error; err != nil {
			return false, 0, err
		}
		return true, ratio, nil
	}
	var inventory models.Inventory
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND item_id = ? AND quantity > 0", userID, def.ItemID).Order("raid_extract desc, id asc").First(&inventory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, 0, nil
		}
		return false, 0, err
	}
	if err := tx.Model(&models.Inventory{}).Where("user_id = ? AND id = ?", userID, inventory.ID).Update("quantity", gorm.Expr("quantity - 1")).Error; err != nil {
		return false, 0, err
	}
	if inventory.Quantity <= 1 {
		if err := tx.Delete(&inventory).Error; err != nil {
			return false, 0, err
		}
	}
	return true, 1, nil
}

// settleRecoveryForUser 加用户资源锁后结算该用户的恢复计划，供读取角色资源前调用。
func settleRecoveryForUser(db *gorm.DB, userID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		return settleRecoveryForUserTx(tx, userID)
	})
}

// settleRecoveryForUserTx 在调用方已经持有用户资源锁时结算恢复计划。
func settleRecoveryForUserTx(tx *gorm.DB, userID uint) error {
	now := time.Now()
	if err := settleDueHideoutJobsTx(tx, userID, now); err != nil {
		return err
	}
	var plan models.RecoveryPlan
	if err := tx.Where("user_id = ? AND status = ?", userID, "running").Order("id desc").First(&plan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	var character models.Character
	if err := tx.Where("user_id = ?", userID).First(&character).Error; err != nil {
		return err
	}
	var tasks []models.RecoveryTask
	if err := tx.Where("user_id = ? AND recovery_plan_id = ?", userID, plan.ID).Find(&tasks).Error; err != nil {
		return err
	}
	rates, err := hideoutRecoveryRatesTx(tx, userID)
	if err != nil {
		return err
	}
	ratesByResource := map[string]float64{"hp": rates.HP, "energy": rates.Energy, "hydration": rates.Hydration}
	for _, task := range tasks {
		if task.Status == "completed" || task.Status == "failed" {
			continue
		}
		rate := 0.0
		if task.CurrentValue < task.TargetValue {
			if ratesByResource[task.ResourceType] <= 0 {
				// 旧版本可能把失败的即时恢复方式写成 running，必须转为终态。
				task.Status = "failed"
				task.RatePerHour = 0
				task.StartedAt = now
				task.CompleteAt = nil
				if err := tx.Model(&models.RecoveryTask{}).Where("id = ? AND user_id = ?", task.ID, userID).Updates(map[string]interface{}{
					"actual_method": task.ActualMethod, "rate_per_hour": task.RatePerHour, "started_at": task.StartedAt, "complete_at": task.CompleteAt, "status": task.Status,
				}).Error; err != nil {
					return err
				}
				continue
			}
			// 旧版本可能已经记录了即时方式；只要当前有持续速率，就切换到藏身处等待。
			task.ActualMethod = recoveryMethodHideout
			rate = ratesByResource[task.ResourceType]
		}
		elapsed := now.Sub(task.StartedAt).Hours()
		if elapsed > 0 && rate > 0 {
			task.CurrentValue = clampResource(task.CurrentValue+elapsed*rate, 0, task.TargetValue)
		}
		task.RatePerHour = rate
		task.StartedAt = now
		if task.CurrentValue >= task.TargetValue {
			task.CurrentValue = task.TargetValue
			task.Status = "completed"
			complete := now
			task.CompleteAt = &complete
		} else {
			task.Status = "running"
			if rate > 0 {
				complete := now.Add(time.Duration(math.Ceil((task.TargetValue - task.CurrentValue) / rate * float64(time.Hour))))
				task.CompleteAt = &complete
			} else {
				task.CompleteAt = nil
			}
		}
		if err := tx.Model(&models.RecoveryTask{}).Where("id = ? AND user_id = ?", task.ID, userID).Updates(map[string]interface{}{
			"current_value": task.CurrentValue, "actual_method": task.ActualMethod, "rate_per_hour": task.RatePerHour, "started_at": task.StartedAt, "complete_at": task.CompleteAt, "status": task.Status,
		}).Error; err != nil {
			return err
		}
		switch task.ResourceType {
		case "hp":
			character.HP = task.CurrentValue
		case "energy":
			character.Energy = task.CurrentValue
		case "hydration":
			character.Hydration = task.CurrentValue
		}
	}
	if err := tx.Model(&models.Character{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"hp": character.HP, "energy": character.Energy, "hydration": character.Hydration, "needs_updated_at": now,
	}).Error; err != nil {
		return err
	}
	return completeRecoveryPlanIfDoneTx(tx, userID, plan.ID, now)
}

// ensureRecoveryReadyForStartTx 防止恢复计划未达到目标时启动新的探索。
func ensureRecoveryReadyForStartTx(tx *gorm.DB, userID uint) error {
	var plan models.RecoveryPlan
	if err := tx.Where("user_id = ? AND status = ?", userID, "running").Order("id desc").First(&plan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("读取恢复计划: %w", err)
	}
	var character models.Character
	if err := tx.Where("user_id = ?", userID).First(&character).Error; err != nil {
		return fmt.Errorf("读取恢复角色状态: %w", err)
	}
	var tasks []models.RecoveryTask
	if err := tx.Where("user_id = ? AND recovery_plan_id = ?", userID, plan.ID).Find(&tasks).Error; err != nil {
		return fmt.Errorf("读取恢复任务: %w", err)
	}
	for _, task := range tasks {
		// 用极小容差比较浮点目标值，避免精度误差导致未恢复完成的误判。
		if task.Status == "completed" || task.Status == "failed" || resourceValueFromCharacter(character, task.ResourceType) >= task.TargetValue-0.000001 {
			continue
		}
		return fmt.Errorf("角色正在恢复%s，当前 %.1f / %.1f", recoveryResourceName(task.ResourceType), resourceValueFromCharacter(character, task.ResourceType), task.TargetValue)
	}
	return nil
}

// resourceValueFromCharacter 按资源名读取角色表上的当前资源值。
func resourceValueFromCharacter(character models.Character, resource string) float64 {
	switch resource {
	case "hp":
		return character.HP
	case "energy":
		return character.Energy
	case "hydration":
		return character.Hydration
	default:
		return 0
	}
}

// recoveryResourceName 将资源键转为中文名，用于展示与报错文案。
func recoveryResourceName(resource string) string {
	switch resource {
	case "hp":
		return "生命"
	case "energy":
		return "能量"
	case "hydration":
		return "饮水"
	default:
		return "资源"
	}
}

// SettleRecoveryForUser 在读取角色资源或启动行动前结算已经经过的恢复时间。
func SettleRecoveryForUser(db *gorm.DB, userID uint) error {
	return settleRecoveryForUser(db, userID)
}

// completeRecoveryPlanIfDoneTx 当计划内所有任务都到终态时收尾计划：全部完成则 completed，有失败则 failed。
func completeRecoveryPlanIfDoneTx(tx *gorm.DB, userID, planID uint, now time.Time) error {
	var pending int64
	if err := tx.Model(&models.RecoveryTask{}).Where("user_id = ? AND recovery_plan_id = ? AND status NOT IN ?", userID, planID, []string{"completed", "failed"}).Count(&pending).Error; err != nil {
		return err
	}
	if pending > 0 {
		return nil
	}
	var failed int64
	if err := tx.Model(&models.RecoveryTask{}).Where("user_id = ? AND recovery_plan_id = ? AND status = ?", userID, planID, "failed").Count(&failed).Error; err != nil {
		return err
	}
	status := "completed"
	if failed > 0 {
		status = "failed"
	}
	return tx.Model(&models.RecoveryPlan{}).Where("user_id = ? AND id = ?", userID, planID).Updates(map[string]interface{}{"status": status, "completed_at": now}).Error
}

// GetCurrentRecoveryForUser 先结算恢复时间，再返回当前（最新）恢复计划及其任务列表，无计划时返回 nil。
func GetCurrentRecoveryForUser(db *gorm.DB, userID uint) (*RecoveryView, error) {
	if err := settleRecoveryForUser(db, userID); err != nil {
		return nil, err
	}
	var plan models.RecoveryPlan
	if err := db.Where("user_id = ?", userID).Order("id desc").First(&plan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	var tasks []models.RecoveryTask
	if err := db.Where("user_id = ? AND recovery_plan_id = ?", userID, plan.ID).Order("id asc").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return &RecoveryView{Plan: plan, Tasks: tasks}, nil
}

// clampResource 把数值钳制在 [minValue, maxValue] 区间内。
func clampResource(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
