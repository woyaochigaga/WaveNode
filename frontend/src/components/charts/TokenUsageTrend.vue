<template>
  <article class="trend-panel">
    <div class="mb-5 flex items-start justify-between gap-3">
      <div>
        <h3>{{ t('dashboard.tokenUsageTrend') }}</h3>
        <p>{{ t('dashboard.tokenUsageTrendDesc') }}</p>
      </div>
      <span class="trend-count">{{ t('dashboard.dataPoints', { count: trendData.length }) }}</span>
    </div>
    <div v-if="loading" class="flex h-64 items-center justify-center">
      <LoadingSpinner />
    </div>
    <div v-else-if="trendData.length > 0 && chartData" class="h-64">
      <Line :data="chartData" :options="lineOptions" />
    </div>
    <div
      v-else
      class="flex h-64 flex-col items-center justify-center text-center"
    >
      <span class="trend-empty"><Icon name="chart" size="lg" /></span>
      <p class="mt-3 text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('dashboard.noDataAvailable') }}</p>
      <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.noChartDataHint') }}</p>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import type { TrendDataPoint } from '@/types'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

const { t } = useI18n()

const props = defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
}>()

const isDarkMode = ref(document.documentElement.classList.contains('dark'))
let themeObserver: MutationObserver | null = null

// Chart.js 不会自动响应根节点 class 变化，这里同步主题状态以实时更新坐标与网格颜色。
onMounted(() => {
  themeObserver = new MutationObserver(() => {
    isDarkMode.value = document.documentElement.classList.contains('dark')
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})

const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? 'rgba(148, 163, 184, 0.14)' : 'rgba(148, 163, 184, 0.22)',
  input: '#2563eb',
  output: '#0f766e',
  cacheCreation: '#d97706',
  cacheRead: '#0891b2',
  cacheHitRate: '#7c3aed'
}))

const chartData = computed(() => {
  if (!props.trendData?.length) return null

  return {
    labels: props.trendData.map((d) => d.date),
    datasets: [
      {
        label: 'Input',
        data: props.trendData.map((d) => d.input_tokens),
        borderColor: chartColors.value.input,
        backgroundColor: `${chartColors.value.input}20`,
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 4,
        borderWidth: 2,
      },
      {
        label: 'Output',
        data: props.trendData.map((d) => d.output_tokens),
        borderColor: chartColors.value.output,
        backgroundColor: `${chartColors.value.output}20`,
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 4,
        borderWidth: 2,
      },
      {
        label: 'Cache Creation',
        data: props.trendData.map((d) => d.cache_creation_tokens),
        borderColor: chartColors.value.cacheCreation,
        backgroundColor: `${chartColors.value.cacheCreation}20`,
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 4,
        borderWidth: 1.5,
      },
      {
        label: 'Cache Read',
        data: props.trendData.map((d) => d.cache_read_tokens),
        borderColor: chartColors.value.cacheRead,
        backgroundColor: `${chartColors.value.cacheRead}20`,
        fill: true,
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 4,
        borderWidth: 1.5,
      },
      {
        label: 'Cache Hit Rate',
        data: props.trendData.map((d) => {
          const totalPromptTokens = d.input_tokens + d.cache_read_tokens + d.cache_creation_tokens
          return totalPromptTokens > 0 ? (d.cache_read_tokens / totalPromptTokens) * 100 : 0
        }),
        borderColor: chartColors.value.cacheHitRate,
        backgroundColor: `${chartColors.value.cacheHitRate}20`,
        borderDash: [5, 5],
        fill: false,
        tension: 0.35,
        pointRadius: 0,
        pointHoverRadius: 4,
        borderWidth: 1.5,
        yAxisID: 'yPercent'
      }
    ]
  }
})

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
      callbacks: {
        label: (context: any) => {
          if (context.dataset.yAxisID === 'yPercent') {
            return `${context.dataset.label}: ${context.raw.toFixed(1)}%`
          }
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        },
        footer: (tooltipItems: any) => {
          const dataIndex = tooltipItems[0]?.dataIndex
          if (dataIndex !== undefined && props.trendData[dataIndex]) {
            const data = props.trendData[dataIndex]
            return `Actual: $${formatCost(data.actual_cost)} | Standard: $${formatCost(data.cost)}`
          }
          return ''
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
    },
    yPercent: {
      position: 'right' as const,
      min: 0,
      max: 100,
      grid: {
        drawOnChartArea: false
      },
      ticks: {
        color: chartColors.value.cacheHitRate,
        font: {
          size: 10
        },
        callback: (value: string | number) => `${value}%`
      }
    }
  }
}))

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatCost = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}
</script>

<style scoped>
.trend-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.82);
  padding: 1.1rem;
}

.trend-panel h3 {
  color: #0f172a;
  font-size: 0.875rem;
  font-weight: 700;
}

.trend-panel h3 + p {
  margin-top: 0.25rem;
  color: #64748b;
  font-size: 0.7rem;
}

.trend-count {
  flex-shrink: 0;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 999px;
  background: rgba(248, 250, 252, 0.9);
  padding: 0.28rem 0.6rem;
  color: #64748b;
  font-size: 0.65rem;
  font-weight: 700;
}

.trend-empty {
  display: flex;
  width: 2.75rem;
  height: 2.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgba(37, 99, 235, 0.08);
  color: #2563eb;
}

:global(.dark .trend-panel) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(4, 8, 17, 0.78);
}

:global(.dark .trend-panel h3) { color: #f8fafc; }
:global(.dark .trend-panel h3 + p) { color: #94a3b8; }

:global(.dark .trend-count) {
  border-color: rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.82);
  color: #94a3b8;
}

:global(.dark .trend-empty) {
  background: rgba(37, 99, 235, 0.12);
  color: #93c5fd;
}
</style>
