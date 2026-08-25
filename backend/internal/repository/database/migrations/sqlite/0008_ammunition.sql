-- 引入口径、分级弹药、护甲等级，以及 Session 与恢复预设的实际携弹状态。
CREATE TABLE ammo_defs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    caliber_id TEXT NOT NULL,
    level INTEGER NOT NULL,
    flesh_damage_multiplier REAL NOT NULL,
    armor_damage_multiplier REAL NOT NULL,
    price INTEGER NOT NULL,
    rounds_per_slot INTEGER NOT NULL,
    merchant_category TEXT NOT NULL,
    rep_requirement INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_ammo_caliber_level ON ammo_defs (caliber_id, level);

ALTER TABLE weapon_defs ADD COLUMN caliber_id TEXT NOT NULL DEFAULT '';
ALTER TABLE armor_defs ADD COLUMN protection_level INTEGER NOT NULL DEFAULT 0;
ALTER TABLE enemy_defs ADD COLUMN ammo_id TEXT NOT NULL DEFAULT '';
ALTER TABLE enemy_defs ADD COLUMN ammo_rounds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN ammo_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN ammo_rounds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_loadouts ADD COLUMN preset_ammo_id TEXT NOT NULL DEFAULT '';
ALTER TABLE player_loadouts ADD COLUMN preset_ammo_rounds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_loadouts ADD COLUMN preset2_ammo_id TEXT NOT NULL DEFAULT '';
ALTER TABLE player_loadouts ADD COLUMN preset2_ammo_rounds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_loadouts ADD COLUMN preset3_ammo_id TEXT NOT NULL DEFAULT '';
ALTER TABLE player_loadouts ADD COLUMN preset3_ammo_rounds INTEGER NOT NULL DEFAULT 0;

CREATE INDEX idx_weapon_defs_caliber_id ON weapon_defs (caliber_id);
