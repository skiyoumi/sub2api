import { computed, onMounted, watch } from 'vue'
import { useAdminSettingsStore, useAppStore, useAuthStore } from '@/stores'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import { getVisibleSidebarMenuEntries, isSidebarSectionVisible, type SidebarMenuContext, type SidebarSection } from '@/utils/sidebarMenus'
import { orderSidebarItems } from '@/utils/sidebarMenuOrder'
import type { CustomMenuItem } from '@/types'

export function useSidebarMenus() {
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
  const flags = Object.fromEntries(
    Object.entries(FeatureFlags).map(([name, flag]) => [name, makeSidebarFlag(flag)]),
  )
  const context = computed<SidebarMenuContext>(() => ({
    isAdmin: authStore.isAdmin,
    isSimpleMode: authStore.isSimpleMode,
    backendModeEnabled: appStore.backendModeEnabled,
    features: {
      ...Object.fromEntries(Object.entries(flags).map(([key, enabled]) => [key, enabled()])),
      opsMonitoring: adminSettingsStore.opsMonitoringEnabled,
      adminPayment: adminSettingsStore.paymentEnabled,
      batchImage: canUseBatchImage.value,
    },
  }))
  const sections = computed(() => (['user', 'admin'] as const).filter(section => isSidebarSectionVisible(section, context.value)))
  function getEntries(section: SidebarSection, customItems: readonly CustomMenuItem[], order?: readonly string[]) {
    return orderSidebarItems(getVisibleSidebarMenuEntries(section, customItems, context.value), order)
  }
  onMounted(() => { void refreshBatchImageAccess() })
  watch(() => authStore.isAdmin, (isAdmin) => {
    if (isAdmin) void adminSettingsStore.fetch()
  }, { immediate: true })
  return { sections, getEntries }
}
