<template>
  <canvas ref="canvas" class="mf-background" aria-hidden="true" />
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { createMofangBackground } from './mofangBackground'

const props = defineProps<{ light: boolean; animate: boolean; page: 'home' | 'login' }>()
const canvas = ref<HTMLCanvasElement | null>(null)
let renderer: ReturnType<typeof createMofangBackground> | undefined
let frame = 0
let clock = 0
let previous = 0
let observer: ResizeObserver | undefined
let width = 0
let height = 0
let ratio = 1
let visible = true

function paint() {
  const ctx = canvas.value?.getContext('2d')
  if (!ctx || !renderer || !width || !height) return
  ctx.setTransform(1, 0, 0, 1, 0, 0)
  ctx.clearRect(0, 0, width * ratio, height * ratio)
  const scale = Math.max(width / 1600, height / 1000)
  ctx.setTransform(scale * ratio, 0, 0, scale * ratio,
    (width - 1600 * scale) * .5 * ratio, (height - 1000 * scale) * .5 * ratio)
  renderer.draw(clock, props.light, props.page, !props.animate)
}
function resize() {
  if (!canvas.value) return
  const rect = canvas.value.getBoundingClientRect()
  width = rect.width
  height = rect.height
  ratio = Math.min(window.devicePixelRatio || 1, 1.5)
  canvas.value.width = Math.round(width * ratio)
  canvas.value.height = Math.round(height * ratio)
  paint()
}
function tick(now: number) {
  if (!props.animate || !visible) { frame = 0; return }
  if (!previous) previous = now
  const elapsed = now - previous
  if (elapsed >= 1000 / 30) {
    clock += Math.min(elapsed, 70)
    previous = now
    paint()
  }
  frame = requestAnimationFrame(tick)
}
function syncMotion() {
  cancelAnimationFrame(frame)
  frame = 0
  previous = 0
  paint()
  if (props.animate && visible) frame = requestAnimationFrame(tick)
}
function onVisibility() { visible = !document.hidden; syncMotion() }
function onPointer(event: PointerEvent) {
  renderer?.pointer(event.clientX / window.innerWidth * 2 - 1, event.clientY / window.innerHeight * 2 - 1)
}
watch(() => [props.animate, props.light, props.page], syncMotion)
onMounted(() => {
  const ctx = canvas.value?.getContext('2d')
  if (!ctx || !canvas.value) return
  renderer = createMofangBackground(ctx)
  observer = new ResizeObserver(resize)
  observer.observe(canvas.value)
  visible = !document.hidden
  resize()
  syncMotion()
  document.addEventListener('visibilitychange', onVisibility)
  window.addEventListener('pointermove', onPointer, { passive: true })
})
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  observer?.disconnect()
  document.removeEventListener('visibilitychange', onVisibility)
  window.removeEventListener('pointermove', onPointer)
})
</script>
