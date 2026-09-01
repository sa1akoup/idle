# 本地启动与调试

本文针对当前项目的本地开发环境，默认使用 Windows PowerShell、SQLite、Go backend 和 Vue + Vite frontend。

## 1. 运行结构

| 部分 | 地址 | 说明 |
| --- | --- | --- |
| Backend | **http://localhost:8081** | Go + Gin，提供 **/api** 接口和 Session scheduler |
| Frontend | **http://localhost:5173** | Vue + Vite 开发服务器 |
| Database | **backend/idle.db** | 本地默认 SQLite 文件 |

Vite 会把 **/api** 请求代理到 **http://localhost:8081**。浏览器正常访问时只需要打开 Frontend 地址。

## 2. 环境要求

安装并确认以下工具可用：

- Go **1.21** 或更高版本
- Node.js LTS 和 npm
- PowerShell
- 可访问 **localhost:8081** 和 **localhost:5173** 的浏览器

在 PowerShell 中检查版本：

~~~powershell
go version
node --version
npm --version
~~~

## 3. 首次启动 Backend

打开第一个 PowerShell 窗口：

~~~powershell
Set-Location D:\idle\backend
go mod download
~~~

开发环境默认配置如下：

- **DATABASE_DRIVER=sqlite**
- **DATABASE_URL=idle.db**
- **MIGRATE_ON_START=true**
- **SEED_ON_START=true**
- **HTTP_ADDR=:8081**

为了让当前窗口的行为明确，可以显式设置配置后启动：

~~~powershell
$env:APP_ENV = "development"
$env:DATABASE_DRIVER = "sqlite"
$env:DATABASE_URL = "idle.db"
$env:MIGRATE_ON_START = "true"
$env:SEED_ON_START = "true"
$env:HTTP_ADDR = ":8081"
go run .
~~~

Backend 启动后会自动执行 migration 和 seed，然后监听 **:8081**。保持这个窗口运行，日志和 panic 会直接显示在窗口中；停止服务使用 **Ctrl+C**。

验证健康检查：

~~~powershell
Invoke-RestMethod -Uri "http://localhost:8081/api/health"
~~~

需要手动执行数据库操作时，在 **D:\idle\backend** 目录运行：

~~~powershell
go run . migrate
go run . seed
~~~

**seed** 会写入默认静态数据。只有在首次初始化或明确需要刷新 seed 数据时执行，避免把它当作每次启动命令。

## 4. 启动 Frontend

打开第二个 PowerShell 窗口：

~~~powershell
Set-Location D:\idle\frontend
pnpm install
$env:VITE_API_TARGET = "http://localhost:8081"
pnpm run dev
~~~

当前仓库固定使用 `pnpm-lock.yaml` 和 `pnpm@11.8.0`。全新检出或需要按锁定版本重建依赖时使用 **pnpm install --frozen-lockfile**；只有主动更新依赖时才使用 **pnpm install**，并提交更新后的 lockfile。

启动后打开：

~~~text
http://localhost:5173
~~~

建议通过页面完成注册和登录，再检查浏览器 DevTools 的 Network 面板，确认请求是否经过 **/api** 并返回成功状态。

## 5. 快速启动脚本

项目根目录提供 **start.ps1**，可以一次启动两个服务：

~~~powershell
Set-Location D:\idle
.\start.ps1
~~~

该脚本会启动 Go 和 npm 进程并把入口 PID 记录到 `.run/dev.pid`，同时支持停止：

```powershell
.\start.ps1 -Stop
```

`-Stop` 会校验 `.run/dev.pid` 中记录的入口 PID 和启动时间，再递归停止对应进程树（包括 `go run` 派生的孙进程）。端口命中但无法证明属于本项目的进程只会提示，不会自动强杀。调试定位启动错误时仍建议使用上面的两个可见 PowerShell 窗口前台运行。

## 6. 本地测试与构建

### Backend

在 **D:\idle\backend** 执行：

~~~powershell
go build ./...
go vet ./...
go test -count=1 ./...
go test -race ./...
~~~

**go test -race ./...** 需要 CGO 和可用的 C 编译环境。纯引擎、Session 调度、事件流、弹药补给和用户资源隔离都已纳入 Backend 测试基线；任何失败都应先按代码回归排查。

### Frontend

在 **D:\idle\frontend** 执行：

~~~powershell
pnpm run build
~~~

当前 **package.json** 没有独立的 frontend test script。**pnpm run** 可以查看项目已定义的脚本。

## 7. 常用调试方式

### Backend 断点调试

在 IDE 中把工作目录设为 **D:\idle\backend**，启动命令使用：

~~~text
go run .
~~~

调试配置至少需要保留以下环境变量：

~~~text
APP_ENV=development
DATABASE_DRIVER=sqlite
DATABASE_URL=idle.db
MIGRATE_ON_START=true
SEED_ON_START=true
HTTP_ADDR=:8081
~~~

Session 相关逻辑主要位于 **backend/internal/service**，纯探索规则位于 **backend/internal/engine**。调试调度问题时同时观察 Backend 日志和数据库中的 **sessions.next_run_at**、**sessions.current_run_started_at**、**sessions.last_processed_at**、**sessions.lease_until** 和 **sessions.pending_run_index**。

实时事件保存在 **session_events**。重点检查：

- `sequence` 是否在每局内连续，`offset_sec` 是否单调不倒退。
- `available_at` 是否已到当前时间；未来事件不会通过 REST 或 SSE 提前返回。
- `(session_id, run_index, sequence)` 是否保持唯一。
- Pending Run 恢复时是否使用原结果结算，禁止重新模拟。

### Frontend 调试

浏览器打开 **http://localhost:5173** 后：

1. 在 Network 中过滤 **/api**，确认请求 URL、状态码和响应体。
2. 在 Console 中检查 Vue、Axios 和运行时错误。
3. 实时页同时检查 **/session/:id/events** 和 **/session/:id/events/stream**；SSE 请求应保持 pending 并持续收到事件或 heartbeat。
4. 修改 Vue/TypeScript 后，Vite 会自动热更新；Backend 修改后需要重新启动 **go run .**。

### 修改 Backend 端口

如果 **8081** 已被占用，在 Backend 窗口设置新端口：

~~~powershell
$env:HTTP_ADDR = ":8082"
go run .
~~~

然后在 Frontend 窗口把代理目标改为新端口：

~~~powershell
$env:VITE_API_TARGET = "http://localhost:8082"
pnpm run dev
~~~

### 修改 Frontend 端口

Vite 默认使用 **5173**。临时改为 **5174**：

~~~powershell
pnpm run dev -- --port 5174
~~~

如果 Backend 的 **ALLOWED_ORIGINS** 被显式设置，需要同时加入新的 Frontend 地址：

~~~powershell
$env:ALLOWED_ORIGINS = "http://localhost:5174"
~~~

开发环境不设置 **ALLOWED_ORIGINS** 时，Backend 默认允许 **localhost:5173** 和 **127.0.0.1:5173**。

## 8. 重置本地 SQLite 数据

重置会清除本地用户、Session、库存和其他 SQLite 数据。必须先停止 Backend，并保留备份。

~~~powershell
Set-Location D:\idle\backend
$dbBackupName = "idle.db.bak-{0}" -f (Get-Date -Format "yyyyMMddHHmmss")
Rename-Item -LiteralPath "idle.db" -NewName $dbBackupName
go run .
~~~

Backend 会在下次启动时创建新的 **idle.db** 并执行 migration/seed。确认新库可用后，再按需处理备份文件；不要在服务仍运行时直接移动或删除 SQLite 文件。

## 9. 本地验收清单

- **GET http://localhost:8081/api/health** 返回成功
- 页面可以打开，且静态资源没有 404
- 可以注册、登录和刷新当前用户信息
- 浏览器 Network 中 **/api** 请求没有指向错误端口
- 部署页可以选择同口径弹药和携弹量，开始行动后自动进入实时探索界面
- 实时页可以看到节点、容器、掉落、战斗和结算事件，事件不会重复或倒序
- 刷新页面或短暂断网后，REST 补拉与 SSE 重连不会丢失已有事件
- 高级弹药耗尽后会补购当前可售的同口径最高等级弹药；无法补购时正常结束 Session
- 已完成、已中止和失败 Session 可以从日志页打开历史事件
- Backend 日志没有数据库连接、migration 或 scheduler 错误

## 10. 常见问题

### 页面打开但接口 502/连接失败

确认 Backend 窗口仍在运行，并检查：

~~~powershell
Invoke-RestMethod -Uri "http://localhost:8081/api/health"
~~~

如果健康检查失败，先处理 Backend 启动日志；如果健康检查成功，检查 Frontend 窗口中的 **VITE_API_TARGET**。

### idle.db 不在预期位置

SQLite 相对路径以 Backend 进程的工作目录为基准。请从 **D:\idle\backend** 运行 **go run .**，或把 **DATABASE_URL** 设置为绝对路径。

### 实时事件停在“重新连接”

1. 确认 Backend 仍在运行，并检查 `/api/health`。
2. 在 Network 中确认 `/api/session/:id/events/stream` 返回 `200`，响应类型为 `text/event-stream`。
3. 检查当前用户是否拥有该 Session；越权访问不会返回事件。
4. 调用 `/api/session/:id/events?afterId=最后事件ID`，确认是否有已到 `available_at` 的补拉事件。
5. Backend 重启后前端会按 1～30 秒指数退避重连，无需刷新或重新开始 Session。

### Postgres 本地调试

本地默认使用 SQLite。需要验证 Postgres 时，设置：

~~~powershell
$env:APP_ENV = "development"
$env:DATABASE_DRIVER = "postgres"
$env:DATABASE_URL = "postgres://idle:URL_ENCODED_PASSWORD@localhost:5432/idle?sslmode=disable"
$env:MIGRATE_ON_START = "true"
$env:SEED_ON_START = "true"
go run .
~~~

运行前确认数据库和用户已创建。不要把真实密码写入仓库或提交到 Git。
