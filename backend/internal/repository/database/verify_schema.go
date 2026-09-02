// 迁移后结构校验：比对关键表/列注册清单，防止迁移静默未生效导致启动后运行期报错。
package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type schemaExpectation struct {
	table   string
	columns []string
}

// requiredSchema 是应用正确运行所依赖的最小表/列集合；
// 新增迁移涉及这些表时请同步更新清单，使缺失问题在启动阶段即可定位。
var requiredSchema = []schemaExpectation{
	{table: "characters", columns: []string{"user_id", "resource_version", "hp", "energy", "hydration", "needs_updated_at"}},
	{table: "sessions", columns: []string{"status", "terminal_reason", "recovery_policy_json", "armor_instance_id", "next_run_at", "engine_version"}},
	{table: "session_runs", columns: []string{"start_hp", "end_hp", "start_energy", "end_energy", "start_hydration", "end_hydration", "item_instance_changes"}},
	{table: "session_events", columns: []string{"run_index", "sequence", "event_type", "available_at"}},
	{table: "player_loadouts", columns: []string{"consumables", "consumable_refs", "carried_ammo", "preset_consumable_refs", "preset2_consumable_refs", "preset3_consumable_refs", "armor_instance_id"}},
	{table: "inventories", columns: []string{"item_id", "quantity", "raid_extract"}},
	{table: "armor_instances", columns: []string{"armor_id", "cur_durability", "max_durability", "repair_count", "status"}},
	{table: "item_use_defs", columns: []string{"instance_required", "usable_in_session", "usable_in_hideout"}},
	{table: "item_instances", columns: []string{"current_durability", "max_durability", "status", "location_type", "raid_extract"}},
	{table: "recovery_plans", columns: []string{"policy_json", "status"}},
	{table: "recovery_tasks", columns: []string{"resource_type", "rate_per_hour", "status"}},
	{table: "facility_level_defs", columns: []string{"hp_recovery_per_hour", "energy_recovery_per_hour", "hydration_recovery_per_hour", "repair_kit_discount_percent", "fuel_consumption_reduction_percent", "physical_skill_growth_percent", "stress_recovery_per_hour", "fuel_slot_count", "requires_power"}},
	{table: "facility_requirements", columns: []string{"requirement_type", "reference_id", "quantity"}},
	{table: "facility_jobs", columns: []string{"job_type", "complete_at", "status"}},
	{table: "facility_runtime_states", columns: []string{"enabled", "state_json"}},
	{table: "recipe_defs", columns: []string{"name", "facility_id", "required_level", "inputs_json", "output_item_id", "craft_seconds"}},
	{table: "enemy_template_defs", columns: []string{"kind", "tier", "weapon_pool", "armor_pool", "backpack_pool", "ammo_level_min", "ammo_level_max", "intellect_base", "intellect_flux", "intellect_floor", "intellect_cap", "resist_base", "resist_flux", "resist_floor", "resist_cap"}},
	{table: "user_data_migrations", columns: []string{"version", "completed_at", "processed_users", "created_instances", "stripped_refs"}},
	{table: "schema_migrations", columns: []string{"checksum"}},
}

// VerifySchema 校验当前数据库具备应用依赖的关键结构。
// 失败时汇总全部缺失项，通常意味着某个版本的迁移未生效或被改动。
func VerifySchema(db *gorm.DB, driver string) error {
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("不支持的数据库驱动: %s", driver)
	}
	migrator := db.Migrator()
	var missing []string
	for _, expectation := range requiredSchema {
		if !migrator.HasTable(expectation.table) {
			missing = append(missing, fmt.Sprintf("缺少表 %s", expectation.table))
			continue
		}
		for _, column := range expectation.columns {
			if !migrator.HasColumn(expectation.table, column) {
				missing = append(missing, fmt.Sprintf("表 %s 缺少列 %s", expectation.table, column))
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"数据库结构不完整，疑似部分迁移未生效或文件被改动（参考 docs/database-upgrade.md）：%s",
		strings.Join(missing, "；"))
}
