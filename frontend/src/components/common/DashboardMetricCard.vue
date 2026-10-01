<template>
  <article
    class="dashboard-metric"
    :class="[`dashboard-metric--${tone}`, { 'dashboard-metric--featured': featured }]"
  >
    <div class="flex min-w-0 items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <Icon class="dashboard-metric__icon" :name="icon" size="sm" :stroke-width="2" />
        <div class="flex min-w-0 items-center gap-1">
          <p class="truncate text-xs font-semibold text-gray-500 dark:text-dark-300">{{ label }}</p>
          <HelpTooltip v-if="help || $slots.help" width-class="w-80">
            <slot name="help">{{ help }}</slot>
          </HelpTooltip>
        </div>
      </div>
      <div v-if="$slots.status" class="shrink-0">
        <slot name="status" />
      </div>
    </div>

    <div class="mt-4 min-w-0">
      <div class="dashboard-metric__value">
        <slot name="value">{{ value }}</slot>
      </div>

      <div v-if="$slots.default" class="mt-2 min-h-5 text-xs leading-5 text-gray-500 dark:text-dark-300">
        <slot />
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'

type MetricIcon =
  | 'key'
  | 'server'
  | 'chart'
  | 'userPlus'
  | 'cube'
  | 'database'
  | 'bolt'
  | 'clock'
  | 'dollar'
  | 'users'

withDefaults(defineProps<{
  label: string
  value?: string | number
  icon: MetricIcon
  tone?: 'teal' | 'blue' | 'violet' | 'amber' | 'rose' | 'slate'
  help?: string
  featured?: boolean
}>(), {
  value: '-',
  tone: 'teal',
  help: '',
  featured: false,
})
</script>

<style scoped>
.dashboard-metric {
  --metric-rgb: 13, 148, 136;
  position: relative;
  min-width: 0;
  min-height: 9rem;
  background: rgba(255, 255, 255, 0.94);
  padding: 1.25rem;
  transition: background-color 180ms ease;
}

.dashboard-metric:hover {
  background: rgba(248, 250, 252, 0.98);
}

.dashboard-metric__icon {
  flex-shrink: 0;
  color: rgb(var(--metric-rgb));
}

.dashboard-metric__value {
  overflow-wrap: anywhere;
  color: #0f172a;
  font-size: clamp(1.45rem, 2vw, 1.9rem);
  font-weight: 700;
  line-height: 1.15;
  font-variant-numeric: tabular-nums;
}

.dashboard-metric--blue { --metric-rgb: 37, 99, 235; }
.dashboard-metric--violet { --metric-rgb: 124, 58, 237; }
.dashboard-metric--amber { --metric-rgb: 217, 119, 6; }
.dashboard-metric--rose { --metric-rgb: 225, 29, 72; }
.dashboard-metric--slate { --metric-rgb: 71, 85, 105; }

:global(.dark .dashboard-metric) {
  background: rgba(4, 8, 17, 0.96);
}

:global(.dark .dashboard-metric:hover) {
  background: rgba(9, 15, 27, 0.98);
}

:global(.dark .dashboard-metric__value) {
  color: #f8fafc;
}

:global(.dashboard-metrics-grid) {
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.22);
}

:global(.dashboard-metrics-grid--compact .dashboard-metric) {
  min-height: 7.75rem;
  padding-block: 1rem;
}

:global(.dark .dashboard-metrics-grid) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(148, 163, 184, 0.12);
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-metric {
    transition: none;
  }

}
</style>
