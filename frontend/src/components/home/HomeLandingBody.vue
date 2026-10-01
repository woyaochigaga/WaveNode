<template>
  <main
    ref="landingBody"
    class="relative z-10 flex-1 px-4 pb-16 pt-10 sm:px-6 sm:pt-16 lg:pb-24 lg:pt-20"
    :style="interactionStyle"
    @pointermove="handlePointerMove"
    @pointerleave="resetPointer"
  >
    <!-- 首屏背景只使用 CSS 图层，不阻挡链接、按钮和卡片的真实交互。 -->
    <div class="hero-interaction-layer" aria-hidden="true">
      <div class="hero-grid"></div>
      <div class="hero-trace hero-trace--one"></div>
      <div class="hero-trace hero-trace--two"></div>
      <div class="hero-sweep"></div>
      <div class="hero-cursor-light"></div>
    </div>
    <div class="home-aurora" aria-hidden="true"></div>

    <div class="landing-content mx-auto max-w-7xl">
      <section
        ref="heroSection"
        class="hero-section home-reveal relative grid items-center gap-12 py-8 lg:grid-cols-[1.05fr_0.95fr] lg:gap-16 lg:py-12"
        data-reveal
      >
        <div class="relative z-10 text-center lg:text-left">
          <h1 class="home-title-shine mx-auto max-w-3xl break-words text-4xl font-bold leading-[1.08] tracking-tight sm:text-5xl lg:mx-0 lg:text-6xl xl:text-7xl">
            {{ siteName }}
          </h1>
          <p class="mx-auto mt-6 max-w-2xl whitespace-pre-wrap break-words text-base leading-8 text-gray-600 dark:text-dark-300 sm:text-lg lg:mx-0">
            {{ siteSubtitle }}
          </p>
          <div class="mt-8 flex flex-wrap items-center justify-center gap-3 lg:justify-start">
            <SpecularButton
              :to="isAuthenticated ? dashboardPath : '/login'"
              size="lg"
              :radius="14"
              :intensity="1.3"
              :shine-size="12"
              :shine-fade="36"
              tint="#0f766e"
              :tint-opacity="0.94"
              line-color="#ccfbf1"
              base-color="#2dd4bf"
              text-color="#f0fdfa"
              class="landing-specular-cta group"
            >
              <span>{{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}</span>
              <Icon name="arrowRight" size="md" :stroke-width="2" class="transition-transform duration-300 group-hover:translate-x-1" />
            </SpecularButton>
          </div>
        </div>

        <div class="relative mx-auto w-full max-w-xl lg:mx-0 lg:justify-self-end">
          <div class="terminal-halo" aria-hidden="true"></div>
          <div class="terminal-container">
            <div class="terminal-window">
              <div class="terminal-header">
                <div class="terminal-buttons" aria-hidden="true">
                  <span class="terminal-dot terminal-dot--close"></span>
                  <span class="terminal-dot terminal-dot--minimize"></span>
                  <span class="terminal-dot terminal-dot--maximize"></span>
                </div>
                <span class="terminal-title">gateway / request.flow</span>
                <span class="terminal-live"><span></span> live</span>
              </div>
              <div class="terminal-body">
                <div class="terminal-meta">REQUEST&nbsp;&nbsp; 01 / 01</div>
                <div class="code-line line-1">
                  <span class="code-prompt">$</span>
                  <span class="code-cmd">curl</span>
                  <span class="code-flag">-X POST</span>
                  <span class="code-url">/v1/messages</span>
                </div>
                <div class="code-line line-2"><span class="code-comment"># Routing to upstream...</span></div>
                <div class="code-line line-3">
                  <span class="code-success">200 OK</span>
                  <span class="code-response">{ "content": "Hello!" }</span>
                </div>
                <div class="code-line line-4">
                  <span class="code-prompt">$</span>
                  <span class="cursor"></span>
                </div>
              </div>
              <div class="terminal-footer">
                <span>SECURE ROUTING</span>
                <span>STREAM READY</span>
                <span>STATUS&nbsp; 200</span>
              </div>
            </div>
          </div>
          <div class="terminal-float terminal-float--top" aria-hidden="true">
            <span class="terminal-float-dot"></span>
            <span>Unified API</span>
          </div>
          <div class="terminal-float terminal-float--bottom" aria-hidden="true">
            <Icon name="shield" size="sm" />
            <span>Secure by design</span>
          </div>
        </div>
        <button
          type="button"
          class="hero-scroll-cue"
          :aria-label="t('home.features.unifiedGateway')"
          @click="scrollToFeatures"
        >
          <span class="hero-scroll-cue__line"></span>
          <Icon name="arrowDown" size="sm" />
        </button>
        <div class="hero-scroll-meter" aria-hidden="true"><span></span></div>
      </section>

      <section class="home-reveal mt-14 border-y border-gray-200/70 py-6 dark:border-dark-700/70 sm:mt-20 sm:py-7" data-reveal style="--reveal-delay: 80ms">
        <div class="flex flex-wrap items-center justify-center gap-x-8 gap-y-4 sm:gap-x-12">
          <div v-for="tag in featureTags" :key="tag.key" class="flex items-center gap-2.5 text-sm font-medium text-gray-600 dark:text-dark-300">
            <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-500/10 text-primary-600 dark:text-primary-300">
              <Icon :name="tag.icon" size="sm" />
            </span>
            <span>{{ t(tag.key) }}</span>
          </div>
        </div>
      </section>

      <section id="home-features" class="mt-16 scroll-mt-24 sm:mt-24">
        <div class="mb-7 home-reveal sm:mb-9" data-reveal>
          <p class="mb-2 text-xs font-bold uppercase tracking-[0.2em] text-primary-600 dark:text-primary-400">
            {{ t('home.features.unifiedGateway') }}
          </p>
          <h2 class="max-w-2xl text-2xl font-semibold tracking-tight text-gray-900 dark:text-white sm:text-3xl">
            {{ siteSubtitle }}
          </h2>
        </div>

        <div class="feature-gallery home-reveal" data-reveal style="--reveal-delay: 120ms">
          <CircularGallery
            :items="galleryItems"
            :is-dark="isDark"
            :aria-label="t('home.solutions.title')"
          />
        </div>
      </section>

      <section class="home-reveal mt-16 sm:mt-24" data-reveal style="--reveal-delay: 100ms">
        <div class="mb-7 flex flex-col gap-3 sm:mb-8 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p class="mb-2 text-xs font-bold uppercase tracking-[0.2em] text-primary-600 dark:text-primary-400">
              {{ t('home.providers.supported') }}
            </p>
            <h2 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white sm:text-3xl">
              {{ t('home.providers.title') }}
            </h2>
          </div>
          <p class="max-w-xl text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('home.providers.description') }}
          </p>
        </div>

        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div
            v-for="(provider, index) in providerCards"
            :key="provider.key"
            class="home-provider home-reveal relative isolate flex min-w-0 items-center gap-3 overflow-hidden rounded-xl border border-gray-200/70 bg-white/55 px-4 py-4 backdrop-blur transition-colors hover:border-primary-300/70 hover:bg-white/85 dark:border-white/[0.08] dark:bg-[#070a12]/80 dark:hover:border-primary-800 dark:hover:bg-[#0a0f1c]"
            data-reveal
            :style="{ '--reveal-delay': (140 + index * 65) + 'ms' }"
            @pointermove="updateSpotlight"
            @pointerleave="resetCardTilt"
          >
            <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br text-sm font-bold text-white shadow-sm" :class="provider.markTone">{{ provider.mark }}</span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-semibold text-gray-800 dark:text-dark-100">
                {{ provider.labelKey ? t(provider.labelKey) : provider.label }}
              </span>
              <span class="mt-1 block text-[10px] font-semibold uppercase tracking-wider" :class="provider.statusKey === 'home.providers.soon' ? 'text-gray-500 dark:text-dark-400' : 'text-primary-700 dark:text-primary-400'">
                {{ t(provider.statusKey) }}
              </span>
            </span>
          </div>
        </div>
      </section>

      <HomeLandingCta
        class="home-reveal mt-16 sm:mt-24"
        data-reveal
        style="--reveal-delay: 80ms"
        :site-name="siteName"
        :site-subtitle="siteSubtitle"
        :is-authenticated="isAuthenticated"
        :dashboard-path="dashboardPath"
      />
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import CircularGallery from '@/components/ui/CircularGallery.vue'
import SpecularButton from '@/components/ui/SpecularButton.vue'
import HomeLandingCta from '@/components/home/HomeLandingCta.vue'

defineProps<{
  siteName: string
  siteSubtitle: string
  isDark: boolean
  isAuthenticated: boolean
  dashboardPath: string
}>()

const { t } = useI18n()
const landingBody = ref<HTMLElement | null>(null)
const heroSection = ref<HTMLElement | null>(null)
const interactionStyle = ref<Record<string, string>>({
  '--pointer-x': '50%',
  '--pointer-y': '28%',
  '--pointer-shift-x': '0px',
  '--pointer-shift-y': '0px',
  '--hero-rotate-x': '0deg',
  '--hero-rotate-y': '0deg',
  '--scroll-progress': '0',
  '--scroll-shift-y': '0px',
})
let revealObserver: IntersectionObserver | null = null
let pointerFrame: number | null = null
let scrollFrame: number | null = null
let pointerX = 0
let pointerY = 0

// 标签配置沿用原有翻译键，避免改变多语言内容。
const featureTags = [
  { key: 'home.tags.subscriptionToApi', icon: 'swap' },
  { key: 'home.tags.stickySession', icon: 'shield' },
  { key: 'home.tags.realtimeBilling', icon: 'chart' },
] as const

// 复用现有多语言能力文案组成六项画廊，避免首页出现未配置的新营销内容。
const galleryItems = computed(() => [
  {
    code: '01 / ACCESS',
    title: t('home.features.unifiedGateway'),
    description: t('home.features.unifiedGatewayDesc'),
    accent: '#0ea5e9',
    secondary: '#14b8a6',
  },
  {
    code: '02 / ROUTE',
    title: t('home.features.multiAccount'),
    description: t('home.features.multiAccountDesc'),
    accent: '#14b8a6',
    secondary: '#22c55e',
  },
  {
    code: '03 / BILL',
    title: t('home.features.balanceQuota'),
    description: t('home.features.balanceQuotaDesc'),
    accent: '#8b5cf6',
    secondary: '#3b82f6',
  },
  {
    code: '04 / SESSION',
    title: t('home.tags.stickySession'),
    description: t('home.comparison.items.stability.us'),
    accent: '#06b6d4',
    secondary: '#14b8a6',
  },
  {
    code: '05 / MODELS',
    title: t('home.comparison.items.models.feature'),
    description: t('home.comparison.items.models.us'),
    accent: '#3b82f6',
    secondary: '#6366f1',
  },
  {
    code: '06 / CONTROL',
    title: t('home.comparison.items.control.feature'),
    description: t('home.comparison.items.control.us'),
    accent: '#f59e0b',
    secondary: '#f97316',
  },
])

// 服务商列表保留首页现有顺序及支持状态。
const providerCards = [
  { key: 'claude', labelKey: 'home.providers.claude', label: '', mark: 'C', markTone: 'from-orange-400 to-orange-600', statusKey: 'home.providers.supported' },
  { key: 'gpt', labelKey: '', label: 'GPT', mark: 'G', markTone: 'from-emerald-500 to-green-700', statusKey: 'home.providers.supported' },
  { key: 'gemini', labelKey: 'home.providers.gemini', label: '', mark: 'G', markTone: 'from-blue-500 to-indigo-600', statusKey: 'home.providers.supported' },
  { key: 'antigravity', labelKey: 'home.providers.antigravity', label: '', mark: 'A', markTone: 'from-rose-500 to-pink-700', statusKey: 'home.providers.supported' },
  { key: 'more', labelKey: 'home.providers.more', label: '', mark: '+', markTone: 'from-gray-500 to-gray-700', statusKey: 'home.providers.soon' },
] as const

// 使用一帧合并鼠标更新，避免高频 pointermove 直接触发重复样式计算。
function renderPointerPosition() {
  pointerFrame = null
  interactionStyle.value = {
    ...interactionStyle.value,
    '--pointer-x': String(50 + pointerX * 50) + '%',
    '--pointer-y': String(28 + pointerY * 42) + '%',
    '--pointer-shift-x': String(pointerX * 18) + 'px',
    '--pointer-shift-y': String(pointerY * 12) + 'px',
    '--hero-rotate-x': String(pointerY * -1.2) + 'deg',
    '--hero-rotate-y': String(pointerX * 1.4) + 'deg',
  }
}

function schedulePointerRender() {
  if (pointerFrame !== null) return
  pointerFrame = window.requestAnimationFrame(renderPointerPosition)
}

// 鼠标控制首屏光束和终端的轻微视差，触摸设备保持静态布局。
function handlePointerMove(event: PointerEvent) {
  if (event.pointerType === 'touch') return
  const bounds = landingBody.value?.getBoundingClientRect()
  if (!bounds) return

  pointerX = Math.max(-1, Math.min(1, ((event.clientX - bounds.left) / bounds.width - 0.5) * 2))
  pointerY = Math.max(-1, Math.min(1, ((event.clientY - bounds.top) / bounds.height - 0.28) * 1.5))
  schedulePointerRender()
}

function resetPointer() {
  pointerX = 0
  pointerY = 0
  schedulePointerRender()
}

// 将滚动距离转换成首屏偏移量，让滚轮行为参与背景和 Hero 的过渡。
function updateScrollProgress() {
  if (scrollFrame !== null) return
  scrollFrame = window.requestAnimationFrame(() => {
    scrollFrame = null
    const heroHeight = heroSection.value?.offsetHeight || window.innerHeight
    const progress = Math.max(0, Math.min(1, window.scrollY / Math.max(heroHeight * 0.72, 1)))
    interactionStyle.value = {
      ...interactionStyle.value,
      '--scroll-progress': String(progress),
      '--scroll-shift-y': String(progress * -34) + 'px',
    }
  })
}

// 将当前指针位置写入 CSS 变量，驱动卡片上的柔和聚光与轻微倾斜。
function updateSpotlight(event: PointerEvent) {
  const card = event.currentTarget
  if (!(card instanceof HTMLElement) || event.pointerType === 'touch') return

  const bounds = card.getBoundingClientRect()
  const x = (event.clientX - bounds.left) / bounds.width - 0.5
  const y = (event.clientY - bounds.top) / bounds.height - 0.5
  card.style.setProperty('--spotlight-x', String(event.clientX - bounds.left) + 'px')
  card.style.setProperty('--spotlight-y', String(event.clientY - bounds.top) + 'px')
  card.style.setProperty('--card-rotate-x', String(y * -4) + 'deg')
  card.style.setProperty('--card-rotate-y', String(x * 4) + 'deg')
}

function resetCardTilt(event: PointerEvent) {
  const card = event.currentTarget
  if (!(card instanceof HTMLElement)) return
  card.style.setProperty('--card-rotate-x', '0deg')
  card.style.setProperty('--card-rotate-y', '0deg')
}

function scrollToFeatures() {
  const target = document.getElementById('home-features')
  if (!target) return

  target.scrollIntoView({
    behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth',
    block: 'start',
  })
}

onMounted(() => {
  const targets = landingBody.value?.querySelectorAll<HTMLElement>('[data-reveal]')
  if (!targets?.length) return

  // 不支持 IntersectionObserver 时直接显示全部内容，动画不会成为使用前提。
  if (!('IntersectionObserver' in window)) {
    targets.forEach((target) => target.classList.add('is-visible'))
  } else {
    revealObserver = new IntersectionObserver((entries, observer) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return
        entry.target.classList.add('is-visible')
        observer.unobserve(entry.target)
      })
    }, { threshold: 0.12, rootMargin: '0px 0px -36px 0px' })
    targets.forEach((target) => revealObserver?.observe(target))
  }

  updateScrollProgress()
  window.addEventListener('scroll', updateScrollProgress, { passive: true })
  window.addEventListener('resize', updateScrollProgress)
})

onBeforeUnmount(() => {
  revealObserver?.disconnect()
  window.removeEventListener('scroll', updateScrollProgress)
  window.removeEventListener('resize', updateScrollProgress)
  if (pointerFrame !== null) window.cancelAnimationFrame(pointerFrame)
  if (scrollFrame !== null) window.cancelAnimationFrame(scrollFrame)
})
</script>

<style scoped>
.hero-interaction-layer {
  position: absolute;
  z-index: 0;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  background:
    radial-gradient(42rem circle at var(--pointer-x) var(--pointer-y), rgba(13, 148, 136, 0.16), transparent 62%),
    linear-gradient(180deg, rgba(14, 116, 144, 0.025) 0%, rgba(20, 184, 166, calc(0.045 + var(--scroll-progress) * 0.035)) 52%, transparent 100%);
}

.landing-content {
  position: relative;
  z-index: 1;
}

.hero-grid {
  position: absolute;
  inset: 0;
  opacity: 0.82;
  background-image:
    linear-gradient(rgba(15, 118, 110, 0.075) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 118, 110, 0.075) 1px, transparent 1px),
    linear-gradient(rgba(15, 23, 42, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 23, 42, 0.06) 1px, transparent 1px);
  background-size: 54px 54px, 54px 54px, 216px 216px, 216px 216px;
  transform: translate3d(calc(var(--pointer-shift-x) * -0.45), calc(var(--pointer-shift-y) * -0.45), 0);
  mask-image: linear-gradient(180deg, black 0%, rgba(0, 0, 0, 0.72) 42%, transparent 86%);
  transition: transform 180ms ease-out;
}

.hero-trace {
  position: absolute;
  width: 48rem;
  height: 16rem;
  border: 1px solid rgba(13, 148, 136, 0.22);
  border-radius: 48%;
  box-shadow: 0 0 34px rgba(13, 148, 136, 0.055);
  transform: translate3d(var(--pointer-shift-x), var(--pointer-shift-y), 0) rotate(-12deg);
  transition: transform 420ms cubic-bezier(0.2, 0.7, 0.2, 1);
}

.hero-trace::after {
  position: absolute;
  right: 13%;
  bottom: -1px;
  width: 5px;
  height: 5px;
  border-radius: 9999px;
  background: #2dd4bf;
  box-shadow: 0 0 18px 4px rgba(45, 212, 191, 0.34);
  content: '';
}

.hero-trace--one {
  top: 4rem;
  left: -8rem;
  animation: trace-drift 16s ease-in-out infinite alternate;
}

.hero-trace--two {
  top: 15rem;
  right: -14rem;
  border-color: rgba(37, 99, 235, 0.2);
  transform: translate3d(calc(var(--pointer-shift-x) * -0.8), calc(var(--pointer-shift-y) * -0.8), 0) rotate(18deg);
  animation: trace-drift 21s ease-in-out -8s infinite alternate-reverse;
}

.hero-sweep {
  position: absolute;
  top: -18%;
  left: 42%;
  width: 18rem;
  height: 140%;
  opacity: 0.12;
  background: linear-gradient(90deg, transparent, rgba(94, 234, 212, 0.5), transparent);
  filter: blur(24px);
  transform: translate3d(calc(var(--pointer-shift-x) * -1.5), calc(var(--scroll-progress) * 5rem), 0) rotate(18deg);
  animation: sweep-drift 12s ease-in-out infinite alternate;
}

.hero-cursor-light {
  position: absolute;
  top: var(--pointer-y);
  left: var(--pointer-x);
  width: min(42rem, 78vw);
  aspect-ratio: 1;
  border-radius: 50%;
  opacity: calc(0.3 - var(--scroll-progress) * 0.16);
  background: radial-gradient(circle, rgba(20, 184, 166, 0.18), rgba(37, 99, 235, 0.07) 34%, transparent 68%);
  filter: blur(10px);
  transform: translate(-50%, -50%);
  transition: top 220ms ease-out, left 220ms ease-out, opacity 300ms ease;
}

.hero-section {
  transform-style: preserve-3d;
  will-change: transform;
}

.hero-section.is-visible {
  transform: translate3d(var(--pointer-shift-x), calc(var(--pointer-shift-y) + var(--scroll-shift-y)), 0) rotateX(var(--hero-rotate-x)) rotateY(var(--hero-rotate-y));
}

.hero-scroll-cue {
  position: absolute;
  right: 50%;
  bottom: -4.2rem;
  display: flex;
  align-items: center;
  gap: 0.45rem;
  color: rgba(15, 118, 110, 0.72);
  transform: translateX(50%);
  transition: color 180ms ease, transform 180ms ease;
}

.hero-scroll-cue:hover {
  color: #0f766e;
  transform: translate(50%, 3px);
}

.hero-scroll-cue__line {
  width: 2.8rem;
  height: 1px;
  background: currentColor;
  opacity: 0.4;
}

.hero-scroll-meter {
  position: absolute;
  right: 0;
  bottom: -2.5rem;
  left: 0;
  height: 1px;
  overflow: hidden;
  background: rgba(148, 163, 184, 0.2);
}

.hero-scroll-meter span {
  display: block;
  width: calc(var(--scroll-progress) * 100%);
  height: 100%;
  background: linear-gradient(90deg, transparent, #14b8a6 28%, #60a5fa);
  box-shadow: 0 0 12px rgba(20, 184, 166, 0.65);
  transition: width 120ms linear;
}

.home-aurora {
  position: absolute;
  z-index: 0;
  top: -5rem;
  left: 50%;
  width: min(78rem, 110vw);
  height: 34rem;
  transform: translateX(-50%);
  pointer-events: none;
  opacity: 0.72;
  filter: blur(48px);
}

.home-aurora::before,
.home-aurora::after {
  position: absolute;
  content: '';
  border-radius: 9999px;
  animation: aurora-drift 18s ease-in-out infinite alternate;
}

.home-aurora::before {
  inset: 2% 15% 26% 15%;
  background: linear-gradient(115deg, rgba(45, 212, 191, 0.26), rgba(59, 130, 246, 0.14), rgba(129, 140, 248, 0.12));
}

.home-aurora::after {
  inset: 24% 4% 5% 42%;
  background: linear-gradient(125deg, rgba(20, 184, 166, 0.2), rgba(14, 165, 233, 0.1));
  animation-delay: -7s;
}

.home-title-shine {
  color: transparent;
  background: linear-gradient(110deg, #0f172a 10%, #0f766e 48%, #1d4ed8 74%, #0f172a 100%);
  background-clip: text;
  -webkit-background-clip: text;
  background-size: 180% auto;
  animation: title-shine 10s ease-in-out infinite alternate;
}

:global(.dark .home-title-shine) {
  background-image: linear-gradient(110deg, #f8fafc 10%, #5eead4 48%, #93c5fd 74%, #f8fafc 100%);
}

:global(.dark .hero-interaction-layer) {
  background:
    radial-gradient(42rem circle at var(--pointer-x) var(--pointer-y), rgba(20, 184, 166, 0.11), transparent 62%),
    linear-gradient(180deg, transparent 0%, rgba(8, 47, 73, calc(0.06 + var(--scroll-progress) * 0.04)) 52%, transparent 100%);
}

:global(.dark .hero-grid) {
  opacity: 0.28;
  background-image:
    linear-gradient(rgba(94, 234, 212, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(94, 234, 212, 0.06) 1px, transparent 1px);
}

.feature-gallery {
  margin-inline: clamp(-1rem, -2vw, -0.25rem);
}

:global(.dark .hero-trace) {
  border-color: rgba(45, 212, 191, 0.13);
}

:global(.dark .hero-trace--two) {
  border-color: rgba(96, 165, 250, 0.12);
}

:global(.dark .hero-scroll-cue) {
  color: rgba(94, 234, 212, 0.72);
}

:global(.dark .hero-scroll-cue:hover) {
  color: #99f6e4;
}

.terminal-container { position: relative; z-index: 1; perspective: 1200px; }

.terminal-window {
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 1.15rem;
  background: linear-gradient(145deg, #172033 0%, #0b1220 100%);
  box-shadow: 0 32px 90px -36px rgba(15, 23, 42, 0.65), 0 0 0 1px rgba(255, 255, 255, 0.04), inset 0 1px rgba(255, 255, 255, 0.08);
  transform: rotateY(-3deg) rotateX(2deg);
  transition: transform 500ms cubic-bezier(0.2, 0.75, 0.25, 1), box-shadow 500ms ease;
}

.terminal-window:hover {
  transform: rotateY(0) rotateX(0) translateY(-4px);
  box-shadow: 0 38px 100px -38px rgba(15, 23, 42, 0.72), 0 0 48px rgba(20, 184, 166, 0.12);
}

.terminal-header,
.terminal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  background: rgba(255, 255, 255, 0.025);
  padding: 0.85rem 1rem;
}

.terminal-footer {
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  border-bottom: 0;
  color: #64748b;
  font: 600 0.55rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.12em;
}

.terminal-buttons { display: flex; gap: 0.42rem; }
.terminal-dot { width: 0.58rem; height: 0.58rem; border-radius: 9999px; }
.terminal-dot--close { background: #fb7185; }
.terminal-dot--minimize { background: #fbbf24; }
.terminal-dot--maximize { background: #34d399; }
.terminal-title { color: #94a3b8; font: 500 0.68rem/1 ui-monospace, SFMono-Regular, Menlo, monospace; }

.terminal-live {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: #86efac;
  font: 600 0.62rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

.terminal-live span,
.terminal-float-dot {
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 9999px;
  background: #34d399;
  box-shadow: 0 0 12px rgba(52, 211, 153, 0.8);
}

.terminal-body {
  min-height: 13.6rem;
  padding: 1.5rem 1.2rem 1.1rem;
  color: #e2e8f0;
  font: 500 clamp(0.68rem, 2vw, 0.82rem)/2.2 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.terminal-meta { margin-bottom: 0.75rem; color: #64748b; font-size: 0.58rem; letter-spacing: 0.15em; }
.code-line { display: flex; align-items: center; gap: 0.55rem; flex-wrap: wrap; opacity: 0; animation: line-appear 500ms ease forwards; }
.line-1 { animation-delay: 250ms; }
.line-2 { animation-delay: 850ms; }
.line-3 { animation-delay: 1450ms; }
.line-4 { animation-delay: 2050ms; }
.code-prompt, .code-success { color: #34d399; }
.code-prompt { font-weight: 700; }
.code-cmd { color: #7dd3fc; }
.code-flag { color: #c4b5fd; }
.code-url { color: #5eead4; }
.code-comment { color: #64748b; font-style: italic; }
.code-success { border-radius: 0.3rem; background: rgba(52, 211, 153, 0.12); padding: 0.05rem 0.4rem; font-size: 0.68rem; font-weight: 700; }
.code-response { color: #fcd34d; }
.cursor { display: inline-block; width: 0.48rem; height: 0.95rem; background: #34d399; animation: cursor-blink 1s step-end infinite; }

.terminal-halo { position: absolute; inset: 16% 8%; border-radius: 50%; background: rgba(20, 184, 166, 0.22); filter: blur(55px); }

.terminal-float {
  position: absolute;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 0.8rem;
  background: rgba(255, 255, 255, 0.84);
  padding: 0.65rem 0.85rem;
  color: #334155;
  font-size: 0.7rem;
  font-weight: 600;
  box-shadow: 0 12px 34px rgba(15, 23, 42, 0.12);
  backdrop-filter: blur(12px);
  animation: float-y 5s ease-in-out infinite;
}

.terminal-float--top { top: 12%; right: -1rem; }
.terminal-float--bottom { bottom: 12%; left: -1.25rem; color: #0f766e; animation-delay: -2.4s; }
.terminal-float--bottom :deep(svg) { color: #0d9488; }
:global(.dark .terminal-float) { border-color: rgba(255, 255, 255, 0.12); background: rgba(15, 23, 42, 0.82); color: #e2e8f0; }
:global(.dark .terminal-float--bottom) { color: #99f6e4; }

.home-provider {
  --card-rotate-x: 0deg;
  --card-rotate-y: 0deg;
  transform-style: preserve-3d;
}

.home-provider {
  transform: perspective(900px) rotateX(var(--card-rotate-x)) rotateY(var(--card-rotate-y));
  transition: transform 220ms ease-out, background-color 240ms ease, border-color 240ms ease;
}

.home-provider:hover {
  transform: perspective(900px) translateY(-2px) rotateX(var(--card-rotate-x)) rotateY(var(--card-rotate-y));
}

.home-provider::before {
  position: absolute;
  z-index: 0;
  inset: 0;
  pointer-events: none;
  content: '';
  opacity: 0;
  background: radial-gradient(280px circle at var(--spotlight-x, 50%) var(--spotlight-y, 50%), rgba(45, 212, 191, 0.13), transparent 72%);
  transition: opacity 300ms ease;
}

.home-provider:hover::before { opacity: 1; }
.home-reveal { opacity: 0; transform: translateY(1.2rem); transition: opacity 650ms ease var(--reveal-delay, 0ms), transform 650ms cubic-bezier(0.2, 0.7, 0.2, 1) var(--reveal-delay, 0ms); }
.home-reveal.is-visible { opacity: 1; transform: translateY(0); }
.landing-specular-cta {
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.16),
    0 14px 34px rgba(15, 118, 110, 0.2),
    0 0 0 1px rgba(20, 184, 166, 0.12);
}

@keyframes aurora-drift {
  from { transform: translate3d(-2%, -1%, 0) rotate(-2deg) scale(0.96); }
  to { transform: translate3d(2%, 2%, 0) rotate(2deg) scale(1.04); }
}
@keyframes trace-drift {
  from { margin-left: -1.5rem; }
  to { margin-left: 1.5rem; }
}
@keyframes sweep-drift {
  from { opacity: 0.05; }
  to { opacity: 0.2; }
}
@keyframes title-shine { from { background-position: 0% center; } to { background-position: 100% center; } }
@keyframes line-appear { from { opacity: 0; transform: translateY(0.35rem); } to { opacity: 1; transform: translateY(0); } }
@keyframes cursor-blink { 0%, 50% { opacity: 1; } 51%, 100% { opacity: 0; } }
@keyframes float-y { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-0.4rem); } }

@media (max-width: 640px) {
  .hero-interaction-layer { opacity: 0.72; }
  .hero-sweep,
  .hero-trace { display: none; }
  .terminal-float--top { right: -0.35rem; }
  .terminal-float--bottom { left: -0.35rem; }
  .terminal-footer { font-size: 0.48rem; }
  .hero-scroll-cue { bottom: -3.3rem; }
  .hero-scroll-meter { bottom: -1.8rem; }
}

@media (prefers-reduced-motion: reduce) {
  .hero-sweep,
  .hero-trace,
  .home-aurora::before,
  .home-aurora::after,
  .home-title-shine,
  .terminal-window,
  .terminal-float,
  .code-line,
  .cursor,
  .home-reveal,
  .hero-section,
  .home-provider {
    animation: none;
    transition: none;
  }
  .code-line, .home-reveal { opacity: 1; transform: none; }
  .hero-section.is-visible,
  .home-provider:hover { transform: none; }
}
</style>
