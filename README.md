# 搜打撤放置挂机文字游戏 · V0.2 Online MVP

> 账号登录 · 用户数据隔离 · 纯引擎重放 · 实时探索 · 离线补算 · 分级弹药战斗

## 目录结构
```
D:/idle
├── backend/          # Go + Gin + GORM + SQLite
│   ├── main.go
│   ├── idle.db       # SQLite 游戏存档，首次启动时由 migration/seed 生成
│   └── internal/
│       ├── engine/                # 无数据库依赖、可确定性重放的纯探索引擎
│       ├── config/                # 静态内容 seed 与运行配置
│       ├── models/                # 账号、库存、装备、Session 与事件模型
│       ├── repository/database/   # SQLite/PostgreSQL 连接与版本化 migration
│       ├── service/               # 快照、调度、资源准备与事务结算
│       └── handler/               # HTTP API、鉴权与 SSE 事件流
└── frontend/         # Vue 3 + TypeScript + Vite + Element Plus
    ├── src/views/LiveSessionView.vue       # 实时探索界面
    ├── src/components/SessionTimeline.vue  # 事件时间线
    ├── src/composables/                    # REST/SSE 合并与断线重连
    └── vite.config.ts
```

## 已实现 MVP 范围（对齐文档 11.2）

- [x] 1 张地图废弃城区与 6 个节点
- [x] 12 个自动事件
- [x] 1 名可改名的持久玩家角色
- [x] 6 类武器、轻重护甲、6 类消耗品与多口径 N1～N6 分级弹药
- [x] 自动战斗：发现→绕行→距离→先手→命中→命中区域→穿甲与肉伤→压力→脱离→热度
- [x] 伤势、压力与自动撤离判定
- [x] 纯引擎只读取 `ScenarioSnapshot + RunInput`，同版本、同 seed 和同输入可确定性重放
- [x] Session 保存 `engine_version + scenario_snapshot + scenario_hash`，历史行动不受静态配置更新影响
- [x] 数据库到期任务、原子 lease 和真实时间驱动的离线补算，可在服务重启后继续推进
- [x] `(session_id, run_index)` 与事件序列唯一约束，Pending Run 带完整性哈希，重试不会重复结算
- [x] 实时探索页展示路线、节点、容器、掉落、战斗过程与结算；REST/SSE 支持断线补拉
- [x] 节点容器自动搜集：单个容器可为空或产出多件物品，敌人背包复用容器规则
- [x] 弹药按口径和等级独立占用仓库格，每格最多堆叠 999 发
- [x] 武器商人仅出售 N1～N4 弹药；好感度控制等级，N5～N6 保留给非商店来源
- [x] 预设高级弹药耗尽后，自动购买同口径、当前可购的最高等级弹药；无法补给时以资源不足结束
- [x] 成功撤离按携行容量结算掉落，写回剩余弹药、消耗品、护甲耐久与商人好感度
- [x] 固定商品目录的商人买卖、仓库容量、护甲实例维修
- [x] 用户注册/登录、HttpOnly 会话 Cookie 与 Bearer Token 鉴权
- [x] 角色、装备、仓库、护甲、商人状态和行动记录按用户隔离
- [x] 活跃 Session 锁定相关装备与资源，角色资源版本锁防止并发结算冲突
- [x] 伤势等待通过 `waiting_injury + next_run_at` 恢复，离线期间只补算已经到期的局
- [x] 商人购买/出售支持 `Idempotency-Key`，避免网络重试重复扣款

## 明确不做（11.3）
多人小队、玩家战斗指令、动态地图/警戒、阵营、动态市场、复合甲、配件树

## 启动

### 后端 :8081
```bash
cd D:/idle/backend
go run .
# 健康检查 http://localhost:8081/api/health
```

首次使用 PostgreSQL 或生产环境时，先执行版本化 migration，再初始化静态数据：

```bash
cd D:/idle/backend
go run . migrate
go run . seed
```

运行配置通过环境变量读取：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ENV` | `development` | `production` 环境默认关闭启动时 migration 和 seed |
| `HTTP_ADDR` | `:8081` | HTTP 监听地址 |
| `DATABASE_DRIVER` | `sqlite` | `sqlite` 或 `postgres` |
| `DATABASE_URL` | `idle.db` | SQLite 文件路径或 PostgreSQL DSN |
| `TRUSTED_PROXIES` | 空 | 逗号分隔的可信代理列表 |
| `ALLOWED_ORIGINS` | 本地前端地址 | 逗号分隔的 CORS 来源列表 |
| `MIGRATE_ON_START` | 开发环境 `true` | 是否启动时执行 migration |
| `SEED_ON_START` | 开发环境 `true` | 是否启动时初始化基础数据 |

### 前端 :5173
```bash
cd D:/idle/frontend
npm ci
npm run dev   # http://localhost:5173 代理 /api -> :8081
# 或 npm run build + preview
```

## 核心接口
- `POST /api/auth/register` 注册并自动创建角色与初始装备
- `POST /api/auth/login` 登录并签发 HttpOnly 会话 Cookie
- `POST /api/auth/logout` 撤销当前会话
- `GET /api/auth/me` 获取当前登录用户
- `GET/PUT /api/player`
- `GET /api/weapons|ammos|armors|armor-instances|consumables|loot`
- `GET /api/chestrigs|backpacks|helmets|headsets|maps|enemies`
- `POST /api/session/start` 创建后台挂机会话，返回 `202`，前端随后进入实时行动页
- `GET /api/session/:id` 查看 Session 与单局报告
- `GET /api/session/:id/events` 读取已到时间线事件
- `GET /api/session/:id/events/stream` 通过 SSE 推送实时事件，支持 `Last-Event-ID` 断线续传
- `GET /api/sessions` 历史会话
- `POST /api/session/:id/abort` 中止后台会话
- `GET /api/inventory/capacity` 查看仓库容量
- `GET/PUT /api/loadout` 管理当前装备、弹药类型、携弹量与三套失能补购预设
- `GET /api/loadout/capacity` 查看当前携行容量
- `GET /api/merchants|/api/merchants/:id/catalog` 查看商人与商品
- `POST /api/merchant/purchase|/api/merchant/sell` 买入或出售物品；建议携带 `Idempotency-Key`
- `POST /api/hideout/repair` 将归零护甲加入维修队列，payload 为 `{"armorInstanceId": number}`

## 数值验证

- 命中率：`clamp(武器命中+(操控-50)*0.4+距离+伏击-目标规避-压力*0.25, 5, 95)`
- 遭遇概率：`clamp(节点基础遇敌概率+热度(-撤离模式25点), 5, 90)`；未配置节点的基础值回落为 60
- 伏击先手：单方先发现时先手 +15；单方先发现方按自己武器最优距离段接管接敌距离（被伏击方被拖入对方射程）
- 发现率：`clamp(50+(侦察−隐蔽)*0.5, 10, 90)`，侦察按感知×0.7+智力×0.1（双方真实属性，不再固定 50/40）；耳机听力等级加成玩家发现率（+听力×3）并降低被敌人发现率（−听力×2）；绕行率 `clamp(50+(潜行×0.5+敏捷×0.15+隐蔽−敌人感知×0.45)*0.5, 10, 95)`
- 敌人武器操控由敏捷/感知/压制真实属性推导（替代固定 55），敌人抗压来自模板属性（替代固定 40）
- 搜索暴露率：`clamp(风险等级×8 − (运气×0.15+感知×0.1), 2, 40)`，命中后按惩罚形态结算（当前 expose：耗时翻倍+热度3）；运气额外多搜一件（掷骰≤运气×0.3）
- 生存消耗：能量/饮水每小时基础 8/10，随生存属性降低（下限 5）；自动医疗触发阈值 60% 随医疗属性降低
- 击倒任何带枪敌人可搜缴其战后剩余弹药（按组折算携带占用）；`BossAmmoDrop` 字段已废弃
- 护甲覆盖判定失败时视为命中无甲区域，造成完整肉伤且不损失护甲耐久
- 护甲有效等级随耐久降档：`>50%` 保持原等级，`>25%` 降 1 级，`<=25%` 降 2 级
- 穿甲差值为 `弹药等级 - 护甲有效等级`；实伤保留率依次为 `100% / 90% / 75% / 15% / 8% / 3%`
- N1～N6 肉伤倍率逐级从 `1.20` 降至 `0.90`，护甲伤害倍率从 `0.50` 升至 `1.40`
- 验收：不同口径、弹药等级、护甲等级和耐久应产生可观察的击杀效率与资源消耗差异

## 待调参数（文档12.3）
- 单局时长与离线上限默认值（离线时限当前为 1440 分钟）
- 伤势等待是否消耗会话时限（当前计入）
- 熟练度曲线（当前线性+1/胜场）
