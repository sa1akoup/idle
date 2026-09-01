-- 清理遗留死表：enemy_defs 自 V1.2（0017 enemy_template_defs）起被模板+生成器替代。
-- 迁移尚未发布时先保留一份同库备份；观察期结束后再以新 migration 清理备份表。
CREATE TABLE enemy_defs_backup_0021 AS SELECT * FROM enemy_defs;
DROP TABLE enemy_defs;
