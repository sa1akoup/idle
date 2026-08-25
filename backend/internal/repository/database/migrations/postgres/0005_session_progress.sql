-- 持久化行动模拟进度，支持进程重启后从下一局继续。
ALTER TABLE sessions ADD COLUMN elapsed_min INTEGER NOT NULL DEFAULT 0;
