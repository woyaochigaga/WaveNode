# Sub2API 项目扫描与本地开发指南

> 本文基于当前仓库源码、Compose 文件、Makefile、前后端入口和路由整理，适合第一次接手项目、启动本地环境以及继续开发时查阅。

## 1. 项目定位

Sub2API 是一个 AI API 网关和订阅配额分发平台：用户使用平台生成的 API Key 请求 `/v1` 等兼容接口，系统负责账号池调度、鉴权、计费、限流、配额、失败切换、用量记录和管理后台。

```text
Vue 3 + Vite 前端
        │  /api、/v1、/setup
        ▼
Go + Gin 后端 ── Ent/SQL ── PostgreSQL
        │
        └──── Redis（缓存、限流、队列、并发与临时状态）
```

生产构建会把前端产物输出到 `backend/internal/web/dist`，再通过 Go 的 `embed` 标签嵌入后端二进制；开发时可以单独运行 Vite，由 Vite 代理 API 到后端。

## 2. 已发现的主要能力

### 网关与上游兼容

- OpenAI 兼容接口：`/v1/chat/completions`、`/v1/responses`、`/v1/embeddings`、`/v1/models`、`/v1/usage`。
- Anthropic 兼容接口：`/v1/messages`、`/v1/messages/count_tokens`。
- Gemini 兼容接口：`/v1beta/models` 及模型动作接口。
- Codex 直连接口：`/backend-api/codex/*`。
- Antigravity 专用接口：`/antigravity/v1/*`、`/antigravity/v1beta/*`。
- 多媒体能力：图片生成/编辑、异步图片任务、图片批处理、视频生成/编辑/扩展、TTS、STT、自定义声音、实时接口、Web Search/X Search。
- 支持流式响应、WebSocket/Realtime、模型列表、模型白名单、模型映射、复合路由和失败切换。

### 账号、分组和调度

- 多平台账号池、OAuth/API Key/Setup Token 等账号类型。
- 分组管理、账号绑定、代理配置、TLS 指纹、模型白名单、复合模型路由。
- 粘性会话、账号级/用户级并发控制、RPM/Token 速率限制、连接池隔离、账号健康度和自动冷却。
- 支持 OpenAI、Anthropic、Gemini、Antigravity、Grok、Kimi、智谱、DeepSeek、MiniMax、OpenCode Go 等平台常量。

### 用户、计费和订阅

- 用户注册/登录、邮箱验证、密码重置、OAuth 登录、OIDC、钉钉、微信、LinuxDo、GitHub/Google 等登录绑定能力（具体开关由配置决定）。
- API Key 创建、分组授权、余额和用量查询。
- Token 级用量记录、成本/利润计算、余额校验、平台配额窗口和订阅计划。
- 兑换码、优惠码、推广/联盟额度、公告、通知邮箱、TOTP 双因素认证和 Passkey。
- 支付提供 EasyPay、支付宝、微信支付、Stripe、Airwallex 等实现，支付回调路由位于 `/api/v1/payment/webhook/*`。

### 管理后台与运维

- 管理员仪表盘、用户/账号/分组/API Key/代理/订阅/支付/兑换码管理。
- 用量报表、实时流量、QPS、延迟、错误趋势、账号可用性和容量统计。
- Channel Monitor V1/V2、探测模板、健康检查和配额监控。
- 审计日志、Prompt Audit、安全合规、内容风控、入站拒绝、请求错误和上游错误处理。
- 插件管理和插件 UI 资源服务；插件开发说明见 `docs/PLUGIN_DEVELOPMENT.md`。
- 备份、数据管理、运行时日志配置、告警规则、WebSocket 实时监控。

## 3. 目录和模块地图

| 路径 | 作用 |
| --- | --- |
| `backend/cmd/server` | Go 服务入口、Wire 依赖注入、版本信息 |
| `backend/internal/config` | YAML/环境变量配置、默认值和校验 |
| `backend/internal/setup` | 首次安装、自动初始化、数据库迁移和管理员创建 |
| `backend/internal/server/routes` | 公共、认证、用户、管理员、支付和网关路由 |
| `backend/internal/handler` | HTTP/WebSocket 处理器，按用户/管理员/网关/支付拆分 |
| `backend/internal/service` | 调度、计费、用量、账号、监控、插件、风控等业务逻辑 |
| `backend/internal/repository` | 数据访问层 |
| `backend/ent/schema` | Ent 数据模型定义 |
| `backend/migrations` | 按文件名顺序执行的 PostgreSQL 正向迁移，当前已有 289 个迁移文件 |
| `backend/resources/model-pricing` | 模型价格和上下文窗口数据 |
| `frontend/src/views` | 登录、用户端和管理后台页面 |
| `frontend/src/api` | 前端 API 客户端和业务接口封装 |
| `frontend/src/features` | Channel Monitor V2、Prompt Audit 等相对独立功能 |
| `deploy` | Docker Compose、环境变量模板、安装脚本和运维说明 |
| `docs` | 支付、批量图片、复合分组、插件和监控等专题文档 |

## 4. 本地环境要求

建议准备：

- Docker 20.10+、Docker Compose v2+（推荐路径）。
- Go **1.27.0**，版本由 `backend/go.mod` 和 CI 明确约束。
- Node.js 20+（仓库 Docker 构建镜像使用 Node 24）。
- pnpm 9（CI 和 Dockerfile 均按 pnpm 9 处理锁文件）。
- PostgreSQL 15+；当前 Compose 默认 PostgreSQL 18。
- Redis 7+；当前 Compose 默认 Redis 8。

安装前端包管理器：

```bash
npm install -g pnpm
```

## 5. 推荐启动方式：Docker 跑完整开发栈

该方式使用当前源码构建镜像，包含 Sub2API、PostgreSQL 和 Redis，适合第一次启动和后端联调。

以下命令默认从仓库根目录执行：

```bash
cd deploy
cp .env.example .env
chmod 600 .env
```

至少修改 `deploy/.env` 中的密码和固定密钥。可以用以下命令生成：

```bash
openssl rand -hex 32   # POSTGRES_PASSWORD、JWT_SECRET、TOTP_ENCRYPTION_KEY 均可分别生成
```

建议配置：

```dotenv
POSTGRES_PASSWORD=替换为随机密码
JWT_SECRET=替换为随机密钥
TOTP_ENCRYPTION_KEY=替换为随机密钥
ADMIN_EMAIL=admin@sub2api.local
ADMIN_PASSWORD=本地管理员密码
SERVER_PORT=8080
SERVER_MODE=debug
```

启动当前源码构建的开发 Compose：

```bash
docker compose -f docker-compose.dev.yml up --build -d
docker compose -f docker-compose.dev.yml ps
docker compose -f docker-compose.dev.yml logs -f sub2api
```

访问：<http://localhost:8080>。

首次启动会自动执行数据库迁移并创建管理员。若未设置 `ADMIN_PASSWORD`，从日志中查找自动生成的密码：

```bash
docker compose -f docker-compose.dev.yml logs sub2api | grep "admin password"
```

常用操作：

```bash
docker compose -f docker-compose.dev.yml restart sub2api
docker compose -f docker-compose.dev.yml up --build -d sub2api
docker compose -f docker-compose.dev.yml down
```

注意：`docker-compose.dev.yml` 默认把代理指向宿主机 `host.docker.internal:7897`，这是为特定网络环境准备的。如果本机没有该代理，需在 Compose 文件中移除或改写 `HTTP_PROXY`、`HTTPS_PROXY`、`ALL_PROXY` 三项，否则上游请求或依赖下载可能失败。

## 6. 日常前端开发：Docker 后端 + Vite 热更新

推荐在完整开发栈启动后，另开终端运行 Vite：

```bash
cd frontend
pnpm install --frozen-lockfile
VITE_DEV_PROXY_TARGET=http://localhost:8080 pnpm dev
```

Vite 默认端口是 `3000`，访问 <http://localhost:3000>。`frontend/vite.config.ts` 会把 `/api`、`/v1`、`/setup` 代理到 `VITE_DEV_PROXY_TARGET`，因此页面修改可以热更新，登录和网关请求仍由 Docker 中的后端处理。

前端改动后无需重建镜像；Go 后端改动则需要重新执行：

```bash
cd deploy
docker compose -f docker-compose.dev.yml up --build -d sub2api
```

## 7. 源码方式运行后端

适合需要调试 Go 代码、断点或单独运行后端的场景。此方式要求你自行提供可被宿主机访问的 PostgreSQL 和 Redis。

### 7.1 编译前端产物

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run build
```

构建结果会写入 `backend/internal/web/dist`。

### 7.2 配置后端

可复制模板：

```bash
cp deploy/config.example.yaml backend/config.yaml
```

然后至少修改 `backend/config.yaml` 中的 `database`、`redis`、`jwt.secret`、`server.mode`。配置加载规则是：`CONFIG_FILE` 指定的文件优先；否则从 `DATA_DIR`、`/app/data`、当前目录、`./config`、`/etc/sub2api` 查找 `config.yaml`；环境变量会按下划线映射覆盖 YAML，例如 `DATABASE_HOST` 对应 `database.host`。

如果希望使用自动初始化，也可以在启动前提供 `AUTO_SETUP=true`、`DATABASE_*`、`REDIS_*`、`ADMIN_*`、`JWT_SECRET` 等环境变量。

### 7.3 启动 Go 服务

```bash
cd backend
go mod download
go run ./cmd/server
```

或编译后运行：

```bash
make build
./backend/bin/server
```

健康检查：

```bash
curl http://localhost:8080/health
```

首次未完成配置时，服务会启动安装向导；启用 `AUTO_SETUP=true` 后会直接走自动初始化。

## 8. 测试、检查和生成

根目录 Makefile：

```bash
make build                    # 后端 + 前端
make test                     # 后端测试 + 前端 lint/typecheck/关键 Vitest
make test-backend
make test-frontend
```

后端：

```bash
cd backend
make test-unit                # go test -tags=unit ./...
make test-integration         # go test -tags=integration ./...
make test                     # 全量 go test + golangci-lint
make generate                 # Ent 和 Wire 代码生成
```

前端：

```bash
cd frontend
pnpm run lint:check
pnpm run typecheck
pnpm run test:run
pnpm run build
```

修改 `backend/ent/schema` 后必须执行 `make -C backend generate`，并检查生成的 `backend/ent` 代码；修改 `frontend/package.json` 后必须同步提交 `frontend/pnpm-lock.yaml`。

## 9. 配置和数据注意事项

- `POSTGRES_PASSWORD` 是 Compose 启动必需项。
- `JWT_SECRET` 不固定会导致重启后所有登录会话失效。
- `TOTP_ENCRYPTION_KEY` 不固定会导致已配置的 2FA 密钥重启后无法解密。
- 配置文件、安装锁和运行数据默认位于 `DATA_DIR`；Docker 中通常是 `/app/data`。
- 数据库迁移在启动阶段自动执行，迁移是正向的；生产升级前应备份 PostgreSQL。
- `RUN_MODE=standard` 启用余额、计费和配额检查；`RUN_MODE=simple` 会关闭 SaaS 计费/余额校验，适合内部自用测试，不要误用于生产计费场景。
- `security.url_allowlist.enabled` 默认关闭以方便开发；生产环境应启用 URL 白名单，并谨慎处理私网地址和 HTTP 上游。
- 通过 Nginx 反向代理时，如需支持粘性会话请求头，`http` 块应配置 `underscores_in_headers on;`。

## 10. 从哪里开始读代码

建议按以下顺序熟悉项目：

1. `backend/cmd/server/main.go`：启动、首次安装和主服务生命周期。
2. `backend/internal/config/config.go`：配置来源、默认值、环境变量映射和校验。
3. `backend/internal/server/routes/*.go`：查看 API 分组和鉴权边界。
4. `backend/internal/handler/gateway*`、`backend/internal/service/gateway*`：网关请求解析、调度、转发和计费。
5. `backend/internal/service/account*`、`channel*`、`group*`：账号池、渠道和分组能力。
6. `frontend/src/router/index.ts`、`frontend/src/views`、`frontend/src/api`：前端页面、路由和接口调用。
7. `backend/migrations`、`backend/ent/schema`：数据模型和迁移演进。

## 11. 当前工作区提示

扫描时发现工作区已有用户改动：`README.md`、`README_JA.md` 被删除，`README_CN.md` 被修改。本文未覆盖或回滚这些变更；继续开发前请确认它们是否为有意改动。

## 12. 最短启动清单

```bash
cd deploy
cp .env.example .env
# 编辑 .env：至少设置 POSTGRES_PASSWORD，建议设置 JWT_SECRET/TOTP_ENCRYPTION_KEY/ADMIN_PASSWORD
docker compose -f docker-compose.dev.yml up --build -d
docker compose -f docker-compose.dev.yml logs -f sub2api
```

然后打开 <http://localhost:8080>；如果要前端热更新，再在另一个终端执行：

```bash
cd frontend
pnpm install --frozen-lockfile
VITE_DEV_PROXY_TARGET=http://localhost:8080 pnpm dev
```
