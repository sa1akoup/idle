package service

// 仓库与交易服务：统一处理商品识别、容量计算、现金扣减和装备库存变更。
// 同一物品按来源拆分：局内带出（raidExtract=true，可出售）与市场购买（false）。

import (
	"errors"
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

// ErrPurchaseUnavailable 表示现金或仓位不足等可向玩家展示的交易失败。
var ErrPurchaseUnavailable = errors.New("无法完成购买")

// StorageCapacity 仓库容量快照。
type StorageCapacity struct {
	Capacity int `json:"capacity"`
	Used     int `json:"used"`
}

type catalogItem struct {
	ID               string
	Name             string
	Kind             string
	Category         string
	Price            int // 基准价
	PaidPrice        int // 实际支付价（按商人好感度折算），0 时按基准价
	Weight           int
	Slots            int
	DropWeight       int
	MerchantCategory string
	RepRequirement   int
	ArmorMax         int
}

// PurchaseItem 从商人购买指定数量的商品（不校验商人归属，供测试/内部使用）。
func PurchaseItem(db *gorm.DB, itemID string, quantity int) error {
	if quantity <= 0 || quantity > 99 {
		return fmt.Errorf("购买数量需为 1-99")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		item, err := findCatalogItem(tx, itemID)
		if err != nil {
			return err
		}
		items := make([]catalogItem, quantity)
		for i := range items {
			items[i] = item
		}
		_, err = purchaseCatalogItems(tx, items)
		return err
	})
}

func findCatalogItem(db *gorm.DB, itemID string) (catalogItem, error) {
	var weapon models.WeaponDef
	if err := db.First(&weapon, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: weapon.ID, Name: weapon.Name, Kind: "weapon", Price: weapon.Price, Weight: weapon.Weight, Slots: weapon.Slots, MerchantCategory: weapon.MerchantCategory, RepRequirement: weapon.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取武器商品: %w", err)
	}

	var armor models.ArmorDef
	if err := db.First(&armor, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: armor.ID, Name: armor.Name, Kind: "armor", Price: armor.Price, Weight: armor.Weight, Slots: armor.Slots, MerchantCategory: armor.MerchantCategory, RepRequirement: armor.RepRequirement, ArmorMax: armor.MaxDurability}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取护甲商品: %w", err)
	}

	var consumable models.ConsumableDef
	if err := db.First(&consumable, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: consumable.ID, Name: consumable.Name, Kind: "consumable", Price: consumable.Price, Weight: consumable.Weight, Slots: consumable.Slots, MerchantCategory: consumable.MerchantCategory, RepRequirement: consumable.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取补给商品: %w", err)
	}

	var chestRig models.ChestRigDef
	if err := db.First(&chestRig, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: chestRig.ID, Name: chestRig.Name, Kind: "chestrig", Price: chestRig.Price, Weight: chestRig.Weight, Slots: chestRig.Slots, MerchantCategory: chestRig.MerchantCategory, RepRequirement: chestRig.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取胸挂商品: %w", err)
	}

	var backpack models.BackpackDef
	if err := db.First(&backpack, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: backpack.ID, Name: backpack.Name, Kind: "backpack", Price: backpack.Price, Weight: backpack.Weight, Slots: backpack.Slots, MerchantCategory: backpack.MerchantCategory, RepRequirement: backpack.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取背包商品: %w", err)
	}

	var helmet models.HelmetDef
	if err := db.First(&helmet, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: helmet.ID, Name: helmet.Name, Kind: "helmet", Price: helmet.Price, Weight: helmet.Weight, Slots: helmet.Slots, MerchantCategory: helmet.MerchantCategory, RepRequirement: helmet.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取头盔商品: %w", err)
	}

	var headset models.HeadsetDef
	if err := db.First(&headset, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: headset.ID, Name: headset.Name, Kind: "headset", Price: headset.Price, Weight: headset.Weight, Slots: headset.Slots, MerchantCategory: headset.MerchantCategory, RepRequirement: headset.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取耳机商品: %w", err)
	}

	var loot models.LootItemDef
	if err := db.First(&loot, "id = ?", itemID).Error; err == nil {
		return catalogItem{ID: loot.ID, Name: loot.Name, Kind: "loot", Category: loot.Category, Price: loot.Price, Weight: loot.Weight, Slots: loot.Slots, DropWeight: loot.DropWeight, MerchantCategory: loot.MerchantCategory, RepRequirement: loot.RepRequirement}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取战利品商品: %w", err)
	}

	return catalogItem{}, fmt.Errorf("商品不存在")
}

func purchaseCatalogItems(tx *gorm.DB, items []catalogItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}

	used, err := inventoryUsage(tx)
	if err != nil {
		return 0, err
	}
	if used+len(items) > models.InventoryCapacity {
		return 0, fmt.Errorf("%w：仓库空间不足，还需 %d 个空位", ErrPurchaseUnavailable, used+len(items)-models.InventoryCapacity)
	}

	totalPrice := 0
	for _, item := range items {
		paid := item.PaidPrice
		if paid <= 0 {
			paid = item.Price
		}
		totalPrice += paid
	}
	if err := deductCash(tx, totalPrice); err != nil {
		return 0, err
	}

	for _, item := range items {
		if err := addInventoryItem(tx, item, 1, false); err != nil {
			return 0, err
		}
		if item.Kind == "armor" {
			instance := models.ArmorInstance{
				ArmorID: item.ID, MaxDurability: item.ArmorMax,
				CurDurability: item.ArmorMax, Status: "normal",
			}
			if err := tx.Create(&instance).Error; err != nil {
				return 0, fmt.Errorf("创建护甲实例: %w", err)
			}
		}
	}
	return totalPrice, nil
}

// GetStorageCapacity 返回仓库容量与扣除装备配置后的实际占用。
func GetStorageCapacity(db *gorm.DB) (*StorageCapacity, error) {
	used, err := inventoryUsage(db)
	if err != nil {
		return nil, err
	}
	return &StorageCapacity{Capacity: models.InventoryCapacity, Used: used}, nil
}

// addInventoryItem 按 (itemID, raidExtract) 新增或累加库存。
func addInventoryItem(tx *gorm.DB, item catalogItem, quantity int, raidExtract bool) error {
	var inventory models.Inventory
	err := tx.Where("item_id = ? AND raid_extract = ?", item.ID, raidExtract).First(&inventory).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		inventory = models.Inventory{
			ItemID: item.ID, Name: item.Name, Kind: item.Kind, Category: item.Category,
			Quantity: quantity, Price: item.Price, Weight: item.Weight, Slots: item.Slots,
			RaidExtract: raidExtract, MerchantCategory: item.MerchantCategory, RepRequirement: item.RepRequirement,
		}
		if err := tx.Create(&inventory).Error; err != nil {
			return fmt.Errorf("新增仓库物品 %s: %w", item.Name, err)
		}
	case err != nil:
		return fmt.Errorf("读取仓库物品 %s: %w", item.Name, err)
	default:
		if err := tx.Model(&inventory).Updates(map[string]interface{}{
			"name": item.Name, "kind": item.Kind, "category": item.Category, "price": item.Price,
			"weight": item.Weight, "slots": item.Slots,
			"merchant_category": item.MerchantCategory, "rep_requirement": item.RepRequirement,
			"quantity": gorm.Expr("quantity + ?", quantity),
		}).Error; err != nil {
			return fmt.Errorf("更新仓库物品 %s: %w", item.Name, err)
		}
	}
	return nil
}

// removeInventoryItem 从该物品的总库存中扣除数量（优先扣局内带出），用于失能丢装。
func removeInventoryItem(tx *gorm.DB, itemID string, quantity int) error {
	for _, raid := range []bool{true, false} {
		if quantity <= 0 {
			break
		}
		var inv models.Inventory
		if err := tx.Where("item_id = ? AND raid_extract = ? AND quantity > 0", itemID, raid).First(&inv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("读取仓库物品 %s: %w", itemID, err)
		}
		deduct := inv.Quantity
		if deduct > quantity {
			deduct = quantity
		}
		if err := tx.Model(&models.Inventory{}).Where("id = ?", inv.ID).
			Update("quantity", gorm.Expr("quantity - ?", deduct)).Error; err != nil {
			return fmt.Errorf("扣除仓库物品 %s: %w", itemID, err)
		}
		if err := tx.Where("id = ? AND quantity <= 0", inv.ID).Delete(&models.Inventory{}).Error; err != nil {
			return fmt.Errorf("清理空库存 %s: %w", itemID, err)
		}
		quantity -= deduct
	}
	if quantity > 0 {
		return fmt.Errorf("仓库中的 %s 数量不足", itemID)
	}
	return nil
}

// removeSellableItem removed (selling no longer restricted to raid-extract)

// deductCash 扣减现金，不足则返回 ErrPurchaseUnavailable。
func deductCash(tx *gorm.DB, amount int) error {
	result := tx.Model(&models.Inventory{}).
		Where("item_id = ? AND quantity >= ?", "cash", amount).
		Update("quantity", gorm.Expr("quantity - ?", amount))
	if result.Error != nil {
		return fmt.Errorf("扣除现金: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w：现金不足，本次总价为 ￥%d", ErrPurchaseUnavailable, amount)
	}
	return nil
}

// addCash 增加现金。
func addCash(tx *gorm.DB, amount int) error {
	result := tx.Model(&models.Inventory{}).
		Where("item_id = ?", "cash").
		Update("quantity", gorm.Expr("quantity + ?", amount))
	if result.Error != nil {
		return fmt.Errorf("增加现金: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("现金账户不存在")
	}
	return nil
}

// inventoryUsage 计算仓库已占用容量，扣除当前装备与 3 套预设装备（含补给）的占用。
func inventoryUsage(db *gorm.DB) (int, error) {
	var rows []struct {
		ItemID   string
		Quantity int
	}
	if err := db.Model(&models.Inventory{}).
		Where("item_id <> ? AND quantity > 0", "cash").
		Select("item_id, quantity").Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("计算仓库容量: %w", err)
	}
	alloc, err := loadoutAllocatedItems(db)
	if err != nil {
		return 0, err
	}
	stock := make(map[string]int, len(rows))
	for _, row := range rows {
		stock[row.ItemID] += row.Quantity
	}
	used := 0
	for itemID, quantity := range stock {
		deduct := alloc[itemID]
		if deduct > quantity {
			deduct = quantity
		}
		used += quantity - deduct
	}
	return used, nil
}

// loadoutAllocatedItems 统计当前装备与 3 套预设清单中每件物品各占用的仓库单位。
func loadoutAllocatedItems(db *gorm.DB) (map[string]int, error) {
	loadout, err := GetPlayerLoadout(db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]int{}, nil
		}
		return nil, err
	}
	alloc := make(map[string]int)
	add := func(ids []string) {
		for _, id := range ids {
			if id != "" {
				alloc[id]++
			}
		}
	}
	add([]string{loadout.WeaponID, loadout.ArmorID, loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID})
	add(loadout.Consumables)
	for i := 1; i <= 3; i++ {
		weaponID, armorID, consumables := PresetOf(loadout, i)
		base := []string{weaponID, armorID}
		base = append(base, presetEquipOf(loadout, i)...)
		add(base)
		add(consumables)
	}
	return alloc, nil
}

// presetEquipOf 返回第 N 套（1-3）预设新增装备（胸挂/背包/头盔/耳机）清单。
func presetEquipOf(loadout *models.PlayerLoadout, index int) []string {
	switch index {
	case 2:
		return []string{loadout.Preset2ChestRigID, loadout.Preset2BackpackID, loadout.Preset2HelmetID, loadout.Preset2HeadsetID}
	case 3:
		return []string{loadout.Preset3ChestRigID, loadout.Preset3BackpackID, loadout.Preset3HelmetID, loadout.Preset3HeadsetID}
	default:
		return []string{loadout.PresetChestRigID, loadout.PresetBackpackID, loadout.PresetHelmetID, loadout.PresetHeadsetID}
	}
}

// PresetOf 返回第 N 套（1-3）预设装备清单，供补购与容量计算使用。
func PresetOf(loadout *models.PlayerLoadout, index int) (weaponID, armorID string, consumables []string) {
	switch index {
	case 2:
		return loadout.Preset2WeaponID, loadout.Preset2ArmorID, loadout.Preset2Consumables
	case 3:
		return loadout.Preset3WeaponID, loadout.Preset3ArmorID, loadout.Preset3Consumables
	default:
		return loadout.PresetWeaponID, loadout.PresetArmorID, loadout.PresetConsumables
	}
}

func uniqueItemIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
