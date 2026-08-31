-- V1.1 收尾：清退已退役状态的历史行动行（如 waiting_injury），解除对活跃名额的部分唯一索引占用。
UPDATE sessions SET status = 'failed', terminal_reason = 'legacy_status',
    end_time = COALESCE(end_time, CURRENT_TIMESTAMP)
WHERE status NOT IN ('running', 'success', 'incapacitated', 'failed');
