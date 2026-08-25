// 经济操作幂等服务：为购买与出售提供按用户隔离的请求结果复用。
package service

import (
	"errors"
	"fmt"
	"strings"

	"idle/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func claimEconomicOperation(tx *gorm.DB, userID uint, operationKey, operationType string) (*models.EconomicOperation, bool, error) {
	operationKey = strings.TrimSpace(operationKey)
	if operationKey == "" {
		return nil, false, nil
	}
	var operation models.EconomicOperation
	err := tx.Where("user_id = ? AND operation_key = ?", userID, operationKey).First(&operation).Error
	switch {
	case err == nil:
		if operation.OperationType != operationType {
			return nil, false, fmt.Errorf("幂等键已用于其他经济操作")
		}
		return &operation, true, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, false, fmt.Errorf("读取经济操作: %w", err)
	}
	operation = models.EconomicOperation{
		UserID: userID, OperationKey: operationKey, OperationType: operationType, ResultJSON: "{}",
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&operation)
	if result.Error != nil {
		return nil, false, fmt.Errorf("创建经济操作: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		if err := tx.Where("user_id = ? AND operation_key = ?", userID, operationKey).First(&operation).Error; err != nil {
			return nil, false, fmt.Errorf("读取并发经济操作: %w", err)
		}
		if operation.OperationType != operationType {
			return nil, false, fmt.Errorf("幂等键已用于其他经济操作")
		}
		return &operation, true, nil
	}
	return &operation, false, nil
}
