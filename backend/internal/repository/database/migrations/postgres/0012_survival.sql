-- V1.1 持久化生存资源、恢复计划、物品效果与通用耐久实例。
ALTER TABLE characters ADD COLUMN hp DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE characters ADD COLUMN energy DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE characters ADD COLUMN hydration DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE characters ADD COLUMN needs_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE sessions ADD COLUMN terminal_reason TEXT;
ALTER TABLE sessions ADD COLUMN recovery_policy_json TEXT;
ALTER TABLE sessions ADD COLUMN armor_instance_id INTEGER NOT NULL DEFAULT 0;

ALTER TABLE session_runs ADD COLUMN start_hp DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE session_runs ADD COLUMN end_hp DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE session_runs ADD COLUMN start_energy DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE session_runs ADD COLUMN end_energy DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE session_runs ADD COLUMN start_hydration DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE session_runs ADD COLUMN end_hydration DOUBLE PRECISION NOT NULL DEFAULT 100;
ALTER TABLE session_runs ADD COLUMN item_instance_changes TEXT;

ALTER TABLE player_loadouts ADD COLUMN consumable_refs TEXT;
ALTER TABLE player_loadouts ADD COLUMN preset_consumable_refs TEXT;
ALTER TABLE player_loadouts ADD COLUMN preset2_consumable_refs TEXT;
ALTER TABLE player_loadouts ADD COLUMN preset3_consumable_refs TEXT;
ALTER TABLE player_loadouts ADD COLUMN armor_instance_id INTEGER NOT NULL DEFAULT 0;

ALTER TABLE facility_level_defs ADD COLUMN hp_recovery_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN energy_recovery_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN hydration_recovery_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN repair_kit_discount_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN fuel_consumption_reduction_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN physical_skill_growth_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN stress_recovery_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN fuel_slot_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN requires_power BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE facility_requirements (
    id BIGSERIAL PRIMARY KEY,
    facility_id TEXT NOT NULL,
    level INTEGER NOT NULL,
    requirement_type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    required_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE item_use_defs (
    item_id TEXT PRIMARY KEY,
    hp_recovery DOUBLE PRECISION NOT NULL DEFAULT 0,
    energy_recovery DOUBLE PRECISION NOT NULL DEFAULT 0,
    hydration_recovery DOUBLE PRECISION NOT NULL DEFAULT 0,
    repair_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    fuel_seconds BIGINT NOT NULL DEFAULT 0,
    max_durability DOUBLE PRECISION NOT NULL DEFAULT 0,
    use_durability DOUBLE PRECISION NOT NULL DEFAULT 0,
    use_priority INTEGER NOT NULL DEFAULT 0,
    instance_required BOOLEAN NOT NULL DEFAULT FALSE,
    usable_in_session BOOLEAN NOT NULL DEFAULT FALSE,
    usable_in_hideout BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE item_instances (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    item_id TEXT NOT NULL,
    current_durability DOUBLE PRECISION NOT NULL,
    max_durability DOUBLE PRECISION NOT NULL,
    status TEXT NOT NULL DEFAULT 'normal',
    location_type TEXT NOT NULL DEFAULT 'inventory',
    location_ref TEXT,
    slot_index INTEGER NOT NULL DEFAULT 0,
    raid_extract BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recovery_plans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    source_session_id BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    policy_json TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recovery_tasks (
    id BIGSERIAL PRIMARY KEY,
    recovery_plan_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    resource_type TEXT NOT NULL,
    start_value DOUBLE PRECISION NOT NULL,
    current_value DOUBLE PRECISION NOT NULL,
    target_value DOUBLE PRECISION NOT NULL,
    rate_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0,
    primary_method TEXT NOT NULL,
    actual_method TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    complete_at TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'running',
    detail_json TEXT
);

CREATE TABLE facility_runtime_states (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    facility_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    state_json TEXT
);

CREATE UNIQUE INDEX idx_item_instances_user_id ON item_instances (user_id, id);
CREATE INDEX idx_item_instances_location ON item_instances (user_id, location_type, location_ref, status);
CREATE INDEX idx_item_use_defs_session ON item_use_defs (usable_in_session, instance_required);
CREATE UNIQUE INDEX idx_recovery_plans_session ON recovery_plans (user_id, source_session_id);
CREATE INDEX idx_recovery_tasks_user_status ON recovery_tasks (user_id, status);
CREATE UNIQUE INDEX idx_facility_runtime_user ON facility_runtime_states (user_id, facility_id);
CREATE INDEX idx_facility_requirements_level ON facility_requirements (facility_id, level, sort_order, id);
