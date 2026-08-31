// 装备预设模型与映射：集中处理三套预设的请求字段和持久化字段转换。
package service

import "idle/internal/models"

// SaveLoadoutReq 保存当前装备和 3 套失能后补购预设。
type SaveLoadoutReq struct {
	WeaponID           string   `json:"weaponId" binding:"required"`
	ArmorID            string   `json:"armorId" binding:"required"`
	ChestRigID         string   `json:"chestRigId"`
	BackpackID         string   `json:"backpackId"`
	HelmetID           string   `json:"helmetId"`
	HeadsetID          string   `json:"headsetId"`
	Consumables        []string `json:"consumables"`
	PresetWeaponID     string   `json:"presetWeaponId"`
	PresetArmorID      string   `json:"presetArmorId"`
	PresetChestRigID   string   `json:"presetChestRigId"`
	PresetBackpackID   string   `json:"presetBackpackId"`
	PresetHelmetID     string   `json:"presetHelmetId"`
	PresetHeadsetID    string   `json:"presetHeadsetId"`
	PresetConsumables  []string `json:"presetConsumables"`
	PresetAmmoID       string   `json:"presetAmmoId"`
	PresetAmmoRounds   int      `json:"presetAmmoRounds"`
	PresetName         string   `json:"presetName"`
	Preset2WeaponID    string   `json:"preset2WeaponId"`
	Preset2ArmorID     string   `json:"preset2ArmorId"`
	Preset2ChestRigID  string   `json:"preset2ChestRigId"`
	Preset2BackpackID  string   `json:"preset2BackpackId"`
	Preset2HelmetID    string   `json:"preset2HelmetId"`
	Preset2HeadsetID   string   `json:"preset2HeadsetId"`
	Preset2Consumables []string `json:"preset2Consumables"`
	Preset2AmmoID      string   `json:"preset2AmmoId"`
	Preset2AmmoRounds  int      `json:"preset2AmmoRounds"`
	Preset2Name        string   `json:"preset2Name"`
	Preset3WeaponID    string   `json:"preset3WeaponId"`
	Preset3ArmorID     string   `json:"preset3ArmorId"`
	Preset3ChestRigID  string   `json:"preset3ChestRigId"`
	Preset3BackpackID  string   `json:"preset3BackpackId"`
	Preset3HelmetID    string   `json:"preset3HelmetId"`
	Preset3HeadsetID   string   `json:"preset3HeadsetId"`
	Preset3Consumables []string `json:"preset3Consumables"`
	Preset3AmmoID      string   `json:"preset3AmmoId"`
	Preset3AmmoRounds  int      `json:"preset3AmmoRounds"`
	Preset3Name        string   `json:"preset3Name"`
}

// PresetNameOf 返回第 N 套（1-3）预设名称。
func PresetNameOf(loadout *models.PlayerLoadout, index int) string {
	switch index {
	case 2:
		return loadout.Preset2Name
	case 3:
		return loadout.Preset3Name
	default:
		return loadout.PresetName
	}
}

func presetOfReq(req SaveLoadoutReq, index int) (weaponID, armorID string, consumables []string, equip []string) {
	switch index {
	case 2:
		return req.Preset2WeaponID, req.Preset2ArmorID, req.Preset2Consumables,
			[]string{req.Preset2ChestRigID, req.Preset2BackpackID, req.Preset2HelmetID, req.Preset2HeadsetID}
	case 3:
		return req.Preset3WeaponID, req.Preset3ArmorID, req.Preset3Consumables,
			[]string{req.Preset3ChestRigID, req.Preset3BackpackID, req.Preset3HelmetID, req.Preset3HeadsetID}
	default:
		return req.PresetWeaponID, req.PresetArmorID, req.PresetConsumables,
			[]string{req.PresetChestRigID, req.PresetBackpackID, req.PresetHelmetID, req.PresetHeadsetID}
	}
}

func presetAmmoOfReq(req SaveLoadoutReq, index int) (string, int) {
	switch index {
	case 2:
		return req.Preset2AmmoID, req.Preset2AmmoRounds
	case 3:
		return req.Preset3AmmoID, req.Preset3AmmoRounds
	default:
		return req.PresetAmmoID, req.PresetAmmoRounds
	}
}

func allEmpty(values []string) bool {
	for _, value := range values {
		if value != "" {
			return false
		}
	}
	return true
}
