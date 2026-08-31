# 项目架构与实现逐文件审查报告

> 审查日期：2026-08-31（Asia/Shanghai）  
> 审查对象：`D:\idle` 当前工作树  
> 审查规模：176 个源码、迁移、配置、脚本与文档文件（99 Go、34 SQL、13 Vue、10 TypeScript、7 Markdown，其余为配置与静态资源）  
> 工作树状态：审查时存在大量已修改和未跟踪文件，本报告基于当前文件内容，不代表任一历史 commit；本次仅新增本报告，未修改业务源码。

## 1. 结论摘要

项目的核心架构方向总体合理：纯 `engine` 与数据库解耦、Session 快照和哈希、Pending Run 幂等结算、数据库 lease、资源锁、事务结算、REST/SSE 补拉，这些设计都对应真实的一致性需求，不属于过度设计。

本次确认 2 个高风险缺陷、11 组中风险问题和若干清理项。最优先处理的是恢复计划永久卡住、活动 SQLite 文件直接复制备份、`EnemyTemplateDef` JSON tag 错误、前端请求竞态及当前补给可选择未持有物品。

| 类别 | 结论 |
| --- | --- |
| 确认缺陷 | 10 组，含恢复流程卡死、SQLite 备份一致性、JSON tag、前端竞态与库存过滤 |
| 过度设计 | 主要集中在自建 migration 框架、启动时全量用户数据升级、全局 scheduler 生命周期和事件配置双重校验 |
| 防御性编程 | 主要表现为静默降级、默认用户兼容包装、variadic 兼容参数、通过 ID 推断缺失语义、弱化类型约束 |
| 重复造轮子 | migration/校验/备份框架最明显；图算法和 SSE 当前实现有合理性，不建议为换库而换库 |
| 僵尸代码 | 10 个 Go 函数、1 个 Go 模型、2 个前端类型、1 个无效状态、1 个无效事件绑定、约 30 组旧 CSS selector |
| 不必要兼容 | 默认用户 service 包装、重复维修 API、旧 `/api/nodes`、旧数据每次启动清扫、`SessionEventType | string` |

## 2. 高优先级问题

### P1-01 恢复计划可能永久处于 running，阻止用户再次探索

**位置**：`backend/internal/service/recovery.go:192-231, 341-403, 542-564, 588-611`

`applyRecoveryPolicyTx` 在确认恢复方式实际生效前就把 `actualMethod` 赋为当前尝试的方法。常见方法序列为 `inventory -> hideout -> merchant`。当藏身处具备恢复速率，但商人恢复因现金不足或无可购商品而未生效时，最终 `ActualMethod` 会被覆盖为 `merchant`。

后续结算只给 `ActualMethod == hideout` 的任务计算速率。上述任务会得到 `rate=0`、`status=running`、`complete_at=NULL`，`ensureRecoveryReadyForStartTx` 又会持续阻止新 Session，用户可能永久卡住。

**建议**：只有 `used == true` 时记录即时恢复方式；未达到目标且藏身处速率可用时，明确把最终方式固定为 `hideout`。增加“库存不足、商人购买失败、藏身处可恢复”的回归测试，以及“所有方式均不可用”时的显式失败状态。

### P1-02 SQLite 升级备份直接复制活动数据库文件，备份可能不一致

**位置**：`backend/internal/repository/database/database.go:211-252`

数据库连接已打开后，代码通过 `os.Open + io.Copy` 复制主 `.db` 文件。SQLite 使用 WAL 时，最新提交可能仍在 `-wal` 文件中；直接复制主文件可能遗漏数据，也可能在写入期间得到不一致快照。现有测试只覆盖“是否生成/覆盖备份”和“备份失败是否中止”，未验证恢复后的数据一致性。

**建议**：优先使用 SQLite Online Backup API 或 `VACUUM INTO`；至少在受控锁内执行 checkpoint 并验证备份可打开、可通过 `PRAGMA integrity_check`。升级前备份属于恢复底线，应先修复，再评估迁移框架替换。

## 3. 中优先级问题

### P2-01 `EnemyTemplateDef` 的 JSON tag 无效

**位置**：`backend/internal/models/models.go:376`

```go
HPFloor, HPCap int `json:"hpFloor,hpCap"`
```

两个字段共享 `hpFloor` 名称，`hpCap` 被解析成未知 tag option。`go vet` 与 `gopls check` 均报告错误，`HPCap` 无法按预期序列化。

**建议**：拆成两个字段声明并分别设置 `json:"hpFloor"`、`json:"hpCap"`，补充 JSON round-trip 测试。

### P2-02 当前装备的补给选择未按库存过滤

**位置**：`frontend/src/composables/useCharacterLoadout.ts:150-157`

装备栏通过 `currentOptions` 限制为已持有物品；补给栏无论 `current` 还是 `preset` 都返回全部 `usableInSession` 消耗品。用户可以把未持有补给放入当前装备，自动保存随后被后端拒绝，本地 UI 仍保留失败后的选择。

**建议**：`current` 上下文按库存数量过滤补给；保存失败时恢复到服务端 loadout，避免前端状态与持久化状态分叉。

### P2-03 日志页会重复请求，并可能被旧响应覆盖

**位置**：`frontend/src/views/LogsView.vue:21-26, 71-86`

点击会直接调用 `loadSession`，该函数修改 `selectedId`；`watchEffect` 观察到 `selectedId` 改变后，在旧 `detail` 尚未更新时会再次调用 `loadSession`。快速切换还缺少 request generation 或取消机制，较慢的旧请求可以覆盖当前选择。请求失败时也未清空旧 `detail`，页面可能显示错误 Session 的内容。

**建议**：改为 `watch(selectedId, loadSession, { immediate: true })`，点击只修改 ID；使用递增 request token 或 `AbortController` 丢弃旧响应；失败时清空与当前 ID 不匹配的数据。

### P2-04 商人目录切换存在同类竞态

**位置**：`frontend/src/views/MerchantView.vue:29-46, 88-99`

快速点击多个商人时，旧目录请求可能晚于新请求返回并覆盖 `catalog`。`catalogLoading` 也可能由旧请求提前置为 false。

**建议**：为 `loadCatalog` 增加 request token/取消控制，提交结果前再次确认 `id === selectedId.value`。

### P2-05 启动时用户数据升级职责过重且带破坏性

**位置**：`backend/internal/service/userdata_upgrade.go:18-375`

每次启动都会扫描全部用户，回填耐久实例并清理悬空装备、库存和实例。`requireSeededCatalogForUpgrade` 只验证目录表非空，无法证明单个旧 ID 已获得迁移映射；目录 ID 删除或重命名时，玩家资产可能被直接删除。

该逻辑同时承担 schema 迁移之后的数据适配和持续数据清扫，生命周期、审计和回滚边界不清晰。

**建议**：改为有版本号的一次性 data migration，记录执行版本和受影响行；ID 变化使用显式映射/补偿；持续一致性检查改为只报告或单独运维命令。

### P2-06 全局 scheduler 生命周期不可重启、不可等待

**位置**：`backend/internal/service/scheduler.go:23-129`

包级 `sync.Map`、channel 和两个 `sync.Once` 隐式管理生命周期。`StartSessionScheduler` 会重建 stop channel，但 stop 的 `sync.Once` 不会重置，同进程第二次启动后无法正常停止；worker queue 永不关闭，`StopSessionScheduler` 不等待 worker 完成。

**建议**：封装为 `Scheduler` 实例，接收 `context.Context`，提供 `Start/Stop/Wait`，由 `main` 明确持有。先补二次启动停止和优雅关闭测试，再改结构。

### P2-07 多处数据库错误被伪装成业务状态

**位置**：

- `backend/internal/service/crafting.go:225-230`：查询失败统一返回“不需要实例”。
- `backend/internal/service/recovery.go:55-60`：持久化 JSON 损坏时静默换成默认策略。
- `backend/internal/service/recovery.go:280-283`：缺少设施等级定义时直接跳过。
- `backend/internal/service/generator.go:259-261`：太阳能减耗查询失败时静默使用默认值。
- `backend/internal/service/session_resource_guard.go:104-120`：CSV 解析失败返回空列表，可能放过资源锁保护。

这些位置位于稳定数据边界，静默降级会把数据损坏或查询故障伪装成合法业务状态。

**建议**：返回带上下文错误；只有明确的 `record not found` 且业务允许缺省时才降级。恢复策略损坏应记录 plan/session ID，并阻止继续结算。

### P2-08 自建 migration 框架承担了成熟库的大量能力

**位置**：`backend/internal/repository/database/database.go`、`verify_schema.go`、`migrations_test.go`、34 个双方言 SQL 文件

当前自行实现版本解析、checksum、事务执行、schema 验证、SQLite 备份和双目录一致性测试，维护成本已明显高于项目早期阶段。双方言 migration 本身属于真实需求，问题集中在执行框架和备份能力。

**建议**：短期保留现状并先修备份；中期评估 `goose`、`golang-migrate` 或 Atlas。替换时保留现有版本号和 checksum 审计，不做一次性重写。

### P2-09 事件配置存在双重校验和隐式语义推断

**位置**：`backend/internal/service/event_config_validation.go`、`backend/internal/engine/snapshot.go`、`snapshot_validation.go`、`style.go:100-192`

service 和 engine 分别维护 phase、effect、condition、style 等支持集合。同一约束存在两套实现，新增能力容易只更新一侧。`normalizeEventOption` 还会根据 option ID/effect ref 推断 `Intent/RiskTier/ValueTier`，配置字段虽然已存在，缺失时仍被隐式补齐；新 option ID 会静默落入默认 `search`。

**建议**：共享枚举与单一纯验证入口；seed 显式填写语义字段；推断逻辑只用于一次性旧数据迁移，运行时配置缺字段直接报错。

### P2-10 前端启动全量加载，任一非关键接口失败都会阻断整个应用

**位置**：`frontend/src/composables/useAppWorkspace.ts:131-183`

登录后一次并发请求约 21 个资源，再为每张地图请求 graph。任何一个商人、藏身处、制造或目录接口失败，`Promise.all` 会让整个工作区进入 fatal state。所有视图和 Element Plus 也被入口同步加载，最终单 JS chunk 为 1.13 MB。

**建议**：首屏只加载身份、玩家、Session 和当前视图必需数据；其他视图按进入时加载并独立展示错误。使用动态 import 拆分视图，Element Plus 改按需引入或自动导入。

### P2-11 `start.ps1 -Stop` 可能强杀非本项目进程

**位置**：`start.ps1:16-19, 23-59`

脚本硬编码 `D:/idle`，停止时按 8081/5173 端口反查并递归 `-Force` 结束进程。只要其他程序占用相同端口，也会被当作目标杀掉；PID 文件无法证明按端口找到的进程由本脚本启动。

**建议**：路径改用 `$PSScriptRoot`；优先只停止记录 PID 及其子进程；端口命中但不在该进程树时仅提示，不自动强杀。启动失败时清理已启动的另一端进程。

## 4. 结构性问题与清理项

### 4.1 默认用户兼容包装层

以下函数在生产调用链中无调用或仅被测试调用，统一转发到 `ForUser` 版本并注入 `models.DefaultUserID`：

- `service/carry.go`：`GetCarryCapacity`
- `service/inventory.go`：`PurchaseItem`、`GetStorageCapacity`
- `service/loadout.go`：`GetPlayerLoadout`、`SavePlayerLoadout`、`validateOwnedLoadout`、`ReplaceLostLoadout`
- `service/merchant.go`：`GetMerchants`、`GetMerchantByID`、`applyMerchantPrice`、`PurchaseFromMerchant`、`SellItem`
- `service/merchant_reputation.go`：`AwardReputation`

`inventoryUsage(db, userIDs ...uint)` 和 `handler.NewHandler(db, secureCookie ...bool)` 也使用 variadic 参数维持旧调用形式。认证改造已经完成，这些兼容层继续存在会鼓励新代码绕过 user scope。

**建议**：先把测试改为显式 user ID，再删除包装层、`DefaultUserID` 依赖和无意义 variadic 参数。数据库迁移历史中的默认用户数据仍可保留。

### 4.2 重复 API 和旧视图接口

- `/api/hideout/repair`：当前前端使用，payload 为 `armorInstanceId`。
- `/api/armor/repair`：README 和旧方案文档仍记录，payload 为 `id`。
- 两条路由最终均调用 `QueueArmorRepairForUser`。
- `/api/nodes` 与 `/api/maps/:id/graph` 重复构造节点/容器视图；前端只使用 graph。
- `useAppWorkspace` 仍维护并返回 `nodes`，但 `App.vue` 没有消费。

**建议**：选定 `/api/hideout/repair` 和 graph API 为唯一入口；若已有外部客户端，设置明确弃用版本和日志，否则直接删除旧路由并同步 README。

### 4.3 重复领域公式和规则

- `models.EffectiveSkill` 与 `engine.effectiveSkill`
- `service.characterMaxHP` 与 `engine.calcMaxHP`
- `service.calcPlayerStressThreshold` 与 engine 同名语义公式
- 前端 `liveCapacity` 与后端携行公式
- 前端 `sellPriceFor` 与后端商人价格公式
- 前端商人弹药可售规则与 `scenario_snapshot.go` 重复
- `HideoutView` 根据 facility ID 硬编码升级效果文本，未直接消费后端效果数据

**建议**：Go 纯公式统一放入 engine/domain 包；前端展示所需派生值由 API 返回，前端只做格式化。避免用注释承诺“与后端保持一致”。

### 4.4 多表商品目录适配层重复且易产生 N+1

`findCatalogItem` 按 9 类表顺序查询，`MerchantCatalog` 又分别读取多类目录；crafting、loadout、recovery 等模块重复走相同分支。当前分表模型已经把简单商品引用扩散成大量适配代码。

**建议**：不立即重构 schema。先建立统一只读 catalog repository，支持批量按 ID 查询与请求内缓存；测量 SQL 数量后再决定是否合并表。

### 4.5 预设模型字段爆炸

`PlayerLoadout` 与 `SaveLoadoutRequest` 为三套预设复制约 30 个字段，`presetOf`、`presetFromLoadout`、`presetOfReq` 等函数持续复制映射逻辑。新增第 4 套预设需要修改模型、迁移、服务和前端类型。

**建议**：当前三套固定需求可暂时保留；下一次需要扩展预设数量时再迁移到 `loadout_presets` 子表或 JSON 数组，避免现在进行无收益的大重构。

## 5. 僵尸代码清单

### 5.1 Go 仅声明、无引用

| 文件 | 符号 |
| --- | --- |
| `engine/types_runtime.go:63` | `CloneItemStacks` |
| `service/generator.go:42` | `GetGeneratorViewForUser` |
| `service/generator.go:403` | `sortUint` |
| `service/hideout.go:641` | `hasInventoryQuantity` |
| `service/recovery.go:688` | `sortRecoveryDefinitions` |
| `service/session_engine_helpers.go:12` | `appendEngineReport` |
| `service/session_engine_helpers.go:16` | `classifyResourceUnavailable` |
| `service/session_engine_run_settlement.go:349` | `lootQuantityEngine` |
| `service/session_events.go:208` | `isRecordNotFound` |
| `service/session.go:317` | `splitIDs` |

另有 `inventory.go:346` 的删除函数遗留注释，`engine/snapshot.go:76-80` 读取 `container` 后只执行 `_ = container`。

### 5.2 僵尸模型与前端类型

- `models.EnemyDef` 没有运行时引用。历史 `enemy_defs` 表可因 migration 历史保留，Go 模型没有同样的保留必要。
- `frontend/src/types.ts` 的 `CarryCapacity`、`LootItem` 只有声明，无消费方。
- `SessionEvent.eventType: SessionEventType | string` 实际等价于 `string`，完全抵消 union 的类型检查；这是为未知未来事件保留的兼容写法，当前收益低于类型损失。

### 5.3 前端无效状态、监听与 CSS

- `useAppWorkspace.nodes` 由 graph 派生、返回后无人使用。
- `App.vue` 给 `LogsView` 绑定 `@refresh`，组件未声明也未触发该事件。
- `style.css` 中未在当前 Vue/TS 模板出现的本地 selector 包括：`supplies-field`、`duration-field`、`map-node`、`loadout-columns`、`loadout-column`、`loadout-note`、`current-spacer`、`consumable-group`、`catalog-toolbar`、`facility-strip`、`hideout-layout`、`facility-board`、`hideout-queue`、`facility-grid`、多组 `facility-card__*`、`queue-empty`、`generator-panel`、`live-map__image`、`live-map__node`、`live-notice`。

这些样式跨越多轮 UI 改版，约占 170 行。删除前应通过浏览器截图回归确认动态 class 不依赖它们。

## 6. 其他确认问题

### 6.1 地图路线高亮丢失顺序语义

`MapGraphCanvas.vue:26-28, 68-75` 把有序 `routeNodeIds` 转成 Set，只要一条边的两个端点都在路线节点集合中就高亮。图中存在连接非相邻路线节点的 chord 时，会高亮未实际经过的边。

建议构造连续节点对集合，如 `A->B`、`B->C`，再按边方向和 `bidirectional` 判断。

### 6.2 自动保存可能发生响应乱序

`useCharacterLoadout.ts:266-272` 只对发起请求做 600 ms debounce。请求发出后继续编辑会产生并行 PUT，较旧响应可能最后返回并覆盖 `loadout`，`savingLoadout` 也无法准确表达并发状态。

建议串行化保存或增加递增版本，只接受最后一次提交的响应；卸载组件时清理 timer。

### 6.3 文档与依赖管理漂移

- 同时提交 `package-lock.json` 与 `pnpm-lock.yaml`，`package.json` 未声明 `packageManager`。
- 当前 `node_modules` 来自 pnpm，执行 `npm ls --depth=0` 出现大量 `extraneous`。
- `docs/local-development.md:89`、`docs/server-deployment.md:170` 仍写“仓库未固定 npm lockfile”。
- README 使用 npm，建议保留 `package-lock.json`、删除 pnpm lock，并在 CI/文档使用 `npm ci`；若团队选 pnpm，则反向统一并声明版本。
- README 和在线改造方案仍列出 `/api/armor/repair`，当前前端使用 `/api/hideout/repair`。
- `docs/敌人生成器设计方案-V1.0.md` 仍描述运行时变体复用 `models.EnemyDef`，当前实现已经直接使用 `engine.Enemy`。

### 6.4 前端缺少测试和 lint 基线

`package.json` 只有 dev/build/preview，没有单元测试、组件测试、E2E 或 lint script。当前请求竞态、自动保存、SSE 重连、库存过滤都只能依赖人工回归。

建议优先增加 Vitest，覆盖 composables 的纯状态逻辑；再为登录、装备保存、Session 断线重连增加少量 Playwright 主流程。无需一开始追求全面覆盖。

## 7. 文件规模与职责边界

文件过大不自动等于过度设计，但以下文件已同时承担多种职责，继续增长会显著提高修改风险。

| 文件 | 行数 | 建议边界 |
| --- | ---: | --- |
| `frontend/src/style.css` | 1727 | 按基础布局、业务视图、Element Plus override 拆分；先删旧 selector |
| `service/recovery.go` | 695 | 策略解析、即时恢复、等待恢复、查询视图分离 |
| `service/hideout.go` | 651 | 设施视图、升级、维修、作业结算分离 |
| `frontend/src/types.ts` | 528 | 按 player/catalog/hideout/map/session 领域拆分，取消循环 barrel |
| `models/models.go` | 491 | 静态目录、玩家、敌人模板分文件 |
| `service/merchant.go` | 485 | 目录查询与交易命令分离 |
| `engine/route.go` | 456 | 校验、枚举、评分分离；算法本身保留 |
| `service/loadout.go` | 451 | 当前装备、预设、验证、失能替换分离 |
| `useAppWorkspace.ts` | 435 | 鉴权/启动、资源缓存、命令操作分离或引入 query cache |
| `scenario_snapshot.go` | 430 | 查询加载与 engine 转换分离 |
| `HideoutView.vue` | 426 | Generator、Workbench、Facility Detail 子组件 |
| `engine/types.go` | 411 | 按战斗、地图、事件、运行态拆分 |
| `event_config_validation.go` | 406 | 收敛到共享验证器后自然缩小 |
| `generator.go` | 405 | runtime 结算、燃料命令、view 分离 |

## 8. 验证结果

| 命令 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go test -count=1 -coverprofile=... ./...` | 通过，总语句覆盖率 45.7% |
| 覆盖率分包 | config 68.0%、engine 50.4%、repository/database 72.4%、service 46.7%、handler 0.0% |
| `go vet ./...` | 失败：`models.go:376` 重复/非法 JSON tag |
| `gopls check` | 同样确认 `HPCap` tag 问题 |
| `go mod tidy -diff` | 通过，无差异 |
| `npm run build` | 通过 |
| 前端产物 | JS 1,126.92 kB（gzip 362.74 kB），CSS 417.96 kB（gzip 57.65 kB），Vite 报告 chunk > 500 kB |
| `npm ls --depth=0` | 退出成功，但因 npm/pnpm 混用显示大量 `extraneous` |
| `npm audit --omit=dev` | 未完成：当前 npm 镜像不实现 audit endpoint，返回 404 |

关键覆盖缺口：全部 handler、恢复 merchant 分支、恢复 settle/start guard、scheduler start/stop、auth service、event config validator、generator 大部分命令、前端全部逻辑。

## 9. 建议整改顺序

1. 修复恢复计划实际方式记录和 SQLite 一致性备份，补回归测试。
2. 修复 JSON tag、当前补给库存过滤、Logs/Merchant 请求竞态。
3. 给 recovery、handler、scheduler 增加关键路径测试；前端引入 Vitest 基线。
4. 删除已确认僵尸函数、无效前端状态、旧 CSS 和重复事件绑定。
5. 统一维修 API、lockfile 和开发/部署文档。
6. 将启动时用户数据清扫改为版本化 data migration。
7. 对象化 scheduler，收敛事件验证和领域公式。
8. 最后评估迁移库替换、catalog repository 和 loadout preset 数据模型；这些改动 blast radius 较大，不宜与缺陷修复混在同一批。

## 10. 逐文件审查索引

说明：`通过` 表示未发现与本次目标直接相关的问题；`关联` 表示问题已在前文展开；`历史保留` 表示文件本身应保留，但其运行时镜像或兼容调用可清理。

### 10.1 根目录、依赖与文档

| 文件 | 结论 |
| --- | --- |
| `.gitignore` | 通过；已排除数据库、构建产物、PID 和 Session 手工快照，`session_full_flow.txt`/`session_user_broadcast.txt` 属于明确忽略的本地输出 |
| `README.md` | 关联：旧 `/api/armor/repair`、npm 安装方式需同步；架构概览基本准确 |
| `搜打撤放置游戏设计文档-V0.2.md` | 通过；产品设计文档，完整设计与 MVP 边界有区分，数值仍需以代码/测试为准 |
| `在线方向改造方案-V1.0.md` | 关联：认证改造遗留默认用户包装；API 清单仍含旧维修接口；其分阶段替换原则本身合理 |
| `start.ps1` | 关联：硬编码路径、按端口递归强杀、启动失败缺少回收 |
| `backend/go.mod` | 通过；依赖规模克制，`go mod tidy -diff` 无差异 |
| `backend/go.sum` | 通过；由 Go toolchain 管理 |
| `backend/main.go` | 关联：scheduler 生命周期由全局函数隐式管理；HTTP shutdown 未等待 worker |
| `docs/database-upgrade.md` | 关联：文档宣称 SQLite 备份保护，但实现未保证 WAL 一致性 |
| `docs/local-development.md` | 关联：仍称未提交 npm lockfile；测试说明高估了 recovery/handler 覆盖 |
| `docs/server-deployment.md` | 关联：仍称未提交 npm lockfile；PostgreSQL 备份流程较完整，SQLite 备份表述需修正 |
| `docs/敌人生成器设计方案-V1.0.md` | 关联：仍描述运行时变体复用 `models.EnemyDef`，与当前 `engine.Enemy` 实现漂移 |
| `frontend/package.json` | 关联：缺少 `packageManager`、test、lint；Element Plus 全量注册导致 bundle 偏大 |
| `frontend/package-lock.json` | 关联：与 `pnpm-lock.yaml` 并存，应只保留团队选定的一套 |
| `frontend/pnpm-lock.yaml` | 关联：与 npm 文档/lockfile 冲突，当前 `node_modules` 显示由 pnpm 安装 |
| `frontend/index.html` | 通过；Vite 标准入口 |
| `frontend/vite.config.ts` | 关联：无代码拆分配置；代理配置简洁合理 |
| `frontend/tsconfig.json` | 通过；标准 project references |
| `frontend/tsconfig.app.json` | 通过；启用 unused/fallthrough 检查，但未覆盖未消费 export |
| `frontend/tsconfig.node.json` | 通过 |
| `frontend/public/city-map.svg` | 通过；由 `MapGraphCanvas.vue` 实际引用 |

### 10.2 `backend/internal/config`

| 文件 | 结论 |
| --- | --- |
| `provision.go` | 通过；用户初始化入口职责清晰，当前无直接测试覆盖 |
| `seed.go` | 关联：seed 与用户初始数据耦合较深；默认用户兼容数据仍存在 |
| `seed_ammo.go` | 通过 |
| `seed_ammo_test.go` | 通过；覆盖弹药目录基础约束 |
| `seed_consumables.go` | 通过 |
| `seed_containers.go` | 通过；文件较长但属于静态内容定义 |
| `seed_crafting.go` | 通过；配方 JSON 构造需继续依赖验证器测试 |
| `seed_enemy_templates.go` | 通过；已替代旧 `seed_units.go` 的运行时方向 |
| `seed_equipment.go` | 通过 |
| `seed_events.go` | 关联：大量事件语义依赖 service/engine 双重校验；应显式填写 option 语义字段 |
| `seed_hideout.go` | 关联：设施效果与前端 ID 硬编码文本存在重复 |
| `seed_loot.go` | 通过；静态内容文件偏长但无额外抽象必要 |
| `seed_map.go` | 通过 |
| `seed_materials.go` | 通过 |
| `seed_merchants.go` | 通过 |
| `seed_recovery_progression_test.go` | 通过；只覆盖速率单调性，未覆盖恢复策略执行 |
| `seed_survival.go` | 关联：旧用户 survival 适配与 `userdata_upgrade.go` 职责部分重叠 |
| `settings.go` | 通过；环境读取简单，缺少单测但风险有限 |

### 10.3 `backend/internal/engine`

| 文件 | 结论 |
| --- | --- |
| `battle.go` | 关联：领域公式被 models/service/frontend 重复；战斗主体实现集中且可测试 |
| `battle_ammo_test.go` | 通过；覆盖弹药等级、命中区域和护甲耐久分档 |
| `battle_simulation.go` | 通过；纯模拟边界合理 |
| `battle_trace_test.go` | 通过 |
| `events.go` | 关联：支持集合与 service 验证重复；部分 condition 分支覆盖率为 0 |
| `events_effects.go` | 关联：effect 分支覆盖率低，配置新增时回归风险较高 |
| `events_state.go` | 通过；状态机辅助职责可辨识，部分物品消耗分支未覆盖 |
| `extraction_test.go` | 通过 |
| `loot.go` | 通过；自建容器抽取属于领域规则，无成熟通用库替代必要；多个随机分支未覆盖 |
| `replay_test.go` | 通过；确定性重放、hash 与移动时序测试是架构关键保护 |
| `route.go` | 通过但超限；自建路线枚举有领域评分需求，不认定为重复造轮子，建议按校验/枚举/评分拆文件 |
| `route_test.go` | 通过；覆盖九宫格路径与 anchor 排除 |
| `run.go` | 关联：`joinStrings` 零覆盖但存在调用路径不足；其余为运行期小型辅助 |
| `run_simulation.go` | 通过；版本入口 `SimulateRunVersion` 无覆盖 |
| `run_simulation_detail.go` | 通过但偏长；可按节点推进、撤离、结算阶段拆分 |
| `snapshot.go` | 关联：与 service 双重验证；存在读取后仅 `_ = container` 的残留代码 |
| `snapshot_ammo_test.go` | 通过 |
| `snapshot_validation.go` | 关联：验证集合重复，多个验证 helper 依赖间接覆盖 |
| `style.go` | 关联：按 option ID/effect ref 推断 Intent/Risk/Value 属于隐式兼容逻辑 |
| `survival.go` | 关联：自动治疗/资源恢复多分支覆盖率很低 |
| `trace.go` | 通过；事件单调性校验边界清晰 |
| `trace_test.go` | 通过 |
| `types.go` | 通过但超限；建议按战斗、地图、事件、运行态拆分，避免单一领域类型仓库继续膨胀 |
| `types_runtime.go` | 关联：`CloneItemStacks` 无引用；`SortItemStacks` 有实际调用，应保留 |

### 10.4 `backend/internal/handler`

| 文件 | 结论 |
| --- | --- |
| `handler.go` | 关联：`NewHandler(db, secureCookie ...bool)` 是不必要 variadic 兼容；整个 handler 包覆盖率 0% |
| `handler_auth.go` | 通过实现审查；Cookie/Bearer 边界清晰，但缺少路由级认证测试 |
| `handler_catalog.go` | 关联：service 默认用户包装已无必要；HTTP 参数和错误映射缺少测试 |
| `handler_crafting.go` | 通过实现审查；缺少 handler 测试 |
| `handler_hideout.go` | 关联：`/hideout/repair` 与旧 `/armor/repair` 重复 |
| `handler_session.go` | 关联：`ListNodes` 与 `GetMapGraph` 重复构造视图；旧 `RepairArmor` 路由应弃用 |
| `session_events.go` | 通过实现审查；SSE 的游标补拉、终态截断、write deadline 对应真实需求，不认定为重复造轮子；缺少 HTTP/SSE 测试 |

### 10.5 `backend/internal/models`

| 文件 | 结论 |
| --- | --- |
| `models.go` | 关联：`EnemyDef` 僵尸模型、`HPCap` JSON tag 错误、`EffectiveSkill` 领域公式放置不当、文件超限 |
| `models_hideout.go` | 通过；藏身处持久化模型拆分合理 |
| `models_recipe.go` | 通过 |
| `models_session.go` | 通过；Session/Pending/事件模型集中合理，状态字符串可考虑类型常量但无需大改 |

### 10.6 `backend/internal/repository/database`

| 文件 | 结论 |
| --- | --- |
| `database.go` | 关联：自建 migration 框架和活动 SQLite 文件直接复制备份；其 checksum/版本连续性保护有价值 |
| `verify_schema.go` | 关联：自行维护 schema 验证，和 migration 内容形成第三份结构知识 |
| `migrations_test.go` | 通过；双方言版本、checksum、schema 和备份失败均有覆盖，但缺少备份恢复一致性测试 |

#### PostgreSQL migrations

| 文件 | 结论 |
| --- | --- |
| `migrations/postgres/0001_initial.sql` | 历史保留；包含旧 `enemy_defs` 表，迁移历史不可直接删除 |
| `migrations/postgres/0002_user_scope.sql` | 历史保留；默认用户迁移来源，运行时代码无需继续保留默认用户包装 |
| `migrations/postgres/0003_auth_sessions.sql` | 通过 |
| `migrations/postgres/0004_economic_operations.sql` | 通过；支持幂等经济操作 |
| `migrations/postgres/0005_session_progress.sql` | 通过 |
| `migrations/postgres/0006_exploration_engine.sql` | 通过；快照与 pending run 字段对应真实一致性需求 |
| `migrations/postgres/0007_session_events.sql` | 通过 |
| `migrations/postgres/0008_ammunition.sql` | 历史保留；仍涉及旧 `enemy_defs`，后续模板迁移未删除旧表 |
| `migrations/postgres/0009_resource_lock.sql` | 通过 |
| `migrations/postgres/0010_map_graph.sql` | 通过 |
| `migrations/postgres/0011_hideout.sql` | 通过 |
| `migrations/postgres/0012_survival.sql` | 通过；数据回填较多，发布需依赖备份和集成测试 |
| `migrations/postgres/0013_hideout_expansion.sql` | 通过 |
| `migrations/postgres/0014_hideout_jobs.sql` | 通过 |
| `migrations/postgres/0015_terminate_legacy_sessions.sql` | 历史兼容迁移；一次性终止旧 Session 合理，不应复制到每次启动逻辑 |
| `migrations/postgres/0016_crafting.sql` | 通过 |
| `migrations/postgres/0017_enemy_templates.sql` | 通过；明确新模板体系替代写死敌人定义，旧 Go `EnemyDef` 可清理 |

#### SQLite migrations

| 文件 | 结论 |
| --- | --- |
| `migrations/sqlite/0001_initial.sql` | 历史保留；包含旧 `enemy_defs` 表 |
| `migrations/sqlite/0002_user_scope.sql` | 历史保留；同 PostgreSQL 结论 |
| `migrations/sqlite/0003_auth_sessions.sql` | 通过 |
| `migrations/sqlite/0004_economic_operations.sql` | 通过 |
| `migrations/sqlite/0005_session_progress.sql` | 通过 |
| `migrations/sqlite/0006_exploration_engine.sql` | 通过 |
| `migrations/sqlite/0007_session_events.sql` | 通过 |
| `migrations/sqlite/0008_ammunition.sql` | 历史保留；同 PostgreSQL 结论 |
| `migrations/sqlite/0009_resource_lock.sql` | 通过 |
| `migrations/sqlite/0010_map_graph.sql` | 通过 |
| `migrations/sqlite/0011_hideout.sql` | 通过 |
| `migrations/sqlite/0012_survival.sql` | 通过；SQLite 表重建/默认值差异已单独实现 |
| `migrations/sqlite/0013_hideout_expansion.sql` | 通过 |
| `migrations/sqlite/0014_hideout_jobs.sql` | 通过 |
| `migrations/sqlite/0015_terminate_legacy_sessions.sql` | 历史兼容迁移；同 PostgreSQL 结论 |
| `migrations/sqlite/0016_crafting.sql` | 通过 |
| `migrations/sqlite/0017_enemy_templates.sql` | 通过 |

### 10.7 `backend/internal/service`

| 文件 | 结论 |
| --- | --- |
| `ammunition.go` | 通过；预留、返还、降级补购职责清晰，错误边界较完整 |
| `ammunition_test.go` | 通过；覆盖预留返还、容量、降级补购、现金不足与部分预设库存 |
| `auth.go` | 通过实现审查；密码/Token 流程清晰，但整文件覆盖率 0% |
| `carry.go` | 关联：默认用户包装 `GetCarryCapacity` 可删除；容量公式应避免被前端复制 |
| `crafting.go` | 关联：`instanceRequiredItem` 吞掉数据库错误；目录查询可复用统一 repository |
| `crafting_test.go` | 通过；覆盖等级、材料、并发、容量回滚和延迟交付，文件偏长但测试场景有价值 |
| `economy.go` | 通过；幂等操作 claim 设计合理，错误/冲突分支覆盖不足 |
| `enemygen.go` | 通过；确定性、权重和口径匹配属于领域实现，无需外部随机生成库 |
| `enemygen_test.go` | 通过 |
| `enemygen_integration_test.go` | 通过；覆盖快照物化变体 |
| `engine_state.go` | 通过；DB 模型到纯 engine 状态的转换边界合理 |
| `event_config_validation.go` | 关联：与 engine 验证重复，文件 406 行且主入口覆盖率 0% |
| `facility_requirements.go` | 通过实现审查；需求查询/消耗职责明确，但读取视图分支缺少测试 |
| `generator.go` | 关联：太阳能查询静默降级；`GetGeneratorViewForUser`、`sortUint` 无引用；生命周期职责混杂 |
| `hideout.go` | 关联：文件超限；`hasInventoryQuantity` 无引用；设施视图、升级、维修、作业结算应分文件 |
| `inventory.go` | 关联：默认用户包装、9 类表顺序查询、删除函数遗留注释；交易核心事务实现可保留 |
| `inventory_capacity.go` | 关联：`inventoryUsage` variadic 仅为旧测试兼容；预设字段映射重复 |
| `inventory_test.go` | 通过；覆盖购买、失能替换、容量与现金不足 |
| `item_instances.go` | 通过；实例预留/返还边界清晰，部分失败路径无覆盖 |
| `loadout.go` | 关联：默认用户包装、三套预设字段爆炸、多个转换 helper 零覆盖 |
| `merchant.go` | 关联：默认用户包装、多目录查询、价格公式被前端复制；文件超限 |
| `merchant_ammo_test.go` | 通过 |
| `merchant_reputation.go` | 关联：`AwardReputation` 默认用户包装可删除；整个文件无覆盖 |
| `player_view.go` | 关联：压力阈值公式与 engine 重复；整文件无覆盖 |
| `recovery.go` | 关联：永久 running 缺陷、静默策略降级、公式重复、无引用 helper、关键路径覆盖不足 |
| `scenario_snapshot.go` | 关联：多表目录适配和 event 验证重复；查询与转换职责可拆分 |
| `scheduler.go` | 关联：包级生命周期不可重启/等待，Start/Stop/dispatch 覆盖率 0% |
| `session.go` | 关联：默认 user 时代 helper `splitIDs` 无引用；Session 主编排合理但文件偏长 |
| `session_ammo_integration_test.go` | 通过；覆盖 Session 弹药预留、结算和补购失败 |
| `session_engine_helpers.go` | 关联：`appendEngineReport`、`classifyResourceUnavailable` 无引用 |
| `session_engine_persistence.go` | 通过；持久化适配职责清晰，仓库容量分支覆盖不足 |
| `session_engine_run_settlement.go` | 关联：文件偏长，`lootQuantityEngine` 无引用；结算事务边界有价值 |
| `session_engine_settlement.go` | 通过实现审查；失能预设补购分支覆盖率 0% |
| `session_engine_worker.go` | 关联：worker 与全局 scheduler 生命周期耦合；恢复/续跑分支覆盖不足 |
| `session_engine_worker_test.go` | 通过但覆盖很窄，只验证缺弹药和旧 engine version |
| `session_events.go` | 关联：`isRecordNotFound` 无引用；Pending Run/事件持久化设计合理 |
| `session_events_test.go` | 通过；覆盖 pending hash、exactly-once、游标和终态事件 |
| `session_resource_guard.go` | 关联：消耗品解析失败静默为空，可能削弱 active Session 资源锁 |
| `session_resource_guard_test.go` | 通过；并发卖出、启动、维修和用户隔离测试有价值 |
| `usable_items.go` | 关联：与 catalog 多表适配重复，整文件覆盖率 0% |
| `userdata_upgrade.go` | 关联：每次启动全量扫描和破坏性清理，应版本化 |
| `userdata_upgrade_test.go` | 通过；覆盖旧库升级和未 seed 拒绝，但未覆盖 ID 删除/重命名补偿 |

### 10.8 `frontend/src`

| 文件 | 结论 |
| --- | --- |
| `main.ts` | 关联：全量 `.use(ElementPlus)` 与完整 CSS 是 bundle 主要来源之一 |
| `api.ts` | 通过；统一 axios 与错误提取合理；全局 unauthorized handler 为单 App 实例可接受 |
| `App.vue` | 关联：同步导入全部视图；`LogsView @refresh` 无效；入口承担大量 prop/event 接线 |
| `style.css` | 关联：1727 行、约 30 组旧 selector、多个历史布局并存；应先清理再拆分 |
| `types.ts` | 关联：528 行、`CarryCapacity`/`LootItem` 无引用；与 `types_inventory.ts` 形成 type-only 循环 barrel |
| `types_inventory.ts` | 关联：三套预设字段重复；`SessionEventType | string` 弱化类型；拆分后仍达 281 行 |
| `components/AppSidebar.vue` | 通过；导航按 active Session 动态展示，职责清晰 |
| `components/MapGraphCanvas.vue` | 关联：路线边高亮把有序路径降为 Set；组件样式较长但封装边界合理 |
| `components/SessionTimeline.vue` | 通过；事件展示集中可复用；event payload 仍是弱类型 `Record<string, unknown>`，可按高频事件逐步收紧 |
| `composables/characterLoadoutHelpers.ts` | 关联：三套预设的手工字段映射重复 |
| `composables/useAppWorkspace.ts` | 关联：21+ 请求全量启动、任一失败阻断全局、`nodes` 无消费、命令后重复刷新、文件超限 |
| `composables/useCharacterLoadout.ts` | 关联：当前补给未按库存过滤、自动保存响应乱序、前后端公式/商人规则重复 |
| `composables/useExploreDeployment.ts` | 通过实现审查；启动请求与恢复策略组装清晰，弹药库存每次通过数组扫描，数据量小时可接受 |
| `composables/useSessionStream.ts` | 通过；断线补拉、generation、指数退避、终态关闭都对应真实 SSE 边界，不建议仅为“少写代码”替换库 |
| `views/AuthView.vue` | 通过；使用浏览器表单约束和统一错误提示，边界简洁 |
| `views/CharacterView.vue` | 关联：依赖 `useCharacterLoadout` 的库存过滤/自动保存问题；模板偏长，后续可拆装备区与属性区 |
| `views/ExploreView.vue` | 通过；主要问题来自 composable 和全局加载 |
| `views/HideoutView.vue` | 关联：426 行、按 facility ID 硬编码效果、`{} as HideoutFacility` 隐藏缺失设施；建议拆子组件 |
| `views/InventoryView.vue` | 通过；展示逻辑简单；“物资类别”计数混入实例数属于轻微文案偏差 |
| `views/LiveSessionView.vue` | 通过；实时状态派生集中；路线高亮问题来自 MapGraphCanvas |
| `views/LogsView.vue` | 关联：`watchEffect` 重复请求、响应竞态、失败保留旧 detail |
| `views/MapView.vue` | 通过；当前固定展示首张地图，新增多地图选择前需确认产品需求 |
| `views/MerchantView.vue` | 关联：目录请求竞态、价格公式重复；`onMounted` 与 immediate watch 当前不会稳定重复请求，但结构可简化为单一 watch |

## 11. 明确不建议改动的部分

- 不拆掉 `engine` 的纯函数边界，也不把 GORM 模型传入 engine。
- 不删除 `scenario_hash`、`pending_run_hash`、唯一约束、lease 或资源锁。
- 不因“手写”就替换 SSE；当前实现已处理断线补拉、游标、终态和写超时。
- 不为路线规划强行引入通用图数据库或大型图算法库；当前地图规模和领域评分适合本地实现。
- 不删除历史 migration 中的旧表/字段；只清理运行时无引用模型和兼容入口。
- 不立即合并所有商品表或重做预设 schema；先通过 repository/cache 和测试降低重复成本。

## 12. 总评

项目已经从单用户原型演进为带账号隔离、后台调度、确定性模拟和持久化结算的 Online MVP，核心一致性设计质量高于普通原型。当前主要债务来自连续迭代留下的兼容层、重复适配和集中式文件，而非核心架构方向错误。

建议先处理两个 P1 和前端可复现缺陷，再做僵尸代码清理。migration 库替换、scheduler 对象化、catalog 统一 repository 属于后续结构治理，应独立排期并用现有测试保护行为。
