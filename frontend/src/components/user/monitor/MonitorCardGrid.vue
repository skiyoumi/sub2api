<template>
  <div>
    <div
      v-if="loading && items.length === 0"
      class="grid gap-5 grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      <div
        v-for="i in 6"
        :key="i"
        class="p-5 rounded-2xl min-h-[280px] bg-white/70 dark:bg-dark-800/60 border border-gray-200/80 dark:border-dark-700/70 animate-pulse"
      >
        <div class="flex items-start gap-3">
          <div class="w-9 h-9 rounded-xl bg-gray-200 dark:bg-dark-700"></div>
          <div class="flex-1 space-y-2">
            <div class="h-4 w-2/3 rounded bg-gray-200 dark:bg-dark-700"></div>
            <div class="h-3 w-1/2 rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
          <div class="h-6 w-16 rounded-full bg-gray-200 dark:bg-dark-700"></div>
        </div>
        <div class="mt-5 grid grid-cols-2 gap-2">
          <div class="h-16 rounded-xl bg-gray-100 dark:bg-dark-900/40"></div>
          <div class="h-16 rounded-xl bg-gray-100 dark:bg-dark-900/40"></div>
        </div>
        <div class="mt-6 h-5 w-full rounded bg-gray-100 dark:bg-dark-900/40"></div>
      </div>
    </div>

    <EmptyState
      v-else-if="items.length === 0"
      :title="t('channelStatus.empty.title')"
      :description="t('channelStatus.empty.description')"
    />

    <!-- Group cards by platform so users can scan status per provider. -->
    <div v-else class="space-y-10">
      <section v-for="group in platformGroups" :key="group.provider">
        <div class="mb-4 flex items-center gap-2.5">
          <span
            class="grid h-8 w-8 place-items-center rounded-xl ring-1 ring-black/5 dark:ring-white/10"
            :class="providerBadgeClass(group.provider)"
          >
            <ProviderIcon :provider="group.provider" :size="16" />
          </span>
          <h2
            class="text-sm font-bold uppercase tracking-wider text-gray-700 dark:text-gray-200"
          >
            {{ providerLabel(group.provider) }}
          </h2>
          <span
            class="rounded-full px-2 py-0.5 text-[11px] font-semibold tabular-nums"
            :class="providerBadgeClass(group.provider)"
          >
            {{ group.items.length }}
          </span>
        </div>

        <div
          class="grid gap-5 grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
        >
          <MonitorCard
            v-for="item in group.items"
            :key="item.id"
            :item="item"
            :window="window"
            :availability-value="resolveAvailability(item)"
            :countdown-seconds="countdownSeconds"
            @click="emit('cardClick', item)"
          />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Provider } from '@/api/admin/channelMonitor'
import type { UserMonitorView, UserMonitorDetail } from '@/api/channelMonitor'
import { PROVIDERS } from '@/constants/channelMonitor'
import EmptyState from '@/components/common/EmptyState.vue'
import MonitorCard from './MonitorCard.vue'
import ProviderIcon from './ProviderIcon.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const props = defineProps<{
  items: UserMonitorView[]
  window: '7d' | '15d' | '30d'
  countdownSeconds: number
  loading: boolean
  detailCache: Record<number, UserMonitorDetail>
}>()

const emit = defineEmits<{
  (e: 'cardClick', item: UserMonitorView): void
}>()

const { t } = useI18n()
const { providerLabel, providerBadgeClass } = useChannelMonitorFormat()

interface PlatformGroup {
  provider: UserMonitorView['provider']
  items: UserMonitorView[]
}

/** Group monitor cards by platform, ordered by the canonical provider catalog. */
const platformGroups = computed<PlatformGroup[]>(() => {
  const byProvider = new Map<UserMonitorView['provider'], UserMonitorView[]>()
  for (const item of props.items) {
    const bucket = byProvider.get(item.provider)
    if (bucket) bucket.push(item)
    else byProvider.set(item.provider, [item])
  }
  return [...byProvider.entries()]
    .sort(([a], [b]) => {
      const ia = PROVIDERS.indexOf(a as Provider)
      const ib = PROVIDERS.indexOf(b as Provider)
      return (ia === -1 ? Number.MAX_SAFE_INTEGER : ia) - (ib === -1 ? Number.MAX_SAFE_INTEGER : ib)
    })
    .map(([provider, items]) => ({ provider, items }))
})

function resolveAvailability(item: UserMonitorView): number | null {
  if (props.window === '7d') {
    return item.availability_7d ?? null
  }
  const detail = props.detailCache[item.id]
  if (!detail) return null
  const primary = detail.models.find(m => m.model === item.primary_model)
  if (!primary) return null
  return props.window === '15d' ? primary.availability_15d ?? null : primary.availability_30d ?? null
}
</script>
