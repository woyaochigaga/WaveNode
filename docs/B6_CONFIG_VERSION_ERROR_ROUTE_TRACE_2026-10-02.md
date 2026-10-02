# B6 配置版本、错误码和 Route Trace 最小版实施记录

日期：2026-10-02  
状态：代码实现完成  
范围：动态配置 CAS、请求配置/价格版本、统一重试语义、管理员路由与计费追踪

## 1. 这次解决什么问题

过去遇到失败请求时，管理员能看到状态码和上游错误，但难以回答三件事：

- 请求使用的是哪一版运行时设置和价格数据。
- 网关尝试了哪些账号、为什么切换、最终是否建议客户端重试。
- 失败请求是否已经写入用量记录并产生费用。

B6 在现有设置表、Ops 错误事件和用量表上补齐关联元数据。第一版不保存原始 Prompt、完整响应、Token、Cookie、代理密码或账号名称。

## 2. 动态配置版本与 CAS

`GET /api/v1/admin/settings` 和保存成功后的设置响应新增：

```json
{
  "config_version": "0d3fa449d11a2c1d",
  "config_updated_at": "2026-10-02T10:00:00Z",
  "config_updated_by": 42,
  "config_effective_at": "2026-10-02T10:00:00Z"
}
```

管理端加载页面时保存 `config_version`，提交时原样带回。后端在同一 PostgreSQL 事务中锁定内部版本行，重新计算设置内容指纹，再写入设置和版本元数据：

- 版本一致：提交设置，生成新内容指纹，记录操作者和立即生效时间。
- 版本不一致：返回 HTTP 409 和 `RUNTIME_SETTINGS_VERSION_CONFLICT`，前端重新加载，避免两个管理员相互覆盖。
- 旧客户端不传 `config_version`：继续允许保存，保持兼容；新管理端默认启用冲突保护。

迁移 `241_runtime_settings_version.sql` 只在现有 `settings` 表中幂等创建一条内部元数据记录，不新增业务表。该内部 key 不参与内容哈希，因此版本不会递归改变自身。

每个实例最多缓存设置版本 30 秒。当前写入实例会立即失效本地缓存；其他实例即使错过通知，也会在缓存过期后从数据库读取真实内容指纹并收敛。

## 3. 请求配置和价格版本

网关请求进入时会把以下只读指纹固化到请求上下文：

- `config_version`：当前设置内容指纹。
- `price_version`：主价格目录的 `model_pricing.sha256`，或价格 JSON、回退文件和覆盖文件的联合指纹。

指纹不包含文件绝对路径，部署目录不同的实例仍可比较；读取结果缓存 30 秒。找不到价格文件时明确记录 `fallback`，旧事件没有该字段时显示 `not_captured`，不会拿当前版本冒充历史版本。

## 4. 统一错误和重试语义

内部标准错误响应新增：

```json
{
  "code": 429,
  "reason": "RATE_LIMITED",
  "message": "request was rate limited",
  "retryable": true,
  "retry_after": 1,
  "metadata": {
    "request_id": "..."
  }
}
```

`408/425/429/500/502/503/504` 默认可重试，业务错误可以显式覆盖；有等待时间时同时返回标准 `Retry-After` 响应头。

OpenAI、Anthropic、Gemini 网关已有 JSON 结构保持不变，避免破坏 SDK。网关通过协议无关响应头补充最小诊断信息：

- `X-Client-Request-ID`
- `X-Sub2API-Error-Code`
- `X-Sub2API-Retryable`
- `Retry-After`，仅在已知等待时间时返回；429 没有明确值时使用 1 秒保守默认值

安全错误码只描述错误类别，例如 `request_invalid`、`auth_invalid`、`permission_denied`、`model_unavailable`、`rate_limited`、`upstream_timeout`、`upstream_overloaded` 和 `internal_error`，不包含上游正文。

## 5. Route Trace 管理接口

新增管理员接口：

```text
GET /api/v1/admin/ops/requests/:request_id/route-trace
```

接口位于现有管理员认证路由组内，并继续受 Ops 监控开关约束。查询范围限制为最近 30 天，支持服务端 `request_id` 和客户端 `client_request_id`。示例响应：

```json
{
  "request_id": "req-123",
  "client_request_id": "client-123",
  "requested_model": "gpt-public",
  "final_model": "gpt-upstream",
  "inbound_endpoint": "/v1/chat/completions",
  "upstream_endpoint": "/v1/responses",
  "config_version": "0d3fa449d11a2c1d",
  "price_version": "d7a912dba832891c",
  "candidate_count": 2,
  "retry_count": 1,
  "final_account_ref": "acct_95f9cc57a57e",
  "final_error_code": "rate_limited",
  "final_error_message": "request was rate limited",
  "retryable": true,
  "retry_after": 1,
  "billing_status": "not_charged",
  "billed_amount": 0,
  "usage_record_count": 0,
  "attempts": [
    {
      "account_ref": "acct_2ab31de72809",
      "platform": "openai",
      "status_code": 429,
      "reason": "rate_limited",
      "config_version": "0d3fa449d11a2c1d",
      "price_version": "d7a912dba832891c"
    }
  ]
}
```

账号 ID 使用稳定匿名摘要，便于同一次请求内关联但不暴露账号名称。尝试原因会归一化成有限原因码，不透传原始 `reason`；上游 `message`、`detail`、URL、响应正文和请求正文均不进入响应。

计费状态通过 `request_id/client_request_id` 查询现有 `usage_logs`：

- `charged`：存在正金额用量记录。
- `recorded_zero_cost`：存在用量记录，但金额为 0。
- `not_charged`：没有用量记录。
- `billing_blocked`：错误发生在计费阶段且没有后续落账。
- `lookup_failed`：计费查询失败，Route Trace 主体仍可查看。

## 6. 管理端交互

Ops 错误详情弹窗增加“路由与计费追踪”区域。它异步加载，不阻塞原错误详情，展示：

- 候选尝试数、重试次数、计费状态和客户端是否可重试。
- 模型映射、端点路径、配置版本、价格版本和匿名最终账号。
- 有顺序的上游尝试时间线。
- 加载中、无记录、无权限、失败和重新加载状态。

旧请求没有版本字段时显示 `not_captured`；没有 request ID 时显示不可用提示。权限错误只显示安全提示，不回显后端错误详情。

## 7. 保留、灰度和回退

- Route Trace 不新增存储，复用 `ops_error_logs` 和 `usage_logs`；接口只查最近 30 天，实际清理由现有 Ops/用量保留策略负责。
- 新版本元数据只占 `settings` 中一行，无额外清理任务。
- 关闭 Ops 监控后 Route Trace 与现有 Ops 查询一起不可用；错误详情主体仍正常显示。
- 回滚到旧前端时，新增响应字段会被忽略；回滚到旧后端时，新前端把 Route Trace 显示为不可用或加载失败。
- 如需停用 CAS，可由旧客户端省略 `config_version` 走兼容路径；不建议长期这么做，因为会失去并发覆盖保护。
- 价格文件和设置内容仍是运行时事实来源，版本元数据只用于关联和诊断，不参与请求路由决策。

## 8. 主要源码落点

- 配置版本与 CAS：`backend/internal/service/setting_version.go`、`backend/internal/repository/setting_repo.go`
- 设置接口和管理端冲突处理：`backend/internal/handler/admin/setting_handler*.go`、`frontend/src/views/admin/SettingsView.vue`
- 请求版本中间件：`backend/internal/server/middleware/runtime_versions.go`
- 错误重试契约：`backend/internal/pkg/errors/`、`backend/internal/pkg/response/response.go`
- 网关安全错误响应头：`backend/internal/server/middleware/client_request_id.go`
- Route Trace 聚合与计费查询：`backend/internal/service/route_trace.go`、`backend/internal/repository/ops_repo_route_trace.go`
- 管理接口：`backend/internal/handler/admin/ops_handler.go`、`backend/internal/server/routes/admin.go`
- 管理端展示：`frontend/src/views/admin/ops/components/OpsErrorDetailModal.vue`
- 数据迁移：`backend/migrations/241_runtime_settings_version.sql`

## 9. 验证范围

自动化测试覆盖：

- 配置内容指纹稳定性、忽略元数据行、陈旧版本冲突、成功写入操作者和生效时间。
- PostgreSQL 版本行锁、事务 CAS 和迁移幂等结构。
- 设置/价格版本请求上下文和跨部署路径稳定性。
- 默认及显式重试策略、`Retry-After`、request ID 和网关协议无关响应头。
- Route Trace 脱敏、有限原因码、匿名账号、计费成功/失败降级。
- 管理端 Route Trace 正常展示和 403 隐私状态。

上线后建议用一条可控的 429 请求做 smoke test：记录响应中的 request ID，在 Ops 错误详情打开 Route Trace，确认配置版本、尝试时间线和“未产生费用”三项与实际一致。
