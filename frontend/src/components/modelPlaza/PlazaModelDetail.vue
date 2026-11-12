<template>
  <Teleport to="body">
    <div v-if="model" class="fixed inset-0 z-50 flex justify-end" role="presentation">
      <div class="absolute inset-0 bg-gray-950/50 backdrop-blur-[3px]" @click="$emit('close')"></div>
      <section role="dialog" aria-modal="true" :aria-label="model.name" class="relative flex h-full w-full max-w-[880px] flex-col overflow-hidden bg-white shadow-2xl dark:bg-dark-900">
        <header class="flex items-start justify-between gap-4 border-b border-gray-200 px-5 py-5 dark:border-dark-700 sm:px-7">
          <div class="flex min-w-0 items-start gap-3">
            <span class="mt-1 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300"><PlatformIcon :platform="model.platform as GroupPlatform" size="lg" /></span>
            <div class="min-w-0">
              <h2 class="break-all font-mono text-xl font-bold text-gray-950 dark:text-white">{{ model.name }}</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ platformLabel(model.platform) }} · {{ t('modelPlaza.card.groupSummary', { count: model.offerings.length }) }}</p>
            </div>
          </div>
          <button type="button" class="rounded-full p-2 text-gray-500 hover:bg-gray-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-dark-300 dark:hover:bg-dark-700" :aria-label="t('common.close')" @click="$emit('close')"><Icon name="x" size="sm" /></button>
        </header>

        <div class="overflow-y-auto px-5 py-6 sm:px-7">
          <p class="mb-4 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('modelPlaza.detail.basePriceHint') }}</p>
          <div class="space-y-4">
            <section v-for="offering in sortedOfferings" :key="offering.group.id" class="overflow-hidden rounded-2xl border border-gray-200 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3 bg-gray-50/80 px-4 py-3 dark:bg-dark-800/80">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="font-semibold text-gray-900 dark:text-white">{{ offering.group.name }}</h3>
                  <span class="rounded-full bg-primary-50 px-2 py-0.5 text-[11px] text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">×{{ offeringRate(offering) }}</span>
                  <span v-if="offering.group.is_exclusive" class="text-[11px] text-gray-500 dark:text-dark-400">{{ t('modelPlaza.badges.exclusive') }}</span>
                  <span v-if="offering.group.subscription_type === 'subscription'" class="text-[11px] text-gray-500 dark:text-dark-400">{{ t('modelPlaza.badges.subscription') }}</span>
                </div>
                <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.card.basePeriod') }}</span>
              </div>
              <div class="grid gap-3 p-4 sm:grid-cols-3">
                <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
                  <span class="text-xs text-gray-500 dark:text-dark-400">{{ unitLabel(offering) === 'token' ? t('modelPlaza.table.input') : t('modelPlaza.card.requestPrice') }}</span>
                  <strong class="mt-1 block font-mono text-lg tabular-nums text-gray-950 dark:text-white">{{ displayPrice(priceValues(offering).primary, unitLabel(offering), tokenScale) }}</strong>
                </div>
                <div v-if="unitLabel(offering) === 'token'" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800"><span class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.output') }}</span><strong class="mt-1 block font-mono text-lg tabular-nums text-gray-950 dark:text-white">{{ displayPrice(priceValues(offering).output, 'token', tokenScale) }}</strong></div>
                <div v-if="unitLabel(offering) === 'token'" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800"><span class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.cacheRead') }}</span><strong class="mt-1 block font-mono text-lg tabular-nums text-gray-950 dark:text-white">{{ displayPrice(priceValues(offering).cache, 'token', tokenScale) }}</strong></div>
              </div>
              <div v-if="offering.group.description || priceValues(offering).tiered || priceValues(offering).timed" class="px-4 pb-3 text-xs leading-5 text-gray-500 dark:text-dark-400">
                <p v-if="offering.group.description">{{ offering.group.description }}</p>
                <p v-if="priceValues(offering).tiered || priceValues(offering).timed" class="text-amber-600 dark:text-amber-400">{{ t('modelPlaza.card.variable') }}</p>
              </div>
              <!-- 原有价格表完整呈现官方参考价、分时、长上下文和思考等级规则。 -->
              <details class="border-t border-gray-100 dark:border-dark-700">
                <summary class="cursor-pointer px-4 py-3 text-xs font-medium text-primary-700 hover:bg-gray-50 dark:text-primary-300 dark:hover:bg-dark-800">{{ t('modelPlaza.detail.fullPricing') }}</summary>
                <PlazaModelPricingTable :models="[offering.model]" :platform="offering.group.platform" :rate-multiplier="offering.group.rate_multiplier" :user-rate-multiplier="offering.group.user_rate_multiplier ?? null" :image-rate-independent="offering.group.image_rate_independent" :image-rate-multiplier="offering.group.image_rate_multiplier" :peak-window="peakWindow(offering.group)" :peak-rate-multiplier="offering.group.peak_rate_multiplier" />
              </details>
            </section>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import { platformLabel } from '@/utils/platformColors'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { useAppStore } from '@/stores/app'
import PlazaModelPricingTable from './PlazaModelPricingTable.vue'
import { compareOfferings, displayPrice, offeringRate, priceValues, type CatalogModel, type PlazaOffering } from './catalog'

const props = defineProps<{ model: CatalogModel | null; tokenScale: 1_000_000 | 1_000 }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const sortedOfferings = computed(() => [...(props.model?.offerings ?? [])].sort(compareOfferings))

function unitLabel(offering: PlazaOffering): 'token' | 'request' | 'image' {
  return priceValues(offering).unit
}

function peakWindow(group: ModelPlazaGroup): string {
  return hasPeakRate(group) ? formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)) : ''
}

/** 详情打开时锁定背景滚动；关闭或组件卸载后还原页面原状态。 */
let previousOverflow = ''
watch(() => props.model, model => {
  if (model) {
    previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = previousOverflow
  }
})
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.model) emit('close')
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  document.body.style.overflow = previousOverflow
  window.removeEventListener('keydown', onKeydown)
})
</script>
