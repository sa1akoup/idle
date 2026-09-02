// 弹药种子：按口径生成 N1-N6 可用等级，并初始化玩家可直接携带的按发库存。
package config

import (
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

type ammoCaliberSeed struct {
	ID        string
	Label     string
	Levels    []int
	BasePrice int
}

var ammoCaliberSeeds = []ammoCaliberSeed{
	{ID: "9x18", Label: "9×18mm", Levels: []int{1, 2, 3}, BasePrice: 1},
	{ID: "9x19", Label: "9×19mm", Levels: []int{1, 2, 3, 4}, BasePrice: 1},
	{ID: "45acp", Label: ".45 ACP", Levels: []int{1, 2, 3, 4}, BasePrice: 1},
	{ID: "46x30", Label: "4.6×30mm", Levels: []int{2, 3, 4, 5, 6}, BasePrice: 2},
	{ID: "57x28", Label: "5.7×28mm", Levels: []int{1, 2, 3, 4}, BasePrice: 2},
	{ID: "12g", Label: "12 Gauge", Levels: []int{1, 2, 3, 4}, BasePrice: 2},
	{ID: "545x39", Label: "5.45×39mm", Levels: []int{1, 2, 3, 4, 5, 6}, BasePrice: 1},
	{ID: "556x45", Label: "5.56×45mm", Levels: []int{1, 2, 3, 4, 5, 6}, BasePrice: 1},
	{ID: "762x39", Label: "7.62×39mm", Levels: []int{2, 3, 4, 5, 6}, BasePrice: 2},
	{ID: "762x51", Label: "7.62×51mm", Levels: []int{2, 4, 5, 6}, BasePrice: 3},
	{ID: "762x54r", Label: "7.62×54R", Levels: []int{3, 4, 5, 6}, BasePrice: 3},
	{ID: "9x39", Label: "9×39mm", Levels: []int{2, 3, 4, 5, 6}, BasePrice: 2},
}

var ammoLevelProfiles = map[int]struct {
	FleshDamageMultiplier float64
	ArmorDamageMultiplier float64
	PriceMultiplier       int
	RepRequirement        int
}{
	1: {FleshDamageMultiplier: 1.20, ArmorDamageMultiplier: 0.50, PriceMultiplier: 1},
	2: {FleshDamageMultiplier: 1.10, ArmorDamageMultiplier: 0.65, PriceMultiplier: 2},
	3: {FleshDamageMultiplier: 1.05, ArmorDamageMultiplier: 0.80, PriceMultiplier: 4, RepRequirement: 15},
	4: {FleshDamageMultiplier: 1.00, ArmorDamageMultiplier: 1.00, PriceMultiplier: 8, RepRequirement: 30},
	5: {FleshDamageMultiplier: 0.95, ArmorDamageMultiplier: 1.20, PriceMultiplier: 16, RepRequirement: 30},
	6: {FleshDamageMultiplier: 0.90, ArmorDamageMultiplier: 1.40, PriceMultiplier: 28, RepRequirement: 40},
}

func seedAmmo(db *gorm.DB) error {
	for _, caliber := range ammoCaliberSeeds {
		for _, level := range caliber.Levels {
			profile := ammoLevelProfiles[level]
			ammo := models.AmmoDef{
				ID:                    fmt.Sprintf("ammo_%s_n%d", caliber.ID, level),
				Name:                  fmt.Sprintf("%s N%d级弹", caliber.Label, level),
				CaliberID:             caliber.ID,
				Level:                 level,
				FleshDamageMultiplier: profile.FleshDamageMultiplier,
				ArmorDamageMultiplier: profile.ArmorDamageMultiplier,
				Price:                 caliber.BasePrice * profile.PriceMultiplier,
				RoundsPerSlot:         999,
				MerchantCategory:      "weapon",
				RepRequirement:        profile.RepRequirement,
			}
			if err := upsertSeedDef(db, &ammo, ammo.ID); err != nil {
				return err
			}
		}
	}

	for _, inventory := range initialAmmoInventory() {
		if err := upsertInventory(db, inventory); err != nil {
			return err
		}
	}
	return nil
}

func initialAmmoInventory() []models.Inventory {
	// 与三套初始预设配套：直接可用的最低级无门槛弹药（N1/N2），各备 60 发（够两轮预设携弹）。
	return []models.Inventory{
		{ItemID: "ammo_762x39_n2", Name: "7.62×39mm N2级弹", Kind: "ammo", Quantity: 60, Price: 4, Slots: 1, MerchantCategory: "weapon", RepRequirement: 0},
		{ItemID: "ammo_9x19_n1", Name: "9×19mm N1级弹", Kind: "ammo", Quantity: 60, Price: 1, Slots: 1, MerchantCategory: "weapon", RepRequirement: 0},
		{ItemID: "ammo_12g_n1", Name: "12 Gauge N1级弹", Kind: "ammo", Quantity: 60, Price: 2, Slots: 1, MerchantCategory: "weapon", RepRequirement: 0},
	}
}
