// 商人好感度辅助：处理用户维度的声望奖励。
package service

import (
	"errors"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

// AwardReputation 提升指定商人的好感度。
func AwardReputation(db *gorm.DB, merchantID string, amount int) error {
	return AwardReputationForUser(db, models.DefaultUserID, merchantID, amount)
}

// AwardReputationForUser 提升指定用户的商人好感度。
func AwardReputationForUser(db *gorm.DB, userID uint, merchantID string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("好感度奖励需为正数")
	}

	var merchant models.MerchantDef
	if err := db.First(&merchant, "id = ?", merchantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商人不存在")
		}
		return fmt.Errorf("读取商人: %w", err)
	}
	state := models.UserMerchantState{UserID: userID, MerchantID: merchantID, Reputation: merchant.Reputation, Unlocked: merchant.Open}
	result := db.Where("user_id = ? AND merchant_id = ?", userID, merchantID).FirstOrCreate(&state)
	if result.Error != nil {
		return fmt.Errorf("读取商人状态: %w", result.Error)
	}
	result = db.Model(&models.UserMerchantState{}).
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
