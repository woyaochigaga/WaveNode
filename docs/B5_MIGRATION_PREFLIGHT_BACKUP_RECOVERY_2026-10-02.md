# B5 迁移 preflight、备份校验和恢复演练实施记录

日期：2026-10-02  
状态：代码实现完成，真实环境恢复演练需上线前执行  
范围：PostgreSQL 迁移前检查、S3 备份完整性、临时库恢复验证、管理端操作入口

## 1. 这次解决什么问题

现有项目已经有备份服务、分卷 SHA-256、恢复状态和迁移 checksum，但此前仍有三个实际风险：

- 管理员无法在页面上一次看清“数据库是否可连、迁移是否一致、磁盘/Redis/密钥是否准备好”。
- 普通单文件备份没有整包 checksum 和数据库内容摘要，下载损坏或恢复错版本时缺少证据。
- 恢复操作直接写入生产库，没有先证明这份备份能在隔离数据库中恢复并通过关键字段 smoke test。

本次没有重建备份格式，也没有引入新的数据库迁移，而是在现有记录 JSON 和 `PgDumper` 能力上追加安全证据。

## 2. 迁移 preflight

管理端新增：

```text
GET /api/v1/admin/backups/preflight
```

它只读检查并返回 operation ID，包含：

- PostgreSQL 连接、版本、数据库大小。
- `schema_migrations` 当前版本和嵌入代码中的目标迁移文件。
- `pg_dump`、`psql` 是否存在。
- Redis PING。
- 是否配置跨重启有效的固定加密密钥。
- 临时目录可用空间（按数据库大小的 2 倍估算，至少 1 GiB）。
- `batch_image_jobs` 未结算数量和 `payment_orders` 待处理数量。
- 最近一次通过恢复演练的备份以及距今时间。

检查项分为 `pass`、`warn`、`fail`，只有阻断项为 `fail` 时报告 `ready=false`。未结算任务和备份过旧是提醒，不会伪装成“系统完全安全”。

## 3. 备份完整性清单

每次新备份记录增加：

- `operation_id`：创建操作的稳定 ID。
- `manifest.sha256`、`manifest.size_bytes`：压缩备份文件的整包 SHA-256 和大小。
- 数据库版本、当前/目标迁移文件、数据库大小。
- 关键表行数：`users`、`accounts`、`api_keys`、`groups`、`channels`、`payment_orders`、`redeem_codes`、`security_secrets`。

单文件上传和每个分卷上传都比较对象存储返回的实际大小。大小不一致时，删除已上传对象并把备份标记为失败，避免保存一条看似成功但内容不完整的记录。

旧备份记录没有 manifest 时仍可展示和下载，保持滚动升级兼容；但在支持恢复演练的生产 PostgreSQL 驱动上，旧记录需要重新生成或补充可验证证据后才能恢复。

## 4. 恢复前完整性检查

正式恢复和恢复演练都先把对象下载到临时文件：

1. 单文件校验文件大小；分卷逐卷校验大小和分卷 SHA-256。
2. 有整包 manifest 时再校验整包 SHA-256。
3. 解压 gzip 到黑洞，确认压缩流完整，没有截断。
4. 所有校验通过后才交给 `psql`。

这意味着 S3 超时、分卷缺失、对象被替换或 gzip 损坏都会在写入数据库前失败。

## 5. 临时库恢复演练

管理端新增：

```text
POST /api/v1/admin/backups/:id/verify
```

该操作需要现有 step-up 权限。它会：

1. 取得备份记录和对象存储内容。
2. 创建 `sub2api_verify_<random>` 临时数据库。
3. 使用同一份压缩备份恢复到临时库。
4. 检查关键表是否可读、迁移版本是否与 manifest 一致、账号 `credentials` 是否为有效 JSON、安全密钥是否存在空 key/value。
5. 比较关键表行数；恢复期间如果源库继续写入，差异记录为 warning，不把正常并发写入误判为损坏。
6. 记录验证报告和 `verification_operation_id`，最后终止连接并删除临时库。

只有 `verification_status=passed` 的备份允许正式恢复；演练中、失败或从未演练的备份，管理端恢复按钮禁用，后端也会再次拒绝。

## 6. 兼容与上线要求

- 不新增 PostgreSQL 表，不改变已有 `.sql.gz` 或分卷对象布局。
- 备份记录 JSON 只追加字段；旧记录可读，但缺少新证据时不满足生产恢复门槛。
- Redis 不作为恢复证据来源；它只用于既有跨实例记录锁和 preflight PING。
- 服务账号必须能在同一 PostgreSQL 集群执行 `CREATE DATABASE`、连接临时库、执行 `DROP DATABASE`，并安装 `pg_dump`/`psql`。
- 上线前必须在与生产权限接近的环境完整点击一次“恢复演练”，确认创建临时库、恢复、检查、清理都成功。仅通过单元测试不能替代这一步。
- 真实环境不允许把生产库名写成临时库名；代码使用随机库名并在 finally 清理。

## 7. 主要源码落点

- preflight、manifest、磁盘检查：`backend/internal/service/backup_preflight.go`
- 校验、演练任务和恢复门槛：`backend/internal/service/backup_verification.go`
- 备份/恢复流程：`backend/internal/service/backup_service.go`、`backend/internal/service/backup_restore_state.go`
- PostgreSQL 只读摘要与临时库：`backend/internal/repository/backup_database_inspector.go`
- `pg_dump`/`psql` 目标库执行：`backend/internal/repository/backup_pg_dumper.go`
- Redis 健康检查：`backend/internal/repository/leader_lock_cache.go`
- 管理 API：`backend/internal/handler/admin/backup_handler.go`、`backend/internal/server/routes/admin.go`
- 管理端展示与轮询：`frontend/src/views/admin/BackupView.vue`

## 8. 验证结果

已通过：

```text
go test ./internal/service ./internal/repository ./internal/handler/admin ./internal/server/middleware ./internal/server/routes
go test -tags unit ./internal/service ./internal/repository ./internal/handler/admin ./internal/server/middleware ./internal/server/routes
pnpm typecheck
pnpm exec vitest run src/views/admin/__tests__/BackupView.spec.ts src/i18n/__tests__/localeKeyCompleteness.spec.ts
git diff --check
```

前端备份页测试覆盖：preflight 状态展示、未演练时阻止恢复、恢复演练入口、分卷下载、旧单文件下载、备份轮询、归档保留和删除确认。

后端既有备份恢复/分卷/锁/迁移测试全部通过；部署环境的真实临时库演练属于发布前运维验收，不能在没有 PostgreSQL 实例和 `CREATEDB` 权限的开发机上虚报为已完成。
