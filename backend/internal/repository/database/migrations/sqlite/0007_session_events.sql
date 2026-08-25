-- 持久化局内 RunPlan 和可在线推送、离线回放的结构化事件。
ALTER TABLE sessions ADD COLUMN pending_run_index INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN pending_run_result TEXT NOT NULL DEFAULT '{}';
ALTER TABLE sessions ADD COLUMN pending_run_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE session_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    session_id INTEGER NOT NULL,
    run_index INTEGER NOT NULL,
    sequence INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    offset_sec INTEGER NOT NULL,
    available_at DATETIME NOT NULL,
    node_id TEXT NOT NULL DEFAULT '',
    subject_id TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_session_events_session_run_sequence
ON session_events (session_id, run_index, sequence);
CREATE INDEX idx_session_events_timeline
ON session_events (user_id, session_id, available_at, id);
