import { createI18n } from 'vue-i18n'

type LocaleCode = 'en' | 'zh' | 'zh-TW'

type LocaleMessages = Record<string, any>

const LOCALE_KEY = 'wavenode_locale'
const DEFAULT_LOCALE: LocaleCode = 'en'

const localeLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./locales/en'),
  zh: () => import('./locales/zh'),
  // 繁体文案复用简体键结构，只在选择时加载转换字典，避免维护两份后台文案。
  'zh-TW': async () => {
    const [{ default: messages }, { Converter }] = await Promise.all([
      import('./locales/zh'),
      import('opencc-js/cn2t')
    ])
    const convert = Converter({ from: 'cn', to: 'tw' })
    const translate = (value: unknown): unknown => {
      if (typeof value === 'string') return convert(value)
      if (Array.isArray(value)) return value.map(translate)
      if (value && typeof value === 'object') {
        return Object.fromEntries(Object.entries(value).map(([key, child]) => [key, translate(child)]))
      }
      return value
    }
    const traditional = translate(messages) as LocaleMessages
    // 官网入口优先使用符合台湾用户习惯的词汇，而非单纯逐字转换。
    traditional.home.heroSubtitle = '一組金鑰，串接多種 AI 模型'
    traditional.home.heroDescription = '無須管理多個訂閱帳號，一站式串接 Claude、GPT、Gemini 等主流 AI 服務'
    traditional.home.login = '登入'
    traditional.home.guides = '串接指南'
    traditional.home.seoTitle = 'AI API 中轉與模型串接'
    return { default: traditional }
  }
}

function isLocaleCode(value: string): value is LocaleCode {
  return value === 'en' || value === 'zh' || value === 'zh-TW'
}

function htmlLang(locale: LocaleCode): string {
  return locale === 'zh' ? 'zh-CN' : locale
}

function getDefaultLocale(): LocaleCode {
  const saved = localStorage.getItem(LOCALE_KEY)
  if (saved && isLocaleCode(saved)) {
    return saved
  }

  const browserLang = navigator.language.toLowerCase()
  if (/^zh-(tw|hk|mo|hant)/.test(browserLang)) {
    return 'zh-TW'
  }
  if (browserLang.startsWith('zh')) {
    return 'zh'
  }

  return DEFAULT_LOCALE
}

export const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  messages: {},
  // 禁用 HTML 消息警告 - 引导步骤使用富文本内容（driver.js 支持 HTML）
  // 这些内容是内部定义的，不存在 XSS 风险
  warnHtmlMessage: false
})

const loadedLocales = new Set<LocaleCode>()

export async function loadLocaleMessages(locale: LocaleCode): Promise<void> {
  if (loadedLocales.has(locale)) {
    return
  }

  const loader = localeLoaders[locale]
  const module = await loader()
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(locale)
}

export async function initI18n(): Promise<void> {
  const current = getLocale()
  await loadLocaleMessages(current)
  document.documentElement.setAttribute('lang', htmlLang(current))
}

export async function setLocale(locale: string): Promise<void> {
  if (!isLocaleCode(locale)) {
    return
  }

  await loadLocaleMessages(locale)
  i18n.global.locale.value = locale
  localStorage.setItem(LOCALE_KEY, locale)
  document.documentElement.setAttribute('lang', htmlLang(locale))

  // 同步更新浏览器页签标题，使其跟随语言切换
  const { applyRouteSEO } = await import('@/router/seo')
  const { default: router } = await import('@/router')
  const { useAppStore } = await import('@/stores/app')
  const { useAuthStore } = await import('@/stores/auth')
  const { useAdminSettingsStore } = await import('@/stores/adminSettings')
  const route = router.currentRoute.value
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  applyRouteSEO(route, appStore.siteName, appStore.siteLogo, customMenuItems)
}

export function getLocale(): LocaleCode {
  const current = i18n.global.locale.value
  return isLocaleCode(current) ? current : DEFAULT_LOCALE
}

export const availableLocales = [
  { code: 'en', name: 'English', flag: '🇺🇸' },
  { code: 'zh', name: '简体中文', flag: '🇨🇳' },
  { code: 'zh-TW', name: '繁體中文', flag: '🇹🇼' }
] as const

export default i18n
