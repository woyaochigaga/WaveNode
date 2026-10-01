import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ConnectionTestModal from '../ConnectionTestModal.vue'

const { getGroupTestModels, getKeyTestModels, copyToClipboard } = vi.hoisted(() => ({
  getGroupTestModels: vi.fn(),
  getKeyTestModels: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { groups: { getTestModels: getGroupTestModels } }
}))

vi.mock('@/api', () => ({
  keysAPI: { getTestModels: getKeyTestModels }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

const BaseDialogStub = {
  props: ['show', 'title'],
  emits: ['close'],
  template: '<section v-if="show"><slot /><slot name="footer" /></section>'
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
}

const IconStub = {
  props: ['name'],
  template: '<span>{{ name }}</span>'
}

const mountModal = () => mount(ConnectionTestModal, {
  props: {
    show: false,
    targetType: 'group',
    targetId: 12,
    targetName: 'Primary Group'
  },
  global: {
    stubs: {
      BaseDialog: BaseDialogStub,
      Select: SelectStub,
      Icon: IconStub
    }
  }
})

describe('ConnectionTestModal', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    getGroupTestModels.mockResolvedValue(['claude-haiku-4-5', 'claude-sonnet-4-6'])
    getKeyTestModels.mockResolvedValue(['gpt-5-mini'])
    localStorage.setItem('auth_token', 'test-token')
  })

  it('loads runnable group models when opened', async () => {
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(getGroupTestModels).toHaveBeenCalledWith(12)
    expect(wrapper.find('select').element.value).toBe('claude-haiku-4-5')
  })

  it('renders a successful streamed test result', async () => {
    const encoder = new TextEncoder()
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('data: {"type":"status","status":"route_selected"}\n\n'))
        controller.enqueue(encoder.encode('data: {"type":"test_start","model":"claude-haiku-4-5"}\n\n'))
        controller.enqueue(encoder.encode('data: {"type":"content","text":"pong"}\n\n'))
        controller.enqueue(encoder.encode('data: {"type":"test_complete","success":true}\n\n'))
        controller.close()
      }
    })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(stream, { status: 200 })))
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const startButton = wrapper.findAll('button').find((button) =>
      button.text().includes('common.connectionTest.start')
    )
    expect(startButton).toBeDefined()
    await startButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('pong')
    expect(wrapper.text()).toContain('common.connectionTest.success')
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/admin/groups/12/test'),
      expect.objectContaining({ method: 'POST' })
    )
  })

  it('shows a localized API key preflight error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      code: 400,
      message: 'This API key has expired',
      reason: 'API_KEY_TEST_EXPIRED'
    }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    })))
    const wrapper = mountModal()
    await wrapper.setProps({
      show: true,
      targetType: 'api-key',
      targetId: 25,
      targetName: 'Production Key'
    })
    await flushPromises()

    const startButton = wrapper.findAll('button').find((button) =>
      button.text().includes('common.connectionTest.start')
    )
    await startButton!.trigger('click')
    await flushPromises()

    expect(getKeyTestModels).toHaveBeenCalledWith(25)
    expect(wrapper.text()).toContain('common.connectionTest.errors.keyExpired')
  })
})
