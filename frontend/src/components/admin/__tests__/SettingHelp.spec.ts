import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import SettingHelp from '../SettingHelp.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (_key: string, values: { setting: string }) => `${values.setting}：查看配置说明` }) }))

describe('配置说明问号', () => {
  afterEach(() => { document.body.innerHTML = '' })

  it('悬停显示说明，不会切换关联复选框或提交表单', async () => {
    const wrapper = mount({
      components: { SettingHelp },
      template: '<form @submit.prevent="submits++"><label><input v-model="enabled" type="checkbox" />启用<SettingHelp title="启用">开启后允许访问</SettingHelp></label><span data-testid="submits">{{ submits }}</span></form>',
      data: () => ({ enabled: false, submits: 0 }),
    }, {
      attachTo: document.body,
    })
    const button = wrapper.get('button')
    const trigger = wrapper.get('.group')
    expect(button.attributes('aria-label')).toBe('启用：查看配置说明')
    await trigger.trigger('mouseenter')
    await nextTick()
    expect(wrapper.get<HTMLInputElement>('input').element.checked).toBe(false)
    expect(wrapper.get('[data-testid="submits"]').text()).toBe('0')
    expect(document.querySelector('[role="tooltip"]')?.textContent).toContain('开启后允许访问')
    await trigger.trigger('mouseleave')
    await nextTick()
    expect(document.querySelector('[role="tooltip"]')).toBeNull()
    wrapper.unmount()
  })
})
