// 启动期存量数据适配：按版本对历史用户执行一次耐久实例回填与目录引用清理。
// 适配完成后写入版本记录，后续启动跳过全量扫描；新用户仍由注册/Seed 路径直接初始化。
package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"idle/internal/config"
	"idle/internal/models"

	"gorm.io/gorm"
)

const currentUserDataUpgradeVersion = 1

type userDataUpgradeRecord struct {
	Version          int       `gorm:"primaryKey;column:version"`
	CompletedAt      time.Time `gorm:"column:completed_at"`
	ProcessedUsers   int       `gorm:"column:processed_users"`
	CreatedInstances int       `gorm:"column:created_instances"`
	StrippedRefs     int       `gorm:"column:stripped_refs"`
}

func (userDataUpgradeRecord) TableName() string {
	return "user_data_migrations"
}

// RunUserDataUpgrades 对当前版本的存量数据适配只执行一次，返回首次处理的用户数。
func RunUserDataUpgrades(db *gorm.DB) (int, error) {
	var record userDataUpgradeRecord
	if err := db.Where("version = ?", currentUserDataUpgradeVersion).First(&record).Error; err == nil {
		return record.ProcessedUsers, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("读取用户数据适配记录: %w", err)
	}

	userIDs, err := listUpgradableUserIDs(db)
	if err != nil {
		return 0, err
	}
	if len(userIDs) == 0 {
		return 0, nil
	}
	// 目录种子不完整时绝对不能进入悬空清扫：会把正常资产误判删除。
	// 拒绝执行并提示先补种，是当前部署形态下的安全边界。
	if err := requireSeededCatalogForUpgrade(db); err != nil {
		return 0, err
	}
	catalog, err := loadUpgradeCatalog(db)
	if err != nil {
		return 0, err
	}

	createdInstances := 0
	strippedRefs := 0
	for _, userID := range userIDs {
		txErr := db.Transaction(func(tx *gorm.DB) error {
			before, err := countUserItemInstancesTx(tx, userID)
			if err != nil {
				return err
			}
			if err := config.MigrateUserSurvivalData(tx, userID); err != nil {
				return fmt.Errorf("回填耐久实例: %w", err)
			}
			after, err := countUserItemInstancesTx(tx, userID)
			if err != nil {
				return err
			}
			createdInstances += int(after - before)

			stripped, err := stripDanglingCatalogRefsTx(tx, userID, catalog)
			strippedRefs += stripped
			return err
		})
		if txErr != nil {
			return len(userIDs), fmt.Errorf("用户 %d 存量数据适配失败: %w", userID, txErr)
		}
	}
	if err := db.Create(&userDataUpgradeRecord{
		Version:          currentUserDataUpgradeVersion,
		CompletedAt:      time.Now(),
		ProcessedUsers:   len(userIDs),
		CreatedInstances: createdInstances,
		StrippedRefs:     strippedRefs,
	}).Error; err != nil {
		return len(userIDs), fmt.Errorf("记录用户数据适配版本: %w", err)
	}
	log.Printf("[数据适配] 已处理 %d 个用户：换算耐久实例 %d 件，摘除悬空引用 %d 处", len(userIDs), createdInstances, strippedRefs)
	return len(userIDs), nil
}

// listUpgradableUserIDs 汇总出现过的用户：以 characters 为准（兼容认证改造前的纯本地存档），并并入 users 表。
func listUpgradableUserIDs(db *gorm.DB) ([]uint, error) {
	seen := make(map[uint]struct{})
	result := make([]uint, 0, 8)
	appendID := func(id uint) {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	var characterUsers []uint
	if err := db.Model(&models.Character{}).Where("1 = 1").Distinct().Order("user_id asc").Pluck("user_id", &characterUsers).Error; err != nil {
		return nil, fmt.Errorf("枚举角色用户: %w", err)
	}
	for _, id := range characterUsers {
		appendID(id)
	}
	var registered []uint
	if err := db.Model(&models.User{}).Order("id asc").Pluck("id", &registered).Error; err != nil {
		return nil, fmt.Errorf("枚举注册用户: %w", err)
	}
	for _, id := range registered {
		appendID(id)
	}
	return result, nil
}

func countUserItemInstancesTx(tx *gorm.DB, userID uint) (int64, error) {
	var count int64
	if err := tx.Model(&models.ItemInstance{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计物品实例: %w", err)
	}
	return count, nil
}

// upgradeCatalog 是“目录中仍然存在的物品 ID”集合。
type upgradeCatalog struct {
	ids map[string]struct{}
}

// upgradeCatalogTables 是参与“悬空物品判定”的全部目录定义表；
// 种子完整性校验与 ID 收集必须使用同一份清单，防止两处漂移。
var upgradeCatalogTables = []string{
	"weapon_defs", "armor_defs", "ammo_defs", "consumable_defs",
	"chest_rig_defs", "backpack_defs", "helmet_defs", "headset_defs",
	"loot_item_defs", "item_use_defs",
}

// requireSeededCatalogForUpgrade 校验参与目录判定的定义表均已有种子数据；
// 任一为空都意味着目录尚未初始化或种子流程被中断，此时摘除判定不可信。
func requireSeededCatalogForUpgrade(db *gorm.DB) error {
	var empty []string
	for _, table := range upgradeCatalogTables {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			return fmt.Errorf("检查目录表 %s: %w", table, err)
		}
		if count == 0 {
			empty = append(empty, table)
		}
	}
	if len(empty) > 0 {
		return fmt.Errorf(
			"目录种子未初始化或未完成（空表：%s）。为防止玩家资产被误删，已中止存量数据适配；"+
				"请先执行 go run . seed 完成目录初始化后再重启服务", strings.Join(empty, ", "))
	}
	return nil
}

func (c upgradeCatalog) has(itemID string) bool {
	_, ok := c.ids[itemID]
	return ok
}

// loadUpgradeCatalog 收集全部目录定义表的 ID。口径与快照构建/商人目录一致：
// 装备、弹药、消耗品、掉落品和物品使用效果；事件、地图等非携行目录不参与判定。
func loadUpgradeCatalog(db *gorm.DB) (upgradeCatalog, error) {
	catalog := upgradeCatalog{ids: make(map[string]struct{}, 256)}
	for _, table := range upgradeCatalogTables {
		idColumn := "id"
		if table == "item_use_defs" {
			// 物品使用效果目录以物品 ID 作为主键。
			idColumn = "item_id"
		}
		var ids []string
		if err := db.Table(table).Pluck(idColumn, &ids).Error; err != nil {
			return catalog, fmt.Errorf("读取目录表 %s: %w", table, err)
		}
		for _, id := range ids {
			catalog.ids[id] = struct{}{}
		}
	}
	return catalog, nil
}

// stripDanglingCatalogRefsTx 移除该用户对“已从目录消失”物品的全部引用并逐条记录日志，
// 防止下架物品后角色页、开局、容量等接口级联报错锁死玩家。返回摘除数量。
func stripDanglingCatalogRefsTx(tx *gorm.DB, userID uint, catalog upgradeCatalog) (int, error) {
	stripped := 0

	validInstances, err := loadOwnedInstanceIDsTx(tx, userID)
	if err != nil {
		return 0, err
	}

	var loadout models.PlayerLoadout
	loadoutErr := tx.Where("user_id = ?", userID).First(&loadout).Error
	switch {
	case loadoutErr == nil || loadoutErr == gorm.ErrRecordNotFound:
		// 无装备配置的用户（尚未初始化）只做库存侧清理。
	default:
		return 0, fmt.Errorf("读取当前装备: %w", loadoutErr)
	}

	if loadoutErr == nil {
		type fieldChange struct {
			column string
			field  string
			value  *string
		}
		var changedFields []string
		gearFields := []fieldChange{
			{"weapon_id", "WeaponID", &loadout.WeaponID},
			{"armor_id", "ArmorID", &loadout.ArmorID},
			{"chest_rig_id", "ChestRigID", &loadout.ChestRigID},
			{"backpack_id", "BackpackID", &loadout.BackpackID},
			{"helmet_id", "HelmetID", &loadout.HelmetID},
			{"headset_id", "HeadsetID", &loadout.HeadsetID},
			{"preset_weapon_id", "PresetWeaponID", &loadout.PresetWeaponID},
			{"preset_armor_id", "PresetArmorID", &loadout.PresetArmorID},
			{"preset_chest_rig_id", "PresetChestRigID", &loadout.PresetChestRigID},
			{"preset_backpack_id", "PresetBackpackID", &loadout.PresetBackpackID},
			{"preset_helmet_id", "PresetHelmetID", &loadout.PresetHelmetID},
			{"preset_headset_id", "PresetHeadsetID", &loadout.PresetHeadsetID},
			{"preset2_weapon_id", "Preset2WeaponID", &loadout.Preset2WeaponID},
			{"preset2_armor_id", "Preset2ArmorID", &loadout.Preset2ArmorID},
			{"preset2_chest_rig_id", "Preset2ChestRigID", &loadout.Preset2ChestRigID},
			{"preset2_backpack_id", "Preset2BackpackID", &loadout.Preset2BackpackID},
			{"preset2_helmet_id", "Preset2HelmetID", &loadout.Preset2HelmetID},
			{"preset2_headset_id", "Preset2HeadsetID", &loadout.Preset2HeadsetID},
			{"preset3_weapon_id", "Preset3WeaponID", &loadout.Preset3WeaponID},
			{"preset3_armor_id", "Preset3ArmorID", &loadout.Preset3ArmorID},
			{"preset3_chest_rig_id", "Preset3ChestRigID", &loadout.Preset3ChestRigID},
			{"preset3_backpack_id", "Preset3BackpackID", &loadout.Preset3BackpackID},
			{"preset3_helmet_id", "Preset3HelmetID", &loadout.Preset3HelmetID},
			{"preset3_headset_id", "Preset3HeadsetID", &loadout.Preset3HeadsetID},
			{"preset_ammo_id", "PresetAmmoID", &loadout.PresetAmmoID},
			{"preset2_ammo_id", "Preset2AmmoID", &loadout.Preset2AmmoID},
			{"preset3_ammo_id", "Preset3AmmoID", &loadout.Preset3AmmoID},
		}
		for _, change := range gearFields {
			if *change.value == "" || catalog.has(*change.value) {
				continue
			}
			log.Printf("[数据适配] 用户%d 装备配置摘除悬空物品 %s（%s）", userID, *change.value, change.column)
			*change.value = ""
			changedFields = append(changedFields, change.field)
			stripped++
			if change.column == "armor_id" {
				loadout.ArmorInstanceID = 0
				changedFields = append(changedFields, "ArmorInstanceID")
			}
		}

		filterConsumables := func(column, field string, values []string) {
			kept := make([]string, 0, len(values))
			for _, id := range values {
				if id != "" && !catalog.has(id) {
					log.Printf("[数据适配] 用户%d 补给清单摘除悬空物品 %s（%s）", userID, id, column)
					stripped++
					continue
				}
				kept = append(kept, id)
			}
			if len(kept) != len(values) {
				switch column {
				case "consumables":
					loadout.Consumables = kept
				case "preset_consumables":
					loadout.PresetConsumables = kept
				case "preset2_consumables":
					loadout.Preset2Consumables = kept
				case "preset3_consumables":
					loadout.Preset3Consumables = kept
				}
				changedFields = append(changedFields, field)
			}
		}
		filterConsumables("consumables", "Consumables", loadout.Consumables)
		filterConsumables("preset_consumables", "PresetConsumables", loadout.PresetConsumables)
		filterConsumables("preset2_consumables", "Preset2Consumables", loadout.Preset2Consumables)
		filterConsumables("preset3_consumables", "Preset3Consumables", loadout.Preset3Consumables)

		filterRefs := func(column, field string, refs []models.LoadoutItemRef) {
			kept := make([]models.LoadoutItemRef, 0, len(refs))
			for _, ref := range refs {
				switch {
				case ref.ItemID == "":
					continue
				case !catalog.has(ref.ItemID):
					log.Printf("[数据适配] 用户%d 补给引用摘除悬空物品 %s（%s）", userID, ref.ItemID, column)
					stripped++
				case ref.InstanceID > 0 && !validInstances[ref.InstanceID]:
					log.Printf("[数据适配] 用户%d 补给引用摘除丢失实例 %s#%d（%s）", userID, ref.ItemID, ref.InstanceID, column)
					stripped++
				default:
					kept = append(kept, ref)
				}
			}
			if len(kept) != len(refs) {
				switch column {
				case "consumable_refs":
					loadout.ConsumableRefs = kept
				case "preset_consumable_refs":
					loadout.PresetConsumableRefs = kept
				case "preset2_consumable_refs":
					loadout.Preset2ConsumableRefs = kept
				case "preset3_consumable_refs":
					loadout.Preset3ConsumableRefs = kept
				}
				changedFields = append(changedFields, field)
			}
		}
		filterRefs("consumable_refs", "ConsumableRefs", loadout.ConsumableRefs)
		filterRefs("preset_consumable_refs", "PresetConsumableRefs", loadout.PresetConsumableRefs)
		filterRefs("preset2_consumable_refs", "Preset2ConsumableRefs", loadout.Preset2ConsumableRefs)
		filterRefs("preset3_consumable_refs", "Preset3ConsumableRefs", loadout.Preset3ConsumableRefs)

		if len(changedFields) > 0 {
			if err := tx.Model(&models.PlayerLoadout{}).
				Where("user_id = ? AND id = ?", userID, loadout.ID).
				Select(changedFields).Updates(&loadout).Error; err != nil {
				return stripped, fmt.Errorf("保存清理后的装备配置: %w", err)
			}
		}
	}

	if err := sweepDanglingRows(tx, userID, catalog, "inventories", &models.Inventory{}, &stripped); err != nil {
		return stripped, err
	}
	if err := sweepDanglingRows(tx, userID, catalog, "item_instances", &models.ItemInstance{}, &stripped); err != nil {
		return stripped, err
	}
	if err := sweepDanglingArmorsTx(tx, userID, catalog, &stripped); err != nil {
		return stripped, err
	}
	return stripped, nil
}

func loadOwnedInstanceIDsTx(tx *gorm.DB, userID uint) (map[uint]bool, error) {
	type row struct {
		ID uint `gorm:"column:id"`
	}
	var rows []row
	if err := tx.Model(&models.ItemInstance{}).Where("user_id = ?", userID).Pluck("id", &rows).Error; err != nil {
		return nil, fmt.Errorf("读取用户物品实例: %w", err)
	}
	result := make(map[uint]bool, len(rows))
	for _, r := range rows {
		result[r.ID] = true
	}
	return result, nil
}

// sweepDanglingRows 删除表中属于该用户、但物品 ID 已不在目录里的行。
// 货币行（cash）不属于目录商品，永远保留。
func sweepDanglingRows(tx *gorm.DB, userID uint, catalog upgradeCatalog, table string, model interface{}, stripped *int) error {
	selectColumns := "id, item_id, '' AS kind"
	if table == "inventories" {
		// 只有仓库表带 kind 列，用于识别货币行。
		selectColumns = "id, item_id, kind"
	}
	var rows []struct {
		ID     uint   `gorm:"column:id"`
		ItemID string `gorm:"column:item_id"`
		Kind   string `gorm:"column:kind"`
	}
	if err := tx.Table(table).Select(selectColumns).Where("user_id = ?", userID).Scan(&rows).Error; err != nil {
		return fmt.Errorf("扫描 %s: %w", table, err)
	}
	for _, r := range rows {
		if catalog.has(r.ItemID) || r.ItemID == "cash" || r.Kind == "currency" {
			continue
		}
		if err := tx.Where("id = ?", r.ID).Delete(model).Error; err != nil {
			return fmt.Errorf("删除 %s 悬空行 #%d: %w", table, r.ID, err)
		}
		log.Printf("[数据适配] 用户%d 删除悬空库存行 %s#%d（%s）", userID, r.ItemID, r.ID, table)
		*stripped++
	}
	return nil
}

// sweepDanglingArmorsTx 清理护甲定义已不存在的护甲实例。
func sweepDanglingArmorsTx(tx *gorm.DB, userID uint, catalog upgradeCatalog, stripped *int) error {
	var armors []models.ArmorInstance
	if err := tx.Where("user_id = ?", userID).Find(&armors).Error; err != nil {
		return fmt.Errorf("扫描护甲实例: %w", err)
	}
	for _, armor := range armors {
		if catalog.has(armor.ArmorID) {
			continue
		}
		if err := tx.Delete(&armor).Error; err != nil {
			return fmt.Errorf("删除悬空护甲实例 #%d: %w", armor.ID, err)
		}
		log.Printf("[数据适配] 用户%d 删除悬空护甲实例 %s#%d", userID, armor.ArmorID, armor.ID)
		*stripped++
	}
	return nil
}
