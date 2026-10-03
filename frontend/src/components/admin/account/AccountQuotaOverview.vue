<template>
  <section class="space-y-4" aria-labelledby="account-quota-title">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 id="account-quota-title" class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.stats.quota.title') }}
        </h3>
        <p v-if="usage?.updated_at" class="mt-1 text-xs text-gray-400">
          {{ t('admin.accounts.stats.quota.updatedAt', { time: formatDateTime(usage.updated_at) }) }}
        </p>
      </div>
      <button v-if="refreshable" type="button" class="btn btn-secondary" :disabled="loading" @click="emit('refresh')">
        <Icon name="refresh" size="sm" :class="['mr-1.5', { 'animate-spin': loading }]" />
        {{ loading ? t('admin.accounts.stats.quota.refreshing') : t('admin.accounts.stats.quota.refresh') }}
      </button>
    </div>

    <div v-if="error" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-300">
      {{ error }}
    </div>

    <template v-if="items.length">
      <div v-for="group in visibleGroups" :key="group.key" class="space-y-2">
        <div class="flex items-center gap-2">
          <Icon :name="group.icon" size="sm" class="text-gray-400" />
          <h4 class="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">{{ group.label }}</h4>
          <span class="text-xs text-gray-400">{{ group.items.length }}</span>
        </div>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          <article v-for="item in group.items" :key="item.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <div class="flex items-start justify-between gap-3">
              <h5 class="min-w-0 break-words text-sm font-medium text-gray-800 dark:text-gray-200">
                {{ item.labelKey ? t(item.labelKey) : item.label }}
              </h5>
              <span v-if="item.resetAt" class="shrink-0 text-[11px] text-gray-400" :title="formatDateTime(item.resetAt)">
                {{ formatReset(item.resetAt) }}
              </span>
            </div>

            <div class="mt-2 flex items-end justify-between gap-3">
              <div>
                <div class="text-xs text-gray-400">{{ t('admin.accounts.stats.quota.remaining') }}</div>
                <div class="mt-0.5 text-lg font-semibold text-gray-900 dark:text-white">{{ formatRemaining(item) }}</div>
              </div>
              <div v-if="item.limit != null" class="text-right text-xs text-gray-500 dark:text-gray-400">
                {{ formatAmount(item.used ?? 0, item.unit) }} / {{ formatAmount(item.limit, item.unit) }}
              </div>
            </div>

            <div v-if="item.utilization != null" class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <div
                :class="['h-full rounded-full', utilizationTone(item.utilization)]"
                :style="{ width: `${Math.min(100, Math.max(0, item.utilization))}%` }"
              ></div>
            </div>
            <div class="mt-1.5 flex flex-wrap justify-between gap-2 text-[11px] text-gray-400">
              <span v-if="item.utilization != null">{{ t('admin.accounts.stats.quota.usedPercent', { percent: formatPercent(item.utilization) }) }}</span>
              <span v-if="item.estimatedRemainingCost != null" class="text-primary-600 dark:text-primary-400">
                {{ t('admin.accounts.stats.quota.estimatedRemaining', { amount: formatMoney(item.estimatedRemainingCost) }) }}
              </span>
            </div>
          </article>
        </div>
      </div>
    </template>

    <div v-else-if="loading" class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <div v-for="index in 3" :key="index" class="h-28 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
    </div>
    <div v-else class="rounded-lg border border-dashed border-gray-200 py-8 text-center text-sm text-gray-400 dark:border-dark-700">
      {{ t('admin.accounts.stats.quota.empty') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { buildAccountQuotaItems, type AccountQuotaItem, type AccountQuotaUnit } from './accountQuotaItems'
import type { Account, AccountUsageInfo } from '@/types'

const props = defineProps<{
  account: Account
  usage: AccountUsageInfo | null
  loading: boolean
  error: string
  refreshable: boolean
}>()

const emit = defineEmits<{ refresh: [] }>()
const { t, locale } = useI18n()

const items = computed(() => buildAccountQuotaItems(props.account, props.usage))
const visibleGroups = computed(() => {
  const groups = [
    { key: 'local' as const, label: t('admin.accounts.stats.quota.localLimits'), icon: 'dollar' as const },
    { key: 'upstream' as const, label: t('admin.accounts.stats.quota.upstreamLimits'), icon: 'cloud' as const },
    { key: 'capacity' as const, label: t('admin.accounts.stats.quota.capacityLimits'), icon: 'bolt' as const }
  ]
  return groups
    .map((group) => ({ ...group, items: items.value.filter((item) => item.group === group.key) }))
    .filter((group) => group.items.length > 0)
})

function formatNumber(value: number): string {
  return new Intl.NumberFormat(locale.value, { maximumFractionDigits: 2 }).format(value)
}

function formatMoney(value: number): string {
  return `$${new Intl.NumberFormat(locale.value, { maximumFractionDigits: 4 }).format(value)}`
}

function formatAmount(value: number, unit: AccountQuotaUnit): string {
  if (unit === 'usd') return formatMoney(value)
  if (unit === 'percent') return `${formatPercent(value)}%`
  if (unit === 'tokens' && Math.abs(value) >= 1000) {
    return new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 2 }).format(value)
  }
  return formatNumber(value)
}

function formatRemaining(item: AccountQuotaItem): string {
  if (item.unlimited) return t('admin.accounts.stats.quota.unlimited')
  if (item.remaining != null) return formatAmount(Math.max(0, item.remaining), item.unit)
  if (item.utilization != null) return `${formatPercent(Math.max(0, 100 - item.utilization))}%`
  return '—'
}

function formatPercent(value: number): string {
  return new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 }).format(value)
}

function formatDateTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(locale.value)
}

function formatReset(value: string): string {
  const reset = new Date(value)
  if (Number.isNaN(reset.getTime())) return value
  const remaining = reset.getTime() - Date.now()
  if (remaining <= 0) return t('admin.accounts.stats.quota.resetNow')
  const minutes = Math.ceil(remaining / 60000)
  if (minutes < 60) return t('admin.accounts.stats.quota.resetMinutes', { minutes })
  const hours = Math.ceil(minutes / 60)
  if (hours < 48) return t('admin.accounts.stats.quota.resetHours', { hours })
  return reset.toLocaleDateString(locale.value)
}

function utilizationTone(value: number): string {
  if (value >= 90) return 'bg-red-500'
  if (value >= 70) return 'bg-amber-500'
  return 'bg-primary-500'
}
</script>
