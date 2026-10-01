<template>
  <footer class="relative z-10 border-t border-gray-200/70 px-4 py-7 dark:border-dark-800/80 sm:px-6 sm:py-8">
    <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 text-center sm:flex-row sm:text-left">
      <div class="flex items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="h-7 w-7 rounded-lg object-contain" />
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </div>
      <div class="flex items-center gap-2 sm:gap-3">
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="inline-flex h-9 items-center gap-2 rounded-lg px-2.5 text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-primary-300">
          <Icon name="book" size="sm" />
          {{ t('home.docs') }}
        </a>
        <span v-if="docUrl" class="h-4 w-px bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
        <LocaleSwitcher placement="top" />
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'

defineProps<{
  siteName: string
  siteLogo: string
  docUrl: string
  currentYear: number
}>()

const { t } = useI18n()
</script>
