<template>
  <footer class="relative z-10 border-t border-gray-200/70 px-4 py-7 dark:border-dark-800/80 sm:px-6 sm:py-8">
    <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 text-center sm:flex-row sm:text-left">
      <div class="flex items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="h-7 w-7 rounded-lg object-contain" />
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </div>
      <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition-colors hover:text-primary-700 dark:text-dark-400 dark:hover:text-primary-300">
        <Icon name="book" size="sm" />
        {{ t('home.docs') }}
      </a>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  siteName: string
  siteLogo: string
  docUrl: string
  currentYear: number
}>()

const { t } = useI18n()
</script>
