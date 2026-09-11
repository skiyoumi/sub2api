import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { CustomMenuItem } from '@/types'
import AppSidebar from '../AppSidebar.vue'

const { authStore, appStore, adminSettingsStore } = vi.hoisted(() => ({
  authStore: { isAdmin: false, isSimpleMode: false, user: { id: 7 }, token: 'private-token' },
  appStore: {
    sidebarCollapsed: false,
    mobileOpen: false,
    backendModeEnabled: false,
    sidebarScrollTop: 0,
    publicSettingsLoaded: true,
    siteName: 'Test',
    cachedPublicSettings: { custom_menu_items: [] as CustomMenuItem[] },
    setMobileOpen: vi.fn(),
  },
  adminSettingsStore: { customMenuItems: [] as CustomMenuItem[], fetch: vi.fn() },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
  useAdminSettingsStore: () => adminSettingsStore,
  useOnboardingStore: () => ({ isCurrentStep: () => false }),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/common/VersionBadge.vue', () => ({ default: { template: '<span />' } }))
vi.mock('@/utils/featureFlags', () => ({ FeatureFlags: {}, makeSidebarFlag: () => () => true }))
vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({ canUseBatchImage: { value: false }, refreshBatchImageAccess: vi.fn() }),
}))

const wrappers: ReturnType<typeof mount>[] = []
function menu(overrides: Partial<CustomMenuItem> = {}): CustomMenuItem {
  return {
    id: 'cards', label: 'Card recharge', icon_svg: '<svg></svg>',
    url: 'https://cards.example.com/shop?product=1#buy', visibility: 'user', sort_order: 0,
    ...overrides,
  }
}
async function mountSidebar() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  await router.push('/dashboard')
  await router.isReady()
  const wrapper = mount(AppSidebar, { global: { plugins: [router], stubs: { VersionBadge: true } } })
  wrappers.push(wrapper)
  await flushPromises()
  return { wrapper, router }
}

beforeEach(() => {
  authStore.isAdmin = false
  authStore.isSimpleMode = false
  appStore.cachedPublicSettings.custom_menu_items = []
  adminSettingsStore.customMenuItems = []
  localStorage.clear()
})
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

describe('custom menu opening', () => {
  it.each([undefined, 'iframe'] as const)('keeps %s mode in the current tab', async (open_mode) => {
    appStore.cachedPublicSettings.custom_menu_items = [menu({ open_mode })]
    const { wrapper, router } = await mountSidebar()
    const link = wrapper.get('a[href="/custom/cards"]')
    expect(link.attributes('target')).toBeUndefined()
    await link.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/custom/cards')
  })

  it.each([
    { label: 'regular user', isAdmin: false, isSimpleMode: false, visibility: 'user' as const },
    { label: 'admin personal menu', isAdmin: true, isSimpleMode: false, visibility: 'user' as const },
    { label: 'admin menu', isAdmin: true, isSimpleMode: false, visibility: 'admin' as const },
    { label: 'simple admin menu', isAdmin: true, isSimpleMode: true, visibility: 'admin' as const },
  ])('opens an external link directly for $label', async ({ isAdmin, isSimpleMode, visibility }) => {
    Object.assign(authStore, { isAdmin, isSimpleMode })
    const item = menu({ open_mode: 'new_tab', visibility })
    appStore.cachedPublicSettings.custom_menu_items = visibility === 'user' ? [item] : []
    adminSettingsStore.customMenuItems = [item]
    const { wrapper, router } = await mountSidebar()
    const link = wrapper.get<HTMLAnchorElement>(`a[href="${item.url}"]`)
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toBe('noopener noreferrer')
    expect(link.attributes('to')).toBeUndefined()
    expect(link.attributes('href')).not.toContain('private-token')
    link.element.addEventListener('click', event => event.preventDefault(), { once: true })
    await link.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/dashboard')
  })

  it('opens a Markdown menu through its authenticated page in a new tab', async () => {
    appStore.cachedPublicSettings.custom_menu_items = [menu({ url: 'md:guide', open_mode: 'new_tab' })]
    const { wrapper } = await mountSidebar()
    expect(wrapper.get('a[href="/custom/cards"]').attributes('target')).toBe('_blank')
  })

  it('does not render an executable URL as an external link', async () => {
    appStore.cachedPublicSettings.custom_menu_items = [menu({ url: 'javascript:alert(1)', open_mode: 'new_tab' })]
    const { wrapper } = await mountSidebar()
    expect(wrapper.find('a[href^="javascript:"]').exists()).toBe(false)
    expect(wrapper.get('a[href="/custom/cards"]').attributes('target')).toBeUndefined()
  })
})
