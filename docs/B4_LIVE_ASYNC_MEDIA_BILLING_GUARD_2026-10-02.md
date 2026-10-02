# B4 Live 与异步媒体最小成本护栏实施记录

日期：2026-10-02  
状态：已完成  
范围：Live 会话、Grok/Seedance 异步视频、既有批量图片结算链路

## 1. 目标

B4 是完整不可变账本之前的风险保护层，解决两个直接问题：

1. Live 不能再以零成本无限创建长会话。
2. 异步媒体的创建、轮询、终态和进程恢复必须使用同一个任务级账务身份，不能重复冻结或遗留预留。

本阶段不重写现有账单系统，也不新增数据库表，优先复用余额缓存、`usage_billing_dedup`、使用记录和现有媒体任务快照。

## 2. Live 保护链路

### 2.1 创建前估算与冻结

- 新增 `gateway.live.billing_guard_mode`，支持 `disabled`、`observe`、`enforce`，默认 `enforce`。
- 估算上限为“最长会话分钟数 × Realtime 每分钟价格 × 用户分组倍率”。
- 分组未配置 Realtime 价格时使用系统内置默认价 `$0.05/分钟`；只有显式配置 `0` 才视为免费。
- 余额模式在请求上游前严格冻结最大成本；首次请求余额不足或 Redis 不可用都会拒绝，不允许先创建后发现无法计费。
- 订阅模式不冻结余额，但仍记录最大成本、最长时长和完整计费快照，并由统一账单扣减订阅用量。

### 2.2 不可变计费快照

Live 映射新增并持久化以下快照：保护状态、预留 ID/金额、分组倍率、账号倍率、Realtime 单价、API Key 配额/限速、账号类型/配额和平台。会话结束时只使用创建时快照，避免管理员中途修改价格后产生难以解释的账单。

响应和结构化日志同时记录 `billing_guard_status` 与 `estimated_max_cost`，便于判断请求处于硬限制、仅观察还是免费状态。

### 2.3 统一结束与恢复

- 正常结束、客户端取消、WS 断开、最长时长到期和进程恢复统一调用同一个 finalize。
- 实际费用按会话持续时间计算，并限制在创建时冻结上限内。
- call hash 作为稳定 `request_id` 写入 `usage_billing_dedup`；多个实例或重复 finalize 只会应用一次账单。
- 账单仓储报错或返回空结果时不标记会话完成，在恢复窗口内继续重试。
- Redis 使用有序集合维护未关闭会话索引。进程启动会扫描全部索引记录；Redis 短时不可用时最多重试 6 次，退避上限 30 秒。
- 结算完成后释放余额预留与并发租约，并写入 Realtime 使用记录。

## 3. 异步媒体保护链路

### 3.1 Grok 与 Seedance 视频

- 创建任务前按媒体参数估算最大成本，生成 `async-media:<uuid>` 任务级预留 ID。
- 创建成功后，把预留 ID 和金额写入待结算任务快照；查询状态不会再次预留。
- 成功终态先执行原有持久化账单，账单落地后释放预留。
- `failed`、`cancelled`、`canceled`、`expired` 和删除任务会按快照幂等释放预留。
- 账单暂时失败时只释放结算抢占权，不释放余额预留，后续轮询仍可重试结算。

### 3.2 批量图片

批量图片原本已具备数据库级 hold/capture/release、稳定幂等键和 Worker 恢复测试。本次没有复制一套新实现，只复核并沿用该链路。

## 4. 配置与上线

示例：

```yaml
gateway:
  live:
    max_session_duration_seconds: 3600
    billing_guard_mode: "enforce"
```

模式说明：

| 模式 | 行为 | 建议用途 |
| --- | --- | --- |
| `enforce` | 创建前校验并冻结，结束后实际结算 | 默认生产模式 |
| `observe` | 记录理论最大成本和理论使用成本，不扣款 | 上线前核对价格与数据 |
| `disabled` | 兼容旧路径，不启用新计费保护 | 仅紧急回退；不得长期作为零价入口 |

分组级灰度继续使用管理端已有的 Live 开关和 Realtime 价格配置。需要明确免费时必须填写 `0`，留空不是免费。

## 5. 兼容与数据变更

- 无需 PostgreSQL 迁移。
- Redis Live hash 增加计费快照字段，并新增 `live:call:recovery` 恢复索引。
- 异步视频待结算 JSON 增加预留 ID 和金额，旧 JSON 反序列化时字段为空，保持兼容。
- 滚动发布期间，旧 Live 记录没有保护状态时继续按旧的零费用记录路径关闭，避免已有会话卡死；所有新记录默认进入 `enforce`。

## 6. 故障语义

| 场景 | 结果 |
| --- | --- |
| 余额不足 | 创建前拒绝，不调用上游，不留下预留 |
| Redis 无法完成严格预留 | 返回计费服务不可用，不创建收费任务 |
| 上游创建失败或 429 | 释放本次预留，保留原有账号切换/错误处理 |
| 客户端断开 | Live 观察器接管；终态或超时后统一结算 |
| 账单数据库短时失败 | 保留会话/任务快照和预留，后续重试 |
| 重复轮询或重复 finalize | 稳定请求号去重，不重复扣费或同步缓存 |
| 进程重启 | Live 从恢复索引重挂；异步媒体从任务快照继续结算 |
| 显式免费价格 | 不冻结、不扣费，但仍记录请求和持续时间 |

## 7. 主要源码落点

- Live 计费与恢复：`backend/internal/service/openai_live.go`
- Live 数据契约：`backend/internal/service/openai_live_types.go`
- Redis 快照与恢复索引：`backend/internal/repository/gateway_cache.go`
- 严格余额预留：`backend/internal/service/billing_inflight_reservation.go`
- Live API 错误与日志：`backend/internal/handler/openai_live.go`
- 异步媒体预留入口：`backend/internal/handler/gateway_inflight_reservation.go`
- Grok/Seedance 终态结算：`backend/internal/handler/grok_media.go`
- 配置定义与示例：`backend/internal/config/config.go`、`deploy/config.example.yaml`

## 8. 验证结果

已执行：

```text
go test ./internal/config ./internal/service ./internal/repository ./internal/handler
go test -race ./internal/service -run 'TestPrepareLiveBillingPlan|TestFinalizeLiveCall|TestReleaseGrokVideoBillingReservation' -count=1
go test -race ./internal/repository -run TestGatewayCacheLiveCallIdentityAndController -count=1
go test -race ./internal/handler -run 'TestReserveAsyncMediaBalance|TestAsyncMediaTerminalStatus' -count=1
```

结果：全部通过。完整包测试覆盖配置、服务、仓储和 HTTP 处理层；竞态测试覆盖 Live 结算、Redis 会话映射和异步媒体预留。

## 9. 本阶段边界

- Live 目前按会话持续时间计价，不读取上游逐帧音频用量；最长时长和预留上限负责限制风险。
- B4 仍是现有余额/订阅账单上的保护层，不提供完整不可变账本、账单冲正或历史价格版本。这些能力应在后续账务演进中处理。
- `disabled` 只用于紧急兼容回退。若需要灰度，优先使用 `observe` 或分组 Live 开关，避免重新暴露无限制零价入口。
