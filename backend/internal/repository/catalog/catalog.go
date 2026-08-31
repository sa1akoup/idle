// 只读商品目录仓储：统一跨分表查询、批量读取和请求内缓存，供服务层复用。
package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"idle/internal/models"

	"gorm.io/gorm"
)

// ErrItemNotFound 表示商品 ID 不存在于任何目录表。
var ErrItemNotFound = errors.New("商品不存在")

// Item 是服务层需要的统一商品视图。
// 目录专属字段集中在这里，避免每个调用方重复做分表适配。
type Item struct {
	ID               string
	Name             string
	Kind             string
	Category         string
	Desc             string
	Price            int
	PaidPrice        int
	Weight           int
	Slots            int
	DropWeight       int
	MerchantCategory string
	RepRequirement   int
	ArmorMax         int
	RoundsPerSlot    int
	AmmoLevel        int

	CaliberID       string
	AmmoPerRound    int
	Damage          int
	Penetration     int
	ProtectionLevel int
	Coverage        int
	AddSlots        int
	AddWeight       int
	Protect         int
	HearingLevel    int
}

// Repository 封装一次请求或一次事务内的目录读取。
// Item 和 use definition 均按 ID 缓存，重复读取不会再次访问数据库。
type Repository struct {
	db      *gorm.DB
	items   map[string]Item
	missing map[string]struct{}
	uses    map[string]models.ItemUseDef
	noUses  map[string]struct{}
}

// New 创建一个绑定到当前 GORM 查询上下文的目录仓储。
func New(db *gorm.DB) *Repository {
	return &Repository{
		db:      db,
		items:   make(map[string]Item),
		missing: make(map[string]struct{}),
		uses:    make(map[string]models.ItemUseDef),
		noUses:  make(map[string]struct{}),
	}
}

// FindByID 读取单个商品；同一 Repository 实例内命中缓存时不再查询数据库。
func (r *Repository) FindByID(itemID string) (Item, error) {
	items, err := r.FindByIDs([]string{itemID})
	if err != nil {
		return Item{}, err
	}
	return items[itemID], nil
}

// FindByIDs 批量读取商品。由于目录仍按类型分表，单次批量读取最多访问每张目录表一次。
// 返回值会保留已找到的项目；当部分 ID 缺失时同时返回 ErrItemNotFound，便于调用方决定是否允许缺省。
func (r *Repository) FindByIDs(itemIDs []string) (map[string]Item, error) {
	result := make(map[string]Item, len(itemIDs))
	pending := uniqueIDs(itemIDs)
	uncached := make([]string, 0, len(pending))
	for _, itemID := range pending {
		if item, ok := r.items[itemID]; ok {
			result[itemID] = item
			continue
		}
		if _, ok := r.missing[itemID]; ok {
			continue
		}
		uncached = append(uncached, itemID)
	}

	if len(uncached) > 0 {
		if err := r.queryItems(uncached); err != nil {
			return result, err
		}
		for _, itemID := range uncached {
			if item, ok := r.items[itemID]; ok {
				result[itemID] = item
				continue
			}
			r.missing[itemID] = struct{}{}
		}
	}

	missing := make([]string, 0)
	for _, itemID := range pending {
		if item, ok := r.items[itemID]; ok {
			result[itemID] = item
			continue
		}
		missing = append(missing, itemID)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return result, fmt.Errorf("%w: %s", ErrItemNotFound, strings.Join(missing, ", "))
	}
	return result, nil
}

// FindUseByID 读取单个物品使用定义。没有使用定义属于合法缺省，返回 found=false。
func (r *Repository) FindUseByID(itemID string) (models.ItemUseDef, bool, error) {
	if use, ok := r.uses[itemID]; ok {
		return use, true, nil
	}
	if _, ok := r.noUses[itemID]; ok {
		return models.ItemUseDef{}, false, nil
	}

	var use models.ItemUseDef
	if err := r.db.Where("item_id = ?", itemID).First(&use).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.noUses[itemID] = struct{}{}
			return models.ItemUseDef{}, false, nil
		}
		return models.ItemUseDef{}, false, fmt.Errorf("读取物品使用效果 %s: %w", itemID, err)
	}
	r.uses[itemID] = use
	return use, true, nil
}

// FindUsesByIDs 批量读取物品使用定义。没有定义的 ID 不会报错，由调用方按业务语义处理。
func (r *Repository) FindUsesByIDs(itemIDs []string) (map[string]models.ItemUseDef, error) {
	result := make(map[string]models.ItemUseDef, len(itemIDs))
	pending := make([]string, 0, len(itemIDs))
	for _, itemID := range uniqueIDs(itemIDs) {
		if use, ok := r.uses[itemID]; ok {
			result[itemID] = use
			continue
		}
		if _, ok := r.noUses[itemID]; ok {
			continue
		}
		pending = append(pending, itemID)
	}
	if len(pending) == 0 {
		return result, nil
	}

	var uses []models.ItemUseDef
	if err := r.db.Where("item_id IN ?", pending).Find(&uses).Error; err != nil {
		return result, fmt.Errorf("读取物品使用效果: %w", err)
	}
	for _, use := range uses {
		r.uses[use.ItemID] = use
		result[use.ItemID] = use
	}
	for _, itemID := range pending {
		if _, ok := r.uses[itemID]; !ok {
			r.noUses[itemID] = struct{}{}
		}
	}
	return result, nil
}

// ListByMerchantCategory 返回指定商人类别下的全部目录商品。
// 弹药高等级过滤仍由服务层按商人规则决定，避免仓储混入交易策略。
func (r *Repository) ListByMerchantCategory(category string) ([]Item, error) {
	items := make([]Item, 0)

	var weapons []models.WeaponDef
	if err := r.db.Where("merchant_category = ?", category).Find(&weapons).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Order("caliber_id asc, level asc").Find(&ammos).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Find(&armors).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Find(&consumables).Error; err != nil {
		return nil, fmt.Errorf("读取补给目录: %w", err)
	}
	for _, consumable := range consumables {
		items = append(items, Item{
			ID: consumable.ID, Name: consumable.Name, Kind: "consumable", Desc: consumable.Desc, Price: consumable.Price,
			Weight: consumable.Weight, Slots: consumable.Slots, MerchantCategory: consumable.MerchantCategory, RepRequirement: consumable.RepRequirement,
		})
	}

	var lootItems []models.LootItemDef
	if err := r.db.Where("merchant_category = ?", category).Order("id asc").Find(&lootItems).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Find(&chestRigs).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Find(&backpacks).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Find(&helmets).Error; err != nil {
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
	if err := r.db.Where("merchant_category = ?", category).Find(&headsets).Error; err != nil {
		return nil, fmt.Errorf("读取耳机目录: %w", err)
	}
	for _, headset := range headsets {
		items = append(items, Item{
			ID: headset.ID, Name: headset.Name, Kind: "headset", Price: headset.Price, Weight: headset.Weight, Slots: headset.Slots,
			MerchantCategory: headset.MerchantCategory, RepRequirement: headset.RepRequirement, HearingLevel: headset.HearingLevel,
		})
	}
	return items, nil
}

func (r *Repository) queryItems(itemIDs []string) error {
	var weapons []models.WeaponDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&weapons).Error; err != nil {
		return fmt.Errorf("读取武器商品: %w", err)
	}
	for _, weapon := range weapons {
		r.items[weapon.ID] = Item{
			ID: weapon.ID, Name: weapon.Name, Kind: "weapon", Price: weapon.Price, Weight: weapon.Weight, Slots: weapon.Slots,
			MerchantCategory: weapon.MerchantCategory, RepRequirement: weapon.RepRequirement,
			CaliberID: weapon.CaliberID, AmmoPerRound: weapon.AmmoPerRound, Damage: weapon.Damage, Penetration: weapon.Penetration,
		}
	}

	var armors []models.ArmorDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&armors).Error; err != nil {
		return fmt.Errorf("读取护甲商品: %w", err)
	}
	for _, armor := range armors {
		if _, exists := r.items[armor.ID]; exists {
			continue
		}
		r.items[armor.ID] = Item{
			ID: armor.ID, Name: armor.Name, Kind: "armor", Price: armor.Price, Weight: armor.Weight, Slots: armor.Slots,
			MerchantCategory: armor.MerchantCategory, RepRequirement: armor.RepRequirement,
			ArmorMax: armor.MaxDurability, ProtectionLevel: armor.ProtectionLevel, Coverage: armor.Coverage,
		}
	}

	var ammos []models.AmmoDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&ammos).Error; err != nil {
		return fmt.Errorf("读取弹药商品: %w", err)
	}
	for _, ammo := range ammos {
		if _, exists := r.items[ammo.ID]; exists {
			continue
		}
		r.items[ammo.ID] = Item{
			ID: ammo.ID, Name: ammo.Name, Kind: "ammo", Price: ammo.Price, Slots: 1,
			MerchantCategory: ammo.MerchantCategory, RepRequirement: ammo.RepRequirement,
			RoundsPerSlot: ammo.RoundsPerSlot, AmmoLevel: ammo.Level, CaliberID: ammo.CaliberID,
		}
	}

	var consumables []models.ConsumableDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&consumables).Error; err != nil {
		return fmt.Errorf("读取补给商品: %w", err)
	}
	for _, consumable := range consumables {
		if _, exists := r.items[consumable.ID]; exists {
			continue
		}
		r.items[consumable.ID] = Item{
			ID: consumable.ID, Name: consumable.Name, Kind: "consumable", Desc: consumable.Desc, Price: consumable.Price,
			Weight: consumable.Weight, Slots: consumable.Slots, MerchantCategory: consumable.MerchantCategory, RepRequirement: consumable.RepRequirement,
		}
	}

	var chestRigs []models.ChestRigDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&chestRigs).Error; err != nil {
		return fmt.Errorf("读取胸挂商品: %w", err)
	}
	for _, chestRig := range chestRigs {
		if _, exists := r.items[chestRig.ID]; exists {
			continue
		}
		r.items[chestRig.ID] = Item{
			ID: chestRig.ID, Name: chestRig.Name, Kind: "chestrig", Price: chestRig.Price, Weight: chestRig.Weight, Slots: chestRig.Slots,
			MerchantCategory: chestRig.MerchantCategory, RepRequirement: chestRig.RepRequirement,
			AddSlots: chestRig.AddSlots, AddWeight: chestRig.AddWeight,
		}
	}

	var backpacks []models.BackpackDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&backpacks).Error; err != nil {
		return fmt.Errorf("读取背包商品: %w", err)
	}
	for _, backpack := range backpacks {
		if _, exists := r.items[backpack.ID]; exists {
			continue
		}
		r.items[backpack.ID] = Item{
			ID: backpack.ID, Name: backpack.Name, Kind: "backpack", Price: backpack.Price, Weight: backpack.Weight, Slots: backpack.Slots,
			MerchantCategory: backpack.MerchantCategory, RepRequirement: backpack.RepRequirement,
			AddSlots: backpack.AddSlots, AddWeight: backpack.AddWeight,
		}
	}

	var helmets []models.HelmetDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&helmets).Error; err != nil {
		return fmt.Errorf("读取头盔商品: %w", err)
	}
	for _, helmet := range helmets {
		if _, exists := r.items[helmet.ID]; exists {
			continue
		}
		r.items[helmet.ID] = Item{
			ID: helmet.ID, Name: helmet.Name, Kind: "helmet", Price: helmet.Price, Weight: helmet.Weight, Slots: helmet.Slots,
			MerchantCategory: helmet.MerchantCategory, RepRequirement: helmet.RepRequirement,
			Protect: helmet.Protect, Coverage: helmet.Coverage, ArmorMax: helmet.MaxDurability,
		}
	}

	var headsets []models.HeadsetDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&headsets).Error; err != nil {
		return fmt.Errorf("读取耳机商品: %w", err)
	}
	for _, headset := range headsets {
		if _, exists := r.items[headset.ID]; exists {
			continue
		}
		r.items[headset.ID] = Item{
			ID: headset.ID, Name: headset.Name, Kind: "headset", Price: headset.Price, Weight: headset.Weight, Slots: headset.Slots,
			MerchantCategory: headset.MerchantCategory, RepRequirement: headset.RepRequirement, HearingLevel: headset.HearingLevel,
		}
	}

	var lootItems []models.LootItemDef
	if err := r.db.Where("id IN ?", itemIDs).Find(&lootItems).Error; err != nil {
		return fmt.Errorf("读取战利品商品: %w", err)
	}
	for _, loot := range lootItems {
		if _, exists := r.items[loot.ID]; exists {
			continue
		}
		r.items[loot.ID] = Item{
			ID: loot.ID, Name: loot.Name, Kind: "loot", Category: loot.Category, Desc: loot.Desc, Price: loot.Price,
			Weight: loot.Weight, Slots: loot.Slots, DropWeight: loot.DropWeight,
			MerchantCategory: loot.MerchantCategory, RepRequirement: loot.RepRequirement,
		}
	}
	return nil
}

func uniqueIDs(itemIDs []string) []string {
	seen := make(map[string]struct{}, len(itemIDs))
	result := make([]string, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		if itemID == "" {
			continue
		}
		if _, exists := seen[itemID]; exists {
			continue
		}
		seen[itemID] = struct{}{}
		result = append(result, itemID)
	}
	return result
}
