<template>
  <div v-if="homeContent.trim()" class="mf-custom-home">
    <iframe v-if="customUrl" :src="customUrl" :title="siteName" allowfullscreen />
    <!-- home_content is trusted HTML configured by an administrator, as in the legacy home. -->
    <div v-else v-html="homeContent" />
  </div>
  <MofangShell v-else :class="{ 'mf-compact': compact }">
    <main class="mf-wrap">
      <section class="mf-hero" aria-labelledby="mofang-home-title">
        <div class="mf-hero-copy">
          <div class="mf-eyebrow"><span class="mf-pulse-dot" />{{ t('mofang.eyebrow') }}<Icon name="arrowRight" size="xs" /></div>
          <h1 id="mofang-home-title">{{ t('mofang.heroFirst') }}<br /><span class="mf-gradient-text">{{ t('mofang.heroSecond') }}</span></h1>
          <p>{{ t('mofang.description') }}<br />{{ t('mofang.descriptionSecond') }}</p>
          <div class="mf-hero-actions">
            <RouterLink :to="entryPath" class="mf-button mf-button-primary">{{ t(authStore.isAuthenticated ? 'mofang.console' : 'mofang.start') }}<Icon name="arrowRight" size="sm" /></RouterLink>
            <a v-if="docUrl" class="mf-button" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('mofang.viewDocs') }}<Icon name="book" size="sm" /></a>
          </div>
          <div class="mf-micro-features"><span v-for="key in ['sdk', 'routing', 'billing']" :key="key"><Icon name="check" size="xs" />{{ t('mofang.' + key) }}</span></div>
        </div>
        <MofangScene />
      </section>
      <template v-if="!compact">
        <section class="mf-model-band" :aria-label="t('mofang.models')">
          <div class="mf-model-caption"><b>{{ t('mofang.models') }}</b><br />ONE KEY. ALL MODELS.</div>
          <MofangModels />
          <RouterLink v-if="showModelPlaza" to="/model-plaza" class="mf-model-explore">{{ t('mofang.explore') }}<Icon name="arrowRight" size="xs" /></RouterLink>
        </section>
        <section class="mf-feature-grid">
          <article v-for="feature in features" :key="feature.key" class="mf-feature-card">
            <div class="mf-card-top"><span class="mf-feature-icon"><Icon :name="feature.icon" size="sm" /></span><span class="mf-card-index">{{ feature.index }}</span></div>
            <h2>{{ t('mofang.' + feature.key + 'Title') }}</h2>
            <p>{{ t('mofang.' + feature.key + 'Text') }}<br />{{ t('mofang.' + feature.key + 'More') }}</p>
          </article>
        </section>
      </template>
    </main>
  </MofangShell>
</template>
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import MofangShell from '@/components/mofang/MofangShell.vue'
import MofangScene from '@/components/mofang/MofangScene.vue'
import MofangModels from '@/components/mofang/MofangModels.vue'
import { useMofangBranding } from '@/composables/useMofangBranding'
import { sanitizeUrl } from '@/utils/url'
const { t } = useI18n()
const { siteName, docUrl, entryPath, showModelPlaza, authStore, appStore, settings } = useMofangBranding()
const homeContent = computed(() => settings.value?.home_content || '')
const customUrl = computed(() => sanitizeUrl(homeContent.value))
const compact = computed(() => settings.value?.compact_home_enabled === true)
const features = [
  { key: 'connect', icon: 'key', index: '01 / CONNECT' },
  { key: 'route', icon: 'swap', index: '02 / ROUTING' },
  { key: 'usage', icon: 'chart', index: '03 / INSIGHT' },
] as const
onMounted(() => { if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings() })
</script>
