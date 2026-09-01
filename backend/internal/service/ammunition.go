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

// reserveCarriedAmmo 开局时校验武器口径与携弹下限，并从库存扣出携行弹药（与快照同一事务，口径数据强一致）。
func reserveCarriedAmmo(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, weaponID, ammoID string, rounds int) (engine.CarriedAmmo, error) {
	weapon, ok := snapshot.Weapons[weaponID]
	if !ok {
		return engine.CarriedAmmo{}, fmt.Errorf("武器 %s 不在场景快照中", weaponID)
	}
	if weapon.AmmoPerRound <= 0 {
		if ammoID != "" || rounds != 0 {
			return engine.CarriedAmmo{}, fmt.Errorf("近战武器不能携带弹药")
		}
		return engine.CarriedAmmo{}, nil
	}
	if rounds < weapon.AmmoPerRound {
		return engine.CarriedAmmo{}, fmt.Errorf("携弹量至少为 %d 发", weapon.AmmoPerRound)
	}
	ammo, ok := snapshot.Ammos[ammoID]
	if !ok {
		return engine.CarriedAmmo{}, fmt.Errorf("弹药 %s 不存在", ammoID)
	}
	if ammo.CaliberID != weapon.CaliberID {
		return engine.CarriedAmmo{}, fmt.Errorf("弹药口径 %s 与武器口径 %s 不匹配", ammo.CaliberID, weapon.CaliberID)
	}
	if err := removeInventoryItem(tx, userID, ammoID, rounds); err != nil {
		return engine.CarriedAmmo{}, err
	}
	return carriedAmmoWithPreference(ammo, rounds, ammo, rounds), nil
}

// returnCarriedAmmoTx 会话终态把仍有剩余的携行弹药加回库存并清空携带状态，避免弹药随会话流失。
func returnCarriedAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, state *engine.EngineState) error {
	if state.Ammo.ID == "" || state.Ammo.Rounds <= 0 {
		state.Ammo = engine.CarriedAmmo{}
		return nil
	}
	if err := returnAmmoRoundsTx(tx, userID, snapshot, state.Ammo); err != nil {
		return err
	}
	state.Ammo = engine.CarriedAmmo{}
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
	refill, err := refillSessionAmmoTx(tx, userID, snapshot, weapon.ID, &state)
	return state.Ammo, refill, err
}

// ensureSessionAmmoTx 在下一局开始前保证至少具备一次有效攻击所需的弹药。
// 业务失败时不修改数据库和 state，调用方可以安全地按原状态完成 Session 结算。
func ensureSessionAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, state *engine.EngineState) (*ammoRefillResult, error) {
	weapon, ok := snapshot.Weapons[state.Loadout.WeaponID]
	if !ok {
		return nil, fmt.Errorf("自动补给武器 %s 不在场景快照中", state.Loadout.WeaponID)
	}
	if weapon.AmmoPerRound <= 0 || state.Ammo.Rounds >= weapon.AmmoPerRound {
		return nil, nil
	}
	if state.Ammo.PreferredID == "" || state.Ammo.TargetRounds < weapon.AmmoPerRound {
		return nil, fmt.Errorf("%w：Session 缺少有效弹药补给配置", ErrPurchaseUnavailable)
	}
	return refillSessionAmmoTx(tx, userID, snapshot, weapon.ID, state)
}

// refillSessionAmmoTx 自动补给弹药：先校验偏好口径/等级与目标数量，扣现金后归还旧弹并换为商人供应的新弹。
func refillSessionAmmoTx(tx *gorm.DB, userID uint, snapshot engine.ScenarioSnapshot, weaponID string, state *engine.EngineState) (*ammoRefillResult, error) {
	weapon, ok := snapshot.Weapons[weaponID]
	if !ok || weapon.AmmoPerRound <= 0 {
		return nil, fmt.Errorf("自动补给武器 %s 无效", weaponID)
	}
	preference, ok := snapshot.Ammos[state.Ammo.PreferredID]
	if !ok || preference.CaliberID != weapon.CaliberID || preference.Level != state.Ammo.PreferredLevel || state.Ammo.TargetRounds < weapon.AmmoPerRound {
		return nil, fmt.Errorf("Session 自动补给目标无效")
	}
	fromID, fromLevel := state.Ammo.ID, state.Ammo.Level
	targetRounds := state.Ammo.TargetRounds
	supply, err := selectMerchantAmmoSupply(snapshot, preference.CaliberID, preference.Level)
	if err != nil {
		return nil, err
	}
	totalPrice := supply.UnitPrice * targetRounds
	if err := deductCash(tx, userID, totalPrice); err != nil {
		if errors.Is(err, ErrPurchaseUnavailable) {
			return nil, fmt.Errorf("%w：自动补给 %d 发 N%d 弹药需要 ￥%d", ErrPurchaseUnavailable, targetRounds, supply.Level, totalPrice)
		}
		return nil, err
	}
	// 所有可预期的业务失败都已完成校验。从这里开始的错误必须由外层事务整体回滚。
	if state.Ammo.ID != "" && state.Ammo.Rounds > 0 {
		if err := returnAmmoRoundsTx(tx, userID, snapshot, state.Ammo); err != nil {
			return nil, err
		}
	}
	current := snapshot.Ammos[supply.AmmoID]
	state.Ammo = carriedAmmoWithPreference(current, targetRounds, preference, targetRounds)
	return &ammoRefillResult{
		FromAmmoID: fromID, ToAmmoID: current.ID, FromLevel: fromLevel, ToLevel: current.Level,
		Rounds: state.Ammo.Rounds, UnitPrice: supply.UnitPrice, TotalPrice: totalPrice, Source: "merchant_fallback",
	}, nil
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
