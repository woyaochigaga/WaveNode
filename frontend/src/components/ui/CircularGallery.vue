<template>
  <div
    ref="containerRef"
    class="circular-gallery"
    :class="{ 'is-dark': isDark, 'is-dragging': isDragging, 'is-fallback': useFallback }"
    tabindex="0"
    role="region"
    :aria-label="ariaLabel"
  >
    <div v-if="useFallback" class="circular-gallery__fallback">
      <article v-for="item in items" :key="item.code" class="circular-gallery__fallback-card">
        <span class="circular-gallery__fallback-code" :style="{ color: item.accent }">{{ item.code }}</span>
        <h3>{{ item.title }}</h3>
        <p>{{ item.description }}</p>
      </article>
    </div>

    <ul class="sr-only">
      <li v-for="item in items" :key="'accessible-' + item.code">
        {{ item.title }}: {{ item.description }}
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  Camera,
  Mesh,
  Plane,
  Program,
  Renderer,
  Texture,
  Transform,
  type OGLRenderingContext,
} from 'ogl'

interface CircularGalleryItem {
  code: string
  title: string
  description: string
  accent: string
  secondary: string
}

const props = withDefaults(defineProps<{
  items: CircularGalleryItem[]
  isDark?: boolean
  ariaLabel?: string
  bend?: number
  scrollSpeed?: number
  scrollEase?: number
}>(), {
  isDark: false,
  ariaLabel: 'Feature gallery',
  bend: 2.8,
  scrollSpeed: 1.8,
  scrollEase: 0.075,
})

const containerRef = ref<HTMLElement | null>(null)
const useFallback = ref(false)
const isDragging = ref(false)
let gallery: CircularGalleryApp | null = null

interface GalleryViewport {
  width: number
  height: number
}

interface GalleryScreen {
  width: number
  height: number
}

interface GalleryScroll {
  current: number
  target: number
  last: number
}

function lerp(start: number, end: number, ease: number) {
  return start + (end - start) * ease
}

function hexToRgba(hex: string, alpha: number) {
  const normalized = hex.replace('#', '')
  const value = Number.parseInt(normalized.length === 3
    ? normalized.split('').map((char) => char + char).join('')
    : normalized, 16)
  const red = (value >> 16) & 255
  const green = (value >> 8) & 255
  const blue = value & 255
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`
}

// 将长文案限制在卡片宽度内，避免中英文切换后文字被裁切。
function drawWrappedText(
  context: CanvasRenderingContext2D,
  text: string,
  x: number,
  y: number,
  maxWidth: number,
  lineHeight: number,
  maxLines: number,
) {
  const characters = Array.from(text)
  const lines: string[] = []
  let line = ''

  characters.forEach((character) => {
    const candidate = line + character
    if (context.measureText(candidate).width > maxWidth && line) {
      lines.push(line.trim())
      line = character
    } else {
      line = candidate
    }
  })
  if (line) lines.push(line.trim())

  lines.slice(0, maxLines).forEach((content, index) => {
    const isClipped = index === maxLines - 1 && lines.length > maxLines
    context.fillText(isClipped ? content.replace(/[,.，。\s]+$/, '') + '...' : content, x, y + index * lineHeight)
  })
}

// 功能卡纹理由 Canvas 本地生成，避免第三方图片依赖，并可随主题即时重绘。
function createCardTexture(item: CircularGalleryItem, isDark: boolean) {
  const canvas = document.createElement('canvas')
  canvas.width = 960
  canvas.height = 640
  const context = canvas.getContext('2d')
  if (!context) throw new Error('CircularGallery: Canvas 2D is unavailable')

  const foreground = isDark ? '#f8fafc' : '#0f172a'
  const muted = isDark ? '#94a3b8' : '#475569'
  const surface = isDark ? '#070b14' : '#f8fbfc'
  const line = isDark ? 'rgba(148, 163, 184, 0.12)' : 'rgba(15, 23, 42, 0.10)'

  const background = context.createLinearGradient(0, 0, canvas.width, canvas.height)
  background.addColorStop(0, surface)
  background.addColorStop(0.58, isDark ? '#0b1220' : '#ffffff')
  background.addColorStop(1, isDark ? '#0b1322' : '#eef6f7')
  context.fillStyle = background
  context.fillRect(0, 0, canvas.width, canvas.height)

  context.strokeStyle = line
  context.lineWidth = 1
  for (let x = 0; x <= canvas.width; x += 64) {
    context.beginPath()
    context.moveTo(x, 0)
    context.lineTo(x, canvas.height)
    context.stroke()
  }
  for (let y = 0; y <= canvas.height; y += 64) {
    context.beginPath()
    context.moveTo(0, y)
    context.lineTo(canvas.width, y)
    context.stroke()
  }

  const glow = context.createRadialGradient(760, 112, 12, 760, 112, 360)
  glow.addColorStop(0, hexToRgba(item.accent, isDark ? 0.34 : 0.24))
  glow.addColorStop(1, hexToRgba(item.secondary, 0))
  context.fillStyle = glow
  context.fillRect(360, 0, 600, 480)

  context.strokeStyle = hexToRgba(item.accent, isDark ? 0.52 : 0.66)
  context.lineWidth = 3
  context.beginPath()
  context.arc(770, 168, 112, 0.2, Math.PI * 1.72)
  context.stroke()
  context.beginPath()
  context.arc(770, 168, 72, Math.PI * 0.92, Math.PI * 2.35)
  context.stroke()

  const nodes = [[694, 104], [832, 109], [845, 215], [718, 238]]
  nodes.forEach(([x, y], index) => {
    context.beginPath()
    context.fillStyle = index % 2 ? item.secondary : item.accent
    context.arc(x, y, index === 0 ? 9 : 6, 0, Math.PI * 2)
    context.fill()
  })

  context.fillStyle = hexToRgba(item.accent, isDark ? 0.16 : 0.12)
  context.fillRect(64, 62, 134, 42)
  context.fillStyle = item.accent
  context.font = '700 20px ui-monospace, SFMono-Regular, Menlo, monospace'
  context.textBaseline = 'middle'
  context.fillText(item.code, 84, 84)

  context.fillStyle = foreground
  context.font = '700 52px -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif'
  context.textBaseline = 'alphabetic'
  context.fillText(item.title, 64, 396, 790)

  context.fillStyle = muted
  context.font = '400 27px -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif'
  drawWrappedText(context, item.description, 64, 452, 790, 44, 2)

  const footerGradient = context.createLinearGradient(64, 0, 300, 0)
  footerGradient.addColorStop(0, item.accent)
  footerGradient.addColorStop(1, item.secondary)
  context.fillStyle = footerGradient
  context.fillRect(64, 568, 196, 4)
  context.fillStyle = isDark ? '#64748b' : '#64748b'
  context.font = '600 16px ui-monospace, SFMono-Regular, Menlo, monospace'
  context.fillText('API / ROUTE / READY', 64, 602)

  return canvas
}

class GalleryMedia {
  private readonly gl: OGLRenderingContext
  private readonly mesh: Mesh
  private readonly program: Program
  private readonly texture: Texture
  private readonly index: number
  private readonly total: number
  private readonly bend: number
  private screen: GalleryScreen
  private viewport: GalleryViewport
  private extra = 0
  private x = 0
  width = 1
  private widthTotal = 1

  constructor(options: {
    gl: OGLRenderingContext
    geometry: Plane
    scene: Transform
    item: CircularGalleryItem
    index: number
    total: number
    bend: number
    screen: GalleryScreen
    viewport: GalleryViewport
    isDark: boolean
  }) {
    this.gl = options.gl
    this.index = options.index
    this.total = options.total
    this.bend = options.bend
    this.screen = options.screen
    this.viewport = options.viewport
    this.texture = new Texture(this.gl, { generateMipmaps: true })
    this.texture.image = createCardTexture(options.item, options.isDark)

    this.program = new Program(this.gl, {
      depthTest: false,
      depthWrite: false,
      transparent: true,
      vertex: `
        precision highp float;
        attribute vec3 position;
        attribute vec2 uv;
        uniform mat4 modelViewMatrix;
        uniform mat4 projectionMatrix;
        uniform float uTime;
        uniform float uSpeed;
        varying vec2 vUv;

        void main() {
          vUv = uv;
          vec3 p = position;
          p.z += sin(p.x * 3.2 + uTime) * min(abs(uSpeed) * 0.22, 0.28);
          gl_Position = projectionMatrix * modelViewMatrix * vec4(p, 1.0);
        }
      `,
      fragment: `
        precision highp float;
        uniform sampler2D tMap;
        varying vec2 vUv;

        float roundedBoxSDF(vec2 p, vec2 b, float radius) {
          vec2 d = abs(p) - b + radius;
          return min(max(d.x, d.y), 0.0) + length(max(d, 0.0)) - radius;
        }

        void main() {
          float distance = roundedBoxSDF(vUv - 0.5, vec2(0.48), 0.035);
          float alpha = 1.0 - smoothstep(-0.003, 0.003, distance);
          vec4 color = texture2D(tMap, vUv);
          gl_FragColor = vec4(color.rgb, color.a * alpha);
        }
      `,
      uniforms: {
        tMap: { value: this.texture },
        uTime: { value: Math.random() * 100 },
        uSpeed: { value: 0 },
      },
    })
    this.mesh = new Mesh(this.gl, { geometry: options.geometry, program: this.program })
    this.mesh.setParent(options.scene)
    this.resize()
  }

  resize(screen = this.screen, viewport = this.viewport) {
    this.screen = screen
    this.viewport = viewport
    const planeHeight = viewport.height * (screen.width < 640 ? 0.56 : 0.48)
    this.mesh.scale.y = planeHeight
    this.mesh.scale.x = planeHeight * 1.5
    const padding = screen.width < 640 ? 0.65 : 1.05
    this.width = this.mesh.scale.x + padding
    this.widthTotal = this.width * this.total
    this.x = this.width * this.index
  }

  update(scroll: GalleryScroll, direction: 'left' | 'right', reducedMotion: boolean) {
    this.mesh.position.x = this.x - scroll.current - this.extra
    const x = this.mesh.position.x
    const halfViewport = this.viewport.width / 2
    const bend = Math.abs(this.bend)

    if (bend > 0) {
      const radius = (halfViewport * halfViewport + bend * bend) / (2 * bend)
      const effectiveX = Math.min(Math.abs(x), halfViewport)
      const arc = radius - Math.sqrt(Math.max(radius * radius - effectiveX * effectiveX, 0))
      this.mesh.position.y = this.bend > 0 ? -arc : arc
      this.mesh.rotation.z = (this.bend > 0 ? -1 : 1) * Math.sign(x) * Math.asin(effectiveX / radius)
    }

    const speed = scroll.current - scroll.last
    this.program.uniforms.uTime.value += reducedMotion ? 0 : 0.035
    this.program.uniforms.uSpeed.value = reducedMotion ? 0 : speed

    const halfPlane = this.mesh.scale.x / 2
    if (direction === 'right' && this.mesh.position.x + halfPlane < -halfViewport) {
      this.extra -= this.widthTotal
    } else if (direction === 'left' && this.mesh.position.x - halfPlane > halfViewport) {
      this.extra += this.widthTotal
    }
  }
}

class CircularGalleryApp {
  private readonly container: HTMLElement
  private readonly renderer: Renderer
  private readonly gl: OGLRenderingContext
  private readonly camera: Camera
  private readonly scene = new Transform()
  private readonly geometry: Plane
  private readonly medias: GalleryMedia[]
  private readonly resizeObserver: ResizeObserver | null
  private readonly scroll: GalleryScroll = { current: 0, target: 0, last: 0 }
  private readonly scrollSpeed: number
  private readonly scrollEase: number
  private readonly reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  private screen: GalleryScreen = { width: 1, height: 1 }
  private viewport: GalleryViewport = { width: 1, height: 1 }
  private raf = 0
  private snapTimer: number | null = null
  private pointerId: number | null = null
  private pointerStart = 0
  private scrollStart = 0

  constructor(container: HTMLElement, options: {
    items: CircularGalleryItem[]
    isDark: boolean
    bend: number
    scrollSpeed: number
    scrollEase: number
  }) {
    this.container = container
    this.scrollSpeed = options.scrollSpeed
    this.scrollEase = options.scrollEase
    this.renderer = new Renderer({
      alpha: true,
      antialias: true,
      dpr: Math.min(window.devicePixelRatio || 1, 2),
    })
    this.gl = this.renderer.gl
    this.gl.clearColor(0, 0, 0, 0)
    this.gl.canvas.setAttribute('aria-hidden', 'true')
    this.container.prepend(this.gl.canvas)

    this.camera = new Camera(this.gl)
    this.camera.fov = 45
    this.camera.position.z = 20
    this.measure()
    this.geometry = new Plane(this.gl, { widthSegments: 48, heightSegments: 24 })

    const galleryItems = options.items.concat(options.items)
    this.medias = galleryItems.map((item, index) => new GalleryMedia({
      gl: this.gl,
      geometry: this.geometry,
      scene: this.scene,
      item,
      index,
      total: galleryItems.length,
      bend: options.bend,
      screen: this.screen,
      viewport: this.viewport,
      isDark: options.isDark,
    }))

    const initialWidth = this.medias[0]?.width || 1
    this.scroll.current = initialWidth * Math.floor(options.items.length / 2)
    this.scroll.target = this.scroll.current
    this.scroll.last = this.scroll.current

    this.container.addEventListener('wheel', this.handleWheel, { passive: true })
    this.container.addEventListener('pointerdown', this.handlePointerDown)
    this.container.addEventListener('keydown', this.handleKeyDown)
    window.addEventListener('pointermove', this.handlePointerMove, { passive: false })
    window.addEventListener('pointerup', this.handlePointerUp)
    window.addEventListener('pointercancel', this.handlePointerUp)
    this.resizeObserver = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(this.handleResize)
    this.resizeObserver?.observe(this.container)
    if (!this.resizeObserver) window.addEventListener('resize', this.handleResize)
    this.update()
  }

  private measure = () => {
    this.screen = {
      width: Math.max(this.container.clientWidth, 1),
      height: Math.max(this.container.clientHeight, 1),
    }
    this.renderer.setSize(this.screen.width, this.screen.height)
    this.camera.perspective({ aspect: this.screen.width / this.screen.height })
    const fieldOfView = (this.camera.fov * Math.PI) / 180
    const height = 2 * Math.tan(fieldOfView / 2) * this.camera.position.z
    this.viewport = { width: height * this.camera.aspect, height }
  }

  private handleResize = () => {
    this.measure()
    this.medias.forEach((media) => media.resize(this.screen, this.viewport))
  }

  private scheduleSnap() {
    if (this.snapTimer !== null) window.clearTimeout(this.snapTimer)
    this.snapTimer = window.setTimeout(this.snap, 140)
  }

  private snap = () => {
    const width = this.medias[0]?.width
    if (!width) return
    this.scroll.target = Math.round(this.scroll.target / width) * width
  }

  private handleWheel = (event: WheelEvent) => {
    const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY
    this.scroll.target += Math.sign(delta) * this.scrollSpeed * 0.36
    this.scheduleSnap()
  }

  private handlePointerDown = (event: PointerEvent) => {
    if (event.button !== 0) return
    this.pointerId = event.pointerId
    this.pointerStart = event.clientX
    this.scrollStart = this.scroll.target
    isDragging.value = true
    this.container.setPointerCapture?.(event.pointerId)
  }

  private handlePointerMove = (event: PointerEvent) => {
    if (event.pointerId !== this.pointerId) return
    // 触摸端保留纵向页面滚动，横向手势仍由 touch-action 驱动画廊。
    if (event.pointerType !== 'touch') event.preventDefault()
    this.scroll.target = this.scrollStart + (this.pointerStart - event.clientX) * this.scrollSpeed * 0.012
  }

  private handlePointerUp = (event: PointerEvent) => {
    if (event.pointerId !== this.pointerId) return
    this.pointerId = null
    isDragging.value = false
    this.snap()
  }

  private handleKeyDown = (event: KeyboardEvent) => {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
    event.preventDefault()
    const width = this.medias[0]?.width || 1
    this.scroll.target += event.key === 'ArrowRight' ? width : -width
  }

  private update = () => {
    this.scroll.current = lerp(this.scroll.current, this.scroll.target, this.reducedMotion ? 0.2 : this.scrollEase)
    const direction = this.scroll.current >= this.scroll.last ? 'right' : 'left'
    this.medias.forEach((media) => media.update(this.scroll, direction, this.reducedMotion))
    this.renderer.render({ scene: this.scene, camera: this.camera })
    this.scroll.last = this.scroll.current
    this.raf = window.requestAnimationFrame(this.update)
  }

  destroy() {
    window.cancelAnimationFrame(this.raf)
    if (this.snapTimer !== null) window.clearTimeout(this.snapTimer)
    this.resizeObserver?.disconnect()
    if (!this.resizeObserver) window.removeEventListener('resize', this.handleResize)
    this.container.removeEventListener('wheel', this.handleWheel)
    this.container.removeEventListener('pointerdown', this.handlePointerDown)
    this.container.removeEventListener('keydown', this.handleKeyDown)
    window.removeEventListener('pointermove', this.handlePointerMove)
    window.removeEventListener('pointerup', this.handlePointerUp)
    window.removeEventListener('pointercancel', this.handlePointerUp)
    this.gl.canvas.remove()
    this.gl.getExtension('WEBGL_lose_context')?.loseContext()
    isDragging.value = false
  }
}

async function initializeGallery() {
  gallery?.destroy()
  gallery = null
  useFallback.value = false
  await nextTick()
  const container = containerRef.value
  if (!container || props.items.length === 0) return
  if (typeof WebGLRenderingContext === 'undefined' && typeof WebGL2RenderingContext === 'undefined') {
    useFallback.value = true
    return
  }

  try {
    gallery = new CircularGalleryApp(container, {
      items: props.items,
      isDark: props.isDark,
      bend: props.bend,
      scrollSpeed: props.scrollSpeed,
      scrollEase: props.scrollEase,
    })
  } catch {
    // WebGL 不可用时展示可横向滚动的语义化卡片，核心信息仍可访问。
    useFallback.value = true
  }
}

watch(
  () => [props.items, props.isDark],
  initializeGallery,
  { deep: true },
)

onMounted(initializeGallery)

onBeforeUnmount(() => {
  gallery?.destroy()
  gallery = null
})
</script>

<style scoped>
.circular-gallery {
  position: relative;
  width: 100%;
  height: clamp(24rem, 42vw, 32rem);
  overflow: hidden;
  cursor: grab;
  outline: none;
  touch-action: pan-y;
  mask-image: linear-gradient(90deg, transparent 0%, black 6%, black 94%, transparent 100%);
  -webkit-mask-image: linear-gradient(90deg, transparent 0%, black 6%, black 94%, transparent 100%);
}

.circular-gallery.is-dragging {
  cursor: grabbing;
}

.circular-gallery:focus-visible::after {
  position: absolute;
  inset: 12px;
  border: 1px solid rgba(13, 148, 136, 0.62);
  border-radius: 12px;
  pointer-events: none;
  content: '';
}

.circular-gallery :deep(canvas) {
  display: block;
  width: 100%;
  height: 100%;
}

.circular-gallery__fallback {
  display: flex;
  gap: 1rem;
  height: 100%;
  overflow-x: auto;
  align-items: center;
  padding: 1.5rem 6%;
  scroll-snap-type: x mandatory;
}

.circular-gallery__fallback-card {
  width: min(78vw, 25rem);
  min-width: min(78vw, 25rem);
  min-height: 16rem;
  padding: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 0 22px 54px -38px rgba(15, 23, 42, 0.3);
  scroll-snap-align: center;
}

.circular-gallery__fallback-code {
  font: 700 0.75rem ui-monospace, SFMono-Regular, Menlo, monospace;
}

.circular-gallery__fallback-card h3 {
  margin-top: 4.5rem;
  color: #0f172a;
  font-size: 1.25rem;
  font-weight: 700;
}

.circular-gallery__fallback-card p {
  margin-top: 0.75rem;
  color: #475569;
  font-size: 0.875rem;
  line-height: 1.75;
}

.circular-gallery.is-dark .circular-gallery__fallback-card {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(7, 11, 20, 0.94);
}

.circular-gallery.is-dark .circular-gallery__fallback-card h3 {
  color: #f8fafc;
}

.circular-gallery.is-dark .circular-gallery__fallback-card p {
  color: #94a3b8;
}

@media (max-width: 639px) {
  .circular-gallery {
    height: 23rem;
    mask-image: linear-gradient(90deg, transparent 0%, black 3%, black 97%, transparent 100%);
    -webkit-mask-image: linear-gradient(90deg, transparent 0%, black 3%, black 97%, transparent 100%);
  }
}
</style>
