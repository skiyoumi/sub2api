<template>
  <div class="mf-site" :data-theme="isDark ? 'dark' : 'light'" :class="{ 'mf-motion-paused': !animate }">
    <MofangBackground :light="!isDark" :animate="animate" :page="auth ? 'login' : 'home'" />
    <div class="mf-vignette" aria-hidden="true" />
    <header class="mf-topbar">
      <nav class="mf-nav mf-wrap" :aria-label="t('mofang.tagline')">
        <RouterLink to="/home" class="mf-brand"><MofangLogo /><span class="mf-brand-words"><b>{{ siteName }}</b><small>{{ t('mofang.tagline') }}</small></span><span class="mf-brand-rule" /><span class="mf-brand-en">ModelsCube<em>AI GATEWAY</em></span></RouterLink>
        <div v-if="!auth" class="mf-nav-links">
          <RouterLink v-if="showModelPlaza" to="/model-plaza">{{ t('nav.modelPlaza') }}</RouterLink>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('mofang.docs') }}</a>
        </div>
        <div class="mf-nav-right">
          <button v-if="!reducedMotion" class="mf-motion-button" type="button" :aria-label="t(motionPaused ? 'mofang.play' : 'mofang.pause')" :title="t(motionPaused ? 'mofang.play' : 'mofang.pause')" :aria-pressed="motionPaused" @click="motionPaused = !motionPaused">
            <Icon v-if="motionPaused" name="play" size="sm" /><span v-else class="mf-pause-symbol" aria-hidden="true">Ⅱ</span>
          </button>
          <MofangThemeMenu :model-value="theme" @update:model-value="setTheme" />
          <LocaleSwitcher class="mf-locale" />
          <RouterLink v-if="auth" to="/home" class="mf-back-home"><Icon name="arrowLeft" size="sm" /><span>{{ t('mofang.backHome') }}</span></RouterLink>
          <template v-else>
            <RouterLink v-if="!authStore.isAuthenticated" to="/login" class="mf-nav-login">{{ t('home.login') }}</RouterLink>
            <RouterLink :to="entryPath" :aria-label="t('mofang.console')" class="mf-button mf-button-primary mf-nav-console">{{ t('mofang.console') }}<Icon name="arrowRight" size="sm" /></RouterLink>
          </template>
        </div>
      </nav>
    </header>
    <div class="mf-page"><slot /></div>
    <footer class="mf-footer mf-wrap"><span>© {{ new Date().getFullYear() }} {{ siteName }} · {{ t('mofang.tagline') }}</span><span>{{ t('mofang.motto') }}</span></footer>
  </div>
</template>
<script setup lang="ts">
import { provide } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { useMofangAppearance } from '@/composables/useMofangAppearance'
import { useMofangBranding } from '@/composables/useMofangBranding'
import MofangBackground from './MofangBackground.vue'
import MofangLogo from './MofangLogo.vue'
import MofangThemeMenu from './MofangThemeMenu.vue'
import { mofangDarkKey } from './mofangContext'
import '@/styles/mofang.css'
defineProps<{ auth?: boolean }>()
const { t } = useI18n()
const { theme, isDark, reducedMotion, motionPaused, animate, setTheme } = useMofangAppearance()
const { siteName, docUrl, entryPath, showModelPlaza, authStore } = useMofangBranding()
provide(mofangDarkKey, isDark)
</script>
