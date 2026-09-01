// 事件配置校验：集中维护事件枚举、方案语义和绑定基础约束，供服务启动与快照复用。
package engine

import "fmt"

// ValidateEventCatalog 校验事件定义与绑定的结构约束，不检查数据库目录引用。
// 目录引用由 service 或 ValidateSnapshot 按各自的资源集合继续校验。
func ValidateEventCatalog(catalog EventCatalog, styles []StylePolicy) error {
	if styles == nil {
		styles = DefaultStylePolicies()
	}
	for definitionID, definition := range catalog.Definitions {
		if definitionID != definition.ID {
			return fmt.Errorf("事件快照 key %s 与 ID %s 不一致", definitionID, definition.ID)
		}
		if err := ValidateEventDefinition(definition, styles); err != nil {
			return err
		}
	}

	bindingIDs := make(map[string]bool, len(catalog.Bindings))
	for _, binding := range catalog.Bindings {
		if err := ValidateEventBinding(binding); err != nil {
			return err
		}
		if bindingIDs[binding.ID] {
			return fmt.Errorf("事件绑定 %s 重复", binding.ID)
		}
		bindingIDs[binding.ID] = true
		if _, ok := catalog.Definitions[binding.EventID]; !ok {
			return fmt.Errorf("事件绑定 %s 引用不存在事件 %s", binding.ID, binding.EventID)
		}
	}
	return nil
}

// ValidateEventDefinition 校验单个事件定义的纯结构约束。
func ValidateEventDefinition(definition EventDefinition, styles []StylePolicy) error {
	if definition.ID == "" {
		return fmt.Errorf("事件定义缺少 ID")
	}
	if len(definition.Options) == 0 {
		return fmt.Errorf("事件 %s 没有处理方案", definition.ID)
	}
	if styles == nil {
		styles = DefaultStylePolicies()
	}

	optionIDs := make(map[string]bool, len(definition.Options))
	for _, option := range definition.Options {
		if option.ID == "" || optionIDs[option.ID] {
			return fmt.Errorf("事件 %s 包含重复或空方案 ID", definition.ID)
		}
		optionIDs[option.ID] = true
		if option.Intent == "" {
			return fmt.Errorf("事件 %s 的方案 %s 缺少决策意图", definition.ID, option.ID)
		}
		if !supportedEventIntent(option.Intent) {
			return fmt.Errorf("事件 %s 的方案 %s 使用未知决策意图 %s", definition.ID, option.ID, option.Intent)
		}
		if option.RiskTier < 1 || option.RiskTier > 5 || option.ValueTier < 1 || option.ValueTier > 5 {
			return fmt.Errorf("事件 %s 的方案 %s 风险或收益等级必须为 1-5", definition.ID, option.ID)
		}
		if err := validateEventCheck(definition.ID, option); err != nil {
			return err
		}
		for _, mode := range option.Modes {
			if !supportedEventMode(mode) {
				return fmt.Errorf("事件 %s 的方案 %s 使用未知模式 %s", definition.ID, option.ID, mode)
			}
		}
		for _, style := range option.Styles {
			if _, err := resolveStyle(styles, style); err != nil {
				return fmt.Errorf("事件 %s 的方案 %s 使用未知行动风格 %s", definition.ID, option.ID, style)
			}
		}
		for _, condition := range option.Conditions {
			if err := validateEventCondition(definition.ID, option.ID, condition); err != nil {
				return err
			}
		}
		for _, effect := range append(append([]EventEffect{}, option.SuccessEffects...), option.FailureEffects...) {
			if err := validateEventEffect(definition.ID, option.ID, effect); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateEventBinding 校验事件绑定的纯结构约束。
func ValidateEventBinding(binding EventBinding) error {
	if binding.ID == "" {
		return fmt.Errorf("事件绑定缺少 ID")
	}
	if binding.EventID == "" {
		return fmt.Errorf("事件绑定 %s 缺少事件 ID", binding.ID)
	}
	if !supportedEventPhase(binding.Phase) {
		return fmt.Errorf("事件绑定 %s 使用未知阶段 %s", binding.ID, binding.Phase)
	}
	// TriggerBP 以 10000 为满概率基数，与事件触发掷骰（1-10000）保持一致。
	if binding.TriggerBP < 0 || binding.TriggerBP > 10000 || binding.Weight < 0 || binding.MaxPerRun < 0 || binding.CooldownNodes < 0 {
		return fmt.Errorf("事件绑定 %s 的概率或限制无效", binding.ID)
	}
	switch binding.ScopeType {
	case "global", "map", "map_tag", "node", "node_tag", "extraction", "extraction_tag":
		return nil
	default:
		return fmt.Errorf("事件绑定 %s 使用未知作用域 %s", binding.ID, binding.ScopeType)
	}
}

// validateEventCheck 校验方案判定类型与判定属性是否受支持。
func validateEventCheck(definitionID string, option EventOption) error {
	switch option.Check.Type {
	case "", "none", "fixed", "attribute":
	default:
		return fmt.Errorf("事件 %s 的方案 %s 使用未知判定类型 %s", definitionID, option.ID, option.Check.Type)
	}
	if option.Check.Type == "attribute" && !supportedEventAttribute(option.Check.Attribute) {
		return fmt.Errorf("事件 %s 的方案 %s 使用未知判定属性 %s", definitionID, option.ID, option.Check.Attribute)
	}
	return nil
}

// validateEventCondition 校验条件类型/操作符合法，引用类条件必须带 Ref。
func validateEventCondition(definitionID, optionID string, condition EventCondition) error {
	if !supportedEventCondition(condition.Type) || !supportedConditionOperator(condition.Operator) {
		return fmt.Errorf("事件 %s 的方案 %s 使用未知条件 %s/%s", definitionID, optionID, condition.Type, condition.Operator)
	}
	if (condition.Type == "has_item" || condition.Type == "flag") && condition.Ref == "" {
		return fmt.Errorf("事件 %s 的方案 %s 条件 %s 缺少引用", definitionID, optionID, condition.Type)
	}
	return nil
}

// validateEventEffect 校验效果类型受支持，且引用类效果必须带 Ref。
func validateEventEffect(definitionID, optionID string, effect EventEffect) error {
	if !supportedEventEffect(effect.Type) {
		return fmt.Errorf("事件 %s 的方案 %s 使用未知效果 %s", definitionID, optionID, effect.Type)
	}
	switch effect.Type {
	case "container", "container_pool", "encounter", "consume_item", "set_flag":
		if effect.Ref == "" {
			return fmt.Errorf("事件 %s 的方案 %s 效果 %s 缺少引用", definitionID, optionID, effect.Type)
		}
	case "start_evacuation":
		if effect.Ref != "" && !supportedEvacuationReason(effect.Ref) {
			return fmt.Errorf("事件 %s 的方案 %s 使用未知撤离原因 %s", definitionID, optionID, effect.Ref)
		}
	case "evac_shortcut":
		if effect.Value <= 0 {
			return fmt.Errorf("事件 %s 的方案 %s 撤离捷径缩短时间必须为正数", definitionID, optionID)
		}
	}
	return nil
}

// supportedEventMode 判断方案适用模式是否属于探索/撤离二者之一。
func supportedEventMode(mode string) bool {
	return mode == runModeExploring || mode == runModeEvacuating
}

// supportedEventAttribute 判断判定引用属性是否为角色属性表内成员。
func supportedEventAttribute(attribute string) bool {
	switch attribute {
	case "strength", "agility", "intellect", "charisma", "stealth", "perception", "negotiation", "luck", "survival", "resist", "engineering", "medical":
		return true
	default:
		return false
	}
}

// supportedEventCondition 判断条件类型是否受事件引擎支持。
func supportedEventCondition(conditionType string) bool {
	switch conditionType {
	case "hp_ratio", "stress_ratio", "ammo", "heat", "carry_ratio", "has_item", "flag":
		return true
	default:
		return false
	}
}

// supportedConditionOperator 判断条件比较操作符是否合法（eq/ne/lt/lte/gt/gte）。
func supportedConditionOperator(operator string) bool {
	switch operator {
	case "eq", "ne", "lt", "lte", "gt", "gte":
		return true
	default:
		return false
	}
}

// supportedEventIntent 判断方案决策意图是否在已知意图清单内。
func supportedEventIntent(intent string) bool {
	switch intent {
	case "bypass", "ambush", "engage", "force", "conceal", "secure", "search", "loot", "intel", "unlock", "rush", "withdraw", "treat", "drop", "reroute", "wait":
		return true
	default:
		return false
	}
}

// supportedEventEffect 判断效果类型是否在已知效果清单内。
func supportedEventEffect(effectType string) bool {
	switch effectType {
	case "hp", "stress", "heat", "time", "armor", "ammo", "container", "container_pool", "encounter", "skip_combat", "skip_search", "start_evacuation", "set_flag", "consume_item", "discard_loot", "evac_shortcut":
		return true
	default:
		return false
	}
}

// supportedEvacuationReason 判断事件触发的撤离原因码是否合法。
func supportedEvacuationReason(reason string) bool {
	switch reason {
	case "health", "stress", "ammo", "armor", "carry_full", "target_acquired", "event":
		return true
	default:
		return false
	}
}
