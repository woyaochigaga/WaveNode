<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'

// title 是配置名称，正文通过插槽复用该配置的用途、填写规则和注意事项。
defineProps<{ title: string }>()
const { t } = useI18n()
</script>

<template>
  <HelpTooltip width-class="w-80">
    <template #trigger>
      <button
        type="button"
        class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-gray-400 transition-colors hover:bg-primary-50 hover:text-primary-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-primary-900/30 dark:hover:text-primary-300"
        :aria-label="t('admin.settings.helpLabel', { setting: title })"
        @click.prevent
      >
        <Icon name="questionCircle" size="sm" />
      </button>
    </template>
    <p class="mb-1 font-semibold text-white">{{ title }}</p>
    <div class="whitespace-pre-line font-normal text-gray-200"><slot /></div>
  </HelpTooltip>
</template>
