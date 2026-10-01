<template>
  <section class="quick-panel" aria-labelledby="quick-actions-title">
    <div class="quick-header">
      <p class="quick-eyebrow">{{ t('dashboard.nextStepEyebrow') }}</p>
      <h2 id="quick-actions-title">{{ t('dashboard.quickActions') }}</h2>
      <p>{{ t('dashboard.quickActionsDesc') }}</p>
    </div>

    <div class="space-y-2 p-3">
      <button
        v-for="action in actions"
        :key="action.path"
        v-show="!action.requiresBatchImage || canUseBatchImage"
        type="button"
        class="quick-action group"
        :class="`quick-action--${action.tone}`"
        @click="router.push(action.path)"
      >
        <span class="quick-action__icon">
          <Icon :name="action.icon" size="md" :stroke-width="2" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-semibold text-gray-950 dark:text-white">{{ t(action.labelKey) }}</span>
          <span class="mt-0.5 block text-xs leading-5 text-gray-500 dark:text-dark-300">{{ t(action.descriptionKey) }}</span>
        </span>
        <Icon name="chevronRight" size="sm" class="quick-action__arrow" />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

// 快捷入口集中配置，新增入口时无需复制整段模板。
const actions = [
  { path: '/keys', icon: 'key', tone: 'teal', labelKey: 'dashboard.createApiKey', descriptionKey: 'dashboard.generateNewKey', requiresBatchImage: false },
  { path: '/usage', icon: 'chart', tone: 'blue', labelKey: 'dashboard.viewUsage', descriptionKey: 'dashboard.checkDetailedLogs', requiresBatchImage: false },
  { path: '/batch-image', icon: 'sparkles', tone: 'violet', labelKey: 'dashboard.batchImageAgent', descriptionKey: 'dashboard.batchImageAgentDesc', requiresBatchImage: true },
  { path: '/redeem', icon: 'gift', tone: 'amber', labelKey: 'dashboard.redeemCode', descriptionKey: 'dashboard.addBalanceWithCode', requiresBatchImage: false },
] as const

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>

<style scoped>
.quick-panel {
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.78);
}

.quick-header {
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
  padding: 1.1rem 1.25rem;
}

.quick-header h2 {
  color: #0f172a;
  font-size: 1rem;
  font-weight: 700;
}

.quick-header h2 + p {
  margin-top: 0.25rem;
  color: #64748b;
  font-size: 0.7rem;
  line-height: 1.5;
}

.quick-eyebrow {
  margin-bottom: 0.25rem;
  color: #0f9488;
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
}

.quick-action {
  --action-rgb: 13, 148, 136;
  display: flex;
  width: 100%;
  min-height: 4.5rem;
  align-items: center;
  gap: 0.75rem;
  border: 1px solid transparent;
  border-radius: 8px;
  padding: 0.75rem;
  text-align: left;
  transition: border-color 180ms ease, background-color 180ms ease, transform 180ms ease;
}

.quick-action:hover {
  border-color: rgba(var(--action-rgb), 0.24);
  background: rgba(var(--action-rgb), 0.055);
  transform: translateX(2px);
}

.quick-action--blue { --action-rgb: 37, 99, 235; }
.quick-action--violet { --action-rgb: 124, 58, 237; }
.quick-action--amber { --action-rgb: 217, 119, 6; }

.quick-action__icon {
  display: flex;
  width: 2.35rem;
  height: 2.35rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(var(--action-rgb), 0.18);
  border-radius: 8px;
  background: rgba(var(--action-rgb), 0.08);
  color: rgb(var(--action-rgb));
}

.quick-action__arrow {
  flex-shrink: 0;
  color: #94a3b8;
  transition: color 180ms ease, transform 180ms ease;
}

.quick-action:hover .quick-action__arrow {
  color: rgb(var(--action-rgb));
  transform: translateX(2px);
}

:global(.dark .quick-panel) {
  border-color: rgba(148, 163, 184, 0.12);
  background: rgba(4, 8, 17, 0.78);
}

:global(.dark .quick-header) { border-color: rgba(148, 163, 184, 0.1); }
:global(.dark .quick-header h2) { color: #f8fafc; }
:global(.dark .quick-header h2 + p) { color: #94a3b8; }

:global(.dark .quick-action:hover) {
  border-color: rgba(var(--action-rgb), 0.26);
  background: rgba(var(--action-rgb), 0.09);
}

@media (prefers-reduced-motion: reduce) {
  .quick-action,
  .quick-action__arrow {
    transition: none;
  }

  .quick-action:hover,
  .quick-action:hover .quick-action__arrow {
    transform: none;
  }
}
</style>
