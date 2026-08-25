// 快照模块：校验运行配置、规范化有序集合并生成可审计 hash。
package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CanonicalSnapshotJSON 规范化所有有序集合后输出 JSON；map key 由 encoding/json 按字典序编码。
func CanonicalSnapshotJSON(snapshot ScenarioSnapshot) ([]byte, error) {
	// 先通过 DTO JSON 做深拷贝，规范化排序不能改变调用方持有的快照。
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	var copied ScenarioSnapshot
	if err := json.Unmarshal(encoded, &copied); err != nil {
		return nil, err
	}
	normalized := normalizeSnapshot(copied)
	return json.Marshal(normalized)
}

func SnapshotHash(snapshot ScenarioSnapshot) (string, error) {
	encoded, err := CanonicalSnapshotJSON(snapshot)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func ValidateSnapshot(snapshot ScenarioSnapshot) error {
	if snapshot.SchemaVersion != SchemaVersion {
		return fmt.Errorf("不支持的场景快照 schema version %s", snapshot.SchemaVersion)
	}
	if snapshot.Map.ID == "" {
		return fmt.Errorf("场景快照缺少地图")
	}
	if err := ValidateMapGraph(snapshot.Map, snapshot.Nodes, snapshot.Edges, snapshot.ExtractionPoints); err != nil {
		return err
	}
	if len(snapshot.Styles) == 0 {
		return fmt.Errorf("场景快照缺少行动风格配置")
	}
	styleIDs := make(map[string]bool, len(snapshot.Styles))
	for _, style := range snapshot.Styles {
		if style.ID == "" || styleIDs[style.ID] {
			return fmt.Errorf("场景快照包含重复或空行动风格")
		}
		styleIDs[style.ID] = true
	}
	for _, requiredStyle := range []string{ActionStyleBalanced, ActionStyleStealth, ActionStyleAggressive, ActionStyleGreedy} {
		if !styleIDs[requiredStyle] {
			return fmt.Errorf("场景快照缺少行动风格 %s", requiredStyle)
		}
	}
	nodeIDs := make(map[string]bool, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		nodeIDs[node.ID] = true
		if node.EnemyID != "" {
			if _, ok := snapshot.Enemies[node.EnemyID]; !ok {
				return fmt.Errorf("节点 %s 引用不存在敌人 %s", node.ID, node.EnemyID)
			}
		}
		if node.EncounterRole != "" && len(snapshot.Events.EncounterPools[node.EncounterRole]) == 0 {
			return fmt.Errorf("节点 %s 引用不存在遭遇角色 %s", node.ID, node.EncounterRole)
		}
	}
	for _, assignment := range snapshot.NodeContainerAssignments {
		if !nodeIDs[assignment.NodeID] {
			return fmt.Errorf("节点容器配置引用不存在节点 %s", assignment.NodeID)
		}
		container, ok := snapshot.Containers[assignment.ContainerID]
		if !ok {
			return fmt.Errorf("节点 %s 引用不存在容器 %s", assignment.NodeID, assignment.ContainerID)
		}
		_ = container
	}
	for containerID, container := range snapshot.Containers {
		if container.ID != containerID {
			return fmt.Errorf("容器快照 key %s 与 ID %s 不一致", containerID, container.ID)
		}
		for _, rule := range container.Rules {
			found := false
			for _, item := range snapshot.LootItems {
				if item.Category == rule.ItemCategory {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("容器 %s 引用不存在 loot 分类 %s", containerID, rule.ItemCategory)
			}
		}
	}
	for weaponID, weapon := range snapshot.Weapons {
		if weapon.ID != weaponID {
			return fmt.Errorf("武器快照 key %s 与 ID %s 不一致", weaponID, weapon.ID)
		}
		if weapon.AmmoPerRound > 0 && weapon.CaliberID == "" {
			return fmt.Errorf("远程武器 %s 缺少口径", weaponID)
		}
		if weapon.AmmoPerRound < 0 {
			return fmt.Errorf("武器 %s 单轮耗弹无效", weaponID)
		}
	}
	for ammoID, ammo := range snapshot.Ammos {
		if ammo.ID != ammoID || ammo.CaliberID == "" {
			return fmt.Errorf("弹药快照 %s 的 ID 或口径无效", ammoID)
		}
		if ammo.Level < 1 || ammo.Level > 6 || ammo.FleshDamageMultiplier <= 0 || ammo.ArmorDamageMultiplier <= 0 || ammo.RoundsPerSlot <= 0 {
			return fmt.Errorf("弹药 %s 的等级或伤害配置无效", ammoID)
		}
		supply, ok := snapshot.AmmoSupplies[ammoID]
		if !ok {
			return fmt.Errorf("弹药 %s 缺少商人补给快照", ammoID)
		}
		if supply.AmmoID != ammoID || supply.CaliberID != ammo.CaliberID || supply.Level != ammo.Level || supply.UnitPrice <= 0 {
			return fmt.Errorf("弹药 %s 的商人补给快照无效", ammoID)
		}
		if supply.Available && ammo.Level > 4 {
			return fmt.Errorf("弹药 %s 超过武器商人最高出售等级", ammoID)
		}
	}
	for ammoID := range snapshot.AmmoSupplies {
		if _, ok := snapshot.Ammos[ammoID]; !ok {
			return fmt.Errorf("商人补给快照引用不存在弹药 %s", ammoID)
		}
	}
	for armorID, armor := range snapshot.Armors {
		if armor.ID != armorID || armor.ProtectionLevel < 1 || armor.ProtectionLevel > 6 || armor.MaxDurability <= 0 {
			return fmt.Errorf("护甲 %s 的等级或耐久无效", armorID)
		}
	}
	for enemyID, enemy := range snapshot.Enemies {
		weapon, ok := snapshot.Weapons[enemy.WeaponID]
		if !ok {
			return fmt.Errorf("敌人 %s 引用不存在武器 %s", enemyID, enemy.WeaponID)
		}
		if _, ok := snapshot.Armors[enemy.ArmorID]; !ok {
			return fmt.Errorf("敌人 %s 引用不存在护甲 %s", enemyID, enemy.ArmorID)
		}
		if enemy.BackpackContainerID != "" {
			if _, ok := snapshot.Containers[enemy.BackpackContainerID]; !ok {
				return fmt.Errorf("敌人 %s 引用不存在背包容器 %s", enemyID, enemy.BackpackContainerID)
			}
		}
		if weapon.AmmoPerRound > 0 {
			ammo, ok := snapshot.Ammos[enemy.AmmoID]
			if !ok {
				return fmt.Errorf("敌人 %s 引用不存在弹药 %s", enemyID, enemy.AmmoID)
			}
			if ammo.CaliberID != weapon.CaliberID {
				return fmt.Errorf("敌人 %s 的弹药口径与武器不匹配", enemyID)
			}
			if enemy.AmmoRounds < weapon.AmmoPerRound {
				return fmt.Errorf("敌人 %s 携弹不足以完成一次攻击", enemyID)
			}
		}
	}
	for index, preset := range snapshot.RecoveryPresets {
		if index != preset.Index || index < 1 || index > 3 {
			return fmt.Errorf("补购预设 %d 编号无效", index)
		}
		if preset.Loadout.WeaponID == "" && preset.Loadout.ArmorID == "" &&
			preset.Loadout.ChestRigID == "" && preset.Loadout.BackpackID == "" &&
			preset.Loadout.HelmetID == "" && preset.Loadout.HeadsetID == "" &&
			preset.AmmoID == "" && preset.AmmoRounds == 0 && len(preset.Consumables) == 0 && len(preset.Items) == 0 {
			continue
		}
		if preset.Loadout.WeaponID == "" || preset.Loadout.ArmorID == "" {
			return fmt.Errorf("补购预设 %d 缺少武器或护甲", index)
		}
		weapon, ok := snapshot.Weapons[preset.Loadout.WeaponID]
		if !ok {
			return fmt.Errorf("补购预设 %d 引用不存在武器 %s", index, preset.Loadout.WeaponID)
		}
		if weapon.AmmoPerRound > 0 {
			ammo, ok := snapshot.Ammos[preset.AmmoID]
			if !ok || ammo.CaliberID != weapon.CaliberID || preset.AmmoRounds < weapon.AmmoPerRound {
				return fmt.Errorf("补购预设 %d 的弹药与武器不匹配或数量不足", index)
			}
		} else if preset.AmmoID != "" || preset.AmmoRounds != 0 {
			return fmt.Errorf("补购预设 %d 的近战武器不能配置弹药", index)
		}
		for _, entry := range []struct {
			itemID string
			kind   string
		}{
			{preset.Loadout.WeaponID, "weapon"}, {preset.Loadout.ArmorID, "armor"},
			{preset.Loadout.ChestRigID, "chestrig"}, {preset.Loadout.BackpackID, "backpack"},
			{preset.Loadout.HelmetID, "helmet"}, {preset.Loadout.HeadsetID, "headset"},
		} {
			itemID, kind := entry.itemID, entry.kind
			if itemID == "" {
				continue
			}
			item, ok := snapshot.Items[itemID]
			if !ok || item.Kind != kind {
				return fmt.Errorf("补购预设 %d 的%s商品无效 %s", index, kind, itemID)
			}
		}
		for _, consumable := range preset.Consumables {
			if consumable.Quantity <= 0 {
				return fmt.Errorf("补购预设 %d 的补给 %s 数量无效", index, consumable.ItemID)
			}
			item, ok := snapshot.Items[consumable.ItemID]
			if !ok || item.Kind != "consumable" {
				return fmt.Errorf("补购预设 %d 的补给引用无效 %s", index, consumable.ItemID)
			}
		}
		for _, item := range preset.Items {
			if item.Quantity <= 0 {
				return fmt.Errorf("补购预设 %d 的物品 %s 数量无效", index, item.ItemID)
			}
			if _, ok := snapshot.Items[item.ItemID]; !ok {
				return fmt.Errorf("补购预设 %d 引用不存在商品 %s", index, item.ItemID)
			}
		}
	}
	bindingIDs := make(map[string]bool, len(snapshot.Events.Bindings))
	for _, entry := range snapshot.Events.Bindings {
		if entry.ID == "" || bindingIDs[entry.ID] {
			return fmt.Errorf("事件绑定缺少 ID")
		}
		bindingIDs[entry.ID] = true
		if _, ok := snapshot.Events.Definitions[entry.EventID]; !ok {
			return fmt.Errorf("事件绑定 %s 引用不存在事件 %s", entry.ID, entry.EventID)
		}
		if !supportedEventPhase(entry.Phase) {
			return fmt.Errorf("事件绑定 %s 使用未知阶段 %s", entry.ID, entry.Phase)
		}
		if entry.TriggerBP < 0 || entry.TriggerBP > 10000 || entry.Weight < 0 || entry.MaxPerRun < 0 || entry.CooldownNodes < 0 {
			return fmt.Errorf("事件绑定 %s 的概率或限制无效", entry.ID)
		}
		switch entry.ScopeType {
		case "global", "map", "map_tag", "node", "node_tag", "extraction", "extraction_tag":
		default:
			return fmt.Errorf("事件绑定 %s 使用未知作用域 %s", entry.ID, entry.ScopeType)
		}
	}
	for role, entries := range snapshot.Events.EncounterPools {
		if role == "" {
			return fmt.Errorf("遭遇池角色不能为空")
		}
		entryIDs := make(map[string]bool, len(entries))
		for _, entry := range entries {
			if entry.ID == "" || entryIDs[entry.ID] {
				return fmt.Errorf("遭遇池 %s 包含重复或空条目 ID", role)
			}
			entryIDs[entry.ID] = true
			if entry.MapID != snapshot.Map.ID || entry.Role != role {
				return fmt.Errorf("遭遇池 %s 的条目 %s 地图或角色不匹配", role, entry.ID)
			}
			if entry.Weight < 0 {
				return fmt.Errorf("遭遇池 %s 的条目 %s 权重无效", role, entry.ID)
			}
			if _, ok := snapshot.Enemies[entry.EnemyID]; !ok {
				return fmt.Errorf("遭遇池 %s 的条目 %s 引用不存在敌人 %s", role, entry.ID, entry.EnemyID)
			}
		}
	}
	for definitionID, definition := range snapshot.Events.Definitions {
		for _, option := range definition.Options {
			if option.Check.Type != "" && option.Check.Type != "none" && option.Check.Type != "fixed" && option.Check.Type != "attribute" {
				return fmt.Errorf("事件 %s 使用未知判定类型 %s", definitionID, option.Check.Type)
			}
			for _, mode := range option.Modes {
				if mode != "exploring" && mode != "evacuating" {
					return fmt.Errorf("事件 %s 使用未知模式 %s", definitionID, mode)
				}
			}
			for _, style := range option.Styles {
				if !hasStyle(snapshot.Styles, style) {
					return fmt.Errorf("事件 %s 使用未知行动风格 %s", definitionID, style)
				}
			}
			for _, condition := range option.Conditions {
				if !validCondition(condition) {
					return fmt.Errorf("事件 %s 使用未知条件 %s/%s", definitionID, condition.Type, condition.Operator)
				}
				if condition.Type == "has_item" {
					item, ok := snapshot.Items[condition.Ref]
					if !ok || item.Kind != "consumable" {
						return fmt.Errorf("事件 %s 的物品条件引用无效 %s", definitionID, condition.Ref)
					}
				}
			}
			if option.Check.ItemBonusRef != "" {
				item, ok := snapshot.Items[option.Check.ItemBonusRef]
				if !ok || item.Kind != "consumable" {
					return fmt.Errorf("事件 %s 的判定加成引用无效 %s", definitionID, option.Check.ItemBonusRef)
				}
			}
			for _, effect := range append(append([]EventEffect{}, option.SuccessEffects...), option.FailureEffects...) {
				if !validEffect(effect) {
					return fmt.Errorf("事件 %s 使用未知效果 %s", definitionID, effect.Type)
				}
				switch effect.Type {
				case "container":
					if _, ok := snapshot.Containers[effect.Ref]; !ok {
						return fmt.Errorf("事件 %s 引用不存在容器 %s", definitionID, effect.Ref)
					}
				case "container_pool":
					if !hasAnyContainerPool(snapshot.NodeContainerAssignments, effect.Ref) {
						return fmt.Errorf("事件 %s 引用不存在容器池 %s", definitionID, effect.Ref)
					}
				case "encounter":
					if len(snapshot.Events.EncounterPools[effect.Ref]) == 0 {
						return fmt.Errorf("事件 %s 引用不存在遭遇角色 %s", definitionID, effect.Ref)
					}
				case "consume_item":
					item, ok := snapshot.Items[effect.Ref]
					if !ok || item.Kind != "consumable" {
						return fmt.Errorf("事件 %s 引用不存在消耗品 %s", definitionID, effect.Ref)
					}
				}
			}
		}
	}
	return nil
}
