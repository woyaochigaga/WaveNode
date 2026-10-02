import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import ProxiesView from '../ProxiesView.vue'

const { list, getAllWithCount, testProxy, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  getAllWithCount: vi.fn(),
  testProxy: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { proxies: { list, getAllWithCount, testProxy } }
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))
vi.mock('vue-i18n', async () => ({
  ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')),
  useI18n: () => ({ t: (key: string) => key })
}))

const mountView = () =>
  shallowMount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="table" /></div>' },
        DataTable: {
          props: [
            'data',
            'columns',
            'loading',
            'serverSideSort',
            'defaultSortKey',
            'defaultSortOrder'
          ],
          emits: ['sort'],
          template:
            '<div><div v-for="row in data" :key="row.id"><slot name="cell-actions" :row="row" /></div></div>'
        },
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        },
        Icon: true
      }
    }
  })

describe('代理连接测试报告', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    list.mockResolvedValue({
      items: [
        {
          id: 9,
          name: 'edge-proxy',
          protocol: 'http',
          host: 'proxy.example',
          port: 8080,
          status: 'active'
        }
      ],
      total: 1,
      pages: 1
    })
    getAllWithCount.mockResolvedValue([])
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('失败时展示稳定错误码、阶段、HTTP 状态和调度建议', async () => {
    testProxy.mockResolvedValue({
      success: false,
      status: 'failed',
      stage: 'proxy',
      error_code: 'PROXY_AUTH_FAILED',
      http_status: 407,
      latency_ms: 34,
      tested_at: '2026-10-02T01:02:03Z',
      safe_to_schedule: false,
      message: 'proxy authentication failed'
    })

    const wrapper = mountView()
    await flushPromises()
    const testButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.proxies.testConnection')
    expect(testButton).toBeTruthy()

    await testButton!.trigger('click')
    await flushPromises()

    const report = wrapper.get('[data-testid="proxy-test-report"]')
    expect(report.text()).toContain('admin.proxies.testStatusFailed')
    expect(report.text()).toContain('admin.proxies.testStages.proxy')
    expect(report.text()).toContain('407')
    expect(report.text()).toContain('34ms')
    expect(report.text()).toContain('PROXY_AUTH_FAILED')
    expect(report.text()).toContain('admin.proxies.testNotSchedulable')
    expect(showError).toHaveBeenCalledWith('proxy authentication failed')
  })
})
