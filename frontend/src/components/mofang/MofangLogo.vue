<template>
  <img class="mf-logo" :src="source" :alt="siteName" decoding="async" draggable="false" />
</template>
<script setup lang="ts">
import { computed, inject } from 'vue'
import { useMofangBranding } from '@/composables/useMofangBranding'
import { mofangDarkKey } from './mofangContext'
const { siteName, customLogo } = useMofangBranding()
const dark = inject(mofangDarkKey, computed(() => false))
const source = computed(() => customLogo.value || (dark.value
  ? '/brand/mofang-glass-dark.svg' : '/brand/mofang-glass-light.svg'))
</script>
