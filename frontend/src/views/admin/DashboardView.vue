<template>
  <AppLayout>
    <div class="admin-dashboard space-y-8">
      <div v-if="loading && !stats" class="dashboard-skeleton" aria-label="Loading dashboard">
        <div class="h-28 animate-pulse rounded-lg bg-gray-200/70 dark:bg-dark-800"></div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div v-for="index in 8" :key="index" class="h-40 animate-pulse rounded-lg bg-gray-200/70 dark:bg-dark-800"></div>
        </div>
      </div>

      <template v-else-if="stats">
        <div class="dashboard-topbar">
          <div class="flex min-w-0 flex-wrap items-center gap-2.5">
            <span class="dashboard-status" :class="stats.stats_stale ? 'is-stale' : 'is-live'">
              <span class="dashboard-status__dot"></span>
              {{ stats.stats_stale ? t('admin.dashboard.dataStale') : t('admin.dashboard.dataLive') }}
            </span>
            <span class="text-xs text-gray-500 dark:text-dark-300">
              {{ t('admin.dashboard.updatedAt', { time: statsUpdatedText }) }}
            </span>
          </div>
          <button
            type="button"
            class="dashboard-refresh"
            :disabled="loading || chartsLoading"
            :title="t('admin.dashboard.refreshAll')"
            @click="loadDashboardStats"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading || chartsLoading }" />
            <span>{{ t('admin.dashboard.refreshAll') }}</span>
          </button>
        </div>

        <section aria-labelledby="admin-primary-metrics">
          <div class="dashboard-section-heading">
            <h2 id="admin-primary-metrics">{{ t('admin.dashboard.primaryMetrics') }}</h2>
            <p>{{ t('admin.dashboard.primaryMetricsDesc') }}</p>
          </div>

          <div class="dashboard-metrics-grid grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4">
            <DashboardMetricCard
              :label="t('admin.dashboard.todayRequests')"
              :value="formatNumber(stats.today_requests)"
              icon="chart"
              tone="teal"
              :help="t('admin.dashboard.todayRequestsHelp')"
            >
              {{ t('admin.dashboard.totalRequests') }}: {{ formatNumber(stats.total_requests) }}
            </DashboardMetricCard>

            <DashboardMetricCard
              :label="t('admin.dashboard.todayActualSpend')"
              icon="dollar"
              tone="amber"
            >
              <template #value>${{ formatCost(stats.today_actual_cost) }}</template>
              <template #help>
                <p class="font-semibold text-white">{{ t('admin.dashboard.costHelpTitle') }}</p>
                <p class="mt-1 text-gray-200">{{ t('admin.dashboard.costHelpActual') }}</p>
                <p class="mt-1 text-gray-200">{{ t('admin.dashboard.costHelpAccount') }}</p>
                <p class="mt-1 text-gray-200">{{ t('admin.dashboard.costHelpStandard') }}</p>
              </template>
              <span class="metric-detail metric-detail--amber">
                {{ t('admin.dashboard.accountCost') }} ${{ formatCost(stats.today_account_cost) }}
              </span>
              <span class="metric-divider">/</span>
              <span>{{ t('admin.dashboard.standard') }} ${{ formatCost(stats.today_cost) }}</span>
            </DashboardMetricCard>

            <DashboardMetricCard
              :label="t('admin.dashboard.performance')"
              icon="bolt"
              tone="teal"
            >
              <template #value>
                <span>{{ formatTokens(stats.rpm) }}</span>
                <span class="ml-1 text-sm font-semibold text-gray-500 dark:text-dark-300">RPM</span>
              </template>
              <template #help>
                <p class="font-semibold text-white">{{ t('admin.dashboard.rpmHelpTitle') }}</p>
                <p class="mt-1 text-gray-200">{{ t('admin.dashboard.rpmHelp') }}</p>
                <p class="mt-2 font-semibold text-white">{{ t('admin.dashboard.tpmHelpTitle') }}</p>
                <p class="mt-1 text-gray-200">{{ t('admin.dashboard.tpmHelp') }}</p>
                <p class="mt-2 text-primary-200">{{ t('admin.dashboard.performanceWindowHelp') }}</p>
              </template>
              <span class="metric-detail metric-detail--violet">{{ formatTokens(stats.tpm) }} TPM</span>
            </DashboardMetricCard>

            <DashboardMetricCard
              :label="t('admin.dashboard.avgResponse')"
              :value="formatDuration(stats.average_duration_ms)"
              icon="clock"
              tone="slate"
              :help="t('admin.dashboard.avgResponseHelp')"
            >
              {{ formatNumber(stats.active_users) }} {{ t('admin.dashboard.activeUsers') }}
            </DashboardMetricCard>
          </div>
        </section>

        <section aria-labelledby="admin-system-health">
          <div class="dashboard-section-heading">
            <h2 id="admin-system-health">{{ t('admin.dashboard.systemHealth') }}</h2>
            <p>{{ t('admin.dashboard.systemHealthDesc') }}</p>
          </div>

          <div class="dashboard-metrics-grid dashboard-metrics-grid--compact grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4">
            <DashboardMetricCard
              :label="t('admin.dashboard.apiKeys')"
              :value="formatNumber(stats.total_api_keys)"
              icon="key"
              tone="slate"
              :help="t('admin.dashboard.apiKeysHelp')"
            >
              <span class="metric-detail metric-detail--ok">{{ stats.active_api_keys }} {{ t('common.active') }}</span>
            </DashboardMetricCard>

            <DashboardMetricCard
              :label="t('admin.dashboard.accounts')"
              :value="formatNumber(stats.total_accounts)"
              icon="server"
              tone="slate"
              :help="t('admin.dashboard.accountsHelp')"
            >
              <template #status>
                <span v-if="stats.error_accounts > 0" class="metric-status metric-status--danger">
                  {{ stats.error_accounts }} {{ t('common.error') }}
                </span>
                <span v-else class="metric-status metric-status--ok">{{ t('admin.dashboard.healthy') }}</span>
              </template>
              <span class="metric-detail metric-detail--ok">{{ stats.normal_accounts }} {{ t('common.active') }}</span>
              <span v-if="stats.ratelimit_accounts > 0" class="ml-2 text-amber-600 dark:text-amber-400">
                {{ stats.ratelimit_accounts }} {{ t('admin.dashboard.rateLimited') }}
              </span>
            </DashboardMetricCard>

            <DashboardMetricCard
              :label="t('admin.dashboard.users')"
              :value="formatNumber(stats.total_users)"
              icon="users"
              tone="slate"
              :help="t('admin.dashboard.usersHelp')"
            >
              <span class="metric-detail metric-detail--ok">+{{ stats.today_new_users }} {{ t('admin.dashboard.newToday') }}</span>
              <span class="ml-2">{{ stats.hourly_active_users }} {{ t('admin.dashboard.activeThisHour') }}</span>
            </DashboardMetricCard>

            <DashboardMetricCard
              :label="t('admin.dashboard.totalTokens')"
              :value="formatTokens(stats.total_tokens)"
              icon="database"
              tone="slate"
              :help="t('admin.dashboard.totalTokensHelp')"
            >
              <span>{{ t('admin.dashboard.todayTokens') }} {{ formatTokens(stats.today_tokens) }}</span>
              <span class="metric-divider">/</span>
              <span>{{ t('admin.dashboard.uptime') }} {{ formatUptime(stats.uptime) }}</span>
            </DashboardMetricCard>
          </div>
        </section>

        <section class="dashboard-action-band" aria-labelledby="admin-quick-actions">
          <div class="dashboard-section-heading mb-4">
            <h2 id="admin-quick-actions">{{ t('admin.dashboard.quickActions') }}</h2>
            <p>{{ t('admin.dashboard.quickActionsDesc') }}</p>
          </div>
          <div class="dashboard-actions-grid grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-5">
            <button
              v-for="action in quickActions"
              :key="action.path"
              v-show="!action.requiresBatchImage || canUseBatchImage"
              type="button"
              class="dashboard-action"
              @click="router.push(action.path)"
            >
              <span class="dashboard-action__icon"><Icon :name="action.icon" size="md" :stroke-width="2" /></span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ t(action.labelKey) }}</span>
                <span class="mt-0.5 block text-xs leading-5 text-gray-500 dark:text-dark-300">{{ t(action.descriptionKey) }}</span>
              </span>
              <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" />
            </button>
          </div>
        </section>

        <section class="space-y-5" aria-labelledby="admin-analytics">
          <div class="dashboard-section-heading">
            <h2 id="admin-analytics">{{ t('admin.dashboard.analyticsTitle') }}</h2>
            <p>{{ t('admin.dashboard.analyticsDesc') }}</p>
          </div>

          <div class="dashboard-toolbar">
            <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
              <span class="dashboard-toolbar__label">{{ t('admin.dashboard.timeRange') }}</span>
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="onDateRangeChange"
              />
              <span class="dashboard-range-chip">{{ startDate }} - {{ endDate }}</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="dashboard-toolbar__label hidden sm:inline">{{ t('admin.dashboard.granularity') }}</span>
              <div class="w-28">
                <Select v-model="granularity" :options="granularityOptions" @change="loadChartData" />
              </div>
              <button
                type="button"
                class="dashboard-icon-button"
                :disabled="chartsLoading"
                :title="t('common.refresh')"
                @click="loadChartData"
              >
                <Icon name="refresh" size="sm" :class="{ 'animate-spin': chartsLoading }" />
              </button>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
            <ModelDistributionChart
              :model-stats="modelStats"
              :enable-ranking-view="true"
              :ranking-items="rankingItems"
              :ranking-total-actual-cost="rankingTotalActualCost"
              :ranking-total-requests="rankingTotalRequests"
              :ranking-total-tokens="rankingTotalTokens"
              :loading="chartsLoading"
              :ranking-loading="rankingLoading"
              :ranking-error="rankingError"
              :start-date="startDate"
              :end-date="endDate"
              @ranking-click="goToUserUsage"
            />
            <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
          </div>

          <div class="dashboard-chart-panel">
            <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.userUsageTrend') }}</h3>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ t('admin.dashboard.userUsageTrendDesc') }}</p>
              </div>
              <span class="dashboard-range-chip">TOP 12</span>
            </div>
            <div class="h-72">
              <div v-if="userTrendLoading" class="flex h-full items-center justify-center"><LoadingSpinner size="md" /></div>
              <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
              <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.noDataAvailable') }}
              </div>
            </div>
          </div>
        </section>
      </template>

      <div v-else-if="loadError" class="dashboard-error">
        <Icon name="exclamationCircle" size="lg" />
        <h2>{{ t('admin.dashboard.loadErrorTitle') }}</h2>
        <p>{{ t('admin.dashboard.failedToLoad') }}</p>
        <button type="button" class="btn btn-primary" @click="loadDashboardStats">{{ t('admin.dashboard.retry') }}</button>
      </div>
      <div v-else class="dashboard-error">
        <Icon name="database" size="lg" />
        <h2>{{ t('admin.dashboard.noDataAvailable') }}</h2>
        <button type="button" class="btn btn-primary" @click="loadDashboardStats">{{ t('admin.dashboard.retry') }}</button>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Line } from 'vue-chartjs'
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import DashboardMetricCard from '@/components/common/DashboardMetricCard.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const appStore = useAppStore()
const router = useRouter()
const { t, locale } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const loadError = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12
let themeObserver: MutationObserver | null = null

// 快捷入口集中配置，保持权限判断和路由行为清晰可维护。
const quickActions = [
  { path: '/admin/users', icon: 'users', labelKey: 'admin.dashboard.manageUsers', descriptionKey: 'admin.dashboard.viewUserAccounts', requiresBatchImage: false },
  { path: '/admin/accounts', icon: 'server', labelKey: 'admin.dashboard.manageAccounts', descriptionKey: 'admin.dashboard.configureAiAccounts', requiresBatchImage: false },
  { path: '/admin/groups', icon: 'grid', labelKey: 'admin.dashboard.groupPricing', descriptionKey: 'admin.dashboard.groupPricingDesc', requiresBatchImage: false },
  { path: '/admin/settings', icon: 'cog', labelKey: 'admin.dashboard.systemSettings', descriptionKey: 'admin.dashboard.configureSystem', requiresBatchImage: false },
  { path: '/batch-image', icon: 'sparkles', labelKey: 'admin.dashboard.batchImage', descriptionKey: 'admin.dashboard.batchImageDesc', requiresBatchImage: true },
] as const

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for Select component
const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

// 监听根节点主题类，确保已渲染图表切换主题时同步更新颜色。
const isDarkMode = ref(document.documentElement.classList.contains('dark'))

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

const statsUpdatedText = computed(() => {
  const raw = stats.value?.stats_updated_at
  if (!raw) return t('admin.dashboard.justNow')
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return t('admin.dashboard.justNow')
  return new Intl.DateTimeFormat(locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
})

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, point.tokens)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#3b82f6',
    '#10b981',
    '#f59e0b',
    '#ef4444',
    '#8b5cf6',
    '#ec4899',
    '#14b8a6',
    '#f97316',
    '#6366f1',
    '#84cc16',
    '#06b6d4',
    '#a855f7'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

// 运行时长用最大两级单位展示，避免秒数过大而难以快速理解。
const formatUptime = (seconds: number): string => {
  const safeSeconds = Math.max(0, Math.floor(toFiniteNumber(seconds)))
  const days = Math.floor(safeSeconds / 86400)
  const hours = Math.floor((safeSeconds % 86400) / 3600)
  if (days > 0) return t('admin.dashboard.uptimeDaysHours', { days, hours })
  const minutes = Math.floor((safeSeconds % 3600) / 60)
  if (hours > 0) return t('admin.dashboard.uptimeHoursMinutes', { hours, minutes })
  return t('admin.dashboard.uptimeMinutes', { minutes })
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  if (includeStats) loadError.value = false
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
      loadError.value = false
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && !stats.value) loadError.value = true
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  void refreshBatchImageAccess()
  void loadDashboardStats()
  themeObserver = new MutationObserver(() => {
    isDarkMode.value = document.documentElement.classList.contains('dark')
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})
</script>

<style scoped>
.dashboard-topbar {
  display: flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  padding: 0 0.1rem 0.85rem;
}

.dashboard-status,
.metric-status,
.dashboard-range-chip {
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.68);
  padding: 0.25rem 0.55rem;
  color: #475569;
  font-size: 0.7rem;
  font-weight: 600;
  line-height: 1;
}

.dashboard-status__dot {
  width: 0.42rem;
  height: 0.42rem;
  margin-right: 0.4rem;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 16%, transparent);
}

.dashboard-status.is-live { color: #0f766e; }
.dashboard-status.is-stale { color: #b45309; }

.dashboard-refresh,
.dashboard-icon-button {
  display: inline-flex;
  min-height: 2.5rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.7);
  padding: 0.55rem 0.85rem;
  color: #334155;
  font-size: 0.8rem;
  font-weight: 600;
  transition: transform 180ms ease, background-color 180ms ease, opacity 180ms ease;
}

.dashboard-refresh:hover:not(:disabled),
.dashboard-icon-button:hover:not(:disabled) {
  border-color: rgba(13, 148, 136, 0.28);
  background: rgba(240, 253, 250, 0.82);
  color: #0f766e;
}

.dashboard-refresh:disabled,
.dashboard-icon-button:disabled { cursor: not-allowed; opacity: 0.55; }
.dashboard-icon-button { width: 2.5rem; padding: 0; }

.dashboard-section-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.dashboard-section-heading h2 {
  color: #0f172a;
  font-size: 1.05rem;
  font-weight: 700;
}

.dashboard-section-heading > p {
  max-width: 34rem;
  color: #64748b;
  font-size: 0.78rem;
  line-height: 1.55;
  text-align: right;
}

.metric-divider { margin-inline: 0.35rem; color: #cbd5e1; }
.metric-detail { font-weight: 600; }
.metric-detail--ok { color: #0f766e; }
.metric-detail--amber { color: #b45309; }
.metric-detail--violet { color: #7c3aed; }
.metric-status { background: rgba(248, 250, 252, 0.9); }
.metric-status--ok { border-color: rgba(13, 148, 136, 0.2); color: #0f766e; }
.metric-status--danger { border-color: rgba(225, 29, 72, 0.22); color: #be123c; }

.dashboard-action-band {
  padding-block: 0.25rem;
}

.dashboard-actions-grid {
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.22);
}

.dashboard-action {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
  border: 0;
  border-radius: 0;
  background: rgba(255, 255, 255, 0.94);
  padding: 0.95rem 1rem;
  text-align: left;
  transition: border-color 180ms ease, background-color 180ms ease, transform 180ms ease;
}

.dashboard-action:hover {
  background: rgba(240, 253, 250, 0.92);
}

.dashboard-action__icon {
  display: flex;
  width: 1.5rem;
  height: 1.5rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  color: #0f766e;
}

.dashboard-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.68);
  padding: 0.75rem;
  backdrop-filter: blur(12px);
}

.dashboard-toolbar__label {
  color: #475569;
  font-size: 0.75rem;
  font-weight: 600;
}

.dashboard-chart-panel {
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.78);
  padding: 1rem;
  box-shadow: 0 14px 38px -32px rgba(15, 23, 42, 0.38);
}

.dashboard-error {
  display: flex;
  min-height: 22rem;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: #64748b;
  text-align: center;
}

.dashboard-error h2 { color: #0f172a; font-size: 1.1rem; font-weight: 700; }

:global(.dark .dashboard-status),
:global(.dark .metric-status),
:global(.dark .dashboard-range-chip) {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.78);
}

:global(.dark .dashboard-section-heading h2),
:global(.dark .dashboard-error h2) { color: #f8fafc; }
:global(.dark .dashboard-section-heading > p) { color: #94a3b8; }
:global(.dark .metric-detail--ok) { color: #5eead4; }
:global(.dark .metric-detail--amber) { color: #fbbf24; }
:global(.dark .metric-detail--violet) { color: #c4b5fd; }

:global(.dark .dashboard-action),
:global(.dark .dashboard-toolbar),
:global(.dark .dashboard-chart-panel) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(6, 10, 20, 0.86);
}

:global(.dark .dashboard-topbar) {
  border-color: rgba(148, 163, 184, 0.12);
}

:global(.dark .dashboard-refresh),
:global(.dark .dashboard-icon-button) {
  border-color: rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.7);
  color: #cbd5e1;
}

:global(.dark .dashboard-actions-grid) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(148, 163, 184, 0.12);
}

:global(.dark .dashboard-action:hover) {
  background: rgba(13, 148, 136, 0.08);
}

:global(.dark .dashboard-toolbar__label) { color: #cbd5e1; }

@media (max-width: 767px) {
  .dashboard-section-heading,
  .dashboard-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .dashboard-section-heading > p { text-align: left; }
  .dashboard-range-chip { display: none; }

  .dashboard-topbar {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-refresh,
  .dashboard-icon-button,
  .dashboard-action { transition: none; }
  .dashboard-refresh:hover,
  .dashboard-icon-button:hover,
  .dashboard-action:hover { transform: none; }
}
</style>
