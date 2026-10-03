import { describe, expect, it } from 'vitest'
import { buildAccountQuotaItems } from '../accountQuotaItems'
import type { Account, AccountUsageInfo } from '@/types'

function accountFixture(): Account {
  return {
    id: 7,
    name: 'quota-account',
    platform: 'openai',
    type: 'oauth',
    concurrency: 10,
    current_concurrency: 3,
    quota_daily_limit: 100,
    quota_daily_used: 35,
    status: 'active'
  } as Account
}

describe('buildAccountQuotaItems', () => {
  it('includes exact local remaining quota and runtime capacity', () => {
    const items = buildAccountQuotaItems(accountFixture(), null)

    expect(items.find((item) => item.id === 'local-daily')).toMatchObject({
      used: 35,
      limit: 100,
      remaining: 65,
      utilization: 35,
      unit: 'usd'
    })
    expect(items.find((item) => item.id === 'concurrency')).toMatchObject({
      used: 3,
      limit: 10,
      remaining: 7
    })
  })

  it('normalizes percentage, exact upstream and model quotas', () => {
    const usage = {
      updated_at: '2026-10-03T03:00:00Z',
      five_hour: {
        utilization: 25,
        resets_at: '2026-10-03T08:00:00Z',
        remaining_seconds: 18000,
        window_stats: { requests: 10, tokens: 1000, cost: 10 }
      },
      grok_request_quota: { limit: 100, remaining: 40, reset_at: '2026-10-04T00:00:00Z' },
      antigravity_quota: {
        'gemini-3-pro': { utilization: 60, reset_time: '2026-10-04T00:00:00Z' }
      }
    } as AccountUsageInfo

    const items = buildAccountQuotaItems(accountFixture(), usage)

    expect(items.find((item) => item.id === 'five-hour')).toMatchObject({
      utilization: 25,
      estimatedRemainingCost: 30
    })
    expect(items.find((item) => item.id === 'grok-requests')).toMatchObject({
      used: 60,
      limit: 100,
      remaining: 40
    })
    expect(items.find((item) => item.id === 'antigravity-gemini-3-pro')).toMatchObject({
      utilization: 60
    })
  })
})
