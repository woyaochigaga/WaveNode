<template>
  <BaseDialog
    :show="show"
    :title="t('common.connectionTest.title')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-5">
      <div
        class="flex items-center gap-3 border-b border-gray-200 pb-4 dark:border-dark-600"
      >
        <span
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300"
        >
          <Icon name="terminal" size="md" :stroke-width="1.8" />
        </span>
        <div class="min-w-0">
          <p class="truncate font-semibold text-gray-900 dark:text-gray-100">
            {{ targetName }}
          </p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
            {{ targetType === 'group'
              ? t('common.connectionTest.groupTarget')
              : t('common.connectionTest.keyTarget') }}
          </p>
        </div>
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('common.connectionTest.modelLabel') }}
        </label>
        <Select
          v-model="selectedModel"
          :options="modelOptions"
          :loading="loadingModels"
          :disabled="loadingModels || status === 'connecting'"
          searchable
          :placeholder="modelPlaceholder"
          :empty-text="t('common.connectionTest.noModels')"
        />
        <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">
          {{ t('common.connectionTest.nonBillableHint') }}
        </p>
      </div>

      <div
        ref="terminalRef"
        class="min-h-[190px] max-h-[300px] overflow-y-auto rounded-lg border border-gray-800 bg-[#090d14] p-4 font-mono text-[13px] leading-6 shadow-inner"
        aria-live="polite"
      >
        <div v-if="status === 'idle'" class="flex items-center gap-2 text-gray-500">
          <span class="h-1.5 w-1.5 rounded-full bg-gray-600" />
          {{ t('common.connectionTest.ready') }}
        </div>
        <div v-else-if="status === 'connecting'" class="flex items-center gap-2 text-amber-300">
          <Icon name="refresh" size="sm" class="animate-spin" />
          {{ t('common.connectionTest.connecting') }}
        </div>

        <div
          v-for="(line, index) in outputLines"
          :key="`${index}-${line.text}`"
          :class="line.className"
        >
          {{ line.text }}
        </div>
        <div v-if="streamingContent" class="whitespace-pre-wrap break-words text-emerald-300">
          {{ streamingContent }}<span class="animate-pulse">_</span>
        </div>

        <div
          v-if="status === 'success'"
          class="mt-3 flex items-center gap-2 border-t border-gray-800 pt-3 text-emerald-400"
        >
          <Icon name="check" size="sm" :stroke-width="2" />
          {{ t('common.connectionTest.success') }}
        </div>
        <div
          v-else-if="status === 'error'"
          class="mt-3 flex items-start gap-2 border-t border-gray-800 pt-3 text-red-400"
        >
          <Icon name="x" size="sm" class="mt-1 shrink-0" :stroke-width="2" />
          <span class="break-words">{{ errorMessage }}</span>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-400">
        <span>{{ t('common.connectionTest.actualUpstreamHint') }}</span>
        <button
          v-if="outputLines.length || streamingContent"
          type="button"
          class="inline-flex shrink-0 items-center gap-1.5 text-gray-500 transition-colors hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-300"
          @click="copyOutput"
        >
          <Icon name="copy" size="sm" />
          {{ t('common.connectionTest.copyOutput') }}
        </button>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.close') }}
        </button>
        <button
          type="button"
          class="btn btn-primary inline-flex items-center gap-2"
          :disabled="!canStart"
          @click="startTest"
        >
          <Icon
            :name="status === 'idle' ? 'play' : 'refresh'"
            size="sm"
            :class="status === 'connecting' ? 'animate-spin' : ''"
          />
          {{ actionLabel }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { keysAPI } from '@/api'
import { buildApiUrl } from '@/api/client'
import { ADMIN_UI_REQUEST_HEADER, USER_UI_REQUEST_HEADER } from '@/api/adminUIRequest'
import { useClipboard } from '@/composables/useClipboard'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

type TestStatus = 'idle' | 'connecting' | 'success' | 'error'
type TestTargetType = 'group' | 'api-key'

interface OutputLine {
  text: string
  className: string
}

interface TestEvent {
  type: 'test_start' | 'content' | 'status' | 'test_complete' | 'error'
  text?: string
  model?: string
  status?: string
  success?: boolean
  error?: string
}

const props = defineProps<{
  show: boolean
  targetType: TestTargetType
  targetId: number | null
  targetName: string
}>()

const emit = defineEmits<{
  (event: 'close'): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const terminalRef = ref<HTMLElement | null>(null)
const availableModels = ref<string[]>([])
const selectedModel = ref<string | number | boolean | null>('')
const loadingModels = ref(false)
const status = ref<TestStatus>('idle')
const outputLines = ref<OutputLine[]>([])
const streamingContent = ref('')
const errorMessage = ref('')
let abortController: AbortController | null = null

const modelOptions = computed(() =>
  availableModels.value.map((model) => ({ value: model, label: model }))
)
const modelPlaceholder = computed(() =>
  loadingModels.value
    ? t('common.loading')
    : t('common.connectionTest.modelPlaceholder')
)
const canStart = computed(
  () => Boolean(props.targetId && selectedModel.value) && status.value !== 'connecting'
)
const actionLabel = computed(() => {
  if (status.value === 'connecting') return t('common.connectionTest.testing')
  if (status.value === 'idle') return t('common.connectionTest.start')
  return t('common.connectionTest.retry')
})

// 接口只返回当前至少有一个可调度账号支持的模型。
const loadModels = async () => {
  if (!props.targetId) return
  loadingModels.value = true
  selectedModel.value = ''
  try {
    availableModels.value = props.targetType === 'group'
      ? await adminAPI.groups.getTestModels(props.targetId)
      : await keysAPI.getTestModels(props.targetId)
    selectedModel.value = preferredModel(availableModels.value)
  } catch (error) {
    availableModels.value = []
    setError(readErrorMessage(error))
  } finally {
    loadingModels.value = false
  }
}

// 优先选择轻量文本模型；没有匹配项时保持后端返回顺序。
const preferredModel = (models: string[]): string => {
  const preferred = ['haiku', 'flash', 'mini']
  for (const keyword of preferred) {
    const model = models.find((item) => item.toLowerCase().includes(keyword) && !item.includes('image'))
    if (model) return model
  }
  return models[0] || ''
}

const resetOutput = () => {
  status.value = 'idle'
  outputLines.value = []
  streamingContent.value = ''
  errorMessage.value = ''
}

const abortStream = () => {
  abortController?.abort()
  abortController = null
}

const handleClose = () => {
  abortStream()
  emit('close')
}

const addLine = (text: string, className = 'text-gray-300') => {
  outputLines.value.push({ text, className })
  void scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  if (terminalRef.value) terminalRef.value.scrollTop = terminalRef.value.scrollHeight
}

const setError = (message: string) => {
  status.value = 'error'
  errorMessage.value = message
}

const readErrorMessage = (error: unknown): string => {
  if (typeof error === 'object' && error && 'message' in error) {
    return String((error as { message?: unknown }).message || t('common.unknownError'))
  }
  return error instanceof Error ? error.message : t('common.unknownError')
}

const readHTTPError = async (response: Response): Promise<string> => {
  try {
    const payload = await response.json() as { message?: string; reason?: string }
    const localizedKey = payload.reason ? testErrorTranslationKeys[payload.reason] : undefined
    if (localizedKey) return t(localizedKey)
    return payload.message || t('common.connectionTest.requestFailed', { status: response.status })
  } catch {
    return t('common.connectionTest.requestFailed', { status: response.status })
  }
}

const testErrorTranslationKeys: Record<string, string> = {
  GROUP_TEST_DISABLED: 'common.connectionTest.errors.groupDisabled',
  API_KEY_TEST_DISABLED: 'common.connectionTest.errors.keyDisabled',
  API_KEY_TEST_EXPIRED: 'common.connectionTest.errors.keyExpired',
  API_KEY_TEST_QUOTA_EXHAUSTED: 'common.connectionTest.errors.quotaExhausted',
  API_KEY_TEST_GROUP_REQUIRED: 'common.connectionTest.errors.groupRequired',
  API_KEY_TEST_GROUP_DISABLED: 'common.connectionTest.errors.groupDisabled'
}

const handleEvent = (event: TestEvent) => {
  switch (event.type) {
    case 'status':
      if (event.status === 'route_selected') {
        addLine(t('common.connectionTest.routeSelected'), 'text-cyan-300')
      } else if (event.text) {
        addLine(event.text, 'text-cyan-300')
      }
      break
    case 'test_start':
      addLine(t('common.connectionTest.connected'), 'text-emerald-400')
      if (event.model) addLine(t('common.connectionTest.usingModel', { model: event.model }), 'text-cyan-300')
      addLine(t('common.connectionTest.response'), 'text-amber-300')
      break
    case 'content':
      if (event.text) {
        streamingContent.value += event.text
        void scrollToBottom()
      }
      break
    case 'test_complete':
      if (streamingContent.value) {
        addLine(streamingContent.value, 'whitespace-pre-wrap break-words text-emerald-300')
        streamingContent.value = ''
      }
      if (event.success) {
        status.value = 'success'
      } else {
        setError(event.error || t('common.connectionTest.failed'))
      }
      break
    case 'error':
      setError(event.error || t('common.connectionTest.failed'))
      break
  }
}

// EventSource 仅支持 GET，因此这里手动解析 POST 返回的 SSE 数据流。
const startTest = async () => {
  if (!canStart.value || !props.targetId) return
  resetOutput()
  status.value = 'connecting'
  addLine(t('common.connectionTest.starting', { name: props.targetName }), 'text-blue-300')
  abortStream()
  abortController = new AbortController()

  const path = props.targetType === 'group'
    ? `/admin/groups/${props.targetId}/test`
    : `/keys/${props.targetId}/test`
  const scopeHeader = props.targetType === 'group'
    ? ADMIN_UI_REQUEST_HEADER
    : USER_UI_REQUEST_HEADER

  try {
    const response = await fetch(buildApiUrl(path), {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
        'Content-Type': 'application/json',
        [scopeHeader]: '1'
      },
      body: JSON.stringify({ model_id: String(selectedModel.value || '') }),
      signal: abortController.signal
    })
    if (!response.ok) throw new Error(await readHTTPError(response))

    const reader = response.body?.getReader()
    if (!reader) throw new Error(t('common.connectionTest.noResponse'))

    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      for (const line of lines) {
        if (!line.startsWith('data:')) continue
        const payload = line.slice(5).trim()
        if (!payload) continue
        try {
          handleEvent(JSON.parse(payload) as TestEvent)
        } catch {
          addLine(t('common.connectionTest.invalidResponse'), 'text-red-300')
        }
      }
    }
    if (status.value === 'connecting') {
      throw new Error(t('common.connectionTest.incompleteResponse'))
    }
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      status.value = 'idle'
      return
    }
    const message = readErrorMessage(error)
    setError(message)
    addLine(message, 'text-red-300')
  } finally {
    abortController = null
  }
}

const copyOutput = () => {
  const content = [...outputLines.value.map((line) => line.text), streamingContent.value]
    .filter(Boolean)
    .join('\n')
  copyToClipboard(content, t('common.connectionTest.outputCopied'))
}

watch(
  () => [props.show, props.targetType, props.targetId] as const,
  async ([show]) => {
    if (!show) {
      abortStream()
      return
    }
    resetOutput()
    await loadModels()
  },
  { immediate: true }
)
</script>
