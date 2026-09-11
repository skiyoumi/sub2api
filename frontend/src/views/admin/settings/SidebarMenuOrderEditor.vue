<template>
  <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
    <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.customMenu.orderTitle') }}</h3>
    <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.customMenu.orderHint') }}</p>
    <div class="my-3 flex flex-wrap items-center justify-between gap-3">
      <div class="flex gap-2">
        <button
          v-for="section in sections" :key="section" type="button"
          :data-sidebar-section="section" :aria-pressed="activeSection === section"
          :class="activeSection === section ? 'btn btn-primary btn-sm' : 'btn btn-secondary btn-sm'"
          @click="activeSection = section"
        >{{ t(`admin.settings.customMenu.orderSection.${section}`) }}</button>
      </div>
      <button type="button" class="btn btn-secondary btn-sm" data-reset-menu-order @click="resetOrder">
        {{ t('admin.settings.customMenu.resetOrder') }}
      </button>
    </div>
    <VueDraggable
      v-if="entries.length" :key="activeSection" v-model="entries" :animation="150"
      handle=".menu-order-handle" class="max-h-96 space-y-1.5 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600"
    >
      <div
        v-for="(entry, index) in entries" :key="entry.path" :data-menu-path="entry.path"
        class="flex items-center gap-2 rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-800"
      >
        <span class="menu-order-handle cursor-grab touch-none text-gray-400 active:cursor-grabbing" aria-hidden="true">
          <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <circle v-for="n in 6" :key="n" :cx="n % 2 ? 7 : 13" :cy="4 + Math.floor((n - 1) / 2) * 6" r="1.5" />
          </svg>
        </span>
        <span class="min-w-0 flex-1 truncate text-sm text-gray-700 dark:text-gray-200">{{ entryLabel(entry) }}</span>
        <span v-if="entry.custom" class="text-xs text-primary-600 dark:text-primary-400">{{ t('admin.settings.customMenu.customItem') }}</span>
        <span v-if="entry.simpleModeOnly" class="text-xs text-gray-400">{{ t('admin.settings.customMenu.simpleModeOnly') }}</span>
        <button
          type="button" data-direction="up" :disabled="index === 0"
          :aria-label="`${t('admin.settings.customMenu.moveUp')} ${entryLabel(entry)}`"
          class="rounded px-2 py-1 hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-30 dark:hover:bg-dark-600"
          @click="moveEntry(index, -1)"
        ><span aria-hidden="true">↑</span></button>
        <button
          type="button" data-direction="down" :disabled="index === entries.length - 1"
          :aria-label="`${t('admin.settings.customMenu.moveDown')} ${entryLabel(entry)}`"
          class="rounded px-2 py-1 hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-30 dark:hover:bg-dark-600"
          @click="moveEntry(index, 1)"
        ><span aria-hidden="true">↓</span></button>
      </div>
    </VueDraggable>
    <p v-else class="py-4 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.customMenu.noVisibleMenus') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import type { CustomMenuItem, SidebarMenuOrder } from '@/types'
import { getSidebarMenuEntries, type SidebarMenuEntry, type SidebarSection } from '@/utils/sidebarMenus'
import { orderSidebarItems, reorderVisibleSidebarItems } from '@/utils/sidebarMenuOrder'
import { useSidebarMenus } from '@/composables/useSidebarMenus'

const props = defineProps<{ modelValue: SidebarMenuOrder; customItems: CustomMenuItem[] }>()
const emit = defineEmits<{ 'update:modelValue': [order: SidebarMenuOrder] }>()
const { t } = useI18n()
const { sections, getEntries } = useSidebarMenus()
const activeSection = ref<SidebarSection>('user')
watch(sections, (available) => {
  if (!available.includes(activeSection.value)) activeSection.value = available[0] ?? 'user'
}, { immediate: true })
const entries = computed({
  get: () => getEntries(activeSection.value, props.customItems, props.modelValue[activeSection.value]),
  set: (items: SidebarMenuEntry[]) => emit('update:modelValue', {
    ...props.modelValue,
    [activeSection.value]: reorderVisibleSidebarItems(
      orderSidebarItems(getSidebarMenuEntries(activeSection.value, props.customItems), props.modelValue[activeSection.value]),
      items,
    ),
  }),
})
function entryLabel(entry: SidebarMenuEntry): string {
  return entry.labelKey ? t(entry.labelKey) : entry.label || t('admin.settings.customMenu.unnamedItem')
}
function moveEntry(index: number, direction: -1 | 1) {
  const items = [...entries.value]
  const target = index + direction
  if (target < 0 || target >= items.length) return
  ;[items[index], items[target]] = [items[target], items[index]]
  entries.value = items
}
function resetOrder() {
  emit('update:modelValue', { ...props.modelValue, [activeSection.value]: [] })
}
</script>
