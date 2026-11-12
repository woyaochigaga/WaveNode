import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN } from '@/constants/channel'
import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'

export interface PlazaOffering {
  group: ModelPlazaGroup
  model: PlazaModel
}

export interface CatalogModel {
  key: string
  name: string
  platform: string
  mode: string
  offerings: PlazaOffering[]
  best: PlazaOffering
}

/** 图片独立倍率优先于用户/分组倍率，其余模式按用户实际生效倍率。 */
export function offeringRate(offering: PlazaOffering): number {
  const { group, model } = offering
  return model.pricing?.billing_mode === BILLING_MODE_IMAGE && group.image_rate_independent
    ? (group.image_rate_multiplier ?? 1)
    : (group.user_rate_multiplier ?? group.rate_multiplier)
}

/** 同一平台的同名模型合并为一张卡片；不同平台的同名模型保持独立。 */
export function buildCatalog(groups: ModelPlazaGroup[]): CatalogModel[] {
  const catalog = new Map<string, CatalogModel>()
  for (const group of groups) {
    for (const model of group.models) {
      const key = `${model.platform}\0${model.name}\0${model.pricing?.billing_mode ?? BILLING_MODE_TOKEN}`
      const offering = { group, model }
      const existing = catalog.get(key)
      if (existing) {
        existing.offerings.push(offering)
        if (compareOfferings(offering, existing.best) < 0) existing.best = offering
      } else {
        catalog.set(key, {
          key,
          name: model.name,
          platform: model.platform,
          mode: model.pricing?.billing_mode ?? BILLING_MODE_TOKEN,
          offerings: [offering],
          best: offering
        })
      }
    }
  }
  return [...catalog.values()].sort((left, right) => left.name.localeCompare(right.name))
}

/** 卡片仅比较相同计费单位的基础档；缺失价格时排在有价格的分组之后。 */
export function compareOfferings(left: PlazaOffering, right: PlazaOffering): number {
  const leftPrice = priceValues(left)
  const rightPrice = priceValues(right)
  return (leftPrice.primary ?? Infinity) - (rightPrice.primary ?? Infinity) ||
    (leftPrice.output ?? Infinity) - (rightPrice.output ?? Infinity)
}

/** 返回基础档原始单价（USD/token 或 USD/次），不含分时与高峰倍率。 */
export function priceValues(offering: PlazaOffering): {
  primary: number | null
  output: number | null
  cache: number | null
  unit: 'token' | 'request' | 'image'
  tiered: boolean
  timed: boolean
} {
  const { group, model } = offering
  const pricing = model.pricing
  const mode = pricing?.billing_mode ?? BILLING_MODE_TOKEN
  const rate = offeringRate(offering)
  const intervals = pricing?.intervals ?? []
  const base = intervals.length && pricing
    ? resolveIntervalPrices([...intervals].sort((a, b) => a.min_tokens - b.min_tokens)[0]!, pricing)
    : null
  const timed = Boolean(model.time_pricing?.periods?.length || group.peak_rate_enabled)
  if (mode !== BILLING_MODE_TOKEN) {
    const requestPrice = intervals.find(interval => interval.per_request_price != null)?.per_request_price
      ?? pricing?.per_request_price
    return {
      primary: requestPrice == null ? null : requestPrice * rate,
      output: null,
      cache: null,
      unit: mode === BILLING_MODE_IMAGE ? 'image' : 'request',
      tiered: intervals.length > 1,
      timed
    }
  }
  return {
    primary: (base?.input_price ?? pricing?.input_price) == null ? null : (base?.input_price ?? pricing?.input_price)! * rate,
    output: (base?.output_price ?? pricing?.output_price) == null ? null : (base?.output_price ?? pricing?.output_price)! * rate,
    cache: (base?.cache_read_price ?? pricing?.cache_read_price) == null ? null : (base?.cache_read_price ?? pricing?.cache_read_price)! * rate,
    unit: 'token',
    tiered: intervals.length > 1,
    timed
  }
}

/** 单位切换仅改变展示刻度，按次/按图计费保持原价。 */
export function displayPrice(value: number | null, unit: 'token' | 'request' | 'image', tokenScale: 1_000_000 | 1_000): string {
  return formatScaled(value, unit === 'token' ? tokenScale : 1, 2)
}
