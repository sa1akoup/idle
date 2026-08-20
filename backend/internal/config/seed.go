package config

import (
	"errors"

	"idle/internal/models"

	"gorm.io/gorm"
)

// Seed 初始化全部基础数据，按类别调用各数据文件。
func Seed(db *gorm.DB) error {
	if err := seedPlayer(db); err != nil {
		return err
	}
	if err := seedEquipment(db); err != nil {
		return err
	}
	if err := seedConsumables(db); err != nil {
		return err
	}
	if err := seedLoot(db); err != nil {
		return err
	}
	if err := seedMaterials(db); err != nil {
		return err
	}
	if err := seedMap(db); err != nil {
		return err
	}
	if err := seedContainers(db); err != nil {
		return err
	}
	if err := seedUnits(db); err != nil {
		return err
	}
	if err := seedEvents(db); err != nil {
		return err
	}
	if err := seedMerchants(db); err != nil {
		return err
	}
	if err := seedLoadout(db); err != nil {
		return err
	}
	return nil
}

// seedPlayer 玩家角色
func seedPlayer(db *gorm.DB) error {
	player := models.Character{
		ID: models.PlayerCharacterID, Name: "幸存者", Desc: "在封锁区中寻找生路的行动员",
		Strength: 55, Agility: 55, Intellect: 55, Charisma: 50,
		Stealth: 45, Perception: 50, Negotiation: 40, Luck: 45,
		Survival: 55, Resist: 50, Engineering: 45, Medical: 40,
		MeleeProf: 35, PistolProf: 45, SMGProf: 35, ShotgunProf: 30, RifleProf: 40, SniperProf: 25,
		Trait: "适应：未知区域中的首次判定更加稳定", Injury: "none",
	}
	return db.FirstOrCreate(&player, models.Character{ID: models.PlayerCharacterID}).Error
}

// seedLoadout 装备配置与现金、护甲实例（仅 FirstOrCreate，避免覆盖玩家已保存配置）。
func seedLoadout(db *gorm.DB) error {
	loadout := models.PlayerLoadout{
		ID: models.PlayerLoadoutID, CharacterID: models.PlayerCharacterID,
		WeaponID: "rifle_ak", ArmorID: "light_01", ChestRigID: "chestrig_01", BackpackID: "backpack_01", HelmetID: "helmet_01", HeadsetID: "headset_01",
		Consumables:    []string{"smoke", "toolkit"},
		PresetWeaponID: "rifle_ak", PresetArmorID: "light_01", PresetChestRigID: "chestrig_01", PresetBackpackID: "backpack_01", PresetHelmetID: "helmet_01", PresetHeadsetID: "headset_01",
		PresetName: "标准突击", PresetConsumables: []string{"smoke", "toolkit"},
		Preset2WeaponID: "pistol_glock", Preset2ArmorID: "light_02", Preset2Name: "轻装渗透", Preset2Consumables: []string{"bandage", "ammo_box"},
		Preset3WeaponID: "shotgun_m870", Preset3ArmorID: "heavy_01", Preset3Name: "重装攻坚", Preset3Consumables: []string{"bandage", "medkit"},
	}
	if err := db.FirstOrCreate(&loadout, models.PlayerLoadout{ID: models.PlayerLoadoutID}).Error; err != nil {
		return err
	}

	if err := upsertInventory(db, models.Inventory{ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 5000, Price: 1}); err != nil {
		return err
	}

	armInstances := []models.ArmorInstance{
		{ID: 1, ArmorID: "light_01", MaxDurability: 100, CurDurability: 100, RepairCount: 0, Status: "normal"},
		{ID: 2, ArmorID: "heavy_01", MaxDurability: 150, CurDurability: 150, RepairCount: 0, Status: "normal"},
	}
	for _, ai := range armInstances {
		if err := db.FirstOrCreate(&ai, ai.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

// upsertInventory 仓库行的新增或信息更新（不覆盖玩家数量）。
func upsertInventory(db *gorm.DB, inv models.Inventory) error {
	var stored models.Inventory
	err := db.Where("item_id = ? AND raid_extract = ?", inv.ItemID, inv.RaidExtract).First(&stored).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return db.Create(&inv).Error
	case err != nil:
		return err
	default:
		return db.Model(&stored).Updates(map[string]interface{}{
			"name": inv.Name, "kind": inv.Kind, "category": inv.Category, "price": inv.Price,
			"weight": inv.Weight, "slots": inv.Slots,
			"merchant_category": inv.MerchantCategory, "rep_requirement": inv.RepRequirement,
		}).Error
	}
}
