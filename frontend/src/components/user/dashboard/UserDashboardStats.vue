<template>
  <div class="space-y-8">
    <section aria-labelledby="user-primary-metrics">
      <div class="stats-heading">
        <div>
          <p class="stats-eyebrow">{{ t('dashboard.todayOverview') }}</p>
          <h2 id="user-primary-metrics">{{ t('dashboard.accountOverview') }}</h2>
        </div>
        <p>{{ t('dashboard.accountOverviewDesc') }}</p>
      </div>

      <div
        class="dashboard-metrics-grid grid grid-cols-1 sm:grid-cols-2"
        :class="isSimple ? 'xl:grid-cols-3' : 'xl:grid-cols-4'"
      >
        <DashboardMetricCard
          v-if="!isSimple"
          :label="t('dashboard.balance')"
          icon="dollar"
          tone="teal"
          :help="t('dashboard.balanceHelp')"
          featured
        >
          <template #value>${{ formatBalance(balance) }}</template>
          {{ t('dashboard.availableToSpend') }}
        </DashboardMetricCard>

        <DashboardMetricCard
          :label="t('dashboard.todayRequests')"
          :value="formatNumber(stats.today_requests || 0)"
          icon="chart"
          tone="slate"
          :help="t('dashboard.todayRequestsHelp')"
          featured
        >
          {{ t('dashboard.allTime') }} {{ formatNumber(stats.total_requests || 0) }}
        </DashboardMetricCard>

        <DashboardMetricCard
          :label="t('dashboard.todayActualCost')"
          icon="dollar"
          tone="amber"
          featured
        >
          <template #value>${{ formatCost(stats.today_actual_cost || 0) }}</template>
          <template #help>
            <p class="font-semibold text-white">{{ t('dashboard.costHelpTitle') }}</p>
            <p class="mt-1 text-gray-200">{{ t('dashboard.actualCostHelp') }}</p>
            <p class="mt-1 text-gray-200">{{ t('dashboard.standardCostHelp') }}</p>
          </template>
          <span class="metric-accent">{{ t('dashboard.allTime') }} ${{ formatCost(stats.total_actual_cost || 0) }}</span>
          <span class="metric-divider">/</span>
          <span>{{ t('dashboard.standard') }} ${{ formatCost(stats.today_cost || 0) }}</span>
        </DashboardMetricCard>

        <DashboardMetricCard
          :label="t('dashboard.performance')"
          icon="bolt"
          tone="teal"
          featured
        >
          <template #value>
            <span>{{ formatTokens(stats.rpm || 0) }}</span>
            <span class="ml-1 text-sm font-semibold text-gray-500 dark:text-dark-300">RPM</span>
          </template>
          <template #help>
            <p class="font-semibold text-white">{{ t('dashboard.rpmHelpTitle') }}</p>
            <p class="mt-1 text-gray-200">{{ t('dashboard.rpmHelp') }}</p>
            <p class="mt-2 font-semibold text-white">{{ t('dashboard.tpmHelpTitle') }}</p>
            <p class="mt-1 text-gray-200">{{ t('dashboard.tpmHelp') }}</p>
            <p class="mt-2 text-primary-200">{{ t('dashboard.performanceWindowHelp') }}</p>
          </template>
          <span class="metric-accent metric-accent--violet">{{ formatTokens(stats.tpm || 0) }} TPM</span>
        </DashboardMetricCard>
      </div>
    </section>

    <section aria-labelledby="user-usage-details">
      <div class="stats-heading">
        <div>
          <p class="stats-eyebrow">{{ t('dashboard.usageDetails') }}</p>
          <h2 id="user-usage-details">{{ t('dashboard.resourceUsage') }}</h2>
        </div>
        <p>{{ t('dashboard.resourceUsageDesc') }}</p>
      </div>

      <div class="dashboard-metrics-grid dashboard-metrics-grid--compact grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4">
        <DashboardMetricCard
          :label="t('dashboard.apiKeys')"
          :value="formatNumber(stats.total_api_keys || 0)"
          icon="key"
          tone="slate"
          :help="t('dashboard.apiKeysHelp')"
        >
          <span class="metric-accent">{{ stats.active_api_keys || 0 }} {{ t('common.active') }}</span>
        </DashboardMetricCard>

        <DashboardMetricCard
          :label="t('dashboard.todayTokens')"
          :value="formatTokens(stats.today_tokens || 0)"
          icon="cube"
          tone="slate"
          :help="t('dashboard.tokensHelp')"
        >
          {{ t('dashboard.input') }} {{ formatTokens(stats.today_input_tokens || 0) }}
          <span class="metric-divider">/</span>
          {{ t('dashboard.output') }} {{ formatTokens(stats.today_output_tokens || 0) }}
        </DashboardMetricCard>

        <DashboardMetricCard
          :label="t('dashboard.totalTokens')"
          :value="formatTokens(stats.total_tokens || 0)"
          icon="database"
          tone="slate"
          :help="t('dashboard.totalTokensHelp')"
        >
          {{ t('dashboard.cache') }}
          {{ formatTokens((stats.total_cache_creation_tokens || 0) + (stats.total_cache_read_tokens || 0)) }}
        </DashboardMetricCard>

        <DashboardMetricCard
          :label="t('dashboard.avgResponse')"
          :value="formatDuration(stats.average_duration_ms || 0)"
          icon="clock"
          tone="slate"
          :help="t('dashboard.avgResponseHelp')"
        >
          {{ t('dashboard.averageTime') }}
        </DashboardMetricCard>
      </div>
    </section>

    <section
      v-if="!isSimple && platformCards.length > 0"
      class="platform-section"
      aria-labelledby="platform-breakdown"
    >
      <div class="stats-heading mb-5">
        <div>
          <p class="stats-eyebrow">{{ t('dashboard.platformUsage') }}</p>
          <h2 id="platform-breakdown">{{ t('dashboard.platformBreakdown') }}</h2>
        </div>
        <p>{{ t('dashboard.platformBreakdownDesc') }}</p>
        <span class="platform-count">{{ t('dashboard.platformCount', { count: platformCount }) }}</span>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <article
          v-for="item in platformCards"
          :key="item.platform"
          data-testid="platform-card"
          :data-platform="item.platform"
          class="platform-card"
          :class="{ 'platform-card--other': item.isOther }"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3">
              <span class="platform-mark">{{ platformInitial(item) }}</span>
              <div class="min-w-0">
                <h3 class="truncate text-sm font-semibold text-gray-950 dark:text-white">
                  {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
                </h3>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.totalActualCost') }}</p>
              </div>
            </div>
            <span
              v-if="hasAnyLimit(item.quota) && !item.isOther"
              class="quota-state"
              :class="`quota-state--${quotaRisk(item.quota)}`"
            >
              {{ t(`dashboard.quotaState.${quotaRisk(item.quota)}`) }}
            </span>
          </div>

          <p class="mt-5 font-mono text-2xl font-bold text-gray-950 dark:text-white">
            ${{ formatCost(item.total_actual_cost) }}
          </p>

          <div class="mt-4 space-y-2 border-t border-gray-200/80 pt-4 text-xs dark:border-dark-700">
            <div class="platform-row">
              <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.todayCost') }}</span>
              <span class="font-mono text-gray-900 dark:text-white">${{ formatCost(item.today_actual_cost) }}</span>
            </div>
            <div class="platform-row">
              <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.requests') }}</span>
              <span class="font-mono text-gray-700 dark:text-gray-300">
                {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
              </span>
            </div>
            <div class="platform-row">
              <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.tokens') }}</span>
              <span class="font-mono text-gray-700 dark:text-gray-300">
                {{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}
              </span>
            </div>
          </div>

          <div
            v-if="hasAnyLimit(item.quota) && !item.isOther"
            class="mt-4 space-y-3 border-t border-gray-200/80 pt-4 dark:border-dark-700"
          >
            <div class="flex items-center gap-1">
              <p class="text-[11px] font-semibold uppercase text-gray-500 dark:text-dark-300">
                {{ t('dashboard.platformQuota.title') }}
              </p>
              <HelpTooltip :content="t('dashboard.platformQuota.help')" width-class="w-72" />
            </div>

            <template v-for="window in quotaWindows" :key="window">
              <div v-if="getQuotaLimit(item.quota, window) != null" class="space-y-1.5">
                <!-- 限额为 0 表示该时间窗口完全禁用。 -->
                <template v-if="getQuotaLimit(item.quota, window) === 0">
                  <div class="flex items-center justify-between text-xs">
                    <span class="text-gray-600 dark:text-gray-300">{{ t(`dashboard.platformQuota.${window}`) }}</span>
                    <span class="font-mono text-red-500">{{ t('dashboard.platformQuota.disabled') }}</span>
                  </div>
                  <div
                    class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700"
                    role="progressbar"
                    aria-valuenow="100"
                    aria-valuemin="0"
                    aria-valuemax="100"
                  >
                    <div class="h-full w-full rounded-full bg-red-500"></div>
                  </div>
                </template>

                <!-- 正常配额同时展示金额与百分比，避免用户自行计算。 -->
                <template v-else>
                  <div class="flex items-center justify-between text-xs">
                    <span class="text-gray-600 dark:text-gray-300">{{ t(`dashboard.platformQuota.${window}`) }}</span>
                    <span class="font-mono text-gray-700 dark:text-gray-200">
                      ${{ formatUsd(getQuotaUsage(item.quota, window)) }} /
                      ${{ formatUsd(getQuotaLimit(item.quota, window) || 0) }}
                    </span>
                  </div>
                  <div
                    class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700"
                    role="progressbar"
                    :aria-valuenow="getQuotaPercent(item.quota, window)"
                    aria-valuemin="0"
                    aria-valuemax="100"
                  >
                    <div
                      class="h-full rounded-full transition-all"
                      :class="quotaBarClass(getQuotaPercent(item.quota, window))"
                      :style="{ width: `${getQuotaPercent(item.quota, window)}%` }"
                    ></div>
                  </div>
                  <p class="flex items-center justify-between text-[10px] text-gray-400">
                    <span>{{ getQuotaPercent(item.quota, window) }}%</span>
                    <span v-if="getQuotaResetAt(item.quota, window)">
                      {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(getQuotaResetAt(item.quota, window)) }) }}
                    </span>
                  </p>
                </template>
              </div>
            </template>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DashboardMetricCard from '@/components/common/DashboardMetricCard.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import type { PlatformDashboardStats, UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()

const { t } = useI18n()
const quotaWindows = ['daily', 'weekly', 'monthly'] as const
type QuotaWindow = (typeof quotaWindows)[number]
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
}

const platformLabel = (platform: string) => PLATFORM_LABELS[platform] ?? platform
const platformInitial = (item: FusedPlatformCard) => {
  const label = item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform)
  return label.slice(0, 1).toUpperCase()
}

// 后端按平台聚合时可能无法归属少量记录，这里用“其他”补足差值，避免总额对不上。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  const byPlatform = new Map<string, PlatformDashboardStats>()
  for (const item of props.stats.by_platform ?? []) byPlatform.set(item.platform, item)

  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const quota of props.platformQuotas ?? []) byQuota.set(quota.platform, quota)

  const platforms = new Set<string>(byPlatform.keys())
  for (const [platform, quota] of byQuota) {
    if (hasAnyLimit(quota)) platforms.add(platform)
  }

  const platformOrder = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
  const cards: FusedPlatformCard[] = []

  for (const platform of platforms) {
    const platformStats = byPlatform.get(platform)
    cards.push({
      platform,
      total_actual_cost: platformStats?.total_actual_cost ?? 0,
      today_actual_cost: platformStats?.today_actual_cost ?? 0,
      total_requests: platformStats?.total_requests ?? 0,
      total_tokens: platformStats?.total_tokens ?? 0,
      quota: byQuota.get(platform),
    })
  }

  cards.sort((left, right) => {
    const leftIndex = platformOrder.indexOf(left.platform)
    const rightIndex = platformOrder.indexOf(right.platform)
    if (leftIndex === -1 && rightIndex === -1) return left.platform.localeCompare(right.platform)
    if (leftIndex === -1) return 1
    if (rightIndex === -1) return -1
    return leftIndex - rightIndex
  })

  const summedTotal = cards.reduce((sum, card) => sum + card.total_actual_cost, 0)
  const summedToday = cards.reduce((sum, card) => sum + card.today_actual_cost, 0)
  const remainingTotal = Math.max(0, props.stats.total_actual_cost - summedTotal)
  const remainingToday = Math.max(0, props.stats.today_actual_cost - summedToday)

  if (remainingTotal > OTHER_THRESHOLD || remainingToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: remainingTotal,
      today_actual_cost: remainingToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

const platformCount = computed(() => platformCards.value.filter((card) => !card.isOther).length)

function quotaVal(quota: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return quota?.[key]
}

function hasAnyLimit(quota: PlatformQuotaItem | undefined): boolean {
  if (!quota) return false
  return quota.daily_limit_usd != null || quota.weekly_limit_usd != null || quota.monthly_limit_usd != null
}

function getQuotaLimit(quota: PlatformQuotaItem | undefined, window: QuotaWindow): number | null {
  const value = quotaVal(quota, `${window}_limit_usd`)
  return typeof value === 'number' ? value : null
}

function getQuotaUsage(quota: PlatformQuotaItem | undefined, window: QuotaWindow): number {
  const value = quotaVal(quota, `${window}_usage_usd`)
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function getQuotaResetAt(quota: PlatformQuotaItem | undefined, window: QuotaWindow): string {
  const value = quotaVal(quota, `${window}_window_resets_at`)
  return typeof value === 'string' ? value : ''
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function getQuotaPercent(quota: PlatformQuotaItem | undefined, window: QuotaWindow): number {
  return calcPercent(getQuotaUsage(quota, window), getQuotaLimit(quota, window) ?? 0)
}

function quotaBarClass(percent: number): string {
  if (percent >= 95) return 'bg-red-500'
  if (percent >= 75) return 'bg-amber-500'
  return 'bg-emerald-500'
}

type QuotaRisk = 'disabled' | 'critical' | 'warning' | 'healthy'

// 卡片状态取三个时间窗口里最需要关注的一档，用户一眼就能找到配额风险。
function quotaRisk(quota: PlatformQuotaItem | undefined): QuotaRisk {
  if (!quota) return 'healthy'
  const limits = quotaWindows
    .map((window) => getQuotaLimit(quota, window))
    .filter((limit): limit is number => limit != null)
  if (limits.some((limit) => limit === 0)) return 'disabled'

  const maxPercent = quotaWindows.reduce(
    (maximum, window) => Math.max(maximum, getQuotaPercent(quota, window)),
    0,
  )
  if (maxPercent >= 95) return 'critical'
  if (maxPercent >= 75) return 'warning'
  return 'healthy'
}

const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

function formatUsd(value: number): string {
  if (!Number.isFinite(value)) return '0.00'
  return usdFormatter.format(value)
}

function formatResetTime(iso: string): string {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const formatBalance = (value: number) => usdFormatter.format(value)
const formatNumber = (value: number) => value.toLocaleString()
const formatCost = (value: number) => value.toFixed(4)
const formatTokens = (value: number) => {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return value.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>

<style scoped>
.stats-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.stats-heading h2 {
  color: #0f172a;
  font-size: 1.125rem;
  font-weight: 700;
  line-height: 1.35;
}

.stats-heading > p {
  max-width: 34rem;
  color: #64748b;
  font-size: 0.75rem;
  line-height: 1.5;
  text-align: right;
}

.stats-eyebrow {
  margin-bottom: 0.25rem;
  color: #0f9488;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.metric-accent {
  color: #0f766e;
  font-weight: 600;
}

.metric-accent--violet {
  color: #7c3aed;
}

.metric-divider {
  margin: 0 0.3rem;
  color: #cbd5e1;
}

.platform-section {
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.62);
  padding: 1.25rem;
}

.platform-count {
  flex-shrink: 0;
  border: 1px solid rgba(13, 148, 136, 0.18);
  border-radius: 999px;
  background: rgba(13, 148, 136, 0.08);
  padding: 0.3rem 0.65rem;
  color: #0f766e;
  font-size: 0.7rem;
  font-weight: 700;
}

.platform-card {
  min-width: 0;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 8px;
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.96), rgba(248, 250, 252, 0.8));
  padding: 1rem;
  transition: border-color 200ms ease, box-shadow 200ms ease, transform 200ms ease;
}

.platform-card:hover {
  border-color: rgba(13, 148, 136, 0.35);
  box-shadow: 0 18px 42px -30px rgba(13, 148, 136, 0.45);
  transform: translateY(-2px);
}

.platform-card--other {
  border-style: dashed;
}

.platform-mark {
  display: flex;
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(13, 148, 136, 0.18);
  border-radius: 8px;
  background: rgba(13, 148, 136, 0.08);
  color: #0f766e;
  font-size: 0.8rem;
  font-weight: 800;
}

.platform-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.quota-state {
  flex-shrink: 0;
  border-radius: 999px;
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
  font-weight: 700;
}

.quota-state--healthy { background: #dcfce7; color: #15803d; }
.quota-state--warning { background: #fef3c7; color: #b45309; }
.quota-state--critical,
.quota-state--disabled { background: #ffe4e6; color: #be123c; }

:global(.dark .stats-heading h2) { color: #f8fafc; }
:global(.dark .stats-heading > p) { color: #94a3b8; }
:global(.dark .metric-accent) { color: #5eead4; }
:global(.dark .metric-accent--violet) { color: #c4b5fd; }

:global(.dark .platform-section) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(4, 8, 17, 0.45);
}

:global(.dark .platform-count) {
  border-color: rgba(45, 212, 191, 0.2);
  background: rgba(20, 184, 166, 0.1);
  color: #5eead4;
}

:global(.dark .platform-card) {
  border-color: rgba(148, 163, 184, 0.12);
  background: linear-gradient(145deg, rgba(10, 15, 27, 0.98), rgba(4, 8, 17, 0.92));
}

:global(.dark .platform-mark) {
  border-color: rgba(45, 212, 191, 0.2);
  background: rgba(20, 184, 166, 0.1);
  color: #5eead4;
}

:global(.dark .quota-state--healthy) { background: rgba(34, 197, 94, 0.14); color: #86efac; }
:global(.dark .quota-state--warning) { background: rgba(245, 158, 11, 0.14); color: #fcd34d; }
:global(.dark .quota-state--critical),
:global(.dark .quota-state--disabled) { background: rgba(244, 63, 94, 0.14); color: #fda4af; }

@media (max-width: 767px) {
  .stats-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .stats-heading > p {
    text-align: left;
  }
}

@media (prefers-reduced-motion: reduce) {
  .platform-card {
    transition: none;
  }

  .platform-card:hover {
    transform: none;
  }
}
</style>
