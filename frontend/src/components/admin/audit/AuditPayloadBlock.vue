<template>
  <section>
    <div class="mb-1.5 flex min-h-6 items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <h4 class="text-xs font-bold uppercase text-gray-400">{{ title }}</h4>
        <span
          v-if="contentType"
          class="truncate rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-500 dark:bg-dark-700 dark:text-gray-400"
          :title="contentType"
        >
          {{ contentType }}
        </span>
      </div>
      <button
        v-if="content"
        type="button"
        class="flex h-7 w-7 shrink-0 items-center justify-center rounded text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
        :title="t('common.copy')"
        @click="copyToClipboard(content)"
      >
        <Icon :name="copied ? 'check' : 'copy'" size="sm" />
        <span class="sr-only">{{ t('common.copy') }}</span>
      </button>
    </div>
    <pre
      v-if="content"
      :class="[
        'overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-4 font-mono text-xs leading-relaxed text-gray-600 dark:bg-dark-900 dark:text-gray-400',
        tall ? 'max-h-[26rem]' : 'max-h-48'
      ]"
    >{{ content }}</pre>
    <div
      v-else
      class="flex min-h-20 items-center justify-center rounded-lg border border-dashed border-gray-200 px-4 text-sm text-gray-400 dark:border-dark-700 dark:text-gray-500"
    >
      {{ emptyText }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'

// 审计正文统一组件：负责内容类型标记、空状态、滚动与复制能力。
withDefaults(defineProps<{
  title: string
  content: string
  contentType?: string
  emptyText?: string
  tall?: boolean
}>(), {
  contentType: '',
  emptyText: '—',
  tall: false
})

const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()
</script>
