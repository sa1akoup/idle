package service

// 仓库与交易服务：统一处理商品识别、容量计算、现金扣减和装备库存变更。
// 同一物品按来源拆分：局内带出（raidExtract=true，可出售）与市场购买（false）。

import (
	"errors"
	"fmt"
	"time"

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
	RoundsPerSlot    int
	AmmoLevel        int
}

// PurchaseItem 从商人购买指定数量的商品（不校验商人归属，供测试/内部使用）。
func PurchaseItem(db *gorm.DB, itemID string, quantity int) error {
	return PurchaseItemForUser(db, models.DefaultUserID, itemID, quantity)
}

// PurchaseItemForUser 为指定用户购买商品，供内部流程使用。
func PurchaseItemForUser(db *gorm.DB, userID uint, itemID string, quantity int) error {
	if quantity <= 0 || quantity > 999 {
		return fmt.Errorf("购买数量需为 1-999")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		item, err := findCatalogItem(tx, itemID)
		if err != nil {
			return err
		}
		items := make([]catalogItem, quantity)
		for i := range items {
			items[i] = item
		}
		_, err = purchaseCatalogItems(tx, userID, items)
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

	var ammo models.AmmoDef
	if err := db.First(&ammo, "id = ?", itemID).Error; err == nil {
		return catalogItem{
			ID: ammo.ID, Name: ammo.Name, Kind: "ammo", Price: ammo.Price, Slots: 1,
			MerchantCategory: ammo.MerchantCategory, RepRequirement: ammo.RepRequirement,
			RoundsPerSlot: ammo.RoundsPerSlot, AmmoLevel: ammo.Level,
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogItem{}, fmt.Errorf("读取弹药商品: %w", err)
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

func purchaseCatalogItems(tx *gorm.DB, userID uint, items []catalogItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
		return 0, err
	}

	used, err := inventoryUsage(tx, userID)
	if err != nil {
		return 0, err
	}
	additionalCapacity, err := purchaseCapacityDelta(tx, userID, items)
	if err != nil {
		return 0, err
	}
	capacity, err := storageCapacityForUser(tx, userID)
	if err != nil {
		return 0, err
	}
	if used+additionalCapacity > capacity {
		return 0, fmt.Errorf("%w：仓库空间不足，还需 %d 个空位", ErrPurchaseUnavailable, used+additionalCapacity-capacity)
	}

	totalPrice := 0
	for _, item := range items {
		paid := item.PaidPrice
		if paid <= 0 {
			paid = item.Price
		}
		totalPrice += paid
	}
	if err := deductCash(tx, userID, totalPrice); err != nil {
		return 0, err
	}

	// 弹药单次可购买 999 发，同类商品聚合后只执行一次库存更新。
	quantities := make(map[string]int, len(items))
	definitions := make(map[string]catalogItem, len(items))
	for _, item := range items {
		quantities[item.ID]++
		definitions[item.ID] = item
	}
	for itemID, quantity := range quantities {
		item := definitions[itemID]
		if err := addInventoryItem(tx, userID, item, quantity, false); err != nil {
			return 0, err
		}
		if item.Kind == "armor" {
			for index := 0; index < quantity; index++ {
				instance := models.ArmorInstance{
					UserID:  userID,
					ArmorID: item.ID, MaxDurability: item.ArmorMax,
					CurDurability: item.ArmorMax, Status: "normal",
				}
				if err := tx.Create(&instance).Error; err != nil {
					return 0, fmt.Errorf("创建护甲实例: %w", err)
				}
			}
		}
	}
	return totalPrice, nil
}

func purchaseCapacityDelta(tx *gorm.DB, userID uint, items []catalogItem) (int, error) {
	additional := 0
	ammoAdds := make(map[string]int)
	ammoDefs := make(map[string]catalogItem)
	for _, item := range items {
		if item.Kind != "ammo" {
			additional++
			continue
		}
		ammoAdds[item.ID]++
		ammoDefs[item.ID] = item
	}
	for itemID, rounds := range ammoAdds {
		perSlot := ammoDefs[itemID].RoundsPerSlot
		if perSlot <= 0 {
			return 0, fmt.Errorf("弹药 %s 的每格容量无效", itemID)
		}
		var existing int
		if err := tx.Model(&models.Inventory{}).
			Where("user_id = ? AND item_id = ?", userID, itemID).
			Select("COALESCE(SUM(quantity), 0)").Scan(&existing).Error; err != nil {
			return 0, fmt.Errorf("读取弹药库存 %s: %w", itemID, err)
		}
		additional += ceilDiv(existing+rounds, perSlot) - ceilDiv(existing, perSlot)
	}
	return additional, nil
}

// GetStorageCapacity 返回仓库容量与扣除装备配置后的实际占用。
func GetStorageCapacity(db *gorm.DB) (*StorageCapacity, error) {
	return GetStorageCapacityForUser(db, models.DefaultUserID)
}

// GetStorageCapacityForUser 返回指定用户仓库容量与扣除装备配置后的实际占用。
func GetStorageCapacityForUser(db *gorm.DB, userID uint) (*StorageCapacity, error) {
	if err := settleDueHideoutJobsForUser(db, userID); err != nil {
		return nil, err
	}
	used, err := inventoryUsage(db, userID)
	if err != nil {
		return nil, err
	}
	capacity, err := storageCapacityForUser(db, userID)
	if err != nil {
		return nil, err
	}
	return &StorageCapacity{Capacity: capacity, Used: used}, nil
}

// addInventoryItem 按 (itemID, raidExtract) 新增或累加库存。
func addInventoryItem(tx *gorm.DB, userID uint, item catalogItem, quantity int, raidExtract bool) error {
	var useDef models.ItemUseDef
	if err := tx.Where("item_id = ?", item.ID).First(&useDef).Error; err == nil && useDef.InstanceRequired {
		maxDurability := useDef.MaxDurability
		if maxDurability <= 0 {
			maxDurability = 100
		}
		for i := 0; i < quantity; i++ {
			if err := tx.Create(&models.ItemInstance{
				UserID: userID, ItemID: item.ID, CurrentDurability: maxDurability, MaxDurability: maxDurability,
				Status: "normal", LocationType: "inventory", RaidExtract: raidExtract,
			}).Error; err != nil {
				return fmt.Errorf("新增物品实例 %s: %w", item.Name, err)
			}
		}
		return nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("读取物品效果 %s: %w", item.ID, err)
	}
	var inventory models.Inventory
	err := tx.Where("user_id = ? AND item_id = ? AND raid_extract = ?", userID, item.ID, raidExtract).First(&inventory).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		inventory = models.Inventory{
			UserID: userID,
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
func removeInventoryItem(tx *gorm.DB, userID uint, itemID string, quantity int) error {
	return removeInventoryItemFromSource(tx, userID, itemID, quantity, nil)
}

func removeInventoryItemFromSource(tx *gorm.DB, userID uint, itemID string, quantity int, source *bool) error {
	sources := []bool{true, false}
	if source != nil {
		sources = []bool{*source}
	}
	for _, raid := range sources {
		if quantity <= 0 {
			break
		}
		var inv models.Inventory
		if err := tx.Where("user_id = ? AND item_id = ? AND raid_extract = ? AND quantity > 0", userID, itemID, raid).First(&inv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("读取仓库物品 %s: %w", itemID, err)
		}
		deduct := inv.Quantity
		if deduct > quantity {
			deduct = quantity
		}
		result := tx.Model(&models.Inventory{}).
			Where("user_id = ? AND id = ? AND quantity >= ?", userID, inv.ID, deduct).
			Update("quantity", gorm.Expr("quantity - ?", deduct))
		if result.Error != nil {
			return fmt.Errorf("扣除仓库物品 %s: %w", itemID, result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("仓库中的 %s 数量不足", itemID)
		}
		if err := tx.Where("user_id = ? AND id = ? AND quantity <= 0", userID, inv.ID).Delete(&models.Inventory{}).Error; err != nil {
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
func deductCash(tx *gorm.DB, userID uint, amount int) error {
	result := tx.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, "cash", amount).
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
func addCash(tx *gorm.DB, userID uint, amount int) error {
	result := tx.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, "cash").
		Update("quantity", gorm.Expr("quantity + ?", amount))
	if result.Error != nil {
		return fmt.Errorf("增加现金: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("现金账户不存在")
	}
	return nil
}
