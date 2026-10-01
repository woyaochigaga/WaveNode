<template>
  <section class="recent-panel" aria-labelledby="recent-usage-title">
    <div class="recent-header">
      <div>
        <p class="recent-eyebrow">{{ t('dashboard.activityEyebrow') }}</p>
        <h2 id="recent-usage-title">{{ t('dashboard.recentUsage') }}</h2>
        <p>{{ t('dashboard.recentUsageDesc') }}</p>
      </div>
      <span class="recent-range">{{ t('dashboard.last7Days') }}</span>
    </div>

    <div v-if="loading" class="flex min-h-80 items-center justify-center">
      <LoadingSpinner size="lg" />
    </div>

    <div v-else-if="data.length === 0" class="flex min-h-80 items-center justify-center px-4">
      <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    </div>

    <div v-else class="recent-list">
      <article v-for="log in data" :key="log.id" class="usage-row">
        <div class="flex min-w-0 items-center gap-3">
          <span class="usage-icon"><Icon name="beaker" size="md" /></span>
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-gray-950 dark:text-white" :title="log.model">
              {{ log.model }}
            </p>
            <div class="mt-1 flex flex-wrap items-center gap-1.5 text-[10px] text-gray-500 dark:text-dark-300">
              <span>{{ formatDateTime(log.created_at) }}</span>
              <span class="usage-chip">{{ log.stream ? t('dashboard.streaming') : t('dashboard.synchronous') }}</span>
              <span v-if="log.duration_ms != null" class="usage-chip">{{ formatDuration(log.duration_ms) }}</span>
            </div>
          </div>
        </div>

        <div class="shrink-0 text-right">
          <p class="font-mono text-sm font-bold text-teal-700 dark:text-teal-300">
            ${{ formatCost(log.actual_cost) }}
          </p>
          <p class="mt-1 text-[10px] tabular-nums text-gray-500 dark:text-dark-300">
            {{ formatTokens(totalTokens(log)) }} {{ t('dashboard.tokens') }}
          </p>
        </div>
      </article>

      <router-link to="/usage" class="view-all-link">
        <span>{{ t('dashboard.viewAllUsage') }}</span>
        <Icon name="arrowRight" size="sm" />
      </router-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()

const { t } = useI18n()
const formatCost = (cost: number) => cost.toFixed(4)
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
const formatTokens = (tokens: number) => {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M`
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}K`
  return tokens.toLocaleString()
}

// 最近请求展示完整处理量，包含输入、输出、缓存写入与缓存读取 Token。
const totalTokens = (log: UsageLog) =>
  log.input_tokens + log.output_tokens + log.cache_creation_tokens + log.cache_read_tokens
</script>

<style scoped>
.recent-panel {
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.78);
}

.recent-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
  padding: 1.1rem 1.25rem;
}

.recent-header h2 {
  color: #0f172a;
  font-size: 1rem;
  font-weight: 700;
}

.recent-header h2 + p {
  margin-top: 0.25rem;
  color: #64748b;
  font-size: 0.7rem;
}

.recent-eyebrow {
  margin-bottom: 0.25rem;
  color: #0f9488;
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
}

.recent-range,
.usage-chip {
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 999px;
  background: rgba(248, 250, 252, 0.9);
  color: #64748b;
}

.recent-range {
  flex-shrink: 0;
  padding: 0.28rem 0.6rem;
  font-size: 0.65rem;
  font-weight: 700;
}

.recent-list {
  padding: 0 1.25rem 0.75rem;
}

.usage-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
  padding: 0.9rem 0;
}

.usage-row:last-of-type {
  border-bottom: 0;
}

.usage-icon {
  display: flex;
  width: 2.35rem;
  height: 2.35rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(13, 148, 136, 0.18);
  border-radius: 8px;
  background: rgba(13, 148, 136, 0.08);
  color: #0f766e;
}

.usage-chip {
  padding: 0.12rem 0.35rem;
  font-weight: 600;
}

.view-all-link {
  display: flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  color: #0f766e;
  font-size: 0.75rem;
  font-weight: 700;
  transition: gap 180ms ease, color 180ms ease;
}

.view-all-link:hover {
  gap: 0.65rem;
  color: #115e59;
}

:global(.dark .recent-panel) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(4, 8, 17, 0.78);
}

:global(.dark .recent-header),
:global(.dark .usage-row),
:global(.dark .view-all-link) {
  border-color: rgba(148, 163, 184, 0.1);
}

:global(.dark .recent-header h2) { color: #f8fafc; }
:global(.dark .recent-header h2 + p) { color: #94a3b8; }

:global(.dark .recent-range),
:global(.dark .usage-chip) {
  border-color: rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.82);
  color: #94a3b8;
}

:global(.dark .usage-icon) {
  border-color: rgba(45, 212, 191, 0.18);
  background: rgba(20, 184, 166, 0.1);
  color: #5eead4;
}

:global(.dark .view-all-link) { color: #5eead4; }

@media (max-width: 520px) {
  .usage-row {
    align-items: flex-start;
  }

  .recent-list,
  .recent-header {
    padding-right: 1rem;
    padding-left: 1rem;
  }
}
</style>
