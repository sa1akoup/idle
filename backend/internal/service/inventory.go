package service

// 仓库与交易服务：统一处理商品识别、容量计算、现金扣减和装备库存变更。
// 同一物品按来源拆分：局内带出（raidExtract=true，可出售）与市场购买（false）。

import (
	"errors"
	"fmt"
	"time"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// ErrPurchaseUnavailable 表示现金或仓位不足等可向玩家展示的交易失败。
var ErrPurchaseUnavailable = errors.New("无法完成购买")

// StorageCapacity 仓库容量快照。
type StorageCapacity struct {
	Capacity int `json:"capacity"`
	Used     int `json:"used"`
}

type catalogItem = catalog.Item

// PurchaseItemForUser 为指定用户购买商品，供内部流程使用。
func PurchaseItemForUser(db *gorm.DB, userID uint, itemID string, quantity int) error {
	if quantity <= 0 || quantity > 999 {
		return fmt.Errorf("购买数量需为 1-999")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		item, err := catalog.New(tx).FindByID(itemID)
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

// purchaseCatalogItems 批量执行商品购买：先结算到期的隐藏所产出，再校验容量与现金，最后写入库存并返回实付总额。
func purchaseCatalogItems(tx *gorm.DB, userID uint, items []catalogItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
		return 0, err
	}
	catalogRepo := catalog.New(tx)

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
	// 容量校验：已占用空间 + 本次新增占用不得超过仓库总容量。
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
		if err := addInventoryItemWithCatalog(tx, userID, item, quantity, false, catalogRepo); err != nil {
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

// purchaseCapacityDelta 计算本次购买占用的净新增仓库格数；非弹药每件占 1 格，弹药按每格可容发数向上取整折算。
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
		// 净增格数 = 购买后占用的格子数 − 当前占用的格子数，同类弹药聚合后只计算一次增量。
		additional += ceilDiv(existing+rounds, perSlot) - ceilDiv(existing, perSlot)
	}
	return additional, nil
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
	return addInventoryItemWithCatalog(tx, userID, item, quantity, raidExtract, catalog.New(tx))
}

// addInventoryItemWithCatalog 写入库存：需实例化的物品逐件创建独立实例，普通物品按 (itemID, raidExtract) 聚合行累加数量。
func addInventoryItemWithCatalog(tx *gorm.DB, userID uint, item catalogItem, quantity int, raidExtract bool, catalogRepo *catalog.Repository) error {
	var useDef models.ItemUseDef
	useDef, found, err := catalogRepo.FindUseByID(item.ID)
	if err != nil {
		return err
	}
	if found && useDef.InstanceRequired {
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
	}
	var inventory models.Inventory
	err = tx.Where("user_id = ? AND item_id = ? AND raid_extract = ?", userID, item.ID, raidExtract).First(&inventory).Error
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

// removeInventoryItemFromSource 从指定来源扣除库存，未指定时优先扣局内带出（raidExtract=true）；来源不足会跨来源继续扣。
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
		// 带数量下限的条件原子扣减，防止并发场景下超扣。
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
