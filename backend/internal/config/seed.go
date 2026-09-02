package config

import (
	"errors"
	"time"

	"idle/internal/models"

	"gorm.io/gorm"
)

// Seed 初始化全部基础数据，按类别调用各数据文件。
func Seed(db *gorm.DB) error {
	if err := seedUser(db); err != nil {
		return err
	}
	if err := seedPlayer(db); err != nil {
		return err
	}
	if err := seedEquipment(db); err != nil {
		return err
	}
	if err := seedAmmo(db); err != nil {
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
	if err := seedSurvivalDefinitions(db); err != nil {
		return err
	}
	if err := seedMap(db); err != nil {
		return err
	}
	if err := seedContainers(db); err != nil {
		return err
	}
	if err := seedEnemyTemplates(db); err != nil {
		return err
	}
	if err := seedEvents(db); err != nil {
		return err
	}
	if err := seedMerchants(db); err != nil {
		return err
	}
	if err := seedMerchantStates(db); err != nil {
		return err
	}
	if err := seedHideout(db); err != nil {
		return err
	}
	if err := seedCrafting(db); err != nil {
		return err
	}
	if err := seedLoadout(db); err != nil {
		return err
	}
	if err := seedSurvivalForUser(db, models.DefaultUserID); err != nil {
		return err
	}
	return nil
}

// seedPlayer 玩家角色
func seedPlayer(db *gorm.DB) error {
	return seedPlayerForUser(db, models.DefaultUserID)
}

func seedPlayerForUser(db *gorm.DB, userID uint) error {
	// 初始角色按白板配置：全部属性与武器熟练度为 0（成长交由后续系统），
	// 生命上限由力量动态计算，白板力量为 0 时上限为 90；能量/饮水保持满值。
	player := models.Character{
		UserID: userID, Name: "幸存者", Desc: "在封锁区中寻找生路的行动员",
		Strength: 0, Agility: 0, Intellect: 0, Charisma: 0,
		Stealth: 0, Perception: 0, Negotiation: 0, Luck: 0,
		Survival: 0, Resist: 0, Engineering: 0, Medical: 0,
		MeleeProf: 0, PistolProf: 0, SMGProf: 0, ShotgunProf: 0, RifleProf: 0, SniperProf: 0,
		HP: 90, Energy: 100, Hydration: 100, NeedsUpdatedAt: time.Now(),
	}
	return db.Where("user_id = ?", userID).FirstOrCreate(&player).Error
}

// seedUser 创建认证接口接入前的本地启动用户。
func seedUser(db *gorm.DB) error {
	user := models.User{Username: "local", Status: "active"}
	return db.Where("username = ?", user.Username).FirstOrCreate(&user).Error
}

// seedMerchantStates 为本地启动用户复制商人的初始好感度与解锁状态。
func seedMerchantStates(db *gorm.DB) error {
	return seedMerchantStatesForUser(db, models.DefaultUserID)
}

func seedMerchantStatesForUser(db *gorm.DB, userID uint) error {
	var merchants []models.MerchantDef
	if err := db.Order("sort_order asc, id asc").Find(&merchants).Error; err != nil {
		return err
	}
	for _, merchant := range merchants {
		state := models.UserMerchantState{
			UserID: userID, MerchantID: merchant.ID,
			Reputation: merchant.Reputation, Unlocked: merchant.Open,
		}
		if err := db.Where("user_id = ? AND merchant_id = ?", userID, merchant.ID).
			FirstOrCreate(&state).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedLoadout 装备配置与现金、护甲实例（仅 FirstOrCreate，避免覆盖玩家已保存配置）。
func seedLoadout(db *gorm.DB) error {
	return seedLoadoutForUser(db, models.DefaultUserID)
}

func seedLoadoutForUser(db *gorm.DB, userID uint) error {
	loadout := models.PlayerLoadout{
		UserID:   userID,
		WeaponID: "rifle_ak", ArmorID: "light_01", ChestRigID: "chestrig_01", BackpackID: "backpack_01", HelmetID: "helmet_01", HeadsetID: "headset_01",
		Consumables:    []string{"smoke", "toolkit"},
		PresetWeaponID: "rifle_ak", PresetArmorID: "light_01", PresetChestRigID: "chestrig_01", PresetBackpackID: "backpack_01", PresetHelmetID: "helmet_01", PresetHeadsetID: "headset_01",
		PresetName: "标准突击", PresetConsumables: []string{"smoke", "toolkit"}, PresetAmmoID: "ammo_762x39_n2", PresetAmmoRounds: 30,
		Preset2WeaponID: "pistol_glock", Preset2ArmorID: "light_02", Preset2Name: "轻装渗透", Preset2Consumables: []string{"bandage"}, Preset2AmmoID: "ammo_9x19_n1", Preset2AmmoRounds: 30,
		Preset3WeaponID: "shotgun_m870", Preset3ArmorID: "heavy_01", Preset3Name: "重装攻坚", Preset3Consumables: []string{"bandage", "medkit"}, Preset3AmmoID: "ammo_12g_n1", Preset3AmmoRounds: 30,
	}
	var player models.Character
	if err := db.Where("user_id = ?", userID).First(&player).Error; err != nil {
		return err
	}
	loadout.CharacterID = player.ID
	if err := db.Where("user_id = ? AND character_id = ?", userID, player.ID).
		FirstOrCreate(&loadout).Error; err != nil {
		return err
	}

	if err := upsertInventoryForUser(db, userID, models.Inventory{ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 5000, Price: 1}); err != nil {
		return err
	}

	armInstances := []models.ArmorInstance{
		{UserID: userID, ArmorID: "light_01", MaxDurability: 100, CurDurability: 100, RepairCount: 0, Status: "normal"},
		{UserID: userID, ArmorID: "heavy_01", MaxDurability: 150, CurDurability: 150, RepairCount: 0, Status: "normal"},
	}
	for _, ai := range armInstances {
		if err := db.Where("user_id = ? AND armor_id = ?", ai.UserID, ai.ArmorID).
			FirstOrCreate(&ai).Error; err != nil {
			return err
		}
	}
	return nil
}

// upsertInventory 仓库行的新增或信息更新（不覆盖玩家数量）。
func upsertInventory(db *gorm.DB, inv models.Inventory) error {
	return upsertInventoryForUser(db, models.DefaultUserID, inv)
}

func upsertInventoryForUser(db *gorm.DB, userID uint, inv models.Inventory) error {
	inv.UserID = userID
	var stored models.Inventory
	err := db.Where("user_id = ? AND item_id = ? AND raid_extract = ?", inv.UserID, inv.ItemID, inv.RaidExtract).First(&stored).Error
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
