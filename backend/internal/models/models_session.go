// Session 与持久化运行记录模型：承载行动状态、事件、库存和装备配置。
package models

import "time"

// Session 挂机会话
type Session struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	UserID              uint       `gorm:"index;not null" json:"userId"`
	CharacterID         uint       `json:"characterId"`
	MapID               string     `json:"mapId"`
	Style               string     `json:"style"`          // 行动风格：balanced/stealth/aggressive/greedy
	RecoveryPreset      int        `json:"recoveryPreset"` // 失能后使用的预设装备序号 1-3
	WeaponID            string     `json:"weaponId"`
	ArmorID             string     `json:"armorId"`
	ArmorInstanceID     uint       `json:"armorInstanceId"`
	AmmoID              string     `json:"ammoId"`
	AmmoRounds          int        `json:"ammoRounds"`
	Consumables         string     `json:"consumables"` // csv
	Status              string     `json:"status"`      // running/success/incapacitated
	TerminalReason      string     `json:"terminalReason"`
	RecoveryPolicyJSON  string     `json:"-"`
	Seed                int64      `json:"seed"`
	StartTime           time.Time  `json:"startTime"`
	EndTime             *time.Time `json:"endTime"`
	OfflineLimitMin     int        `json:"offlineLimitMin"`     // 离线调度窗口展示值；精确计算以 OfflineLimitSec 为准
	ElapsedMin          int        `json:"elapsedMin"`          // 游戏时间展示值；精确进度以 ElapsedSec 为准
	OfflineLimitSec     int64      `json:"offlineLimitSec"`     // 离线调度窗口的时间轴秒数，截止判断按局结束时间
	ElapsedSec          int64      `json:"elapsedSec"`          // 已结算游戏时间轴秒数
	CurrentRunStartedAt *time.Time `json:"currentRunStartedAt"` // 本局开始时间
	NextRunAt           *time.Time `json:"nextRunAt"`           // 本局预计结束/结算时间
	LastProcessedAt     *time.Time `json:"lastProcessedAt"`     // 上一局完成结算时间
	LeaseOwner          string     `json:"-"`
	LeaseUntil          *time.Time `json:"-"`
	HeartbeatAt         *time.Time `json:"heartbeatAt"`
	EngineVersion       string     `json:"engineVersion"`
	ScenarioSnapshot    string     `json:"-"`
	ScenarioHash        string     `json:"scenarioHash"`
	InitialStateJSON    string     `json:"-"`
	StateJSON           string     `json:"-"`
	PendingRunIndex     int        `json:"pendingRunIndex"`
	PendingRunResult    string     `json:"-"`
	PendingRunHash      string     `json:"pendingRunHash"`
	TotalRuns           int        `json:"totalRuns"`
	CreatedAt           time.Time  `json:"createdAt"`
}

// SessionRun 单局
type SessionRun struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	UserID              uint      `gorm:"index;not null" json:"userId"`
	SessionID           uint      `gorm:"uniqueIndex:idx_session_runs_session_run,priority:1" json:"sessionId"`
	RunIndex            int       `gorm:"uniqueIndex:idx_session_runs_session_run,priority:2" json:"runIndex"`
	Result              string    `json:"result"`      // success/incapacitated
	DurationMin         int       `json:"durationMin"` // 游戏时间向上取整的分钟展示值
	DurationSec         int64     `json:"durationSec"` // 单局游戏时间轴秒数，权威精度
	Heat                int       `json:"heat"`
	AmmoUsed            int       `json:"ammoUsed"`
	StartHP             float64   `json:"startHp"`
	EndHP               float64   `json:"endHp"`
	StartEnergy         float64   `json:"startEnergy"`
	EndEnergy           float64   `json:"endEnergy"`
	StartHydration      float64   `json:"startHydration"`
	EndHydration        float64   `json:"endHydration"`
	ItemInstanceChanges string    `json:"itemInstanceChanges"`
	Loot                string    `json:"loot"` // JSON array of extracted loot
	StoredLoot          string    `json:"storedLoot"`
	OverflowLoot        string    `json:"overflowLoot"`
	Consumed            string    `gorm:"column:consumed_items" json:"consumedItems"`
	InputState          string    `json:"-"`
	NextState           string    `json:"-"`
	Report              string    `json:"report"` // JSON array of lines
	CreatedAt           time.Time `json:"createdAt"`
}

// SessionEvent 是可在线推送、可离线回放的结构化探索事件。
type SessionEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"userId"`
	SessionID   uint      `gorm:"index;not null" json:"sessionId"`
	RunIndex    int       `gorm:"not null" json:"runIndex"`
	Sequence    int       `gorm:"not null" json:"sequence"`
	EventType   string    `gorm:"column:event_type;not null" json:"eventType"`
	OffsetSec   int64     `gorm:"column:offset_sec;not null" json:"offsetSec"`
	AvailableAt time.Time `gorm:"column:available_at;not null;index" json:"availableAt"`
	NodeID      string    `gorm:"column:node_id" json:"nodeId"`
	SubjectID   string    `gorm:"column:subject_id" json:"subjectId"`
	PayloadJSON string    `gorm:"column:payload_json;not null" json:"-"`
	CreatedAt   time.Time `gorm:"column:created_at;not null" json:"createdAt"`
}

// PlayerLoadout 保存当前携行装备及失能后使用的 3 套预设补购清单。
type PlayerLoadout struct {
	ID                    uint             `gorm:"primaryKey" json:"id"`
	UserID                uint             `gorm:"index;not null" json:"userId"`
	CharacterID           uint             `gorm:"uniqueIndex;not null" json:"characterId"`
	WeaponID              string           `json:"weaponId"`
	ArmorID               string           `json:"armorId"`
	ArmorInstanceID       uint             `json:"armorInstanceId"`
	ChestRigID            string           `json:"chestRigId"`
	BackpackID            string           `json:"backpackId"`
	HelmetID              string           `json:"helmetId"`
	HeadsetID             string           `json:"headsetId"`
	KeyCaseID             string           `json:"keyCaseId"`
	SecureContainerID     string           `json:"secureContainerId"`
	Consumables           []string         `gorm:"serializer:json" json:"consumables"`
	ConsumableRefs        []LoadoutItemRef `gorm:"serializer:json" json:"consumableRefs"`
	CarriedAmmo           []AmmoCell       `gorm:"serializer:json" json:"carriedAmmo"` // 随身携带弹药：最多 4 格，每格 1-60 发
	PresetWeaponID        string           `json:"presetWeaponId"`
	PresetArmorID         string           `json:"presetArmorId"`
	PresetChestRigID      string           `json:"presetChestRigId"`
	PresetBackpackID      string           `json:"presetBackpackId"`
	PresetHelmetID        string           `json:"presetHelmetId"`
	PresetHeadsetID       string           `json:"presetHeadsetId"`
	PresetName            string           `json:"presetName"`
	PresetConsumables     []string         `gorm:"serializer:json" json:"presetConsumables"`
	PresetConsumableRefs  []LoadoutItemRef `gorm:"serializer:json" json:"presetConsumableRefs"`
	PresetAmmoID          string           `json:"presetAmmoId"`
	PresetAmmoRounds      int              `json:"presetAmmoRounds"`
	Preset2WeaponID       string           `json:"preset2WeaponId"`
	Preset2ArmorID        string           `json:"preset2ArmorId"`
	Preset2ChestRigID     string           `json:"preset2ChestRigId"`
	Preset2BackpackID     string           `json:"preset2BackpackId"`
	Preset2HelmetID       string           `json:"preset2HelmetId"`
	Preset2HeadsetID      string           `json:"preset2HeadsetId"`
	Preset2Name           string           `json:"preset2Name"`
	Preset2Consumables    []string         `gorm:"serializer:json" json:"preset2Consumables"`
	Preset2ConsumableRefs []LoadoutItemRef `gorm:"serializer:json" json:"preset2ConsumableRefs"`
	Preset2AmmoID         string           `json:"preset2AmmoId"`
	Preset2AmmoRounds     int              `json:"preset2AmmoRounds"`
	Preset3WeaponID       string           `json:"preset3WeaponId"`
	Preset3ArmorID        string           `json:"preset3ArmorId"`
	Preset3ChestRigID     string           `json:"preset3ChestRigId"`
	Preset3BackpackID     string           `json:"preset3BackpackId"`
	Preset3HelmetID       string           `json:"preset3HelmetId"`
	Preset3HeadsetID      string           `json:"preset3HeadsetId"`
	Preset3Name           string           `json:"preset3Name"`
	Preset3Consumables    []string         `gorm:"serializer:json" json:"preset3Consumables"`
	Preset3ConsumableRefs []LoadoutItemRef `gorm:"serializer:json" json:"preset3ConsumableRefs"`
	Preset3AmmoID         string           `json:"preset3AmmoId"`
	Preset3AmmoRounds     int              `json:"preset3AmmoRounds"`
	UpdatedAt             time.Time        `json:"updatedAt"`
}

// LoadoutItemRef 是装备配置中对聚合补给或耐久实例的引用。
type LoadoutItemRef struct {
	InstanceID uint   `json:"instanceId"`
	ItemID     string `json:"itemId"`
	Quantity   int    `json:"quantity"`
}

// AmmoCell 是随身携带弹药的一个槽位：弹药 ID + 携弹发数（1-60）。
type AmmoCell struct {
	AmmoID string `json:"ammoId"`
	Rounds int    `json:"rounds"`
}

// RecoveryPlan 是 Session 结束后自动执行的恢复计划。
type RecoveryPlan struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"index;not null" json:"userId"`
	SourceSessionID uint       `gorm:"index;not null" json:"sourceSessionId"`
	Status          string     `gorm:"index;not null" json:"status"`
	PolicyJSON      string     `json:"-"`
	StartedAt       time.Time  `json:"startedAt"`
	CompletedAt     *time.Time `json:"completedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// RecoveryTask 保存单项资源的自动恢复进度。
type RecoveryTask struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	RecoveryPlanID uint       `gorm:"index;not null" json:"recoveryPlanId"`
	UserID         uint       `gorm:"index;not null" json:"userId"`
	ResourceType   string     `gorm:"index;not null" json:"resourceType"`
	StartValue     float64    `json:"startValue"`
	CurrentValue   float64    `json:"currentValue"`
	TargetValue    float64    `json:"targetValue"`
	RatePerHour    float64    `json:"ratePerHour"`
	PrimaryMethod  string     `json:"primaryMethod"`
	ActualMethod   string     `json:"actualMethod"`
	StartedAt      time.Time  `json:"startedAt"`
	CompleteAt     *time.Time `json:"completeAt"`
	Status         string     `gorm:"index;not null" json:"status"`
	DetailJSON     string     `json:"-"`
}

// Inventory 仓库
type Inventory struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `gorm:"index;not null" json:"userId"`
	ItemID   string `gorm:"uniqueIndex:idx_inv_user_item_src,priority:2;not null" json:"itemId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`     // currency/material/loot/weapon/armor/ammo/consumable/chestrig/backpack/helmet/headset
	Category string `json:"category"` // loot 子分类
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
	Weight   int    `json:"weight"` // 单件重量 kg
	Slots    int    `json:"slots"`  // 单件占用格数
	// 局内带出：探索掉落获得为 true，市场购买获得为 false（决定能否出售给商人）
	RaidExtract bool `gorm:"uniqueIndex:idx_inv_user_item_src,priority:3" json:"raidExtract"`
	// 商人类别与解锁所需好感度
	MerchantCategory string    `json:"merchantCategory"`
	RepRequirement   int       `json:"repRequirement"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// ItemUseDef 描述物品在行动和藏身处恢复中的统一效果。
type ItemUseDef struct {
	ItemID            string    `gorm:"primaryKey" json:"itemId"`
	HPRecovery        float64   `json:"hpRecovery"`
	EnergyRecovery    float64   `json:"energyRecovery"`
	HydrationRecovery float64   `json:"hydrationRecovery"`
	RepairValue       float64   `json:"repairValue"`
	FuelSeconds       int64     `json:"fuelSeconds"`
	MaxDurability     float64   `json:"maxDurability"`
	UseDurability     float64   `json:"useDurability"`
	UsePriority       int       `json:"usePriority"`
	InstanceRequired  bool      `json:"instanceRequired"`
	UsableInSession   bool      `json:"usableInSession"`
	UsableInHideout   bool      `json:"usableInHideout"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// ItemInstance 是医疗包、维修包和燃料罐等带耐久物品的持久化实例。
type ItemInstance struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index;not null" json:"userId"`
	ItemID            string    `gorm:"index;not null" json:"itemId"`
	CurrentDurability float64   `json:"currentDurability"`
	MaxDurability     float64   `json:"maxDurability"`
	Status            string    `gorm:"index;not null" json:"status"`
	LocationType      string    `gorm:"index;not null" json:"locationType"`
	LocationRef       string    `gorm:"index" json:"locationRef"`
	SlotIndex         int       `json:"slotIndex"`
	RaidExtract       bool      `json:"raidExtract"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
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

// UserMerchantState 用户维度的商人好感度与解锁状态。
type UserMerchantState struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"uniqueIndex:idx_user_merchant,priority:1;not null" json:"userId"`
	MerchantID string    `gorm:"uniqueIndex:idx_user_merchant,priority:2;not null" json:"merchantId"`
	Reputation int       `json:"reputation"`
	Unlocked   bool      `json:"unlocked"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// UserBlackMarketOffer 玩家当前周期的黑市货架；每 6 小时按掉落权重重抽。
type UserBlackMarketOffer struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"uniqueIndex:idx_user_black_offer,priority:1;index:idx_user_black_cycle;not null" json:"userId"`
	ItemID     string    `gorm:"uniqueIndex:idx_user_black_offer,priority:2;not null" json:"itemId"`
	Quantity   int       `json:"quantity"`
	CycleStart time.Time `gorm:"index:idx_user_black_cycle;not null" json:"cycleStart"`
}

// ArmorInstance 护甲实例（耐久）
type ArmorInstance struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	UserID        uint   `gorm:"index;not null" json:"userId"`
	ArmorID       string `json:"armorId"`
	MaxDurability int    `json:"maxDurability"`
	CurDurability int    `json:"curDurability"`
	RepairCount   int    `json:"repairCount"` // 0/1
	Status        string `json:"status"`      // normal/repairing/broken
}
