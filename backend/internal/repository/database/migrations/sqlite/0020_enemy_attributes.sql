-- 敌人模板补充智力/抗性属性：发现率侦察、压力抗性与武器操控按真实属性计算（替代固定 40/55）。
ALTER TABLE enemy_template_defs ADD COLUMN intellect_base INTEGER NOT NULL DEFAULT 40;
ALTER TABLE enemy_template_defs ADD COLUMN intellect_flux INTEGER NOT NULL DEFAULT 8;
ALTER TABLE enemy_template_defs ADD COLUMN intellect_floor INTEGER NOT NULL DEFAULT 20;
ALTER TABLE enemy_template_defs ADD COLUMN intellect_cap INTEGER NOT NULL DEFAULT 100;
ALTER TABLE enemy_template_defs ADD COLUMN resist_base INTEGER NOT NULL DEFAULT 40;
ALTER TABLE enemy_template_defs ADD COLUMN resist_flux INTEGER NOT NULL DEFAULT 8;
ALTER TABLE enemy_template_defs ADD COLUMN resist_floor INTEGER NOT NULL DEFAULT 20;
ALTER TABLE enemy_template_defs ADD COLUMN resist_cap INTEGER NOT NULL DEFAULT 100;