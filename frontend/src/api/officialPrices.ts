import { apiClient } from './client'

/** 来源报价保留原始货币文本和更新时间；price_cny 为来源换算后的人民币金额。 */
export interface OfficialPrice {
  app: string
  app_id: string
  slug: string
  plan: string
  kind: 'subscription' | 'credit'
  billing_period: 'monthly' | 'annual' | null
  country_code: string
  original_price: string
  price_cny: number
  source_updated_at: string
  source_url: string
}

export interface OfficialPricesConfig {
  enabled: boolean
  home_enabled: boolean
  refresh_hours: number
  apps: string[]
  home_countries: string[]
}

export interface OfficialPricesStatus {
  config: OfficialPricesConfig
  syncing: boolean
  last_attempt: string | null
  last_success: string | null
  next_refresh: string | null
  last_error: string
  record_count: number
}

export interface OfficialPricesPublic {
  enabled: boolean
  home_enabled: boolean
  home_countries: string[]
  fetched_at: string | null
  records: OfficialPrice[]
}

export const officialPricesAPI = {
  async getPublic(home = false): Promise<OfficialPricesPublic> {
    return (await apiClient.get('/settings/official-prices', { params: { home } })).data
  },
  async getStatus(): Promise<OfficialPricesStatus> {
    return (await apiClient.get('/admin/settings/official-prices')).data
  },
  async save(config: OfficialPricesConfig): Promise<OfficialPricesStatus> {
    return (await apiClient.put('/admin/settings/official-prices', config)).data
  },
  async refresh(): Promise<OfficialPricesStatus> {
    // 三个公开页面由后端顺序抓取，手动同步允许比普通 API 更长的等待时间。
    return (await apiClient.post('/admin/settings/official-prices/refresh', null, { timeout: 90000 })).data
  }
}
