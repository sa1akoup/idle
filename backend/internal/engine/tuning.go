// 集中调参文件：全部校准系数与基准值在此定义，引擎各公式只读本配置。
// 默认值 = 各阶段展开前的行为（零行为 drift）；后续数值调整只改 DefaultTuning()。
// Tuning 随场景快照固定，保证同版本、同 seed 的确定性重放不受配置漂移影响。
package engine

import "fmt"

// SearchFailPenalty 描述搜索暴露的失败惩罚。Kind 决定惩罚形态，新增形态在
// applySearchFailPenalty 的 switch 中登记即可，其余字段按 Kind 各取所需。
type SearchFailPenalty struct {
	Kind           string  `json:"kind"`           // "expose"：耗时翻倍+热度；预留 "injury"（掉血）/ "lose_item"（丢件）
	TimeMultiplier float64 `json:"timeMultiplier"` // expose：搜索耗时倍率
	Heat           int     `json:"heat"`           // expose：暴露增加的热度
}

// Tuning 是探索引擎全部可调系数的快照配置。
type Tuning struct {
	Combat    CombatTuning    `json:"combat"`
	Encounter EncounterTuning `json:"encounter"`
	Events    EventsTuning    `json:"events"`
	Survival  SurvivalTuning  `json:"survival"`
	Search    SearchTuning    `json:"search"`
	AmmoDrop  AmmoDropTuning  `json:"ammoDrop"`
}

// CombatTuning 战斗判定与伤害压力系数。
type CombatTuning struct {
	// 命中率 = clamp(武器命中 + (操控-50)×HitRateCoef + 距离 + 伏击 - 目标闪避 - 压力×HitRateStressCoef)
	HitRateCoef        float64 `json:"hitRateCoef"`        // 操控偏差系数 0.4
	HitRateStressCoef  float64 `json:"hitRateStressCoef"`  // 压力命中惩罚 0.25
	HitRateMin         float64 `json:"hitRateMin"`         // 5
	HitRateMax         float64 `json:"hitRateMax"`         // 95

	// 压力增量 = max(StressMin, 压制×hitCoeff×StressSuppressCoef×(1-抗压×StressResistCoef) + 实伤×StressDamageCoef - 护甲抗压)
	StressSuppressCoef    float64 `json:"stressSuppressCoef"`    // 交火压力系数 0.2
	StressSuppressMissCoef float64 `json:"stressSuppressMissCoef"` // 未命中时的 hitCoeff 0.5
	StressDamageCoef      float64 `json:"stressDamageCoef"`      // 受创压力系数 0.15
	StressResistCoef      float64 `json:"stressResistCoef"`      // 抗压减压系数 0.005
	StressMin             float64 `json:"stressMin"`             // 单次压力下限 1

	// 先手 = 敏捷×AgilityCoef + 感知×PerceptionCoef + 武器Ready + 护甲先手 - 压力×StressCoef + 伏击加成
	InitiativeAgilityCoef    float64 `json:"initiativeAgilityCoef"`    // 0.35
	InitiativePerceptionCoef float64 `json:"initiativePerceptionCoef"` // 0.35
	InitiativeStressCoef     float64 `json:"initiativeStressCoef"`     // 0.2
	AmbushInitBonus          int     `json:"ambushInitBonus"`          // 提前发现先手 +15
	AmbushStanceBonus        int     `json:"ambushStanceBonus"`        // 风格伏击先手 +10
	ApproachFailInitBonus    int     `json:"approachFailInitBonus"`    // 接近失败时敌人先手 +10
	EscapeFailHitBonus       int     `json:"escapeFailHitBonus"`       // 玩家脱离失败后敌方反击命中 +10

	// 闪避 = clamp(敏捷×AgilityCoef + 护甲机动, 0, Max)
	EvasionAgilityCoef float64 `json:"evasionAgilityCoef"` // 0.12
	EvasionMax         float64 `json:"evasionMax"`         // 35

	// 最大生命 = clamp(100 + (力量-50)×Coef, Min, Max)
	MaxHPBase         float64 `json:"maxHpBase"`         // 100
	MaxHPStrengthCoef float64 `json:"maxHpStrengthCoef"` // 0.2
	MaxHPMin          float64 `json:"maxHpMin"`          // 90
	MaxHPMax          float64 `json:"maxHpMax"`          // 110

	// 有效技能 = 训练×SkillCoef + 主属性×AttrCoef
	SkillCoef float64 `json:"skillCoef"` // 0.75
	AttrCoef  float64 `json:"attrCoef"`  // 0.25
	// 最终武器操控 = 属性操控×ControlAttrCoef + 熟练度×ControlProfCoef
	ControlAttrCoef float64 `json:"controlAttrCoef"` // 0.7
	ControlProfCoef float64 `json:"controlProfCoef"` // 0.3

	// 护甲有效等级随耐久分档：> 高带 / > 低带 / 其余
	ArmorDuraHighBand float64 `json:"armorDuraHighBand"` // 0.50
	ArmorDuraLowBand  float64 `json:"armorDuraLowBand"`  // 0.25

	// 脱离判定 = clamp(50 + (脱离值-追捕值)×Coef, Min, Max)
	EscapeBase float64 `json:"escapeBase"` // 50
	EscapeCoef float64 `json:"escapeCoef"` // 0.8
	EscapeMin  float64 `json:"escapeMin"`  // 10
	EscapeMax  float64 `json:"escapeMax"`  // 95

	// 压力阈值 = 70 + 有效抗压×Coef
	StressThresholdBase       float64 `json:"stressThresholdBase"`       // 70
	StressThresholdResistCoef float64 `json:"stressThresholdResistCoef"` // 0.2
}

// EncounterTuning 发现率、绕行率与节点遭遇概率系数。
type EncounterTuning struct {
	// 侦察 = 感知有效×ReconPerceptionCoef + 智力×ReconIntellectCoef + 装备加成（耳机听力）
	ReconPerceptionCoef float64 `json:"reconPerceptionCoef"` // 0.7
	ReconIntellectCoef  float64 `json:"reconIntellectCoef"`  // 0.1
	// 隐蔽 = 潜行有效×ConcealStealthCoef + 敏捷×ConcealAgilityCoef + 护甲隐蔽
	ConcealStealthCoef float64 `json:"concealStealthCoef"` // 0.7
	ConcealAgilityCoef float64 `json:"concealAgilityCoef"` // 0.1

	// 发现率 = clamp(50 + (侦察-隐蔽)×Coef, Min, Max); 耳机听力分别加减
	FindBase         float64 `json:"findBase"`         // 50
	FindCoef         float64 `json:"findCoef"`         // 0.5
	FindMin          float64 `json:"findMin"`          // 10
	FindMax          float64 `json:"findMax"`          // 90
	HearingPBonus    float64 `json:"hearingPBonus"`    // 玩家发现率 + 听力×3
	HearingEnemyNeg  float64 `json:"hearingEnemyNeg"`  // 敌人发现率 - 听力×2

	// 绕行率 = clamp(50 + (潜行×0.5+敏捷×0.15+隐蔽-敌人感知×0.45)×Coef, Min, Max)
	BypassStealthCoef float64 `json:"bypassStealthCoef"` // 0.5
	BypassAgilityCoef float64 `json:"bypassAgilityCoef"` // 0.15
	BypassAlertCoef   float64 `json:"bypassAlertCoef"`   // 0.45
	BypassCoef        float64 `json:"bypassCoef"`        // 0.5
	BypassMin         float64 `json:"bypassMin"`         // 10
	BypassMax         float64 `json:"bypassMax"`         // 95

	// 接近判定 = clamp(50 + (敏捷×0.35+潜行×0.25+感知×PerceptionCoef+基准-压力×0.15 - 拦截)×Coef, Min, Max)
	ApproachAgilityCoef    float64 `json:"approachAgilityCoef"`    // 0.35
	ApproachStealthCoef    float64 `json:"approachStealthCoef"`    // 0.25
	ApproachBase           float64 `json:"approachBase"`           // 接近固定加成（感知接入后归 0）
	ApproachPerceptionCoef float64 `json:"approachPerceptionCoef"` // 感知加成 0.1（替换原 +6 基准）
	ApproachStressCoef      float64 `json:"approachStressCoef"`      // 0.15
	ApproachBlockPerceptionCoef float64 `json:"approachBlockPerceptionCoef"` // 0.3
	ApproachBlockSuppressCoef   float64 `json:"approachBlockSuppressCoef"`   // 0.3
	ApproachCoef            float64 `json:"approachCoef"`            // 0.8
	ApproachMin             float64 `json:"approachMin"`             // 10
	ApproachMax             float64 `json:"approachMax"`             // 90

	// 节点遭遇概率 = clamp(节点基础值+热度(-撤离修正), Min, Max)
	NodeDefaultChance int `json:"nodeDefaultChance"` // 60（默认节点）
	NodeEvacModifier  int `json:"nodeEvacModifier"`  // 25（撤离模式下调）
	NodeChanceMin     int `json:"nodeChanceMin"`     // 5
	NodeChanceMax     int `json:"nodeChanceMax"`     // 90
}

// EventsTuning 事件属性判定系数：成功率 = clamp(55 + (属性-目标)×Coef, Min, Max)。
type EventsTuning struct {
	CheckBase float64 `json:"checkBase"` // 55
	CheckCoef float64 `json:"checkCoef"` // 1.1
	CheckMin  float64 `json:"checkMin"`  // 5
	CheckMax  float64 `json:"checkMax"`  // 95
}

// SurvivalTuning 生存消耗、自动医疗与压力恢复系数。
type SurvivalTuning struct {
	EnergyDrainPerHour    float64 `json:"energyDrainPerHour"`    // 8
	HydrationDrainPerHour float64 `json:"hydrationDrainPerHour"` // 10
	EnergySurvivalCoef    float64 `json:"energySurvivalCoef"`    // 生存降低能量消耗 0.02
	HydrationSurvivalCoef float64 `json:"hydrationSurvivalCoef"` // 生存降低饮水消耗 0.04
	DrainMin              float64 `json:"drainMin"`              // 消耗下限 5

	AutoHealTrigger      float64 `json:"autoHealTrigger"`      // 血量低于该比例启动自动医疗 0.6
	AutoHealTarget       float64 `json:"autoHealTarget"`       // 回血目标比例 0.8
	AutoHealMedicalCoef  float64 `json:"autoHealMedicalCoef"`  // 医疗属性降低触发阈值 0.002
	AutoRecoverThreshold float64 `json:"autoRecoverThreshold"` // 能量/饮水自动补给阈值 0.8

	StressMoveRecovery    float64 `json:"stressMoveRecovery"`    // 节点间移动减压 5
	StressExploreRecovery float64 `json:"stressExploreRecovery"` // 探索/撤离每分钟减压 5
}

// SearchTuning 搜索风险与运气增产系数。
type SearchTuning struct {
	RiskPerLevel    float64 `json:"riskPerLevel"`    // 每级搜索风险 8
	LuckCoef        float64 `json:"luckCoef"`        // 运气降低暴露率 0.15
	PerceptionCoef  float64 `json:"perceptionCoef"`  // 感知降低暴露率 0.1
	ExposeMin       float64 `json:"exposeMin"`       // 暴露率下限 2
	ExposeMax       float64 `json:"exposeMax"`       // 暴露率上限 40
	LuckBonusCoef   float64 `json:"luckBonusCoef"`   // 运气额外多搜一件的概率系数 0.3
	FailPenalty     SearchFailPenalty `json:"failPenalty"`
}

// AmmoDropTuning 敌人弹药掉落：按组（RoundsPerSlot）折算携带占用。
type AmmoDropTuning struct {
	WeightPerGroup float64 `json:"weightPerGroup"` // 每组重量 kg 0.5
}

// DefaultTuning 返回全部系数的默认值（与各阶段展开前的行为一致）。
func DefaultTuning() Tuning {
	return Tuning{
		Combat: CombatTuning{
			HitRateCoef: 0.4, HitRateStressCoef: 0.25, HitRateMin: 5, HitRateMax: 95,
			StressSuppressCoef: 0.2, StressSuppressMissCoef: 0.5, StressDamageCoef: 0.15,
			StressResistCoef: 0.005, StressMin: 1,
			InitiativeAgilityCoef: 0.35, InitiativePerceptionCoef: 0.35, InitiativeStressCoef: 0.2,
			AmbushInitBonus: 15, AmbushStanceBonus: 10, ApproachFailInitBonus: 10, EscapeFailHitBonus: 10,
			EvasionAgilityCoef: 0.12, EvasionMax: 35,
			MaxHPBase: 100, MaxHPStrengthCoef: 0.2, MaxHPMin: 90, MaxHPMax: 110,
			SkillCoef: 0.75, AttrCoef: 0.25, ControlAttrCoef: 0.7, ControlProfCoef: 0.3,
			ArmorDuraHighBand: 0.50, ArmorDuraLowBand: 0.25,
			EscapeBase: 50, EscapeCoef: 0.8, EscapeMin: 10, EscapeMax: 95,
			StressThresholdBase: 70, StressThresholdResistCoef: 0.2,
		},
		Encounter: EncounterTuning{
			ReconPerceptionCoef: 0.7, ReconIntellectCoef: 0.1,
			ConcealStealthCoef: 0.7, ConcealAgilityCoef: 0.1,
			FindBase: 50, FindCoef: 0.5, FindMin: 10, FindMax: 90,
			HearingPBonus: 3, HearingEnemyNeg: 2,
			BypassStealthCoef: 0.5, BypassAgilityCoef: 0.15, BypassAlertCoef: 0.45,
			BypassCoef: 0.5, BypassMin: 10, BypassMax: 95,
			ApproachAgilityCoef: 0.35, ApproachStealthCoef: 0.25, ApproachBase: 0, ApproachPerceptionCoef: 0.1,
			ApproachStressCoef: 0.15, ApproachBlockPerceptionCoef: 0.3, ApproachBlockSuppressCoef: 0.3,
			ApproachCoef: 0.8, ApproachMin: 10, ApproachMax: 90,
			NodeDefaultChance: 60, NodeEvacModifier: 25, NodeChanceMin: 5, NodeChanceMax: 90,
		},
		Events: EventsTuning{
			CheckBase: 55, CheckCoef: 1.1, CheckMin: 5, CheckMax: 95,
		},
		Survival: SurvivalTuning{
			EnergyDrainPerHour: 8, HydrationDrainPerHour: 10,
			EnergySurvivalCoef: 0.02, HydrationSurvivalCoef: 0.04, DrainMin: 5,
			AutoHealTrigger: 0.6, AutoHealTarget: 0.8, AutoHealMedicalCoef: 0.002,
			AutoRecoverThreshold: 0.8,
			StressMoveRecovery: 5, StressExploreRecovery: 5,
		},
		Search: SearchTuning{
			RiskPerLevel: 8, LuckCoef: 0.15, PerceptionCoef: 0.1,
			ExposeMin: 2, ExposeMax: 40, LuckBonusCoef: 0.3,
			FailPenalty: SearchFailPenalty{Kind: "expose", TimeMultiplier: 2, Heat: 3},
		},
		AmmoDrop: AmmoDropTuning{WeightPerGroup: 0.5},
	}
}

// ValidateTuning 校验集中配置合法（系数非负、钳制区间有序），供快照校验调用。
func ValidateTuning(t Tuning) error {
	groups := []struct {
		name string
		coef float64
	}{
		{"Combat.HitRateCoef", t.Combat.HitRateCoef},
		{"Combat.StressSuppressCoef", t.Combat.StressSuppressCoef},
		{"Combat.StressDamageCoef", t.Combat.StressDamageCoef},
		{"Combat.EscapeCoef", t.Combat.EscapeCoef},
		{"Encounter.FindCoef", t.Encounter.FindCoef},
		{"Encounter.BypassCoef", t.Encounter.BypassCoef},
		{"Encounter.ApproachCoef", t.Encounter.ApproachCoef},
		{"Events.CheckCoef", t.Events.CheckCoef},
		{"Survival.EnergySurvivalCoef", t.Survival.EnergySurvivalCoef},
		{"Survival.HydrationSurvivalCoef", t.Survival.HydrationSurvivalCoef},
		{"Search.RiskPerLevel", t.Search.RiskPerLevel},
		{"Search.LuckBonusCoef", t.Search.LuckBonusCoef},
	}
	for _, group := range groups {
		if group.coef < 0 {
			return fmt.Errorf("调参配置 %s 不能为负", group.name)
		}
	}
	if t.Combat.HitRateMax <= t.Combat.HitRateMin {
		return fmt.Errorf("调参配置 Combat 命中率区间无效")
	}
	if t.Encounter.FindMax <= t.Encounter.FindMin {
		return fmt.Errorf("调参配置 Encounter 发现率区间无效")
	}
	if t.Encounter.BypassMax <= t.Encounter.BypassMin {
		return fmt.Errorf("调参配置 Encounter 绕行率区间无效")
	}
	if t.Encounter.ApproachMax <= t.Encounter.ApproachMin {
		return fmt.Errorf("调参配置 Encounter 接近率区间无效")
	}
	if t.Encounter.NodeChanceMax <= t.Encounter.NodeChanceMin {
		return fmt.Errorf("调参配置 Encounter 节点遭遇率区间无效")
	}
	if t.Events.CheckMax <= t.Events.CheckMin {
		return fmt.Errorf("调参配置 Events 判定区间无效")
	}
	if t.Search.ExposeMax <= t.Search.ExposeMin {
		return fmt.Errorf("调参配置 Search 暴露率区间无效")
	}
	switch t.Search.FailPenalty.Kind {
	case "", "expose":
	default:
		return fmt.Errorf("调参配置 Search 失败惩罚形态 %s 尚未实现", t.Search.FailPenalty.Kind)
	}
	return nil
}