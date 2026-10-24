import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import HomeOfficialPrices from '@/components/home/HomeOfficialPrices.vue'
import OfficialPricesView from '../OfficialPricesView.vue'
import OfficialPricesSettings from '../admin/settings/OfficialPricesSettings.vue'
import { officialPricesAPI, type OfficialPricesPublic, type OfficialPricesStatus } from '@/api/officialPrices'
import zh from '@/i18n/locales/zh/officialPrices'

vi.mock('@/api/officialPrices', () => ({ officialPricesAPI: { getPublic: vi.fn(), getStatus: vi.fn(), save: vi.fn(), refresh: vi.fn() } }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({
    locale: ref('zh'),
    t: (key: string) => {
      let value: unknown = zh
      for (const part of key.split('.')) value = (value as Record<string, unknown> | undefined)?.[part]
      return typeof value === 'string' ? value : key
    }
  })
}))

const data: OfficialPricesPublic = {
  enabled: true, home_enabled: true, home_countries: ['us'], fetched_at: '2026-10-22T10:00:00Z',
  records: [['jp', 120], ['us', 135], ['tw', 130]].map(([country, amount]) => ({ app: 'ChatGPT', app_id: 'chatgpt', slug: 'chatgpt', plan: 'ChatGPT Plus（月付）', kind: 'subscription', billing_period: 'monthly', country_code: String(country), original_price: '$20', price_cny: Number(amount), source_updated_at: '2026-10-22T10:00:00Z', source_url: 'https://www.openprice.cc/official-prices/chatgpt' }))
}
const status: OfficialPricesStatus = { config: { enabled: true, home_enabled: true, refresh_hours: 24, apps: ['chatgpt'], home_countries: ['us'] }, syncing: false, last_attempt: null, last_success: null, next_refresh: null, last_error: '', record_count: 3 }

async function options(path = '/official-prices/chatgpt?country=us') {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/home', component: HomeOfficialPrices }, { path: '/official-prices/:app', name: 'OfficialPrices', component: OfficialPricesView }] })
  await router.push(path)
  await router.isReady()
  const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: { ...zh, common: { loading: '加载中' }, home: { switchToLight: '浅色', switchToDark: '深色' } } } })
  return { router, global: { plugins: [router, i18n], stubs: { Icon: true, PlatformIcon: true, LocaleSwitcher: true } } }
}

describe('订阅价格页面', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(officialPricesAPI.getPublic).mockResolvedValue(structuredClone(data))
    vi.mocked(officialPricesAPI.getStatus).mockResolvedValue(structuredClone(status))
  })

  it('首页开关关闭时不渲染价格，启用时显示国内缺价和地区链接', async () => {
    vi.mocked(officialPricesAPI.getPublic).mockResolvedValueOnce({ ...data, home_enabled: false })
    const hidden = mount(HomeOfficialPrices, await options())
    await flushPromises()
    expect(hidden.find('#official-prices').exists()).toBe(false)
    hidden.unmount()
    const visible = mount(HomeOfficialPrices, await options())
    await flushPromises()
    expect(visible.get('[data-country="cn"]').text()).toContain('暂无报价')
    expect(visible.get('.op-price').text()).toContain('120.00')
    expect(visible.get('[data-country="us"]').attributes('href')).toContain('country=us')
    expect(visible.get('.op-lowest').attributes('href')).toContain('/official-prices/chatgpt?plan=')
    visible.unmount()
  })

  it('从地区链接进入仍显示全部地区；排序和搜索不改变最低价', async () => {
    const wrapper = mount(OfficialPricesView, await options())
    await flushPromises()
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
    expect(wrapper.get('tr.is-selected').attributes('data-country')).toBe('us')
    await wrapper.get('input[type="search"]').setValue('美国')
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.get('.op-price').text()).toContain('120.00')
    await wrapper.get('input[type="search"]').setValue('')
    await wrapper.findAll('select')[1]!.setValue('true')
    expect(wrapper.findAll('tbody tr')[0]!.attributes('data-country')).toBe('us')
    wrapper.unmount()
  })

  it('配置失败不显示可保存的空表单；修改周期后必须先保存才能刷新', async () => {
    vi.mocked(officialPricesAPI.getStatus).mockRejectedValueOnce(new Error('offline'))
    const failed = mount(OfficialPricesSettings, await options())
    await flushPromises()
    expect(failed.find('select').exists()).toBe(false)
    expect(failed.text()).toContain('重新加载')
    failed.unmount()
    const wrapper = mount(OfficialPricesSettings, await options())
    await flushPromises()
    await wrapper.get('select').setValue(18)
    const refresh = wrapper.findAll('button').find(button => button.text() === '立即刷新')!
    expect(refresh.attributes('disabled')).toBeDefined()
    const changed = { ...status, config: { ...status.config, refresh_hours: 18 } }
    vi.mocked(officialPricesAPI.save).mockResolvedValue(changed)
    await wrapper.findAll('button').find(button => button.text() === '保存价格设置')!.trigger('click')
    await flushPromises()
    expect(officialPricesAPI.save).toHaveBeenCalledWith(changed.config)
    expect(refresh.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })
})
