package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"idle/internal/models"

	"gorm.io/gorm"
)

type SessionService struct {
	db *gorm.DB
}

func NewSessionService(db *gorm.DB) *SessionService {
	return &SessionService{db: db}
}

// StartReq 启动挂机会话的请求：地图、行动风格与失能后的预设装备序号。
type StartReq struct {
	MapID          string   `json:"mapId"`
	Style          string   `json:"style"`          // balanced/stealth/aggressive/greedy
	RecoveryPreset int      `json:"recoveryPreset"` // 失能后使用的预设装备序号 1-3
	WeaponID       string   `json:"-"`
	ArmorID        string   `json:"-"`
	ChestRigID     string   `json:"-"`
	BackpackID     string   `json:"-"`
	HelmetID       string   `json:"-"`
	HeadsetID      string   `json:"-"`
	Consumables    []string `json:"-"`
}

// defaultOfflineLimitMin 去离线时限选择后，会话默认持续挂机时长（分钟）。
const defaultOfflineLimitMin = 1440

// Start 创建会话并交给后台模拟，接口立即返回可查询的会话记录。
func (s *SessionService) Start(req StartReq) (*models.Session, error) {
	var c models.Character
	if err := s.db.First(&c, models.PlayerCharacterID).Error; err != nil {
		return nil, fmt.Errorf("玩家角色不存在")
	}
	// 检查伤势
	if c.Injury != "" && c.Injury != "none" && c.InjuryUntil != nil && time.Now().Before(*c.InjuryUntil) {
		return nil, fmt.Errorf("角色伤势恢复中，剩余 %v", time.Until(*c.InjuryUntil).Round(time.Second))
	}
	style, err := resolveActionStyle(req.Style)
	if err != nil {
		return nil, err
	}
	req.Style = string(style)
	if req.RecoveryPreset < 1 || req.RecoveryPreset > 3 {
		return nil, fmt.Errorf("失败预设装备序号需为 1-3")
	}
	if err := s.validateMap(req.MapID); err != nil {
		return nil, err
	}
	var running int64
	if err := s.db.Model(&models.Session{}).Where("status IN ?", []string{"running", "waiting_injury"}).Count(&running).Error; err != nil {
		return nil, fmt.Errorf("读取行动状态: %w", err)
	}
	if running > 0 {
		return nil, fmt.Errorf("已有行动正在进行")
	}
	loadout, err := GetPlayerLoadout(s.db)
	if err != nil {
		return nil, err
	}
	if err := validateOwnedLoadout(s.db, loadout.WeaponID, loadout.ArmorID, loadout.Consumables,
		loadout.ChestRigID, loadout.BackpackID, loadout.HelmetID, loadout.HeadsetID); err != nil {
		return nil, err
	}
	presetWeaponID, presetArmorID, _ := PresetOf(loadout, req.RecoveryPreset)
	if presetWeaponID == "" || presetArmorID == "" {
		return nil, fmt.Errorf("预设装备 %d 未配置，请先在角色页面配置", req.RecoveryPreset)
	}
	req.WeaponID = loadout.WeaponID
	req.ArmorID = loadout.ArmorID
	req.ChestRigID = loadout.ChestRigID
	req.BackpackID = loadout.BackpackID
	req.HelmetID = loadout.HelmetID
	req.HeadsetID = loadout.HeadsetID
	req.Consumables = loadout.Consumables
	seed := time.Now().UnixNano()
	sess := models.Session{
		CharacterID:     models.PlayerCharacterID,
		MapID:           req.MapID,
		Style:           req.Style,
		RecoveryPreset:  req.RecoveryPreset,
		WeaponID:        req.WeaponID,
		ArmorID:         req.ArmorID,
		Consumables:     strings.Join(req.Consumables, ","),
		Status:          "running",
		Seed:            seed,
		StartTime:       time.Now(),
		OfflineLimitMin: defaultOfflineLimitMin,
	}
	if err := s.db.Create(&sess).Error; err != nil {
		return nil, err
	}
	go s.runSession(sess.ID)
	return &sess, nil
}

func (s *SessionService) validateMap(mapID string) error {
	var gameMap models.MapDef
	if err := s.db.First(&gameMap, "id = ?", mapID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("地图不存在")
		}
		return fmt.Errorf("读取地图: %w", err)
	}
	var nodes []models.NodeDef
	if err := s.db.Where("map_id = ?", mapID).Find(&nodes).Error; err != nil {
		return fmt.Errorf("读取地图节点: %w", err)
	}
	if err := validateDirectedRoute(nodes, gameMap); err != nil {
		return fmt.Errorf("地图路线无效: %w", err)
	}
	return nil
}

func (s *SessionService) runSession(id uint) {
	if err := s.simulateSession(id); err != nil {
		now := time.Now()
		if updateErr := s.db.Model(&models.Session{}).Where("id = ? AND status IN ?", id, []string{"running", "waiting_injury"}).Updates(map[string]interface{}{
			"status": "failed", "end_time": now,
		}).Error; updateErr != nil {
			log.Printf("session %d failed: %v; status update failed: %v", id, err, updateErr)
			return
		}
		log.Printf("session %d failed: %v", id, err)
	}
}

func (s *SessionService) simulateSession(id uint) error {
	var sess models.Session
	if err := s.db.First(&sess, id).Error; err != nil {
		return fmt.Errorf("读取行动会话: %w", err)
	}
	var c models.Character
	if err := s.db.First(&c, sess.CharacterID).Error; err != nil {
		return fmt.Errorf("读取行动角色: %w", err)
	}
	var gameMap models.MapDef
	if err := s.db.First(&gameMap, "id = ?", sess.MapID).Error; err != nil {
		return fmt.Errorf("读取行动地图: %w", err)
	}
	var nodes []models.NodeDef
	if err := s.db.Where("map_id = ?", sess.MapID).Find(&nodes).Error; err != nil {
		return fmt.Errorf("读取地图节点: %w", err)
	}
	if len(nodes) == 0 {
		return fmt.Errorf("地图没有可探索节点")
	}
	events, err := loadEventManager(s.db, gameMap)
	if err != nil {
		return err
	}
	nodeIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	nodeContainers, err := loadNodeContainers(s.db, nodeIDs)
	if err != nil {
		return err
	}
	containerCatalog, err := loadLootContainers(s.db)
	if err != nil {
		return err
	}
	lootCatalog, err := loadLootCatalog(s.db)
	if err != nil {
		return err
	}

	rng := rand.New(rand.NewSource(sess.Seed))
	totalMin := 0
	maxMin := sess.OfflineLimitMin
	runIdx := 0
	for totalMin < maxMin {
		if err := s.refreshSession(&sess); err != nil {
			return err
		}
		if sess.Status == "aborted" {
			return nil
		}
		var weapon models.WeaponDef
		if err := s.db.First(&weapon, "id=?", sess.WeaponID).Error; err != nil {
			return fmt.Errorf("读取行动武器: %w", err)
		}
		var armor models.ArmorDef
		if err := s.db.First(&armor, "id=?", sess.ArmorID).Error; err != nil {
			return fmt.Errorf("读取行动护甲: %w", err)
		}
		if weapon.AmmoPerRound > 0 {
			hasAmmo, err := hasInventoryItem(s.db, "ammo_box")
			if err != nil {
				return err
			}
			if !hasAmmo {
				break
			}
		}
		runIdx++
		armorInstance, err := s.currentArmorInstance(sess.ArmorID)
		if err != nil {
			return err
		}
		carry, err := GetCarryCapacity(s.db)
		if err != nil {
			return err
		}
		freeSlots := carry.TotalSlots - carry.UsedSlots
		freeWeight := carry.TotalWeight - carry.UsedWeight
		outcome, err := s.simulateSingleRun(&c, weapon, armor, armorInstance.CurDurability, nodes, gameMap, events, nodeContainers, containerCatalog, lootCatalog, rng, sess.Style, splitIDs(sess.Consumables), freeSlots, freeWeight, runIdx)
		if err != nil {
			return err
		}
		run := outcome.run
		run.SessionID = sess.ID
		run.RunIndex = runIdx
		run.DurationMin = outcome.duration
		run.Injury = outcome.injury
		finished := outcome.finished
		injury := outcome.injury
		if run.Result == "incapacitated" {
			outcome.skipResourceConsumption = true
			paid, replaceErr := ReplaceLostLoadout(s.db, sess.RecoveryPreset)
			if replaceErr != nil {
				if !errors.Is(replaceErr, ErrPurchaseUnavailable) {
					return fmt.Errorf("处理失能丢装: %w", replaceErr)
				}
				if err := appendRunReport(run, fmt.Sprintf(">> 携行装备已丢失，按预设 %d 补购失败：%s", sess.RecoveryPreset, replaceErr)); err != nil {
					return err
				}
				sess.WeaponID = ""
				sess.ArmorID = ""
				sess.Consumables = ""
				finished = true
			} else {
				loadout, loadoutErr := GetPlayerLoadout(s.db)
				if loadoutErr != nil {
					return loadoutErr
				}
				sess.WeaponID = loadout.WeaponID
				sess.ArmorID = loadout.ArmorID
				sess.Consumables = strings.Join(loadout.Consumables, ",")
				if err := appendRunReport(run, fmt.Sprintf(">> 携行装备已丢失，按预设 %d 补购完成（￥%d），准备继续探索", sess.RecoveryPreset, paid)); err != nil {
					return err
				}
				finished = false
			}
		}
		outcome.finished = finished
		if err := s.settleRun(&sess, outcome, armorInstance.ID); err != nil {
			return err
		}
		finished = outcome.finished
		totalMin += outcome.duration
		sess.TotalRuns = runIdx
		if err := s.db.Model(&models.Session{}).Where("id = ? AND status = ?", sess.ID, "running").Updates(map[string]interface{}{
			"total_runs": sess.TotalRuns, "weapon_id": sess.WeaponID, "armor_id": sess.ArmorID, "consumables": sess.Consumables,
		}).Error; err != nil {
			return fmt.Errorf("更新行动进度: %w", err)
		}

		// 处理伤势等待（MVP：10/30/60秒，按分钟模拟为1/3/6分钟等待，不消耗模拟时间但阻塞下一局）
		if injury == "light" || injury == "heavy" || injury == "lethal" {
			waitMin := map[string]int{"light": 1, "heavy": 3, "lethal": 6}[injury]
			// 写入角色伤势
			until := time.Now().Add(time.Duration(waitMin) * time.Minute)
			// 实际MVP用秒级演示，这里用分钟便于观察
			// 为演示挂机连续性，伤势等待计入会话时间
			totalMin += waitMin
			c.Injury = injury
			c.InjuryUntil = &until
			if err := s.db.Model(&models.Character{}).Where("id=?", c.ID).Updates(map[string]interface{}{"injury": injury, "injury_until": until}).Error; err != nil {
				return fmt.Errorf("保存角色伤势: %w", err)
			}
			if err := s.db.Model(&models.Session{}).Where("id = ? AND status = ?", sess.ID, "running").Update("status", "waiting_injury").Error; err != nil {
				return fmt.Errorf("保存行动等待状态: %w", err)
			}
			if totalMin >= maxMin {
				break
			}
			if err := s.refreshSession(&sess); err != nil {
				return err
			}
			if sess.Status == "aborted" {
				return nil
			}
			// 伤势恢复后继续
			c.Injury = "none"
			c.InjuryUntil = nil
			if err := s.db.Model(&models.Character{}).Where("id=?", c.ID).Updates(map[string]interface{}{"injury": "none", "injury_until": nil}).Error; err != nil {
				return fmt.Errorf("清除角色伤势: %w", err)
			}
			if err := s.db.Model(&models.Session{}).Where("id = ? AND status = ?", sess.ID, "waiting_injury").Update("status", "running").Error; err != nil {
				return fmt.Errorf("恢复行动运行状态: %w", err)
			}
		}
		// 被俘/失能结束会话
		if finished {
			break
		}
		if totalMin >= maxMin {
			break
		}
		// 经验成长：简化每局胜场给熟练度+1
		if run.Result == "success" || run.Result == "partial" {
			switch weapon.Category {
			case "melee":
				c.MeleeProf++
			case "pistol":
				c.PistolProf++
			case "smg":
				c.SMGProf++
			case "shotgun":
				c.ShotgunProf++
			case "rifle":
				c.RifleProf++
			case "sniper":
				c.SniperProf++
			}
			if c.MeleeProf > 100 {
				c.MeleeProf = 100
			}
			if c.PistolProf > 100 {
				c.PistolProf = 100
			}
			if c.SMGProf > 100 {
				c.SMGProf = 100
			}
			if c.ShotgunProf > 100 {
				c.ShotgunProf = 100
			}
			if c.RifleProf > 100 {
				c.RifleProf = 100
			}
			if c.SniperProf > 100 {
				c.SniperProf = 100
			}
			if err := s.db.Model(&c).Updates(map[string]interface{}{
				"melee_prof": c.MeleeProf, "pistol_prof": c.PistolProf, "smg_prof": c.SMGProf,
				"shotgun_prof": c.ShotgunProf, "rifle_prof": c.RifleProf, "sniper_prof": c.SniperProf,
			}).Error; err != nil {
				return fmt.Errorf("保存武器熟练度: %w", err)
			}
		}
	}
	now := time.Now()
	result := s.db.Model(&models.Session{}).Where("id = ? AND status IN ?", sess.ID, []string{"running", "waiting_injury"}).Updates(map[string]interface{}{
		"status": "finished", "end_time": now, "total_runs": sess.TotalRuns, "weapon_id": sess.WeaponID,
		"armor_id": sess.ArmorID, "consumables": sess.Consumables,
	})
	if result.Error != nil {
		return fmt.Errorf("保存行动结果: %w", result.Error)
	}
	return nil
}

func (s *SessionService) refreshSession(sess *models.Session) error {
	var current models.Session
	if err := s.db.Select("status", "weapon_id", "armor_id", "consumables").First(&current, sess.ID).Error; err != nil {
		return fmt.Errorf("刷新行动状态: %w", err)
	}
	sess.Status = current.Status
	sess.WeaponID = current.WeaponID
	sess.ArmorID = current.ArmorID
	sess.Consumables = current.Consumables
	return nil
}

func (s *SessionService) currentArmorInstance(armorID string) (*models.ArmorInstance, error) {
	var instance models.ArmorInstance
	if err := s.db.Where("armor_id = ? AND status = ?", armorID, "normal").Order("id asc").First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("护甲 %s 已损坏或没有可用实例", armorID)
		}
		return nil, fmt.Errorf("读取护甲耐久: %w", err)
	}
	return &instance, nil
}

type lootDrop struct {
	item        catalogItem
	quantity    int
	containerID string
	source      string
}

type simulatedRun struct {
	run      *models.SessionRun
	duration int
	injury   string
	finished bool
	// 失能时携行装备与本局消耗一起丢失，补购后的新装备不能再次扣除旧局资源。
	skipResourceConsumption bool
	armorDurability         int
	loot                    []lootDrop
	extractedLoot           []lootDrop
	consumedItems           map[string]int
}

const ammoPerBox = 30

// settleRun 将一局行动的非战斗结果一次性写入库存、护甲和行动记录。
func (s *SessionService) settleRun(sess *models.Session, outcome *simulatedRun, armorInstanceID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		storedLoot, overflow, err := fitLootToStorage(tx, outcome.extractedLoot)
		if err != nil {
			return err
		}
		outcome.extractedLoot = storedLoot
		outcome.run.Loot = encodeLoot(storedLoot)
		for _, drop := range storedLoot {
			if err := addInventoryItem(tx, drop.item, drop.quantity, true); err != nil {
				return err
			}
		}
		if len(storedLoot) > 0 {
			if err := appendRunReport(outcome.run, fmt.Sprintf(">> 实际带回 %d 件物品", lootQuantity(storedLoot))); err != nil {
				return err
			}
		}
		if overflow > 0 {
			if err := appendRunReport(outcome.run, fmt.Sprintf(">> 基地仓库空间不足，放弃 %d 件物品", overflow)); err != nil {
				return err
			}
		}

		if outcome.run.Result != "incapacitated" {
			status := "normal"
			if outcome.armorDurability <= 0 {
				status = "broken"
				outcome.finished = true
				if err := appendRunReport(outcome.run, ">> 护甲耐久归零，行动结束，需先维修护甲"); err != nil {
					return err
				}
			}
			if err := tx.Model(&models.ArmorInstance{}).Where("id = ?", armorInstanceID).Updates(map[string]interface{}{
				"cur_durability": maxInt(outcome.armorDurability, 0), "status": status,
			}).Error; err != nil {
				return fmt.Errorf("保存护甲耐久: %w", err)
			}
		}

		if outcome.skipResourceConsumption {
			if err := appendRunReport(outcome.run, ">> 失能：本局剩余携行资源随装备一并丢失"); err != nil {
				return err
			}
		} else if err := consumeRunResources(tx, sess, outcome); err != nil {
			return err
		}
		if err := tx.Create(outcome.run).Error; err != nil {
			return fmt.Errorf("保存单局记录: %w", err)
		}
		return nil
	})
}

func fitLootToStorage(tx *gorm.DB, loot []lootDrop) ([]lootDrop, int, error) {
	if len(loot) == 0 {
		return nil, 0, nil
	}
	used, err := inventoryUsage(tx)
	if err != nil {
		return nil, 0, err
	}
	space := models.InventoryCapacity - used
	stored := make([]lootDrop, 0, len(loot))
	overflow := 0
	for _, drop := range loot {
		kept := drop.quantity
		if kept > space {
			kept = space
		}
		if kept > 0 {
			copyDrop := drop
			copyDrop.quantity = kept
			stored = append(stored, copyDrop)
			space -= kept
		}
		overflow += drop.quantity - kept
	}
	return stored, overflow, nil
}

func consumeRunResources(tx *gorm.DB, sess *models.Session, outcome *simulatedRun) error {
	if outcome.run.AmmoUsed > 0 {
		boxes := (outcome.run.AmmoUsed + ammoPerBox - 1) / ammoPerBox
		if err := removeInventoryItem(tx, "ammo_box", boxes); err != nil {
			return fmt.Errorf("扣除弹药: %w", err)
		}
	}
	consumables := splitIDs(sess.Consumables)
	consume := func(itemID string) error {
		if !containsID(consumables, itemID) {
			return nil
		}
		if err := removeInventoryItem(tx, itemID, 1); err != nil {
			return fmt.Errorf("扣除%s: %w", itemID, err)
		}
		var count int64
		if err := tx.Model(&models.Inventory{}).Where("item_id = ? AND quantity > 0", itemID).Count(&count).Error; err != nil {
			return fmt.Errorf("读取%s库存: %w", itemID, err)
		}
		if count == 0 {
			consumables = removeID(consumables, itemID)
		}
		return nil
	}
	for itemID, quantity := range outcome.consumedItems {
		for i := 0; i < quantity; i++ {
			if err := consume(itemID); err != nil {
				return err
			}
		}
	}
	sess.Consumables = strings.Join(consumables, ",")
	if err := tx.Model(&models.PlayerLoadout{}).Where("id = ?", models.PlayerLoadoutID).
		Select("Consumables").Updates(&models.PlayerLoadout{Consumables: consumables}).Error; err != nil {
		return fmt.Errorf("保存当前补给: %w", err)
	}
	return nil
}

func hasInventoryItem(db *gorm.DB, itemID string) (bool, error) {
	var count int64
	if err := db.Model(&models.Inventory{}).Where("item_id = ? AND quantity > 0", itemID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("读取%s库存: %w", itemID, err)
	}
	return count > 0, nil
}

func splitIDs(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func containsID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func removeID(ids []string, target string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != target {
			result = append(result, id)
		}
	}
	return result
}

func maxInt(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func appendRunReport(run *models.SessionRun, line string) error {
	var lines []string
	if err := json.Unmarshal([]byte(run.Report), &lines); err != nil {
		return fmt.Errorf("读取单局报告: %w", err)
	}
	lines = append(lines, line)
	encoded, err := json.Marshal(lines)
	if err != nil {
		return fmt.Errorf("更新单局报告: %w", err)
	}
	run.Report = string(encoded)
	return nil
}

func (s *SessionService) simulateSingleRun(c *models.Character, weapon models.WeaponDef, armor models.ArmorDef, armorDurability int, nodes []models.NodeDef, gameMap models.MapDef, events *eventManager, nodeContainers map[string][]models.NodeContainerDef, containerCatalog map[string]lootContainer, lootCatalog map[string]catalogItem, rng *rand.Rand, style string, consumables []string, carrySlots int, carryWeight float64, runIdx int) (*simulatedRun, error) {
	byID := make(map[string]models.NodeDef, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	actionStyle, err := resolveActionStyle(style)
	if err != nil {
		return nil, err
	}
	nodeContainerPools := nodeContainers
	nodeContainers = materializeNodeContainers(nodes, nodeContainers, rng)
	if _, ok := byID[gameMap.StartNodeID]; !ok {
		return nil, fmt.Errorf("起点节点 %s 不存在", gameMap.StartNodeID)
	}
	if err := validateDirectedRoute(nodes, gameMap); err != nil {
		return nil, err
	}

	lines := []string{fmt.Sprintf("=== 第%d局开始 地图:%s 风格:%s ===", runIdx, gameMap.Name, actionStylePolicy(actionStyle).Label)}
	playerActor := BuildPlayerActor(*c, weapon, armor)
	playerActor.ArmorDurability = float64(armorDurability)
	availableItems := make(map[string]bool, len(consumables))
	for _, itemID := range consumables {
		availableItems[itemID] = true
	}
	state := &eventRunState{
		Character: c, Player: &playerActor, Mode: runModeExploring, Style: actionStyle,
		CarrySlots: carrySlots, CarryWeight: carryWeight,
		AvailableItems: availableItems, ConsumedItems: make(map[string]int), Flags: make(map[string]bool),
		EventCounts: make(map[string]int), LastEventVisit: make(map[string]int), Lines: &lines,
	}
	loot := make([]lootDrop, 0)
	state.CollectContainer = func(containerID, source string) error {
		container, ok := containerCatalog[containerID]
		if !ok {
			return fmt.Errorf("容器 %s 不存在", containerID)
		}
		state.Duration += container.Def.SearchTime
		lines = append(lines, fmt.Sprintf("  搜索容器 %s（标签:%s，价值%d级，风险%d，耗时%d分钟）", container.Def.Name, strings.Join(container.Def.Tags, "/"), container.Def.ValueTier, container.Def.SearchRisk, container.Def.SearchTime))
		rolls := container.Def.RollMin
		if container.Def.RollMax > container.Def.RollMin {
			rolls += rng.Intn(container.Def.RollMax - container.Def.RollMin + 1)
		}
		if rolls <= 0 {
			lines = append(lines, "    容器为空")
			return nil
		}
		for i := 0; i < rolls; i++ {
			rule, ok := chooseContainerRule(container, rng)
			if !ok {
				lines = append(lines, "    容器没有可用掉落规则")
				break
			}
			item, ok := chooseLootItem(lootCatalog, rule.ItemCategory, rng)
			if !ok {
				lines = append(lines, fmt.Sprintf("    %s 分类暂无物品", rule.ItemCategory))
				continue
			}
			quantity := rule.MinQuantity
			if rule.MaxQuantity > rule.MinQuantity {
				quantity += rng.Intn(rule.MaxQuantity - rule.MinQuantity + 1)
			}
			needSlots := item.Slots * quantity
			needWeight := float64(item.Weight * quantity)
			if quantity <= 0 || state.LootSlots+needSlots > state.CarrySlots || state.LootWeight+needWeight > state.CarryWeight {
				if quantity > 0 {
					state.CarryBlocked = true
					lines = append(lines, fmt.Sprintf("    容量不足，放弃 %s x%d", item.Name, quantity))
				}
				continue
			}
			loot = append(loot, lootDrop{item: item, quantity: quantity, containerID: containerID, source: source})
			state.LootSlots += needSlots
			state.LootWeight += needWeight
			lines = append(lines, fmt.Sprintf("    获得 %s x%d", item.Name, quantity))
		}
		return nil
	}
	state.HasContainerPool = func(poolID string) bool {
		return poolID != "" && hasNodeContainerPool(nodeContainerPools[state.Node.ID], poolID)
	}
	state.CollectContainerPool = func(poolID, source string, count int) error {
		if count <= 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			assignment, ok := chooseNodeContainerPool(nodeContainerPools[state.Node.ID], poolID, rng)
			if !ok {
				return fmt.Errorf("节点 %s 没有可用事件奖励池 %s", state.Node.ID, poolID)
			}
			lines = append(lines, fmt.Sprintf("  事件奖励池 %s 按权重抽到 %s", poolID, assignment.ContainerID))
			if err := state.CollectContainer(assignment.ContainerID, source); err != nil {
				return err
			}
			if state.CarryBlocked {
				break
			}
		}
		return nil
	}
	state.DiscardLoot = func(quantity int) int {
		discarded := 0
		for i := len(loot) - 1; i >= 0 && discarded < quantity; i-- {
			remove := minInt(loot[i].quantity, quantity-discarded)
			loot[i].quantity -= remove
			state.LootSlots -= loot[i].item.Slots * remove
			state.LootWeight -= float64(loot[i].item.Weight * remove)
			discarded += remove
			if loot[i].quantity == 0 {
				loot = append(loot[:i], loot[i+1:]...)
			}
		}
		return discarded
	}

	result := ""
	finishedSession := false
	currentNodeID := gameMap.StartNodeID
	for step := 0; step < len(nodes)+1; step++ {
		node, ok := byID[currentNodeID]
		if !ok {
			return nil, fmt.Errorf("路线节点 %s 不存在", currentNodeID)
		}
		state.Node = node
		state.VisitSequence++
		state.resetNodeActions()
		modeAtEntry := state.Mode
		if modeAtEntry == runModeExploring {
			lines = append(lines, fmt.Sprintf("[%02d:%02d] 进入节点 %s，探索%d分钟，距离%s", state.Duration/60, state.Duration%60, node.Name, node.ExploreTime, node.Distance))
		} else {
			lines = append(lines, fmt.Sprintf("[%02d:%02d] 撤离途中抵达 %s，移动%d分钟，距离%s", state.Duration/60, state.Duration%60, node.Name, node.ExploreTime, node.Distance))
		}
		state.Duration += node.ExploreTime

		if err := events.Trigger(state, eventPhaseEnterNode, rng); err != nil {
			return nil, err
		}
		evaluateAutomaticEvacuation(state, weapon)
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}
		if state.Mode == runModeEvacuating {
			if err := events.Trigger(state, eventPhaseEvacStep, rng); err != nil {
				return nil, err
			}
			evaluateAutomaticEvacuation(state, weapon)
		}

		if err := events.Trigger(state, eventPhasePreEncounter, rng); err != nil {
			return nil, err
		}
		evaluateAutomaticEvacuation(state, weapon)
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}

		enemyDef, enemyDefeated, encounterCleared, encountered, err := s.resolveNodeEncounter(state, events, node, rng)
		if err != nil {
			return nil, err
		}
		if state.Player.HP <= 0 {
			lines = append(lines, ">> 玩家失去行动能力")
			result = "incapacitated"
			finishedSession = true
			break
		}
		if err := events.Trigger(state, eventPhasePostEncounter, rng); err != nil {
			return nil, err
		}
		evaluateAutomaticEvacuation(state, weapon)
		if err := startEvacuationEvents(events, state, rng); err != nil {
			return nil, err
		}

		if state.Mode == runModeExploring && enemyDefeated && enemyDef.BackpackContainerID != "" {
			if err := state.CollectContainer(enemyDef.BackpackContainerID, "敌人背包"); err != nil {
				return nil, err
			}
		}
		canSearch := !encountered || encounterCleared
		if state.Mode == runModeExploring && canSearch && !state.SkipSearch {
			if err := events.Trigger(state, eventPhasePreSearch, rng); err != nil {
				return nil, err
			}
			evaluateAutomaticEvacuation(state, weapon)
			if state.Mode == runModeExploring && !state.SkipSearch {
				// 容器已经在单局开始时按节点权重生成，搜索阶段只按生成顺序处理。
				for _, assignment := range nodeContainers[node.ID] {
					for i := 0; i < assignment.Count; i++ {
						if err := state.CollectContainer(assignment.ContainerID, "节点:"+node.Name); err != nil {
							return nil, err
						}
						if state.CarryBlocked {
							break
						}
					}
					if state.CarryBlocked {
						break
					}
				}
				if err := events.Trigger(state, eventPhasePostSearch, rng); err != nil {
					return nil, err
				}
			}
			evaluateAutomaticEvacuation(state, weapon)
			if err := startEvacuationEvents(events, state, rng); err != nil {
				return nil, err
			}
		}

		if node.ID == gameMap.ExtractionNodeID {
			if err := events.Trigger(state, eventPhaseAtExtraction, rng); err != nil {
				return nil, err
			}
			if state.Player.HP <= 0 {
				lines = append(lines, ">> 玩家在撤离点失去行动能力")
				result = "incapacitated"
				finishedSession = true
				break
			}
			result = "success"
			if state.EvacuationEmergency {
				result = "emergency"
			}
			lines = append(lines, fmt.Sprintf(">> 抵达撤离点 %s，完成%s", node.Name, extractionLabel(result)))
			break
		}

		state.Player.Stress = clamp(state.Player.Stress-float64(node.ExploreTime)*5, 0, state.Player.StressThreshold)
		if state.EvacShortcut {
			state.Duration = maxInt(state.Duration-2, 0)
			lines = append(lines, ">> 使用临时捷径，缩短下一段移动耗时")
		}
		nextNode, found, err := nextForwardNode(node, byID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("节点 %s 不是撤离点却没有下一节点", node.ID)
		}
		currentNodeID = nextNode.ID
	}
	if result == "" {
		return nil, fmt.Errorf("单局节点推进超过路线长度，地图配置可能存在环路")
	}

	injury := "none"
	if result == "incapacitated" || state.Player.HP <= 0 {
		injury = "lethal"
	} else if state.Player.HP < state.Player.MaxHP*0.3 {
		injury = "heavy"
	} else if state.Player.HP < state.Player.MaxHP || (state.EvacuationEmergency && state.EvacuationReason == "stress") {
		injury = "light"
	}
	lines = append(lines, fmt.Sprintf("=== 本局结束 结果:%s 耗时:%d分 热度:%d 弹药:%d 伤势:%s ===", result, state.Duration, state.Heat, state.AmmoUsed, injury))
	if len(loot) == 0 {
		lines = append(lines, ">> 本局没有搜集到可带回的物资")
	} else {
		lines = append(lines, fmt.Sprintf(">> 本局搜集到 %d 件物品，占用 %d 格 / %.1fkg", lootQuantity(loot), state.LootSlots, state.LootWeight))
	}
	extractedLoot := selectExtractedLoot(result, loot)
	if lootQuantity(extractedLoot) < lootQuantity(loot) {
		lines = append(lines, fmt.Sprintf(">> %s仅保留 %d 件物品", extractionLabel(result), lootQuantity(extractedLoot)))
	}
	reportJSON, err := json.Marshal(lines)
	if err != nil {
		return nil, fmt.Errorf("生成单局报告: %w", err)
	}
	run := &models.SessionRun{Result: result, Heat: state.Heat, AmmoUsed: state.AmmoUsed, Loot: encodeLoot(extractedLoot), Report: string(reportJSON)}
	return &simulatedRun{
		run: run, duration: state.Duration, injury: injury, finished: finishedSession,
		armorDurability: int(math.Round(state.Player.ArmorDurability)), loot: loot,
		extractedLoot: extractedLoot, consumedItems: state.ConsumedItems,
	}, nil
}

func (s *SessionService) resolveNodeEncounter(state *eventRunState, events *eventManager, node models.NodeDef, rng *rand.Rand) (models.EnemyDef, bool, bool, bool, error) {
	if state.SkipDefaultCombat {
		*state.Lines = append(*state.Lines, "  已根据事件结果避开本节点交战")
		return models.EnemyDef{}, false, false, false, nil
	}
	enemyID := node.EnemyID
	forced := state.EncounterRole != ""
	encounterRole := node.EncounterRole
	if forced {
		var err error
		enemyID, err = events.ResolveEnemyID(state.EncounterRole, rng)
		if err != nil {
			return models.EnemyDef{}, false, false, false, err
		}
		encounterRole = state.EncounterRole
	}
	if enemyID == "" {
		*state.Lines = append(*state.Lines, "  当前节点没有配置敌人，安全通过")
		return models.EnemyDef{}, false, false, false, nil
	}
	encounterProbability := 60 + state.Heat
	if state.Mode == runModeEvacuating {
		encounterProbability = 35 + state.Heat
	}
	encounterProbability = minInt(encounterProbability, 90)
	if !forced && rng.Intn(100) >= encounterProbability {
		*state.Lines = append(*state.Lines, "  未遭遇敌人，安全通过")
		return models.EnemyDef{}, false, false, false, nil
	}
	var enemyDef models.EnemyDef
	if err := s.db.First(&enemyDef, "id = ?", enemyID).Error; err != nil {
		return models.EnemyDef{}, false, false, false, fmt.Errorf("读取敌人 %s: %w", enemyID, err)
	}
	var enemyWeapon models.WeaponDef
	if err := s.db.First(&enemyWeapon, "id = ?", enemyDef.WeaponID).Error; err != nil {
		return models.EnemyDef{}, false, false, false, fmt.Errorf("读取敌方武器 %s: %w", enemyDef.WeaponID, err)
	}
	var enemyArmor models.ArmorDef
	if err := s.db.First(&enemyArmor, "id = ?", enemyDef.ArmorID).Error; err != nil {
		return models.EnemyDef{}, false, false, false, fmt.Errorf("读取敌方护甲 %s: %w", enemyDef.ArmorID, err)
	}
	enemyActor := BuildEnemyActor(enemyDef, enemyWeapon, enemyArmor)
	policy := actionStylePolicy(state.Style)
	approach := policy.encounterApproach(encounterRole)
	*state.Lines = append(*state.Lines, fmt.Sprintf("  %s风格对%s选择%s", policy.Label, enemyDef.Name, approach))
	forceEscape := state.Mode == runModeEvacuating && state.EvacuationReason == "carry_full"
	if forceEscape {
		*state.Lines = append(*state.Lines, "  当前因负重撤离，战斗内优先尝试脱离")
	}
	battleResult := SimulateEncounter(state.Player, &enemyActor, node.Distance, state.Heat, state.hasItem("smoke"), approach, policy, forceEscape, rng)
	*state.Lines = append(*state.Lines, battleResult.Lines...)
	state.Heat += battleResult.HeatAdd
	state.AmmoUsed += battleResult.AmmoUsed
	if battleResult.SmokeUsed {
		state.consumeItem("smoke")
	}
	enemyDefeated := battleResult.EnemyHP <= 0
	encounterCleared := enemyDefeated || battleResult.Winner == "player_suppress" || (battleResult.Winner == "escape" && !battleResult.EscapeSuccess)
	return enemyDef, enemyDefeated, encounterCleared, true, nil
}

func evaluateAutomaticEvacuation(state *eventRunState, weapon models.WeaponDef) {
	if state.Player.HP <= 0 {
		return
	}
	policy := actionStylePolicy(state.Style)
	if state.Player.HP < state.Player.MaxHP*policy.HealthEvacRatio {
		state.beginEvacuation("health", false)
	}
	if state.Player.Stress >= state.Player.StressThreshold*policy.StressEvacRatio {
		state.beginEvacuation("stress", false)
	}
	if weapon.AmmoPerRound > 0 && state.Player.Ammo <= 0 {
		state.beginEvacuation("ammo", true)
	}
	if state.Player.ArmorDurability <= 0 {
		state.beginEvacuation("armor", true)
	}
	if state.CarryBlocked || state.carryRatio() >= policy.CarryEvacRatio {
		state.beginEvacuation("carry_full", false)
	}
}

func startEvacuationEvents(events *eventManager, state *eventRunState, rng *rand.Rand) error {
	if !state.EvacuationPending || state.EvacuationStarted {
		return nil
	}
	state.EvacuationPending = false
	state.EvacuationStarted = true
	return events.Trigger(state, eventPhaseEvacStart, rng)
}

type lootSummary struct {
	ItemID      string `json:"itemId"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Quantity    int    `json:"quantity"`
	ContainerID string `json:"containerId"`
	Source      string `json:"source"`
}

func lootQuantity(loot []lootDrop) int {
	total := 0
	for _, drop := range loot {
		total += drop.quantity
	}
	return total
}

func encodeLoot(loot []lootDrop) string {
	summaries := make([]lootSummary, 0, len(loot))
	for _, drop := range loot {
		summaries = append(summaries, lootSummary{
			ItemID: drop.item.ID, Name: drop.item.Name, Category: drop.item.Category,
			Quantity: drop.quantity, ContainerID: drop.containerID, Source: drop.source,
		})
	}
	encoded, _ := json.Marshal(summaries)
	return string(encoded)
}

func selectExtractedLoot(result string, loot []lootDrop) []lootDrop {
	if result == "success" {
		return cloneLoot(loot)
	}
	if result != "emergency" && result != "partial" {
		return nil
	}
	keep := (lootQuantity(loot) + 1) / 2
	return takeLootUnits(loot, keep)
}

func takeLootUnits(loot []lootDrop, quantity int) []lootDrop {
	if quantity <= 0 {
		return nil
	}
	result := make([]lootDrop, 0, len(loot))
	for _, drop := range loot {
		if quantity <= 0 {
			break
		}
		kept := drop.quantity
		if kept > quantity {
			kept = quantity
		}
		copyDrop := drop
		copyDrop.quantity = kept
		result = append(result, copyDrop)
		quantity -= kept
	}
	return result
}

func cloneLoot(loot []lootDrop) []lootDrop {
	return append([]lootDrop(nil), loot...)
}

func extractionLabel(result string) string {
	switch result {
	case "emergency":
		return "紧急撤离"
	case "partial":
		return "部分撤离"
	default:
		return "撤离"
	}
}

func getAttrValue(c *models.Character, attr string) int {
	switch attr {
	case "strength":
		return c.Strength
	case "agility":
		return c.Agility
	case "intellect":
		return c.Intellect
	case "charisma":
		return c.Charisma
	case "perception":
		return c.Perception
	case "stealth":
		return c.Stealth
	case "negotiation":
		return c.Negotiation
	case "engineering":
		return c.Engineering
	case "medical":
		return c.Medical
	case "luck":
		return c.Luck
	case "survival":
		return c.Survival
	case "resist":
		return c.Resist
	}
	return 50
}

func minInt(value, ceiling int) int {
	if value > ceiling {
		return ceiling
	}
	return value
}

func (s *SessionService) GetSession(id uint) (*models.Session, []models.SessionRun, error) {
	var sess models.Session
	if err := s.db.First(&sess, id).Error; err != nil {
		return nil, nil, err
	}
	var runs []models.SessionRun
	if err := s.db.Where("session_id = ?", id).Order("run_index asc").Find(&runs).Error; err != nil {
		return nil, nil, fmt.Errorf("读取行动记录: %w", err)
	}
	return &sess, runs, nil
}

func (s *SessionService) ListSessions() ([]models.Session, error) {
	var list []models.Session
	err := s.db.Order("id desc").Limit(20).Find(&list).Error
	return list, err
}

func (s *SessionService) Abort(id uint) error {
	var sess models.Session
	if err := s.db.First(&sess, id).Error; err != nil {
		return err
	}
	if sess.Status == "running" || sess.Status == "waiting_injury" {
		now := time.Now()
		if err := s.db.Model(&models.Session{}).Where("id = ? AND status IN ?", id, []string{"running", "waiting_injury"}).Updates(map[string]interface{}{
			"status": "aborted", "end_time": now,
		}).Error; err != nil {
			return fmt.Errorf("中止行动: %w", err)
		}
	}
	return nil
}

// SimulatePreview 100次快速模拟
func (s *SessionService) SimulatePreview(req StartReq) (map[string]interface{}, error) {
	var c models.Character
	if err := s.db.First(&c, models.PlayerCharacterID).Error; err != nil {
		return nil, err
	}
	style, err := resolveActionStyle(req.Style)
	if err != nil {
		return nil, err
	}
	req.Style = string(style)
	if err := s.validateMap(req.MapID); err != nil {
		return nil, err
	}
	var gameMap models.MapDef
	if err := s.db.First(&gameMap, "id = ?", req.MapID).Error; err != nil {
		return nil, fmt.Errorf("读取预测地图: %w", err)
	}
	if req.WeaponID == "" || req.ArmorID == "" {
		loadout, err := GetPlayerLoadout(s.db)
		if err != nil {
			return nil, err
		}
		req.WeaponID = loadout.WeaponID
		req.ArmorID = loadout.ArmorID
		req.ChestRigID = loadout.ChestRigID
		req.BackpackID = loadout.BackpackID
		req.HelmetID = loadout.HelmetID
		req.HeadsetID = loadout.HeadsetID
		req.Consumables = loadout.Consumables
	}
	if err := validateOwnedLoadout(s.db, req.WeaponID, req.ArmorID, req.Consumables,
		req.ChestRigID, req.BackpackID, req.HelmetID, req.HeadsetID); err != nil {
		return nil, err
	}
	var weapon models.WeaponDef
	if err := s.db.First(&weapon, "id=?", req.WeaponID).Error; err != nil {
		return nil, fmt.Errorf("读取预测武器: %w", err)
	}
	var armor models.ArmorDef
	if err := s.db.First(&armor, "id=?", req.ArmorID).Error; err != nil {
		return nil, fmt.Errorf("读取预测护甲: %w", err)
	}
	var nodes []models.NodeDef
	if err := s.db.Where("map_id = ?", req.MapID).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("读取预测节点: %w", err)
	}
	events, err := loadEventManager(s.db, gameMap)
	if err != nil {
		return nil, err
	}
	nodeIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	nodeContainers, err := loadNodeContainers(s.db, nodeIDs)
	if err != nil {
		return nil, err
	}
	containerCatalog, err := loadLootContainers(s.db)
	if err != nil {
		return nil, err
	}
	lootCatalog, err := loadLootCatalog(s.db)
	if err != nil {
		return nil, err
	}
	carry, err := GetCarryCapacity(s.db)
	if err != nil {
		return nil, err
	}
	freeSlots := carry.TotalSlots - carry.UsedSlots
	freeWeight := carry.TotalWeight - carry.UsedWeight

	win, evac, incap := 0, 0, 0
	totalAmmo := 0
	totalTime := 0
	n := 100
	for i := 0; i < n; i++ {
		rng := rand.New(rand.NewSource(int64(i * 1000)))
		outcome, err := s.simulateSingleRun(&c, weapon, armor, armor.MaxDurability, nodes, gameMap, events, nodeContainers, containerCatalog, lootCatalog, rng, req.Style, req.Consumables, freeSlots, freeWeight, 1)
		if err != nil {
			return nil, err
		}
		totalAmmo += outcome.run.AmmoUsed
		totalTime += outcome.duration
		switch outcome.run.Result {
		case "success":
			win++
		case "emergency":
			evac++
		case "incapacitated":
			incap++
		}
	}
	averageAmmo := totalAmmo / n
	averageTime := totalTime / n
	return map[string]interface{}{
		"成功撤离":   fmt.Sprintf("%d%%", win*100/n),
		"被迫撤离":   fmt.Sprintf("%d%%", evac*100/n),
		"失去行动能力": fmt.Sprintf("%d%%", incap*100/n),
		"预计弹药消耗": fmt.Sprintf("%d~%d", maxInt(averageAmmo-2, 0), averageAmmo+2),
		"预计单局耗时": fmt.Sprintf("%d~%d 分钟", maxInt(averageTime-1, 0), averageTime+2),
		"重伤风险": func() string {
			if incap > 10 {
				return "高"
			}
			if evac > 30 {
				return "中"
			}
			return "低"
		}(),
	}, nil
}
