# 服务器部署

本文给出一套适用于当前项目的单机生产部署方式：Linux + PostgreSQL + systemd + Nginx。项目当前没有 Docker 部署文件，因此本文不覆盖 Docker/Kubernetes 方案。

## 1. 部署结构

~~~text
浏览器
  |
  | HTTPS / SSE
  v
Nginx :443
  |-- 静态文件 -> /opt/idle/frontend/dist
  +-- /api/* -> 127.0.0.1:8081
                      |
                      +-> systemd idle.service -> PostgreSQL
~~~

建议目录：

~~~text
/opt/idle/                         项目目录
/opt/idle/backend/                 Backend 源码
/opt/idle/frontend/dist/           Frontend 构建产物
/opt/idle/bin/idle                 Backend 可执行文件
/etc/idle/idle.env                 生产环境变量，不进入 Git
/etc/systemd/system/idle.service   systemd 服务文件
~~~

Backend 只监听 **127.0.0.1:8081**，外部流量统一由 Nginx 接收。生产环境必须使用 HTTPS，并把 **ALLOWED_ORIGINS** 设置为实际的 HTTPS 域名。

## 2. 前置条件和风险边界

服务器需要：

- Linux 发行版，下面命令以 Debian/Ubuntu 为例
- Go **1.21** 或更高版本
- Node.js LTS 和 npm
- PostgreSQL、Nginx、systemd
- 一个已解析到服务器的域名
- 具备服务器 sudo 权限的维护账号

执行前确认以下事项：

1. 已确认维护窗口，安装软件、重载 Nginx 和重启 Backend 都会影响服务。
2. 第一次 migration 和每次会改变 schema 的升级前，都已经完成 PostgreSQL 备份。
3. 数据库密码、Session 相关配置和其他密钥只写入服务器环境文件，不写入代码仓库。
4. 当前 **0006～0009** migration 会依次增加纯引擎调度、实时事件、分级弹药和用户资源版本锁。生产数据库执行前必须保留可恢复备份。

## 3. 安装系统依赖

以下命令会修改系统软件包和服务配置。执行前确认目标系统为 Debian/Ubuntu，回滚需要按发行版包管理策略移除新增软件包，并恢复原有 Nginx 配置。

~~~bash
sudo apt update
sudo apt install -y git gcc nginx postgresql
~~~

确认工具版本：

~~~bash
go version
node --version
npm --version
psql --version
nginx -v
~~~

如果 Go 或 Node.js 版本低于项目构建要求，先通过组织认可的安装源升级，再继续部署。

## 4. 创建运行用户和项目目录

这一步会创建系统用户并写入 **/opt/idle**。用户创建后不应使用 root 身份运行 Backend；回滚时保留项目目录中的上一份代码或发布包。

~~~bash
sudo useradd --system --home-dir /opt/idle --shell /usr/sbin/nologin idle
sudo install -d -m 755 -o idle -g idle /opt/idle
~~~

拉取代码时把占位符替换为实际仓库地址和发布版本：

~~~bash
sudo -u idle git clone https://git.example.com/your-org/idle.git /opt/idle
sudo -u idle git -C /opt/idle checkout your-release-tag
sudo install -d -m 755 -o idle -g idle /opt/idle/bin
~~~

如果 **/opt/idle** 已经有线上版本，请先备份当前版本信息，再按“升级和回滚”章节操作，不要直接覆盖未确认的生产目录。

## 5. 创建 PostgreSQL 数据库

创建数据库角色和数据库属于一次性管理操作。密码会被交互式读取，不要把密码写进 shell 历史或文档。

~~~bash
sudo -u postgres createuser --login --pwprompt idle
sudo -u postgres createdb --owner=idle idle
~~~

确认连接：

~~~bash
sudo -u postgres psql --dbname=idle --command='SELECT current_database(), current_user;'
~~~

## 6. 配置生产环境变量

创建配置目录并编辑环境文件。环境文件包含数据库凭据，权限应限制为 root 可写、**idle** 用户可读。

~~~bash
sudo install -d -m 750 -o root -g idle /etc/idle
sudoedit /etc/idle/idle.env
sudo chown root:idle /etc/idle/idle.env
sudo chmod 640 /etc/idle/idle.env
~~~

写入以下内容，并替换域名和数据库密码：

~~~text
APP_ENV=production
HTTP_ADDR=127.0.0.1:8081
DATABASE_DRIVER=postgres
DATABASE_URL=postgres://idle:URL_ENCODED_PASSWORD@127.0.0.1:5432/idle?sslmode=disable
TRUSTED_PROXIES=127.0.0.1
ALLOWED_ORIGINS=https://example.com
MIGRATE_ON_START=false
SEED_ON_START=false
~~~

说明：

- **DATABASE_URL** 中的密码必须进行 URL encoding，尤其注意 **@**、**:**、**/**、**?** 和 **#**。
- **MIGRATE_ON_START=false** 可以避免每次服务重启都自动执行 migration；升级时手动执行并检查结果。
- **SEED_ON_START=false** 可以避免服务重启时重复执行 seed。首次安装是否执行 seed 见下一节。
- **ALLOWED_ORIGINS** 不要写末尾 **/**。如果存在多个前端来源，用逗号分隔。

## 7. 构建 Backend 和 Frontend

构建会写入 **/opt/idle/bin** 和 **frontend/dist**，建议在停机窗口外先完成编译；只有切换版本、执行 migration 和重启服务时才进入维护窗口。

以 **idle** 用户打开 shell：

~~~bash
sudo -u idle -H bash
~~~

在该 shell 中构建 Backend：

~~~bash
cd /opt/idle/backend
go mod download
CGO_ENABLED=1 go build -o /opt/idle/bin/idle .
~~~

继续构建 Frontend：

~~~bash
cd /opt/idle/frontend
npm ci
npm run build
~~~

完成后退出 **idle** shell：

~~~bash
exit
~~~

项目同时引入 SQLite driver，Linux 构建安装 **gcc** 并使用 **CGO_ENABLED=1**，避免编译阶段缺少 CGO 环境。

当前仓库固定使用 `package-lock.json` 和 `npm@11.9.0`，部署时使用 **npm ci** 保证依赖可复现。只有主动更新依赖时才使用 **npm install**，并提交更新后的 lockfile。

## 8. 备份并执行数据库 migration

migration 会修改 schema，当前 migration 还可能改变旧 Session 状态。以下步骤属于高风险数据操作：备份文件必须确认可读、可恢复，确认目标数据库名称后再执行后续命令。

创建 PostgreSQL custom-format 备份：

~~~bash
sudo install -d -m 700 -o postgres -g postgres /var/backups/idle
backup_tag="$(date +%Y%m%d%H%M%S)"
backup_file="/var/backups/idle/idle-$backup_tag.dump"
sudo -u postgres pg_dump --format=custom --file="$backup_file" idle
sudo chmod 600 "$backup_file"
sudo -u postgres pg_restore --list "$backup_file" > /dev/null
~~~

加载生产环境变量并执行 migration：

~~~bash
sudo -u idle -H bash
set -a
. /etc/idle/idle.env
set +a
cd /opt/idle/backend
/opt/idle/bin/idle migrate
exit
~~~

第一次部署需要初始化静态数据时，再显式执行一次：

~~~bash
sudo -u idle -H bash
set -a
. /etc/idle/idle.env
set +a
cd /opt/idle/backend
/opt/idle/bin/idle seed
exit
~~~

后续升级只有在发布说明明确要求时执行 **seed**。不要把 **seed** 放入 systemd 的每次启动流程。

当前关键 migration：

| 版本 | 作用 | 主要风险 |
| --- | --- | --- |
| `0006_exploration_engine` | Session 快照、时间调度、lease、Pending Run 与单局唯一约束 | 旧 active Session 状态处理 |
| `0007_session_events` | 持久化实时/历史事件及事件唯一约束 | 事件表增长与索引写入 |
| `0008_ammunition` | 弹药定义、库存弹药、预设弹药和敌人弹药/护甲 | seed 与已有装备配置一致性 |
| `0009_resource_lock` | 角色 `resource_version` 并发锁 | 并发结算与交易冲突处理 |

不考虑旧数据时仍需执行全部 migration，禁止只创建最新表结构；`schema_migrations` 用于记录已应用版本。

## 9. 配置 systemd

创建 **/etc/systemd/system/idle.service**：

~~~bash
sudoedit /etc/systemd/system/idle.service
~~~

写入：

~~~ini
[Unit]
Description=Idle backend service
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=idle
Group=idle
WorkingDirectory=/opt/idle/backend
EnvironmentFile=/etc/idle/idle.env
ExecStart=/opt/idle/bin/idle
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
~~~

加载并启动服务：

~~~bash
sudo systemctl daemon-reload
sudo systemctl enable idle
sudo systemctl start idle
sudo systemctl status idle --no-pager
~~~

本机健康检查：

~~~bash
curl --fail http://127.0.0.1:8081/api/health
~~~

查看日志：

~~~bash
sudo journalctl -u idle -n 100 --no-pager
sudo journalctl -u idle -f
~~~

### systemd 变更回滚

如果服务文件配置错误，先停止服务，恢复修改前的 **/etc/systemd/system/idle.service**，再重新加载并启动：

~~~bash
sudo systemctl stop idle
sudo systemctl daemon-reload
sudo systemctl start idle
sudo systemctl status idle --no-pager
~~~

如果新版本二进制启动失败，恢复上一版本二进制后重复上述流程。不要通过反复重启掩盖数据库连接或 migration 错误，应先查看 **journalctl**。

## 10. 配置 Nginx

创建站点配置：

~~~bash
sudoedit /etc/nginx/sites-available/idle
~~~

将 **example.com** 替换为实际域名：

~~~nginx
server {
    listen 80;
    server_name example.com;

    root /opt/idle/frontend/dist;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_read_timeout 1h;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
~~~

SSE 依赖 **proxy_buffering off**，避免事件被 Nginx 聚合后延迟显示；**proxy_read_timeout 1h** 允许长连接保持。Backend 同时返回 `X-Accel-Buffering: no` 并每 20 秒发送 heartbeat。

启用配置前先检查语法，再重载 Nginx：

~~~bash
sudo ln -s /etc/nginx/sites-available/idle /etc/nginx/sites-enabled/idle
sudo nginx -t
sudo systemctl reload nginx
~~~

Nginx 配置变更会影响站点流量；**nginx -t** 失败时不要 reload。回滚时恢复原配置文件或移除本次新增的站点链接，重新执行 **nginx -t**，确认通过后再 reload。不要移除其他业务站点的配置。

## 11. HTTPS

生产环境应使用 HTTPS，避免登录 Cookie 和用户请求在公网明文传输。可以使用组织已有的证书系统或 ACME 工具完成证书申请和 Nginx 配置；证书工具可能会自动改写 Nginx 配置，执行前保存当前配置备份。

启用 HTTPS 后：

1. 确认域名 HTTPS 页面可以打开。
2. 把 **/etc/idle/idle.env** 中的 **ALLOWED_ORIGINS** 改为 **https://实际域名**。
3. 执行 **sudo systemctl restart idle** 使环境变量生效。
4. 执行 **sudo nginx -t**，通过后 **sudo systemctl reload nginx**。
5. 重新验证登录、刷新页面和 **/api/health**。

如果证书更新导致 Nginx 配置异常，恢复证书工具修改前的配置，先通过 **nginx -t**，再 reload。

## 12. 线上验收

在服务器上检查 Backend：

~~~bash
curl --fail http://127.0.0.1:8081/api/health
sudo systemctl is-active idle
~~~

从外部检查 Nginx 和 Frontend：

~~~bash
curl --fail --head https://example.com/
curl --fail https://example.com/api/health
~~~

浏览器验收：

- 首页静态资源加载成功，没有 404 或 mixed content。
- 可以注册、登录、刷新当前用户信息。
- Network 中 **/api** 请求使用当前域名，返回状态码正常。
- 可以从部署页开始行动，并在实时行动页看到事件时间线。
- 容器、掉落、战斗攻击、弹药补给和单局结算事件会按时间出现。
- 刷新实时页或短暂中断网络后，REST 补拉与 SSE 重连不丢失、不重复事件。
- 中止行动后不再显示未来事件，历史页仍可查看已经发生的事件。
- 重启 Backend 后，到期 Session 可以重新领取 lease 并继续结算 Pending Run。
- systemd 日志没有数据库、migration、scheduler 或 panic 错误。

## 13. 升级流程

下面流程按“有停机窗口、先备份、手动 migration”的方式执行：

1. 记录当前版本：**git -C /opt/idle rev-parse HEAD**。
2. 备份 PostgreSQL，并确认 **pg_restore --list** 成功。
3. 更新代码并切换到目标 release tag 或 commit。
4. 重新构建 Backend 和 Frontend。
5. 停止 **idle** 服务。
6. 使用生产环境变量执行 **/opt/idle/bin/idle migrate**。
7. 启动 **idle** 服务，检查 **systemctl status** 和 **/api/health**。
8. 检查 Nginx 语法并 reload，然后执行浏览器验收。

停止服务和执行 migration 期间，用户请求会中断或无法完成；在高峰期不要执行。

## 14. 回滚流程

### 代码或构建产物回滚

如果数据库 schema 没有发生不兼容变化：

1. 停止 **idle** 服务。
2. 恢复上一版本代码、Backend 二进制和 **frontend/dist**。
3. 启动 **idle** 服务。
4. 检查本机健康接口、Nginx 和浏览器登录流程。

### 数据库回滚

如果目标版本已经执行了改变 schema 或数据的 migration，仅恢复旧二进制不够。必须在确认备份文件和目标数据库后，在维护窗口恢复数据库：

~~~bash
sudo systemctl stop idle
sudo -u postgres dropdb idle
sudo -u postgres createdb --owner=idle idle
sudo -u postgres pg_restore --clean --if-exists --no-owner --dbname=idle /var/backups/idle/idle-YYYYMMDDHHMMSS.dump
sudo systemctl start idle
sudo systemctl status idle --no-pager
~~~

**dropdb** 会删除当前数据库，属于不可逆的高风险操作。执行前必须再次确认 **BACKUP_FILE**、数据库名称和维护窗口；不要把这段命令用于未经确认的环境。恢复后使用与备份匹配的代码版本，并重新检查 **/api/health**、登录和 Session 流程。

## 15. 常见故障

### idle.service 启动失败

~~~bash
sudo systemctl status idle --no-pager
sudo journalctl -u idle -n 200 --no-pager
~~~

重点检查 **/etc/idle/idle.env** 的 **DATABASE_URL**、文件权限和 PostgreSQL 是否正在运行。

### 本机健康检查失败

确认 **HTTP_ADDR=127.0.0.1:8081** 与 systemd 配置一致，并检查端口监听：

~~~bash
sudo ss -ltnp | grep ':8081'
~~~

### Nginx 返回 502

先直接访问 Backend：

~~~bash
curl --fail http://127.0.0.1:8081/api/health
~~~

如果失败，处理 systemd 或数据库问题；如果成功，检查 Nginx **proxy_pass**、站点配置和 **sudo nginx -t** 输出。

### 页面刷新后出现 404

确认 Frontend 已通过 **npm run build** 生成 **/opt/idle/frontend/dist/index.html**，并保留 **location /** 中的 **try_files $uri $uri/ /index.html;**。

### CORS 或登录异常

优先让 Frontend 和 Backend 通过同一个 HTTPS 域名提供服务。若使用独立 Frontend 域名，确认 **ALLOWED_ORIGINS** 是完整的来源地址，协议、域名和端口必须完全匹配。

### SSE 反复重连或事件延迟

1. 确认 Nginx 的 `/api/` location 保留 **proxy_buffering off** 和 **proxy_read_timeout 1h**。
2. 使用浏览器 Network 检查 `/api/session/:id/events/stream` 是否返回 `200` 和 `text/event-stream`。
3. 查看 `journalctl`，确认 scheduler、数据库 lease 和事件查询没有报错。
4. 调用 `/api/session/:id/events?afterId=最后事件ID`，验证 REST 是否能返回已到 `available_at` 的缺失事件。
5. 多实例部署时检查各实例系统时间同步；事件展示依赖 `available_at`。

## 16. SQLite 单机替代方案

如果只是单机低并发部署，可以把环境变量改为：

~~~text
DATABASE_DRIVER=sqlite
DATABASE_URL=/opt/idle/data/idle.db
~~~

并创建数据目录：

~~~bash
sudo install -d -m 750 -o idle -g idle /opt/idle/data
~~~

SQLite 连接会自动附加 `_busy_timeout=5000`，用于等待短暂写锁；它仍只适合单 Backend 实例。部署前备份 **idle.db**，不要让多个 Backend 进程同时写同一个文件；需要多实例、独立扩容或更高并发时使用 PostgreSQL。

## 数据库升级

生产升级顺序固定为 `go run . migrate && go run . seed` 后再启动服务：启动时会自动执行版本化
SQL 迁移、结构完整性校验和对玩家存量数据的幂等适配。适配依赖目录种子——若跳过 seed 启动，
适配会拒绝执行并提示补种，绝不会在空目录下运行以免误删玩家资产。

SQLite 仅在确有待应用迁移时生成备份 `idle.db.pre-upgrade.bak`，普通启动不触碰该文件；
因权限等原因无法生成备份时迁移直接中止。PostgreSQL 环境请务必在部署前执行 `pg_dump`。
迁移内容校验和会检测对已发布迁移文件的任何改动。完整流程与破坏性变更配方见
[database-upgrade.md](./database-upgrade.md)。
