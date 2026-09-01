-- 节点基础遇敌概率：0-100，0 表示由引擎使用默认值（旧节点行默认回落为 60）。
ALTER TABLE node_defs ADD COLUMN encounter_chance INTEGER NOT NULL DEFAULT 0;