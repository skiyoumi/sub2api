<template>
  <div class="mf-model-list" :class="{ 'mf-model-list--compact': compact }" :aria-label="t('mofang.models')">
    <span v-for="model in models" :key="model.name" class="mf-provider" :class="'mf-provider--' + model.kind">
      <i aria-hidden="true">{{ model.symbol }}</i>{{ model.name }}
    </span>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ compact?: boolean }>()
const { t } = useI18n()
const all = [
  { name: 'Claude', symbol: '✳', kind: 'claude' }, { name: 'GPT', symbol: '◎', kind: 'gpt' },
  { name: 'Gemini', symbol: '✦', kind: 'gemini' }, { name: 'Antigravity', symbol: '⌁', kind: 'antigravity' },
]
const models = computed(() => props.compact ? all.slice(0, 3) : all)
</script>
