// 玩家视图：在角色本体之上补充资源上限与当前恢复速度，供前端角色页展示。
package service

import (
	"idle/internal/engine"
	"idle/internal/models"

	"gorm.io/gorm"
)

// RecoveryPerHourView 是三类资源当前每小时的被动恢复速度。
type RecoveryPerHourView struct {
	HP        float64 `json:"hp"`
	Energy    float64 `json:"energy"`
	Hydration float64 `json:"hydration"`
	Stress    float64 `json:"stress"`
}

// PlayerView 是 GET /api/player 的响应视图。
type PlayerView struct {
	models.Character
	HPMax           float64             `json:"hpMax"`
	EnergyMax       float64             `json:"energyMax"`
	HydrationMax    float64             `json:"hydrationMax"`
	StressMax       float64             `json:"stressMax"`
	RecoveryPerHour RecoveryPerHourView `json:"recoveryPerHour"`
}

// BuildPlayerViewForUser 组装带资源上限与回复速度的玩家视图。
func BuildPlayerViewForUser(db *gorm.DB, character models.Character) (PlayerView, error) {
	rates, err := recoveryRatesForUser(db, character.UserID)
	if err != nil {
		return PlayerView{}, err
	}
	return PlayerView{
		Character:    character,
		HPMax:        engine.CalcMaxHP(character.Strength),
		EnergyMax:    100,
		HydrationMax: 100,
		StressMax:    engine.CalcStressThreshold(engine.EffectiveSkill(character.Resist, character.Strength)),
		RecoveryPerHour: RecoveryPerHourView{
			HP:        rates.HP,
			Energy:    rates.Energy,
			Hydration: rates.Hydration,
			Stress:    rates.Stress,
		},
	}, nil
}

// recoveryRatesForUser 在独立事务内取当前恢复速度（内部会先结算发电机燃料游标）。
func recoveryRatesForUser(db *gorm.DB, userID uint) (recoveryRates, error) {
	var rates recoveryRates
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		var err error
		rates, err = hideoutRecoveryRatesTx(tx, userID)
		return err
	})
	return rates, err
}
