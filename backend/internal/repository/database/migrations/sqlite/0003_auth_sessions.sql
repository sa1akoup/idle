-- 登录会话。
CREATE TABLE auth_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions (user_id);
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions (expires_at);
