<template>
  <section class="card overflow-hidden" aria-labelledby="official-prices-settings-title">
    <div class="border-b border-gray-200 p-6 dark:border-dark-700">
      <h2 id="official-prices-settings-title" class="text-lg font-semibold">{{ t('officialPrices.admin.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.description') }}</p>
    </div>
    <div v-if="loading" class="p-6" role="status">{{ t('common.loading') }}</div>
    <div v-else-if="!form" class="space-y-3 p-6" role="alert">
      <p>{{ t('officialPrices.loadError') }}</p>
      <button type="button" class="btn btn-secondary" @click="load">{{ t('officialPrices.retry') }}</button>
    </div>
    <div v-else class="space-y-7 p-6">
      <div class="flex items-center justify-between gap-6">
        <div><label for="official-prices-enabled" class="font-medium">{{ t('officialPrices.admin.enabled') }}</label><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.enabledHint') }}</p></div>
        <Toggle id="official-prices-enabled" v-model="form.enabled" :aria-label="t('officialPrices.admin.enabled')" :disabled="!!busy" />
      </div>
      <div class="flex items-center justify-between gap-6">
        <div><label for="official-prices-home" class="font-medium">{{ t('officialPrices.admin.homeEnabled') }}</label><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.homeHint') }}</p></div>
        <Toggle id="official-prices-home" v-model="form.home_enabled" :aria-label="t('officialPrices.admin.homeEnabled')" :disabled="!!busy || !form.enabled" />
      </div>
      <label class="block max-w-sm">
        <span class="mb-2 block font-medium">{{ t('officialPrices.admin.interval') }}</span>
        <select v-model="form.refresh_hours" class="input w-full" :disabled="!!busy">
          <option :value="5">{{ t('officialPrices.admin.hours', { count: 5 }) }}</option>
          <option :value="18">{{ t('officialPrices.admin.hours', { count: 18 }) }}</option>
          <option :value="24">{{ t('officialPrices.admin.day') }}</option>
          <option :value="168">{{ t('officialPrices.admin.week') }}</option>
        </select>
      </label>
      <fieldset :disabled="!!busy">
        <legend class="mb-3 font-medium">{{ t('officialPrices.admin.apps') }}</legend>
        <div class="flex flex-wrap gap-3">
          <label v-for="app in officialApps" :key="app.slug" class="flex cursor-pointer items-center gap-2 rounded-xl border border-gray-200 px-4 py-3 dark:border-dark-600">
            <input v-model="form.apps" type="checkbox" :value="app.slug" class="rounded text-primary-600">{{ app.name }}
          </label>
        </div>
        <p v-if="!form.apps.length" class="mt-2 text-sm text-red-600">{{ t('officialPrices.admin.emptyApps') }}</p>
      </fieldset>
      <fieldset :disabled="!!busy">
        <legend class="font-medium">{{ t('officialPrices.admin.countries') }} <span class="text-sm font-normal text-gray-500">{{ form.home_countries.length }}/8</span></legend>
        <p class="mb-3 mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.countriesHint') }}</p>
        <input v-model="search" type="search" class="input mb-3 max-w-sm" :placeholder="t('officialPrices.search')" :aria-label="t('officialPrices.search')" @keydown.enter.prevent>
        <div class="grid max-h-64 grid-cols-2 gap-2 overflow-y-auto sm:grid-cols-3 lg:grid-cols-4">
          <label v-for="code in countries" :key="code" class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2.5 text-sm dark:border-dark-600" :class="{ 'opacity-40': countryDisabled(code) }">
            <input v-model="form.home_countries" type="checkbox" :value="code" :disabled="countryDisabled(code)" class="rounded text-primary-600"><span aria-hidden="true">{{ countryFlag(code) }}</span>{{ countryName(code, locale) }}
          </label>
        </div>
        <p v-if="!countries.length" class="mt-3 text-sm text-gray-500">{{ t('officialPrices.noMatch') }}</p>
      </fieldset>
      <!-- 轮询只更新同步状态，不覆盖管理员尚未保存的选择。 -->
      <div v-if="status" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800" aria-live="polite">
        <dl class="grid gap-4 text-sm sm:grid-cols-3">
          <div><dt class="text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.lastSuccess') }}</dt><dd class="mt-1 font-medium">{{ status.last_success ? priceDate(status.last_success, locale) : t('officialPrices.admin.notSynced') }}</dd></div>
          <div><dt class="text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.nextRefresh') }}</dt><dd class="mt-1 font-medium">{{ status.syncing ? t('officialPrices.admin.syncing') : priceDate(status.next_refresh, locale) }}</dd></div>
          <div><dt class="text-gray-500 dark:text-dark-400">{{ t('officialPrices.admin.recordCount') }}</dt><dd class="mt-1 font-medium">{{ status.record_count }}</dd></div>
        </dl>
        <p v-if="status.last_error" class="mt-3 text-sm text-amber-700 dark:text-amber-400">{{ t('officialPrices.admin.lastError') }} {{ status.last_error }}</p>
        <p v-if="!status.config.enabled" class="mt-3 text-sm text-gray-500">{{ t('officialPrices.admin.defaultOff') }}</p>
      </div>
      <p v-if="feedback" role="status" :class="failed ? 'text-red-600 dark:text-red-400' : 'text-primary-600 dark:text-primary-400'">{{ feedback }}</p>
      <div class="flex flex-wrap items-center gap-3 border-t border-gray-200 pt-5 dark:border-dark-700">
        <button type="button" class="btn btn-primary" :disabled="!!busy || !form.apps.length" @click="save">{{ t('officialPrices.admin.save') }}</button>
        <button type="button" class="btn btn-secondary" :disabled="!!busy || status?.syncing || !status?.config.enabled || dirty" :title="dirty || !status?.config.enabled ? t('officialPrices.admin.saveFirst') : undefined" @click="refresh">{{ busy === 'refresh' || status?.syncing ? t('officialPrices.admin.syncing') : t('officialPrices.admin.refresh') }}</button>
        <router-link to="/home" target="_blank" rel="noopener" class="text-sm text-primary-600 dark:text-primary-400">{{ t('officialPrices.admin.preview') }} ↗</router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { officialPricesAPI, type OfficialPricesConfig, type OfficialPricesStatus } from '@/api/officialPrices'
import { countryFlag, countryName, officialApps, officialCountries, priceDate } from '@/utils/officialPrices'

const { t, locale } = useI18n()
const form = ref<OfficialPricesConfig | null>(null)
const status = ref<OfficialPricesStatus | null>(null)
const loading = ref(true)
const busy = ref<false | 'save' | 'refresh'>(false)
const feedback = ref('')
const failed = ref(false)
const search = ref('')
let poll: ReturnType<typeof setInterval> | undefined
let polling = false
let disposed = false
let statusGeneration = 0 // 保存后丢弃之前发起的轮询响应，避免旧状态覆盖新配置。
const dirty = computed(() => JSON.stringify(form.value) !== JSON.stringify(status.value?.config))
const countries = computed(() => officialCountries.filter(code => code !== 'cn' && `${code} ${countryName(code, locale.value)}`.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())))

// 中国大陆固定显示；这里仅配置额外代表地区，最多八个。
function countryDisabled(code: string): boolean {
  return !!form.value && form.value.home_countries.length >= 8 && !form.value.home_countries.includes(code)
}

function applyStatus(value: OfficialPricesStatus) {
  status.value = value
  form.value = { ...value.config, apps: [...value.config.apps], home_countries: [...value.config.home_countries] }
}

async function load() {
  loading.value = true
  try { applyStatus(await officialPricesAPI.getStatus()) } catch { form.value = null } finally { loading.value = false }
}

async function save() {
  if (!form.value || busy.value || !form.value.apps.length) return
  busy.value = 'save'
  statusGeneration++
  feedback.value = ''
  try {
    applyStatus(await officialPricesAPI.save(form.value))
    failed.value = false
    feedback.value = t('officialPrices.admin.saved')
  } catch {
    failed.value = true
    feedback.value = t('officialPrices.admin.failed')
  } finally { busy.value = false }
}

async function refresh() {
  if (busy.value || dirty.value || !status.value?.config.enabled) return
  busy.value = 'refresh'
  statusGeneration++
  feedback.value = ''
  try {
    status.value = await officialPricesAPI.refresh()
    failed.value = false
    feedback.value = t('officialPrices.admin.refreshed')
  } catch {
    failed.value = true
    feedback.value = t('officialPrices.admin.failed')
  } finally { busy.value = false }
}

async function pollStatus() {
  if (busy.value || loading.value || polling) return
  polling = true
  const generation = statusGeneration
  try {
    const value = await officialPricesAPI.getStatus()
    if (!disposed && !busy.value && generation === statusGeneration) status.value = value
  } catch { /* 临时网络失败保留最近一次状态，下轮自动重试。 */ } finally { polling = false }
}

onMounted(() => {
  void load()
  poll = setInterval(pollStatus, 15000)
})
onUnmounted(() => { disposed = true; clearInterval(poll) })
</script>
