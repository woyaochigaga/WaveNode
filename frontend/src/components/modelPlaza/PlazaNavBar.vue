<template>
  <header
    class="glass sticky top-0 z-30 border-b border-gray-200/50 dark:border-dark-700/50"
  >
    <div class="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-3.5 sm:px-6">
      <!-- 站名返回首页，文档入口沿用公开配置。 -->
      <div class="flex min-w-0 flex-1 items-center gap-3">
        <RouterLink v-if="settings" to="/home" class="flex min-w-0 items-center gap-3" :title="siteName">
          <span class="truncate text-base font-semibold text-gray-950 dark:text-white">
            {{ siteName }}
          </span>
        </RouterLink>
        <template v-else>
          <span class="h-5 w-28 animate-pulse rounded bg-gray-200 dark:bg-dark-700" aria-hidden="true"></span>
        </template>
      </div>

      <nav class="flex shrink-0 items-center gap-1 text-sm text-gray-600 dark:text-dark-300" :aria-label="t('nav.modelPlaza')">
        <span class="hidden rounded-lg bg-primary-50 px-2.5 py-2 font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300 sm:inline-flex">{{ t('nav.modelPlaza') }}</span>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 hover:text-primary-600 dark:hover:text-primary-300" :title="t('nav.docs')" :aria-label="t('nav.docs')">
          <Icon name="book" size="sm" />
          <span class="hidden md:inline">{{ t('nav.docs') }}</span>
        </a>
        <button type="button" class="rounded-lg p-2.5 hover:text-primary-600 dark:hover:text-primary-300" :title="isDark ? t('home.switchToLight') : t('home.switchToDark')" :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="toggleTheme">
          <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
        </button>
      </nav>

      <RouterLink
        v-if="isAuthenticated"
        :to="backTarget"
        class="inline-flex flex-shrink-0 items-center justify-center gap-1.5 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
      >
        {{ t('modelPlaza.nav.backToDashboard') }}
      </RouterLink>
      <RouterLink
        v-else
        :to="{ path: '/login', query: { redirect: '/model-plaza' } }"
        class="inline-flex flex-shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500"
      >
        {{ t('modelPlaza.nav.login') }}
      </RouterLink>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || 'WaveNode')
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const backTarget = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const isDark = ref(document.documentElement.classList.contains('dark'))

/** 切换独立广场主题，并保持与首页相同的本地主题偏好。 */
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>
