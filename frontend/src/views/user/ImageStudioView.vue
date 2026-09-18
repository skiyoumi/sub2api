<template>
  <AppLayout>
    <div class="studio">
      <section class="studio-hero">
        <div class="hero-copy">
          <p class="eyebrow">
            <Icon name="sparkles" size="md" /> {{ t('imageStudio.eyebrow') }}
          </p>
          <h1>{{ t('imageStudio.headline') }}</h1>
          <p class="hero-description">{{ t('imageStudio.intro') }}</p>
          <div class="hero-features">
            <span>✧ {{ t('imageStudio.featureText') }}</span>
            <span>▧ {{ t('imageStudio.featureImage') }}</span>
            <span>◈ {{ t('imageStudio.featureMultiple') }}</span>
          </div>
        </div>
        <div class="hero-art" aria-hidden="true">
          <div class="art-panel art-0">
          </div>
          <div class="art-panel art-4">
          </div>
          <div class="art-panel art-1">
          </div>
        </div>
      </section>

      <section class="binding card" :aria-label="t('imageStudio.key')">
        <div class="binding-field">
          <div class="field-heading">
            <label for="studio-key">{{ t('imageStudio.key') }}</label>
            <RouterLink to="/keys">{{ t('imageStudio.manageKeys') }} ↗</RouterLink>
          </div>
          <select
            id="studio-key"
            v-model="keyID"
            class="input"
            :disabled="loadingKeys || submitting"
          >
            <option value="">{{ loadingKeys ? t('common.loading') : t('imageStudio.chooseKey') }}</option>
            <option v-for="key in keys" :key="key.id" :value="String(key.id)">{{ key.name }} · {{ key.group?.name }} · ••••{{ key.key.slice(-4) }}</option>
          </select>
          <p class="field-hint" v-if="selectedKey">
            <span class="group-chip">{{ selectedKey.group?.name }}</span>
          </p>
        </div>
        <span class="binding-arrow" aria-hidden="true">→</span>
        <div class="binding-field">
          <div class="field-heading">
            <label for="studio-model">{{ t('imageStudio.models') }}</label>
            <span v-if="models.length && !loadingModels" class="synced">✓ {{ t('imageStudio.synced') }}</span>
          </div>
          <div class="model-row">
            <select
              id="studio-model"
              v-model="model"
              class="input"
              :disabled="!selectedKey || loadingModels || submitting || !models.length"
            >
              <option value="">{{ loadingModels ? t('common.loading') : t('imageStudio.chooseModel') }}</option>
              <option v-for="item in models" :key="item" :value="item">{{ item }}</option>
            </select>
            <button
              class="btn btn-secondary"
              :aria-label="t('imageStudio.refresh')"
              :disabled="!selectedKey || loadingModels || submitting"
              @click="loadBinding"
            >
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loadingModels }" />
            </button>
          </div>
          <p class="field-hint">{{ t('imageStudio.modelHint') }}</p>
        </div>
      </section>
      <p v-if="!loadingKeys && !keys.length" class="notice" role="status">{{ t('imageStudio.noKeys') }}</p>
      <p v-if="selectedKey && !loadingModels && !models.length && !loadError" class="notice" role="status">{{ t('imageStudio.noModels') }}</p>
      <p v-if="storageEnabled === false" class="notice" role="status">{{ t('imageStudio.unavailable') }}</p>
      <p v-if="loadError" class="notice error" role="alert">
        {{ loadError }} <button @click="loadBinding">{{ t('imageStudio.refresh') }}</button>
      </p>

      <div class="workspace">
        <section class="composer card">
          <div class="mode-tabs" :aria-label="t('imageStudio.title')">
            <button
              v-for="option in ['text', 'image']"
              :key="option"
              :class="{ active: mode === option }"
              :aria-pressed="mode === option"
              @click="setMode(option)"
            >
              {{ t(option === 'text' ? 'imageStudio.textMode' : 'imageStudio.imageMode') }}
            </button>
          </div>
          <label for="studio-prompt" class="field-heading">{{ t('imageStudio.prompt') }}</label>
          <textarea
            id="studio-prompt"
            ref="promptInput"
            v-model="prompt"
            class="input prompt-input"
            maxlength="8000"
            :placeholder="t('imageStudio.promptPlaceholder')"
            @keydown.ctrl.enter.prevent="generate"
            @keydown.meta.enter.prevent="generate"
          >
</textarea>
          <span class="character-count">{{ prompt.length }} / 8000</span>
          <div class="reference-zone" @dragover.prevent @drop.prevent="dropReference">
            <template v-if="referenceURL">
              <img :src="referenceURL" :alt="reference?.name" />
              <div class="reference-name">
                {{ reference?.name }}<button class="text-primary-600" @click="clearReference">{{ t('imageStudio.remove') }}</button>
              </div>
            </template>
            <label v-else class="reference-upload">
              <span class="text-2xl">＋</span>
              <span>{{ t('imageStudio.reference') }}</span>
              <small>{{ t('imageStudio.referenceHint') }}</small>
              <input
                type="file"
                class="sr-only"
                accept="image/png,image/jpeg,image/webp"
                @change="pickReference"
              />
            </label>
          </div>
          <div class="composer-footer">
            <div ref="settingsRoot" class="settings-root" @keydown.esc="settingsOpen = false">
              <button
                class="settings-trigger"
                :aria-expanded="settingsOpen"
                aria-controls="studio-properties"
                @click="settingsOpen = !settingsOpen"
              >
                ☷ &nbsp; {{ ratio === 'custom' ? dimensions || t('imageStudio.customSize') : ratio }}<template v-if="ratio !== 'custom'"> · {{ resolution }}</template> · {{ t(`imageStudio.${quality}`) }} · ×{{ count }} <span>⌃</span>
              </button>
              <div
                v-if="settingsOpen"
                id="studio-properties"
                class="properties"
                :aria-label="t('imageStudio.settings')"
              >
                <label for="studio-ratio" class="field-heading">{{ t('imageStudio.chooseRatio') }}</label>
                <select id="studio-ratio" v-model="ratio" class="input">
                  <option v-for="r in ratios" :key="r" :value="r">{{ r }} · {{ t(`imageStudio.ratioNames.${r.replace(':', '_')}`) }}</option>
                  <option value="custom">{{ t('imageStudio.customSize') }}</option>
                </select>
                <p class="ratio-description">{{ ratio === 'custom' ? t('imageStudio.customHint') : t(`imageStudio.ratioHints.${ratio.replace(':', '_')}`) }}</p>
                <div v-if="ratio === 'custom'" class="custom-dimensions">
                  <label for="studio-width">{{ t('imageStudio.width') }}<input id="studio-width" v-model.number="customWidth" class="input" type="number" min="1" step="1" /></label>
                  <span>×</span>
                  <label for="studio-height">{{ t('imageStudio.height') }}<input id="studio-height" v-model.number="customHeight" class="input" type="number" min="1" step="1" /></label>
                </div>
                <p v-if="!dimensions" class="error" role="alert">{{ t('imageStudio.invalidDimensions') }}</p>
                <p v-else class="dimension-preview">{{ t('imageStudio.requestSize') }}: {{ dimensions }} px</p>
                <p class="field-hint">{{ t('imageStudio.requestTargetHint') }}</p>
                <p v-if="capabilities.grok" class="field-hint">{{ t('imageStudio.grokSizeHint') }}</p>
                <template v-if="ratio !== 'custom'">
                <p>{{ t('imageStudio.resolution') }}</p>
                <div class="property-options">
                  <button
                    v-for="r in ['1K', '2K', '4K']"
                    :key="r"
                    :disabled="!capabilities.resolutions.includes(r)"
                    :class="{ selected: resolution === r }"
                    :aria-pressed="resolution === r"
                    @click="resolution = r"
                  >
                    {{ r }}
                  </button>
                </div>
                </template>
                <p>{{ t('imageStudio.quality') }}</p>
                <div class="property-options">
                  <button
                    v-for="q in qualities"
                    :key="q"
                    :disabled="!capabilities.quality && q !== 'auto'"
                    :class="{ selected: quality === q }"
                    :aria-pressed="quality === q"
                    @click="quality = q"
                  >
                    {{ t(`imageStudio.${q}`) }}
                  </button>
                </div>
                <label for="studio-count" class="count-label">{{ t('imageStudio.count') }}<strong>{{ count }}</strong>
                </label>
                <input
                  id="studio-count"
                  v-model.number="count"
                  type="range"
                  min="1"
                  max="4"
                  step="1"
                />
                <div class="range-labels">
                  <span>1</span>
                  <span>4</span>
                </div>
              </div>
              <p class="field-hint">{{ t('imageStudio.capabilityHint') }}</p>
            </div>
            <button class="generate-button" :disabled="!canGenerate" @click="generate">
              <Icon name="sparkles" size="md" />
              <span>{{ t(submitting ? 'imageStudio.submitting' : 'imageStudio.generate') }}<small>Ctrl + Enter</small>
              </span>
            </button>
          </div>
          <p v-if="selectedKey" class="field-hint">{{ t('imageStudio.usingKey', { name: selectedKey.name }) }}</p>
          <p v-if="generationError" class="notice error" role="alert">{{ generationError }}</p>
        </section>

        <section class="results card">
          <div class="result-heading">
            <h2>{{ t(historyOpen ? 'imageStudio.history' : 'imageStudio.results') }}</h2>
            <button class="history-button" @click="toggleHistory">
              <Icon name="clock" size="sm" />{{ t(historyOpen ? 'imageStudio.back' : 'imageStudio.history') }}
            </button>
          </div>
          <template v-if="historyOpen">
            <p class="field-hint">{{ t('imageStudio.historyHint') }}</p>
            <div class="history-list">
              <article v-for="task in visibleTasks" :key="task.id" class="history-item">
                <button class="history-select" @click="selectTask(task)">
                  <span>{{ formatTime(task.created_at) }}<small>{{ task.request?.model }}</small></span>
                  <span :class="{ 'text-primary-600': task.status === 'completed' }">{{ statusLabel(task.status) }} →</span>
                </button>
                <p v-if="task.request?.size || task.result?.data?.some(image => image.width && image.height)" class="history-dimensions">
                  <span v-if="task.request?.size">{{ t('imageStudio.submittedSize') }} {{ formatSize(task.request.size) }}</span>
                  <span v-if="taskOutputSizes(task)">{{ t('imageStudio.actualSize') }} {{ taskOutputSizes(task) }}</span>
                </p>
                <p v-if="task.request?.prompt" class="history-prompt">{{ task.request.prompt }}</p>
                <div v-if="task.request?.prompt" class="prompt-actions">
                  <button @click="copyTaskPrompt(task)">{{ t('imageStudio.copyPrompt') }}</button>
                  <button @click="reuseTaskPrompt(task)">{{ t('imageStudio.usePrompt') }}</button>
                </div>
              </article>
              <p v-if="!visibleTasks.length && !historyError" class="empty-history">{{ t(historyLoading || loadingModels ? 'common.loading' : !selectedKey ? 'imageStudio.chooseKey' : 'imageStudio.noHistory') }}</p>
            </div>
          </template>
          <template v-else>
            <div v-if="activeTask?.status === 'processing'" class="empty-result" role="status">
              <Icon name="sparkles" size="xl" class="animate-pulse text-primary-500" />
              <h3>{{ t('imageStudio.generating') }}</h3>
              <p>{{ t('imageStudio.generatingHint') }}</p>
            </div>
            <div v-else-if="activeTask?.status === 'failed'" class="empty-result" role="alert">
              <h3>{{ t('imageStudio.failed') }}</h3>
              <p>{{ activeTask.error?.message }}</p>
            </div>
            <template v-else-if="imageURLs.length">
              <div class="preview">
                <img :src="imageURLs[activeImage]" :alt="t('imageStudio.results')" />
                <span class="preview-count">{{ activeImage + 1 }} / {{ imageURLs.length }}</span>
              </div>
              <div class="thumbnail-row">
                <button
                  v-for="(url, index) in imageURLs"
                  :key="url"
                  :class="{ selected: index === activeImage }"
                  :aria-label="`${t('imageStudio.results')} ${index + 1}`"
                  @click="activeImage = index"
                >
                  <img :src="url" alt="" />
                </button>
              </div>
              <div class="result-actions">
                <span class="expiry">
                  <Icon name="clock" size="sm" />{{ t('imageStudio.expires', { minutes: remainingMinutes }) }}</span>
                <button class="btn btn-secondary btn-sm" :disabled="!!originalAction" @click="reuseImage">{{ t(originalAction === 'reuse' ? 'common.loading' : 'imageStudio.reuse') }}</button>
                <button class="btn btn-primary btn-sm" :disabled="!!originalAction" @click="downloadImage">
                  <Icon name="download" size="sm" />{{ t(originalAction === 'download' ? 'imageStudio.loadingOriginal' : 'imageStudio.download') }}
                </button>
              </div>
            </template>
            <div v-else class="empty-result">
              <div class="empty-art">✧</div>
              <h3>{{ t(loadingImages ? 'common.loading' : 'imageStudio.emptyTitle') }}</h3>
              <p>{{ t('imageStudio.emptyHint') }}</p>
            </div>
            <p v-if="imageError" class="notice error" role="alert">{{ imageError }}</p>
            <div v-if="activeTask?.request?.prompt" class="saved-prompt">
              <strong>{{ t('imageStudio.originalPrompt') }}</strong>
              <p>{{ activeTask.request.prompt }}</p>
              <div class="prompt-actions">
                <button @click="copyTaskPrompt(activeTask)">{{ t('imageStudio.copyPrompt') }}</button>
                <button @click="reuseTaskPrompt(activeTask)">{{ t('imageStudio.usePrompt') }}</button>
              </div>
            </div>
          </template>
          <p v-if="historyError" class="notice error" role="alert">{{ historyError }} <button @click="pollTasks">{{ t('imageStudio.refresh') }}</button></p>
          <p class="retention">
            <Icon name="clock" size="sm" />{{ t('imageStudio.retention') }}
          </p>
        </section>
      </div>

      <section class="plaza">
        <div class="plaza-heading">
          <h2>
            <Icon name="sparkles" size="lg" />{{ t('imageStudio.plaza') }}
          </h2>
          <p>{{ t('imageStudio.plazaHint') }}</p>
        </div>
        <div class="categories">
          <button
            v-for="c in ['featured', ...categories]"
            :key="c"
            :class="{ active: category === c }"
            :aria-pressed="category === c"
            @click="category = c"
          >
            {{ t(`imageStudio.${c}`) }}
          </button>
        </div>
        <div class="prompt-grid">
          <button
            v-for="item in filteredPrompts"
            :key="item.name"
            class="prompt-card"
            @click="applyPrompt(item.name)"
          >
            <span
              class="prompt-art"
              :class="`art-${item.index}`"
              role="img"
              :aria-label="t(`imageStudio.${item.name}Title`)"
            >
            </span>
            <strong>{{ t(`imageStudio.${item.name}Title`) }}</strong>
            <p>{{ t(`imageStudio.${item.name}Prompt`) }}</p>
            <span class="use-prompt">{{ t('imageStudio.use') }} →</span>
          </button>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { keysAPI } from '@/api/keys'
import { fetchStudioImage, listStudioModels, listStudioTasks, studioCapabilities, studioDimensions, studioRatios, submitStudioTask, type StudioTask } from '@/api/imageStudio'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { readStudioSession, saveStudioSelection } from '@/utils/imageStudioSession'

const { t } = useI18n()
const app = useAppStore()
const userID = useAuthStore().user?.id || 0
const { copyToClipboard } = useClipboard()
const savedSession = readStudioSession(userID)
const keys = ref<ApiKey[]>([])
const keyID = ref('')
const model = ref('')
const models = ref<string[]>([])
const loadingKeys = ref(true)
const loadingModels = ref(false)
const submitting = ref(false)
const storageEnabled = ref<boolean | null>(null)
const loadError = ref('')
const generationError = ref('')
const prompt = ref('')
const mode = ref('text')
const reference = ref<File>()
const referenceURL = ref('')
const settingsOpen = ref(false)
const settingsRoot = ref<HTMLElement>()
const promptInput = ref<HTMLTextAreaElement>()
const ratios = studioRatios
const qualities = ['auto', 'low', 'medium', 'high']
const ratio = ref('1:1')
const resolution = ref('1K')
const quality = ref('auto')
const count = ref(1)
const customWidth = ref(1024)
const customHeight = ref(1024)
const dimensions = computed(() => { try { return studioDimensions(ratio.value, resolution.value, customWidth.value, customHeight.value) } catch { return '' } })
const tasks = ref<StudioTask[]>([])
const activeTaskID = ref('')
const activeImage = ref(0)
const historyOpen = ref(false)
const historyLoading = ref(false)
const historyError = ref('')
const imageURLs = ref<string[]>([])
const originalBlobs = new Map<number, Blob>()
const originalURLs = new Map<number, string>()
const originalAction = ref<'download' | 'reuse' | ''>('')
const imageError = ref('')
const loadingImages = ref(false)
const now = ref(Date.now())
const categories = ['product', 'poster', 'illustration', 'portrait', 'architecture']
const category = ref('featured')
const filteredPrompts = computed(() => categories.map((name, index) => ({ name, index })).filter(item => category.value === 'featured' || item.name === category.value))
const selectedKey = computed(() => keys.value.find(key => String(key.id) === keyID.value))
const capabilities = computed(() => studioCapabilities(model.value))
const visibleTasks = computed(() => tasks.value.filter(task => task.expires_at * 1000 > now.value))
const activeTask = computed(() => visibleTasks.value.find(task => task.id === activeTaskID.value))
const taskImages = computed(() => (activeTask.value?.result?.data || []).filter(image => !!image.url))
const remainingMinutes = computed(() => Math.max(0, Math.ceil(((activeTask.value?.expires_at || 0) * 1000 - now.value) / 60000)))
const canGenerate = computed(() => !!dimensions.value && !!selectedKey.value && selectedKey.value.status === 'active' && (!selectedKey.value.expires_at || Date.parse(selectedKey.value.expires_at) > now.value) && (!selectedKey.value.quota || selectedKey.value.quota_used < selectedKey.value.quota) && models.value.includes(model.value) && !!prompt.value.trim() && !loadingModels.value && !submitting.value && storageEnabled.value === true && (mode.value !== 'image' || !!reference.value))
let bindingController: AbortController | undefined
let imageController: AbortController | undefined
let originalController: AbortController | undefined
let bindingSequence = 0
let imageSequence = 0
let disposed = false
let pollingSequence: number | undefined
let taskRevision = 0
let tick: ReturnType<typeof setInterval> | undefined

const message = (error: unknown) => error instanceof Error ? error.message : t('errors.networkError')
const formatTime = (timestamp: number) => new Date(timestamp * 1000).toLocaleString()
const formatSize = (size?: string) => size?.replace(/^(\d+)x(\d+)$/, '$1 × $2') || ''
const taskOutputSizes = (task: StudioTask) => [...new Set((task.result?.data || []).filter(item => item.width && item.height).map(item => `${item.width} × ${item.height}`))].join(' / ')
const statusLabel = (status: string) => t(`imageStudio.${status === 'processing' ? 'generating' : status === 'failed' ? 'failed' : 'completed'}`)

async function loadKeys() {
  try {
    let page = 1
    const all: ApiKey[] = []
    while (true) {
      const result = await keysAPI.list(page, 100)
      all.push(...result.items)
      if (page >= result.pages || !result.items.length) break
      page++
    }
    if (disposed) return
    keys.value = all.filter(key => key.status !== 'inactive' && key.group?.allow_image_generation && ['openai', 'grok', 'composite'].includes(key.group.platform))
    if (keys.value.some(key => String(key.id) === savedSession.keyID)) keyID.value = savedSession.keyID
  } catch (error) { if (!disposed) loadError.value = message(error) }
  finally { loadingKeys.value = false }
}

async function loadBinding() {
  const saved = readStudioSession(userID).selections[keyID.value]
  saveStudioSelection(userID, keyID.value)
  const sequence = ++bindingSequence
  bindingController?.abort()
  bindingController = new AbortController()
  models.value = []
  model.value = ''
  tasks.value = []
  activeTaskID.value = ''
  storageEnabled.value = null
  loadError.value = ''
  generationError.value = ''
  historyError.value = ''
  clearImages()
  const key = selectedKey.value
  if (!key) { loadingModels.value = false; return }
  loadingModels.value = true
  const signal = bindingController.signal
  const results = await Promise.allSettled([listStudioModels(key.key, signal), listStudioTasks(key.key, signal)])
  if (sequence !== bindingSequence || disposed) return
  const [modelResult, taskResult] = results
  if (modelResult.status === 'fulfilled') { models.value = modelResult.value; model.value = modelResult.value.includes(saved?.model || '') ? saved!.model : modelResult.value[0] || '' }
  else loadError.value = message(modelResult.reason)
  if (taskResult.status === 'fulfilled') {
    tasks.value = taskResult.value.data || []
    storageEnabled.value = taskResult.value.enabled
    activeTaskID.value = visibleTasks.value.find(task => task.status === 'processing')?.id || visibleTasks.value.find(task => task.id === saved?.taskID)?.id || visibleTasks.value[0]?.id || ''
    if (!prompt.value && activeTask.value?.request?.prompt) restoreTaskInputs(activeTask.value)
  }
  else historyError.value = message(taskResult.reason)
  loadingModels.value = false
  rememberSelection()
}

function clearImages() {
  ++imageSequence
  imageController?.abort()
  originalController?.abort()
  imageURLs.value.forEach(url => URL.revokeObjectURL(url))
  originalURLs.forEach(url => URL.revokeObjectURL(url))
  originalURLs.clear()
  originalBlobs.clear()
  originalAction.value = ''
  imageURLs.value = []
  activeImage.value = 0
  imageError.value = ''
  loadingImages.value = false
}

async function loadImages() {
  clearImages()
  const sequence = imageSequence
  const task = activeTask.value
  const key = selectedKey.value
  if (!key || task?.status !== 'completed') return
  const images = taskImages.value
  if (!images.length) return
  imageController = new AbortController()
  const signal = imageController.signal
  loadingImages.value = true
  try {
    const blobs = await Promise.all(images.map(async (image, index) => {
      let original = !image.preview_url
      let blob: Blob
      try { blob = await fetchStudioImage(key.key, image.preview_url || image.url!, signal) }
      catch (error) {
        if (!image.preview_url || signal.aborted) throw error
        // An unavailable optional preview must not hide a valid original.
        blob = await fetchStudioImage(key.key, image.url!, signal)
        original = true
      }
      if (original && sequence === imageSequence && !disposed) originalBlobs.set(index, blob)
      return blob
    }))
    if (disposed || sequence !== imageSequence || !activeTask.value) return
    imageURLs.value = blobs.map(blob => URL.createObjectURL(blob))
  } catch (error) { if (sequence === imageSequence && !disposed) imageError.value = message(error) }
  finally { if (sequence === imageSequence) loadingImages.value = false }
}

async function pollTasks() {
  if (pollingSequence === bindingSequence || !selectedKey.value || loadingModels.value || disposed) return
  const key = selectedKey.value
  const sequence = bindingSequence
  const revision = taskRevision
  pollingSequence = sequence
  historyLoading.value = true
  try {
    const result = await listStudioTasks(key.key, bindingController?.signal)
    if (sequence !== bindingSequence || disposed) return
    if (revision !== taskRevision) return
    const next = (result.data || []).filter(task => task.expires_at * 1000 > Date.now())
    const newest = next[0]
    const isNew = newest && !tasks.value.some(task => task.id === newest.id)
    tasks.value = next
    if (isNew && !historyOpen.value) activeTaskID.value = newest.id
    if (!activeTask.value) activeTaskID.value = visibleTasks.value[0]?.id || ''
    storageEnabled.value = result.enabled
    historyError.value = ''
  } catch (error) { if (sequence === bindingSequence && !disposed) historyError.value = message(error) }
  finally { if (pollingSequence === sequence) { pollingSequence = undefined; historyLoading.value = false } }
}

function rememberSelection() {
  if (!loadingModels.value && selectedKey.value && !disposed) saveStudioSelection(userID, keyID.value, { model: model.value, taskID: activeTaskID.value })
}
function toggleHistory() { historyOpen.value = !historyOpen.value; if (historyOpen.value) void pollTasks() }

async function generate() {
  if (!canGenerate.value || !selectedKey.value) return
  const key = selectedKey.value
  submitting.value = true
  generationError.value = ''
  settingsOpen.value = false
  try {
    const task = await submitStudioTask(key.key, { model: model.value, prompt: prompt.value.trim(), ratio: ratio.value, resolution: resolution.value, quality: quality.value, count: count.value, width: customWidth.value, height: customHeight.value, reference: reference.value })
    if (disposed || key.id !== selectedKey.value?.id) return
    ++taskRevision
    tasks.value = [task, ...tasks.value.filter(item => item.id !== task.id)]
    activeTaskID.value = task.id
    historyOpen.value = false
  } catch (error) { if (!disposed) generationError.value = message(error) }
  finally { submitting.value = false }
}

function clearReference() { if (referenceURL.value) URL.revokeObjectURL(referenceURL.value); referenceURL.value = ''; reference.value = undefined }
function setMode(value: string) { mode.value = value; if (value === 'text') clearReference() }
function setReference(file?: File) {
  if (!file) return
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type) || file.size > 10 * 1024 * 1024) { generationError.value = t('imageStudio.referenceInvalid'); return }
  clearReference()
  reference.value = file
  referenceURL.value = URL.createObjectURL(file)
  mode.value = 'image'
  generationError.value = ''
}
function pickReference(event: Event) { const input = event.target as HTMLInputElement; setReference(input.files?.[0]); input.value = '' }
function dropReference(event: DragEvent) { setReference(event.dataTransfer?.files[0]) }
function applyPrompt(name: string) { prompt.value = t(`imageStudio.${name}Prompt`); promptInput.value?.focus(); app.showSuccess(t('imageStudio.applied')) }
function selectTask(task: StudioTask) { activeTaskID.value = task.id; historyOpen.value = false }
function restoreTaskInputs(task: StudioTask) {
  const input = task.request
  if (!input) return
  prompt.value = input.prompt
  if (input.quality && qualities.includes(input.quality) && capabilities.value.quality) quality.value = input.quality
  if (input.n && input.n >= 1 && input.n <= 4) count.value = input.n
  if (input.aspect_ratio && ratios.includes(input.aspect_ratio)) ratio.value = input.aspect_ratio
  if (input.size && /^\d+x\d+$/.test(input.size)) {
    const preset = capabilities.value.resolutions.flatMap(res => ratios.map(r => ({ ratio: r, resolution: res }))).find(option => studioDimensions(option.ratio, option.resolution) === input.size)
    if (preset) { ratio.value = preset.ratio; resolution.value = preset.resolution }
    else { const [width, height] = input.size.split('x').map(Number); ratio.value = 'custom'; customWidth.value = width!; customHeight.value = height! }
  }
}
function copyTaskPrompt(task: StudioTask) { if (task.expires_at * 1000 > Date.now() && task.request?.prompt) void copyToClipboard(task.request.prompt) }
function reuseTaskPrompt(task: StudioTask) {
  if (task.expires_at * 1000 <= Date.now() || !task.request?.prompt) return
  prompt.value = task.request.prompt
  historyOpen.value = false
  promptInput.value?.focus()
  app.showSuccess(t('imageStudio.applied'))
}
async function useOriginalImage(action: 'download' | 'reuse') {
  const task = activeTask.value
  const key = selectedKey.value
  const index = activeImage.value
  const image = taskImages.value[index]
  if (originalAction.value || !key || !task || task.expires_at * 1000 <= Date.now() || !image?.url) return
  const sequence = imageSequence
  originalAction.value = action
  imageError.value = ''
  originalController = new AbortController()
  try {
    const blob = originalBlobs.get(index) || await fetchStudioImage(key.key, image.url, originalController.signal)
    if (disposed || sequence !== imageSequence || task.expires_at * 1000 <= Date.now()) return
    originalBlobs.set(index, blob)
    const ext = blob.type.split('/')[1] || 'png'
    if (action === 'reuse') {
      setReference(new File([blob], `reference.${ext}`, { type: blob.type }))
    } else {
      let url = originalURLs.get(index)
      if (!url) { url = URL.createObjectURL(blob); originalURLs.set(index, url) }
      const link = document.createElement('a')
      link.href = url
      link.download = `${task.id}-${index + 1}.${ext}`
      link.click()
    }
  } catch (error) { if (sequence === imageSequence && !disposed) imageError.value = message(error) }
  finally { if (sequence === imageSequence) originalAction.value = '' }
}
function downloadImage() { void useOriginalImage('download') }
function reuseImage() { void useOriginalImage('reuse') }
function closeSettings(event: MouseEvent) { if (!settingsRoot.value?.contains(event.target as Node)) settingsOpen.value = false }

watch(keyID, loadBinding)
watch(model, () => { if (!capabilities.value.resolutions.includes(resolution.value)) resolution.value = '1K'; if (!capabilities.value.quality) quality.value = 'auto' })
watch([model, activeTaskID], rememberSelection)
watch(() => `${keyID.value}:${activeTask.value?.id || ''}:${activeTask.value?.status || ''}`, loadImages)
onMounted(() => {
  void loadKeys()
  let seconds = 0
  tick = setInterval(() => {
    now.value = Date.now()
    tasks.value = tasks.value.filter(task => task.expires_at * 1000 > now.value)
    if (++seconds % 4 === 0 && document.visibilityState !== 'hidden') void pollTasks()
  }, 1000)
  document.addEventListener('visibilitychange', refreshVisibleTasks)
  document.addEventListener('click', closeSettings)
})
function refreshVisibleTasks() { if (document.visibilityState === 'visible') void pollTasks() }
onBeforeUnmount(() => { rememberSelection(); disposed = true; ++bindingSequence; bindingController?.abort(); clearImages(); clearReference(); tasks.value = []; if (tick) clearInterval(tick); document.removeEventListener('click', closeSettings); document.removeEventListener('visibilitychange', refreshVisibleTasks) })
</script>

<style scoped>
.studio {
  max-width: 1600px;
  margin: auto;
  color: #162337;
}
.studio-hero {
  position: relative;
  display: flex;
  overflow: hidden;
  min-height: 184px;
  border-radius: 18px;
  background: linear-gradient(115deg,#e4f7f5,#f5faf6 75%,#edf6ed);
  margin-bottom: 18px;
}
.hero-copy {
  position: relative;
  z-index: 1;
  padding: 26px 30px;
  width: 72%;
}
.eyebrow {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #0d9488;
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 1px;
}
.hero-copy h1 {
  font-size: clamp(25px,2.5vw,38px);
  font-weight: 750;
  letter-spacing: -1px;
  margin: 9px 0;
}
.hero-description {
  font-size: 13px;
  color: #627181;
  max-width: 700px;
  line-height: 1.8;
}
.hero-features {
  display: flex;
  gap: 24px;
  margin-top: 16px;
  font-size: 12px;
  color: #487d78;
}
.hero-art {
  position: absolute;
  right: 8px;
  top: -12px;
  bottom: -16px;
  width: 32%;
  display: flex;
  gap: 10px;
  transform: rotate(9deg);
}
.art-panel {
  width: 33%;
  border: 6px solid white;
  box-shadow: 0 5px 15px #213b2820;
  background-image: url('@/assets/image-studio/inspiration.png');
  background-size: 500% 100%;
}
.art-panel:nth-child(2) {
  transform: translateY(22px);
}
.art-0 {
  background-position: 0% center;
}
.art-1 {
  background-position: 25% center;
}
.art-2 {
  background-position: 50% center;
}
.art-3 {
  background-position: 75% center;
}
.art-4 {
  background-position: 100% center;
}
.card {
  background: #fff;
  border: 1px solid #e4ebef;
  border-radius: 14px;
  box-shadow: 0 1px 3px #12302b03;
}
.binding {
  display: flex;
  gap: 24px;
  align-items: center;
  padding: 20px 24px;
  margin-bottom: 18px;
}
.binding-field {
  flex: 1;
  min-width: 0;
}
.field-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}
.field-heading a,.synced {
  color: #0d9488;
  font-size: 11px;
  font-weight: 500;
}
.synced,.group-chip {
  border-radius: 6px;
  background: #eff9f7;
  padding: 3px 7px;
}
.binding-arrow {
  color: #8495a5;
  font-size: 24px;
}
.model-row {
  display: flex;
  gap: 8px;
}
.model-row select {
  min-width: 0;
  flex: 1;
}
.field-hint {
  font-size: 11px;
  color: #8492a4;
  margin-top: 7px;
  line-height: 1.6;
}
.workspace {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}
.composer,.results {
  padding: 20px;
  min-width: 0;
}
.mode-tabs {
  display: flex;
  max-width: 310px;
  background: #f3f6f8;
  border-radius: 9px;
  padding: 3px;
  margin-bottom: 20px;
}
.mode-tabs button {
  flex: 1;
  padding: 8px 12px;
  border-radius: 7px;
  font-size: 13px;
  color: #57677b;
}
.mode-tabs .active {
  background: #dff7f2;
  color: #0d9488;
  font-weight: 600;
}
.prompt-input {
  width: 100%;
  height: 144px;
  resize: vertical;
  font-size: 13px;
  line-height: 1.9;
}
.character-count {
  display: block;
  margin: 5px 0 10px;
  color: #8b98a7;
  font-size: 11px;
}
.reference-zone {
  min-height: 103px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #d8e3e8;
  background: #f8fafc;
  border-radius: 9px;
  gap: 12px;
  padding: 12px;
}
.reference-upload {
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  color: #61738a;
  font-size: 13px;
  width: 100%;
}
.reference-upload small {
  font-size: 10px;
  color: #92a0b1;
}
.reference-zone img {
  height: 76px;
  width: 76px;
  object-fit: cover;
  border-radius: 6px;
}
.reference-name {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  overflow-wrap: anywhere;
}
.composer-footer {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-top: 20px;
}
.settings-root {
  position: relative;
}
.settings-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid #74c6bd;
  background: #f1fbf9;
  border-radius: 8px;
  padding: 10px;
  color: #255a59;
  font-size: 12px;
  white-space: nowrap;
}
.properties {
  position: absolute;
  bottom: 72px;
  left: 0;
  background: white;
  border: 1px solid #dde5ea;
  box-shadow: 0 12px 40px #132a4326;
  width: 360px;
  max-width: 80vw;
  max-height: 70vh;
  overflow-y: auto;
  padding: 20px;
  border-radius: 16px;
  z-index: 20;
}
.properties p {
  font-size: 12px;
  margin: 0 0 8px;
}
.property-options {
  display: flex;
  gap: 8px;
  margin-bottom: 17px;
}
.property-options button {
  flex: 1;
  border: 1px solid transparent;
  border-radius: 9px;
  padding: 6px 4px;
  font-size: 12px;
  color: #758395;
}
.property-options button.selected {
  border-color: #14b8a6;
  color: #0d9488;
  background: #edfcf8;
}
.property-options button:disabled {
  opacity: .3;
  cursor: not-allowed;
}
.custom-dimensions {
  display: flex;
  align-items: end;
  gap: 10px;
  margin-bottom: 12px;
}
.custom-dimensions label { flex: 1; min-width: 0; font-size: 12px; }
.custom-dimensions input { margin-top: 6px; }
.custom-dimensions > span { padding-bottom: 10px; }
.properties .ratio-description { color: #64748b; line-height: 1.7; margin: 8px 0 14px; }
.properties .dimension-preview { color: #0d9488; margin-bottom: 16px; }
.saved-prompt { padding: 14px; margin-top: 16px; background: #f3f9f8; border-radius: 10px; font-size: 12px; }
.saved-prompt > p { white-space: pre-wrap; overflow-wrap: anywhere; max-height: 140px; overflow: auto; margin-top: 8px; line-height: 1.7; }
.history-dimensions { display: flex; flex-wrap: wrap; gap: 4px 12px; color: #64748b; font-size: 11px; margin-bottom: 6px; }
.prompt-actions { display: flex; gap: 16px; margin-top: 10px; }
.prompt-actions button { color: #0d9488; font-size: 12px; }
.history-item { border-bottom: 1px solid #edf1f5; padding: 12px 0; }
.history-select small { display: block; color: #8292a7; margin-top: 5px; }
.history-prompt { color: #64748b; font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
:global(.dark) .saved-prompt { background: #142d2b;
}
.count-label {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  margin: 10px 0;
}
.properties input[type=range] {
  width: 100%;
  accent-color: #0d9488;
}
.range-labels {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: #9aa3af;
}
.generate-button {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 10px;
  background: linear-gradient(110deg,#0d9488,#079fa6);
  color: white;
  border-radius: 9px;
  padding: 12px 20px;
  font-size: 14px;
  min-width: 160px;
  box-shadow: 0 4px 14px #0d948817;
}
.generate-button small {
  display: block;
  font-size: 9px;
  opacity: .6;
  margin-top: 2px;
}
.generate-button:disabled {
  opacity: .45;
  cursor: not-allowed;
}
.result-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}
.result-heading h2,.plaza h2 {
  font-size: 18px;
  font-weight: 650;
}
.history-button {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  border: 1px solid #e2e8ef;
  padding: 7px 9px;
  border-radius: 7px;
  color: #637185;
}
.results {
  display: flex;
  flex-direction: column;
}
.empty-result {
  display: flex;
  flex: 1;
  min-height: 310px;
  align-items: center;
  justify-content: center;
  text-align: center;
  flex-direction: column;
  background: radial-gradient(ellipse at center,#f0f9f6,#f8fafb);
  border-radius: 10px;
  padding: 30px;
}
.empty-art {
  font-size: 64px;
  color: #90c5bc;
}
.empty-result h3 {
  font-weight: 600;
  font-size: 16px;
  margin: 14px 0 8px;
}
.empty-result p {
  font-size: 12px;
  color: #8a98a8;
  max-width: 340px;
  line-height: 1.8;
}
.preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 300px;
  background: #f5f8fa;
  border-radius: 10px;
  overflow: hidden;
}
.preview img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.preview-count {
  position: absolute;
  right: 10px;
  bottom: 10px;
  padding: 3px 7px;
  border-radius: 5px;
  color: white;
  background: #142921a8;
  font-size: 10px;
}
.thumbnail-row {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}
.thumbnail-row button {
  height: 55px;
  width: 75px;
  padding: 2px;
  border: 2px solid transparent;
  border-radius: 7px;
}
.thumbnail-row button.selected {
  border-color: #0d9488;
}
.thumbnail-row img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 4px;
}
.result-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  margin-top: 12px;
}
.expiry {
  display: flex;
  gap: 4px;
  align-items: center;
  font-size: 10px;
  color: #a17645;
  margin-right: auto;
}
.retention {
  display: flex;
  gap: 6px;
  align-items: center;
  font-size: 11px;
  color: #8a98aa;
  line-height: 1.6;
  margin-top: 14px;
}
.history-list {
  min-height: 330px;
  max-height: 390px;
  overflow: auto;
}
.history-list .history-select {
  display: flex;
  width: 100%;
  justify-content: space-between;
  text-align: left;
  gap: 10px;
  padding-bottom: 8px;
  font-size: 12px;
}
.empty-history {
  padding: 70px 20px;
  text-align: center;
  color: #8a98a8;
  font-size: 13px;
}
.plaza {
  margin-top: 26px;
}
.plaza-heading {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}
.plaza h2 {
  display: flex;
  gap: 9px;
  align-items: center;
}
.plaza h2 svg {
  color: #0d9488;
}
.plaza-heading p {
  font-size: 11px;
  color: #8896a7;
}
.categories {
  display: flex;
  gap: 7px;
  flex-wrap: wrap;
  margin: 14px 0;
}
.categories button {
  font-size: 11px;
  color: #798698;
  background: #edf2f6;
  padding: 6px 16px;
  border-radius: 20px;
}
.categories .active {
  background: #d6f6ef;
  color: #0d9488;
  font-weight: 600;
}
.prompt-grid {
  display: grid;
  grid-template-columns: repeat(5,minmax(0,1fr));
  gap: 12px;
}
.prompt-card {
  text-align: left;
  background: white;
  border: 1px solid #e3eaf0;
  border-radius: 10px;
  overflow: hidden;
  padding: 5px;
  transition: transform .18s,border-color .18s;
}
.prompt-card:hover {
  transform: translateY(-3px);
  border-color: #60beaf;
}
.prompt-art {
  display: block;
  aspect-ratio: 3 / 4;
  border-radius: 6px;
  background-image: url('@/assets/image-studio/inspiration.png');
  /* Five portrait panels share one sprite; auto height preserves their proportions. */
  background-size: 500% auto;
  background-repeat: no-repeat;
}
.prompt-card strong {
  display: block;
  font-size: 12px;
  margin: 10px 6px 5px;
}
.prompt-card p {
  font-size: 10px;
  color: #91a0b0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  margin: 0 6px;
}
.use-prompt {
  display: block;
  color: #0d9488;
  font-size: 11px;
  margin: 10px 6px 7px;
}
.notice {
  font-size: 12px;
  line-height: 1.7;
  padding: 10px 14px;
  background: #fff8e8;
  color: #9a6b27;
  border-radius: 8px;
  margin-bottom: 12px;
}
.notice.error {
  background: #fff1f2;
  color: #b54b56;
}
.notice button {
  text-decoration: underline;
  margin-left: 8px;
}
:global(.dark) .studio {
  color: #e2e8f0;
}
:global(.dark) .studio .card,:global(.dark) .properties,:global(.dark) .prompt-card {
  background: #172333;
  border-color: #2a394b;
}
:global(.dark) .studio-hero {
  background: linear-gradient(115deg,#123a39,#1b2d30);
}
:global(.dark) .hero-description {
  color: #a1b7b9;
}
:global(.dark) .hero-art {
  opacity: .65;
}
:global(.dark) .mode-tabs,:global(.dark) .reference-zone,:global(.dark) .preview {
  background: #1c2b3c;
}
:global(.dark) .empty-result {
  background: radial-gradient(ellipse,#18322f,#1a2636);
}
:global(.dark) .categories button {
  background: #223044;
}
:global(.dark) .mode-tabs .active,:global(.dark) .categories .active,:global(.dark) .settings-trigger,:global(.dark) .group-chip {
  background: #13423e;
  color: #5eead4;
}
@media(min-width:1600px) {
  .prompt-input {
    height: 170px;
  }
  .preview {
    height: 330px;
  }
}
@media(max-width:1100px) {
  .workspace {
    grid-template-columns: 1fr;
  }
  .hero-copy {
    width: 80%;
  }
  .hero-art {
    width: 26%;
    opacity: .7;
  }
  .prompt-grid {
    grid-template-columns: repeat(3,minmax(0,1fr));
  }
  .preview {
    height: 420px;
  }
}
@media(max-width:640px) {
  .hero-copy {
    padding: 22px;
    width: 100%;
  }
  .hero-art {
    display: none;
  }
  .hero-features {
    gap: 12px;
    font-size: 10px;
  }
  .binding {
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
    padding: 18px;
  }
  .binding-arrow {
    display: none;
  }
  .composer,.results {
    padding: 16px;
  }
  .composer-footer {
    flex-wrap: wrap;
  }
  .properties {
    position: fixed;
    left: 16px;
    right: 16px;
    bottom: 16px;
    width: auto;
    max-width: none;
    max-height: calc(100dvh - 96px);
    overflow-y: auto;
    z-index: 60;
  }
  .generate-button {
    flex: 1;
  }
  .prompt-grid {
    grid-template-columns: repeat(2,minmax(0,1fr));
  }
  .plaza-heading {
    gap: 7px;
  }
  .preview {
    height: 300px;
  }
}
</style>
