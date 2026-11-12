<template>
  <aside class="rounded-[20px] border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800/80 lg:sticky lg:top-20 lg:self-start">
    <div class="flex items-start justify-between gap-3 border-b border-gray-100 pb-4 dark:border-dark-700">
      <div>
        <h2 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('modelPlaza.filters.title') }}</h2>
        <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('modelPlaza.filters.hint') }}</p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <button type="button" class="text-xs text-gray-500 hover:text-primary-600 dark:text-dark-400" @click="$emit('reset')">{{ t('modelPlaza.filters.clear') }}</button>
        <button type="button" class="rounded-lg border border-gray-200 px-2 py-1 text-xs text-gray-600 dark:border-dark-600 dark:text-dark-300 lg:hidden" :aria-expanded="mobileExpanded" aria-controls="plaza-filters" @click="mobileExpanded = !mobileExpanded">{{ mobileExpanded ? t('modelPlaza.filters.collapse') : t('modelPlaza.filters.expand') }}</button>
      </div>
    </div>

    <!-- 模型数按平台/计费方式去重，分组和倍率仍可用于细化同一模型的报价。 -->
    <div id="plaza-filters" class="plaza-filters-body" :class="{ 'plaza-filters-expanded': mobileExpanded }">
    <div class="mt-4" role="group" :aria-label="t('modelPlaza.filters.platformLabel')">
      <h3 class="mb-3 text-xs font-semibold text-gray-900 dark:text-white">{{ t('modelPlaza.filters.platformLabel') }}</h3>
      <div class="flex flex-wrap gap-2">
        <button v-for="option in platforms" :key="option.value" type="button" class="plaza-filter-chip" :class="platform === option.value ? 'plaza-filter-chip-active' : ''" :aria-pressed="platform === option.value" @click="$emit('update:platform', option.value)">
          <PlatformIcon v-if="option.value !== 'all'" :platform="option.value as GroupPlatform" size="xs" />
          <span>{{ option.value === 'all' ? t('modelPlaza.filters.allPlatforms') : platformLabel(option.value) }}</span>
          <span class="plaza-filter-count">{{ option.count }}</span>
        </button>
      </div>
    </div>

    <div class="mt-4 border-t border-gray-100 pt-4 dark:border-dark-700" role="group" :aria-label="t('modelPlaza.filters.modeLabel')">
      <h3 class="mb-3 text-xs font-semibold text-gray-900 dark:text-white">{{ t('modelPlaza.filters.modeLabel') }}</h3>
      <div class="flex flex-wrap gap-2">
        <button v-for="option in modes" :key="option.value" type="button" class="plaza-filter-chip" :class="mode === option.value ? 'plaza-filter-chip-active' : ''" :aria-pressed="mode === option.value" @click="$emit('update:mode', option.value)">
          {{ modeLabel(option.value) }} <span class="plaza-filter-count">{{ option.count }}</span>
        </button>
      </div>
    </div>

    <div class="mt-4 border-t border-gray-100 pt-4 dark:border-dark-700" role="group" :aria-label="t('modelPlaza.filters.groupLabel')">
      <h3 class="mb-3 text-xs font-semibold text-gray-900 dark:text-white">{{ t('modelPlaza.filters.groupLabel') }}</h3>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="plaza-filter-chip" :class="groupId === 'all' ? 'plaza-filter-chip-active' : ''" :aria-pressed="groupId === 'all'" @click="$emit('update:groupId', 'all')">{{ t('modelPlaza.filters.all') }}</button>
        <button v-for="group in groups" :key="group.id" type="button" class="plaza-filter-chip" :class="groupId === group.id ? 'plaza-filter-chip-active' : ''" :aria-pressed="groupId === group.id" @click="$emit('update:groupId', group.id)">{{ group.name }}</button>
      </div>
    </div>

    <div v-if="rates.length > 1" class="mt-4 border-t border-gray-100 pt-4 dark:border-dark-700" role="group" :aria-label="t('modelPlaza.filters.rateLabel')">
      <h3 class="mb-3 text-xs font-semibold text-gray-900 dark:text-white">{{ t('modelPlaza.filters.rateLabel') }}</h3>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="plaza-filter-chip" :class="rate === 'all' ? 'plaza-filter-chip-active' : ''" :aria-pressed="rate === 'all'" @click="$emit('update:rate', 'all')">{{ t('modelPlaza.filters.all') }}</button>
        <button v-for="value in rates" :key="value" type="button" class="plaza-filter-chip tabular-nums" :class="rate === value ? 'plaza-filter-chip-active' : ''" :aria-pressed="rate === value" @click="$emit('update:rate', value)">×{{ value }}</button>
      </div>
    </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'

defineProps<{
  platforms: Array<{ value: string; count: number }>
  modes: Array<{ value: string; count: number }>
  groups: Array<{ id: number; name: string }>
  rates: number[]
  platform: string
  mode: string
  groupId: number | 'all'
  rate: number | 'all'
}>()

defineEmits<{
  'update:platform': [value: string]
  'update:mode': [value: string]
  'update:groupId': [value: number | 'all']
  'update:rate': [value: number | 'all']
  reset: []
}>()

const { t } = useI18n()
const mobileExpanded = ref(false)

/** 筛选标签只使用接口提供的真实计费方式，不推断模型能力。 */
function modeLabel(mode: string): string {
  const label: Record<string, string> = {
    all: 'allModes', token: 'tokenMode', image: 'imageMode', video: 'videoMode', per_request: 'requestMode'
  }
  return t(`modelPlaza.filters.${label[mode] ?? 'requestMode'}`)
}
</script>

<style scoped>
.plaza-filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  max-width: 100%;
  border: 1px solid rgb(156 163 175 / 0.25);
  border-radius: 999px;
  padding: 0.3rem 0.55rem;
  font-size: 0.75rem;
  line-height: 1.2rem;
  color: #6b7280;
  transition: border-color 150ms, color 150ms, background-color 150ms;
}
.plaza-filter-chip:hover, .plaza-filter-chip-active {
  border-color: rgb(20 184 166 / 0.55);
  color: #0f766e;
  background: rgb(20 184 166 / 0.08);
}
.plaza-filter-chip:focus-visible { outline: 2px solid #14b8a6; outline-offset: 2px; }
.plaza-filter-count { border-radius: 999px; padding: 0 0.35rem; background: rgb(156 163 175 / 0.12); font-variant-numeric: tabular-nums; }
.dark .plaza-filter-chip { color: #a5b4c5; border-color: rgb(148 163 184 / 0.2); }
.dark .plaza-filter-chip:hover, .dark .plaza-filter-chip-active { color: #5eead4; border-color: rgb(45 212 191 / 0.55); background: rgb(20 184 166 / 0.12); }
@media (max-width: 1023px) {
  .plaza-filters-body { display: none; }
  .plaza-filters-expanded { display: block; }
}
</style>
