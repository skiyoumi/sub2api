<template>
  <article class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm transition-shadow hover:shadow-md dark:border-dark-700 dark:bg-dark-800">
    <header class="flex items-start justify-between gap-3 px-4 pb-3 pt-4">
      <div class="flex min-w-0 items-center gap-2.5">
        <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gray-50 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-white dark:ring-dark-600">
          <PlatformIcon :platform="platform as GroupPlatform" size="lg" />
        </div>
        <div class="min-w-0">
          <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white" :title="model.name">{{ model.name }}</h3>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ platformLabel }} · {{ billingLabel }}</p>
        </div>
      </div>
      <span class="shrink-0 rounded-md bg-gray-100 px-2 py-1 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-dark-300">{{ groupName }}</span>
    </header>

    <div v-if="isToken" class="grid grid-cols-1 gap-2 px-4 pb-4 sm:grid-cols-3">
      <PriceMetric :label="t('modelPricing.input')" :current="paid(model.pricing?.input_price)" :official="official(model.official_pricing?.input_price)" />
      <PriceMetric :label="t('modelPricing.cacheInput')" :current="paid(model.pricing?.cache_read_price)" :official="official(model.official_pricing?.cache_read_price)" />
      <PriceMetric :label="t('modelPricing.output')" :current="paid(model.pricing?.output_price)" :official="official(model.official_pricing?.output_price)" />
    </div>
    <div v-else class="px-4 pb-4">
      <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <PriceMetric
          v-for="(price, index) in requestPrices"
          :key="index"
          :label="price.label"
          :current="price.value"
          compact
        />
      </div>
    </div>

    <footer class="flex items-center justify-between border-t border-gray-100 px-4 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
      <span>{{ t('modelPricing.effectiveRate') }}</span>
      <span class="font-mono font-medium">x{{ effectiveRate }}</span>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'
import { formatScaled } from '@/utils/pricing'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN } from '@/constants/channel'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

const props = defineProps<{
  model: PlazaModel
  groupName: string
  platform: string
  rateMultiplier: number
  userRateMultiplier?: number | null
}>()

const { t } = useI18n()
const effectiveRate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const isToken = computed(() => (props.model.pricing?.billing_mode || BILLING_MODE_TOKEN) === BILLING_MODE_TOKEN)
const platformLabel = computed(() => props.platform.charAt(0).toUpperCase() + props.platform.slice(1))
const billingLabel = computed(() => isToken.value ? t('modelPricing.perToken') : t('modelPricing.perRequest'))

function paid(value: number | null | undefined): number | undefined {
  return value == null ? undefined : value * effectiveRate.value * 1_000_000
}

function official(value: number | null | undefined): number | undefined {
  return value == null ? undefined : value * 1_000_000
}

const requestPrices = computed(() => {
  const intervals = props.model.pricing?.intervals ?? []
  const values: Array<{ label: string; value: number | undefined }> = intervals
    .filter(item => item.per_request_price != null)
    .slice(0, 3)
    .map((item, index) => ({
      label: item.tier_label || `${t('modelPricing.tier')} ${index + 1}`,
      value: (item.per_request_price ?? 0) * effectiveRate.value
    }))
  if (values.length === 0) {
    const basePrice = props.model.pricing?.per_request_price
    values.push({
      label: (props.model.pricing?.billing_mode === BILLING_MODE_IMAGE ? t('modelPricing.perImage') : t('modelPricing.perRequest')),
      value: basePrice == null ? undefined : basePrice * effectiveRate.value
    })
  }
  return values
})

const PriceMetric = defineComponent({
  props: {
    label: { type: String, required: true },
    current: { type: Number, default: null },
    official: { type: Number, default: null },
    compact: { type: Boolean, default: false }
  },
  setup(metric) {
    const comparison = computed(() => {
      if (metric.current == null || metric.official == null || metric.official <= 0) return null
      const percent = (metric.current / metric.official - 1) * 100
      return {
        premium: percent > 0,
        percent: Math.abs(percent)
      }
    })
    const money = (value: number | undefined) => value == null ? '-' : formatScaled(value, 1, 2)
    return () => h('div', { class: 'min-w-0 rounded-lg border border-gray-100 bg-gray-50/70 p-2 dark:border-dark-700 dark:bg-dark-900/40' }, [
      h('div', { class: 'mb-1.5 truncate text-xs text-gray-500 dark:text-dark-400', title: metric.label }, metric.label),
      h('div', { class: 'flex items-baseline justify-between gap-2 text-xs' }, [
        h('span', { class: 'text-gray-500 dark:text-dark-400' }, t('modelPricing.current')),
        h('strong', { class: 'whitespace-nowrap font-mono text-sm text-gray-900 dark:text-white' }, money(metric.current))
      ]),
      !metric.compact ? h('div', { class: 'mt-1 flex items-baseline justify-between gap-2 text-xs' }, [
        h('span', { class: 'text-gray-400' }, t('modelPricing.official')),
        h('span', { class: 'whitespace-nowrap font-mono text-gray-600 dark:text-dark-300' }, money(metric.official))
      ]) : null,
      comparison.value != null ? h('div', {
        class: comparison.value.premium
          ? 'mt-2 overflow-hidden rounded bg-red-50 text-[10px] dark:bg-red-950/30'
          : 'mt-2 overflow-hidden rounded bg-emerald-50 text-[10px] dark:bg-emerald-950/30'
      }, [
        h('div', {
          class: comparison.value.premium
            ? 'flex h-5 items-center whitespace-nowrap bg-gradient-to-r from-amber-500 to-red-500 px-2 text-white'
            : 'flex h-5 items-center whitespace-nowrap bg-gradient-to-r from-emerald-500 to-teal-500 px-2 text-white',
          style: { width: `${Math.max(48, Math.min(100, comparison.value.percent))}%` }
        }, `${comparison.value.premium ? t('modelPricing.premium') : t('modelPricing.save')} ${comparison.value.percent.toFixed(1)}%`)
      ]) : h('div', { class: 'mt-2 h-5 rounded bg-gray-100 dark:bg-dark-700' })
    ])
  }
})
</script>
