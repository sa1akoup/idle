-- V1.2 藏身处原版参考数据和扩展设施字段。
ALTER TABLE facility_level_defs ADD COLUMN original_cost INTEGER NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN original_currency TEXT NOT NULL DEFAULT '';
ALTER TABLE facility_level_defs ADD COLUMN original_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE facility_level_defs ADD COLUMN effects_json TEXT NOT NULL DEFAULT '';
