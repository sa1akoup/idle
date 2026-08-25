-- 持久化局内 RunPlan 和可在线推送、离线回放的结构化事件。
ALTER TABLE sessions ADD COLUMN pending_run_index INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN pending_run_result TEXT NOT NULL DEFAULT '{}';
ALTER TABLE sessions ADD COLUMN pending_run_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE session_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    session_id BIGINT NOT NULL,
    run_index INTEGER NOT NULL,
    sequence INTEGER NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    offset_sec BIGINT NOT NULL,
    available_at TIMESTAMPTZ NOT NULL,
    node_id VARCHAR(128) NOT NULL DEFAULT '',
    subject_id VARCHAR(128) NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_session_events_session_run_sequence
ON session_events (session_id, run_index, sequence);
CREATE INDEX idx_session_events_timeline
ON session_events (user_id, session_id, available_at, id);
