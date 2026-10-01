<template>
  <section
    class="landing-cta"
    @pointermove="updatePointerGlow"
    @pointerleave="resetPointerGlow"
  >
    <div class="landing-cta__grid" aria-hidden="true"></div>

    <div class="landing-cta__copy">
      <p class="landing-cta__kicker">
        <Icon name="sparkles" size="sm" />
        <span>{{ t('home.cta.title') }}</span>
      </p>
      <h2>{{ siteName }}</h2>
      <p class="landing-cta__subtitle">{{ siteSubtitle }}</p>

      <div class="landing-cta__capabilities">
        <span v-for="item in capabilities" :key="item.key">
          <Icon :name="item.icon" size="sm" />
          {{ t(item.key) }}
        </span>
      </div>

      <SpecularButton
        :to="isAuthenticated ? dashboardPath : '/login'"
        size="lg"
        :radius="14"
        :intensity="1.35"
        :shine-size="12"
        :shine-fade="36"
        tint="#0f766e"
        :tint-opacity="0.94"
        line-color="#ccfbf1"
        base-color="#2dd4bf"
        text-color="#f0fdfa"
        class="landing-cta__button group"
      >
        <span>{{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}</span>
        <Icon name="arrowRight" size="md" class="transition-transform duration-300 group-hover:translate-x-1" />
      </SpecularButton>
    </div>

    <div
      ref="routeConsole"
      class="route-console"
      role="img"
      :aria-label="t('home.cta.routeTitle')"
    >
      <div class="route-console__header" aria-hidden="true">
        <span>{{ t('home.cta.routeTitle') }}</span>
        <span class="route-console__status">
          <i></i>
          {{ t('home.cta.routeReady') }}
        </span>
      </div>

      <div class="route-track" aria-hidden="true">
        <div class="route-node route-node--source">
          <Icon name="terminal" size="md" />
          <span>
            <strong>{{ t('home.cta.endpoint') }}</strong>
            <small>/v1/messages</small>
          </span>
        </div>

        <div class="route-connector"><i></i></div>

        <div class="route-node route-node--router">
          <Icon name="shield" size="md" />
          <span>
            <strong>{{ t('home.cta.router') }}</strong>
            <small>Auto</small>
          </span>
        </div>

        <div class="route-connector route-connector--delayed"><i></i></div>

        <div class="route-targets">
          <span><i class="route-target route-target--claude"></i>Claude</span>
          <span><i class="route-target route-target--gpt"></i>GPT</span>
          <span><i class="route-target route-target--gemini"></i>Gemini</span>
        </div>
      </div>

      <div class="route-console__footer" aria-hidden="true">
        <span v-for="item in capabilities" :key="item.key">
          <Icon name="check" size="xs" />
          {{ t(item.key) }}
        </span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import SpecularButton from '@/components/ui/SpecularButton.vue'

defineProps<{
  siteName: string
  siteSubtitle: string
  isAuthenticated: boolean
  dashboardPath: string
}>()

const { t } = useI18n()
const routeConsole = ref<HTMLElement | null>(null)

const capabilities = [
  { key: 'home.features.unifiedGateway', icon: 'terminal' },
  { key: 'home.features.multiAccount', icon: 'shield' },
  { key: 'home.features.balanceQuota', icon: 'chart' },
] as const

function updatePointerGlow(event: PointerEvent) {
  if (event.pointerType === 'touch' || !routeConsole.value) return
  const bounds = routeConsole.value.getBoundingClientRect()
  routeConsole.value.style.setProperty('--route-x', `${event.clientX - bounds.left}px`)
  routeConsole.value.style.setProperty('--route-y', `${event.clientY - bounds.top}px`)
}

function resetPointerGlow() {
  routeConsole.value?.style.setProperty('--route-x', '50%')
  routeConsole.value?.style.setProperty('--route-y', '45%')
}
</script>

<style scoped>
.landing-cta {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(31rem, 1.1fr);
  align-items: center;
  gap: clamp(2.5rem, 7vw, 6rem);
  overflow: hidden;
  border-block: 1px solid rgba(20, 184, 166, 0.26);
  padding: clamp(2.5rem, 6vw, 4.75rem) clamp(0.25rem, 3vw, 2.5rem);
  background: rgba(255, 255, 255, 0.34);
}

.landing-cta::before {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 48%;
  width: 1px;
  background: linear-gradient(transparent, rgba(20, 184, 166, 0.24), transparent);
  content: '';
}

.landing-cta__grid {
  position: absolute;
  inset: 0 0 0 48%;
  opacity: 0.35;
  background-image:
    linear-gradient(rgba(15, 118, 110, 0.1) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 118, 110, 0.1) 1px, transparent 1px);
  background-size: 32px 32px;
  mask-image: linear-gradient(90deg, transparent, black 28%, black);
}

.landing-cta__copy,
.route-console {
  position: relative;
  z-index: 1;
}

.landing-cta__copy h2 {
  margin-top: 0.8rem;
  color: #0f172a;
  font-size: clamp(2rem, 4vw, 3.25rem);
  font-weight: 700;
  line-height: 1.06;
  letter-spacing: 0;
}

.landing-cta__kicker {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  color: #0f766e;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
}

.landing-cta__subtitle {
  max-width: 32rem;
  margin-top: 1rem;
  color: #475569;
  font-size: 1rem;
  line-height: 1.8;
}

.landing-cta__capabilities {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.25rem;
  margin-top: 1.5rem;
}

.landing-cta__capabilities span {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: #475569;
  font-size: 0.8rem;
  font-weight: 600;
}

.landing-cta__capabilities svg {
  color: #0d9488;
}

.landing-cta__button {
  margin-top: 1.75rem;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.16),
    0 14px 34px rgba(15, 118, 110, 0.2),
    0 0 0 1px rgba(20, 184, 166, 0.12);
}

.route-console {
  --route-x: 50%;
  --route-y: 45%;
  overflow: hidden;
  border: 1px solid rgba(94, 234, 212, 0.2);
  border-radius: 14px;
  background: #07111c;
  box-shadow:
    0 30px 70px -36px rgba(2, 6, 23, 0.68),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.route-console::before {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(18rem circle at var(--route-x) var(--route-y), rgba(20, 184, 166, 0.15), transparent 70%);
  content: '';
}

.route-console__header,
.route-console__footer {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.9rem 1rem;
  color: #94a3b8;
  font: 600 0.63rem/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.route-console__header {
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}

.route-console__status {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: #6ee7b7;
}

.route-console__status i {
  width: 0.4rem;
  height: 0.4rem;
  border-radius: 999px;
  background: #34d399;
  box-shadow: 0 0 12px rgba(52, 211, 153, 0.85);
  animation: route-status 1.8s ease-in-out infinite;
}

.route-track {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: minmax(7rem, 1fr) 3.2rem minmax(7rem, 0.9fr) 3.2rem minmax(7rem, 1fr);
  align-items: center;
  padding: clamp(2.4rem, 6vw, 4.25rem) 1.1rem;
}

.route-node {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.7rem;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.74);
  padding: 0.85rem;
  color: #cbd5e1;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.035);
}

.route-node > svg {
  flex: 0 0 auto;
  color: #5eead4;
}

.route-node span {
  min-width: 0;
}

.route-node strong,
.route-node small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.route-node strong {
  color: #f1f5f9;
  font-size: 0.75rem;
  font-weight: 650;
}

.route-node small {
  margin-top: 0.25rem;
  color: #64748b;
  font: 500 0.61rem/1.2 ui-monospace, SFMono-Regular, Menlo, monospace;
}

.route-node--router {
  border-color: rgba(45, 212, 191, 0.28);
  background: rgba(13, 148, 136, 0.09);
}

.route-connector {
  position: relative;
  height: 1px;
  overflow: visible;
  background: rgba(94, 234, 212, 0.2);
}

.route-connector::before,
.route-connector::after {
  position: absolute;
  top: -2px;
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: #2dd4bf;
  content: '';
}

.route-connector::before { left: -1px; }
.route-connector::after { right: -1px; }

.route-connector i {
  position: absolute;
  top: -3px;
  left: 0;
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: #99f6e4;
  box-shadow: 0 0 14px 3px rgba(45, 212, 191, 0.45);
  animation: request-flow 2.4s ease-in-out infinite;
}

.route-connector--delayed i {
  animation-delay: -1.2s;
}

.route-targets {
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.64);
}

.route-targets span {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.55rem 0.7rem;
  color: #cbd5e1;
  font-size: 0.68rem;
  font-weight: 600;
}

.route-targets span + span {
  border-top: 1px solid rgba(148, 163, 184, 0.1);
}

.route-target {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
}

.route-target--claude { background: #fb923c; }
.route-target--gpt { background: #34d399; }
.route-target--gemini { background: #60a5fa; }

.route-console__footer {
  justify-content: flex-start;
  flex-wrap: wrap;
  border-top: 1px solid rgba(148, 163, 184, 0.12);
  color: #64748b;
}

.route-console__footer span {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.route-console__footer svg {
  color: #2dd4bf;
}

:global(.dark .landing-cta) {
  border-color: rgba(45, 212, 191, 0.16);
  background: rgba(3, 6, 15, 0.42);
}

:global(.dark .landing-cta__copy h2) { color: #f8fafc; }
:global(.dark .landing-cta__kicker) { color: #5eead4; }
:global(.dark .landing-cta__subtitle),
:global(.dark .landing-cta__capabilities span) { color: #94a3b8; }

@keyframes request-flow {
  0% { left: 0; opacity: 0; }
  18% { opacity: 1; }
  82% { opacity: 1; }
  100% { left: calc(100% - 7px); opacity: 0; }
}

@keyframes route-status {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}

@media (max-width: 1023px) {
  .landing-cta {
    grid-template-columns: 1fr;
    gap: 2.5rem;
    padding-inline: clamp(0.25rem, 4vw, 2rem);
  }

  .landing-cta::before { display: none; }
  .landing-cta__grid { inset: 48% 0 0; }
  .landing-cta__copy { max-width: 40rem; }
  .route-console { width: 100%; }
}

@media (max-width: 639px) {
  .landing-cta {
    padding-block: 2.75rem;
  }

  .landing-cta__capabilities {
    display: grid;
    grid-template-columns: 1fr;
  }

  .route-track {
    grid-template-columns: minmax(5.5rem, 1fr) 1.5rem minmax(5.5rem, 0.9fr) 1.5rem minmax(5.5rem, 1fr);
    padding: 2.25rem 0.65rem;
  }

  .route-node {
    justify-content: center;
    padding: 0.72rem 0.45rem;
  }

  .route-node > svg { display: none; }
  .route-node strong { font-size: 0.65rem; }
  .route-node small { font-size: 0.5rem; }
  .route-targets span { gap: 0.35rem; padding: 0.48rem; font-size: 0.58rem; }
  .route-console__footer { display: none; }
  .route-console__header { align-items: flex-start; flex-direction: column; }
}

@media (prefers-reduced-motion: reduce) {
  .route-console__status i,
  .route-connector i {
    animation: none;
  }
}
</style>
