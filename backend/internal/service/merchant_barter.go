package service

// 商人兑换：钥匙包与安全箱用局内带出材料换，可另加合同门槛。
import (
	"errors"
	"fmt"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
)

// MerchantBarterCost 兑换材料：需求与当前局内带出持有量。
type MerchantBarterCost struct {
	ItemID string `json:"itemId"`
	Name   string `json:"name"`
	Need   int    `json:"need"`
	Have   int    `json:"have"`
}

type merchantBarterOffer struct {
	UnlockQuestID string
	Costs         []models.RecipeInput
}

var merchantBarters = map[string]merchantBarterOffer{
	"keycase_09": {Costs: []models.RecipeInput{
		{ItemID: "hydrogen_peroxide", Quantity: 3},
		{ItemID: "saline_solution", Quantity: 3},
		{ItemID: "cat_figurine", Quantity: 1},
	}},
	"keycase_12": {Costs: []models.RecipeInput{
		{ItemID: "bronze_lion", Quantity: 1},
		{ItemID: "horse_figurine", Quantity: 2},
		{ItemID: "duct_tape", Quantity: 4},
		{ItemID: "insulating_tape", Quantity: 4},
		{ItemID: "pack_of_nails", Quantity: 4},
	}},
	"secure_02": {UnlockQuestID: "clothing_secure_2", Costs: []models.RecipeInput{
		{ItemID: "duct_tape", Quantity: 2},
		{ItemID: "sewing_kit", Quantity: 1},
	}},
	"secure_04": {UnlockQuestID: "clothing_secure_4", Costs: []models.RecipeInput{
		{ItemID: "cat_figurine", Quantity: 1},
		{ItemID: "saline_solution", Quantity: 2},
		{ItemID: "hydrogen_peroxide", Quantity: 2},
	}},
	"secure_06": {UnlockQuestID: "clothing_secure_6", Costs: []models.RecipeInput{
		{ItemID: "horse_figurine", Quantity: 1},
		{ItemID: "insulating_tape", Quantity: 4},
		{ItemID: "pack_of_nails", Quantity: 4},
		{ItemID: "bundle_of_wires", Quantity: 2},
	}},
	"secure_09": {UnlockQuestID: "clothing_secure_9", Costs: []models.RecipeInput{
		{ItemID: "bronze_lion", Quantity: 1},
		{ItemID: "cat_figurine", Quantity: 1},
		{ItemID: "gold_chain", Quantity: 1},
	}},
}

func isMerchantBarter(itemID string) bool {
	_, ok := merchantBarters[itemID]
	return ok
}

func attachMerchantBarter(db *gorm.DB, userID uint, item *MerchantCatalogItem) error {
	offer, ok := merchantBarters[item.ID]
	if !ok || len(offer.Costs) == 0 {
		return nil
	}
	views, err := merchantBarterViews(db, userID, offer.Costs)
	if err != nil {
		return err
	}
	item.BarterCosts = views
	item.Price = 0
	item.Detail += " · 局内带出兑换"
	locked, reason, err := merchantBarterLockReason(db, userID, offer.UnlockQuestID)
	if err != nil {
		return err
	}
	if locked {
		item.BarterLocked = true
		item.BarterLockReason = reason
		item.Detail += " · " + reason
	}
	return nil
}

func merchantBarterViews(db *gorm.DB, userID uint, costs []models.RecipeInput) ([]MerchantBarterCost, error) {
	itemIDs := make([]string, 0, len(costs))
	for _, cost := range costs {
		itemIDs = append(itemIDs, cost.ItemID)
	}
	catalogItems, err := catalog.New(db).FindByIDs(itemIDs)
	if err != nil {
		return nil, err
	}
	views := make([]MerchantBarterCost, 0, len(costs))
	for _, cost := range costs {
		have, err := ownedRaidExtractQuantityTx(db, userID, cost.ItemID)
		if err != nil {
			return nil, err
		}
		name := cost.ItemID
		if item, ok := catalogItems[cost.ItemID]; ok {
			name = item.Name
		}
		views = append(views, MerchantBarterCost{
			ItemID: cost.ItemID, Name: name, Need: cost.Quantity, Have: have,
		})
	}
	return views, nil
}

func merchantBarterLockReason(db *gorm.DB, userID uint, questID string) (bool, string, error) {
	if questID == "" {
		return false, "", nil
	}
	var quest models.QuestDef
	if err := db.Where("id = ?", questID).First(&quest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, "完成对应合同后可兑换", nil
		}
		return false, "", fmt.Errorf("读取兑换合同 %s: %w", questID, err)
	}
	var state models.UserQuest
	err := db.Where("user_id = ? AND quest_id = ? AND status = ?", userID, questID, models.QuestStatusCompleted).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, "完成合同「" + quest.Name + "」后可兑换", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("读取兑换合同进度 %s: %w", questID, err)
	}
	return false, "", nil
}

func purchaseMerchantBarterTx(tx *gorm.DB, userID uint, merchant *models.MerchantDef, item catalogItem, quantity int, operation *models.EconomicOperation) error {
	offer, ok := merchantBarters[item.ID]
	if !ok {
		return fmt.Errorf("该物品没有兑换配方")
	}
	if quantity != 1 {
		return fmt.Errorf("兑换每次限 1 件")
	}
	if item.RepRequirement > merchant.Reputation {
		return fmt.Errorf("%w：好感度不足，无法兑换 %s（需 %d）", ErrMerchantUnavailable, item.ID, item.RepRequirement)
	}
	locked, reason, err := merchantBarterLockReason(tx, userID, offer.UnlockQuestID)
	if err != nil {
		return err
	}
	if locked {
		return fmt.Errorf("%w：%s", ErrMerchantUnavailable, reason)
	}
	for _, cost := range offer.Costs {
		if err := consumeRequirementItemTx(tx, userID, cost.ItemID, cost.Quantity); err != nil {
			return fmt.Errorf("兑换材料不足：%w", err)
		}
	}
	item.SkipCash = true
	item.PaidPrice = 0
	if _, err := purchaseCatalogItems(tx, userID, []catalogItem{item}); err != nil {
		return err
	}
	if operation != nil {
		operation.ResultJSON = `{"ok":true,"barter":true}`
		return tx.Save(operation).Error
	}
	return nil
}
