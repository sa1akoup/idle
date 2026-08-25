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

func activeSessionsForUser(tx *gorm.DB, userID uint) ([]models.Session, error) {
	var sessions []models.Session
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status IN ?", userID, []string{"running", "waiting_injury"})
	if err := query.Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("读取活跃行动资源: %w", err)
	}
	return sessions, nil
}

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
		for _, id := range splitSessionConsumables(session.Consumables) {
			protected[id] = struct{}{}
		}
	}
	if _, ok := protected[itemID]; ok {
		return fmt.Errorf("%w：%s", ErrActiveSessionResourceLocked, itemID)
	}
	return nil
}

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

func splitSessionConsumables(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	parts, err := reader.Read()
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

// RepairArmorForUser 在事务内完成护甲状态读取、活跃行动检查和维修更新。
func RepairArmorForUser(db *gorm.DB, userID, armorInstanceID uint) (int, error) {
	newMax := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		var inst models.ArmorInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND id = ?", userID, armorInstanceID).First(&inst).Error; err != nil {
			return err
		}
		if err := ensureArmorRepairAllowed(tx, userID, inst.ArmorID); err != nil {
			return err
		}
		if inst.RepairCount >= 1 {
			return fmt.Errorf("已达维修上限，报废")
		}
		if inst.CurDurability > 0 {
			return fmt.Errorf("仅归零护甲可维修")
		}
		newMax = inst.MaxDurability / 2
		if err := tx.Model(&models.ArmorInstance{}).Where("user_id = ? AND id = ?", userID, inst.ID).Updates(map[string]interface{}{
			"max_durability": newMax, "cur_durability": newMax, "repair_count": inst.RepairCount + 1, "status": "normal",
		}).Error; err != nil {
			return fmt.Errorf("护甲维修失败: %w", err)
		}
		return nil
	})
	return newMax, err
}
