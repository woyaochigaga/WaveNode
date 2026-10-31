<template>
  <header class="landing-header sticky top-2 z-50 px-3 sm:px-5">
    <nav
      class="landing-nav mx-auto flex w-full flex-wrap items-center justify-between gap-3 rounded-2xl px-4 py-2.5 sm:gap-4 sm:px-5"
      :class="{ 'is-scrolled': isScrolled }"
    >
      <div class="flex min-w-0 flex-1 items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="landing-logo h-10 w-10 shrink-0 rounded-xl object-contain" />
        <span class="max-w-[42vw] truncate text-sm font-semibold tracking-tight text-gray-900 dark:text-white sm:text-base">{{ siteName }}</span>
      </div>

      <div class="flex max-w-full flex-wrap items-center justify-end gap-1.5 sm:gap-2">
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="flex h-10 w-10 items-center justify-center rounded-xl text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white" :title="t('home.viewDocs')">
          <Icon name="book" size="md" />
        </a>
        <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="flex h-10 items-center gap-2 rounded-xl px-2.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white" :title="t('nav.modelPlaza')">
          <Icon name="grid" size="md" />
          <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
        </router-link>
        <button type="button" class="flex h-10 w-10 items-center justify-center rounded-xl text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white" :title="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="emit('toggle-theme')">
          <Icon v-if="isDark" name="sun" size="md" />
          <Icon v-else name="moon" size="md" />
        </button>
        <SpecularButton
          :to="isAuthenticated ? dashboardPath : '/login'"
          size="sm"
          :radius="12"
          :intensity="1.25"
          :shine-size="12"
          :shine-fade="34"
          tint="#0f766e"
          :tint-opacity="0.94"
          line-color="#ccfbf1"
          base-color="#2dd4bf"
          text-color="#f0fdfa"
          class="header-dashboard-button group ml-1"
        >
          <Icon :name="isAuthenticated ? 'grid' : 'login'" size="sm" class="text-primary-100" />
          <span>{{ isAuthenticated ? t('home.dashboard') : t('home.login') }}</span>
          <Icon name="arrowRight" size="sm" class="transition-transform duration-300 group-hover:translate-x-1" />
        </SpecularButton>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import SpecularButton from '@/components/ui/SpecularButton.vue'

defineProps<{
  siteName: string
  siteLogo: string
  docUrl: string
  showModelPlazaEntry: boolean
  isDark: boolean
  isAuthenticated: boolean
  dashboardPath: string
}>()

const emit = defineEmits<{
  (event: 'toggle-theme'): void
}>()

const { t } = useI18n()
const isScrolled = ref(false)

// 滚动超过首个间距后进入紧凑吸顶态，同时驱动导航宽度和高度动画。
function updateScrollState() {
  isScrolled.value = window.scrollY > 12
}

onMounted(() => {
  updateScrollState()
  window.addEventListener('scroll', updateScrollState, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', updateScrollState)
})
</script>

<style scoped>
.landing-header {
  /* 吸顶容器不再挤占首屏，让 Hero 背景能够延伸到导航下方。 */
  min-height: 62px;
  margin-bottom: -62px;
  background: transparent;
}

.landing-nav {
  position: relative;
  z-index: 1;
  max-width: 80rem;
  min-height: 62px;
  border-color: transparent;
  background: transparent;
  box-shadow: none;
  transition:
    max-width 480ms cubic-bezier(0.22, 1, 0.36, 1),
    min-height 360ms cubic-bezier(0.22, 1, 0.36, 1),
    padding 360ms cubic-bezier(0.22, 1, 0.36, 1),
    background-color 280ms ease,
    border-color 280ms ease,
    border-radius 360ms ease,
    box-shadow 360ms ease,
    transform 360ms cubic-bezier(0.22, 1, 0.36, 1);
}

.landing-header::before {
  content: none;
}

.landing-logo {
  transition:
    width 360ms cubic-bezier(0.22, 1, 0.36, 1),
    height 360ms cubic-bezier(0.22, 1, 0.36, 1),
    border-radius 360ms ease;
}

.landing-nav.is-scrolled {
  max-width: 68rem;
  min-height: 54px;
  padding-top: 0.45rem;
  padding-bottom: 0.45rem;
  border-color: rgba(20, 184, 166, 0.2);
  border-radius: 14px;
  background: rgba(237, 248, 247, 0.9);
  box-shadow:
    0 18px 50px -28px rgba(15, 118, 110, 0.32),
    0 0 0 1px rgba(255, 255, 255, 0.32);
  transform: translateY(-2px);
}

.landing-nav.is-scrolled .landing-logo {
  width: 2.125rem;
  height: 2.125rem;
  border-radius: 10px;
}

.header-dashboard-button {
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.14),
    0 9px 24px rgba(15, 118, 110, 0.2),
    0 0 0 1px rgba(20, 184, 166, 0.12);
}

:global(.dark .landing-nav) {
  border-color: transparent;
  background: transparent;
  box-shadow: none;
}

:global(.dark .landing-nav.is-scrolled) {
  border-color: rgba(94, 234, 212, 0.22);
  background: rgba(9, 31, 42, 0.9);
  box-shadow:
    0 22px 60px -28px rgba(3, 34, 43, 0.72),
    0 0 0 1px rgba(94, 234, 212, 0.06);
}

@media (max-width: 639px) {
  .landing-nav,
  .landing-nav.is-scrolled {
    max-width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .landing-nav,
  .landing-logo {
    transition: none;
  }
}
</style>
