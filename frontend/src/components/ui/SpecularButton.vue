<template>
  <component
    :is="componentType"
    ref="buttonRef"
    :to="to"
    :type="to ? undefined : type"
    :disabled="to ? undefined : disabled"
    :aria-disabled="disabled || undefined"
    class="specular-button"
    :class="'specular-button--' + size"
    :style="buttonStyle"
    v-bind="$attrs"
  >
    <span ref="effectRef" class="specular-button__effect" aria-hidden="true"></span>
    <span class="specular-button__label"><slot /></span>
  </component>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type ComponentPublicInstance, type CSSProperties } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { Color, Mesh, Program, Renderer, Triangle } from 'ogl'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  to?: RouteLocationRaw
  type?: 'button' | 'submit' | 'reset'
  size?: 'sm' | 'md' | 'lg'
  radius?: number
  tint?: string
  tintOpacity?: number
  blur?: number
  textColor?: string
  lineColor?: string
  baseColor?: string
  intensity?: number
  shineSize?: number
  shineFade?: number
  thickness?: number
  speed?: number
  followMouse?: boolean
  proximity?: number
  autoAnimate?: boolean
  disabled?: boolean
}>(), {
  type: 'button',
  size: 'md',
  radius: 14,
  tint: '#030712',
  tintOpacity: 0.96,
  blur: 10,
  textColor: '#f8fafc',
  lineColor: '#ffffff',
  baseColor: '#475569',
  intensity: 1.15,
  shineSize: 10,
  shineFade: 40,
  thickness: 1,
  speed: 0.35,
  followMouse: true,
  proximity: 250,
  autoAnimate: true,
  disabled: false,
})

const PAD = 20
const buttonRef = ref<HTMLElement | ComponentPublicInstance | null>(null)
const effectRef = ref<HTMLElement | null>(null)
const componentType = computed(() => props.to ? RouterLink : 'button')
const buttonStyle = computed<CSSProperties>(() => ({
  '--sb-radius': String(props.radius) + 'px',
  '--sb-tint': props.tint,
  '--sb-tint-opacity': props.tintOpacity,
  '--sb-blur': String(props.blur) + 'px',
  '--sb-text-color': props.textColor,
}))

let cleanupEffect: (() => void) | null = null

const VERTEX_SHADER = `#version 300 es
in vec2 position;
void main() {
  gl_Position = vec4(position, 0.0, 1.0);
}
`

const FRAGMENT_SHADER = `#version 300 es
precision highp float;

uniform vec2 uCenter;
uniform vec2 uHalfSize;
uniform float uRadius;
uniform float uAngle;
uniform float uPx;
uniform vec3 uLineColor;
uniform vec3 uBaseColor;
uniform float uIntensity;
uniform float uShineSize;
uniform float uShineFade;
uniform float uThickness;
uniform float uBaseWidth;

out vec4 fragColor;

float sdRoundedRect(vec2 p, vec2 b, float r) {
  vec2 q = abs(p) - b + r;
  return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - r;
}

float gaussianLine(float d, float sigma) {
  float x = d / (sigma + 1e-6);
  float k = mix(1.0, 1.6, smoothstep(0.0, 1.5, x));
  return exp(-k * x * x);
}

void main() {
  vec2 p = gl_FragCoord.xy - uCenter;
  float d = sdRoundedRect(p, uHalfSize, uRadius);
  vec2 light = vec2(cos(uAngle), sin(uAngle));
  float base = (1.0 - smoothstep(0.0, uBaseWidth, abs(d))) * 0.45;
  vec2 normal = normalize(p / (uHalfSize * uHalfSize) + 1e-6);
  float phi = acos(clamp(abs(dot(normal, light)), 0.0, 1.0));
  float rim = 1.0 - smoothstep(uShineSize - uShineFade, uShineSize + uShineFade + 1e-4, phi);
  float line = gaussianLine(d, uThickness);
  float edgeClamp = 1.0 - smoothstep(0.5 * uPx, 3.0 * uPx, abs(d));
  float highlight = line * rim * edgeClamp * uIntensity;
  vec3 color = uBaseColor * base + uLineColor * highlight;
  float alpha = clamp(base + highlight, 0.0, 1.0);
  fragColor = vec4(color, alpha);
}
`

// RouterLink 的模板引用指向组件实例，这里统一取得最终的按钮或链接 DOM。
function getButtonElement() {
  const target = buttonRef.value
  if (target instanceof HTMLElement) return target
  return target?.$el instanceof HTMLElement ? target.$el : null
}

// 初始化 React Bits 同源的镜面边缘 Shader，并返回完整的资源清理函数。
function setupSpecularEffect() {
  const button = getButtonElement()
  const effect = effectRef.value
  if (!button || !effect || typeof WebGL2RenderingContext === 'undefined') return

  let renderer: Renderer
  try {
    renderer = new Renderer({
      alpha: true,
      premultipliedAlpha: true,
      antialias: true,
      dpr: Math.min(window.devicePixelRatio || 1, 2),
      webgl: 2,
    })
  } catch {
    // WebGL2 初始化失败时保留 CSS 玻璃按钮，不影响跳转功能。
    return
  }

  const dpr = renderer.dpr
  const gl = renderer.gl
  gl.clearColor(0, 0, 0, 0)
  gl.enable(gl.BLEND)
  gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)

  const geometry = new Triangle(gl)
  if (geometry.attributes.uv) delete geometry.attributes.uv
  const program = new Program(gl, {
    vertex: VERTEX_SHADER,
    fragment: FRAGMENT_SHADER,
    transparent: true,
    depthTest: false,
    depthWrite: false,
    cullFace: false,
    uniforms: {
      uCenter: { value: [0, 0] },
      uHalfSize: { value: [1, 1] },
      uRadius: { value: 0 },
      uAngle: { value: 2.4 },
      uPx: { value: dpr },
      uLineColor: { value: [1, 1, 1] },
      uBaseColor: { value: [0.28, 0.33, 0.41] },
      uIntensity: { value: 1 },
      uShineSize: { value: 0.17 },
      uShineFade: { value: 0.7 },
      uThickness: { value: 1 },
      uBaseWidth: { value: dpr },
    },
  })
  const mesh = new Mesh(gl, { geometry, program })
  effect.appendChild(gl.canvas)

  const size = { width: 1, height: 1 }
  const resize = () => {
    const rect = button.getBoundingClientRect()
    size.width = rect.width
    size.height = rect.height
    renderer.setSize(rect.width + PAD * 2, rect.height + PAD * 2)
    program.uniforms.uCenter.value = [(PAD + rect.width / 2) * dpr, (PAD + rect.height / 2) * dpr]
    program.uniforms.uHalfSize.value = [(rect.width / 2) * dpr, (rect.height / 2) * dpr]
  }

  const resizeObserver = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(resize)
  resizeObserver?.observe(button)
  if (!resizeObserver) window.addEventListener('resize', resize)
  resize()

  let pointerAngle: number | null = null
  let proximity = 0

  // event 提供全局指针位置，用于计算按钮边缘的入射光方向与接近强度。
  const handlePointerMove = (event: PointerEvent) => {
    const rect = button.getBoundingClientRect()
    const centerX = rect.left + rect.width / 2
    const centerY = rect.top + rect.height / 2
    const distanceX = Math.max(rect.left - event.clientX, 0, event.clientX - rect.right)
    const distanceY = Math.max(rect.top - event.clientY, 0, event.clientY - rect.bottom)
    const distance = Math.hypot(distanceX, distanceY)

    if (distance === 0) {
      const normalizedX = (event.clientX - centerX) / (rect.width / 2)
      const normalizedY = (centerY - event.clientY) / (rect.height / 2)
      pointerAngle = Math.atan2(2 / rect.height, -2 / rect.width) + normalizedX * 0.3 + normalizedY * 0.15
    } else {
      pointerAngle = Math.atan2(centerY - event.clientY, event.clientX - centerX)
    }

    const distanceRatio = Math.max(0, 1 - distance / Math.max(props.proximity, 1))
    proximity = distanceRatio * distanceRatio * (3 - 2 * distanceRatio)
  }
  window.addEventListener('pointermove', handlePointerMove, { passive: true })

  let angle = 2.4
  let idleAngle = 2.4
  let brightness = 0
  let lastFrame = performance.now()
  let animationFrame = 0
  const lineColor = new Color()
  const baseColor = new Color()

  // now 为当前帧时间，用阻尼插值让高光平滑跟随鼠标或自动巡航。
  const render = (now: number) => {
    animationFrame = window.requestAnimationFrame(render)
    const delta = Math.min((now - lastFrame) / 1000, 0.05)
    lastFrame = now
    idleAngle += props.speed * delta

    const targetAngle = props.followMouse && pointerAngle !== null && (!props.autoAnimate || proximity > 0)
      ? pointerAngle
      : idleAngle
    const difference = ((targetAngle - angle + Math.PI * 3) % (Math.PI * 2)) - Math.PI
    angle += difference * (1 - Math.exp(-delta * 7))
    const targetBrightness = props.autoAnimate ? 1 : proximity
    brightness += (targetBrightness - brightness) * (1 - Math.exp(-delta * 8))

    lineColor.set(props.lineColor)
    baseColor.set(props.baseColor)
    program.uniforms.uAngle.value = angle
    program.uniforms.uRadius.value = Math.min(props.radius, Math.min(size.width, size.height) / 2) * dpr
    program.uniforms.uLineColor.value = [lineColor.r, lineColor.g, lineColor.b]
    program.uniforms.uBaseColor.value = [baseColor.r, baseColor.g, baseColor.b]
    program.uniforms.uIntensity.value = props.intensity * brightness
    program.uniforms.uShineSize.value = (props.shineSize * Math.PI) / 180
    program.uniforms.uShineFade.value = (props.shineFade * Math.PI) / 180
    program.uniforms.uThickness.value = props.thickness * dpr
    renderer.render({ scene: mesh })
  }
  animationFrame = window.requestAnimationFrame(render)

  cleanupEffect = () => {
    window.cancelAnimationFrame(animationFrame)
    resizeObserver?.disconnect()
    if (!resizeObserver) window.removeEventListener('resize', resize)
    window.removeEventListener('pointermove', handlePointerMove)
    if (gl.canvas.parentNode === effect) effect.removeChild(gl.canvas)
    gl.getExtension('WEBGL_lose_context')?.loseContext()
  }
}

onMounted(setupSpecularEffect)
onBeforeUnmount(() => cleanupEffect?.())
</script>

<style scoped>
.specular-button {
  --sb-radius: 14px;
  --sb-tint: #030712;
  --sb-tint-opacity: 0.96;
  --sb-blur: 10px;
  --sb-text-color: #f8fafc;

  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0;
  border: 1px solid rgba(148, 163, 184, 0.38);
  border-radius: var(--sb-radius);
  outline: none;
  color: var(--sb-text-color);
  background:
    radial-gradient(circle at 50% 0%, rgba(255, 255, 255, 0.12), transparent 58%),
    color-mix(in srgb, var(--sb-tint) calc(var(--sb-tint-opacity) * 100%), transparent);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.12),
    inset 0 -1px 0 rgba(255, 255, 255, 0.03),
    0 8px 24px rgba(2, 6, 23, 0.2);
  backdrop-filter: blur(var(--sb-blur));
  -webkit-backdrop-filter: blur(var(--sb-blur));
  font-family: inherit;
  font-weight: 600;
  line-height: 1;
  letter-spacing: 0;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  isolation: isolate;
  transition:
    transform 180ms ease,
    border-color 180ms ease,
    box-shadow 180ms ease;
}

.specular-button:hover {
  border-color: rgba(226, 232, 240, 0.62);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.16),
    0 12px 32px rgba(2, 6, 23, 0.3),
    0 0 24px rgba(45, 212, 191, 0.08);
  transform: translateY(-1px);
}

.specular-button:active {
  transform: translateY(0) scale(0.97);
}

.specular-button:focus-visible {
  outline: 2px solid rgba(45, 212, 191, 0.75);
  outline-offset: 3px;
}

.specular-button[aria-disabled='true'] {
  opacity: 0.55;
  pointer-events: none;
}

.specular-button--sm {
  min-height: 38px;
  padding: 0 16px;
  font-size: 0.8125rem;
}

.specular-button--md {
  min-height: 44px;
  padding: 0 22px;
  font-size: 0.9375rem;
}

.specular-button--lg {
  min-height: 50px;
  padding: 0 28px;
  font-size: 1rem;
}

.specular-button__effect {
  position: absolute;
  z-index: 1;
  inset: -20px;
  pointer-events: none;
}

.specular-button__effect canvas {
  display: block;
  width: 100%;
  height: 100%;
}

.specular-button__label {
  position: relative;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
}

@media (prefers-reduced-motion: reduce) {
  .specular-button {
    transition: none;
  }

  .specular-button:hover,
  .specular-button:active {
    transform: none;
  }
}
</style>
