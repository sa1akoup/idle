-- 为每个角色增加用户级资源锁版本，事务通过更新该行串行化资源操作。
ALTER TABLE characters ADD COLUMN resource_version INTEGER NOT NULL DEFAULT 0;
