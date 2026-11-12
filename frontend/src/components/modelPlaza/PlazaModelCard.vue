<template>
  <article class="flex min-h-[254px] min-w-0 flex-col rounded-[20px] border border-gray-200 bg-white p-5 transition-colors hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800/80 dark:hover:border-primary-700">
    <div class="flex items-start gap-3">
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300">
        <PlatformIcon :platform="model.platform as GroupPlatform" size="lg" />
      </span>
      <div class="min-w-0 flex-1">
        <h3 class="truncate font-mono text-sm font-semibold text-gray-950 dark:text-white" :title="model.name">{{ model.name }}</h3>
        <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">{{ platformLabel(model.platform) }}</span>
      </div>
      <div class="flex shrink-0 items-center gap-1">
        <button type="button" class="rounded-full border border-gray-200 px-2.5 py-1 text-xs text-gray-600 hover:border-primary-400 hover:text-primary-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:border-dark-600 dark:text-dark-300" @click="$emit('details', model)">{{ t('modelPlaza.card.details') }}</button>
        <button type="button" class="rounded-full border border-gray-200 p-1.5 text-gray-500 hover:border-primary-400 hover:text-primary-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:border-dark-600 dark:text-dark-300" :aria-label="t('modelPlaza.card.copyName')" @click="copyToClipboard(model.name)"><Icon name="copy" size="xs" /></button>
      </div>
    </div>

    <!-- 最低价来自同一个分组的基础档；动态时段与上下文阶梯在详情中展开。 -->
    <div class="mt-4 min-h-[60px] font-mono text-sm font-semibold tabular-nums text-gray-950 dark:text-white">
      <template v-if="prices.primary != null">
        <div><span class="mr-2 font-sans text-xs font-normal text-gray-500 dark:text-dark-400">{{ prices.unit === 'token' ? t('modelPlaza.table.input') : priceUnit }}</span>{{ displayPrice(prices.primary, prices.unit, tokenScale) }}</div>
        <div v-if="prices.output != null" class="mt-1"><span class="mr-2 font-sans text-xs font-normal text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.output') }}</span>{{ displayPrice(prices.output, prices.unit, tokenScale) }}</div>
        <div v-if="prices.cache != null" class="mt-1 text-xs text-gray-500 dark:text-dark-400"><span class="mr-2 font-sans font-normal">{{ t('modelPlaza.table.cacheRead') }}</span>{{ displayPrice(prices.cache, prices.unit, tokenScale) }}</div>
      </template>
      <span v-else class="font-sans text-sm font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.detail.noPricing') }}</span>
    </div>

    <div class="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
      <span v-if="prices.primary != null" class="rounded-full bg-primary-50 px-2 py-0.5 font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">{{ t('modelPlaza.card.lowestBase') }}</span>
      <span v-if="prices.tiered || prices.timed" class="text-amber-600 dark:text-amber-400">{{ t('modelPlaza.card.variable') }}</span>
    </div>
    <p class="mt-3 line-clamp-2 min-h-[36px] text-xs leading-[18px] text-gray-500 dark:text-dark-400">{{ t('modelPlaza.card.groupSummary', { count: model.offerings.length }) }} · {{ groupNames }}</p>
    <div class="mt-auto flex items-center justify-between border-t border-gray-100 pt-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
      <span class="truncate">{{ model.best.group.name }} <span class="ml-1 text-primary-600 dark:text-primary-300">×{{ effectiveRate }}</span></span>
      <span class="ml-2 shrink-0">{{ unitLabel }}</span>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'
import { displayPrice, offeringRate, priceValues, type CatalogModel } from './catalog'

const props = defineProps<{ model: CatalogModel; tokenScale: 1_000_000 | 1_000 }>()
defineEmits<{ details: [model: CatalogModel] }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const prices = computed(() => priceValues(props.model.best))
const effectiveRate = computed(() => offeringRate(props.model.best))
const groupNames = computed(() => props.model.offerings.map(offering => offering.group.name).join(' · '))
const priceUnit = computed(() => prices.value.unit === 'image' ? t('modelPlaza.table.perImage') : t('modelPlaza.table.perRequest'))
const unitLabel = computed(() => prices.value.unit === 'token' ? (props.tokenScale === 1_000_000 ? '/ 1M' : '/ 1K') : priceUnit.value)
</script>
