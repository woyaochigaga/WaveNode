<template>
  <AppLayout>
    <div class="user-dashboard space-y-8">
      <div v-if="loading && !stats" class="dashboard-skeleton" aria-label="Loading dashboard">
        <div class="h-32 animate-pulse rounded-lg bg-gray-200/70 dark:bg-dark-800"></div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div v-for="index in 8" :key="index" class="h-40 animate-pulse rounded-lg bg-gray-200/70 dark:bg-dark-800"></div>
        </div>
      </div>

      <template v-else-if="stats">
        <header class="dashboard-hero">
          <div class="dashboard-hero__grid" aria-hidden="true"></div>
          <div class="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div class="max-w-2xl">
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <span class="dashboard-status" :class="isRefreshing ? 'is-refreshing' : 'is-live'">
                  <span class="dashboard-status__dot"></span>
                  {{ isRefreshing ? t('dashboard.updating') : t('dashboard.dataLive') }}
                </span>
                <span class="text-xs text-gray-500 dark:text-dark-300">
                  {{ t('dashboard.updatedAt', { time: lastUpdatedText }) }}
                </span>
              </div>
              <p class="dashboard-eyebrow">{{ t('dashboard.consoleEyebrow') }}</p>
              <h1 class="mt-1 text-2xl font-bold text-gray-950 dark:text-white sm:text-3xl">
                {{ t('dashboard.welcomeUser', { name: displayName }) }}
              </h1>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ t('dashboard.welcomeMessage') }}
              </p>
            </div>

            <button
              type="button"
              class="dashboard-refresh"
              :disabled="isRefreshing"
              :title="t('dashboard.refreshAll')"
              @click="refreshAll"
            >
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': isRefreshing }" />
              <span>{{ t('dashboard.refreshAll') }}</span>
            </button>
          </div>
        </header>

        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          :platform-quotas="platformQuotas"
        />

        <UserDashboardCharts
          v-model:start-date="startDate"
          v-model:end-date="endDate"
          v-model:granularity="granularity"
          :loading="loadingCharts"
          :trend="trendData"
          :models="modelStats"
          @date-range-change="loadCharts"
          @granularity-change="loadCharts"
          @refresh="loadCharts"
        />

        <div class="grid grid-cols-1 gap-5 xl:grid-cols-3">
          <div class="xl:col-span-2">
            <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
          </div>
          <div class="xl:col-span-1">
            <UserDashboardQuickActions />
          </div>
        </div>
      </template>

      <div v-else-if="loadError" class="dashboard-error">
        <span class="dashboard-error__icon"><Icon name="exclamationCircle" size="lg" /></span>
        <h2>{{ t('dashboard.loadErrorTitle') }}</h2>
        <p>{{ t('dashboard.loadErrorDescription') }}</p>
        <button type="button" class="btn btn-primary" @click="refreshAll">{{ t('dashboard.retry') }}</button>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { getMyPlatformQuotas } from '@/api/user'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { formatDateLocalInput } from '@/utils/format'

const authStore = useAuthStore()
const appStore = useAppStore()
const { t, locale } = useI18n()
const user = computed(() => authStore.user)
const displayName = computed(() => user.value?.username || user.value?.email || t('dashboard.defaultUser'))

const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadError = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const lastUpdatedAt = ref<Date | null>(null)

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref<'day' | 'hour'>('day')

const isRefreshing = computed(() => loading.value || loadingCharts.value || loadingUsage.value)
const lastUpdatedText = computed(() => {
  if (!lastUpdatedAt.value) return t('dashboard.justNow')
  return new Intl.DateTimeFormat(locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(lastUpdatedAt.value)
})

const loadStats = async () => {
  loading.value = true
  loadError.value = false
  try {
    const [, dashboardStats] = await Promise.all([
      authStore.refreshUser(),
      usageAPI.getDashboardStats(),
    ])
    stats.value = dashboardStats
    lastUpdatedAt.value = new Date()
  } catch (error) {
    if (!stats.value) loadError.value = true
    appStore.showError(t('dashboard.loadErrorDescription'))
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const [trendResponse, modelResponse] = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value,
      }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value }),
    ])
    trendData.value = trendResponse.trend || []
    modelStats.value = modelResponse.models || []
  } catch (error) {
    appStore.showError(t('dashboard.chartLoadError'))
    console.error('Failed to load dashboard charts:', error)
  } finally {
    loadingCharts.value = false
  }
}

const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const response = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = response.items.slice(0, 5)
  } catch (error) {
    appStore.showError(t('dashboard.recentLoadError'))
    console.error('Failed to load recent usage:', error)
  } finally {
    loadingUsage.value = false
  }
}

const loadPlatformQuotas = async () => {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (error) {
    console.warn('Failed to load platform quotas:', error)
    platformQuotas.value = []
  }
}

// 四类数据互不依赖，并行刷新可缩短用户等待时间；单项失败不会清空其他已加载内容。
const refreshAll = async () => {
  await Promise.allSettled([loadStats(), loadCharts(), loadRecent(), loadPlatformQuotas()])
}

onMounted(() => {
  void refreshAll()
})
</script>

<style scoped>
.user-dashboard {
  position: relative;
}

.dashboard-hero {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background:
    radial-gradient(circle at 82% 12%, rgba(56, 189, 248, 0.14), transparent 28%),
    radial-gradient(circle at 18% 100%, rgba(20, 184, 166, 0.16), transparent 32%),
    rgba(255, 255, 255, 0.76);
  padding: 1.5rem;
  box-shadow: 0 22px 60px -42px rgba(15, 23, 42, 0.42);
}

.dashboard-hero__grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(15, 118, 110, 0.055) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 118, 110, 0.055) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: linear-gradient(to right, black, transparent 82%);
  pointer-events: none;
}

.dashboard-eyebrow {
  color: #0f9488;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.dashboard-status {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  border: 1px solid rgba(34, 197, 94, 0.2);
  border-radius: 999px;
  background: rgba(34, 197, 94, 0.08);
  padding: 0.3rem 0.6rem;
  color: #15803d;
  font-size: 0.68rem;
  font-weight: 700;
}

.dashboard-status.is-refreshing {
  border-color: rgba(245, 158, 11, 0.25);
  background: rgba(245, 158, 11, 0.09);
  color: #b45309;
}

.dashboard-status__dot {
  width: 0.4rem;
  height: 0.4rem;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 18%, transparent);
}

.dashboard-refresh {
  display: inline-flex;
  min-height: 2.5rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid rgba(15, 118, 110, 0.25);
  border-radius: 8px;
  background: #0f172a;
  padding: 0.65rem 0.9rem;
  color: white;
  font-size: 0.8rem;
  font-weight: 700;
  transition: transform 180ms ease, background-color 180ms ease, box-shadow 180ms ease;
}

.dashboard-refresh:hover:not(:disabled) {
  background: #0f766e;
  box-shadow: 0 14px 28px -18px rgba(15, 118, 110, 0.7);
  transform: translateY(-1px);
}

.dashboard-refresh:disabled {
  cursor: wait;
  opacity: 0.65;
}

.dashboard-error {
  display: flex;
  min-height: 22rem;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  border: 1px dashed rgba(148, 163, 184, 0.4);
  border-radius: 8px;
  color: #64748b;
  text-align: center;
}

.dashboard-error__icon {
  display: flex;
  width: 3rem;
  height: 3rem;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgba(244, 63, 94, 0.1);
  color: #e11d48;
}

.dashboard-error h2 {
  color: #0f172a;
  font-size: 1rem;
  font-weight: 700;
}

.dashboard-error p {
  max-width: 28rem;
  font-size: 0.8rem;
}

:global(.dark .dashboard-hero) {
  border-color: rgba(148, 163, 184, 0.12);
  background:
    radial-gradient(circle at 82% 12%, rgba(14, 165, 233, 0.13), transparent 30%),
    radial-gradient(circle at 18% 100%, rgba(20, 184, 166, 0.12), transparent 34%),
    rgba(4, 8, 17, 0.94);
  box-shadow: 0 24px 64px -40px rgba(0, 0, 0, 0.9);
}

:global(.dark .dashboard-hero__grid) {
  background-image:
    linear-gradient(rgba(94, 234, 212, 0.055) 1px, transparent 1px),
    linear-gradient(90deg, rgba(94, 234, 212, 0.055) 1px, transparent 1px);
}

:global(.dark .dashboard-status.is-live) {
  border-color: rgba(74, 222, 128, 0.2);
  background: rgba(34, 197, 94, 0.1);
  color: #86efac;
}

:global(.dark .dashboard-status.is-refreshing) {
  color: #fcd34d;
}

:global(.dark .dashboard-refresh) {
  border-color: rgba(45, 212, 191, 0.28);
  background: rgba(15, 23, 42, 0.9);
}

:global(.dark .dashboard-error h2) {
  color: #f8fafc;
}

@media (max-width: 640px) {
  .dashboard-hero {
    padding: 1.1rem;
  }

  .dashboard-refresh {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-refresh {
    transition: none;
  }

  .dashboard-refresh:hover:not(:disabled) {
    transform: none;
  }
}
</style>
