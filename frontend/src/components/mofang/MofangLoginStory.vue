<template>
  <section class="mf-login-story">
    <div class="mf-story-kicker">THE NEXT CONNECTION STARTS HERE</div>
    <h1>{{ t('mofang.storyFirst') }}<br /><span class="mf-gradient-text">{{ t('mofang.storySecond') }}</span></h1>
    <p>{{ t('mofang.storyText') }}</p>
    <MofangScene />
    <div class="mf-story-models"><span class="mf-model-caption">{{ t('mofang.manyModels') }}</span><MofangModels compact /></div>
  </section>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import MofangScene from './MofangScene.vue'
import MofangModels from './MofangModels.vue'
const { t } = useI18n()
</script>
