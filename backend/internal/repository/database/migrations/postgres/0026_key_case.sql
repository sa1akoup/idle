-- 钥匙包装备与配装字段。钥匙实例仍走 item_instances.location_type。
CREATE TABLE key_case_defs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_slots INTEGER NOT NULL DEFAULT 3,
    price INTEGER NOT NULL DEFAULT 0,
    weight INTEGER NOT NULL DEFAULT 1,
    slots INTEGER NOT NULL DEFAULT 1,
    merchant_category TEXT NOT NULL DEFAULT 'clothing',
    rep_requirement INTEGER NOT NULL DEFAULT 0
);
ALTER TABLE player_loadouts ADD COLUMN key_case_id TEXT NOT NULL DEFAULT '';
