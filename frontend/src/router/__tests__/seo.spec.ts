import { afterEach, describe, expect, it, vi } from 'vitest'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { applyRouteSEO } from '@/router/seo'
import { i18n } from '@/i18n'

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      locale: { value: 'zh' },
      t: (key: string) => ({
        'home.seoTitle': 'AI API 中转与模型接入',
        'home.seoDescription': '一个 API Key 接入主流 AI 服务',
        'modelPlaza.description': '按分组浏览可用模型与价格',
      }[key] ?? key),
    },
  },
}))

function route(overrides: Partial<RouteLocationNormalizedLoaded> = {}): RouteLocationNormalizedLoaded {
  return {
    name: 'Home',
    path: '/home',
    fullPath: '/home',
    params: {},
    query: {},
    hash: '',
    meta: { requiresAuth: false, seoIndexable: true, titleKey: 'home.seoTitle', descriptionKey: 'home.seoDescription' },
    matched: [],
    redirectedFrom: undefined,
    ...overrides,
  } as RouteLocationNormalizedLoaded
}

describe('applyRouteSEO', () => {
  afterEach(() => {
    document.head.querySelectorAll('meta[name="description"], meta[name="robots"], meta[name="googlebot"], meta[property^="og:"], meta[name^="twitter:"]').forEach((element) => element.remove())
    document.head.querySelector('link[rel="canonical"]')?.remove()
    document.head.querySelector('#wavenode-seo-jsonld')?.remove()
  })

  it('为公开页面写入可索引元信息和 canonical', () => {
    applyRouteSEO(route(), 'WaveNode')

    expect(document.title).toBe('AI API 中转与模型接入 - WaveNode')
    expect(document.querySelector('meta[name="robots"]')?.getAttribute('content')).toBe('index,follow')
    expect(document.querySelector('meta[name="description"]')?.getAttribute('content')).toBe('一个 API Key 接入主流 AI 服务')
    expect(document.querySelector('link[rel="canonical"]')?.getAttribute('href')).toBe(`${window.location.origin}/home`)
    expect(document.querySelector('#wavenode-seo-jsonld')?.textContent).toContain('WebSite')
  })

  it('为私有页面关闭索引并去掉查询参数影响', () => {
    applyRouteSEO(route({ name: 'Dashboard', path: '/dashboard', fullPath: '/dashboard?tab=usage', meta: { requiresAuth: true, title: 'Dashboard' } }), 'WaveNode')

    expect(document.querySelector('meta[name="robots"]')?.getAttribute('content')).toBe('noindex,nofollow')
    expect(document.querySelector('meta[name="googlebot"]')?.getAttribute('content')).toBe('noindex,nofollow')
    expect(document.querySelector('link[rel="canonical"]')?.getAttribute('href')).toBe(`${window.location.origin}/dashboard`)
    expect(document.querySelector('#wavenode-seo-jsonld')).toBeNull()
  })

  it('繁体中文使用对应的页面语言代码', () => {
    i18n.global.locale.value = 'zh-TW'
    applyRouteSEO(route(), 'WaveNode')
    expect(document.documentElement.lang).toBe('zh-TW')
    i18n.global.locale.value = 'zh'
  })
})
