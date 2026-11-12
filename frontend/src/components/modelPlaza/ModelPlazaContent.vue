<template>
  <div class="model-plaza-shell">
    <section class="plaza-hero relative overflow-hidden px-5 py-12 text-center sm:px-8 sm:py-16">
      <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top,_rgba(45,212,191,0.13),_transparent_65%)]" aria-hidden="true"></div>
      <div class="relative mx-auto max-w-[680px]">
        <h1 class="text-4xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-5xl">{{ t('modelPlaza.title') }}</h1>
        <p class="mt-4 text-sm text-gray-500 dark:text-dark-300">{{ t('modelPlaza.heroCount', { count: catalog.length }) }}</p>
        <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t('modelPlaza.description') }}</p>
        <label class="sr-only" for="plaza-model-search">{{ t('modelPlaza.filters.modelLabel') }}</label>
        <div class="relative mx-auto mt-7 max-w-[680px]">
          <Icon name="search" size="sm" class="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-400" />
          <input id="plaza-model-search" ref="searchInput" v-model="searchQuery" type="search" :placeholder="t('modelPlaza.filters.searchPlaceholder')" class="w-full rounded-xl border border-gray-200 bg-white py-3 pl-11 pr-14 text-sm text-gray-900 outline-none transition-colors placeholder:text-gray-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/15 dark:border-dark-600 dark:bg-dark-900 dark:text-white dark:placeholder:text-dark-400" />
          <kbd class="absolute right-4 top-1/2 hidden -translate-y-1/2 rounded border border-gray-200 px-1.5 py-0.5 text-[11px] text-gray-400 dark:border-dark-600 sm:block">⌘K</kbd>
        </div>
      </div>
    </section>

    <!-- 全局价格说明沿用后台 Markdown 配置，保持与真实计费规则同步。 -->
    <div v-if="descriptionHtml" class="plaza-description mt-5 rounded-2xl border border-primary-100 bg-primary-50/60 px-5 py-4 text-sm dark:border-primary-900/40 dark:bg-primary-950/20" v-html="descriptionHtml"></div>

    <div v-if="loading" class="flex min-h-[300px] items-center justify-center" role="status" :aria-label="t('modelPlaza.loading')"><div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600/25 border-t-primary-600"></div></div>
    <div v-else-if="error" role="alert" class="mt-6 rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
      {{ t('modelPlaza.loadFailed') }}
      <button type="button" class="ml-2 font-semibold underline" @click="emit('retry')">{{ t('common.tryAgain') }}</button>
    </div>
    <div v-else class="mt-8 grid items-start gap-5 lg:grid-cols-[280px_minmax(0,1fr)]">
      <PlazaFilterBar :platforms="platformOptions" :modes="modeOptions" :groups="groupOptions" :rates="rates" :platform="selectedPlatform" :mode="selectedMode" :group-id="selectedGroupId" :rate="selectedRate" @update:platform="selectedPlatform = $event" @update:mode="selectedMode = $event" @update:group-id="selectedGroupId = $event" @update:rate="selectedRate = $event" @reset="resetFilters" />

      <div class="min-w-0">
        <div class="flex flex-wrap items-center justify-between gap-3 rounded-[18px] border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800/80">
          <p class="text-sm text-gray-600 dark:text-dark-300"><strong class="font-mono font-semibold text-gray-950 dark:text-white">{{ filteredCatalog.length }}</strong> {{ t('modelPlaza.card.modelCount') }}</p>
          <div class="flex flex-wrap items-center gap-2 text-xs">
            <div class="inline-flex rounded-full border border-gray-200 p-0.5 dark:border-dark-600" role="group" :aria-label="t('modelPlaza.card.priceUnit')">
              <button v-for="scale in [1_000_000, 1_000] as const" :key="scale" type="button" class="rounded-full px-2.5 py-1 font-medium transition-colors" :class="tokenScale === scale ? 'bg-primary-600 text-white' : 'text-gray-500 dark:text-dark-300'" :aria-pressed="tokenScale === scale" @click="tokenScale = scale">/{{ scale === 1_000_000 ? '1M' : '1K' }}</button>
            </div>
            <label class="sr-only" for="plaza-sort">{{ t('modelPlaza.card.sortLabel') }}</label>
            <select id="plaza-sort" v-model="sortBy" class="rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-700 outline-none focus:border-primary-500 dark:border-dark-600 dark:bg-dark-900 dark:text-dark-200">
              <option value="name">{{ t('modelPlaza.card.sortName') }}</option>
              <option value="priceAsc">{{ t('modelPlaza.card.sortPriceAsc') }}</option>
              <option value="priceDesc">{{ t('modelPlaza.card.sortPriceDesc') }}</option>
            </select>
          </div>
        </div>

        <div v-if="filteredCatalog.length" class="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <PlazaModelCard v-for="model in pageModels" :key="model.key" :model="model" :token-scale="tokenScale" @details="activeKey = $event.key" />
        </div>
        <div v-else class="mt-4 rounded-[20px] border border-dashed border-gray-300 bg-white px-5 py-20 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800/60 dark:text-dark-400">
          <p>{{ catalog.length ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}</p>
          <button v-if="hasActiveFilter" type="button" class="mt-3 font-medium text-primary-600 underline dark:text-primary-300" @click="resetFilters">{{ t('modelPlaza.filters.clear') }}</button>
        </div>

        <nav v-if="totalPages > 1" class="mt-6 flex items-center justify-center gap-4 text-sm text-gray-500 dark:text-dark-300" :aria-label="t('modelPlaza.card.pagination')">
          <button type="button" class="rounded-lg border border-gray-200 px-3 py-1.5 disabled:opacity-40 dark:border-dark-600" :disabled="page === 1" @click="page--">{{ t('modelPlaza.card.previous') }}</button>
          <span class="tabular-nums">{{ t('modelPlaza.card.page', { current: page, total: totalPages }) }}</span>
          <button type="button" class="rounded-lg border border-gray-200 px-3 py-1.5 disabled:opacity-40 dark:border-dark-600" :disabled="page === totalPages" @click="page++">{{ t('modelPlaza.card.next') }}</button>
        </nav>
        <p v-if="!isAuthenticated && catalog.length" class="mt-5 text-center text-xs text-gray-400 dark:text-dark-500">{{ t('modelPlaza.anonymousHint') }}</p>
      </div>
    </div>

    <PlazaModelDetail :model="activeModel" :token-scale="tokenScale" @close="activeKey = null" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import type { ModelPlazaResponse } from '@/api/modelPlaza'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaModelCard from './PlazaModelCard.vue'
import PlazaModelDetail from './PlazaModelDetail.vue'
import { buildCatalog, priceValues } from './catalog'

const props = defineProps<{ response: ModelPlazaResponse | null; loading: boolean; error?: boolean; embedded?: boolean }>()
const emit = defineEmits<{ retry: [] }>()
const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)
const searchInput = ref<HTMLInputElement | null>(null)
const searchQuery = ref('')
const selectedPlatform = ref('all')
const selectedMode = ref('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const sortBy = ref<'name' | 'priceAsc' | 'priceDesc'>('name')
const tokenScale = ref<1_000_000 | 1_000>(1_000_000)
const page = ref(1)
const activeKey = ref<string | null>(null)
const PAGE_SIZE = 18

const catalog = computed(() => buildCatalog(props.response?.groups ?? []))
const descriptionHtml = computed(() => {
  const markdown = props.response?.description?.trim()
  return markdown ? DOMPurify.sanitize(marked.parse(markdown) as string) : ''
})
const groupOptions = computed(() => (props.response?.groups ?? []).map(group => ({ id: group.id, name: group.name })))
const rates = computed(() => [...new Set((props.response?.groups ?? []).map(group => group.user_rate_multiplier ?? group.rate_multiplier))].sort((a, b) => a - b))
const platformOptions = computed(() => [
  { value: 'all', count: catalog.value.length },
  ...[...new Set(catalog.value.map(model => model.platform))].sort().map(value => ({ value, count: catalog.value.filter(model => model.platform === value).length }))
])
const modeOptions = computed(() => [
  { value: 'all', count: catalog.value.length },
  ...[...new Set(catalog.value.map(model => model.mode))].map(value => ({ value, count: catalog.value.filter(model => model.mode === value).length }))
])

/** 先筛选分组报价再合并同名模型，卡片最低价与详情始终基于当前可见分组。 */
const filteredCatalog = computed(() => {
  const groups = (props.response?.groups ?? [])
    .filter(group => selectedGroupId.value === 'all' || group.id === selectedGroupId.value)
    .filter(group => selectedRate.value === 'all' || (group.user_rate_multiplier ?? group.rate_multiplier) === selectedRate.value)
  const query = searchQuery.value.trim().toLowerCase()
  return buildCatalog(groups)
    .filter(model => selectedPlatform.value === 'all' || model.platform === selectedPlatform.value)
    .filter(model => selectedMode.value === 'all' || model.mode === selectedMode.value)
    .filter(model => !query || `${model.name} ${model.platform} ${model.offerings.map(offering => offering.group.name).join(' ')}`.toLowerCase().includes(query))
    .sort((left, right) => {
      if (sortBy.value === 'name') return left.name.localeCompare(right.name)
      // 按次/按图与 Token 单位不可直接比价，先按计费方式分组再按金额排序。
      if (left.mode !== right.mode) return left.mode.localeCompare(right.mode)
      const leftPrice = priceValues(left.best).primary
      const rightPrice = priceValues(right.best).primary
      if (leftPrice == null) return rightPrice == null ? left.name.localeCompare(right.name) : 1
      if (rightPrice == null) return -1
      return (sortBy.value === 'priceAsc' ? leftPrice - rightPrice : rightPrice - leftPrice) || left.name.localeCompare(right.name)
    })
})
const totalPages = computed(() => Math.ceil(filteredCatalog.value.length / PAGE_SIZE))
const pageModels = computed(() => filteredCatalog.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))
const activeModel = computed(() => filteredCatalog.value.find(model => model.key === activeKey.value) ?? null)
const hasActiveFilter = computed(() => searchQuery.value.trim() !== '' || selectedPlatform.value !== 'all' || selectedMode.value !== 'all' || selectedGroupId.value !== 'all' || selectedRate.value !== 'all')

watch([searchQuery, selectedPlatform, selectedMode, selectedGroupId, selectedRate, sortBy], () => { page.value = 1 })
watch(rates, values => { if (selectedRate.value !== 'all' && !values.includes(selectedRate.value)) selectedRate.value = 'all' })
watch(groupOptions, groups => { if (selectedGroupId.value !== 'all' && !groups.some(group => group.id === selectedGroupId.value)) selectedGroupId.value = 'all' })
watch(totalPages, total => { if (page.value > total) page.value = Math.max(1, total) })

function resetFilters() {
  searchQuery.value = ''
  selectedPlatform.value = 'all'
  selectedMode.value = 'all'
  selectedGroupId.value = 'all'
  selectedRate.value = 'all'
}

/** 与参考站一致，键盘快捷键只在未编辑其他输入框时聚焦搜索。 */
function onShortcut(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    searchInput.value?.focus()
  }
}
onMounted(() => window.addEventListener('keydown', onShortcut))
onBeforeUnmount(() => window.removeEventListener('keydown', onShortcut))
</script>

<style scoped>
.plaza-description { line-height: 1.7; overflow-wrap: anywhere; }
.plaza-description :deep(h1), .plaza-description :deep(h2), .plaza-description :deep(h3) { @apply mb-2 mt-3 font-semibold text-gray-900 first:mt-0 dark:text-white; }
.plaza-description :deep(p) { @apply mb-2 text-gray-700 last:mb-0 dark:text-dark-200; }
.plaza-description :deep(a) { @apply text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300; }
.plaza-description :deep(ul) { @apply mb-2 list-disc pl-5; }
.plaza-description :deep(ol) { @apply mb-2 list-decimal pl-5; }
</style>
