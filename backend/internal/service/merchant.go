package service

// 商人服务：处理商品目录、好感度价格、购买校验、出售与商人好感度奖励。
import (
	"errors"
	"fmt"
	"math"

	"idle/internal/models"

	"gorm.io/gorm"
)

// MerchantCatalogItem 商人商品，价格已按商人好感度计算。
type MerchantCatalogItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Detail    string `json:"detail"`
	BasePrice int    `json:"basePrice"`
	Price     int    `json:"price"`
	SellPrice int    `json:"sellPrice"`
	Weight    int    `json:"weight"`
	Slots     int    `json:"slots"`
	RepReq    int    `json:"repRequirement"`
}

// buyMultiplier 根据好感度降低购买价格，最低为基准价的 50%。
func buyMultiplier(rep int) float64 { return math.Max(0.5, 1.0-float64(rep)*0.005) }

// sellMultiplier 根据好感度提高出售价格，最高为基准价的 45%。
func sellMultiplier(rep int) float64 { return math.Min(0.45, 0.3+float64(rep)*0.003) }

func roundPrice(base int, multiplier float64) int {
	return int(math.Round(float64(base) * multiplier))
}

// GetMerchants 返回按展示顺序排列的商人。
func GetMerchants(db *gorm.DB) ([]models.MerchantDef, error) {
	var list []models.MerchantDef
	if err := db.Order("sort_order asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("读取商人: %w", err)
	}
	return list, nil
}

// GetMerchantByID 按 ID 读取商人。
func GetMerchantByID(db *gorm.DB, id string) (*models.MerchantDef, error) {
	var merchant models.MerchantDef
	if err := db.First(&merchant, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("商人不存在")
		}
		return nil, fmt.Errorf("读取商人: %w", err)
	}
	return &merchant, nil
}

// MerchantCatalog 返回指定商人当前可售的商品目录。
func MerchantCatalog(db *gorm.DB, merchant *models.MerchantDef) ([]MerchantCatalogItem, error) {
	if !merchant.Open {
		return nil, nil
	}

	buyPrice := buyMultiplier(merchant.Reputation)
	sellPrice := sellMultiplier(merchant.Reputation)
	items := make([]MerchantCatalogItem, 0)

	var weapons []models.WeaponDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&weapons).Error; err != nil {
		return nil, err
	}
	for _, weapon := range weapons {
		items = append(items, MerchantCatalogItem{
			ID: weapon.ID, Name: weapon.Name, Kind: "weapon",
			Detail:    fmt.Sprintf("伤害 %d / 穿透 %d", weapon.Damage, weapon.Penetration),
			BasePrice: weapon.Price, Price: roundPrice(weapon.Price, buyPrice), SellPrice: roundPrice(weapon.Price, sellPrice),
			Weight: weapon.Weight, Slots: weapon.Slots, RepReq: weapon.RepRequirement,
		})
	}

	var armors []models.ArmorDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&armors).Error; err != nil {
		return nil, err
	}
	for _, armor := range armors {
		items = append(items, MerchantCatalogItem{
			ID: armor.ID, Name: armor.Name, Kind: "armor",
			Detail:    fmt.Sprintf("防护 %d / 覆盖 %d%%", armor.Protect, armor.Coverage),
			BasePrice: armor.Price, Price: roundPrice(armor.Price, buyPrice), SellPrice: roundPrice(armor.Price, sellPrice),
			Weight: armor.Weight, Slots: armor.Slots, RepReq: armor.RepRequirement,
		})
	}

	var consumables []models.ConsumableDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&consumables).Error; err != nil {
		return nil, err
	}
	for _, consumable := range consumables {
		items = append(items, MerchantCatalogItem{
			ID: consumable.ID, Name: consumable.Name, Kind: "consumable", Detail: consumable.Desc,
			BasePrice: consumable.Price, Price: roundPrice(consumable.Price, buyPrice), SellPrice: roundPrice(consumable.Price, sellPrice),
			Weight: consumable.Weight, Slots: consumable.Slots, RepReq: consumable.RepRequirement,
		})
	}

	var chestRigs []models.ChestRigDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&chestRigs).Error; err != nil {
		return nil, err
	}
	for _, chestRig := range chestRigs {
		items = append(items, MerchantCatalogItem{
			ID: chestRig.ID, Name: chestRig.Name, Kind: "chestrig",
			Detail:    fmt.Sprintf("格数 +%d / 负重 +%dkg", chestRig.AddSlots, chestRig.AddWeight),
			BasePrice: chestRig.Price, Price: roundPrice(chestRig.Price, buyPrice), SellPrice: roundPrice(chestRig.Price, sellPrice),
			Weight: chestRig.Weight, Slots: chestRig.Slots, RepReq: chestRig.RepRequirement,
		})
	}

	var backpacks []models.BackpackDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&backpacks).Error; err != nil {
		return nil, err
	}
	for _, backpack := range backpacks {
		items = append(items, MerchantCatalogItem{
			ID: backpack.ID, Name: backpack.Name, Kind: "backpack",
			Detail:    fmt.Sprintf("格数 +%d / 负重 +%dkg", backpack.AddSlots, backpack.AddWeight),
			BasePrice: backpack.Price, Price: roundPrice(backpack.Price, buyPrice), SellPrice: roundPrice(backpack.Price, sellPrice),
			Weight: backpack.Weight, Slots: backpack.Slots, RepReq: backpack.RepRequirement,
		})
	}

	var helmets []models.HelmetDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&helmets).Error; err != nil {
		return nil, err
	}
	for _, helmet := range helmets {
		items = append(items, MerchantCatalogItem{
			ID: helmet.ID, Name: helmet.Name, Kind: "helmet",
			Detail:    fmt.Sprintf("防护 %d / 覆盖 %d%%", helmet.Protect, helmet.Coverage),
			BasePrice: helmet.Price, Price: roundPrice(helmet.Price, buyPrice), SellPrice: roundPrice(helmet.Price, sellPrice),
			Weight: helmet.Weight, Slots: helmet.Slots, RepReq: helmet.RepRequirement,
		})
	}

	var headsets []models.HeadsetDef
	if err := db.Where("merchant_category = ?", merchant.Category).Find(&headsets).Error; err != nil {
		return nil, err
	}
	for _, headset := range headsets {
		items = append(items, MerchantCatalogItem{
			ID: headset.ID, Name: headset.Name, Kind: "headset",
			Detail:    fmt.Sprintf("听力 Lv.%d", headset.HearingLevel),
			BasePrice: headset.Price, Price: roundPrice(headset.Price, buyPrice), SellPrice: roundPrice(headset.Price, sellPrice),
			Weight: headset.Weight, Slots: headset.Slots, RepReq: headset.RepRequirement,
		})
	}

	return items, nil
}

// applyMerchantPrice 根据商品归属商人计算实际购买价，并校验商人状态与好感度解锁。
func applyMerchantPrice(tx *gorm.DB, item *catalogItem) error {
	if item.MerchantCategory == "" {
		return fmt.Errorf("物品 %s 不属于任何商人", item.ID)
	}
	merchant, err := GetMerchantByID(tx, item.MerchantCategory)
	if err != nil {
		return err
	}
	if !merchant.Open {
		return fmt.Errorf("物品 %s 的商人暂未开放", item.ID)
	}
	if item.RepRequirement > merchant.Reputation {
		return fmt.Errorf("好感度不足，无法购买 %s（需 %d）", item.ID, item.RepRequirement)
	}
	item.PaidPrice = roundPrice(item.Price, buyMultiplier(merchant.Reputation))
	return nil
}

// PurchaseFromMerchant 从指定商人购买商品，校验商人归属与好感度解锁。
func PurchaseFromMerchant(db *gorm.DB, merchantID, itemID string, quantity int) error {
	if quantity <= 0 || quantity > 99 {
		return fmt.Errorf("购买数量需为 1-99")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		merchant, err := GetMerchantByID(tx, merchantID)
		if err != nil {
			return err
		}
		if !merchant.Open {
			return fmt.Errorf("该商人暂未开放")
		}

		item, err := findCatalogItem(tx, itemID)
		if err != nil {
			return err
		}
		if item.MerchantCategory != merchant.Category {
			return fmt.Errorf("该商人不经营此类物品")
		}
		if err := applyMerchantPrice(tx, &item); err != nil {
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

// SellItem 将物品出售给对应商人，返回获得的现金。
func SellItem(db *gorm.DB, merchantID, itemID string, quantity int) (int, error) {
	if quantity <= 0 || quantity > 99 {
		return 0, fmt.Errorf("出售数量需为 1-99")
	}

	total := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		merchant, err := GetMerchantByID(tx, merchantID)
		if err != nil {
			return err
		}
		if !merchant.Open {
			return fmt.Errorf("该商人暂未开放")
		}

		var sample models.Inventory
		if err := tx.Where("item_id = ? AND quantity > 0", itemID).First(&sample).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("仓库中没有 %s", itemID)
			}
			return fmt.Errorf("读取可出售物品: %w", err)
		}
		if sample.MerchantCategory != merchant.Category {
			return fmt.Errorf("该商人不收购此类物品")
		}

		var sum struct{ Qty int }
		if err := tx.Model(&models.Inventory{}).
			Where("item_id = ? AND quantity > 0", itemID).
			Select("COALESCE(SUM(quantity), 0) AS qty").Scan(&sum).Error; err != nil {
			return fmt.Errorf("统计可出售数量: %w", err)
		}
		if sum.Qty < quantity {
			return fmt.Errorf("%s 可出售数量不足（当前 %d）", itemID, sum.Qty)
		}

		price := roundPrice(sample.Price, sellMultiplier(merchant.Reputation))
		total = price * quantity
		if err := addCash(tx, total); err != nil {
			return err
		}
		return removeInventoryItem(tx, itemID, quantity)
	})
	return total, err
}

// AwardReputation 提升指定商人的好感度。
func AwardReputation(db *gorm.DB, merchantID string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("好感度奖励需为正数")
	}

	result := db.Model(&models.MerchantDef{}).
		Where("id = ?", merchantID).
		Update("reputation", gorm.Expr("reputation + ?", amount))
	if result.Error != nil {
		return fmt.Errorf("提升好感度: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("商人不存在")
	}
	return nil
}
