import { describe, expect, it } from 'vitest'
import { i18n, loadLocaleMessages, availableLocales } from '@/i18n'
import zh from '../locales/zh'

describe('繁体中文语言包', () => {
  it('按需转换整套文案并校订官网用语', async () => {
    await loadLocaleMessages('zh-TW')
    const traditional = i18n.global.getLocaleMessage('zh-TW') as typeof zh

    expect(availableLocales.map(({ code }) => code)).toContain('zh-TW')
    expect(Object.keys(traditional)).toEqual(Object.keys(zh))
    expect(traditional.home.heroSubtitle).toBe('一組金鑰，串接多種 AI 模型')
    expect(traditional.home.login).toBe('登入')
    expect(traditional.home.guides).toBe('串接指南')
    expect(traditional.keyUsage.daysLeft).toContain('{days}')
  })
})
