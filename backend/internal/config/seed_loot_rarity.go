package config

import "idle/internal/models"

const (
	lootRarityCommon    = "common"
	lootRarityUncommon  = "uncommon"
	lootRarityRare      = "rare"
	lootRaritySuperrare = "superrare"
	lootRarityLegendary = "legendary"
)

// 原版物品 _props.Rarity / SpawnChance 对照：
// Common 垃圾件；Rare 如 Tetriz SpawnChance 0.4；Superrare 如 Virtex 0.05。
// LEDX、比特币属于局内终局件，只在高价值容器出现。
var lootRarityByID = map[string]string{
	"ledx":                     lootRarityLegendary,
	"physical_bitcoin":         lootRarityLegendary,
	"virtex":                   lootRarityLegendary,
	"graphics_card":            lootRaritySuperrare,
	"tetriz":                   lootRaritySuperrare,
	"vpx_flash_storage_module": lootRaritySuperrare,
	"military_circuit_board":   lootRaritySuperrare,
	"military_battery":         lootRaritySuperrare,
	"military_power_filter":    lootRaritySuperrare,
	"military_flash_drive":     lootRaritySuperrare,
	"portable_defibrillator":   lootRaritySuperrare,
	"ophthalmoscope":           lootRaritySuperrare,
	"aceso_analyzer":           lootRaritySuperrare,
	"golden_rooster":           lootRaritySuperrare,
	"bronze_lion":              lootRaritySuperrare,
	"lucky_scav_junk_box":      lootRaritySuperrare,
	"roler_watch":              lootRaritySuperrare,
	"gold_skull_ring":          lootRaritySuperrare,
	"golden_egg":               lootRaritySuperrare,
	"chain_prokill":            lootRaritySuperrare,
	"moonshine":                lootRaritySuperrare,
	"terragroup_blue_folder":   lootRaritySuperrare,
	"sas_drive":                lootRaritySuperrare,
	"secure_flash_drive":       lootRaritySuperrare,
	"key_customs_office":       lootRarityUncommon,
	"key_clinic_pharmacy":      lootRarityUncommon,
	"key_warehouse_office":     lootRarityUncommon,
	"salewa":                   lootRarityUncommon,
	"ifak":                     lootRarityUncommon,
	"car_battery":              lootRarityRare,
	"spark_plug":               lootRarityUncommon,
	"printed_circuit_board":    lootRarityUncommon,
	"gas_analyzer":             lootRarityRare,
	"intelligence_folder":      lootRarityRare,
	"gold_chain":               lootRarityRare,
	"golden_neck_chain":        lootRarityRare,
	"chainlet":                 lootRarityRare,
	"duck_figurine":            lootRarityRare,
	"horse_figurine":           lootRarityRare,
	"cat_figurine":             lootRarityRare,
	"antique_teapot":           lootRaritySuperrare,
	"antique_vase":             lootRaritySuperrare,
	"military_cable":           lootRaritySuperrare,
	"set_of_tools":             lootRarityRare,
	"electric_drill":           lootRarityRare,
	"analog_thermometer":       lootRarityRare,
	"pressure_gauge":           lootRarityRare,
	"propane_tank":             lootRarityRare,
	"metal_fuel_tank":          lootRarityRare,
	"cms":                      lootRarityRare,
	"medical_tools":            lootRarityRare,
	"silver_badge":             lootRaritySuperrare,
}

func applyLootRarity(items []models.LootItemDef) {
	for i, item := range items {
		rarity := lootRarityByID[item.ID]
		if rarity == "" {
			rarity = lootRarityFromPrice(item.Price)
		}
		items[i].Rarity = rarity
		items[i].DropWeight = lootRarityDropWeight(rarity)
	}
}

func lootRarityFromPrice(price int) string {
	switch {
	case price <= 100:
		return lootRarityCommon
	case price <= 250:
		return lootRarityUncommon
	case price <= 550:
		return lootRarityRare
	default:
		return lootRaritySuperrare
	}
}

func lootRarityDropWeight(rarity string) int {
	switch rarity {
	case lootRarityLegendary:
		return 1
	case lootRaritySuperrare:
		return 2
	case lootRarityRare:
		return 4
	case lootRarityUncommon:
		return 12
	default:
		return 40
	}
}
