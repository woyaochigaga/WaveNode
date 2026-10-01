<template>
  <section class="space-y-5" aria-labelledby="user-analytics">
    <div class="chart-heading">
      <div>
        <p class="chart-eyebrow">{{ t('dashboard.analyticsEyebrow') }}</p>
        <h2 id="user-analytics">{{ t('dashboard.analyticsTitle') }}</h2>
      </div>
      <p>{{ t('dashboard.analyticsDesc') }}</p>
    </div>

    <div class="chart-toolbar">
      <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
        <span class="toolbar-label">{{ t('dashboard.timeRange') }}</span>
        <DateRangePicker
          :start-date="startDate"
          :end-date="endDate"
          @update:start-date="$emit('update:startDate', $event)"
          @update:end-date="$emit('update:endDate', $event)"
          @change="$emit('dateRangeChange', $event)"
        />
        <span class="range-chip">{{ startDate }} - {{ endDate }}</span>
      </div>

      <div class="flex items-center gap-2">
        <span class="toolbar-label hidden sm:inline">{{ t('dashboard.granularity') }}</span>
        <div class="w-28">
          <Select
            :model-value="granularity"
            :options="granularityOptions"
            @update:model-value="onGranularityUpdate"
            @change="$emit('granularityChange')"
          />
        </div>
        <button
          type="button"
          class="toolbar-button"
          :disabled="loading"
          :title="t('common.refresh')"
          @click="$emit('refresh')"
        >
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          <span class="sr-only">{{ t('common.refresh') }}</span>
        </button>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
      <article class="chart-panel">
        <div class="mb-5 flex items-start justify-between gap-3">
          <div>
            <h3>{{ t('dashboard.modelDistribution') }}</h3>
            <p>{{ t('dashboard.modelDistributionDesc') }}</p>
          </div>
          <span class="range-chip">{{ t('dashboard.modelCount', { count: models.length }) }}</span>
        </div>

        <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/65 backdrop-blur-sm dark:bg-dark-900/65">
          <LoadingSpinner size="md" />
        </div>

        <div v-if="modelData" class="flex min-h-64 flex-col items-center gap-5 md:flex-row md:items-start">
          <div class="relative h-48 w-48 shrink-0">
            <Doughnut :data="modelData" :options="doughnutOptions" />
            <div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
              <span class="text-2xl font-bold text-gray-950 dark:text-white">{{ models.length }}</span>
              <span class="text-[10px] font-semibold uppercase text-gray-400">{{ t('dashboard.modelsUsed') }}</span>
            </div>
          </div>

          <div class="max-h-64 w-full min-w-0 flex-1 overflow-auto">
            <table class="w-full text-xs">
              <thead class="sticky top-0 bg-white/95 text-gray-500 backdrop-blur dark:bg-dark-900/95 dark:text-gray-400">
                <tr>
                  <th class="pb-2 text-left font-semibold">{{ t('dashboard.model') }}</th>
                  <th class="pb-2 text-right font-semibold">{{ t('dashboard.requests') }}</th>
                  <th class="pb-2 text-right font-semibold">{{ t('dashboard.tokens') }}</th>
                  <th class="pb-2 text-right font-semibold">{{ t('dashboard.actual') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="model in models"
                  :key="model.model"
                  class="border-t border-gray-100 transition-colors hover:bg-gray-50/80 dark:border-dark-700 dark:hover:bg-dark-800/60"
                >
                  <td class="max-w-[130px] truncate py-2.5 pr-2 font-medium text-gray-900 dark:text-white" :title="model.model">
                    {{ model.model }}
                  </td>
                  <td class="py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-400">{{ formatNumber(model.requests) }}</td>
                  <td class="py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-400">{{ formatTokens(model.total_tokens) }}</td>
                  <td class="py-2.5 text-right font-mono font-semibold text-teal-700 dark:text-teal-300">${{ formatCost(model.actual_cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-else class="flex min-h-64 flex-col items-center justify-center text-center">
          <span class="empty-icon"><Icon name="chart" size="lg" /></span>
          <p class="mt-3 text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('dashboard.noDataAvailable') }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.noChartDataHint') }}</p>
        </div>
      </article>

      <TokenUsageTrend :trend-data="trend" :loading="loading" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  ArcElement,
  Tooltip,
  Legend,
} from 'chart.js'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import {
  formatCostFixed as formatCost,
  formatNumberLocaleString as formatNumber,
  formatTokensK as formatTokens,
} from '@/utils/format'

ChartJS.register(ArcElement, Tooltip, Legend)

const props = defineProps<{
  loading: boolean
  startDate: string
  endDate: string
  granularity: 'day' | 'hour'
  trend: TrendDataPoint[]
  models: ModelStat[]
}>()

const emit = defineEmits<{
  'update:startDate': [value: string]
  'update:endDate': [value: string]
  'update:granularity': [value: 'day' | 'hour']
  dateRangeChange: [value: unknown]
  granularityChange: []
  refresh: []
}>()

const { t } = useI18n()
const granularityOptions = computed(() => [
  { value: 'day', label: t('dashboard.day') },
  { value: 'hour', label: t('dashboard.hour') },
])

function onGranularityUpdate(value: string | number | boolean | null) {
  if (value === 'day' || value === 'hour') {
    emit('update:granularity', value)
  }
}

const chartPalette = ['#0f766e', '#2563eb', '#d97706', '#7c3aed', '#e11d48', '#0891b2', '#4f46e5', '#65a30d']
const modelData = computed(() => {
  if (!props.models.length) return null
  return {
    labels: props.models.map((model) => model.model),
    datasets: [{
      data: props.models.map((model) => model.total_tokens),
      backgroundColor: props.models.map((_, index) => chartPalette[index % chartPalette.length]),
      borderWidth: 0,
      hoverOffset: 5,
    }],
  }
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '70%',
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: { label?: string; parsed: number }) =>
          `${context.label || ''}: ${formatTokens(context.parsed)} ${t('dashboard.tokens')}`,
      },
    },
  },
}))
</script>

<style scoped>
.chart-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
}

.chart-heading h2 {
  color: #0f172a;
  font-size: 1.125rem;
  font-weight: 700;
}

.chart-heading > p {
  max-width: 34rem;
  color: #64748b;
  font-size: 0.75rem;
  line-height: 1.5;
  text-align: right;
}

.chart-eyebrow {
  margin-bottom: 0.25rem;
  color: #0f9488;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.chart-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.72);
  padding: 0.8rem;
}

.toolbar-label {
  color: #475569;
  font-size: 0.75rem;
  font-weight: 700;
}

.range-chip {
  flex-shrink: 0;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 999px;
  background: rgba(248, 250, 252, 0.9);
  padding: 0.28rem 0.6rem;
  color: #64748b;
  font-size: 0.65rem;
  font-weight: 700;
}

.toolbar-button {
  display: flex;
  width: 2.35rem;
  height: 2.35rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(15, 118, 110, 0.2);
  border-radius: 8px;
  background: rgba(13, 148, 136, 0.08);
  color: #0f766e;
  transition: background-color 180ms ease, transform 180ms ease;
}

.toolbar-button:hover:not(:disabled) {
  background: rgba(13, 148, 136, 0.15);
  transform: translateY(-1px);
}

.toolbar-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.chart-panel {
  position: relative;
  min-width: 0;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.82);
  padding: 1.1rem;
}

.chart-panel h3 {
  color: #0f172a;
  font-size: 0.875rem;
  font-weight: 700;
}

.chart-panel h3 + p {
  margin-top: 0.25rem;
  color: #64748b;
  font-size: 0.7rem;
}

.empty-icon {
  display: flex;
  width: 2.75rem;
  height: 2.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgba(13, 148, 136, 0.08);
  color: #0f766e;
}

:global(.dark .chart-heading h2),
:global(.dark .chart-panel h3) { color: #f8fafc; }
:global(.dark .chart-heading > p),
:global(.dark .chart-panel h3 + p) { color: #94a3b8; }

:global(.dark .chart-toolbar),
:global(.dark .chart-panel) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(4, 8, 17, 0.78);
}

:global(.dark .toolbar-label) { color: #cbd5e1; }

:global(.dark .range-chip) {
  border-color: rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.82);
  color: #94a3b8;
}

:global(.dark .toolbar-button),
:global(.dark .empty-icon) {
  border-color: rgba(45, 212, 191, 0.18);
  background: rgba(20, 184, 166, 0.1);
  color: #5eead4;
}

@media (max-width: 767px) {
  .chart-heading,
  .chart-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .chart-heading > p {
    text-align: left;
  }

  .chart-toolbar > div:last-child {
    width: 100%;
    justify-content: flex-end;
  }
}

@media (prefers-reduced-motion: reduce) {
  .toolbar-button {
    transition: none;
  }

  .toolbar-button:hover:not(:disabled) {
    transform: none;
  }
}
</style>
