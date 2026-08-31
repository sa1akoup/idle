# 数据库升级与旧存档适配指南

本文说明后端如何安全地演进数据库结构，以及新增迁移、破坏性变更和数据回填的标准流程。

## 升级链路总览

新版本启动时自动完成三步（顺序固定，均在调度器启动之前）：

1. **Schema 迁移** `database.Migrate`：按版本号顺序执行内嵌 SQL（sqlite/postgres 各一套），
   整体单事务、失败即回滚并在启动时报错，绝不带着半升级结构运行。
2. **完整性校验** `database.VerifySchema`：比对关键表/列注册清单
   （`internal/repository/database/verify_schema.go`），缺列时拒绝启动并指出疑似未生效的迁移。
3. **存量数据适配** `service.RunUserDataUpgrades`：对所有用户幂等执行——
   - 聚合库存中的"实例必需品"换算为耐久实例行；
   - 摘除装备配置/库存/护甲实例中已不在目录的悬空引用（每条摘除均打 `[数据适配]` WARN 日志）。

   目录种子不完整（任一目录表为空）时适配会**拒绝执行**并提示先 `go run . seed`——
   空目录下悬空判定不可信，宁可拒绝服务也不能误删玩家资产。

SQLite 仅在**确有待应用迁移**时创建升级前备份 `<库名>.pre-upgrade.bak`；若因权限/磁盘等原因无法生成备份，迁移会直接中止等待人工处理（内存库豁免）。PostgreSQL 请在部署前手动 `pg_dump`。

## 保护机制

| 机制 | 说明 |
| --- | --- |
| 迁移校验和 | 每次应用迁移时记录文件 SHA256。历史行为 NULL 时以当前内容静默补记为基线；此后修改任何已发布迁移都会导致启动报错并指出版本号 |
| 升级前备份 | 仅当确有待应用迁移时生成 `.pre-upgrade.bak`；普通启动绝不触碰已有备份；备份无法生成则中止迁移 |
| 目录种子守卫 | 存量数据适配执行前校验全部目录表非空，否则拒绝运行（防止空目录把正常资产误判为悬空删除） |
| SQLite 语法静态检查 | 单测扫描全部 sqlite 迁移，禁止 `ADD COLUMN ... DEFAULT CURRENT_*` 与表达式默认值（该语法在旧形状库上会直接失败，曾造成启动被卡死） |
| 双方言版本一致性 | 单测比对 sqlite/postgres 版本集合，防止单侧漏写 |
| 引擎快照版本 | 行动中的 Session 冻结快照与引擎版本；跨引擎版本的重启会走 failSession 结算且**解码失败也能终态化**（不会卡死 running 死循环重派） |

## 新增一个迁移的流程

1. 同时创建 `migrations/sqlite/NNNN_name.sql` 与 `migrations/postgres/NNNN_name.sql`（编号连续递增，正文注释逐字一致）。
2. 首行写一句中文目的说明；涉及存量行改写的语句必须配一行 why。
3. 若新增了关键表/列，同步更新 `verify_schema.go` 的注册清单（否则升级测试会提示清单缺失）。
4. 在 `userdata_upgrade.go` 需要的地方补充对应的数据适配步骤（保持幂等：重复执行无副作用）。
5. 跑 `go test ./internal/repository/database/ ./internal/service/ -count=1`，
   视情况在演练测试中补充"旧形状数据"用例。

## 破坏性变更配方

### 加列
- 默认值只能是**字面量**（SQLite 对 ADD COLUMN 的硬限制）：`ADD COLUMN x REAL NOT NULL DEFAULT 0`。
- 需要"当前时间"语义时：建列为常量默认值 + 同文件内 `UPDATE ... SET x = CURRENT_TIMESTAMP` 回填，
  并确保代码写入路径总是显式赋值（参考 0012 的 needs_updated_at）。

### 删列 / 改名 / 改约束（表重建模板）
SQLite 不支持这些操作，需要整表重建：

```sql
CREATE TABLE foo_new (... 新结构 ...);
INSERT INTO foo_new (colA, colB) SELECT col_a, col_b FROM foo;  -- 仅挑选仍需要的列
DROP TABLE foo;
ALTER TABLE foo_new RENAME TO foo;
-- 重建原索引
```

注意：两套方言都要写完整脚本；Go 模型同 PR 内同步增删字段；模型移除字段即可让旧列成为孤儿列（本项目惯例是不追删）。

### 数据回填（改写存量行）
优先放在 migration SQL 里做集合运算（参考 0006 清退旧状态、0012 回填时间戳）。
逻辑超出 SQL 表达能力（需解析 JSON、依赖 Go 计算规则）时，放进 `RunUserDataUpgrades`
的一个幂等步骤中，由每次启动对全体用户执行。

### 变更物品/装备 ID
1. seed 中保留旧行（装备类 Def 本就不删）或显式编写迁移映射；
2. 依赖悬空引用自愈兜底（引用会被摘除并记日志，玩家可玩性不受损）；
3. ItemUseDef 是唯一会被 seed 真删的目录——删效果前先确认没有玩法依赖，必要时保留空效果行而不是删行。

## 已知局限

静态检查与演练测试无法复刻个别 SQLite 引擎层面的文件级怪异行为（如本次旧 idle.db 对
非常量默认值的判定与新建库不一致）。这类风险的最终防线是上线前用真实库副本执行一次
`go run . migrate` 冒烟验证。

## 发布前 Checklist

- [ ] 两方言迁移成对、版本连续、首行有说明
- [ ] `go vet ./... && go test ./... -count=1` 全绿（含语法静态检查与演练测试）
- [ ] 若动了关键表/列：更新 verify_schema 注册清单
- [ ] 若影响存量玩家数据：补 RunUserDataUpgrades 幂等步骤 + 演练测试用例
- [ ] 用上一真实版本的数据库副本执行 `go run . migrate` 冒烟
- [ ] PostgreSQL 环境部署文档提醒 pg_dump 备份
