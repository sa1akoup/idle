-- V1.2 存量数据适配版本记录：避免服务每次启动重复扫描和清扫全部用户。
CREATE TABLE user_data_migrations (
    version INTEGER PRIMARY KEY,
    completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_users INTEGER NOT NULL DEFAULT 0,
    created_instances INTEGER NOT NULL DEFAULT 0,
    stripped_refs INTEGER NOT NULL DEFAULT 0
);
