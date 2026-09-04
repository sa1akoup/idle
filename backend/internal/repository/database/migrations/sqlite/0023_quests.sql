-- 商人合同：静态任务定义与玩家进度。上交物品只消耗 raid_extract 库存。
CREATE TABLE quest_defs (
    id TEXT PRIMARY KEY,
    merchant_id TEXT NOT NULL,
    chain_index INTEGER NOT NULL DEFAULT 1,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    prerequisite_id TEXT NOT NULL DEFAULT '',
    objective_type TEXT NOT NULL,
    objective_json TEXT NOT NULL DEFAULT '{}',
    reward_json TEXT NOT NULL DEFAULT '{}',
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_quest_defs_merchant ON quest_defs (merchant_id, chain_index, sort_order);

CREATE TABLE user_quests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    quest_id TEXT NOT NULL,
    status TEXT NOT NULL,
    progress_json TEXT NOT NULL DEFAULT '{}',
    accepted_at DATETIME,
    completed_at DATETIME,
    UNIQUE (user_id, quest_id)
);
CREATE INDEX idx_user_quests_user_status ON user_quests (user_id, status);
