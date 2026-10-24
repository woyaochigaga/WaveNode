import type { OfficialPrice } from '@/api/officialPrices'

/** 首页优先展示常用月付套餐；切换套餐后所有地区都使用同一套餐进行比较。 */
export const officialApps = [
  { slug: 'chatgpt', name: 'ChatGPT', mark: 'G', plan: 'ChatGPT Plus（月付）' },
  { slug: 'claude', name: 'Claude', mark: 'C', plan: 'Claude Pro（月付）' },
  { slug: 'grok', name: 'Grok', mark: 'x', plan: 'SuperGrok（月付）' }
]

export const officialCountries = ['cn', 'us', 'jp', 'gb', 'sg', 'tw', 'au', 'ca', 'de', 'in', 'id', 'ph', 'eg', 'pk', 'vn', 'tr', 'ng', 'my', 'ar', 'mx', 'co', 'kz', 'br', 'th', 'il', 'ae', 'cl', 'ch', 'za', 'sa', 'kr', 'dk', 'no']

/** code 为来源的两位地区码，locale 决定地区名称的显示语言。 */
export function countryName(code: string, locale: string): string {
  return new Intl.DisplayNames([locale], { type: 'region' }).of(code.toUpperCase()) || code.toUpperCase()
}

// 用 Unicode 旗帜避免为每个地区引入远程图片资源。
export function countryFlag(code: string): string {
  return /^[a-z]{2}$/i.test(code)
    ? String.fromCodePoint(...[...code.toUpperCase()].map(char => char.charCodeAt(0) + 127397))
    : ''
}

/** amount 已是来源换算的人民币值，此处仅按界面语言格式化，不重复换汇。 */
export function cnyPrice(amount: number, locale: string): string {
  return new Intl.NumberFormat(locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(amount)
}

/** value 为来源或同步时间，未同步时以占位符展示。 */
export function priceDate(value: string | null, locale: string): string {
  if (!value || !Number.isFinite(Date.parse(value))) return '—'
  return new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

/** 仅从该产品现有套餐中选默认项，不能用另一个套餐的低价填补缺失地区。 */
export function preferredPricePlan(records: OfficialPrice[], slug: string): string {
  const plans = records.filter(price => price.slug === slug)
  const preferred = officialApps.find(app => app.slug === slug)?.plan
  return plans.find(price => price.plan === preferred)?.plan
    || plans.find(price => price.kind === 'subscription' && price.billing_period === 'monthly')?.plan
    || plans[0]?.plan || ''
}

/** 从全部 records 中选出 slug 产品的 plan 套餐，不修改原始快照顺序。 */
export function sortedPlanPrices(records: OfficialPrice[], slug: string, plan: string): OfficialPrice[] {
  return records.filter(price => price.slug === slug && price.plan === plan)
    .sort((a, b) => a.price_cny - b.price_cny || a.country_code.localeCompare(b.country_code))
}

/** 最低价取完整地区集合；首页参考地区只影响普通报价列，国内始终保留且不做替代。 */
export function homePriceSummary(records: OfficialPrice[], slug: string, plan: string, countries: string[]) {
  const prices = sortedPlanPrices(records, slug, plan)
  const lowest = prices[0]
  return {
    lowest,
    regionCount: prices.length,
    regions: [...new Set(['cn', ...countries.filter(code => code !== lowest?.country_code)])]
      .map(code => ({ code, price: prices.find(price => price.country_code === code) }))
  }
}
