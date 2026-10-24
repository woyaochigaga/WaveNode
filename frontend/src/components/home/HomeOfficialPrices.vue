<template>
  <section v-if="data?.enabled && data.home_enabled && cards.length" id="official-prices" class="op-section" aria-labelledby="home-prices-title">
    <div class="op-heading">
      <div>
        <p class="op-eyebrow"><span class="op-status-dot" />{{ t('officialPrices.eyebrow') }}</p>
        <h2 id="home-prices-title">{{ t('officialPrices.title') }}</h2>
        <p class="op-muted">{{ t('officialPrices.subtitle') }}</p>
      </div>
    </div>

    <div class="op-cards">
      <article v-for="card in cards" :key="card.slug" class="op-card">
        <div class="op-card-top">
          <span class="op-brand" :class="`op-brand-${card.slug}`">
            <PlatformIcon :platform="card.slug === 'chatgpt' ? 'openai' : card.slug === 'claude' ? 'anthropic' : 'grok'" size="md" />
          </span>
          <h3>{{ card.name }}</h3>
          <span class="op-meta">{{ t('officialPrices.regions', { count: card.summary.regionCount }) }}</span>
        </div>
        <label class="sr-only" :for="`home-plan-${card.slug}`">{{ card.name }} {{ t('officialPrices.plan') }}</label>
        <select :id="`home-plan-${card.slug}`" v-model="selectedPlans[card.slug]" class="op-select op-plan-select">
          <option v-for="plan in card.plans" :key="plan" :value="plan">{{ plan }}</option>
        </select>

        <!-- 最低价拥有独立强调区；整个区域是实际路由链接，键盘和触屏均可使用。 -->
        <router-link v-if="card.summary.lowest" :to="detailLink(card.slug, card.plan)" class="op-lowest" :aria-label="`${card.name} ${t('officialPrices.allRegions')}`">
          <span class="op-lowest-label"><Icon name="arrowDown" size="sm" />{{ t('officialPrices.lowest') }}</span>
          <span class="op-price"><span class="op-currency">¥</span>{{ cnyPrice(card.summary.lowest.price_cny, locale) }}</span>
          <span class="op-lowest-bottom">
            <span>{{ countryFlag(card.summary.lowest.country_code) }} {{ countryName(card.summary.lowest.country_code, locale) }}</span>
            <Icon name="arrowRight" size="md" />
          </span>
        </router-link>

        <div class="op-region-list">
          <router-link v-for="region in card.summary.regions" :key="region.code" :to="detailLink(card.slug, card.plan, region.code)" class="op-region-row" :data-country="region.code">
            <span class="op-region-name"><span aria-hidden="true">{{ countryFlag(region.code) }}</span>{{ region.code === 'cn' ? t('officialPrices.domestic') : countryName(region.code, locale) }}</span>
            <span v-if="region.price" class="op-region-amount">¥{{ cnyPrice(region.price.price_cny, locale) }}</span>
            <span v-else class="op-meta">{{ t('officialPrices.unavailable') }}</span>
          </router-link>
        </div>
        <p class="op-source-date">{{ t('officialPrices.sourceUpdated') }} {{ priceDate(card.summary.lowest?.source_updated_at || null, locale) }}</p>
        <router-link :to="detailLink(card.slug, card.plan)" class="op-card-link">{{ t('officialPrices.allRegions') }}<Icon name="arrowRight" size="sm" /></router-link>
      </article>
    </div>
    <!-- 展示本站最近一次成功同步时间，与每条报价的更新时间区分。 -->
    <div class="op-footnote"><p>{{ t('officialPrices.notice') }}</p><span>{{ t('officialPrices.lastUpdated') }} {{ priceDate(data.fetched_at, locale) }}</span></div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { officialPricesAPI, type OfficialPricesPublic } from '@/api/officialPrices'
import { cnyPrice, countryFlag, countryName, homePriceSummary, officialApps, preferredPricePlan, priceDate } from '@/utils/officialPrices'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import '@/styles/officialPrices.css'

const { t, locale } = useI18n()
const data = ref<OfficialPricesPublic | null>(null)
// 各产品独立记住所选套餐，切换一个产品时不会改变其他卡片。
const selectedPlans = reactive<Record<string, string>>({})
const cards = computed(() => officialApps.flatMap(app => {
  const records = data.value?.records || []
  const plans = [...new Set(records.filter(price => price.slug === app.slug && price.kind === 'subscription').map(price => price.plan))]
  if (!plans.length) return []
  const plan = selectedPlans[app.slug] || preferredPricePlan(records, app.slug)
  return [{ ...app, plans, plan, summary: homePriceSummary(records, app.slug, plan, data.value?.home_countries || []) }]
}))

function detailLink(slug: string, plan: string, country?: string) {
  return { name: 'OfficialPrices', params: { app: slug }, query: { plan, ...(country ? { country } : {}) } }
}

onMounted(async () => {
  try {
    data.value = await officialPricesAPI.getPublic(true)
    for (const app of officialApps) selectedPlans[app.slug] = preferredPricePlan(data.value.records, app.slug)
  } catch {
    // 价格是可选首页板块；服务不可用时隐藏板块，主站首页仍可正常访问。
    data.value = null
  }
})
</script>
