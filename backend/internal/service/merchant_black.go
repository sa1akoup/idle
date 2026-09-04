package service

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

const (
	blackMarketRefreshInterval = 6 * time.Hour
	blackMarketOfferCount      = 24
	blackMarketID              = "black"
)

func isBlackMarket(merchant *models.MerchantDef) bool {
	return merchant != nil && (merchant.ID == blackMarketID || merchant.Category == blackMarketID)
}

func merchantAcceptsItem(merchant *models.MerchantDef, itemCategory string) bool {
	if isBlackMarket(merchant) {
		return true
	}
	return itemCategory == merchant.Category
}

func playerSellMultiplier(merchant *models.MerchantDef) float64 {
	if isBlackMarket(merchant) {
		return math.Min(0.28, 0.20+float64(merchant.Reputation)*0.002)
	}
	return sellMultiplier(merchant.Reputation)
}

func merchantBuyMultiplier(merchant *models.MerchantDef) float64 {
	if isBlackMarket(merchant) {
		return 2 * buyMultiplier(merchant.Reputation)
	}
	return buyMultiplier(merchant.Reputation)
}

func blackMarketCycleStart(now time.Time) time.Time {
	unix := now.UTC().Unix()
	interval := int64(blackMarketRefreshInterval / time.Second)
	return time.Unix((unix/interval)*interval, 0).UTC()
}

func blackMarketOfferWeight(item catalog.Item) int {
	if item.Price <= 0 || isMerchantBarter(item.ID) {
		return 0
	}
	if item.Kind == "loot" {
		return item.DropWeight
	}
	if item.Kind == "ammo" {
		switch {
		case item.AmmoLevel >= 6:
			return 1
		case item.AmmoLevel == 5:
			return 2
		case item.AmmoLevel == 4:
			return 4
		case item.AmmoLevel == 3:
			return 12
		default:
			return 40
		}
	}
	switch {
	case item.Price >= 2000:
		return 1
	case item.Price >= 1000:
		return 2
	case item.Price >= 400:
		return 4
	case item.Price >= 150:
		return 12
	default:
		return 40
	}
}

func blackMarketOfferQuantity(item catalog.Item) int {
	switch item.Kind {
	case "ammo":
		if item.RoundsPerSlot > 0 {
			return item.RoundsPerSlot
		}
		return 30
	case "weapon", "armor", "chestrig", "backpack", "helmet", "headset":
		return 1
	}
	weight := blackMarketOfferWeight(item)
	if weight >= 40 {
		return 3
	}
	if weight >= 12 {
		return 2
	}
	return 1
}

type blackMarketPick struct {
	Item     catalog.Item
	Quantity int
}

func pickBlackMarketOffers(rng *rand.Rand, pool []catalog.Item, slots int) []blackMarketPick {
	type weightedItem struct {
		item   catalog.Item
		weight int
	}
	candidates := make([]weightedItem, 0, len(pool))
	for _, item := range pool {
		weight := blackMarketOfferWeight(item)
		if weight <= 0 {
			continue
		}
		candidates = append(candidates, weightedItem{item: item, weight: weight})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].item.ID < candidates[j].item.ID
	})
	if slots > len(candidates) {
		slots = len(candidates)
	}
	picks := make([]blackMarketPick, 0, slots)
	for len(picks) < slots && len(candidates) > 0 {
		total := 0
		for _, candidate := range candidates {
			total += candidate.weight
		}
		roll := rng.Intn(total)
		chosen := 0
		for index, candidate := range candidates {
			roll -= candidate.weight
			if roll < 0 {
				chosen = index
				break
			}
		}
		item := candidates[chosen].item
		picks = append(picks, blackMarketPick{Item: item, Quantity: blackMarketOfferQuantity(item)})
		candidates = append(candidates[:chosen], candidates[chosen+1:]...)
	}
	return picks
}

func blackMarketCatalog(db *gorm.DB, userID uint, merchant *models.MerchantDef) (MerchantCatalogResult, error) {
	var result MerchantCatalogResult
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		now := time.Now()
		offers, cycleStart, err := ensureBlackMarketStockTx(tx, userID, now)
		if err != nil {
			return err
		}
		catalogItems, err := catalogItemsFromBlackOffers(tx, merchant, offers)
		if err != nil {
			return err
		}
		next := cycleStart.Add(blackMarketRefreshInterval)
		result = MerchantCatalogResult{
			Items:          catalogItems,
			NextRefreshAt:  &next,
			AcceptsAny:     true,
			PlayerSellRate: playerSellMultiplier(merchant),
		}
		return nil
	})
	if err != nil {
		return MerchantCatalogResult{}, err
	}
	return result, nil
}

func ensureBlackMarketStockTx(tx *gorm.DB, userID uint, now time.Time) ([]models.UserBlackMarketOffer, time.Time, error) {
	cycleStart := blackMarketCycleStart(now)
	var offers []models.UserBlackMarketOffer
	if err := tx.Where("user_id = ?", userID).Order("item_id asc").Find(&offers).Error; err != nil {
		return nil, time.Time{}, fmt.Errorf("读取黑市货架: %w", err)
	}
	if len(offers) > 0 && offers[0].CycleStart.Equal(cycleStart) {
		return offers, cycleStart, nil
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserBlackMarketOffer{}).Error; err != nil {
		return nil, time.Time{}, fmt.Errorf("清空过期黑市货架: %w", err)
	}
	pool, err := catalog.New(tx).ListAll()
	if err != nil {
		return nil, time.Time{}, err
	}
	seed := int64(userID) ^ cycleStart.Unix()
	picks := pickBlackMarketOffers(rand.New(rand.NewSource(seed)), pool, blackMarketOfferCount)
	offers = make([]models.UserBlackMarketOffer, 0, len(picks))
	for _, pick := range picks {
		offers = append(offers, models.UserBlackMarketOffer{
			UserID: userID, ItemID: pick.Item.ID, Quantity: pick.Quantity, CycleStart: cycleStart,
		})
	}
	if len(offers) > 0 {
		if err := tx.Create(&offers).Error; err != nil {
			return nil, time.Time{}, fmt.Errorf("写入黑市货架: %w", err)
		}
	}
	return offers, cycleStart, nil
}

func catalogItemsFromBlackOffers(tx *gorm.DB, merchant *models.MerchantDef, offers []models.UserBlackMarketOffer) ([]MerchantCatalogItem, error) {
	itemIDs := make([]string, 0, len(offers))
	qtyByID := make(map[string]int, len(offers))
	for _, offer := range offers {
		if offer.Quantity <= 0 {
			continue
		}
		itemIDs = append(itemIDs, offer.ItemID)
		qtyByID[offer.ItemID] = offer.Quantity
	}
	if len(itemIDs) == 0 {
		return []MerchantCatalogItem{}, nil
	}
	repo := catalog.New(tx)
	found, err := repo.FindByIDs(itemIDs)
	if err != nil {
		return nil, err
	}
	useByID, err := repo.FindUsesByIDs(itemIDs)
	if err != nil {
		return nil, err
	}
	buyPrice := merchantBuyMultiplier(merchant)
	sellPrice := playerSellMultiplier(merchant)
	items := make([]MerchantCatalogItem, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		item, ok := found[itemID]
		if !ok {
			continue
		}
		entry := MerchantCatalogItem{
			ID: item.ID, Name: item.Name, Kind: item.Kind, Buyable: true, Category: item.Category,
			BasePrice: item.Price, Price: roundPrice(item.Price, buyPrice), SellPrice: roundPrice(item.Price, sellPrice),
			Weight: item.Weight, Slots: item.Slots, Stock: qtyByID[itemID],
		}
		switch item.Kind {
		case "weapon":
			entry.Detail = fmt.Sprintf("伤害 %d / 近战穿透 %d", item.Damage, item.Penetration)
			if item.AmmoPerRound > 0 {
				entry.Detail = fmt.Sprintf("伤害 %d / 口径 %s", item.Damage, item.CaliberID)
			}
		case "ammo":
			entry.Detail = fmt.Sprintf("口径 %s / N%d / 单发价格", item.CaliberID, item.AmmoLevel)
			entry.Slots = 1
		case "armor":
			entry.Detail = fmt.Sprintf("A%d / 覆盖 %d%%", item.ProtectionLevel, item.Coverage)
		case "consumable":
			use := useByID[item.ID]
			entry.Detail = usableItemDetail(item.Desc, use)
			entry.HPRecovery, entry.EnergyRecovery, entry.HydrationRecovery = use.HPRecovery, use.EnergyRecovery, use.HydrationRecovery
			entry.RepairValue, entry.FuelSeconds, entry.MaxDurability, entry.InstanceRequired = use.RepairValue, use.FuelSeconds, use.MaxDurability, use.InstanceRequired
		case "loot":
			use := useByID[item.ID]
			entry.Detail = usableItemDetail(item.Desc, use)
			entry.HPRecovery, entry.EnergyRecovery, entry.HydrationRecovery = use.HPRecovery, use.EnergyRecovery, use.HydrationRecovery
			entry.RepairValue, entry.FuelSeconds, entry.MaxDurability, entry.InstanceRequired = use.RepairValue, use.FuelSeconds, use.MaxDurability, use.InstanceRequired
		case "chestrig", "backpack":
			entry.Detail = fmt.Sprintf("格数 +%d / 负重 +%dkg", item.AddSlots, item.AddWeight)
		case "helmet":
			entry.Detail = fmt.Sprintf("防护 %d / 覆盖 %d%%", item.Protect, item.Coverage)
		case "headset":
			entry.Detail = fmt.Sprintf("听力 Lv.%d", item.HearingLevel)
		case "keycase":
			entry.Detail = fmt.Sprintf("钥匙格 %d", item.AddSlots)
		case "secure":
			entry.Detail = fmt.Sprintf("口袋 %d 格 · 失能保搜刮", item.AddSlots)
		}
		items = append(items, entry)
	}
	return items, nil
}

func purchaseBlackMarketTx(tx *gorm.DB, userID uint, merchant *models.MerchantDef, itemID string, quantity int, operation *models.EconomicOperation) error {
	if isMerchantBarter(itemID) {
		return fmt.Errorf("该物品只能在服装商人处以物兑换")
	}
	_, _, err := ensureBlackMarketStockTx(tx, userID, time.Now())
	if err != nil {
		return err
	}
	var offer models.UserBlackMarketOffer
	if err := tx.Where("user_id = ? AND item_id = ?", userID, itemID).First(&offer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("该物品本轮未上架")
		}
		return fmt.Errorf("读取黑市库存: %w", err)
	}
	if offer.Quantity < quantity {
		return fmt.Errorf("%s 黑市库存不足（当前 %d）", itemID, offer.Quantity)
	}
	item, err := catalog.New(tx).FindByID(itemID)
	if err != nil {
		return err
	}
	item.PaidPrice = roundPrice(item.Price, merchantBuyMultiplier(merchant))
	copies := make([]catalogItem, quantity)
	for i := range copies {
		copies[i] = item
	}
	if _, err := purchaseCatalogItems(tx, userID, copies); err != nil {
		return err
	}
	offer.Quantity -= quantity
	if offer.Quantity <= 0 {
		if err := tx.Delete(&offer).Error; err != nil {
			return fmt.Errorf("扣减黑市库存: %w", err)
		}
	} else if err := tx.Model(&offer).Update("quantity", offer.Quantity).Error; err != nil {
		return fmt.Errorf("扣减黑市库存: %w", err)
	}
	if operation != nil {
		operation.ResultJSON = `{"ok":true}`
		return tx.Save(operation).Error
	}
	return nil
}
