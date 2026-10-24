<template>
  <div class="op-page">
    <header class="op-topbar">
      <router-link to="/home"><Icon name="arrowLeft" size="sm" />{{ t('officialPrices.backHome') }}</router-link>
      <div class="flex items-center gap-3"><LocaleSwitcher /><button type="button" class="rounded-lg p-2" :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="toggleTheme"><Icon :name="isDark ? 'sun' : 'moon'" size="md" /></button></div>
    </header>
    <main class="op-detail">
      <div class="op-detail-heading">
        <p class="op-eyebrow"><span class="op-status-dot" />{{ t('officialPrices.eyebrow') }}</p>
        <h1>{{ t('officialPrices.title') }}</h1>
        <p class="op-muted">{{ t('officialPrices.description') }}</p>
      </div>
      <div v-if="loading" class="op-empty" role="status">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="op-empty" role="alert"><p>{{ t('officialPrices.loadError') }}</p><button type="button" class="btn btn-primary" @click="load">{{ t('officialPrices.retry') }}</button></div>
      <div v-else-if="!data?.enabled" class="op-empty">{{ t('officialPrices.disabled') }}</div>
      <div v-else-if="!activeApp || !prices.length" class="op-empty"><p>{{ t('officialPrices.empty') }}</p><p class="op-muted">{{ t('officialPrices.emptyHint') }}</p></div>
      <template v-else>
        <nav class="op-product-tabs" :aria-label="t('officialPrices.admin.apps')">
          <router-link v-for="app in availableApps" :key="app.slug" :to="{ name: 'OfficialPrices', params: { app: app.slug } }" class="op-tab" :aria-current="app.slug === activeApp.slug ? 'page' : undefined">{{ app.name }}</router-link>
        </nav>
        <div class="op-toolbar">
          <label><span class="sr-only">{{ t('officialPrices.plan') }}</span><select :value="plan" class="op-select w-full" @change="changePlan"><option v-for="name in plans" :key="name" :value="name">{{ name }}</option></select></label>
          <label><span class="sr-only">{{ t('officialPrices.search') }}</span><input v-model="search" type="search" :placeholder="t('officialPrices.search')" class="op-input w-full"></label>
          <label><span class="sr-only">{{ t('officialPrices.sort') }}</span><select v-model="descending" class="op-select w-full"><option :value="false">{{ t('officialPrices.ascending') }}</option><option :value="true">{{ t('officialPrices.descending') }}</option></select></label>
        </div>
        <div class="op-detail-summary mb-6">
          <div class="op-lowest">
            <span class="op-lowest-label"><Icon name="arrowDown" size="sm" />{{ t('officialPrices.lowest') }}</span>
            <span class="op-price"><span class="op-currency">¥</span>{{ cnyPrice(prices[0]!.price_cny, locale) }}</span>
            <span class="op-lowest-bottom">{{ countryFlag(prices[0]!.country_code) }} {{ countryName(prices[0]!.country_code, locale) }} · {{ plan }}</span>
          </div>
          <div class="op-detail-info"><span class="op-eyebrow">{{ activeApp.name }}</span><strong>{{ t('officialPrices.regions', { count: prices.length }) }}</strong><span class="op-meta">{{ t('officialPrices.synced') }} {{ priceDate(data.fetched_at, locale) }}</span><span v-if="!prices.some(price => price.country_code === 'cn')" class="op-meta">{{ t('officialPrices.domesticHint') }}</span></div>
        </div>
        <div class="op-table-wrap">
          <table class="op-table">
            <thead><tr><th>{{ t('officialPrices.country') }}</th><th>{{ t('officialPrices.original') }}</th><th>{{ t('officialPrices.converted') }}</th><th>{{ t('officialPrices.sourceUpdated') }}</th></tr></thead>
            <tbody>
              <tr v-for="price in filteredPrices" :key="price.country_code" :class="{ 'is-lowest': price.price_cny === prices[0]!.price_cny, 'is-selected': price.country_code === route.query.country }" :data-country="price.country_code">
                <td><span class="op-rank">{{ prices.indexOf(price) + 1 }}</span><span class="mr-2" aria-hidden="true">{{ countryFlag(price.country_code) }}</span>{{ countryName(price.country_code, locale) }}<span v-if="price.price_cny === prices[0]!.price_cny" class="op-badge">{{ t('officialPrices.lowestShort') }}</span></td>
                <td>{{ price.original_price }}</td><td class="font-semibold tabular-nums">¥{{ cnyPrice(price.price_cny, locale) }}</td><td class="op-meta">{{ priceDate(price.source_updated_at, locale) }}</td>
              </tr>
              <tr v-if="!filteredPrices.length"><td colspan="4" class="text-center">{{ t('officialPrices.noMatch') }}</td></tr>
            </tbody>
          </table>
        </div>
        <div class="op-footnote"><p>{{ t('officialPrices.notice') }}</p></div>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { officialPricesAPI, type OfficialPricesPublic } from '@/api/officialPrices'
import { cnyPrice, countryFlag, countryName, officialApps, preferredPricePlan, priceDate, sortedPlanPrices } from '@/utils/officialPrices'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import '@/styles/officialPrices.css'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const data = ref<OfficialPricesPublic | null>(null)
const loading = ref(true)
const error = ref(false)
const search = ref('')
const descending = ref(false)
const isDark = ref(document.documentElement.classList.contains('dark'))
const availableApps = computed(() => officialApps.filter(app => data.value?.records.some(price => price.slug === app.slug)))
const activeApp = computed(() => availableApps.value.find(app => app.slug === route.params.app))
const plans = computed(() => [...new Set(data.value?.records.filter(price => price.slug === activeApp.value?.slug).map(price => price.plan) || [])])
// URL 保存套餐选择，分享链接或刷新页面都能回到同一组地区报价。
const plan = computed(() => typeof route.query.plan === 'string' && plans.value.includes(route.query.plan)
  ? route.query.plan : preferredPricePlan(data.value?.records || [], activeApp.value?.slug || ''))
const prices = computed(() => sortedPlanPrices(data.value?.records || [], activeApp.value?.slug || '', plan.value))
// 地区链接只高亮目标行，仍保留全部地区；搜索与倒序均不改变最低价的计算范围。
const filteredPrices = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  const rows = prices.value.filter(price => !query || `${price.country_code} ${countryName(price.country_code, locale.value)}`.toLocaleLowerCase().includes(query))
  return descending.value ? [...rows].reverse() : rows
})

async function load() {
  loading.value = true
  error.value = false
  try { data.value = await officialPricesAPI.getPublic() } catch { error.value = true } finally { loading.value = false }
}

function changePlan(event: Event) {
  void router.replace({ query: { ...route.query, plan: (event.target as HTMLSelectElement).value } })
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

watch(() => route.params.app, () => { search.value = '' })
onMounted(load)
</script>
