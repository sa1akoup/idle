// 账号初始化：为新用户创建角色、装备配置、初始库存、护甲实例和商人状态。
package config

import (
	"gorm.io/gorm"
)

// ProvisionUser 为新注册用户创建可直接进入游戏的初始数据。
func ProvisionUser(db *gorm.DB, userID uint) error {
	if err := seedPlayerForUser(db, userID); err != nil {
		return err
	}
	if err := seedLoadoutForUser(db, userID); err != nil {
		return err
	}
	inventories := append(initialEquipmentInventory(), initialConsumableInventory()...)
	inventories = append(inventories, initialAmmoInventory()...)
	inventories = append(inventories, initialMaterialInventory()...)
	for _, inventory := range inventories {
		if err := upsertInventoryForUser(db, userID, inventory); err != nil {
			return err
		}
	}
	if err := seedSurvivalForUser(db, userID); err != nil {
		return err
	}
	if err := seedMerchantStatesForUser(db, userID); err != nil {
		return err
	}
	return seedHideoutForUser(db, userID)
}
