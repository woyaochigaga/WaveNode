# Sub2API 项目启动与 Docker Compose 手工发布指南

> 适用范围：本项目本地开发、单台服务器部署和每周手工迭代。命令默认在项目根目录执行。

## 一、先理解项目由什么组成

本项目主要包含三部分：

- `frontend`：Vue/Vite 前端源码，负责页面、交互和前端接口调用。
- `backend`：Go 后端源码，负责 API、认证和业务逻辑。
- PostgreSQL、Redis：运行时依赖。PostgreSQL 保存正式数据，Redis 保存缓存或运行状态。

项目默认 Docker 构建会先打包前端，再编译 Go 后端，并把前端资源嵌入应用。因此，默认生产发布不是单独替换 `build` 文件夹，而是重新构建应用镜像。

## 二、本地启动前的准备

### 1. 安装必要软件

需要安装：

- Git：下载和管理代码。
- Node.js：运行前端工具。建议使用项目要求的 LTS 版本。
- pnpm：安装前端依赖。可执行 `corepack enable` 后使用项目指定的 pnpm。
- Go：只有需要直接编译或调试后端时才必须安装。
- Docker Desktop：推荐安装，用于启动 PostgreSQL、Redis 和完整应用。

检查安装结果：

```bash
git --version
node --version
pnpm --version
go version
docker --version
docker compose version
```

命令的作用是打印版本号。任意命令找不到，都先安装对应软件，不要继续排查项目代码。

### 2. 获取代码

```bash
git clone <项目仓库地址> sub2api
cd sub2api
```

`git clone` 下载源码，`cd` 进入项目根目录。已经有源码时只执行 `cd`。

### 3. 准备环境变量

```bash
cp .env.example .env
```

如果仓库没有 `.env.example`，就复制项目提供的环境模板或参考 Compose 文件中的变量。`.env` 只放本机配置和密钥，不要提交到 Git。

至少确认数据库地址、Redis 地址、应用端口、管理员初始化配置和密钥。生产环境的密钥不要直接沿用本地值。

## 三、本地推荐启动方式：Docker Compose



### 1. 启动依赖服务

先查看 Compose 服务名：

```bash
docker compose config --services
```

该命令只解析配置，不启动服务。确认配置没有报错后启动：

```bash
docker compose up -d postgres redis
```

`-d` 表示后台运行；PostgreSQL 和 Redis 的容器启动，但数据会保存在 Compose 配置声明的持久化卷中。

### 2. 查看服务状态和日志

```bash
docker compose ps
docker compose logs -f postgres
docker compose logs -f redis
```

`ps` 查看容器是否为 `Up`；`logs -f` 持续查看日志，按 `Ctrl+C` 退出日志查看，不会停止容器。

### 3. 启动完整应用

如果 Compose 文件提供完整应用服务，执行：

```bash
docker compose up -d
docker compose ps
```

然后访问 Compose 文件中映射的应用端口，例如 `http://localhost:<端口>`。实际端口以项目 Compose 配置为准，不要盲目使用示例端口。

### 4. 本地修改前端并验证

进入前端目录安装依赖并启动开发服务器：

```bash
cd frontend
pnpm install
pnpm run dev
```

`pnpm install` 安装依赖；`pnpm run dev` 启动 Vite 热更新服务器。终端会显示访问地址。改动页面后，浏览器通常会自动刷新。

检查前端代码和类型：

```bash
pnpm run check:i18n
pnpm exec vue-tsc -b
```

构建生产前端：

```bash
pnpm run build
```

本项目 Vite 产物目录是 `backend/internal/web/dist`，不是根目录下固定的 `build`。默认 Docker 构建会继续把该产物编译进 Go 应用。

### 5. 停止本地环境

```bash
docker compose stop
```

只停止容器，数据卷仍然保留。下次可以再次 `docker compose start`。

```bash
docker compose down
```

删除容器和网络，但通常保留数据卷。**不要执行** `docker compose down -v`**，除非你明确要删除本地数据库和 Redis 数据。**

## 四、Docker Compose 手工发布到服务器



### 1. 服务器准备

服务器安装 Docker 和 Compose 插件，并准备一个部署目录，例如：

```bash
sudo mkdir -p /opt/sub2api
sudo chown "$USER":"$USER" /opt/sub2api
cd /opt/sub2api
```

服务器只保存 Compose 文件、生产环境 `.env`、备份和部署脚本；密钥不要写进镜像。

### 2. 第一次部署

把生产用的 `compose.yml` 和 `.env` 上传到服务器：

```bash
scp compose.yml user@服务器IP:/opt/sub2api/
scp .env.production user@服务器IP:/opt/sub2api/.env
```

服务器上检查并启动：

```bash
cd /opt/sub2api
docker compose config
docker compose pull
docker compose up -d
docker compose ps
```

`config` 检查最终配置；`pull` 下载镜像；`up -d` 按配置创建或更新容器；`ps` 验证容器状态。

### 3. 本地构建新版本镜像

每次迭代先在本地提交代码并确认测试通过，然后构建带版本号的镜像。示例版本为 `v1.0.1`：

```bash
docker build -t your-registry/sub2api:v1.0.1 .
docker push your-registry/sub2api:v1.0.1
```

`docker build` 执行前端打包和 Go 编译；`docker push` 把镜像上传到镜像仓库。仓库地址按你的 Docker Hub、GHCR 或私有仓库调整。

Compose 文件中的应用镜像也要改成明确版本：

```yaml
services:
  app:
    image: your-registry/sub2api:v1.0.1
```

不要只使用 `latest`，否则无法判断线上到底运行哪一版，也不方便回滚。

### 4. 发布前备份 PostgreSQL

先确认数据库容器或数据库服务名：

```bash
docker compose ps
```

使用 PostgreSQL 客户端导出备份，变量名以实际 Compose 配置为准：

```bash
mkdir -p backups
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' > "backups/db-$(date +%Y%m%d-%H%M%S).sql"
```

备份完成后检查文件是否存在、大小是否合理，并定期把备份复制到服务器以外的位置。Redis 是否需要持久化和备份，要根据项目实际用途确认；不要未经确认直接清空 Redis。

### 5. 更新应用容器

```bash
cd /opt/sub2api
docker compose pull app
docker compose up -d app
docker compose ps
docker compose logs --tail=200 app
```

这几条命令只拉取并重建应用服务。PostgreSQL 和 Redis 容器、数据卷不会被删除。前端或后端只要有任意一方改动，都使用这套流程，因为当前项目默认把两者打包在同一个应用镜像中。

### 6. 发布后验证

依次检查：

1. 容器状态是否为 `Up` 或健康状态正常。
2. 首页能否打开，静态资源是否加载成功。
3. 登录、核心页面和核心 API 是否正常。
4. 应用日志是否出现数据库连接、Redis 连接或迁移错误。
5. PostgreSQL 和 Redis 容器是否仍然运行。

如果应用有健康检查地址，可以执行：

```bash
curl -f http://127.0.0.1:<应用端口>/健康检查路径
```



### 7. 发布失败时回滚

把 Compose 中的镜像版本改回上一个可用版本，例如 `v1.0.0`，然后执行：

```bash
docker compose pull app
docker compose up -d app
docker compose logs --tail=200 app
```

如果失败原因是数据库结构已经改变，不能只盲目回滚应用镜像；需要先确认迁移是否兼容，必要时使用发布前备份恢复数据库。

## 五、前端和后端分别修改时怎么做



### 只修改前端

```text
修改 frontend
→ 本地 pnpm run build / 本地功能验证
→ 构建新应用镜像
→ 上传新镜像
→ 备份数据库
→ 只更新 app 容器
→ 验证页面和接口
```

数据库和 Redis 不需要重建。

### 只修改后端

```text
修改 backend
→ 本地运行或构建验证
→ 构建新应用镜像
→ 检查是否有数据库迁移
→ 有迁移则先备份并按项目规则执行
→ 更新 app 容器
→ 验证 API 和页面
```



### 前后端都修改

使用同一个版本镜像一起发布，避免前端调用了线上还不存在的新 API。

## 六、最容易出错的地方

- 不要在生产服务器执行 `docker compose down -v`，它可能删除数据库和 Redis 数据卷。
- 不要把生产 `.env` 提交到 Git，也不要把数据库密码写入 Dockerfile。
- 不要直接覆盖正在运行容器里的文件；应构建新镜像并替换容器。
- 不要只看容器启动成功就认为发布成功，要实际验证登录和核心功能。
- 不要在没有备份、没有回滚版本的情况下做数据库结构变更。
- 前端构建成功不代表后端 API 一定兼容，涉及接口字段时要联调。



## 七、每周迭代的最小流程

```text
1. 本地修改代码
2. 本地启动并测试
3. 提交 Git，写清本次改动
4. 生成新版本号，例如 v1.0.1
5. 构建并推送新镜像
6. 服务器备份数据库
7. 修改 Compose 镜像版本
8. docker compose pull app
9. docker compose up -d app
10. 查看日志并验证线上功能
11. 出问题则切回上一版本
```

这套流程是手工发布，但已经具备企业发布的核心边界：版本可追踪、数据独立、应用可回滚、发布后可验证。后续接入 GitHub Actions 时，只需自动执行构建、测试、推送和部署，不需要改变数据库和 Redis 的存储方式。

## 八、根据 DEV_GUIDE.md 配置项目

项目的 Docker 配置位于 `deploy` 目录。实际启动时应先进入该目录：

```bash
cd /Users/chenjiantao/Documents/project/WaveNode/sub2api/deploy
```

项目提供了两种主要 Compose 文件：


| 文件                         | 数据存储方式                                        | 适用场景          |
| -------------------------- | --------------------------------------------- | ------------- |
| `docker-compose.yml`       | Docker named volume                           | 快速启动，迁移和备份不直观 |
| `docker-compose.local.yml` | 当前目录下的 `data/`、`postgres_data/`、`redis_data/` | 推荐，便于备份、迁移和排查 |


建议本地和单服务器部署优先使用 `docker-compose.local.yml`。

### 1. 复制环境变量模板

```bash
cd /Users/chenjiantao/Documents/project/WaveNode/sub2api/deploy
cp .env.example .env
chmod 600 .env
```

`.env.example` 是变量模板，`.env` 是当前机器真正读取的配置。`chmod 600` 让只有当前用户可以读写密钥文件。

### 2. 必须配置的变量

打开 `.env`：

```bash
nano .env
```

至少检查并修改以下变量：

```dotenv
# 应用访问端口
BIND_HOST=0.0.0.0
SERVER_PORT=8080

# 数据库初始化账号；生产环境必须修改密码
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=004126
POSTGRES_DB=sub2api

# 首次登录管理员
ADMIN_EMAIL=woyaochigaga@gmail.com
ADMIN_PASSWORD=004126

# 必须固定，不能每次启动随机生成，否则重启后登录会失效
JWT_SECRET=openssl_rand_hex_32_生成的值

# 如果启用 2FA，也必须固定，否则已保存的 2FA 配置会失效
TOTP_ENCRYPTION_KEY=openssl_rand_hex_32_生成的值

# 时区
TZ=Asia/Shanghai

# 完整 SaaS 模式；内部自用才考虑 simple
RUN_MODE=standard
```

生成两个安全密钥：

```bash
openssl rand -hex 32
```

每执行一次会输出一个随机值，分别填入 `JWT_SECRET` 和 `TOTP_ENCRYPTION_KEY`。不要把真实值发到聊天、提交到 Git 或写进 Dockerfile。

### 3. PostgreSQL 配置说明

Compose 内部应用连接数据库时使用服务名 `postgres`，不是 `localhost`：

```text
应用容器 → postgres:5432
```

常用变量：


| 变量                        | 作用         | 初学者建议        |
| ------------------------- | ---------- | ------------ |
| `POSTGRES_USER`           | 数据库用户      | 保持 `sub2api` |
| `POSTGRES_PASSWORD`       | 数据库密码      | 必须改成随机强密码    |
| `POSTGRES_DB`             | 数据库名       | 保持 `sub2api` |
| `DATABASE_MAX_OPEN_CONNS` | 应用最大数据库连接数 | 小服务器先使用模板默认值 |
| `DATABASE_MAX_IDLE_CONNS` | 应用空闲连接数    | 不要随意调大       |


不要把 `DATABASE_HOST` 改成 `127.0.0.1` 或 `localhost`，Compose 环境下应用应通过 `postgres` 服务名连接。

### 4. Redis 配置说明

应用通过 Compose 服务名连接 Redis：

```text
应用容器 → redis:6379
```

常用变量：

```dotenv
REDIS_PORT=6379
REDIS_USERNAME=
REDIS_PASSWORD=
REDIS_DB=0
REDIS_ENABLE_TLS=false
```

本地开发可以不设置 Redis 密码。生产环境如果需要密码，应同时在 Compose 的 Redis 服务和应用变量中配置，不能只改一边。

### 5. 首次启动前创建持久化目录

使用本地目录版 Compose 时执行：

```bash
mkdir -p data postgres_data redis_data
```

三个目录分别用于应用数据、PostgreSQL 数据和 Redis 数据。不要删除它们；删除 `postgres_data` 会丢失数据库，删除 `data` 可能丢失应用生成的配置和日志。

### 6. 检查配置再启动

```bash
docker compose -f docker-compose.local.yml config
docker compose -f docker-compose.local.yml config --services
```

预期服务包括：

```text
postgres
redis
sub2api
```

如果提示 `POSTGRES_PASSWORD is required`，说明 `.env` 没有被读取或密码为空。先确认当前目录、文件名和权限：

```bash
pwd
ls -la .env docker-compose.local.yml
```



### 7. 启动和首次初始化

```bash
docker compose -f docker-compose.local.yml up -d
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs --tail=200 sub2api
```

Compose 会启动 PostgreSQL 和 Redis，应用使用 `AUTO_SETUP=true` 自动连接依赖、执行数据库迁移、生成配置并创建管理员账号。数据库迁移按文件名顺序执行，已执行记录保存在 `schema_migrations` 表中。

如果没有设置 `ADMIN_PASSWORD`，系统可能在首次启动日志中生成密码，可以查看：

```bash
docker compose -f docker-compose.local.yml logs sub2api | grep -i "admin password"
```

首次启动后访问：

```text
http://localhost:8080
```

如果修改了 `SERVER_PORT`，访问地址中的端口也要跟着修改。

### 8. 哪些配置改了需要重建或重启


| 修改内容                      | 操作                                  |
| ------------------------- | ----------------------------------- |
| `.env` 中的端口、密钥、数据库密码、运行模式 | `docker compose ... up -d` 重新创建应用容器 |
| 只改前端或后端源码                 | 构建新镜像，再更新 `sub2api`                 |
| `config.yaml`             | 确认挂载后重启 `sub2api`                   |
| 数据库表结构                    | 先备份，再发布包含迁移的新版本                     |
| PostgreSQL/Redis 数据       | 不要删除目录或数据卷                          |




### 9. DEV_GUIDE.md 中的重要开发要求

- 前端包管理器必须使用 `pnpm`，不要混用 npm 生成的 `node_modules`。
- 前端依赖安装使用 `cd frontend && pnpm install`。
- 后端单元测试：`cd backend && go test -tags=unit ./...`。
- 后端集成测试：`cd backend && go test -tags=integration ./...`。
- 修改 Ent Schema 后，在 `backend` 目录执行 `go generate ./ent`，并提交生成文件。
- CI 使用 Go 1.27.0；本机 Go 版本与项目 `backend/go.mod` 要求不一致时，以项目要求为准。
- 数据库迁移是向前执行的；已经执行的迁移不会自动回滚。需要回退时依赖数据库备份或人工补偿 SQL。



### 10. 推荐的当前本地启动命令

```bash
cd /Users/chenjiantao/Documents/project/WaveNode/sub2api/deploy
cp .env.example .env                         # 只在第一次执行
chmod 600 .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml config
docker compose -f docker-compose.local.yml up -d
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs --tail=200 sub2api
```

如果你使用的是项目当前目录下的 `docker-compose.yml`，可以把上面每条命令中的 `-f docker-compose.local.yml` 去掉；但推荐保留 local 版本，因为它的目录数据更容易备份和迁移。

# 本地开发



## 终端 1：后端

cd backend && go run ./cmd/server

## 终端 2：前端

cd frontend && pnpm run dev