import type { Account, AccountUsageInfo, UsageProgress, WindowStats } from '@/types'

export type AccountQuotaGroup = 'local' | 'upstream' | 'capacity'
export type AccountQuotaUnit = 'usd' | 'requests' | 'tokens' | 'count' | 'percent'

export interface AccountQuotaItem {
  id: string
  group: AccountQuotaGroup
  label?: string
  labelKey?: string
  used?: number
  limit?: number
  remaining?: number
  utilization?: number
  unit: AccountQuotaUnit
  resetAt?: string | null
  estimatedRemainingCost?: number
  unlimited?: boolean
}

function finiteNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function positiveNumber(value: unknown): number | undefined {
  const parsed = finiteNumber(value)
  return parsed != null && parsed > 0 ? parsed : undefined
}

function exactQuotaItem(options: {
  id: string
  group: AccountQuotaGroup
  label?: string
  labelKey?: string
  used?: unknown
  limit?: unknown
  remaining?: unknown
  unit: AccountQuotaUnit
  resetAt?: string | null
}): AccountQuotaItem | null {
  const limit = positiveNumber(options.limit)
  const explicitRemaining = finiteNumber(options.remaining)
  if (limit == null && explicitRemaining == null) return null

  const used = finiteNumber(options.used)
  const remaining = explicitRemaining ?? (limit != null ? Math.max(0, limit - (used ?? 0)) : undefined)
  const utilization = limit != null && used != null ? Math.max(0, (used / limit) * 100) : undefined
  return { ...options, used, limit, remaining, utilization }
}

function estimatedRemainingCost(progress: UsageProgress): number | undefined {
  const utilization = finiteNumber(progress.utilization)
  const currentCost = finiteNumber(progress.window_stats?.cost)
  if (utilization == null || currentCost == null || utilization <= 0 || currentCost <= 0) return undefined
  const estimate = currentCost * Math.max(0, 100 - utilization) / utilization
  return Number.isFinite(estimate) ? estimate : undefined
}

function progressItem(id: string, label: string, progress: UsageProgress | null | undefined): AccountQuotaItem | null {
  if (!progress) return null
  const usedRequests = finiteNumber(progress.used_requests)
  const requestLimit = positiveNumber(progress.limit_requests)
  if (requestLimit != null) {
    return {
      id,
      group: 'upstream',
      label,
      used: usedRequests ?? 0,
      limit: requestLimit,
      remaining: Math.max(0, requestLimit - (usedRequests ?? 0)),
      utilization: finiteNumber(progress.utilization),
      unit: 'requests',
      resetAt: progress.resets_at,
      estimatedRemainingCost: estimatedRemainingCost(progress)
    }
  }
  return {
    id,
    group: 'upstream',
    label,
    utilization: finiteNumber(progress.utilization) ?? 0,
    unit: 'percent',
    resetAt: progress.resets_at,
    estimatedRemainingCost: estimatedRemainingCost(progress)
  }
}

function statsTokens(stats: WindowStats | null | undefined): number | undefined {
  return finiteNumber(stats?.tokens)
}

function numericString(value: unknown): number | undefined {
  if (typeof value !== 'string' || !value.trim()) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function unixResetAt(resetAt: string | null | undefined, resetUnix: number | null | undefined): string | null | undefined {
  if (resetAt) return resetAt
  return resetUnix != null && resetUnix > 0 ? new Date(resetUnix * 1000).toISOString() : undefined
}

/**
 * 将各平台不同形态的额度快照归一化，供统计弹窗统一展示。
 * 未配置上限且没有上游剩余值的项目不会生成，避免把“未知”误报为“无限”。
 */
export function buildAccountQuotaItems(account: Account, usage: AccountUsageInfo | null): AccountQuotaItem[] {
  const items: Array<AccountQuotaItem | null> = [
    exactQuotaItem({
      id: 'local-daily', group: 'local', labelKey: 'admin.accounts.stats.quota.daily',
      used: account.quota_daily_used, limit: account.quota_daily_limit, unit: 'usd', resetAt: account.quota_daily_reset_at
    }),
    exactQuotaItem({
      id: 'local-weekly', group: 'local', labelKey: 'admin.accounts.stats.quota.weekly',
      used: account.quota_weekly_used, limit: account.quota_weekly_limit, unit: 'usd', resetAt: account.quota_weekly_reset_at
    }),
    exactQuotaItem({
      id: 'local-total', group: 'local', labelKey: 'admin.accounts.stats.quota.total',
      used: account.quota_used, limit: account.quota_limit, unit: 'usd'
    }),
    exactQuotaItem({
      id: 'session-window', group: 'local', labelKey: 'admin.accounts.stats.quota.sessionWindow',
      used: account.current_window_cost, limit: account.window_cost_limit, unit: 'usd', resetAt: account.session_window_end
    }),
    exactQuotaItem({
      id: 'concurrency', group: 'capacity', labelKey: 'admin.accounts.stats.quota.concurrency',
      used: account.current_concurrency, limit: account.concurrency, unit: 'count'
    }),
    exactQuotaItem({
      id: 'sessions', group: 'capacity', labelKey: 'admin.accounts.stats.quota.sessions',
      used: account.active_sessions, limit: account.max_sessions, unit: 'count'
    }),
    exactQuotaItem({
      id: 'rpm', group: 'capacity', labelKey: 'admin.accounts.stats.quota.rpm',
      used: account.current_rpm, limit: account.base_rpm, unit: 'requests'
    })
  ]

  const resetCredits = account.extra?.codex_reset_credit_snapshot
  if (resetCredits?.available_count != null) {
    const expirations = (resetCredits.credits ?? [])
      .map((credit) => credit.expires_at)
      .filter((value): value is string => Boolean(value))
      .sort()
    items.push({
      id: 'codex-reset-credits',
      group: 'upstream',
      labelKey: 'admin.accounts.stats.quota.codexResetCredits',
      remaining: resetCredits.available_count,
      unit: 'count',
      resetAt: expirations[0]
    })
  }

  const codexCredits = account.extra?.codex_credits_snapshot?.credits
  if (codexCredits?.unlimited) {
    items.push({
      id: 'codex-credits', group: 'upstream', labelKey: 'admin.accounts.stats.quota.codexCredits',
      unit: 'count', unlimited: true
    })
  } else {
    const codexBalance = numericString(codexCredits?.balance)
    if (codexBalance != null) {
      items.push({
        id: 'codex-credits', group: 'upstream', labelKey: 'admin.accounts.stats.quota.codexCredits',
        remaining: codexBalance, unit: 'count'
      })
    }
  }

  const ollamaData = account.ollama_cloud_usage?.snapshot?.data
  if (ollamaData?.five_hour) {
    items.push({
      id: 'ollama-five-hour', group: 'upstream', labelKey: 'admin.accounts.stats.quota.ollamaFiveHour',
      utilization: ollamaData.five_hour.used_percent, unit: 'percent', resetAt: ollamaData.five_hour.reset_at
    })
  }
  if (ollamaData?.seven_day) {
    items.push({
      id: 'ollama-seven-day', group: 'upstream', labelKey: 'admin.accounts.stats.quota.ollamaSevenDay',
      utilization: ollamaData.seven_day.used_percent, unit: 'percent', resetAt: ollamaData.seven_day.reset_at
    })
  }
  const ollamaBalance = numericString(ollamaData?.balance)
  if (ollamaBalance != null) {
    items.push({
      id: 'ollama-balance', group: 'upstream', labelKey: 'admin.accounts.stats.quota.ollamaBalance',
      remaining: ollamaBalance, unit: 'usd'
    })
  }

  const openCodeData = account.opencode_go_usage?.snapshot?.data
  const openCodeWindows = [
    ['opencode-rolling', 'admin.accounts.stats.quota.openCodeRolling', openCodeData?.rolling],
    ['opencode-weekly', 'admin.accounts.stats.quota.openCodeWeekly', openCodeData?.weekly],
    ['opencode-monthly', 'admin.accounts.stats.quota.openCodeMonthly', openCodeData?.monthly]
  ] as const
  for (const [id, labelKey, window] of openCodeWindows) {
    if (!window) continue
    items.push({
      id, group: 'upstream', labelKey, utilization: window.percent, unit: 'percent', resetAt: window.resets_at
    })
  }

  if (!usage) return items.filter((item): item is AccountQuotaItem => item != null)

  const windows: Array<[string, string, UsageProgress | null | undefined]> = [
    ['five-hour', '5h', usage.five_hour],
    ['seven-day', '7d', usage.seven_day],
    ['seven-day-sonnet', '7d Sonnet', usage.seven_day_sonnet],
    ['seven-day-fable', '7d Fable', usage.seven_day_fable],
    ['thirty-day', '30d', usage.thirty_day],
    ['gemini-shared-daily', 'Gemini Daily', usage.gemini_shared_daily],
    ['gemini-pro-daily', 'Gemini Pro Daily', usage.gemini_pro_daily],
    ['gemini-flash-daily', 'Gemini Flash Daily', usage.gemini_flash_daily],
    ['gemini-shared-minute', 'Gemini / min', usage.gemini_shared_minute],
    ['gemini-pro-minute', 'Gemini Pro / min', usage.gemini_pro_minute],
    ['gemini-flash-minute', 'Gemini Flash / min', usage.gemini_flash_minute]
  ]
  for (const [id, label, progress] of windows) items.push(progressItem(id, label, progress))

  for (const [model, quota] of Object.entries(usage.antigravity_quota ?? {})) {
    items.push({
      id: `antigravity-${model}`,
      group: 'upstream',
      label: model,
      utilization: finiteNumber(quota.utilization) ?? 0,
      unit: 'percent',
      resetAt: quota.reset_time
    })
  }

  items.push(
    exactQuotaItem({
      id: 'grok-requests', group: 'upstream', labelKey: 'admin.accounts.stats.quota.grokRequests',
      limit: usage.grok_request_quota?.limit, remaining: usage.grok_request_quota?.remaining,
      used: usage.grok_request_quota?.limit != null && usage.grok_request_quota?.remaining != null
        ? usage.grok_request_quota.limit - usage.grok_request_quota.remaining : undefined,
      unit: 'requests', resetAt: unixResetAt(usage.grok_request_quota?.reset_at, usage.grok_request_quota?.reset_unix)
    }),
    exactQuotaItem({
      id: 'grok-tokens', group: 'upstream', labelKey: 'admin.accounts.stats.quota.grokTokens',
      limit: usage.grok_token_quota?.limit, remaining: usage.grok_token_quota?.remaining,
      used: usage.grok_token_quota?.limit != null && usage.grok_token_quota?.remaining != null
        ? usage.grok_token_quota.limit - usage.grok_token_quota.remaining : undefined,
      unit: 'tokens', resetAt: unixResetAt(usage.grok_token_quota?.reset_at, usage.grok_token_quota?.reset_unix)
    }),
    exactQuotaItem({
      id: 'grok-free-24h', group: 'upstream', labelKey: 'admin.accounts.stats.quota.grokFreeTokens',
      used: statsTokens(usage.grok_local_usage_24h), limit: usage.grok_free_token_limit, unit: 'tokens'
    }),
    exactQuotaItem({
      id: 'grok-monthly', group: 'upstream', labelKey: 'admin.accounts.stats.quota.monthlyBilling',
      used: usage.grok_billing?.monthly_used ?? (usage.grok_billing?.used_cents != null ? usage.grok_billing.used_cents / 100 : undefined),
      limit: usage.grok_billing?.monthly_limit ?? (usage.grok_billing?.monthly_limit_cents != null ? usage.grok_billing.monthly_limit_cents / 100 : undefined),
      unit: 'usd', resetAt: usage.grok_billing?.billing_period_end
    }),
    exactQuotaItem({
      id: 'grok-on-demand', group: 'upstream', labelKey: 'admin.accounts.stats.quota.onDemand',
      used: usage.grok_billing?.on_demand_used, limit: usage.grok_billing?.on_demand_cap, unit: 'usd'
    }),
    exactQuotaItem({
      id: 'grok-prepaid', group: 'upstream', labelKey: 'admin.accounts.stats.quota.prepaidBalance',
      remaining: usage.grok_billing?.prepaid_balance, unit: 'usd'
    })
  )

  for (const [index, credit] of (usage.ai_credits ?? []).entries()) {
    const amount = finiteNumber(credit.amount)
    if (amount == null) continue
    items.push({
      id: `ai-credit-${index}`,
      group: 'upstream',
      label: credit.credit_type || `AI Credit ${index + 1}`,
      remaining: amount,
      unit: 'count'
    })
  }

  return items.filter((item): item is AccountQuotaItem => item != null)
}
