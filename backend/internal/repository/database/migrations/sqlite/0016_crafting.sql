-- V1.1 工作台制造：配方目录。
CREATE TABLE recipe_defs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    facility_id TEXT NOT NULL,
    required_level INTEGER NOT NULL DEFAULT 1,
    inputs_json TEXT NOT NULL DEFAULT '[]',
    output_item_id TEXT NOT NULL,
    output_quantity INTEGER NOT NULL DEFAULT 1,
    craft_seconds INTEGER NOT NULL DEFAULT 60,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_recipe_defs_facility ON recipe_defs (facility_id, required_level, sort_order);
