<template>
  <div ref="root" class="mf-theme-control" @keydown.esc.stop.prevent="close(true)">
    <button ref="trigger" class="mf-theme-trigger" type="button" :aria-label="t('mofang.appearance')"
      aria-haspopup="menu" :aria-expanded="open" @click="open = !open" @keydown.down.prevent="openMenu">
      <Icon :name="options.find(option => option.value === modelValue)?.icon || 'grid'" size="sm" />
      <span>{{ t('mofang.' + modelValue) }}</span><Icon name="chevronDown" size="xs" />
    </button>
    <div v-if="open" ref="menu" class="mf-theme-menu" role="menu" :aria-label="t('mofang.appearance')" @keydown="onMenuKey">
      <button v-for="option in options" :key="option.value" type="button" role="menuitemradio"
        :aria-checked="modelValue === option.value" @click="select(option.value)">
        <Icon :name="option.icon" size="sm" /><span>{{ t('mofang.' + option.value) }}</span>
        <Icon v-if="modelValue === option.value" name="check" size="xs" class="mf-menu-check" />
      </button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { MofangTheme } from '@/composables/useMofangAppearance'
defineProps<{ modelValue: MofangTheme }>()
const emit = defineEmits<{ 'update:modelValue': [value: MofangTheme] }>()
const { t } = useI18n()
const open = ref(false)
const root = ref<HTMLElement | null>(null)
const menu = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const options = [
  { value: 'light', icon: 'sun' }, { value: 'dark', icon: 'moon' }, { value: 'system', icon: 'grid' },
] as const
function close(focus = false) { open.value = false; if (focus) trigger.value?.focus() }
function select(value: MofangTheme) { emit('update:modelValue', value); close(true) }
async function openMenu() { open.value = true; await nextTick(); menu.value?.querySelector('button')?.focus() }
function onMenuKey(event: KeyboardEvent) {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const buttons = Array.from(menu.value?.querySelectorAll('button') || [])
  const index = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1
    : (index + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length
  buttons[next]?.focus()
}
function outside(event: PointerEvent) { if (event.target instanceof Node && !root.value?.contains(event.target)) close() }
onMounted(() => document.addEventListener('pointerdown', outside))
onBeforeUnmount(() => document.removeEventListener('pointerdown', outside))
</script>
