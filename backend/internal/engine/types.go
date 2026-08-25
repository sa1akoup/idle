// 探索纯引擎的领域 DTO：只描述模拟输入、场景配置与结果，不依赖数据库模型。
package engine

const (
	SchemaVersion = "exploration-snapshot-v4"
	EngineVersion = "exploration-engine-v4"
)

// Map 是探索路线的不可变快照。
type Map struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Desc          string   `json:"desc"`
	StartNodeID   string   `json:"startNodeId"`
	LayoutColumns int      `json:"layoutColumns"`
	LayoutRows    int      `json:"layoutRows"`
	Tags          []string `json:"tags"`
}

// Node 是路线节点的不可变快照。
type Node struct {
	ID             string   `json:"id"`
	MapID          string   `json:"mapId"`
	Name           string   `json:"name"`
	PositionX      int      `json:"positionX"`
	PositionY      int      `json:"positionY"`
	ExploreTime    int      `json:"exploreTime"`
	Distance       string   `json:"distance"`
	EnemyID        string   `json:"enemyId"`
	EncounterRole  string   `json:"encounterRole"`
	ContainerSlots int      `json:"containerSlots"`
	ValueTier      int      `json:"valueTier"`
	Tags           []string `json:"tags"`
}

// MapEdge 是固定在快照中的节点移动边。
type MapEdge struct {
	ID            uint   `json:"id"`
	MapID         string `json:"mapId"`
	FromNodeID    string `json:"fromNodeId"`
	ToNodeID      string `json:"toNodeId"`
	MoveTime      int    `json:"moveTime"`
	Bidirectional bool   `json:"bidirectional"`
}

// ExtractionPoint 是固定在快照中的地图外撤离终点。
type ExtractionPoint struct {
	ID           string   `json:"id"`
	MapID        string   `json:"mapId"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	AnchorNodeID string   `json:"anchorNodeId"`
	TravelTime   int      `json:"travelTime"`
	Enabled      bool     `json:"enabled"`
	IconKey      string   `json:"iconKey"`
	Tags         []string `json:"tags"`
}

// NodeContainerAssignment 描述普通搜索池和事件奖励池的挂载关系。
type NodeContainerAssignment struct {
	ID          uint   `json:"id"`
	NodeID      string `json:"nodeId"`
	ContainerID string `json:"containerId"`
	Pool        string `json:"pool"`
	Count       int    `json:"count"`
	Weight      int    `json:"weight"`
}

type Container struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Tags       []string        `json:"tags"`
	ValueTier  int             `json:"valueTier"`
	SearchRisk int             `json:"searchRisk"`
	SearchTime int             `json:"searchTime"`
	RollMin    int             `json:"rollMin"`
	RollMax    int             `json:"rollMax"`
	Rules      []ContainerRule `json:"rules"`
}

type ContainerRule struct {
	ID           uint   `json:"id"`
	ItemCategory string `json:"itemCategory"`
	Weight       int    `json:"weight"`
	MinQuantity  int    `json:"minQuantity"`
	MaxQuantity  int    `json:"maxQuantity"`
}

// LootItem 是探索产出与库存结算共用的物品快照。
type LootItem struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Category         string `json:"category"`
	Desc             string `json:"desc"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	DropWeight       int    `json:"dropWeight"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// ItemDefinition 覆盖胸挂、背包、头盔、耳机、消耗品及武器护甲的结算配置。
type ItemDefinition struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	Name             string `json:"name"`
	Category         string `json:"category"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	AddSlots         int    `json:"addSlots"`
	AddWeight        int    `json:"addWeight"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
	ArmorMax         int    `json:"armorMax"`
}

type Weapon struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Category         string `json:"category"`
	CaliberID        string `json:"caliberId"`
	Hit              int    `json:"hit"`
	Damage           int    `json:"damage"`
	Penetration      int    `json:"penetration"`
	Suppress         int    `json:"suppress"`
	Ready            int    `json:"ready"`
	AmmoPerRound     int    `json:"ammoPerRound"`
	Noise            int    `json:"noise"`
	Reliability      int    `json:"reliability"`
	CloseMod         int    `json:"closeMod"`
	MidMod           int    `json:"midMod"`
	FarMod           int    `json:"farMod"`
	Price            int    `json:"price"`
	Weight           int    `json:"weight"`
	Slots            int    `json:"slots"`
	MerchantCategory string `json:"merchantCategory"`
	RepRequirement   int    `json:"repRequirement"`
}

// Ammo 是固定在场景快照中的分级弹药配置。
type Ammo struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	CaliberID             string  `json:"caliberId"`
	Level                 int     `json:"level"`
	FleshDamageMultiplier float64 `json:"fleshDamageMultiplier"`
	ArmorDamageMultiplier float64 `json:"armorDamageMultiplier"`
	Price                 int     `json:"price"`
	RoundsPerSlot         int     `json:"roundsPerSlot"`
	MerchantCategory      string  `json:"merchantCategory"`
	RepRequirement        int     `json:"repRequirement"`
}

// AmmoSupply 固定 Session 创建时武器商人的弹药价格和解锁状态，供局间自动补给使用。
type AmmoSupply struct {
	AmmoID    string `json:"ammoId"`
	CaliberID string `json:"caliberId"`
	Level     int    `json:"level"`
	UnitPrice int    `json:"unitPrice"`
	Available bool   `json:"available"`
}

type Armor struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Protect          int    `json:"protect"`
	ProtectionLevel  int    `json:"protectionLevel"`
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

type Enemy struct {
	ID                  string `json:"id"`
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

type EventCondition struct {
	Type     string  `json:"type"`
	Operator string  `json:"operator"`
	Ref      string  `json:"ref,omitempty"`
	Value    float64 `json:"value,omitempty"`
}

type EventCheck struct {
	Type         string `json:"type"`
	Attribute    string `json:"attribute,omitempty"`
	Target       int    `json:"target,omitempty"`
	ItemBonusRef string `json:"itemBonusRef,omitempty"`
	ItemBonus    int    `json:"itemBonus,omitempty"`
}

type EventEffect struct {
	Type  string `json:"type"`
	Ref   string `json:"ref,omitempty"`
	Value int    `json:"value,omitempty"`
}

type EventOption struct {
	ID             string           `json:"id"`
	Modes          []string         `json:"modes,omitempty"`
	Styles         []string         `json:"styles,omitempty"`
	Intent         string           `json:"intent,omitempty"`
	RiskTier       int              `json:"riskTier,omitempty"`
	ValueTier      int              `json:"valueTier,omitempty"`
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

type EventDefinition struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Desc           string        `json:"desc"`
	Category       string        `json:"category"`
	Tags           []string      `json:"tags"`
	ExclusiveGroup string        `json:"exclusiveGroup"`
	RepeatPolicy   string        `json:"repeatPolicy"`
	Options        []EventOption `json:"options"`
}

type EventBinding struct {
	ID            string `json:"id"`
	EventID       string `json:"eventId"`
	ScopeType     string `json:"scopeType"`
	ScopeID       string `json:"scopeId"`
	Phase         string `json:"phase"`
	TriggerBP     int    `json:"triggerBp"`
	Weight        int    `json:"weight"`
	Priority      int    `json:"priority"`
	MaxPerRun     int    `json:"maxPerRun"`
	CooldownNodes int    `json:"cooldownNodes"`
	Enabled       bool   `json:"enabled"`
}

type EncounterPoolEntry struct {
	ID      string `json:"id"`
	MapID   string `json:"mapId"`
	Role    string `json:"role"`
	EnemyID string `json:"enemyId"`
	Weight  int    `json:"weight"`
}

type EventCatalog struct {
	Definitions    map[string]EventDefinition      `json:"definitions"`
	Bindings       []EventBinding                  `json:"bindings"`
	EncounterPools map[string][]EncounterPoolEntry `json:"encounterPools"`
}

type StylePolicy struct {
	ID                string         `json:"id"`
	Label             string         `json:"label"`
	Description       string         `json:"description"`
	HealthEvacRatio   float64        `json:"healthEvacRatio"`
	StressEvacRatio   float64        `json:"stressEvacRatio"`
	CarryEvacRatio    float64        `json:"carryEvacRatio"`
	PatrolApproach    string         `json:"patrolApproach"`
	ValueWeight       int            `json:"valueWeight"`
	RiskWeight        int            `json:"riskWeight"`
	MoveTimeWeight    int            `json:"moveTimeWeight"`
	ExploreTimeWeight int            `json:"exploreTimeWeight"`
	LengthWeight      int            `json:"lengthWeight"`
	IntentBias        map[string]int `json:"intentBias"`
	CheckIntentBonus  map[string]int `json:"checkIntentBonus"`
}

type RecoveryItem struct {
	ItemID    string `json:"itemId"`
	Quantity  int    `json:"quantity"`
	UnitPrice int    `json:"unitPrice"`
	Available bool   `json:"available"`
}

type RecoveryPreset struct {
	Index       int            `json:"index"`
	Loadout     LoadoutState   `json:"loadout"`
	AmmoID      string         `json:"ammoId"`
	AmmoRounds  int            `json:"ammoRounds"`
	Consumables []ItemStack    `json:"consumables"`
	Items       []RecoveryItem `json:"items"`
}

// ScenarioSnapshot 是启动 Session 时固定下来的全部运行配置。
type ScenarioSnapshot struct {
	SchemaVersion            string                    `json:"schemaVersion"`
	Map                      Map                       `json:"map"`
	Nodes                    []Node                    `json:"nodes"`
	Edges                    []MapEdge                 `json:"edges"`
	ExtractionPoints         []ExtractionPoint         `json:"extractionPoints"`
	NodeContainerAssignments []NodeContainerAssignment `json:"nodeContainerAssignments"`
	Containers               map[string]Container      `json:"containers"`
	LootItems                map[string]LootItem       `json:"lootItems"`
	Items                    map[string]ItemDefinition `json:"items"`
	Weapons                  map[string]Weapon         `json:"weapons"`
	Ammos                    map[string]Ammo           `json:"ammos"`
	AmmoSupplies             map[string]AmmoSupply     `json:"ammoSupplies"`
	Armors                   map[string]Armor          `json:"armors"`
	Enemies                  map[string]Enemy          `json:"enemies"`
	Events                   EventCatalog              `json:"events"`
	Styles                   []StylePolicy             `json:"styles"`
	RecoveryPresets          map[int]RecoveryPreset    `json:"recoveryPresets"`
}

type CharacterState struct {
	Name        string `json:"name"`
	Strength    int    `json:"strength"`
	Agility     int    `json:"agility"`
	Intellect   int    `json:"intellect"`
	Charisma    int    `json:"charisma"`
	Stealth     int    `json:"stealth"`
	Perception  int    `json:"perception"`
	Negotiation int    `json:"negotiation"`
	Luck        int    `json:"luck"`
	Survival    int    `json:"survival"`
	Resist      int    `json:"resist"`
	Engineering int    `json:"engineering"`
	Medical     int    `json:"medical"`
	MeleeProf   int    `json:"meleeProf"`
	PistolProf  int    `json:"pistolProf"`
	SMGProf     int    `json:"smgProf"`
	ShotgunProf int    `json:"shotgunProf"`
	RifleProf   int    `json:"rifleProf"`
	SniperProf  int    `json:"sniperProf"`
	Trait       string `json:"trait"`
	Fatigue     int    `json:"fatigue"`
	Stress      int    `json:"stress"`
	Injury      string `json:"injury"`
}

type ItemStack struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type LoadoutState struct {
	WeaponID   string `json:"weaponId"`
	ArmorID    string `json:"armorId"`
	ChestRigID string `json:"chestRigId"`
	BackpackID string `json:"backpackId"`
	HelmetID   string `json:"helmetId"`
	HeadsetID  string `json:"headsetId"`
}

type CarryState struct {
	TotalSlots  int     `json:"totalSlots"`
	UsedSlots   int     `json:"usedSlots"`
	TotalWeight float64 `json:"totalWeight"`
	UsedWeight  float64 `json:"usedWeight"`
}

// CarriedAmmo 是已从仓库转入 Session 的实际弹药，跨局按发数连续扣减。
type CarriedAmmo struct {
	ID             string `json:"id"`
	CaliberID      string `json:"caliberId"`
	Level          int    `json:"level"`
	Rounds         int    `json:"rounds"`
	PreferredID    string `json:"preferredId"`
	PreferredLevel int    `json:"preferredLevel"`
	TargetRounds   int    `json:"targetRounds"`
}
