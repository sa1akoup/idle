// 弹药持久化适配：校验口径、在 Session 启动时预留携弹，并在终态安全返还剩余弹药。
package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

type ammoRefillResult struct {
	StackIndex int // 原携带弹药栈的 0-based 索引。
	FromAmmoID string
	ToAmmoID   string
	FromLevel  int
	ToLevel    int
	Rounds     int
	UnitPrice  int
	TotalPrice int
	Source     string
}

// decodeSessionAmmoState 从 Session 行解码场景快照与连续状态，供结算/失败返还弹药等场景使用。
func decodeSessionAmmoState(sess models.Session) (engine.ScenarioSnapshot, engine.EngineState, error) {
	var snapshot engine.ScenarioSnapshot
	if err := json.Unmarshal([]byte(sess.ScenarioSnapshot), &snapshot); err != nil {
		return engine.ScenarioSnapshot{}, engine.EngineState{}, fmt.Errorf("读取 Session 场景快照: %w", err)
	}
	var state engine.EngineState
	if err := json.Unmarshal([]byte(sess.StateJSON), &state); err != nil {
		return engine.ScenarioSnapshot{}, engine.EngineState{}, fmt.Errorf("读取 Session 连续状态: %w", err)
	}
	return snapshot, state, nil
}

// compactAmmoCells 去除前端固定槽位产生的空占位，保留非空槽位的原始顺序。
func compactAmmoCells(cells []models.AmmoCell) []models.AmmoCell {
	result := make([]models.AmmoCell, 0, len(cells))
	for _, cell := range cells {
		if cell.AmmoID == "" && cell.Rounds == 0 {
			continue
		}
		result = append(result, cell)
	}
	return result
}

// reserveCarriedAmmoStacks 开局时按携带弹药槽位校验口径与下限，并从库存逐栈扣出发数。
// 返回带自动补给偏好的携带栈（每栈偏好自身，供耗尽时按等级自动补给）。
func reserveCarriedAmmoStacks(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, weaponID string, cells []models.AmmoCell) ([]engine.CarriedAmmo, error) {
	cells = compactAmmoCells(cells)
	weapon, ok := snapshot.Weapons[weaponID]
	if !ok {
		return nil, fmt.Errorf("武器 %s 不在场景快照中", weaponID)
	}
	if weapon.AmmoPerRound <= 0 {
		if len(cells) > 0 {
			return nil, fmt.Errorf("近战武器不能携带弹药")
		}
		return nil, nil
	}
	if len(cells) == 0 {
		return nil, fmt.Errorf("未配置携带弹药，请先在角色页面设置")
	}
	stacks := make([]engine.CarriedAmmo, 0, len(cells))
	usable := 0
	for _, cell := range cells {
		if cell.AmmoID == "" || cell.Rounds <= 0 {
			return nil, fmt.Errorf("携带弹药栏位配置不完整")
		}
		ammo, ok := snapshot.Ammos[cell.AmmoID]
		if !ok {
			return nil, fmt.Errorf("弹药 %s 不存在", cell.AmmoID)
		}
		if ammo.CaliberID != weapon.CaliberID {
			return nil, fmt.Errorf("弹药 %s 口径与武器 %s 不匹配", ammo.Name, weapon.Name)
		}
		if err := removeInventoryItem(tx, userID, cell.AmmoID, cell.Rounds); err != nil {
			return nil, err
		}
		if cell.Rounds >= weapon.AmmoPerRound {
			usable++
		}
		stacks = append(stacks, carriedAmmoWithPreference(ammo, cell.Rounds, ammo, cell.Rounds))
	}
	if usable == 0 {
		return nil, fmt.Errorf("携带弹药不足以完成一次攻击，请适当增加发数")
	}
	return stacks, nil
}

// returnCarriedAmmoTx 会话终态把弹药池各栈剩余发数加回库存并清空携带状态，避免弹药随会话流失。
func returnCarriedAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, state *engine.EngineState) error {
	stacks := engine.CarriedAmmoStacks(state)
	if len(stacks) == 0 {
		state.Ammo = engine.CarriedAmmo{}
		return nil
	}
	for _, stack := range stacks {
		if stack.ID == "" || stack.Rounds <= 0 {
			continue
		}
		if err := returnAmmoRoundsTx(tx, userID, snapshot, stack); err != nil {
			return err
		}
	}
	state.Ammo = engine.CarriedAmmo{}
	state.AmmoStacks = nil
	return nil
}

// reservePresetAmmoTx 按失能恢复预设预留弹药：库存足够则直接扣出，不足则记录偏好弹药并走商人自动补给。
func reservePresetAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, preset engine.RecoveryPreset) (engine.CarriedAmmo, *ammoRefillResult, error) {
	if preset.AmmoID == "" && preset.AmmoRounds == 0 {
		return engine.CarriedAmmo{}, nil, nil
	}
	weapon, ok := snapshot.Weapons[preset.Loadout.WeaponID]
	if !ok {
		return engine.CarriedAmmo{}, nil, fmt.Errorf("恢复预设武器 %s 不存在", preset.Loadout.WeaponID)
	}
	ammo, ok := snapshot.Ammos[preset.AmmoID]
	if !ok {
		return engine.CarriedAmmo{}, nil, fmt.Errorf("恢复预设弹药 %s 不存在", preset.AmmoID)
	}
	if weapon.AmmoPerRound <= 0 || ammo.CaliberID != weapon.CaliberID || preset.AmmoRounds < weapon.AmmoPerRound {
		return engine.CarriedAmmo{}, nil, fmt.Errorf("恢复预设弹药与武器不匹配或数量不足")
	}
	available, err := ammoInventoryQuantity(tx, userID, ammo.ID)
	if err != nil {
		return engine.CarriedAmmo{}, nil, err
	}
	// 以预设数量为上限截取库存可用数；不足一发则改用商人补给分支。
	reserveRounds := available
	if reserveRounds > preset.AmmoRounds {
		reserveRounds = preset.AmmoRounds
	}
	if reserveRounds >= weapon.AmmoPerRound {
		if err := removeInventoryItem(tx, userID, ammo.ID, reserveRounds); err != nil {
			return engine.CarriedAmmo{}, nil, fmt.Errorf("预留恢复预设弹药: %w", err)
		}
		carried := carriedAmmoWithPreference(ammo, reserveRounds, ammo, preset.AmmoRounds)
		return carried, &ammoRefillResult{
			ToAmmoID: ammo.ID, ToLevel: ammo.Level, Rounds: reserveRounds, Source: "preset_warehouse",
		}, nil
	}
	state := engine.EngineState{Ammo: engine.CarriedAmmo{
		CaliberID: ammo.CaliberID, PreferredID: ammo.ID, PreferredLevel: ammo.Level, TargetRounds: preset.AmmoRounds,
	}}
	refills, err := refillSessionAmmoTx(tx, userID, snapshot, weapon.ID, &state)
	if err != nil {
		return engine.CarriedAmmo{}, nil, err
	}
	if len(refills) == 0 {
		return engine.CarriedAmmo{}, nil, fmt.Errorf("恢复预设弹药补给结果为空")
	}
	return state.Ammo, refills[0], nil
}

// ensureSessionAmmoTx 在下一局开始前保证至少具备一次有效攻击所需的弹药，并返回逐栈补给结果。
// 业务失败时不修改数据库和 state，调用方可以安全地按原状态完成 Session 结算。
func ensureSessionAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, state *engine.EngineState) ([]*ammoRefillResult, error) {
	weapon, ok := snapshot.Weapons[state.Loadout.WeaponID]
	if !ok {
		return nil, fmt.Errorf("自动补给武器 %s 不在场景快照中", state.Loadout.WeaponID)
	}
	if weapon.AmmoPerRound <= 0 || engine.HasUsableCarriedAmmoStack(snapshot, weapon, engine.CarriedAmmoStacks(state)) {
		return nil, nil
	}
	preferred := state.Ammo
	if preferred.PreferredID == "" {
		preferred = engine.BestCarriedAmmoSummary(engine.CarriedAmmoStacks(state))
		state.Ammo = preferred
	}
	if preferred.PreferredID == "" || preferred.TargetRounds < weapon.AmmoPerRound {
		return nil, fmt.Errorf("%w：Session 缺少有效弹药补给配置", ErrPurchaseUnavailable)
	}
	return refillSessionAmmoTx(tx, userID, snapshot, weapon.ID, state)
}

// refillSessionAmmoTx 自动补给弹药：先校验偏好口径/等级与目标数量，扣现金后归还旧弹并换为商人供应的新弹。
// 补给成功后按原栈顺序生成多个补给栈，每个栈保留自己的自动补给偏好。
func refillSessionAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, weaponID string, state *engine.EngineState) ([]*ammoRefillResult, error) {
	weapon, ok := snapshot.Weapons[weaponID]
	if !ok || weapon.AmmoPerRound <= 0 {
		return nil, fmt.Errorf("自动补给武器 %s 无效", weaponID)
	}
	stacks := engine.CarriedAmmoStacks(state)
	if len(stacks) == 0 && state.Ammo.PreferredID != "" {
		// 恢复预设的自动补给路径只有全局偏好，没有实际旧栈，包装成一个合成栈。
		stacks = []engine.CarriedAmmo{state.Ammo}
	}
	if len(stacks) == 0 {
		return nil, fmt.Errorf("Session 自动补给目标无效")
	}
	type refillPlan struct {
		index        int
		stack        engine.CarriedAmmo
		preference   engine.Ammo
		supply       engine.AmmoSupply
		targetRounds int
	}
	plans := make([]refillPlan, 0, len(stacks))
	totalPrice := 0
	for index, stack := range stacks {
		preferredID, preferredLevel, targetRounds := stack.PreferredID, stack.PreferredLevel, stack.TargetRounds
		if preferredID == "" {
			preferredID, preferredLevel, targetRounds = state.Ammo.PreferredID, state.Ammo.PreferredLevel, state.Ammo.TargetRounds
		}
		if preferredID == "" && stack.ID != "" {
			preferredID, preferredLevel, targetRounds = stack.ID, stack.Level, stack.Rounds
		}
		preference, ok := snapshot.Ammos[preferredID]
		if !ok || preference.CaliberID != weapon.CaliberID || preference.Level != preferredLevel || targetRounds < weapon.AmmoPerRound {
			return nil, fmt.Errorf("Session 自动补给目标无效")
		}
		supply, err := selectMerchantAmmoSupply(snapshot, preference.CaliberID, preference.Level)
		if err != nil {
			return nil, err
		}
		plans = append(plans, refillPlan{
			index: index, stack: stack, preference: preference, supply: supply, targetRounds: targetRounds,
		})
		totalPrice += supply.UnitPrice * targetRounds
	}
	if err := deductCash(tx, userID, totalPrice); err != nil {
		if errors.Is(err, ErrPurchaseUnavailable) {
			return nil, fmt.Errorf("%w：自动补给需要 ￥%d", ErrPurchaseUnavailable, totalPrice)
		}
		return nil, err
	}
	// 所有可预期的业务失败都已完成校验。从这里开始的错误必须由外层事务整体回滚。
	refilledStacks := make([]engine.CarriedAmmo, len(plans))
	results := make([]*ammoRefillResult, 0, len(plans))
	for _, plan := range plans {
		stack := plan.stack
		if stack.ID != "" && stack.Rounds > 0 {
			if err := returnAmmoRoundsTx(tx, userID, snapshot, stack); err != nil {
				return nil, err
			}
		}
		current := snapshot.Ammos[plan.supply.AmmoID]
		refilled := carriedAmmoWithPreference(current, plan.targetRounds, plan.preference, plan.targetRounds)
		refilledStacks[plan.index] = refilled
		results = append(results, &ammoRefillResult{
			StackIndex: plan.index, FromAmmoID: stack.ID, ToAmmoID: current.ID,
			FromLevel: stack.Level, ToLevel: current.Level, Rounds: refilled.Rounds,
			UnitPrice: plan.supply.UnitPrice, TotalPrice: plan.supply.UnitPrice * plan.targetRounds, Source: "merchant_fallback",
		})
	}
	state.AmmoStacks = refilledStacks
	state.Ammo = engine.BestCarriedAmmoSummary(refilledStacks)
	return results, nil
}

// selectMerchantAmmoSupply 从快照商人供应中选择可用且等级不超过偏好等级的最高级同口径弹药。
func selectMerchantAmmoSupply(snapshot engine.ScenarioSnapshot, caliberID string, preferredLevel int) (engine.AmmoSupply, error) {
	var selected engine.AmmoSupply
	for _, supply := range snapshot.AmmoSupplies {
		if !supply.Available || supply.CaliberID != caliberID || supply.Level > preferredLevel {
			continue
		}
		if selected.AmmoID == "" || supply.Level > selected.Level {
			// 在同口径可用供应中优先选等级更高（但不超偏好等级）的弹药。
			selected = supply
		}
	}
	if selected.AmmoID == "" {
		return engine.AmmoSupply{}, fmt.Errorf("%w：武器商人没有可用的 %s 弹药", ErrPurchaseUnavailable, caliberID)
	}
	return selected, nil
}

// returnAmmoRoundsTx 校验剩余弹药与场景快照一致后，把指定发数弹药作为普通商品加回库存。
func returnAmmoRoundsTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, carried engine.CarriedAmmo) error {
	ammo, ok := snapshot.Ammos[carried.ID]
	if !ok || ammo.CaliberID != carried.CaliberID || ammo.Level != carried.Level {
		return fmt.Errorf("Session 剩余弹药 %s 与场景快照不一致", carried.ID)
	}
	item := catalogItem{
		ID: ammo.ID, Name: ammo.Name, Kind: "ammo", Price: ammo.Price, Slots: 1,
		MerchantCategory: ammo.MerchantCategory, RepRequirement: ammo.RepRequirement,
		RoundsPerSlot: ammo.RoundsPerSlot, AmmoLevel: ammo.Level,
	}
	if err := addInventoryItem(tx, userID, item, carried.Rounds, false); err != nil {
		return fmt.Errorf("返还 Session 剩余弹药: %w", err)
	}
	return nil
}

// carriedAmmoWithPreference 构造携行弹药状态：当前实际携带的弹药，加上自动补给偏好的目标弹药与目标数量。
func carriedAmmoWithPreference(current engine.Ammo, rounds int, preferred engine.Ammo, targetRounds int) engine.CarriedAmmo {
	return engine.CarriedAmmo{
		ID: current.ID, CaliberID: current.CaliberID, Level: current.Level, Rounds: rounds,
		PreferredID: preferred.ID, PreferredLevel: preferred.Level, TargetRounds: targetRounds,
	}
}

// ammoInventoryQuantity 汇总某弹药在库存中的所有行数量总和。
func ammoInventoryQuantity(db *gorm.DB, userID uint, ammoID string) (int, error) {
	var quantity int
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, ammoID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&quantity).Error; err != nil {
		return 0, fmt.Errorf("读取弹药库存 %s: %w", ammoID, err)
	}
	return quantity, nil
}
