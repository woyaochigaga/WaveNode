import { describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import AppHeader from '../AppHeader.vue'

const { auth } = vi.hoisted(() => ({ auth: { isAdmin: true, isSimpleMode: false, user: { role: 'admin', username: '测试用户', balance: 0 } } }))
vi.mock('@/stores', () => ({
  useAuthStore: () => auth,
  useAppStore: () => ({ cachedPublicSettings: {}, contactInfo: '', docUrl: '' }),
  useOnboardingStore: () => ({ replay: vi.fn() }),
}))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({ customMenuItems: [] }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ meta: {}, params: {} }), useRouter: () => ({ push: vi.fn() }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/utils/featureFlags', async (importOriginal) => ({ ...await importOriginal<typeof import('@/utils/featureFlags')>(), isFeatureFlagEnabled: () => false }))

describe('右上角系统设置入口', () => {
  it.each([true, false])('按管理员身份显示入口，并在点击后关闭菜单（管理员：%s）', async (isAdmin) => {
    auth.isAdmin = isAdmin
    auth.user.role = isAdmin ? 'admin' : 'user'
    const wrapper = shallowMount(AppHeader, {
      global: {
        mocks: { $t: (key: string) => key },
        stubs: { transition: { template: '<div><slot /></div>' }, RouterLink: { props: ['to'], template: '<a :href="to" @click.prevent><slot /></a>' } },
      },
    })
    await wrapper.get('[aria-label="common.userMenu"]').trigger('click')
    const link = wrapper.find('a[href="/admin/settings"]')
    expect(link.exists()).toBe(isAdmin)
    if (isAdmin) {
      expect(link.text()).toBe('nav.settings')
      await link.trigger('click')
      expect(wrapper.find('.dropdown').exists()).toBe(false)
    }
    wrapper.unmount()
  })
})
