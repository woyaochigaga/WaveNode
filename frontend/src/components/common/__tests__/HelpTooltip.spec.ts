import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

function getTooltipElement(): HTMLDivElement {
  const tooltip = document.body.querySelector('[role="tooltip"]')
  if (!(tooltip instanceof HTMLDivElement)) {
    throw new Error('tooltip element not found')
  }
  return tooltip
}

describe('HelpTooltip', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('keeps the existing hover interaction by default', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'hover details',
      },
    })

    const trigger = wrapper.get('.group')
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    await trigger.trigger('mouseenter')
    await nextTick()
    expect(getTooltipElement().textContent).toContain('hover details')

    await trigger.trigger('mouseleave')
    await nextTick()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    wrapper.unmount()
  })

  it('keeps a hover tooltip open while the pointer moves between the trigger and the tooltip', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'copyable details',
      },
    })

    const trigger = wrapper.get('.group')
    await trigger.trigger('mouseenter')
    await nextTick()
    const tooltip = getTooltipElement()
    expect(tooltip.style.display).not.toBe('none')

    await trigger.trigger('mouseleave', { relatedTarget: tooltip })
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    tooltip.dispatchEvent(new MouseEvent('mouseleave', { relatedTarget: trigger.element }))
    await nextTick()
    expect(tooltip.style.display).not.toBe('none')

    tooltip.dispatchEvent(new MouseEvent('mouseleave', { relatedTarget: null }))
    await nextTick()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    wrapper.unmount()
  })

  it('supports click-to-toggle details and closes on outside click', async () => {
    const wrapper = mount(HelpTooltip, {
      attachTo: document.body,
      props: {
        content: 'click details',
        trigger: 'click',
      },
    })

    const trigger = wrapper.get('.group')
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    await trigger.trigger('click')
    await nextTick()
    const tooltip = getTooltipElement()
    expect(tooltip.style.display).not.toBe('none')
    expect(tooltip.textContent).toContain('click details')

    const closeButton = tooltip.querySelector('button[aria-label="Close"]')
    if (!(closeButton instanceof HTMLButtonElement)) {
      throw new Error('close button not found')
    }
    closeButton.click()
    await nextTick()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    await trigger.trigger('click')
    await nextTick()
    expect(getTooltipElement().textContent).toContain('click details')

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    wrapper.unmount()
  })

  it('keeps a scrolled tooltip inside the viewport and closes with Escape', async () => {
    const wrapper = mount(HelpTooltip, { attachTo: document.body, props: { content: 'details', trigger: 'click' } })
    const trigger = wrapper.get('.group')
    await trigger.trigger('click')
    const tooltip = getTooltipElement()
    vi.stubGlobal('scrollY', 700)
    vi.stubGlobal('innerWidth', 1024)
    vi.stubGlobal('innerHeight', 768)
    vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({ top: 5, bottom: 25, left: 990, width: 20 } as DOMRect)
    vi.spyOn(tooltip, 'getBoundingClientRect').mockReturnValue({ width: 320, height: 100 } as DOMRect)
    window.dispatchEvent(new Event('scroll'))
    await nextTick()
    expect(tooltip.style.top).toBe('33px')
    expect(tooltip.style.left).toBe('696px')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
    wrapper.unmount()
  })
})
