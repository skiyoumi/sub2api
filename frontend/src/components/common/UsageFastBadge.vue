<template>
  <span
    v-if="label"
    data-testid="usage-fast-badge"
    :title="`${t('usage.serviceTier')}: ${label}`"
    class="inline-flex shrink-0 items-center gap-0.5 whitespace-nowrap rounded bg-emerald-100 px-1 py-px align-middle text-[10px] font-semibold leading-tight text-emerald-700 ring-1 ring-inset ring-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-300 dark:ring-emerald-500/30"
  >
    <Icon name="bolt" size="xs" aria-hidden="true" />
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getUsageServiceTierLabel, normalizeUsageServiceTier } from '@/utils/usageServiceTier'

const props = defineProps<{
  serviceTier?: string | null
}>()

const { t } = useI18n()
const label = computed(() => {
  const tier = normalizeUsageServiceTier(props.serviceTier)
  return tier === 'priority' || tier === 'ultrafast'
    ? getUsageServiceTierLabel(tier, t)
    : null
})
</script>
