-- 战利品稀有度：对齐原版 Common / Rare / Superrare，用于容器档位过滤与掉落权重。
ALTER TABLE loot_item_defs ADD COLUMN rarity TEXT NOT NULL DEFAULT 'common';
