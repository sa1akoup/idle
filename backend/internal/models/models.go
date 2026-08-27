package models

import "time"

// DefaultUserID 认证接口接入前使用的本地启动用户。
const DefaultUserID uint = 1

// PlayerCharacterID 保留给本地测试数据与旧结构引用。
const PlayerCharacterID uint = 1

// PlayerLoadoutID 保留给本地测试数据与旧结构引用。
const PlayerLoadoutID uint = 1

// InventoryCapacity 仓库可容纳的非现金物品总数（当前装备与预设装备不计入）。
const InventoryCapacity = 120

// BaseCarrySlots 基础可携带物品格数（不含胸挂/背包加成）。
const BaseCarrySlots = 20

// BaseCarryWeight 基础可携带负重：初始 50kg +（力量-50）*0.3。
func BaseCarryWeight(strength int) float64 {
	return 50 + float64(strength-50)*0.3
}

// User 用户账号。
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	Email        *string   `gorm:"uniqueIndex" json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// AuthSession 登录会话 token 的摘要记录。
type AuthSession struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"userId"`
	TokenHash string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// EconomicOperation 记录可重放的经济操作结果。
type EconomicOperation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"uniqueIndex:idx_economic_operation,priority:1;not null" json:"userId"`
	OperationKey  string    `gorm:"uniqueIndex:idx_economic_operation,priority:2;not null" json:"-"`
	OperationType string    `gorm:"not null" json:"-"`
	ResultJSON    string    `gorm:"not null" json:"-"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Character 角色
type Character struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UserID      uint   `gorm:"uniqueIndex:idx_characters_user_id;not null" json:"userId"`
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

	// 持久化生存资源。HP 使用 Strength 计算动态上限，Energy/Hydration 上限为 100。
	HP             float64   `json:"hp"`
	Energy         float64   `json:"energy"`
	Hydration      float64   `json:"hydration"`
	NeedsUpdatedAt time.Time `json:"needsUpdatedAt"`
	Stress         int       `json:"stress"`
	// ResourceVersion 用于在事务开始时获取该用户的资源行锁。
	ResourceVersion int64     `gorm:"not null;default:0" json:"-"`
	CreatedAt       time.Time `json:"createdAt"`
}

// FacilityRequirement 是设施升级时的单项前置条件。
type FacilityRequirement struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	FacilityID      string  `gorm:"index;not null" json:"facilityId"`
	Level           int     `gorm:"index;not null" json:"level"`
	RequirementType string  `gorm:"not null" json:"requirementType"` // item/facility/trader/skill
	ReferenceID     string  `gorm:"not null" json:"referenceId"`
	Quantity        int     `json:"quantity"`
	RequiredValue   float64 `json:"requiredValue"`
	SortOrder       int     `json:"sortOrder"`
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
	CaliberID    string `gorm:"index" json:"caliberId"`
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

// AmmoDef 弹药定义：同口径使用 N1-N6 穿甲等级区分弹种，价格按单发计算。
type AmmoDef struct {
	ID                    string  `gorm:"primaryKey" json:"id"`
	Name                  string  `json:"name"`
	CaliberID             string  `gorm:"uniqueIndex:idx_ammo_caliber_level,priority:1;not null" json:"caliberId"`
	Level                 int     `gorm:"uniqueIndex:idx_ammo_caliber_level,priority:2;not null" json:"level"`
	FleshDamageMultiplier float64 `json:"fleshDamageMultiplier"`
	ArmorDamageMultiplier float64 `json:"armorDamageMultiplier"`
	Price                 int     `json:"price"` // 单发价格
	RoundsPerSlot         int     `json:"roundsPerSlot"`
	MerchantCategory      string  `json:"merchantCategory"`
	RepRequirement        int     `json:"repRequirement"`
}

// ArmorDef 护甲定义
type ArmorDef struct {
	ID               string `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"` // light/heavy
	Protect          int    `json:"protect"`
	ProtectionLevel  int    `json:"protectionLevel"` // A1-A6
	Coverage         int    `json:"coverage"`        // 0-100
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
	ID            string   `gorm:"primaryKey" json:"id"`
	Name          string   `json:"name"`
	Desc          string   `json:"desc"`
	StartNodeID   string   `json:"startNodeId"`
	LayoutColumns int      `json:"layoutColumns"`
	LayoutRows    int      `json:"layoutRows"`
	Tags          []string `gorm:"serializer:json" json:"tags"`
}

// NodeDef 节点
type NodeDef struct {
	ID             string   `gorm:"primaryKey" json:"id"`
	MapID          string   `json:"mapId"`
	Name           string   `json:"name"`
	PositionX      int      `json:"positionX"`
	PositionY      int      `json:"positionY"`
	ExploreTime    int      `json:"exploreTime"` // 分钟
	Distance       string   `json:"distance"`    // close/mid/far
	EnemyID        string   `json:"enemyId"`
	EncounterRole  string   `json:"encounterRole"`  // patrol/guard/elite 等行为角色
	ContainerSlots int      `json:"containerSlots"` // 本节点每局生成的容器槽位
	ValueTier      int      `json:"valueTier"`      // 节点整体价值等级，1-5
	Tags           []string `gorm:"serializer:json" json:"tags"`
}

// MapEdgeDef 地图节点之间的移动边；Bidirectional 决定是否允许反向通行。
type MapEdgeDef struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	MapID         string `gorm:"uniqueIndex:idx_map_edge,priority:1;not null" json:"mapId"`
	FromNodeID    string `gorm:"uniqueIndex:idx_map_edge,priority:2;not null" json:"fromNodeId"`
	ToNodeID      string `gorm:"uniqueIndex:idx_map_edge,priority:3;not null" json:"toNodeId"`
	MoveTime      int    `gorm:"not null" json:"moveTime"`
	Bidirectional bool   `gorm:"not null;default:true" json:"bidirectional"`
}

// ExtractionPointDef 独立撤离点，通过 AnchorNodeID 挂接到探索图。
type ExtractionPointDef struct {
	ID           string   `gorm:"primaryKey" json:"id"`
	MapID        string   `gorm:"index;not null" json:"mapId"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	AnchorNodeID string   `gorm:"index;not null" json:"anchorNodeId"`
	TravelTime   int      `json:"travelTime"`
	Enabled      bool     `gorm:"not null;default:true" json:"enabled"`
	IconKey      string   `json:"iconKey"`
	Tags         []string `gorm:"serializer:json" json:"tags"`
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
	AmmoID              string `json:"ammoId"`
	AmmoRounds          int    `json:"ammoRounds"`
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
