<template>
  <AppLayout>
    <div class="flex min-h-[calc(100vh-7.5rem)] gap-4">
      <aside class="hidden w-64 shrink-0 overflow-hidden rounded-lg border border-gray-200 bg-white lg:block dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-100 p-3 dark:border-dark-700">
          <div class="relative">
            <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="search" class="input h-9 pl-9 text-sm" :placeholder="t('modelPricing.search')" />
          </div>
        </div>
        <div class="max-h-[calc(100vh-12rem)] overflow-y-auto p-3">
          <section v-for="section in platformSections" :key="section.platform" class="mb-4 last:mb-0">
            <div class="mb-2 flex items-center justify-between px-1">
              <span class="text-xs font-semibold uppercase text-gray-700 dark:text-gray-200">{{ section.platform }}</span>
              <span class="text-[11px] text-gray-400">{{ section.groups.length }}</span>
            </div>
            <button
              v-for="group in section.groups"
              :key="group.id"
              type="button"
              class="mb-1 flex w-full items-center justify-between gap-2 rounded-md px-2.5 py-2 text-left text-sm transition-colors"
              :class="selectedGroupId === group.id ? 'bg-primary-50 text-primary-700 ring-1 ring-primary-200 dark:bg-primary-950/30 dark:text-primary-300 dark:ring-primary-800' : 'text-gray-600 hover:bg-gray-50 dark:text-dark-300 dark:hover:bg-dark-700'"
              @click="selectedGroupId = group.id"
            >
              <span class="truncate">{{ group.name }}</span>
              <span class="shrink-0 font-mono text-[11px] text-gray-400">x{{ effectiveRate(group) }}</span>
            </button>
          </section>
        </div>
      </aside>

      <main class="min-w-0 flex-1 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900/30">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <div class="min-w-0">
            <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ selectedGroup?.name || t('modelPricing.title') }}</h2>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ selectedGroup ? `${selectedGroup.platform} · ${selectedGroup.models.length} ${t('modelPricing.models')}` : t('modelPricing.description') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex rounded-lg border border-gray-200 p-0.5 dark:border-dark-600">
              <button type="button" class="rounded-md px-3 py-1.5 text-xs" :class="mode === 'cards' ? 'bg-primary-600 text-white' : 'text-gray-500'" @click="mode = 'cards'">{{ t('modelPricing.cards') }}</button>
              <button type="button" class="rounded-md px-3 py-1.5 text-xs" :class="mode === 'table' ? 'bg-primary-600 text-white' : 'text-gray-500'" @click="mode = 'table'">{{ t('modelPricing.table') }}</button>
            </div>
            <button type="button" class="btn-ghost btn-icon border border-gray-200 dark:border-dark-600" :title="t('common.refresh')" @click="load">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-2 border-b border-gray-100 p-3 dark:border-dark-700 sm:grid-cols-2 lg:hidden">
          <select v-model.number="selectedGroupId" class="input h-9 text-sm">
            <option v-for="group in data?.groups ?? []" :key="group.id" :value="group.id">
              {{ group.name }} · x{{ effectiveRate(group) }}
            </option>
          </select>
          <div class="relative">
            <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="search" class="input h-9 pl-9 text-sm" :placeholder="t('modelPricing.search')" />
          </div>
        </div>

        <div v-if="loading" class="flex min-h-80 items-center justify-center"><LoadingSpinner /></div>
        <div v-else-if="loadFailed" class="p-10 text-center text-sm text-red-500">{{ t('modelPricing.loadFailed') }}</div>
        <div v-else-if="!selectedGroup" class="flex min-h-80 flex-col items-center justify-center px-6 text-center">
          <Icon name="calculator" size="xl" class="mb-3 text-gray-300 dark:text-dark-600" />
          <p class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('modelPricing.notConfigured') }}</p>
          <p class="mt-1 text-xs text-gray-400">{{ t('modelPricing.notConfiguredHint') }}</p>
        </div>
        <div v-else class="p-4">
          <p v-if="data?.description" class="mb-4 rounded-md border border-blue-100 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-900 dark:bg-blue-950/30 dark:text-blue-300">{{ data.description }}</p>
          <div v-if="mode === 'cards'" class="grid grid-cols-1 gap-4 xl:grid-cols-2 2xl:grid-cols-3">
            <ModelPriceCard
              v-for="model in filteredModels"
              :key="model.name"
              :model="model"
              :group-name="selectedGroup.name"
              :platform="selectedGroup.platform"
              :rate-multiplier="selectedGroup.rate_multiplier"
              :user-rate-multiplier="selectedGroup.user_rate_multiplier"
            />
          </div>
          <div v-else class="overflow-hidden rounded-lg border border-gray-200 px-4 dark:border-dark-700">
            <PlazaModelPricingTable
              :models="filteredModels"
              :platform="selectedGroup.platform"
              :rate-multiplier="selectedGroup.rate_multiplier"
              :user-rate-multiplier="selectedGroup.user_rate_multiplier"
            />
          </div>
          <p v-if="filteredModels.length === 0" class="py-12 text-center text-sm text-gray-400">{{ t('modelPricing.noResults') }}</p>
        </div>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ModelPriceCard from '@/components/modelPricing/ModelPriceCard.vue'
import PlazaModelPricingTable from '@/components/modelPlaza/PlazaModelPricingTable.vue'
import { getModelPricing } from '@/api/modelPricing'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'

const { t } = useI18n()
const data = ref<ModelPlazaResponse | null>(null)
const loading = ref(true)
const loadFailed = ref(false)
const selectedGroupId = ref<number | null>(null)
const mode = ref<'cards' | 'table'>('cards')
const search = ref('')

const platformSections = computed(() => {
  const map = new Map<string, ModelPlazaGroup[]>()
  for (const group of data.value?.groups ?? []) {
    const list = map.get(group.platform) ?? []
    list.push(group)
    map.set(group.platform, list)
  }
  return [...map.entries()].map(([platform, groups]) => ({ platform, groups }))
})

const selectedGroup = computed(() => data.value?.groups.find(group => group.id === selectedGroupId.value) ?? null)
const filteredModels = computed(() => {
  const query = search.value.trim().toLowerCase()
  const models = selectedGroup.value?.models ?? []
  return query ? models.filter(model => model.name.toLowerCase().includes(query)) : models
})

function effectiveRate(group: ModelPlazaGroup) {
  return group.user_rate_multiplier ?? group.rate_multiplier
}

watch(() => data.value?.groups, groups => {
  if (!groups?.some(group => group.id === selectedGroupId.value)) selectedGroupId.value = groups?.[0]?.id ?? null
})

async function load() {
  loading.value = true
  loadFailed.value = false
  try {
    data.value = await getModelPricing()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
