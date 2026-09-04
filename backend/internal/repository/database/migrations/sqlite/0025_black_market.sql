-- 黑市按用户、按 6 小时周期刷新的货架。
CREATE TABLE user_black_market_offers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    item_id TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    cycle_start DATETIME NOT NULL,
    UNIQUE (user_id, item_id)
);
CREATE INDEX idx_user_black_cycle ON user_black_market_offers (user_id, cycle_start);
