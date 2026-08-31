package service

// 商人服务：处理商品目录、好感度价格、购买校验、出售与商人好感度奖励。
import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// MerchantCatalogItem 商人商品，价格已按商人好感度计算；Buyable 表示是否允许购买。
type MerchantCatalogItem struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Kind              string  `json:"kind"`
	Buyable           bool    `json:"buyable"`
	Category          string  `json:"category"`
	Detail            string  `json:"detail"`
	BasePrice         int     `json:"basePrice"`
	Price             int     `json:"price"`
	SellPrice         int     `json:"sellPrice"`
	Weight            int     `json:"weight"`
	Slots             int     `json:"slots"`
	RepReq            int     `json:"repRequirement"`
	HPRecovery        float64 `json:"hpRecovery"`
	EnergyRecovery    float64 `json:"energyRecovery"`
	HydrationRecovery float64 `json:"hydrationRecovery"`
	RepairValue       float64 `json:"repairValue"`
	FuelSeconds       int64   `json:"fuelSeconds"`
	MaxDurability     float64 `json:"maxDurability"`
	InstanceRequired  bool    `json:"instanceRequired"`
}

var ErrMerchantUnavailable = errors.New("商人商品不可用")

// buyMultiplier 根据好感度降低购买价格，最低为基准价的 50%。
func buyMultiplier(rep int) float64 { return math.Max(0.5, 1.0-float64(rep)*0.005) }

// sellMultiplier 根据好感度提高出售价格，最高为基准价的 45%。
func sellMultiplier(rep int) float64 { return math.Min(0.45, 0.3+float64(rep)*0.003) }

func roundPrice(base int, multiplier float64) int {
	return int(math.Round(float64(base) * multiplier))
}

// GetMerchantsForUser 返回指定用户看到的商人状态。
func GetMerchantsForUser(db *gorm.DB, userID uint) ([]models.MerchantDef, error) {
	var list []models.MerchantDef
	if err := db.Order("sort_order asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("读取商人: %w", err)
	}
	var states []models.UserMerchantState
	if err := db.Where("user_id = ?", userID).Find(&states).Error; err != nil {
		return nil, fmt.Errorf("读取商人状态: %w", err)
	}
	stateByMerchant := make(map[string]models.UserMerchantState, len(states))
	for _, state := range states {
		stateByMerchant[state.MerchantID] = state
	}
	for i := range list {
		if state, ok := stateByMerchant[list[i].ID]; ok {
			list[i].Reputation = state.Reputation
			list[i].Open = state.Unlocked
		}
	}
	return list, nil
}

// GetMerchantByIDForUser 读取指定用户的商人状态。
func GetMerchantByIDForUser(db *gorm.DB, userID uint, id string) (*models.MerchantDef, error) {
	var merchant models.MerchantDef
	if err := db.First(&merchant, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("商人不存在")
		}
		return nil, fmt.Errorf("读取商人: %w", err)
	}
	var state models.UserMerchantState
	if err := db.Where("user_id = ? AND merchant_id = ?", userID, id).First(&state).Error; err == nil {
		merchant.Reputation = state.Reputation
		merchant.Open = state.Unlocked
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("读取商人状态: %w", err)
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
	catalogRepo := catalog.New(db)
	catalogItems, err := catalogRepo.ListByMerchantCategory(merchant.Category)
	if err != nil {
		return nil, err
	}
	itemIDs := make([]string, 0, len(catalogItems))
	for _, item := range catalogItems {
		itemIDs = append(itemIDs, item.ID)
	}
	useByID, err := catalogRepo.FindUsesByIDs(itemIDs)
	if err != nil {
		return nil, err
	}

	for _, item := range catalogItems {
		base := MerchantCatalogItem{
			ID: item.ID, Name: item.Name, Kind: item.Kind,
			Buyable:   true,
			BasePrice: item.Price, Price: roundPrice(item.Price, buyPrice), SellPrice: roundPrice(item.Price, sellPrice),
			Weight: item.Weight, Slots: item.Slots, RepReq: item.RepRequirement,
		}
		switch item.Kind {
		case "weapon":
			base.Detail = fmt.Sprintf("伤害 %d / 近战穿透 %d", item.Damage, item.Penetration)
			if item.AmmoPerRound > 0 {
				base.Detail = fmt.Sprintf("伤害 %d / 口径 %s", item.Damage, item.CaliberID)
			}
		case "ammo":
			if item.AmmoLevel > 4 {
				base.Buyable = false
			}
			base.Detail = fmt.Sprintf("口径 %s / N%d / 单发价格", item.CaliberID, item.AmmoLevel)
			base.Slots = 1
		case "armor":
			base.Detail = fmt.Sprintf("A%d / 覆盖 %d%%", item.ProtectionLevel, item.Coverage)
		case "consumable":
			use := useByID[item.ID]
			base.Detail = usableItemDetail(item.Desc, use)
			base.HPRecovery, base.EnergyRecovery, base.HydrationRecovery = use.HPRecovery, use.EnergyRecovery, use.HydrationRecovery
			base.RepairValue, base.FuelSeconds, base.MaxDurability, base.InstanceRequired = use.RepairValue, use.FuelSeconds, use.MaxDurability, use.InstanceRequired
		case "loot":
			use, ok := useByID[item.ID]
			if !ok || (!use.UsableInSession && !use.UsableInHideout && use.RepairValue <= 0 && use.FuelSeconds <= 0) {
				base.Buyable = false
			}
			base.Category = item.Category
			base.Detail = usableItemDetail(item.Desc, use)
			base.HPRecovery, base.EnergyRecovery, base.HydrationRecovery = use.HPRecovery, use.EnergyRecovery, use.HydrationRecovery
			base.RepairValue, base.FuelSeconds, base.MaxDurability, base.InstanceRequired = use.RepairValue, use.FuelSeconds, use.MaxDurability, use.InstanceRequired
		case "chestrig", "backpack":
			base.Detail = fmt.Sprintf("格数 +%d / 负重 +%dkg", item.AddSlots, item.AddWeight)
		case "helmet":
			base.Detail = fmt.Sprintf("防护 %d / 覆盖 %d%%", item.Protect, item.Coverage)
		case "headset":
			base.Detail = fmt.Sprintf("听力 Lv.%d", item.HearingLevel)
		default:
			continue
		}
		items = append(items, base)
	}
	return items, nil
}

func usableItemDetail(desc string, use models.ItemUseDef) string {
	detail := desc
	if use.HPRecovery > 0 {
		detail += fmt.Sprintf(" · HP +%.0f", use.HPRecovery)
	}
	if use.EnergyRecovery > 0 {
		detail += fmt.Sprintf(" · 能量 +%.0f", use.EnergyRecovery)
	}
	if use.HydrationRecovery > 0 {
		detail += fmt.Sprintf(" · 饮水 +%.0f", use.HydrationRecovery)
	}
	if use.RepairValue > 0 {
		detail += fmt.Sprintf(" · 维修值 %.0f", use.RepairValue)
	}
	if use.FuelSeconds > 0 {
		detail += fmt.Sprintf(" · 燃料 %d 分钟", use.FuelSeconds/60)
	}
	return detail
}

func applyMerchantPriceForUser(tx *gorm.DB, userID uint, item *catalogItem) error {
	if item.Kind == "ammo" && item.AmmoLevel > 4 {
		return fmt.Errorf("%w：武器商人最高只出售 N4 弹药", ErrMerchantUnavailable)
	}
	if item.MerchantCategory == "" {
		return fmt.Errorf("%w：物品 %s 不属于任何商人", ErrMerchantUnavailable, item.ID)
	}
	merchant, err := GetMerchantByIDForUser(tx, userID, item.MerchantCategory)
	if err != nil {
		return err
	}
	if !merchant.Open {
		return fmt.Errorf("%w：物品 %s 的商人暂未开放", ErrMerchantUnavailable, item.ID)
	}
	if item.RepRequirement > merchant.Reputation {
		return fmt.Errorf("%w：好感度不足，无法购买 %s（需 %d）", ErrMerchantUnavailable, item.ID, item.RepRequirement)
	}
	item.PaidPrice = roundPrice(item.Price, buyMultiplier(merchant.Reputation))
	return nil
}

// PurchaseFromMerchantForUser 为指定用户购买商品。
func PurchaseFromMerchantForUser(db *gorm.DB, userID uint, merchantID, itemID string, quantity int) error {
	return PurchaseFromMerchantForUserWithKey(db, userID, "", merchantID, itemID, quantity)
}

// PurchaseFromMerchantForUserWithKey 为指定用户执行可重放的购买操作。
func PurchaseFromMerchantForUserWithKey(db *gorm.DB, userID uint, operationKey, merchantID, itemID string, quantity int) error {
	if quantity <= 0 || quantity > 999 {
		return fmt.Errorf("购买数量需为 1-999")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		operation, replay, err := claimEconomicOperation(tx, userID, operationKey, "purchase")
		if err != nil {
			return err
		}
		if replay {
			return nil
		}
		merchant, err := GetMerchantByIDForUser(tx, userID, merchantID)
		if err != nil {
			return err
		}
		if !merchant.Open {
			return fmt.Errorf("该商人暂未开放")
		}

		item, err := catalog.New(tx).FindByID(itemID)
		if err != nil {
			return err
		}
		if item.MerchantCategory != merchant.Category {
			return fmt.Errorf("该商人不经营此类物品")
		}
		if err := applyMerchantPriceForUser(tx, userID, &item); err != nil {
			return err
		}

		items := make([]catalogItem, quantity)
		for i := range items {
			items[i] = item
		}
		_, err = purchaseCatalogItems(tx, userID, items)
		if err != nil {
			return err
		}
		if operation != nil {
			operation.ResultJSON = `{"ok":true}`
			return tx.Save(operation).Error
		}
		return nil
	})
}

// SellItemForUser 为指定用户出售物品。
func SellItemForUser(db *gorm.DB, userID uint, merchantID, itemID string, quantity int) (int, error) {
	return SellItemForUserWithKey(db, userID, "", merchantID, itemID, quantity)
}

// SellItemForUserWithKey 为指定用户执行可重放的出售操作。
func SellItemForUserWithKey(db *gorm.DB, userID uint, operationKey, merchantID, itemID string, quantity int) (int, error) {
	if quantity <= 0 || quantity > 99 {
		return 0, fmt.Errorf("出售数量需为 1-99")
	}

	total := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		operation, replay, err := claimEconomicOperation(tx, userID, operationKey, "sell")
		if err != nil {
			return err
		}
		if replay {
			var result struct {
				Total int `json:"total"`
			}
			if err := json.Unmarshal([]byte(operation.ResultJSON), &result); err != nil {
				return fmt.Errorf("读取出售结果: %w", err)
			}
			total = result.Total
			return nil
		}
		merchant, err := GetMerchantByIDForUser(tx, userID, merchantID)
		if err != nil {
			return err
		}
		if !merchant.Open {
			return fmt.Errorf("该商人暂未开放")
		}
		if err := ensureItemNotInActiveSession(tx, userID, itemID); err != nil {
			return err
		}

		var useDef models.ItemUseDef
		if err := tx.Where("item_id = ?", itemID).First(&useDef).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("读取出售物品效果: %w", err)
		}
		if useDef.InstanceRequired {
			var instances []models.ItemInstance
			if err := tx.Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ? AND current_durability > 0", userID, itemID, "inventory", "normal").
				Order("current_durability asc, id asc").Limit(quantity).Find(&instances).Error; err != nil {
				return fmt.Errorf("读取可出售物品实例: %w", err)
			}
			if len(instances) < quantity {
				return fmt.Errorf("%s 可出售数量不足（当前 %d）", itemID, len(instances))
			}
			item, err := catalog.New(tx).FindByID(itemID)
			if err != nil {
				return err
			}
			if item.MerchantCategory != merchant.Category {
				return fmt.Errorf("该商人不收购此类物品")
			}
			price := roundPrice(item.Price, sellMultiplier(merchant.Reputation))
			total = price * quantity
			if err := addCash(tx, userID, total); err != nil {
				return err
			}
			for _, instance := range instances {
				if err := tx.Delete(&instance).Error; err != nil {
					return fmt.Errorf("删除出售物品实例 %d: %w", instance.ID, err)
				}
			}
			if operation != nil {
				resultJSON, err := json.Marshal(map[string]int{"total": total})
				if err != nil {
					return err
				}
				operation.ResultJSON = string(resultJSON)
				return tx.Save(operation).Error
			}
			return nil
		}

		var sample models.Inventory
		if err := tx.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).First(&sample).Error; err != nil {
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
			Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).
			Select("COALESCE(SUM(quantity), 0) AS qty").Scan(&sum).Error; err != nil {
			return fmt.Errorf("统计可出售数量: %w", err)
		}
		if sum.Qty < quantity {
			return fmt.Errorf("%s 可出售数量不足（当前 %d）", itemID, sum.Qty)
		}

		price := roundPrice(sample.Price, sellMultiplier(merchant.Reputation))
		total = price * quantity
		if err := addCash(tx, userID, total); err != nil {
			return err
		}
		if err := removeInventoryItem(tx, userID, itemID, quantity); err != nil {
			return err
		}
		if operation != nil {
			resultJSON, err := json.Marshal(map[string]int{"total": total})
			if err != nil {
				return err
			}
			operation.ResultJSON = string(resultJSON)
			return tx.Save(operation).Error
		}
		return nil
	})
	return total, err
}
