import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PlazaFilterBar from '../PlazaFilterBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const props = {
  platforms: [{ value: 'all', count: 2 }, { value: 'openai', count: 1 }, { value: 'anthropic', count: 1 }],
  modes: [{ value: 'all', count: 2 }, { value: 'token', count: 1 }, { value: 'image', count: 1 }],
  groups: [{ id: 1, name: 'Standard' }, { id: 2, name: 'Plus' }],
  rates: [0.5, 1],
  platform: 'all',
  mode: 'all',
  groupId: 'all' as const,
  rate: 'all' as const
}

describe('PlazaFilterBar', () => {
  it('emits platform, billing mode, group, and rate selections', async () => {
    const wrapper = mount(PlazaFilterBar, { props, global: { stubs: { PlatformIcon: true } } })
    const facets = wrapper.findAll('[role="group"]')
    await facets[0]!.findAll('button')[1]!.trigger('click')
    await facets[1]!.findAll('button')[2]!.trigger('click')
    await facets[2]!.findAll('button')[1]!.trigger('click')
    await facets[3]!.findAll('button')[1]!.trigger('click')
    expect(wrapper.emitted('update:platform')?.[0]).toEqual(['openai'])
    expect(wrapper.emitted('update:mode')?.[0]).toEqual(['image'])
    expect(wrapper.emitted('update:groupId')?.[0]).toEqual([1])
    expect(wrapper.emitted('update:rate')?.[0]).toEqual([0.5])
  })

  it('shows real model counts and resets all facets', async () => {
    const wrapper = mount(PlazaFilterBar, { props, global: { stubs: { PlatformIcon: true } } })
    expect(wrapper.findAll('.plaza-filter-count').map(count => count.text())).toEqual(['2', '1', '1', '2', '1', '1'])
    const toggle = wrapper.get('[aria-controls="plaza-filters"]')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    await wrapper.findAll('button')[0]!.trigger('click')
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })
})
