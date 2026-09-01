// 活跃行动资源保护：在同一数据库事务内锁定 Session 状态，防止装备、携带物和护甲被并发修改。
package service

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"idle/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrActiveSessionResourceLocked = errors.New("活跃行动正在使用该资源")

// lockUserResourcesTx 通过更新用户唯一的角色行获得数据库级资源锁。
// 该锁必须在所有会修改 Session、库存、现金、装备或护甲的事务中最先获取。
func lockUserResourcesTx(tx *gorm.DB, userID uint) error {
	result := tx.Model(&models.Character{}).
		Where("user_id = ?", userID).
		UpdateColumn("resource_version", gorm.Expr("resource_version + ?", 1))
	if result.Error != nil {
		return fmt.Errorf("锁定用户资源: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("锁定用户资源失败：角色不存在")
	}
	return nil
}

// activeSessionsForUser 查询（行锁保护的）活跃运行中会话，供各类资源占用判定使用。
func activeSessionsForUser(tx *gorm.DB, userID uint) ([]models.Session, error) {
	var sessions []models.Session
	// 使用 FOR UPDATE 行锁锁定活跃会话行，防止结算或资源操作并发读到过期状态。
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status = ?", userID, "running")
	if err := query.Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("读取活跃行动资源: %w", err)
	}
	return sessions, nil
}

// ensureLoadoutMutationAllowed 有活跃会话时禁止修改当前装备，防止开局后换装造成快照与背包分裂。
func ensureLoadoutMutationAllowed(tx *gorm.DB, userID uint) error {
	sessions, err := activeSessionsForUser(tx, userID)
	if err != nil {
		return err
	}
	if len(sessions) > 0 {
		return fmt.Errorf("%w：行动结束前不能修改当前装备", ErrActiveSessionResourceLocked)
	}
	return nil
}

// ensureItemNotInActiveSession 检查物品是否被当前配装或任意活跃会话携带，被占用则拒绝出售/转移等操作。
func ensureItemNotInActiveSession(tx *gorm.DB, userID uint, itemID string) error {
	sessions, err := activeSessionsForUser(tx, userID)
	if err != nil {
		return err
	}
	protected := make(map[string]struct{})
	var loadout models.PlayerLoadout
	if err := tx.Where("user_id = ?", userID).First(&loadout).Error; err == nil {
		for _, id := range []string{loadout.WeaponID, loadout.ArmorID, loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID} {
			if id != "" {
				protected[id] = struct{}{}
			}
		}
		for _, id := range loadout.Consumables {
			if id != "" {
				protected[id] = struct{}{}
			}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("读取当前装备: %w", err)
	}
	for _, session := range sessions {
		for _, id := range []string{session.WeaponID, session.ArmorID, session.AmmoID} {
			if id != "" {
				protected[id] = struct{}{}
			}
		}
		consumables, err := splitSessionConsumables(session.Consumables)
		if err != nil {
			return fmt.Errorf("解析行动携带补给: %w", err)
		}
		for _, id := range consumables {
			protected[id] = struct{}{}
		}
	}
	if _, ok := protected[itemID]; ok {
		return fmt.Errorf("%w：%s", ErrActiveSessionResourceLocked, itemID)
	}
	return nil
}

// ensureArmorRepairAllowed 有活跃会话正在使用该护甲时禁止维修。
func ensureArmorRepairAllowed(tx *gorm.DB, userID uint, armorID string) error {
	sessions, err := activeSessionsForUser(tx, userID)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.ArmorID == armorID {
			return fmt.Errorf("%w：护甲 %s 正在行动中", ErrActiveSessionResourceLocked, armorID)
		}
	}
	return nil
}

// splitSessionConsumables 将会话行的补给 CSV 字符串解析为物品 ID 列表，并兼容前导空格。
func splitSessionConsumables(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	parts, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("行动补给 CSV 格式无效: %w", err)
	}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result, nil
}
