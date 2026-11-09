import { i18n } from '@/i18n'
import type { CustomMenuItem } from '@/types'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { resolveRouteDocumentTitle, resolveRouteMetaKeys } from './title'

const DEFAULT_DESCRIPTION = '一个密钥，畅用多个 AI 模型。无需管理多个订阅账号，一站式接入 Claude、GPT、Gemini 等主流 AI 服务。'

function translatedValue(key: string | undefined): string | undefined {
  if (!key) return undefined
  const value = i18n.global.t(key)
  return value !== key && typeof value === 'string' ? value : undefined
}

function setMeta(attribute: 'name' | 'property', key: string, content: string): void {
  let element = document.head.querySelector<HTMLMetaElement>(`meta[${attribute}="${key}"]`)
  if (!element) {
    element = document.createElement('meta')
    element.setAttribute(attribute, key)
    document.head.appendChild(element)
  }
  element.setAttribute('content', content)
}

function setCanonical(url: string): void {
  let element = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]')
  if (!element) {
    element = document.createElement('link')
    element.rel = 'canonical'
    document.head.appendChild(element)
  }
  element.href = url
}

function safeLogoUrl(siteLogo?: string): string {
  const value = siteLogo?.trim()
  return value && ((value.startsWith('/') && !value.startsWith('//')) || /^https?:\/\//i.test(value))
    ? new URL(value, window.location.origin).toString()
    : `${window.location.origin}/logo.svg`
}

function setJsonLd(payload: Record<string, unknown>): void {
  let element = document.head.querySelector<HTMLScriptElement>('#wavenode-seo-jsonld')
  if (!element) {
    element = document.createElement('script')
    element.id = 'wavenode-seo-jsonld'
    element.type = 'application/ld+json'
    document.head.appendChild(element)
  }
  // 避免站点配置中的特殊字符提前结束 script 标签，保证结构化数据始终是合法 JSON。
  element.textContent = JSON.stringify(payload).replace(/</g, '\\u003c').replace(/>/g, '\\u003e').replace(/&/g, '\\u0026')
}

function resolveDescription(route: RouteLocationNormalizedLoaded): string {
  const { descriptionKey } = resolveRouteMetaKeys(route)
  const routeDescription = translatedValue(descriptionKey)
  if (routeDescription) return routeDescription

  if (route.name === 'ModelPlaza') return translatedValue('modelPlaza.description') || DEFAULT_DESCRIPTION
  if (route.name === 'OfficialPrices') return translatedValue('officialPrices.description') || DEFAULT_DESCRIPTION
  if (route.name === 'LegalDocument') return translatedValue('legal.notFoundDescription') || DEFAULT_DESCRIPTION
  return DEFAULT_DESCRIPTION
}

function isIndexable(route: RouteLocationNormalizedLoaded): boolean {
  return route.meta.seoIndexable === true && route.meta.requiresAuth === false
}

/** 更新 SPA 路由对应的搜索引擎元信息，私有页面统一输出 noindex。 */
export function applyRouteSEO(
  route: RouteLocationNormalizedLoaded,
  siteName?: string,
  siteLogo?: string,
  customMenuItems: CustomMenuItem[] = [],
): void {
  const normalizedSiteName = siteName?.trim() || 'WaveNode'
  const title = resolveRouteDocumentTitle(route, normalizedSiteName, customMenuItems)
  const description = resolveDescription(route)
  const indexable = isIndexable(route)
  const canonical = new URL(route.path || '/home', window.location.origin).toString()
  const logo = safeLogoUrl(siteLogo)

  document.title = title
  document.documentElement.lang = i18n.global.locale.value === 'zh' ? 'zh-CN' : i18n.global.locale.value
  setMeta('name', 'description', description)
  setMeta('name', 'robots', indexable ? 'index,follow' : 'noindex,nofollow')
  setMeta('name', 'googlebot', indexable ? 'index,follow' : 'noindex,nofollow')
  setMeta('property', 'og:type', 'website')
  setMeta('property', 'og:site_name', normalizedSiteName)
  setMeta('property', 'og:title', title)
  setMeta('property', 'og:description', description)
  setMeta('property', 'og:url', canonical)
  setMeta('property', 'og:image', logo)
  setMeta('name', 'twitter:card', 'summary')
  setMeta('name', 'twitter:title', title)
  setMeta('name', 'twitter:description', description)
  setMeta('name', 'twitter:image', logo)
  setCanonical(canonical)
  if (indexable) {
    setJsonLd({
      '@context': 'https://schema.org',
      '@type': route.name === 'Home' ? 'WebSite' : 'WebPage',
      name: title,
      url: route.name === 'Home' ? window.location.origin + '/' : canonical,
      description,
      publisher: { '@type': 'Organization', name: normalizedSiteName, logo: { '@type': 'ImageObject', url: logo } },
    })
  } else {
    document.head.querySelector('#wavenode-seo-jsonld')?.remove()
  }
}
