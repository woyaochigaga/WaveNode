import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'

const mocks = vi.hoisted(() => ({
  getRequestErrorDetail: vi.fn(),
  listRequestErrorUpstreamErrors: vi.fn(),
  getRouteTrace: vi.fn()
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestErrorDetail: mocks.getRequestErrorDetail,
    getUpstreamErrorDetail: vi.fn(),
    listRequestErrorUpstreamErrors: mocks.listRequestErrorUpstreamErrors,
    getRouteTrace: mocks.getRouteTrace
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError: vi.fn() })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('OpsErrorDetailModal', () => {
  beforeEach(() => {
    mocks.getRequestErrorDetail.mockReset()
    mocks.listRequestErrorUpstreamErrors.mockReset()
    mocks.listRequestErrorUpstreamErrors.mockResolvedValue({ items: [] })
    mocks.getRouteTrace.mockReset()
    mocks.getRouteTrace.mockResolvedValue({
      request_id: 'rid-1',
      requested_model: 'gpt-public',
      final_model: 'gpt-upstream',
      inbound_endpoint: '/v1/chat/completions',
      upstream_endpoint: '/v1/responses',
      config_version: 'cfg-123',
      price_version: 'price-456',
      candidate_count: 2,
      retry_count: 1,
      final_account_ref: 'acct_abc',
      final_error_code: 'rate_limited',
      final_error_message: 'request was rate limited',
      retryable: true,
      retry_after: 1,
      billing_status: 'not_charged',
      billed_amount: 0,
      usage_record_count: 0,
      attempts: [
        { account_ref: 'acct_first', platform: 'openai', status_code: 429, stage: 'upstream', reason: 'rate_limited' }
      ]
    })
  })

  it('prioritizes upstream root cause and deduplicates diagnostic payloads', async () => {
    mocks.getRequestErrorDetail.mockResolvedValue({
      id: 1,
      created_at: '2026-08-19T00:00:00Z',
      phase: 'request',
      type: 'upstream_error',
      error_owner: 'provider',
      error_source: 'gateway',
      severity: 'P1',
      status_code: 502,
      upstream_status_code: 429,
      platform: 'openai',
      model: 'gpt-5.6',
      resolved: false,
      request_id: 'rid-1',
      message: 'All available accounts exhausted',
      error_body: '{"error":"same"}',
      upstream_error_message: 'provider rate limit exhausted',
      upstream_error_detail: '{"error":"same"}',
      upstream_errors: '[]',
      account_name: 'account',
      group_name: 'group',
      is_business_limited: false
    })

    const wrapper = shallowMount(OpsErrorDetailModal, {
      props: { show: true, errorId: 1, errorType: 'request' },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /></div>' },
          Icon: true
        }
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('provider rate limit exhausted')
    expect(wrapper.text()).toContain('admin.ops.errorDetail.upstreamStatus')
    expect(wrapper.text()).toContain('429')
    expect(wrapper.findAll('pre')).toHaveLength(2)
    expect(wrapper.text()).not.toContain('admin.ops.errorDetail.payloads.upstream_detail')
    expect(mocks.getRouteTrace).toHaveBeenCalledWith('rid-1')
    expect(wrapper.get('[data-testid="route-trace"]').text()).toContain('cfg-123')
    expect(wrapper.get('[data-testid="route-trace"]').text()).toContain('price-456')
    expect(wrapper.get('[data-testid="route-trace"]').text()).toContain('acct_first')
  })

  it('shows the permission state without exposing backend details', async () => {
    mocks.getRequestErrorDetail.mockResolvedValue({
      id: 2,
      created_at: '2026-08-19T00:00:00Z',
      phase: 'request',
      type: 'upstream_error',
      error_owner: 'provider',
      error_source: 'gateway',
      severity: 'P1',
      status_code: 502,
      platform: 'openai',
      model: 'gpt-5.6',
      resolved: false,
      request_id: 'rid-forbidden',
      message: 'failed',
      is_business_limited: false
    })
    mocks.getRouteTrace.mockRejectedValue({ status: 403, message: 'sensitive backend detail' })

    const wrapper = shallowMount(OpsErrorDetailModal, {
      props: { show: true, errorId: 2, errorType: 'request' },
      global: { stubs: { BaseDialog: { template: '<div><slot /></div>' }, Icon: true } }
    })
    await flushPromises()

    const trace = wrapper.get('[data-testid="route-trace"]').text()
    expect(trace).toContain('admin.ops.errorDetail.routeTrace.forbidden')
    expect(trace).not.toContain('sensitive backend detail')
  })
})
