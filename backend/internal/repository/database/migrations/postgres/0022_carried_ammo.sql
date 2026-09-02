-- 随身携带弹药：角色页随身补给下方新增 4 格弹药槽（JSON 数组，每格 {ammoId, rounds}）。
-- 可空列与 consumables 同款：空槽在读取层兜底为 []，避免写路径因零值触发 NOT NULL。
ALTER TABLE player_loadouts ADD COLUMN carried_ammo TEXT;