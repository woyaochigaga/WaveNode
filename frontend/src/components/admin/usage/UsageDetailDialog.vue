<template>
  <BaseDialog
    :show="show"
    :title="t('usage.detail.title')"
    width="extra-wide"
    :close-on-click-outside="true"
    @close="emit('close')"
  >
    <div v-if="loading" class="flex min-h-64 items-center justify-center">
      <div class="h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-primary-500 dark:border-dark-600 dark:border-t-primary-400"></div>
    </div>

    <div v-else-if="loadError" class="flex min-h-64 flex-col items-center justify-center gap-3 text-center">
      <Icon name="exclamationCircle" size="lg" class="text-red-500" />
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('usage.detail.loadFailed') }}</p>
      <button type="button" class="btn btn-secondary" @click="loadDetail">{{ t('common.tryAgain') }}</button>
    </div>

    <div v-else-if="result" class="min-w-0 space-y-4 py-1">
      <div class="border-b border-gray-200 pb-4 dark:border-dark-700">
        <div class="flex flex-wrap items-center gap-2">
          <span
            class="inline-flex items-center gap-1.5 text-sm font-semibold"
            :class="(result.detail?.status_code || 200) < 400 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-current"></span>
            {{ result.detail?.status_code || 200 }}
          </span>
          <span class="font-mono text-xs font-semibold text-gray-600 dark:text-gray-300">
            {{ result.detail?.method || 'POST' }}
          </span>
          <span class="min-w-0 break-all font-mono text-sm text-gray-900 dark:text-white">
            {{ result.detail?.path || result.usage.inbound_endpoint || '-' }}
          </span>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-x-5 gap-y-3 sm:grid-cols-4">
          <MetaItem :label="t('usage.model')" :value="result.usage.model" />
          <MetaItem :label="t('usage.type')" :value="requestTypeLabel" />
          <MetaItem :label="t('usage.tokens')" :value="totalTokens.toLocaleString()" />
          <MetaItem :label="t('usage.cost')" :value="`$${result.usage.actual_cost.toFixed(6)}`" />
          <MetaItem :label="t('usage.latency')" :value="formatLatency(result.usage.duration_ms)" />
          <MetaItem :label="t('usage.time')" :value="formatDateTime(result.usage.created_at)" />
          <MetaItem class="col-span-2" :label="t('admin.usage.requestId')" :value="result.usage.request_id" mono />
        </div>
      </div>

      <div class="grid grid-cols-4 border-b border-gray-200 dark:border-dark-700" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          role="tab"
          :aria-selected="activeTab === tab.value"
          class="h-10 border-b-2 px-2 text-sm font-medium transition-colors"
          :class="activeTab === tab.value
            ? 'border-primary-500 text-primary-600 dark:text-primary-400'
            : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="activeTab = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>

      <div v-if="!result.detail" class="flex min-h-56 flex-col items-center justify-center gap-3 text-center">
        <Icon name="document" size="xl" class="text-gray-300 dark:text-dark-500" />
        <div>
          <p class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('usage.detail.notCaptured') }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('usage.detail.notCapturedHint') }}</p>
        </div>
      </div>

      <template v-else>
        <div v-if="activeTab === 'overview'" class="divide-y divide-gray-100 dark:divide-dark-700">
          <OverviewRow :label="t('usage.detail.requestContentType')" :value="result.detail.request_content_type || '-'" mono />
          <OverviewRow :label="t('usage.detail.responseContentType')" :value="result.detail.response_content_type || '-'" mono />
          <OverviewRow :label="t('usage.detail.inputSize')" :value="formatBytes(byteSize(result.detail.request_body))" />
          <OverviewRow :label="t('usage.detail.outputSize')" :value="formatBytes(byteSize(result.detail.response_body))" />
          <OverviewRow :label="t('usage.detail.expiresAt')" :value="formatDateTime(result.detail.expires_at)" />
          <div v-if="result.detail.request_truncated || result.detail.response_truncated" class="flex gap-2 py-3 text-sm text-amber-700 dark:text-amber-300">
            <Icon name="exclamationTriangle" size="sm" class="mt-0.5 shrink-0" />
            <span>{{ t('usage.detail.truncatedHint') }}</span>
          </div>
        </div>

        <PartList
          v-else-if="activeTab === 'input'"
          :parts="requestParts"
          :empty-text="t('usage.detail.noInput')"
        />

        <PartList
          v-else-if="activeTab === 'output'"
          :parts="responseParts"
          :empty-text="t('usage.detail.noOutput')"
        />

        <div v-else class="space-y-5">
          <RawPayload
            :title="t('usage.detail.rawRequest')"
            :content="prettyUsagePayload(result.detail.request_body)"
            @copy="copyPayload"
          />
          <RawPayload
            :title="t('usage.detail.rawResponse')"
            :content="prettyUsagePayload(result.detail.response_body)"
            @copy="copyPayload"
          />
        </div>
      </template>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, watch, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminUsageAPI, type AdminUsageDetailResponse } from '@/api/admin/usage'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import { resolveUsageRequestType } from '@/utils/usageRequestType'
import {
  classifyUsageRequest,
  classifyUsageResponse,
  prettyUsagePayload,
  type UsageDetailPart,
  type UsageDetailPartType,
} from '@/utils/usageDetail'

const props = defineProps<{ show: boolean; usageId: number | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const loadError = ref(false)
const result = ref<AdminUsageDetailResponse | null>(null)
const activeTab = ref<'overview' | 'input' | 'output' | 'raw'>('overview')
let requestSequence = 0

const tabs = computed(() => [
  { value: 'overview' as const, label: t('usage.detail.overview') },
  { value: 'input' as const, label: t('usage.detail.input') },
  { value: 'output' as const, label: t('usage.detail.output') },
  { value: 'raw' as const, label: t('usage.detail.raw') },
])

const requestParts = computed(() => classifyUsageRequest(result.value?.detail?.request_body || ''))
const responseParts = computed(() => classifyUsageResponse(result.value?.detail?.response_body || ''))
const totalTokens = computed(() => {
  const usage = result.value?.usage
  return usage ? usage.input_tokens + usage.output_tokens + usage.cache_creation_tokens + usage.cache_read_tokens : 0
})
const requestTypeLabel = computed(() => {
  if (!result.value) return '-'
  const type = resolveUsageRequestType(result.value.usage)
  return type === 'ws_v2' ? t('usage.ws') : t(`usage.${type}`)
})

/** 序列号避免快速切换行时旧请求覆盖当前弹窗。 */
async function loadDetail() {
  if (!props.show || !props.usageId) return
  const sequence = ++requestSequence
  loading.value = true
  loadError.value = false
  activeTab.value = 'overview'
  try {
    const response = await adminUsageAPI.getDetail(props.usageId)
    if (sequence === requestSequence) result.value = response
  } catch {
    if (sequence === requestSequence) {
      result.value = null
      loadError.value = true
    }
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

watch(() => [props.show, props.usageId] as const, ([show]) => {
  if (show) loadDetail()
  else requestSequence++
}, { immediate: true })

function formatLatency(value: number | null): string {
  if (value == null) return '-'
  if (value < 1000) return `${value} ms`
  return `${(value / 1000).toFixed(2)} s`
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`
  return `${(value / 1024).toFixed(1)} KB`
}

/** 按 UTF-8 实际存储字节计算，避免中文正文被 String.length 低估。 */
function byteSize(value: string): number {
  return new TextEncoder().encode(value).length
}

async function copyPayload(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    appStore.showSuccess(t('keys.copied'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

const MetaItem = defineComponent({
  props: { label: String, value: String, mono: Boolean },
  setup(itemProps) {
    return () => h('div', { class: 'min-w-0' }, [
      h('div', { class: 'text-[11px] text-gray-400 dark:text-gray-500' }, itemProps.label),
      h('div', { class: ['mt-0.5 break-all text-sm font-medium text-gray-800 dark:text-gray-100', itemProps.mono ? 'font-mono text-xs' : ''] }, itemProps.value || '-'),
    ])
  },
})

const OverviewRow = defineComponent({
  props: { label: String, value: String, mono: Boolean },
  setup(rowProps) {
    return () => h('div', { class: 'grid grid-cols-[minmax(9rem,0.35fr)_1fr] gap-4 py-3 text-sm' }, [
      h('span', { class: 'text-gray-500 dark:text-gray-400' }, rowProps.label),
      h('span', { class: ['break-all text-gray-800 dark:text-gray-100', rowProps.mono ? 'font-mono text-xs' : ''] }, rowProps.value || '-'),
    ])
  },
})

const partIcons: Record<UsageDetailPartType, 'chat' | 'lightbulb' | 'terminal' | 'document' | 'cube' | 'cog'> = {
  text: 'chat', reasoning: 'lightbulb', tool: 'terminal', image: 'document', audio: 'document',
  file: 'document', parameters: 'cog', json: 'cube',
}

const PartList = defineComponent({
  props: {
    parts: { type: Array as PropType<UsageDetailPart[]>, required: true },
    emptyText: { type: String, required: true },
  },
  setup(listProps) {
    return () => listProps.parts.length === 0
      ? h('div', { class: 'py-16 text-center text-sm text-gray-500 dark:text-gray-400' }, listProps.emptyText)
      : h('div', { class: 'divide-y divide-gray-100 dark:divide-dark-700' }, listProps.parts.map((part) =>
        h('section', { class: 'grid grid-cols-[2rem_minmax(0,1fr)] gap-3 py-4' }, [
          h('div', { class: 'flex h-8 w-8 items-center justify-center text-primary-500' }, [h(Icon, { name: partIcons[part.type], size: 'sm' })]),
          h('div', { class: 'min-w-0' }, [
            h('div', { class: 'flex flex-wrap items-center gap-2 text-xs font-semibold text-gray-700 dark:text-gray-200' }, [part.label, part.role ? ` · ${part.role}` : '']),
            h('pre', { class: 'mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-5 text-gray-600 dark:text-gray-300' }, part.content),
          ]),
        ])
      ))
  },
})

const RawPayload = defineComponent({
  emits: ['copy'],
  props: { title: { type: String, required: true }, content: { type: String, required: true } },
  setup(rawProps, { emit: rawEmit }) {
    return () => h('section', { class: 'min-w-0' }, [
      h('div', { class: 'mb-2 flex items-center justify-between gap-3' }, [
        h('h4', { class: 'text-sm font-semibold text-gray-800 dark:text-gray-100' }, rawProps.title),
        h('button', { type: 'button', title: t('keys.copyToClipboard'), class: 'rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200', onClick: () => rawEmit('copy', rawProps.content) }, [h(Icon, { name: 'copy', size: 'sm' })]),
      ]),
      h('pre', { class: 'max-h-80 overflow-auto rounded border border-gray-200 bg-gray-50 p-4 whitespace-pre-wrap break-words font-mono text-xs leading-5 text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300' }, rawProps.content || '-'),
    ])
  },
})
</script>
