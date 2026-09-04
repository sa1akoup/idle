-- 安全箱定义与配装字段。口袋格数只影响探索搜刮与失能保物，不在局外存货。
CREATE TABLE secure_container_defs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    inner_slots INTEGER NOT NULL DEFAULT 1,
    price INTEGER NOT NULL DEFAULT 0,
    weight INTEGER NOT NULL DEFAULT 1,
    slots INTEGER NOT NULL DEFAULT 1,
    merchant_category TEXT NOT NULL DEFAULT '',
    rep_requirement INTEGER NOT NULL DEFAULT 0,
    unlock_quest_id TEXT NOT NULL DEFAULT ''
);
ALTER TABLE player_loadouts ADD COLUMN secure_container_id TEXT NOT NULL DEFAULT '';
