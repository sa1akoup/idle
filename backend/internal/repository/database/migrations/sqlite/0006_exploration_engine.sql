-- 固定探索引擎版本、连续状态、现实时间游标与数据库 lease。
ALTER TABLE sessions ADD COLUMN offline_limit_sec INTEGER NOT NULL DEFAULT 86400;
ALTER TABLE sessions ADD COLUMN elapsed_sec INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN next_run_at DATETIME;
ALTER TABLE sessions ADD COLUMN current_run_started_at DATETIME;
ALTER TABLE sessions ADD COLUMN last_processed_at DATETIME;
ALTER TABLE sessions ADD COLUMN lease_owner TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN lease_until DATETIME;
ALTER TABLE sessions ADD COLUMN heartbeat_at DATETIME;
ALTER TABLE sessions ADD COLUMN engine_version TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN scenario_snapshot TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN scenario_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN state_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE sessions ADD COLUMN initial_state_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE session_runs ADD COLUMN duration_sec INTEGER NOT NULL DEFAULT 0;
ALTER TABLE session_runs ADD COLUMN consumed_items TEXT NOT NULL DEFAULT '[]';
ALTER TABLE session_runs ADD COLUMN stored_loot TEXT NOT NULL DEFAULT '[]';
ALTER TABLE session_runs ADD COLUMN overflow_loot TEXT NOT NULL DEFAULT '[]';
ALTER TABLE session_runs ADD COLUMN input_state TEXT NOT NULL DEFAULT '{}';
ALTER TABLE session_runs ADD COLUMN next_state TEXT NOT NULL DEFAULT '{}';

-- 旧 Session 没有可重放快照，直接终止，避免悬挂 active 状态阻塞新行动。
UPDATE sessions
SET status = 'failed', end_time = COALESCE(end_time, CURRENT_TIMESTAMP)
WHERE status IN ('running', 'waiting_injury')
  AND next_run_at IS NULL;

-- 唯一索引创建前清理可能存在的重复 active 记录，保留最早创建的一条。
UPDATE sessions
SET status = 'failed', end_time = COALESCE(end_time, CURRENT_TIMESTAMP)
WHERE id IN (
    SELECT duplicate.id
    FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY id) AS row_number
        FROM sessions
        WHERE status IN ('running', 'waiting_injury')
    ) AS duplicate
    WHERE duplicate.row_number > 1
);

CREATE INDEX idx_sessions_scheduler ON sessions (status, next_run_at, lease_until);
CREATE UNIQUE INDEX idx_sessions_user_active ON sessions (user_id) WHERE status IN ('running', 'waiting_injury');
CREATE UNIQUE INDEX idx_session_runs_session_run ON session_runs (session_id, run_index);
