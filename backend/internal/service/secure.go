package service

import (
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

const defaultSecureContainerID = "secure_01"

func ensureStarterSecureContainerTx(tx *gorm.DB, userID uint) error {
	var count int64
	if err := tx.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ? AND quantity > 0", userID, defaultSecureContainerID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		item := models.Inventory{
			UserID: userID, ItemID: defaultSecureContainerID, Name: "简易安全袋", Kind: "secure",
			Quantity: 1, Price: 400, Weight: 1, Slots: 1,
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("发放简易安全袋: %w", err)
		}
	}
	return tx.Model(&models.PlayerLoadout{}).Where("user_id = ? AND (secure_container_id = ? OR secure_container_id IS NULL)", userID, "").
		Update("secure_container_id", defaultSecureContainerID).Error
}
