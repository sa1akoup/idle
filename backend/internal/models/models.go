package models

import "time"

// PlayerCharacterID 玩家角色在 MVP 中使用固定主键，避免出现预设角色池。
const PlayerCharacterID uint = 1

// PlayerLoadoutID 玩家角色的装备配置使用固定主键。
const PlayerLoadoutID uint = 1

// InventoryCapacity 仓库可容纳的非现金物品总数（当前装备与预设装备不计入）。
const InventoryCapacity = 120

// BaseCarrySlots 基础可携带物品格数（不含胸挂/背包加成）。
const BaseCarrySlots = 20

// BaseCarryWeight 基础可携带负重：初始 50kg +（力量-50）*0.3。
func BaseCarryWeight(strength int) float64 {
	return 50 + float64(strength-50)*0.3
}

// Character 角色
type Character struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	Strength    int    `json:"strength"` // 0-100
	Agility     int    `json:"agility"`
	Intellect   int    `json:"intellect"`
	Charisma    int    `json:"charisma"`
	Stealth     int    `json:"stealth"`     // 潜行
	Perception  int    `json:"perception"`  // 感知
	Negotiation int    `json:"negotiation"` // 交涉
	Luck        int    `json:"luck"`
	Survival    int    `json:"survival"` // 生存
	Resist      int    `json:"resist"`   // 抗压
	Engineering int    `json:"engineering"`
	Medical     int    `json:"medical"`
	// 武器熟练度
	MeleeProf   int `json:"meleeProf"`
	PistolProf  int `json:"pistolProf"`
	SMGProf     int `json:"smgProf"`
	ShotgunProf int `json:"shotgunProf"`
	RifleProf   int `json:"rifleProf"`
	SniperProf  int `json:"sniperProf"`

	Trait string `json:"trait"`

	// 状态
	Fatigue     int        `json:"fatigue"`
	Stress      int        `json:"stress"`
	Injury      string     `json:"injury"` // none/light/heavy/lethal
	InjuryUntil *time.Time `json:"injuryUntil"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// EffectiveSkill 计算有效子技能
func EffectiveSkill(train, mainAttr int) float64 {
	return float64(train)*0.75 + float64(mainAttr)*0.25
}

// WeaponDef 武器定义
type WeaponDef struct {
	ID           string `gorm:"primaryKey" json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"` // melee/pistol/smg/shotgun/rifle/sniper
	Hit          int    `json:"hit"`
	Damage       int    `json:"damage"`
	Penetration  int    `json:"penetration"`
	Suppress     int    `json:"suppress"`
	Ready        int    `json:"ready"`
	AmmoPerRound int    `json:"ammoPerRound"`
	Noise        int    `json:"noise"`
	Reliability  int    `json:"reliability"`
	CloseMod     int    `json:"closeMod"`
	MidMod       int    `json:"midMod"`
	FarMod       int    `json:"farMod"`
	Price        int    `json:"price"`
	Weight       int    `json:"weight"`
	Slots        int    `json:"slots"`
	// 商人类别（weapon/clothing/medical/mechanical）与解锁所需好感度
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// ArmorDef 护甲定义
type ArmorDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"` // light/heavy
	Protect          int    `json:"protect"`
	Coverage         int    `json:"coverage"` // 0-100
	Mobility         int    `json:"mobility"`
	Initiative       int    `json:"initiative"`
	Conceal          int    `json:"conceal"`
	AntiSuppress     int    `json:"antiSuppress"`
	Escape           int    `json:"escape"`
	MaxDurability    int    `json:"maxDurability"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// LootContainerDef 容器定义：一次搜索可以抽取多个物品，也允许抽取数量为 0。
type LootContainerDef struct {
	ID         string   `gorm:"primaryKey" json:"id"`
	Name       string   `json:"name"`
	Tags       []string `gorm:"serializer:json" json:"tags"`
	ValueTier  int      `json:"valueTier"`  // 容器自身价值等级，1-5
	SearchRisk int      `json:"searchRisk"` // 搜索风险，供行动风格排序
	SearchTime int      `json:"searchTime"` // 分钟
	RollMin    int      `json:"rollMin"`
	RollMax    int      `json:"rollMax"`
}

// LootContainerRule 容器内的物品分类权重与单次抽取数量。
type LootContainerRule struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ContainerID  string `gorm:"index" json:"containerId"`
	ItemCategory string `json:"itemCategory"`
	Weight       int    `json:"weight"`
	MinQuantity  int    `json:"minQuantity"`
	MaxQuantity  int    `json:"maxQuantity"`
}

// NodeContainerPoolSearch 普通节点搜索使用的默认容器池。
const NodeContainerPoolSearch = "search"

// NodeContainerDef 节点上的容器挂载关系；Pool 用于区分普通搜索与事件奖励池。
type NodeContainerDef struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	NodeID      string `gorm:"index" json:"nodeId"`
	ContainerID string `gorm:"index" json:"containerId"`
	Pool        string `gorm:"index" json:"pool"` // search 或具体事件奖励池名称
	Count       int    `json:"count"`             // 兼容固定挂载；容器池模式下由节点槽位生成
	Weight      int    `json:"weight"`            // 节点容器类型抽取权重
}

// ConsumableDef 消耗品
type ConsumableDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	Desc             string `json:"desc"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// LootItemDef 可搜集战利品：参考 Tarkov Wiki 的 barter、工具、情报、医疗、食品与贵重物分类。
// 这类物品只在探索中生成，通常通过对应商人出售兑现价值。
type LootItemDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	Category         string `json:"category"` // tool/material/electronics/info/medical/food/valuable/fuel/weaponpart
	Desc             string `json:"desc"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	DropWeight       int    `json:"dropWeight"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// ChestRigDef 胸挂：增加可携带物品格数与负重。
type ChestRigDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	AddSlots         int    `json:"addSlots"`  // 增加可携带格数
	AddWeight        int    `json:"addWeight"` // 增加可携带负重 kg
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// BackpackDef 背包：增加可携带物品格数与负重。
type BackpackDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	AddSlots         int    `json:"addSlots"`
	AddWeight        int    `json:"addWeight"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// HelmetDef 头盔：占位，暂与护甲一致提供防御，后续引入部位概率再调整。
type HelmetDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	Protect          int    `json:"protect"`
	Coverage         int    `json:"coverage"`
	Mobility         int    `json:"mobility"`
	Initiative       int    `json:"initiative"`
	Conceal          int    `json:"conceal"`
	AntiSuppress     int    `json:"antiSuppress"`
	Escape           int    `json:"escape"`
	MaxDurability    int    `json:"maxDurability"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// HeadsetDef 耳机：听力提升等级（占位），后续提升部分事件成功率/降低被伏击率。
type HeadsetDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	HearingLevel     int    `json:"hearingLevel"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// MapDef 地图
type MapDef struct {
	ID               string   `gorm:"primaryKey" json:"id"`
	Name             string   `json:"name"`
	Desc             string   `json:"desc"`
	StartNodeID      string   `json:"startNodeId"`
	ExtractionNodeID string   `json:"extractionNodeId"`
	Tags             []string `gorm:"serializer:json" json:"tags"`
}

// NodeDef 节点
type NodeDef struct {
	ID             string   `gorm:"primaryKey" json:"id"`
	MapID          string   `json:"mapId"`
	Name           string   `json:"name"`
	RouteOrder     int      `json:"routeOrder"`  // 单向路线中的顺序
	ExploreTime    int      `json:"exploreTime"` // 分钟
	Distance       string   `json:"distance"`    // close/mid/far
	EnemyID        string   `json:"enemyId"`
	EncounterRole  string   `json:"encounterRole"`  // patrol/guard/elite 等行为角色
	ContainerSlots int      `json:"containerSlots"` // 本节点每局生成的容器槽位
	ValueTier      int      `json:"valueTier"`      // 节点整体价值等级，1-5
	Connections    string   `json:"connections"`    // 仅允许填写向前的出口，csv
	Tags           []string `gorm:"serializer:json" json:"tags"`
}

// EnemyDef 聚合敌人
type EnemyDef struct {
	ID                  string `gorm:"primaryKey" json:"id"`
	Name                string `json:"name"`
	HP                  int    `json:"hp"`
	StressThreshold     int    `json:"stressThreshold"`
	Perception          int    `json:"perception"`
	Stealth             int    `json:"stealth"`
	Agility             int    `json:"agility"`
	WeaponID            string `json:"weaponId"`
	ArmorID             string `json:"armorId"`
	Evasion             int    `json:"evasion"`
	Mobility            int    `json:"mobility"`
	Suppress            int    `json:"suppress"`
	BackpackContainerID string `json:"backpackContainerId"`
}

// EventCondition 描述事件选项的运行时前置条件。
type EventCondition struct {
	Type     string  `json:"type"`
	Operator string  `json:"operator"`
	Ref      string  `json:"ref,omitempty"`
	Value    float64 `json:"value,omitempty"`
}

// EventCheck 描述事件出现后的属性或固定概率判定。
type EventCheck struct {
	Type         string `json:"type"`
	Attribute    string `json:"attribute,omitempty"`
	Target       int    `json:"target,omitempty"`
	ItemBonusRef string `json:"itemBonusRef,omitempty"`
	ItemBonus    int    `json:"itemBonus,omitempty"`
}

// EventEffect 是事件引擎支持的强类型效果，不执行任意脚本；container_pool 按当前节点奖励池权重抽取容器。
type EventEffect struct {
	Type  string `json:"type"`
	Ref   string `json:"ref,omitempty"`
	Value int    `json:"value,omitempty"`
}

// EventOption 是系统根据探索/撤离模式自动选择的事件处理方案。
type EventOption struct {
	ID             string           `json:"id"`
	Modes          []string         `json:"modes,omitempty"`
	Styles         []string         `json:"styles,omitempty"`
	Intent         string           `json:"intent,omitempty"`    // 自动决策意图，如 bypass/search/withdraw
	RiskTier       int              `json:"riskTier,omitempty"`  // 方案风险等级，1-5
	ValueTier      int              `json:"valueTier,omitempty"` // 方案收益等级，1-5
	StyleBias      map[string]int   `json:"styleBias,omitempty"`
	CheckBonus     map[string]int   `json:"checkBonus,omitempty"`
	Priority       int              `json:"priority"`
	Conditions     []EventCondition `json:"conditions,omitempty"`
	Check          EventCheck       `json:"check"`
	SuccessText    string           `json:"successText"`
	FailureText    string           `json:"failureText"`
	SuccessEffects []EventEffect    `json:"successEffects,omitempty"`
	FailureEffects []EventEffect    `json:"failureEffects,omitempty"`
}

// EventDef 通用事件定义，地图与节点通过 EventBinding 绑定。
type EventDef struct {
	ID             string        `gorm:"primaryKey" json:"id"`
	Name           string        `json:"name"`
	Desc           string        `json:"desc"`
	Category       string        `json:"category"`
	Tags           []string      `gorm:"serializer:json" json:"tags"`
	ExclusiveGroup string        `json:"exclusiveGroup"`
	RepeatPolicy   string        `json:"repeatPolicy"` // repeat/once_per_node/once_per_run
	Options        []EventOption `gorm:"serializer:json" json:"options"`
}

// EventBinding 将通用事件绑定到全局、地图标签、地图、节点标签或具体节点。
type EventBinding struct {
	ID            string `gorm:"primaryKey" json:"id"`
	EventID       string `gorm:"index;not null" json:"eventId"`
	ScopeType     string `gorm:"index;not null" json:"scopeType"`
	ScopeID       string `gorm:"index" json:"scopeId"`
	Phase         string `gorm:"index;not null" json:"phase"`
	TriggerBP     int    `json:"triggerBp"`
	Weight        int    `json:"weight"`
	Priority      int    `json:"priority"`
	MaxPerRun     int    `json:"maxPerRun"`
	CooldownNodes int    `json:"cooldownNodes"`
	Enabled       bool   `json:"enabled"`
}

// EncounterPoolEntry 将通用敌人角色映射到具体地图敌人。
type EncounterPoolEntry struct {
	ID      string `gorm:"primaryKey" json:"id"`
	MapID   string `gorm:"uniqueIndex:idx_encounter_pool,priority:1;not null" json:"mapId"`
	Role    string `gorm:"uniqueIndex:idx_encounter_pool,priority:2;not null" json:"role"`
	EnemyID string `gorm:"uniqueIndex:idx_encounter_pool,priority:3;not null" json:"enemyId"`
	Weight  int    `json:"weight"`
}

// Session 挂机会话
type Session struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	CharacterID     uint       `json:"characterId"`
	MapID           string     `json:"mapId"`
	Style           string     `json:"style"`          // 行动风格：balanced/stealth/aggressive/greedy
	RecoveryPreset  int        `json:"recoveryPreset"` // 失能后使用的预设装备序号 1-3
	WeaponID        string     `json:"weaponId"`
	ArmorID         string     `json:"armorId"`
	Consumables     string     `json:"consumables"` // csv
	Status          string     `json:"status"`      // running/waiting_injury/finished/aborted/failed
	Seed            int64      `json:"seed"`
	StartTime       time.Time  `json:"startTime"`
	EndTime         *time.Time `json:"endTime"`
	OfflineLimitMin int        `json:"offlineLimitMin"` // 分钟
	TotalRuns       int        `json:"totalRuns"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// SessionRun 单局
type SessionRun struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SessionID   uint      `json:"sessionId"`
	RunIndex    int       `json:"runIndex"`
	Result      string    `json:"result"` // success/partial/emergency/captured/incapacitated
	DurationMin int       `json:"durationMin"`
	Heat        int       `json:"heat"`
	AmmoUsed    int       `json:"ammoUsed"`
	Injury      string    `json:"injury"`
	Loot        string    `json:"loot"`   // JSON array of extracted loot
	Report      string    `json:"report"` // JSON array of lines
	CreatedAt   time.Time `json:"createdAt"`
}

// PlayerLoadout 保存当前携行装备及失能后使用的 3 套预设补购清单。
type PlayerLoadout struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	CharacterID        uint      `gorm:"uniqueIndex;not null" json:"characterId"`
	WeaponID           string    `json:"weaponId"`
	ArmorID            string    `json:"armorId"`
	ChestRigID         string    `json:"chestRigId"`
	BackpackID         string    `json:"backpackId"`
	HelmetID           string    `json:"helmetId"`
	HeadsetID          string    `json:"headsetId"`
	Consumables        []string  `gorm:"serializer:json" json:"consumables"`
	PresetWeaponID     string    `json:"presetWeaponId"`
	PresetArmorID      string    `json:"presetArmorId"`
	PresetChestRigID   string    `json:"presetChestRigId"`
	PresetBackpackID   string    `json:"presetBackpackId"`
	PresetHelmetID     string    `json:"presetHelmetId"`
	PresetHeadsetID    string    `json:"presetHeadsetId"`
	PresetName         string    `json:"presetName"`
	PresetConsumables  []string  `gorm:"serializer:json" json:"presetConsumables"`
	Preset2WeaponID    string    `json:"preset2WeaponId"`
	Preset2ArmorID     string    `json:"preset2ArmorId"`
	Preset2ChestRigID  string    `json:"preset2ChestRigId"`
	Preset2BackpackID  string    `json:"preset2BackpackId"`
	Preset2HelmetID    string    `json:"preset2HelmetId"`
	Preset2HeadsetID   string    `json:"preset2HeadsetId"`
	Preset2Name        string    `json:"preset2Name"`
	Preset2Consumables []string  `gorm:"serializer:json" json:"preset2Consumables"`
	Preset3WeaponID    string    `json:"preset3WeaponId"`
	Preset3ArmorID     string    `json:"preset3ArmorId"`
	Preset3ChestRigID  string    `json:"preset3ChestRigId"`
	Preset3BackpackID  string    `json:"preset3BackpackId"`
	Preset3HelmetID    string    `json:"preset3HelmetId"`
	Preset3HeadsetID   string    `json:"preset3HeadsetId"`
	Preset3Name        string    `json:"preset3Name"`
	Preset3Consumables []string  `gorm:"serializer:json" json:"preset3Consumables"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Inventory 仓库
type Inventory struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	ItemID   string `gorm:"uniqueIndex:idx_inv_src,priority:1;not null" json:"itemId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`     // currency/material/loot/weapon/armor/consumable/chestrig/backpack/helmet/headset
	Category string `json:"category"` // loot 子分类
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
	Weight   int    `json:"weight"` // 单件重量 kg
	Slots    int    `json:"slots"`  // 单件占用格数
	// 局内带出：探索掉落获得为 true，市场购买获得为 false（决定能否出售给商人）
	RaidExtract bool `gorm:"uniqueIndex:idx_inv_src,priority:2" json:"raidExtract"`
	// 商人类别与解锁所需好感度
	MerchantCategory string    `json:"merchantCategory"`
	RepRequirement   int       `json:"repRequirement"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// MerchantDef 商人
type MerchantDef struct {
	ID         string `gorm:"primaryKey" json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"` // weapon/clothing/medical/mechanical/black/union
	Reputation int    `json:"reputation"`
	Desc       string `json:"desc"`
	Open       bool   `json:"open"` // 占位商人暂不开放交易
	SortOrder  int    `json:"sortOrder"`
}

// ArmorInstance 护甲实例（耐久）
type ArmorInstance struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	ArmorID       string `json:"armorId"`
	MaxDurability int    `json:"maxDurability"`
	CurDurability int    `json:"curDurability"`
	RepairCount   int    `json:"repairCount"` // 0/1
	Status        string `json:"status"`      // normal/repairing/broken
}
