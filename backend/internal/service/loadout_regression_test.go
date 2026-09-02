package service

// 装备配置回归测试：覆盖卸下武器/护甲、损坏护甲仍可装备、空护甲读取出征耐久的场景。

import (
	"testing"

	"idle/internal/models"

	"gorm.io/gorm"
)

// newLoadoutRegressionDB 构造带完整初始配装（武器+护甲+正常实例）的测试库。
func newLoadoutRegressionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newInventoryTestDB(t)
	var character models.Character
	if err := db.Where("user_id = ?", models.DefaultUserID).First(&character).Error; err != nil {
		t.Fatalf("读取测试角色: %v", err)
	}
	if err := db.Create(&models.PlayerLoadout{
		UserID: models.DefaultUserID, CharacterID: character.ID,
		WeaponID: "weapon", ArmorID: "armor",
	}).Error; err != nil {
		t.Fatalf("创建初始配装: %v", err)
	}
	for _, inv := range []*models.Inventory{
		{UserID: models.DefaultUserID, ItemID: "weapon", Kind: "weapon", Quantity: 1},
		{UserID: models.DefaultUserID, ItemID: "armor", Kind: "armor", Quantity: 1},
	} {
		if err := db.Create(inv).Error; err != nil {
			t.Fatalf("创建仓库装备 %s: %v", inv.ItemID, err)
		}
	}
	if err := db.Create(&models.ArmorInstance{
		UserID: models.DefaultUserID, ArmorID: "armor",
		MaxDurability: 80, CurDurability: 80, Status: "normal",
	}).Error; err != nil {
		t.Fatalf("创建护甲实例: %v", err)
	}
	var armorInstance models.ArmorInstance
	if err := db.Where("user_id = ? AND armor_id = ?", models.DefaultUserID, "armor").First(&armorInstance).Error; err != nil {
		t.Fatalf("读取护甲实例: %v", err)
	}
	if err := db.Model(&models.PlayerLoadout{}).Where("user_id = ?", models.DefaultUserID).
		Update("armor_instance_id", armorInstance.ID).Error; err != nil {
		t.Fatalf("写入护甲实例引用: %v", err)
	}
	return db
}

func TestSaveLoadoutAllowsUnequipArmorAndWeapon(t *testing.T) {
	db := newLoadoutRegressionDB(t)

	// 卸下护甲：armorId 为空应保存成功，且不要求仓库仍有护甲实例可用。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "",
	}); err != nil {
		t.Fatalf("卸下护甲保存失败: %v", err)
	}
	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取配装: %v", err)
	}
	if loadout.ArmorID != "" {
		t.Fatalf("护甲应为空，实际 %q", loadout.ArmorID)
	}
	if loadout.ArmorInstanceID != 0 {
		t.Fatalf("卸下护甲后护甲实例引用应清零，实际 %d", loadout.ArmorInstanceID)
	}

	// 继续卸下武器：weaponId 为空同样应保存成功。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "", ArmorID: "",
	}); err != nil {
		t.Fatalf("卸下武器保存失败: %v", err)
	}
	loadout, err = GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取配装: %v", err)
	}
	if loadout.WeaponID != "" || loadout.ArmorID != "" {
		t.Fatalf("武器与护甲应为空，实际 weapon=%q armor=%q", loadout.WeaponID, loadout.ArmorID)
	}
}

func TestSaveLoadoutAcceptsBrokenArmorInstance(t *testing.T) {
	db := newLoadoutRegressionDB(t)

	// 场景复刻：出征后护甲耐久归零被标记 broken，配装仍保留该护甲。
	if err := db.Model(&models.ArmorInstance{}).
		Where("user_id = ? AND armor_id = ?", models.DefaultUserID, "armor").
		Updates(map[string]interface{}{"cur_durability": 0, "status": "broken"}).Error; err != nil {
		t.Fatalf("标记护甲损坏: %v", err)
	}

	// 损坏护甲仍应可保存在当前配装（只失去防护等级，不阻断换装/出征）。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "armor",
	}); err != nil {
		t.Fatalf("穿着损坏护甲保存失败: %v", err)
	}

	// 同一状态下卸下护甲也应成功。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "",
	}); err != nil {
		t.Fatalf("卸下损坏护甲失败: %v", err)
	}
}

func TestFindCurrentArmorInstanceHandlesEmptyAndBroken(t *testing.T) {
	db := newLoadoutRegressionDB(t)

	// 未穿护甲：返回 nil 而非报错，允许无甲出征。
	instance, err := findCurrentArmorInstance(db, models.DefaultUserID, "")
	if err != nil {
		t.Fatalf("空护甲读取出征耐久报错: %v", err)
	}
	if instance != nil {
		t.Fatalf("空护甲应返回 nil 实例")
	}

	// 仅损坏实例：仍可读取，出征耐久按 0 处理。
	if err := db.Model(&models.ArmorInstance{}).
		Where("user_id = ? AND armor_id = ?", models.DefaultUserID, "armor").
		Updates(map[string]interface{}{"cur_durability": 0, "status": "broken"}).Error; err != nil {
		t.Fatalf("标记护甲损坏: %v", err)
	}
	instance, err = findCurrentArmorInstance(db, models.DefaultUserID, "armor")
	if err != nil {
		t.Fatalf("损坏护甲读取出征耐久报错: %v", err)
	}
	if instance == nil || instance.CurDurability != 0 {
		t.Fatalf("损坏护甲实例不符: %+v", instance)
	}

	// 同时存在损坏与正常实例时优先取正常实例（normal 顺序在前）。
	if err := db.Create(&models.ArmorInstance{
		UserID: models.DefaultUserID, ArmorID: "armor",
		MaxDurability: 80, CurDurability: 80, Status: "normal",
	}).Error; err != nil {
		t.Fatalf("创建正常护甲实例: %v", err)
	}
	instance, err = findCurrentArmorInstance(db, models.DefaultUserID, "armor")
	if err != nil {
		t.Fatalf("双实例读取出征耐久报错: %v", err)
	}
	if instance == nil || instance.Status != "normal" {
		t.Fatalf("应优先选中正常实例，实际 %+v", instance)
	}
}

// TestSaveLoadoutAcceptsCarriedAmmoCells 验证随身弹药槽：最多 4 格、同弹药可多槽、
// 发数 1-60 且不低于单轮消耗、仓库有对应实弹；超上限与发数越界被拒绝。
func TestSaveLoadoutAcceptsCarriedAmmoCells(t *testing.T) {
	db := newLoadoutRegressionDB(t)
	for _, ammo := range []*models.AmmoDef{
		{ID: "ammo_cal_n1", Name: "N1", CaliberID: "cal_t", Level: 1, RoundsPerSlot: 999},
		{ID: "ammo_cal_n2", Name: "N2", CaliberID: "cal_t", Level: 2, RoundsPerSlot: 999},
	} {
		if err := db.Create(ammo).Error; err != nil {
			t.Fatalf("创建弹药: %v", err)
		}
	}
	if err := db.Model(&models.WeaponDef{}).Where("id = ?", "weapon").
		Updates(map[string]interface{}{"caliber_id": "cal_t", "ammo_per_round": 3}).Error; err != nil {
		t.Fatalf("配置测试武器口径: %v", err)
	}
	for _, ammoID := range []string{"ammo_cal_n1", "ammo_cal_n2"} {
		quantity := 120
		if ammoID == "ammo_cal_n1" {
			quantity = 100
		}
		if err := db.Create(&models.Inventory{UserID: models.DefaultUserID, ItemID: ammoID, Kind: "ammo", Quantity: quantity}).Error; err != nil {
			t.Fatalf("创建弹药库存: %v", err)
		}
	}
	cells := []models.AmmoCell{
		{AmmoID: "ammo_cal_n1", Rounds: 30},
		{AmmoID: "ammo_cal_n2", Rounds: 30},
		{AmmoID: "ammo_cal_n2", Rounds: 30}, // 同等级弹药允许两格
		{},
	}
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "armor", CarriedAmmo: cells,
	}); err != nil {
		t.Fatalf("保存携带弹药失败: %v", err)
	}
	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取配装: %v", err)
	}
	if len(loadout.CarriedAmmo) != 4 || loadout.CarriedAmmo[1].AmmoID != "ammo_cal_n2" || loadout.CarriedAmmo[1].Rounds != 30 {
		t.Fatalf("随身弹药槽持久化异常: %+v", loadout.CarriedAmmo)
	}

	// 超过 4 格拒绝。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "armor", CarriedAmmo: append(cells, models.AmmoCell{AmmoID: "ammo_cal_n1", Rounds: 10}),
	}); err == nil {
		t.Fatal("超过 4 格携带弹药应被拒绝")
	}
	// 发数超过 60 拒绝。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "armor", CarriedAmmo: []models.AmmoCell{{AmmoID: "ammo_cal_n1", Rounds: 61}},
	}); err == nil {
		t.Fatal("单格超过 60 发应被拒绝")
	}
	// 仓库实弹不足拒绝（同弹药两格合计 100 发，库存仅 120）。
	if _, err := SavePlayerLoadoutForUser(db, models.DefaultUserID, SaveLoadoutReq{
		WeaponID: "weapon", ArmorID: "armor", CarriedAmmo: []models.AmmoCell{{AmmoID: "ammo_cal_n1", Rounds: 60}, {AmmoID: "ammo_cal_n1", Rounds: 60}},
	}); err == nil {
		t.Fatal("仓库弹药不足应被拒绝")
	}
}

// TestFillDefaultPresetAmmoTx 验证存量数据升级为"弹药为空"的预设回填最低级无门槛弹药 30 发，
// 且不覆盖已手工配置过弹药的预设、不给近战武器配弹。
func TestFillDefaultPresetAmmoTx(t *testing.T) {
	db := newLoadoutRegressionDB(t)

	// 补充带口径的武器与弹药定义：cal_test 有 N1/N2（无门槛）与 N3（门槛 15）。
	if err := db.Create(&models.WeaponDef{ID: "rifle_test", Name: "测试步枪", CaliberID: "cal_test", AmmoPerRound: 3}).Error; err != nil {
		t.Fatalf("创建测试武器: %v", err)
	}
	for _, ammo := range []*models.AmmoDef{
		{ID: "ammo_cal_n1", Name: "N1", CaliberID: "cal_test", Level: 1, RepRequirement: 0},
		{ID: "ammo_cal_n2", Name: "N2", CaliberID: "cal_test", Level: 2, RepRequirement: 0},
		{ID: "ammo_cal_n3", Name: "N3", CaliberID: "cal_test", Level: 3, RepRequirement: 15},
	} {
		if err := db.Create(ammo).Error; err != nil {
			t.Fatalf("创建测试弹药 %s: %v", ammo.ID, err)
		}
	}
	var character models.Character
	if err := db.Where("user_id = ?", models.DefaultUserID).Order("id asc").First(&character).Error; err != nil {
		t.Fatalf("读取角色: %v", err)
	}
	// 预设1 弹药为空 → 应回填 N1（同口径最低级且无门槛）30 发；
	// 预设2 已配置 N3 → 保持原样；
	// 预设3 为无口径武器（视同近战）→ 保持为空。
	if err := db.Model(&models.PlayerLoadout{}).Where("user_id = ?", models.DefaultUserID).
		Updates(map[string]interface{}{
			"preset_weapon_id": "rifle_test", "preset_ammo_id": "", "preset_ammo_rounds": 0,
			"preset2_weapon_id": "rifle_test", "preset2_ammo_id": "ammo_cal_n3", "preset2_ammo_rounds": 60,
			"preset3_weapon_id": "weapon", "preset3_ammo_id": "", "preset3_ammo_rounds": 0,
		}).Error; err != nil {
		t.Fatalf("准备测试预设弹药: %v", err)
	}

	if err := fillDefaultPresetAmmoTx(db, models.DefaultUserID); err != nil {
		t.Fatalf("回填预设弹药失败: %v", err)
	}
	loadout, err := GetPlayerLoadoutForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取配装: %v", err)
	}
	if loadout.PresetAmmoID != "ammo_cal_n1" || loadout.PresetAmmoRounds != 30 {
		t.Fatalf("预设1 默认弹药不符: %s/%d", loadout.PresetAmmoID, loadout.PresetAmmoRounds)
	}
	if loadout.Preset2AmmoID != "ammo_cal_n3" || loadout.Preset2AmmoRounds != 60 {
		t.Fatalf("预设2 已配置弹药被覆盖: %s/%d", loadout.Preset2AmmoID, loadout.Preset2AmmoRounds)
	}
	if loadout.Preset3AmmoID != "" || loadout.Preset3AmmoRounds != 0 {
		t.Fatalf("预设3 无口径武器不应配弹: %s/%d", loadout.Preset3AmmoID, loadout.Preset3AmmoRounds)
	}
	// 回填预设时需配套补足仓库余量，否则开局按预设扣弹会缺货；未回填的弹药不动。
	stockN1, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_cal_n1")
	if err != nil || stockN1 != defaultPresetAmmoStock {
		t.Fatalf("回填后 N1 库存 = %d（期望 %d）: %v", stockN1, defaultPresetAmmoStock, err)
	}
	stockN3, err := ammoInventoryQuantity(db, models.DefaultUserID, "ammo_cal_n3")
	if err != nil || stockN3 != 0 {
		t.Fatalf("未回填的 N3 库存应保持 0，实际 %d: %v", stockN3, err)
	}
}
