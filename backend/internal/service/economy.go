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

// claimEconomicOperation 认领幂等操作：有键则加载既有结果（重放返回 true），无键则创建新记录；返回 false 表示首次执行。
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
	// 幂等键唯一约束兜底：并发下插入冲突则回读对方已写入的记录，按既有结果重放。
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
