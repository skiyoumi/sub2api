<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex flex-wrap items-center gap-x-5 gap-y-3">
          <label class="flex cursor-pointer items-center gap-3">
            <Toggle v-model="form.enabled" />
            <span>
              <span class="block text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.modelPricing.enablePage') }}</span>
              <span class="block text-xs text-gray-500 dark:text-dark-400">{{ form.enabled ? t('common.enabled') : t('common.disabled') }}</span>
            </span>
          </label>
          <div class="hidden h-8 w-px bg-gray-200 dark:bg-dark-700 sm:block"></div>
          <div class="flex items-center gap-4 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('admin.modelPricing.selectedGroups') }} <strong class="ml-1 text-sm text-gray-900 dark:text-white">{{ form.groups.length }}</strong></span>
            <span>{{ t('admin.modelPricing.selectedModels') }} <strong class="ml-1 text-sm text-gray-900 dark:text-white">{{ selectedModelCount }}</strong></span>
          </div>
        </div>
        <button class="btn-primary inline-flex min-w-24 items-center justify-center gap-2" :disabled="saving || loading" @click="save">
          <Icon name="check" size="sm" />
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>

      <div class="grid min-h-[560px] overflow-hidden rounded-lg border border-gray-200 bg-white lg:grid-cols-[300px,minmax(0,1fr)] dark:border-dark-700 dark:bg-dark-800">
        <aside class="border-b border-gray-200 lg:border-b-0 lg:border-r dark:border-dark-700">
          <div class="border-b border-gray-100 p-3 dark:border-dark-700">
            <div class="relative">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input v-model="groupSearch" class="input h-9 pl-9 text-sm" :placeholder="t('admin.modelPricing.searchGroups')" />
            </div>
          </div>

          <div v-if="loading" class="flex min-h-64 items-center justify-center"><LoadingSpinner /></div>
          <div v-else class="max-h-[620px] overflow-y-auto p-2">
            <button
              v-for="group in filteredGroups"
              :key="group.id"
              type="button"
              class="mb-1 flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-left transition-colors last:mb-0"
              :class="activeGroupId === group.id ? 'bg-primary-50 ring-1 ring-primary-200 dark:bg-primary-950/30 dark:ring-primary-800' : 'hover:bg-gray-50 dark:hover:bg-dark-700/60'"
              @click="activeGroupId = group.id"
            >
              <input
                type="checkbox"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="isGroupSelected(group.id)"
                :aria-label="group.name"
                @click.stop
                @change="toggleGroup(group)"
              />
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-medium text-gray-800 dark:text-dark-100">{{ group.name }}</span>
                <span class="mt-0.5 flex items-center gap-1.5 text-[11px] text-gray-400">
                  <PlatformIcon :platform="group.platform as GroupPlatform" size="xs" />
                  {{ platformName(group.platform) }} · x{{ group.rate_multiplier }}
                </span>
              </span>
              <span
                class="shrink-0 rounded px-1.5 py-0.5 text-[11px]"
                :class="isGroupSelected(group.id) ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' : 'bg-gray-100 text-gray-400 dark:bg-dark-700'"
              >
                {{ selectedCount(group.id) }}/{{ group.models.length }}
              </span>
            </button>
            <p v-if="filteredGroups.length === 0" class="px-4 py-10 text-center text-sm text-gray-400">{{ t('admin.modelPricing.noMatchingGroups') }}</p>
          </div>
        </aside>

        <main class="min-w-0">
          <template v-if="activeGroup">
            <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex min-w-0 items-center gap-3">
                <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gray-50 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-white dark:ring-dark-600">
                  <PlatformIcon :platform="activeGroup.platform as GroupPlatform" size="lg" />
                </div>
                <div class="min-w-0">
                  <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ activeGroup.name }}</h3>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ activeGroup.models.length }} {{ t('admin.modelPricing.availableModels') }}</p>
                </div>
              </div>
              <div class="flex items-center gap-3">
                <button type="button" class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="selectAllModels(activeGroup)">{{ t('admin.modelPricing.selectAllModels') }}</button>
                <button type="button" class="text-xs font-medium text-gray-500 hover:text-red-600 dark:text-dark-300 dark:hover:text-red-400" @click="clearModels(activeGroup.id)">{{ t('admin.modelPricing.clearModels') }}</button>
              </div>
            </div>

            <div class="border-b border-gray-100 p-4 dark:border-dark-700">
              <div class="relative max-w-sm">
                <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input v-model="modelSearch" class="input h-9 pl-9 text-sm" :placeholder="t('admin.modelPricing.searchModelsOnly')" />
              </div>
            </div>

            <div class="p-4">
              <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
                <label
                  v-for="model in activeModels"
                  :key="model.name"
                  class="flex min-w-0 cursor-pointer items-center gap-2.5 rounded-md border px-3 py-2.5 transition-colors"
                  :class="isModelSelected(activeGroup.id, model.name) ? 'border-primary-200 bg-primary-50/60 dark:border-primary-800 dark:bg-primary-950/20' : 'border-gray-200 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/50'"
                >
                  <input
                    type="checkbox"
                    class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :checked="isModelSelected(activeGroup.id, model.name)"
                    @change="toggleModel(activeGroup.id, model.name)"
                  />
                  <span class="truncate text-sm text-gray-700 dark:text-dark-200" :title="model.name">{{ model.name }}</span>
                </label>
              </div>
              <p v-if="activeModels.length === 0" class="py-12 text-center text-sm text-gray-400">{{ t('admin.modelPricing.noMatchingModels') }}</p>
            </div>
          </template>

          <div v-else class="flex min-h-[420px] flex-col items-center justify-center px-6 text-center">
            <Icon name="calculator" size="xl" class="mb-3 text-gray-300 dark:text-dark-600" />
            <p class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.modelPricing.selectGroupHint') }}</p>
          </div>
        </main>
      </div>

      <section class="border-t border-gray-200 pt-4 dark:border-dark-700">
        <div class="grid gap-3 lg:grid-cols-[220px,minmax(0,1fr)] lg:items-start">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.modelPricing.priceNote') }}</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.modelPricing.priceNoteHint') }}</p>
          </div>
          <div>
            <textarea v-model="form.description" rows="3" maxlength="4000" class="input resize-y text-sm" :placeholder="t('admin.modelPricing.priceNotePlaceholder')"></textarea>
            <p class="mt-1 text-right text-xs text-gray-400">{{ form.description.length }}/4000</p>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useAppStore } from '@/stores/app'
import { getModelPricingConfig, updateModelPricingConfig, type ModelPricingConfig } from '@/api/modelPricing'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const groups = ref<ModelPlazaGroup[]>([])
const loading = ref(true)
const saving = ref(false)
const groupSearch = ref('')
const modelSearch = ref('')
const activeGroupId = ref<number | null>(null)
const form = reactive<ModelPricingConfig>({ enabled: false, description: '', groups: [] })

const selectedModelCount = computed(() => form.groups.reduce((sum, group) => sum + group.models.length, 0))
const filteredGroups = computed(() => {
  const query = groupSearch.value.trim().toLowerCase()
  return query
    ? groups.value.filter(group => group.name.toLowerCase().includes(query) || group.platform.toLowerCase().includes(query))
    : groups.value
})
const activeGroup = computed(() => groups.value.find(group => group.id === activeGroupId.value) ?? null)
const activeModels = computed(() => {
  const models = activeGroup.value?.models ?? []
  const query = modelSearch.value.trim().toLowerCase()
  return query ? models.filter(model => model.name.toLowerCase().includes(query)) : models
})

watch(activeGroupId, () => { modelSearch.value = '' })

function selection(groupId: number) {
  return form.groups.find(item => item.group_id === groupId)
}

function isGroupSelected(groupId: number) {
  return selection(groupId) != null
}

function selectedCount(groupId: number) {
  return selection(groupId)?.models.length ?? 0
}

function isModelSelected(groupId: number, model: string) {
  return selection(groupId)?.models.includes(model) ?? false
}

function toggleGroup(group: ModelPlazaGroup) {
  const index = form.groups.findIndex(item => item.group_id === group.id)
  if (index >= 0) form.groups.splice(index, 1)
  else form.groups.push({ group_id: group.id, models: group.models.map(model => model.name) })
  activeGroupId.value = group.id
}

function ensureSelection(groupId: number) {
  let selected = selection(groupId)
  if (!selected) {
    selected = { group_id: groupId, models: [] }
    form.groups.push(selected)
  }
  return selected
}

function toggleModel(groupId: number, model: string) {
  const selected = ensureSelection(groupId)
  const index = selected.models.indexOf(model)
  if (index >= 0) selected.models.splice(index, 1)
  else selected.models.push(model)
  if (selected.models.length === 0) {
    form.groups.splice(form.groups.findIndex(item => item.group_id === groupId), 1)
  }
}

function selectAllModels(group: ModelPlazaGroup) {
  ensureSelection(group.id).models = group.models.map(model => model.name)
}

function clearModels(groupId: number) {
  const index = form.groups.findIndex(item => item.group_id === groupId)
  if (index >= 0) form.groups.splice(index, 1)
}

function platformName(platform: string) {
  return platform.charAt(0).toUpperCase() + platform.slice(1)
}

async function load() {
  loading.value = true
  try {
    const result = await getModelPricingConfig()
    groups.value = result.groups
    Object.assign(form, result.config)
    activeGroupId.value = result.config.groups[0]?.group_id ?? result.groups[0]?.id ?? null
  } catch {
    appStore.showError(t('admin.modelPricing.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const saved = await updateModelPricingConfig({
      enabled: form.enabled,
      description: form.description,
      groups: form.groups.filter(group => group.models.length > 0).map(group => ({ group_id: group.group_id, models: [...group.models] }))
    })
    Object.assign(form, saved)
    appStore.showSuccess(t('admin.modelPricing.saveSuccess'))
  } catch {
    appStore.showError(t('admin.modelPricing.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
