<template>
  <BaseDialog :show="show" :title="t('keys.ccsImport.title')" width="normal" @close="emit('close')">
    <div class="space-y-4">
      <fieldset>
        <legend class="input-label">{{ t('keys.ccsImport.app') }}</legend>
        <div class="flex flex-wrap gap-x-5 gap-y-2">
          <label v-for="option in appOptions" :key="option.value" class="flex cursor-pointer items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
            <input v-model="form.app" type="radio" :value="option.value" class="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ option.label }}
          </label>
        </div>
      </fieldset>

      <div>
        <label class="input-label">{{ t('keys.ccsImport.name') }}</label>
        <input v-model="form.name" class="input" maxlength="100" />
      </div>

      <div>
        <label class="input-label">{{ t('keys.ccsImport.mainModel') }} <span class="text-red-500">*</span></label>
        <select v-model="form.model" class="input" :disabled="loadingModels">
          <option v-for="model in models" :key="model" :value="model">{{ model }}</option>
        </select>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          {{ loadingModels ? t('keys.ccsImport.loadingModels') : t('keys.ccsImport.modelHint', { count: models.length }) }}
        </p>
      </div>

      <template v-if="form.app === 'claude'">
        <div v-for="field in claudeFields" :key="field.key">
          <label class="input-label">{{ field.label }}</label>
          <select v-model="form[field.key]" class="input" :disabled="loadingModels">
            <option v-for="model in models" :key="model" :value="model">{{ model }}</option>
          </select>
        </div>
      </template>

      <p v-if="loadError" class="text-sm text-red-600 dark:text-red-400">{{ loadError }}</p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loadingModels || !form.model || !form.name.trim()" @click="submit">
          {{ t('keys.ccsImport.open') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { CcSwitchDefaults, GroupPlatform } from '@/types'
import type { CcSwitchApp } from '@/utils/ccswitchImport'

const props = defineProps<{
  show: boolean
  apiKey: string
  baseUrl: string
  providerName: string
  platform?: GroupPlatform | null
  defaults?: CcSwitchDefaults
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'submit', value: { app: CcSwitchApp; name: string; model: string; haikuModel?: string; sonnetModel?: string; opusModel?: string }): void
}>()

const { t } = useI18n()
const models = ref<string[]>([])
const loadingModels = ref(false)
const loadError = ref('')
const form = reactive({ app: 'claude' as CcSwitchApp, name: '', model: '', haikuModel: '', sonnetModel: '', opusModel: '' })

const appOptions = computed(() => [
  { value: 'claude' as const, label: 'Claude' },
  { value: 'codex' as const, label: 'Codex' },
  { value: 'gemini' as const, label: 'Gemini' },
  { value: 'opencode' as const, label: 'OpenCode' },
])

const claudeFields = computed(() => [
  { key: 'haikuModel' as const, label: t('keys.ccsImport.haikuModel') },
  { key: 'sonnetModel' as const, label: t('keys.ccsImport.sonnetModel') },
  { key: 'opusModel' as const, label: t('keys.ccsImport.opusModel') },
])

function preferredApp(): CcSwitchApp {
  if (props.platform === 'openai') return 'codex'
  if (props.platform === 'gemini') return 'gemini'
  return 'claude'
}

function pick(preferred: string | undefined, pattern?: RegExp): string {
  if (preferred && models.value.includes(preferred)) return preferred
  return (pattern ? models.value.find(model => pattern.test(model)) : undefined) || models.value[0] || ''
}

function applyDefaults() {
  const defaults = props.defaults || {}
  const appDefault = form.app === 'codex'
    ? defaults.codex
    : form.app === 'gemini'
      ? defaults.gemini
      : form.app === 'opencode'
        ? defaults.opencode
        : defaults.claude?.model
  form.model = pick(
    appDefault,
    form.app === 'claude' ? /sonnet/i : undefined,
  )
  form.haikuModel = pick(defaults.claude?.haiku, /haiku/i)
  form.sonnetModel = pick(defaults.claude?.sonnet, /sonnet/i)
  form.opusModel = pick(defaults.claude?.opus, /opus/i)
}

async function loadModels() {
  loadingModels.value = true
  loadError.value = ''
  try {
    const gatewayRoot = props.baseUrl.replace(/\/+$/, '').replace(/\/v1$/i, '')
    const response = await fetch(`${gatewayRoot}/v1/models`, { headers: { Authorization: `Bearer ${props.apiKey}` } })
    if (!response.ok) throw new Error(`${t('keys.ccsImport.loadFailed')} (${response.status})`)
    const body = await response.json()
    models.value = Array.from(new Set((Array.isArray(body?.data) ? body.data : []).map((item: { id?: unknown }) => String(item?.id || '').trim()).filter(Boolean)))
    if (!models.value.length) throw new Error(t('keys.ccsImport.noModels'))
    applyDefaults()
  } catch (error) {
    models.value = []
    loadError.value = error instanceof Error ? error.message : t('keys.ccsImport.loadFailed')
  } finally {
    loadingModels.value = false
  }
}

watch(() => props.show, (show) => {
  if (!show) return
  form.app = preferredApp()
  form.name = props.providerName
  void loadModels()
}, { immediate: true })

watch(() => form.app, () => {
  if (form.app === 'opencode') {
    form.name = 'modelscube'
  } else if (form.name === 'modelscube') {
    form.name = props.providerName
  }
  if (models.value.length) applyDefaults()
})

function submit() {
  emit('submit', {
    app: form.app,
    name: form.name.trim(),
    model: form.model,
    haikuModel: form.app === 'claude' ? form.haikuModel : undefined,
    sonnetModel: form.app === 'claude' ? form.sonnetModel : undefined,
    opusModel: form.app === 'claude' ? form.opusModel : undefined,
  })
}
</script>
