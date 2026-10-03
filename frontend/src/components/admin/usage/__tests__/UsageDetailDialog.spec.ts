import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UsageDetailDialog from '../UsageDetailDialog.vue'

const mocks = vi.hoisted(() => ({ getDetail: vi.fn() }))

vi.mock('@/api/admin/usage', () => ({ adminUsageAPI: { getDetail: mocks.getDetail } }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))
vi.mock('@/utils/format', () => ({ formatDateTime: () => '2026-10-04 12:00:00' }))
vi.mock('@/utils/usageRequestType', () => ({ resolveUsageRequestType: () => 'sync' }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const usage = {
  id: 42,
  model: 'gpt-test',
  inbound_endpoint: '/v1/responses',
  input_tokens: 10,
  output_tokens: 5,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  actual_cost: 0.012345,
  duration_ms: 1234,
  created_at: '2026-10-04T04:00:00Z',
  request_id: 'client:req-42',
}

function mountDialog() {
  return mount(UsageDetailDialog, {
    props: { show: true, usageId: 42 },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        Icon: true,
      },
    },
  })
}

describe('UsageDetailDialog', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows a clear unavailable state for historical or expired details', async () => {
    mocks.getDetail.mockResolvedValue({ usage, detail: null })
    const wrapper = mountDialog()
    await flushPromises()

    expect(mocks.getDetail).toHaveBeenCalledWith(42)
    expect(wrapper.text()).toContain('usage.detail.notCaptured')
    expect(wrapper.text()).toContain('client:req-42')
  })

  it('renders categorized input and output tabs from the captured payload', async () => {
    mocks.getDetail.mockResolvedValue({
      usage,
      detail: {
        id: 1,
        request_id: 'client:req-42',
        api_key_id: 3,
        user_id: 2,
        method: 'POST',
        path: '/v1/responses',
        status_code: 200,
        request_content_type: 'application/json',
        response_content_type: 'application/json',
        request_body: JSON.stringify({ messages: [{ role: 'user', content: 'hello input' }] }),
        response_body: JSON.stringify({ output: [{ type: 'output_text', text: 'hello output' }] }),
        request_truncated: false,
        response_truncated: false,
        created_at: '2026-10-04T04:00:00Z',
        expires_at: '2026-10-11T04:00:00Z',
      },
    })
    const wrapper = mountDialog()
    await flushPromises()

    const tabs = wrapper.findAll('[role="tab"]')
    await tabs[1].trigger('click')
    expect(wrapper.text()).toContain('hello input')
    await tabs[2].trigger('click')
    expect(wrapper.text()).toContain('hello output')
  })
})
