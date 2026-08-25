-- 可重放经济操作。
CREATE TABLE economic_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    operation_key TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    result_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME,
    UNIQUE (user_id, operation_key)
);

CREATE INDEX idx_economic_operations_user_id ON economic_operations (user_id);
