package catalog

import (
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

// ListByMerchantCategory 返回指定商人类别下的全部目录商品。
// 弹药高等级过滤仍由服务层按商人规则决定，避免仓储混入交易策略。
func (r *Repository) ListByMerchantCategory(category string) ([]Item, error) {
	return r.listCatalogItems(category)
}

// ListAll 返回全部目录商品，供黑市等跨类货架使用。
func (r *Repository) ListAll() ([]Item, error) {
	return r.listCatalogItems("")
}

func applyMerchantCategory(db *gorm.DB, category string) *gorm.DB {
	if category == "" {
		return db
	}
	return db.Where("merchant_category = ?", category)
}

func (r *Repository) listCatalogItems(category string) ([]Item, error) {
	items := make([]Item, 0)

	var weapons []models.WeaponDef
	if err := applyMerchantCategory(r.db, category).Find(&weapons).Error; err != nil {
		return nil, fmt.Errorf("读取武器目录: %w", err)
	}
	for _, weapon := range weapons {
		items = append(items, Item{
			ID: weapon.ID, Name: weapon.Name, Kind: "weapon", Price: weapon.Price, Weight: weapon.Weight, Slots: weapon.Slots,
			MerchantCategory: weapon.MerchantCategory, RepRequirement: weapon.RepRequirement,
			CaliberID: weapon.CaliberID, AmmoPerRound: weapon.AmmoPerRound, Damage: weapon.Damage, Penetration: weapon.Penetration,
		})
	}

	var ammos []models.AmmoDef
	if err := applyMerchantCategory(r.db, category).Order("caliber_id asc, level asc").Find(&ammos).Error; err != nil {
		return nil, fmt.Errorf("读取弹药目录: %w", err)
	}
	for _, ammo := range ammos {
		items = append(items, Item{
			ID: ammo.ID, Name: ammo.Name, Kind: "ammo", Price: ammo.Price, Slots: 1,
			MerchantCategory: ammo.MerchantCategory, RepRequirement: ammo.RepRequirement,
			RoundsPerSlot: ammo.RoundsPerSlot, AmmoLevel: ammo.Level, CaliberID: ammo.CaliberID,
		})
	}

	var armors []models.ArmorDef
	if err := applyMerchantCategory(r.db, category).Find(&armors).Error; err != nil {
		return nil, fmt.Errorf("读取护甲目录: %w", err)
	}
	for _, armor := range armors {
		items = append(items, Item{
			ID: armor.ID, Name: armor.Name, Kind: "armor", Price: armor.Price, Weight: armor.Weight, Slots: armor.Slots,
			MerchantCategory: armor.MerchantCategory, RepRequirement: armor.RepRequirement,
			ArmorMax: armor.MaxDurability, ProtectionLevel: armor.ProtectionLevel, Coverage: armor.Coverage,
		})
	}

	var consumables []models.ConsumableDef
	if err := applyMerchantCategory(r.db, category).Find(&consumables).Error; err != nil {
		return nil, fmt.Errorf("读取补给目录: %w", err)
	}
	for _, consumable := range consumables {
		items = append(items, Item{
			ID: consumable.ID, Name: consumable.Name, Kind: "consumable", Desc: consumable.Desc, Price: consumable.Price,
			Weight: consumable.Weight, Slots: consumable.Slots, MerchantCategory: consumable.MerchantCategory, RepRequirement: consumable.RepRequirement,
		})
	}

	var lootItems []models.LootItemDef
	if err := applyMerchantCategory(r.db, category).Order("id asc").Find(&lootItems).Error; err != nil {
		return nil, fmt.Errorf("读取战利品目录: %w", err)
	}
	for _, loot := range lootItems {
		items = append(items, Item{
			ID: loot.ID, Name: loot.Name, Kind: "loot", Category: loot.Category, Desc: loot.Desc, Price: loot.Price,
			Weight: loot.Weight, Slots: loot.Slots, DropWeight: loot.DropWeight,
			MerchantCategory: loot.MerchantCategory, RepRequirement: loot.RepRequirement,
		})
	}

	var chestRigs []models.ChestRigDef
	if err := applyMerchantCategory(r.db, category).Find(&chestRigs).Error; err != nil {
		return nil, fmt.Errorf("读取胸挂目录: %w", err)
	}
	for _, chestRig := range chestRigs {
		items = append(items, Item{
			ID: chestRig.ID, Name: chestRig.Name, Kind: "chestrig", Price: chestRig.Price, Weight: chestRig.Weight, Slots: chestRig.Slots,
			MerchantCategory: chestRig.MerchantCategory, RepRequirement: chestRig.RepRequirement,
			AddSlots: chestRig.AddSlots, AddWeight: chestRig.AddWeight,
		})
	}

	var backpacks []models.BackpackDef
	if err := applyMerchantCategory(r.db, category).Find(&backpacks).Error; err != nil {
		return nil, fmt.Errorf("读取背包目录: %w", err)
	}
	for _, backpack := range backpacks {
		items = append(items, Item{
			ID: backpack.ID, Name: backpack.Name, Kind: "backpack", Price: backpack.Price, Weight: backpack.Weight, Slots: backpack.Slots,
			MerchantCategory: backpack.MerchantCategory, RepRequirement: backpack.RepRequirement,
			AddSlots: backpack.AddSlots, AddWeight: backpack.AddWeight,
		})
	}

	var helmets []models.HelmetDef
	if err := applyMerchantCategory(r.db, category).Find(&helmets).Error; err != nil {
		return nil, fmt.Errorf("读取头盔目录: %w", err)
	}
	for _, helmet := range helmets {
		items = append(items, Item{
			ID: helmet.ID, Name: helmet.Name, Kind: "helmet", Price: helmet.Price, Weight: helmet.Weight, Slots: helmet.Slots,
			MerchantCategory: helmet.MerchantCategory, RepRequirement: helmet.RepRequirement,
			Protect: helmet.Protect, Coverage: helmet.Coverage, ArmorMax: helmet.MaxDurability,
		})
	}

	var headsets []models.HeadsetDef
	if err := applyMerchantCategory(r.db, category).Find(&headsets).Error; err != nil {
		return nil, fmt.Errorf("读取耳机目录: %w", err)
	}
	for _, headset := range headsets {
		items = append(items, Item{
			ID: headset.ID, Name: headset.Name, Kind: "headset", Price: headset.Price, Weight: headset.Weight, Slots: headset.Slots,
			MerchantCategory: headset.MerchantCategory, RepRequirement: headset.RepRequirement, HearingLevel: headset.HearingLevel,
		})
	}

	var keyCases []models.KeyCaseDef
	if r.db.Migrator().HasTable(&models.KeyCaseDef{}) {
		if err := applyMerchantCategory(r.db, category).Find(&keyCases).Error; err != nil {
			return nil, fmt.Errorf("读取钥匙包目录: %w", err)
		}
	}
	for _, keyCase := range keyCases {
		items = append(items, Item{
			ID: keyCase.ID, Name: keyCase.Name, Kind: "keycase", Price: keyCase.Price, Weight: keyCase.Weight, Slots: keyCase.Slots,
			MerchantCategory: keyCase.MerchantCategory, RepRequirement: keyCase.RepRequirement, AddSlots: keyCase.KeySlots,
		})
	}

	var secureContainers []models.SecureContainerDef
	if r.db.Migrator().HasTable(&models.SecureContainerDef{}) {
		if err := applyMerchantCategory(r.db, category).Find(&secureContainers).Error; err != nil {
			return nil, fmt.Errorf("读取安全箱目录: %w", err)
		}
	}
	for _, container := range secureContainers {
		items = append(items, Item{
			ID: container.ID, Name: container.Name, Kind: "secure", Price: container.Price, Weight: container.Weight, Slots: container.Slots,
			MerchantCategory: container.MerchantCategory, RepRequirement: container.RepRequirement, AddSlots: container.InnerSlots,
		})
	}
	return items, nil
}
