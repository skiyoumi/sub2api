<template>
  <AppLayout>
    <div class="grid min-h-[620px] overflow-hidden rounded-lg border border-gray-200 bg-white lg:grid-cols-[280px,minmax(0,1fr)] dark:border-dark-700 dark:bg-dark-800">
      <aside class="border-b border-gray-200 lg:border-b-0 lg:border-r dark:border-dark-700">
        <div class="border-b border-gray-100 p-3 dark:border-dark-700">
          <div class="relative">
            <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="search" class="input h-9 pl-9 text-sm" :placeholder="t('admin.ccSwitchDefaults.searchGroups')" />
          </div>
        </div>
        <div v-if="loadingGroups" class="flex min-h-64 items-center justify-center"><LoadingSpinner /></div>
        <div v-else class="max-h-[680px] overflow-y-auto p-2">
          <button
            v-for="group in filteredGroups"
            :key="group.id"
            type="button"
            class="mb-1 flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-left transition-colors"
            :class="activeGroup?.id === group.id ? 'bg-primary-50 ring-1 ring-primary-200 dark:bg-primary-950/30 dark:ring-primary-800' : 'hover:bg-gray-50 dark:hover:bg-dark-700/60'"
            @click="selectGroup(group)"
          >
            <PlatformIcon :platform="group.platform" size="sm" />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-medium text-gray-800 dark:text-dark-100">{{ group.name }}</span>
              <span class="text-xs text-gray-400">{{ group.platform }}</span>
            </span>
          </button>
          <p v-if="!filteredGroups.length" class="px-4 py-10 text-center text-sm text-gray-400">{{ t('admin.ccSwitchDefaults.noGroups') }}</p>
        </div>
      </aside>

      <main class="min-w-0">
        <template v-if="activeGroup">
          <header class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ activeGroup.name }}</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.ccSwitchDefaults.modelCount', { count: models.length }) }}</p>
            </div>
            <button class="btn btn-primary min-w-24" :disabled="saving || loadingModels" @click="save">
              <Icon name="check" size="sm" class="mr-2" />
              {{ saving ? t('common.saving') : t('common.save') }}
            </button>
          </header>

          <div v-if="loadingModels" class="flex min-h-96 items-center justify-center"><LoadingSpinner /></div>
          <div v-else-if="loadError" class="p-6 text-sm text-red-600 dark:text-red-400">{{ loadError }}</div>
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <section class="p-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Claude</h3>
              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <ModelSelect v-model="form.claude.model" :label="t('admin.ccSwitchDefaults.mainModel')" :models="models" />
                <ModelSelect v-model="form.claude.haiku" label="Haiku" :models="models" />
                <ModelSelect v-model="form.claude.sonnet" label="Sonnet" :models="models" />
                <ModelSelect v-model="form.claude.opus" label="Opus" :models="models" />
              </div>
            </section>
            <section class="grid gap-4 p-5 md:grid-cols-3">
              <ModelSelect v-model="form.codex" label="Codex" :models="models" />
              <ModelSelect v-model="form.gemini" label="Gemini" :models="models" />
              <ModelSelect v-model="form.opencode" label="OpenCode" :models="models" />
            </section>
          </div>
        </template>
        <div v-else class="flex min-h-[520px] flex-col items-center justify-center px-6 text-center">
          <Icon name="brain" size="xl" class="mb-3 text-gray-300 dark:text-dark-600" />
          <p class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.ccSwitchDefaults.selectGroup') }}</p>
        </div>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { groupsAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, CcSwitchDefaults } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const groups = ref<AdminGroup[]>([])
const activeGroup = ref<AdminGroup | null>(null)
const models = ref<string[]>([])
const search = ref('')
const loadingGroups = ref(true)
const loadingModels = ref(false)
const saving = ref(false)
const loadError = ref('')
const form = reactive({ claude: { model: '', haiku: '', sonnet: '', opus: '' }, codex: '', gemini: '', opencode: '' })

const ModelSelect = defineComponent({
  props: { modelValue: { type: String, default: '' }, label: { type: String, required: true }, models: { type: Array as () => string[], required: true } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('label', { class: 'block min-w-0' }, [
      h('span', { class: 'input-label' }, props.label),
      h('select', { class: 'input', value: props.modelValue, onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value) }, [
        h('option', { value: '' }, t('admin.ccSwitchDefaults.autoSelect')),
        ...props.models.map(model => h('option', { value: model }, model)),
      ]),
    ])
  },
})

const filteredGroups = computed(() => {
  const query = search.value.trim().toLowerCase()
  return query ? groups.value.filter(group => group.name.toLowerCase().includes(query) || group.platform.includes(query)) : groups.value
})

function resetForm(defaults?: CcSwitchDefaults) {
  form.claude.model = defaults?.claude?.model || ''
  form.claude.haiku = defaults?.claude?.haiku || ''
  form.claude.sonnet = defaults?.claude?.sonnet || ''
  form.claude.opus = defaults?.claude?.opus || ''
  form.codex = defaults?.codex || ''
  form.gemini = defaults?.gemini || ''
  form.opencode = defaults?.opencode || ''
}

async function selectGroup(group: AdminGroup) {
  activeGroup.value = group
  resetForm(group.models_list_config?.cc_switch_defaults)
  loadingModels.value = true
  loadError.value = ''
  try {
    models.value = await groupsAPI.getModelsListCandidates(group.id, group.platform)
  } catch (error) {
    models.value = []
    loadError.value = t('admin.ccSwitchDefaults.loadModelsFailed')
  } finally {
    loadingModels.value = false
  }
}

async function save() {
  if (!activeGroup.value) return
  saving.value = true
  try {
    const current = activeGroup.value.models_list_config || { enabled: false, models: [] }
    const updated = await groupsAPI.update(activeGroup.value.id, {
      models_list_config: {
        enabled: current.enabled,
        models: current.models || [],
        cc_switch_defaults: {
          claude: { ...form.claude },
          codex: form.codex,
          gemini: form.gemini,
          opencode: form.opencode,
        },
      },
    })
    const index = groups.value.findIndex(group => group.id === updated.id)
    if (index >= 0) groups.value[index] = updated
    activeGroup.value = updated
    appStore.showSuccess(t('admin.ccSwitchDefaults.saved'))
  } catch (error) {
    appStore.showError(t('admin.ccSwitchDefaults.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    groups.value = await groupsAPI.getAllIncludingInactive()
    if (groups.value.length) await selectGroup(groups.value[0])
  } catch (error) {
    appStore.showError(t('admin.ccSwitchDefaults.loadGroupsFailed'))
  } finally {
    loadingGroups.value = false
  }
})
</script>
