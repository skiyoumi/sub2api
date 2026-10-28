import { computed } from 'vue'
import { useAppStore, useAuthStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

export function useMofangBranding() {
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const settings = computed(() => appStore.cachedPublicSettings)
  const siteName = computed(() => {
    const name = settings.value?.site_name || appStore.siteName
    return name && name !== 'Sub2API' ? name : '模方'
  })
  const customLogo = computed(() => {
    const value = sanitizeUrl(settings.value?.site_logo || appStore.siteLogo || '', {
      allowRelative: true, allowDataUrl: true,
    })
    return value === '/logo.svg' ? '' : value
  })
  const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
  const consolePath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
  const entryPath = computed(() => authStore.isAuthenticated ? consolePath.value : '/login')
  const showModelPlaza = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza)
    && (authStore.isAuthenticated || settings.value?.model_plaza_require_auth !== true))
  return { siteName, customLogo, docUrl, consolePath, entryPath, showModelPlaza, authStore, appStore, settings }
}
