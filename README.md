# 搜打撤放置挂机文字游戏 · V0.2 MVP

> 单一玩家角色 · 废弃城区 · 自动事件与战斗 · 多局连续会话

## 目录结构
```
D:/idle
├── backend/          # Go + Gin + GORM + SQLite
│   ├── main.go
│   ├── idle.db       # SQLite 游戏存档，首次启动时由 seed 生成
│   └── internal/
│       ├── models/models.go
│       ├── config/seed.go
│       ├── service/battle.go   # 8阶段战斗引擎
│       ├── service/session.go  # 事件+挂机会话循环
│       └── handler/handler.go
└── frontend/         # Vue 3 + TypeScript + Vite + Element Plus
    ├── src/App.vue   # 七项功能工作台
    ├── src/views/    # 功能页面
    └── vite.config.ts
```

## 已实现 MVP 范围（对齐文档 11.2）

- [x] 1 张地图废弃城区与 6 个节点
- [x] 12 个自动事件
- [x] 1 名可改名的持久玩家角色
- [x] 6 类武器、轻重护甲与 6 类消耗品
- [x] 8阶段自动战斗：发现→绕行→先手→命中→护甲→压力→脱离→热度（`service/battle.go:1`）
- [x] 伤势、压力与自动撤离判定
- [x] 后台多局连续会话，持续至离线时限、失能、护甲损坏、资源耗尽或手动中止
- [x] 100次预模拟（出发预测），会话保存随机 seed 供报告追踪
- [x] 节点容器自动搜集：单个容器可为空或产出多件物品，敌人背包复用容器规则
- [x] 成功撤离按携行容量结算容器掉落，写回弹药、消耗品、护甲耐久与商人好感度
- [x] 固定商品目录的商人买卖、仓库容量、护甲实例维修

## 明确不做（11.3）
多人小队、行动倾向、动态地图/警戒、阵营、动态市场、复合甲、配件树

## 启动

### 后端 :8081
```bash
cd D:/idle/backend
go run .
# 健康检查 http://localhost:8081/api/health
```

### 前端 :5173
```bash
cd D:/idle/frontend
npm install
npm run dev   # http://localhost:5173 代理 /api -> :8081
# 或 npm run build + preview
```

## 核心接口
- `GET/PUT /api/player`
- `GET /api/weapons|armors|consumables|maps|nodes`
- `POST /api/session/preview` 100次模拟预测
- `POST /api/session/start` 创建后台挂机会话，返回 `202`
- `GET /api/session/:id` 查看报告
- `GET /api/sessions` 历史会话
- `POST /api/session/:id/abort` 中止后台会话
- `GET /api/inventory/capacity` 查看仓库容量
- `GET/PUT /api/loadout` 管理当前装备与三套失能补购预设
- `GET /api/merchants|/api/merchants/:id/catalog` 查看商人与商品
- `POST /api/merchant/purchase|/api/merchant/sell` 买入或出售物品
- `POST /api/armor/repair` 维修归零护甲

## 数值验证
- 武器基线对齐文档5.2，护甲对齐5.4
- 命中率 `clamp(基础+(操控-50)*0.4+距离-规避-压力*0.25,5,95)`
- 护甲减伤 `protect/(protect+pen)` 上限80%，耐久损失 `damage*(0.15+穿透比*0.35)`
- 验收：不同装备组合产生可观察的撤离率和资源消耗差异

## 待调参数（文档12.3）
- 单局时长与离线上限默认值（离线时限当前为1440分钟）
- 伤势等待是否消耗会话时限（当前计入）
- 熟练度曲线（当前线性+1/胜场）
