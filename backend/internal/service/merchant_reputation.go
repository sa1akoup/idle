// 商人好感度辅助：处理用户维度的声望奖励。
package service

import (
	"errors"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

const (
	sellReputationThreshold     = 200
	sellReputationHighThreshold = 1000
	weaponMerchantID            = "weapon"
)

// AwardReputationForUser 提升指定用户的商人好感度。
func AwardReputationForUser(db *gorm.DB, userID uint, merchantID string, amount int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		return awardReputationTx(tx, userID, merchantID, amount)
	})
}

func awardReputationTx(tx *gorm.DB, userID uint, merchantID string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("好感度奖励需为正数")
	}

	var merchant models.MerchantDef
	if err := tx.First(&merchant, "id = ?", merchantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商人不存在")
		}
		return fmt.Errorf("读取商人: %w", err)
	}
	state := models.UserMerchantState{UserID: userID, MerchantID: merchantID, Reputation: merchant.Reputation, Unlocked: merchant.Open}
	result := tx.Where("user_id = ? AND merchant_id = ?", userID, merchantID).FirstOrCreate(&state)
	if result.Error != nil {
		return fmt.Errorf("读取商人状态: %w", result.Error)
	}
	result = tx.Model(&models.UserMerchantState{}).
		Where("user_id = ? AND merchant_id = ?", userID, merchantID).
		Update("reputation", gorm.Expr("reputation + ?", amount))
	if result.Error != nil {
		return fmt.Errorf("提升好感度: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("商人不存在")
	}
	return nil
}

func sellReputationAmount(total int) int {
	amount := 0
	if total >= sellReputationThreshold {
		amount++
	}
	if total >= sellReputationHighThreshold {
		amount++
	}
	return amount
}

func grantSellReputationTx(tx *gorm.DB, userID uint, merchantID string, total int) error {
	amount := sellReputationAmount(total)
	if amount == 0 {
		return nil
	}
	return awardReputationTx(tx, userID, merchantID, amount)
}

func grantSessionSuccessReputationTx(tx *gorm.DB, userID uint) error {
	var merchant models.MerchantDef
	if err := tx.First(&merchant, "id = ?", weaponMerchantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("读取武器商人: %w", err)
	}
	return awardReputationTx(tx, userID, weaponMerchantID, 1)
}
