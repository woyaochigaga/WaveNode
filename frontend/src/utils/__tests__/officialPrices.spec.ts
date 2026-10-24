import { describe, expect, it } from 'vitest'
import type { OfficialPrice } from '@/api/officialPrices'
import { homePriceSummary, sortedPlanPrices } from '../officialPrices'

function price(country: string, amount: number, plan = 'Plus'): OfficialPrice {
  return { app: 'ChatGPT', app_id: 'chatgpt', slug: 'chatgpt', plan, country_code: country, price_cny: amount, original_price: '$20', kind: 'subscription', billing_period: 'monthly', source_updated_at: '2026-10-22T10:00:00Z', source_url: 'https://www.openprice.cc/official-prices/chatgpt' }
}

describe('订阅价格比较', () => {
  it('最低价使用全部地区，国内缺价不会用台湾报价填补', () => {
    const summary = homePriceSummary([price('tw', 130), price('us', 135), price('jp', 120)], 'chatgpt', 'Plus', ['us', 'tw'])
    expect(summary.lowest?.country_code).toBe('jp')
    expect(summary.regions.find(region => region.code === 'cn')?.price).toBeUndefined()
    expect(summary.regionCount).toBe(3)
    expect(summary.regions.map(region => region.code)).toEqual(['cn', 'us', 'tw'])
  })

  it('比较同产品同套餐，年付或其他产品的更低数字不会混入', () => {
    const records = [price('us', 135), price('jp', 120), price('us', 1, 'Annual'), { ...price('us', 2), slug: 'claude' }]
    expect(sortedPlanPrices(records, 'chatgpt', 'Plus').map(row => row.price_cny)).toEqual([120, 135])
    expect(records[0]?.country_code).toBe('us')
  })

  it('保留零元报价和并列最低价，不重复显示已强调的地区', () => {
    const summary = homePriceSummary([price('us', 0), price('jp', 0)], 'chatgpt', 'Plus', ['jp', 'us', 'us'])
    expect(summary.lowest?.price_cny).toBe(0)
    expect(summary.regions.map(region => region.code)).toEqual(['cn', 'us'])
  })
})
