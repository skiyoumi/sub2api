import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { CustomMenuItem, SidebarMenuOrder } from '@/types'
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
    cachedPublicSettings: { custom_menu_items: [] as CustomMenuItem[], sidebar_menu_order: {} as SidebarMenuOrder },
    setMobileOpen: vi.fn(),
  },
  adminSettingsStore: { customMenuItems: [] as CustomMenuItem[], sidebarMenuOrder: {} as SidebarMenuOrder, fetch: vi.fn() },
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
  appStore.cachedPublicSettings.sidebar_menu_order = {}
  adminSettingsStore.sidebarMenuOrder = {}
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


describe('mixed sidebar menu order', () => {
  it.each([false, true])('interleaves custom links with built-ins for user menus (admin=%s)', async (isAdmin) => {
    authStore.isAdmin = isAdmin
    const item = menu({ open_mode: 'new_tab' })
    appStore.cachedPublicSettings.custom_menu_items = [item, menu({ id: 'private', visibility: 'admin' })]
    appStore.cachedPublicSettings.sidebar_menu_order = {
      user: ['/purchase', '/custom/cards', '/orders', '/batch-image', '/custom/private', '/no-longer-exists'],
    }
    const { wrapper } = await mountSidebar()
    const section = wrapper.findAll('nav .sidebar-section')[isAdmin ? 1 : 0]
    const hrefs = section.findAll('a').map(link => link.attributes('href'))
    expect(hrefs.slice(0, 3)).toEqual(['/purchase', item.url, '/orders'])
    expect(hrefs).not.toContain('/batch-image')
    expect(hrefs).not.toContain('/custom/private')
    expect(hrefs).not.toContain('/no-longer-exists')
    expect(hrefs).toContain('/profile')
    expect(hrefs.includes('/dashboard')).toBe(!isAdmin)
  })

  it.each([false, true])('orders admin menus while respecting simple mode (%s)', async (isSimpleMode) => {
    Object.assign(authStore, { isAdmin: true, isSimpleMode })
    adminSettingsStore.customMenuItems = [menu({ visibility: 'admin' })]
    adminSettingsStore.sidebarMenuOrder = {
      admin: ['/admin/settings', '/custom/cards', '/admin/users', '/keys'],
      user: ['/profile'],
    }
    const { wrapper } = await mountSidebar()
    const hrefs = wrapper.findAll('nav .sidebar-section')[0].findAll('a').map(link => link.attributes('href'))
    expect(hrefs.slice(0, 2)).toEqual(['/admin/settings', '/custom/cards'])
    expect(hrefs.includes('/admin/users')).toBe(!isSimpleMode)
    expect(hrefs.includes('/keys')).toBe(isSimpleMode)
    expect(hrefs).not.toContain('/profile')
    expect(hrefs).toContain('/admin/dashboard')
  })

  it('preserves the legacy default order when no mixed order is configured', async () => {
    appStore.cachedPublicSettings.custom_menu_items = [menu({ id: 'later', sort_order: 2 }), menu({ sort_order: 1 })]
    const { wrapper } = await mountSidebar()
    const hrefs = wrapper.findAll('nav a').map(link => link.attributes('href'))
    expect(hrefs[0]).toBe('/dashboard')
    expect(hrefs.slice(-3)).toEqual(['/profile', '/custom/cards', '/custom/later'])
  })
})
