import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, h, reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { CustomMenuItem, SidebarMenuOrder } from '@/types'
import type { SidebarSection } from '@/utils/sidebarMenus'
import AppSidebar from '../AppSidebar.vue'
import SidebarMenuOrderEditor from '@/views/admin/settings/SidebarMenuOrderEditor.vue'

const authStore = reactive({ isAdmin: true, isSimpleMode: false })
const appStore = reactive({
  sidebarCollapsed: false, mobileOpen: false, backendModeEnabled: false,
  sidebarScrollTop: 0, publicSettingsLoaded: true, siteName: 'Test', setMobileOpen: vi.fn(),
  cachedPublicSettings: {
    custom_menu_items: [] as CustomMenuItem[], sidebar_menu_order: {} as SidebarMenuOrder,
    channel_monitor_enabled: true, payment_enabled: true, available_channels_enabled: false,
    subscription_enabled: true, payment_balance_disabled: false,
    affiliate_enabled: false, risk_control_enabled: false, plugin_management_enabled: false,
  },
})
const adminSettingsStore = reactive({
  customMenuItems: [] as CustomMenuItem[], sidebarMenuOrder: {} as SidebarMenuOrder,
  opsMonitoringEnabled: false, paymentEnabled: false, fetch: vi.fn(),
})
const batchAccess = ref(false)
const editorOrder = ref<SidebarMenuOrder>({})
vi.mock('@/stores', () => ({
  useAppStore: () => appStore, useAuthStore: () => authStore,
  useAdminSettingsStore: () => adminSettingsStore,
  useOnboardingStore: () => ({ isCurrentStep: () => false }),
}))
// Use the real feature registry and its opt-in/opt-out behavior.
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({ canUseBatchImage: batchAccess, refreshBatchImageAccess: vi.fn() }),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/common/VersionBadge.vue', () => ({ default: { template: '<span />' } }))

const wrappers: ReturnType<typeof mount>[] = []
beforeEach(() => {
  Object.assign(authStore, { isAdmin: true, isSimpleMode: false })
  appStore.backendModeEnabled = false
  appStore.cachedPublicSettings = {
    custom_menu_items: [{ id: 'cards', label: 'Custom user link', url: 'https://example.com', icon_svg: '', visibility: 'user', sort_order: 0 }],
    sidebar_menu_order: {}, channel_monitor_enabled: true, payment_enabled: true,
    subscription_enabled: true, payment_balance_disabled: false,
    available_channels_enabled: false, affiliate_enabled: false,
    risk_control_enabled: false, plugin_management_enabled: false,
  }
  Object.assign(adminSettingsStore, {
    customMenuItems: [{ id: 'private', label: 'Custom admin link', url: 'https://example.com/admin', icon_svg: '', visibility: 'admin', sort_order: 0 }],
    sidebarMenuOrder: {}, opsMonitoringEnabled: false, paymentEnabled: false,
  })
  batchAccess.value = false
  editorOrder.value = {}
  localStorage.clear()
})
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

async function mountMenus() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }] })
  await router.push('/admin/settings')
  await router.isReady()
  const wrapper = mount(defineComponent({ setup() {
    const customItems = computed(() => [...appStore.cachedPublicSettings.custom_menu_items, ...adminSettingsStore.customMenuItems])
    return () => h('div', [
      h(AppSidebar),
      h(SidebarMenuOrderEditor, {
        modelValue: editorOrder.value, customItems: customItems.value,
        'onUpdate:modelValue': (order: SidebarMenuOrder) => { editorOrder.value = order },
      }),
    ])
  } }), { global: { plugins: [router] } })
  wrappers.push(wrapper)
  await flushPromises()
  return { sidebar: wrapper.getComponent(AppSidebar), editor: wrapper.getComponent(SidebarMenuOrderEditor) }
}
function expectSameMenus(menus: Awaited<ReturnType<typeof mountMenus>>, section: SidebarSection) {
  const sidebarSection = menus.sidebar.findAll('nav .sidebar-section')[authStore.isAdmin && section === 'user' ? 1 : 0]
  const rendered = Array.from(sidebarSection.element.children)
    .filter(element => element.classList.contains('sidebar-link'))
    .map(element => element.querySelector('.sidebar-label .truncate')?.textContent?.trim())
  const editable = menus.editor.findAll('[data-menu-path]').map(row => row.get('.min-w-0').text())
  expect(editable).toEqual(rendered)
}

describe('sidebar and menu editor share actual menu visibility', () => {
  it.each([
    { subscription: true, balanceDisabled: false, label: 'nav.buySubscription' },
    { subscription: false, balanceDisabled: false, label: 'nav.recharge' },
    { subscription: true, balanceDisabled: true, label: 'nav.subscribe' },
  ])('updates both menus for billing mode $label', async ({ subscription, balanceDisabled, label }) => {
    const menus = await mountMenus()
    Object.assign(appStore.cachedPublicSettings, {
      subscription_enabled: subscription, payment_balance_disabled: balanceDisabled,
    })
    await flushPromises()
    expect(menus.sidebar.find('a[href="/subscriptions"]').exists()).toBe(subscription)
    expect(menus.sidebar.find('a[href="/admin/subscriptions"]').exists()).toBe(subscription)
    expect(menus.sidebar.get('a[href="/purchase"] .sidebar-label .truncate').text()).toBe(label)
    expect(menus.editor.find('[data-menu-path="/subscriptions"]').exists()).toBe(subscription)
    expectSameMenus(menus, 'user')
    await menus.editor.get('[data-sidebar-section="admin"]').trigger('click')
    expect(menus.editor.find('[data-menu-path="/admin/subscriptions"]').exists()).toBe(subscription)
    expectSameMenus(menus, 'admin')
  })

  it.each([
    { name: 'standard admin', isAdmin: true, isSimpleMode: false, backendMode: false },
    { name: 'simple admin', isAdmin: true, isSimpleMode: true, backendMode: false },
    { name: 'standard user', isAdmin: false, isSimpleMode: false, backendMode: false },
    { name: 'simple user', isAdmin: false, isSimpleMode: true, backendMode: false },
    { name: 'admin in backend-only mode', isAdmin: true, isSimpleMode: false, backendMode: true },
    { name: 'user in backend-only mode', isAdmin: false, isSimpleMode: false, backendMode: true },
  ])('matches every visible section for $name', async ({ isAdmin, isSimpleMode, backendMode }) => {
    Object.assign(authStore, { isAdmin, isSimpleMode })
    appStore.backendModeEnabled = backendMode
    const menus = await mountMenus()
    const sections = menus.editor.findAll('[data-sidebar-section]')
    expect(sections.length).toBe(isAdmin ? (isSimpleMode ? 1 : 2) : (backendMode ? 0 : 1))
    for (const button of sections) {
      await button.trigger('click')
      const section = button.attributes('data-sidebar-section') === 'admin' ? 'admin' : 'user'
      expectSameMenus(menus, section)
      const paths = menus.editor.findAll('[data-menu-path]').map(row => row.attributes('data-menu-path'))
      expect(paths).not.toContain('/batch-image')
      expect(paths).not.toContain('/available-channels')
      expect(paths).not.toContain('/admin/ops')
      expect(paths).not.toContain('/admin/plugins')
      expect(paths).not.toContain('/admin/orders')
      if (isAdmin) expect(paths).not.toContain('/dashboard')
      if (section === 'user') expect(paths).not.toContain('/custom/private')
      if (section === 'admin') expect(paths.includes('/keys')).toBe(isSimpleMode)
    }
    if (!sections.length) {
      expect(menus.editor.find('[data-menu-path]').exists()).toBe(false)
      expect(menus.sidebar.find('nav .sidebar-section').exists()).toBe(false)
    }
  })

  it('reacts to feature flags, per-account access and mode changes in both views', async () => {
    const menus = await mountMenus()
    Object.assign(appStore.cachedPublicSettings, {
      available_channels_enabled: true, channel_monitor_enabled: false, payment_enabled: false,
      affiliate_enabled: true, risk_control_enabled: true, plugin_management_enabled: true,
    })
    batchAccess.value = true
    Object.assign(adminSettingsStore, { opsMonitoringEnabled: true, paymentEnabled: true })
    await flushPromises()
    expectSameMenus(menus, 'user')
    expect(menus.editor.find('[data-menu-path="/batch-image"]').exists()).toBe(true)
    expect(menus.editor.find('[data-menu-path="/available-channels"]').exists()).toBe(true)
    expect(menus.editor.find('[data-menu-path="/monitor"]').exists()).toBe(false)
    expect(menus.editor.find('[data-menu-path="/purchase"]').exists()).toBe(false)
    await menus.editor.get('[data-sidebar-section="admin"]').trigger('click')
    expectSameMenus(menus, 'admin')
    expect(menus.editor.find('[data-menu-path="/admin/ops"]').exists()).toBe(true)
    expect(menus.editor.find('[data-menu-path="/admin/orders"]').exists()).toBe(true)
    const channels = menus.sidebar.findAll('nav button').find(button => button.text() === 'nav.channelManagement')!
    await channels.trigger('click')
    expect(menus.sidebar.find('a[href="/admin/channels/pricing"]').exists()).toBe(true)
    expect(menus.sidebar.find('a[href="/admin/channels/monitor"]').exists()).toBe(false)
    appStore.cachedPublicSettings.channel_monitor_enabled = true
    await flushPromises()
    expect(menus.sidebar.find('a[href="/admin/channels/monitor"]').exists()).toBe(true)
    await menus.editor.get('[data-sidebar-section="user"]').trigger('click')
    authStore.isSimpleMode = true
    await flushPromises()
    expect(menus.editor.find('[data-sidebar-section="user"]').exists()).toBe(false)
    expectSameMenus(menus, 'admin')
  })

  it('preserves hidden positions across a visible reorder, save and re-enable', async () => {
    editorOrder.value = { user: ['/profile', '/available-channels', '/custom/cards', '/keys'], admin: ['/admin/settings'] }
    appStore.cachedPublicSettings.sidebar_menu_order = editorOrder.value
    const menus = await mountMenus()
    await menus.editor.get('[data-menu-path="/custom/cards"] [data-direction="up"]').trigger('click')
    expect(editorOrder.value.user?.slice(0, 4)).toEqual(['/custom/cards', '/available-channels', '/profile', '/keys'])
    expect(editorOrder.value.admin).toEqual(['/admin/settings'])
    appStore.cachedPublicSettings.sidebar_menu_order = editorOrder.value
    await flushPromises()
    expectSameMenus(menus, 'user')
    appStore.cachedPublicSettings.available_channels_enabled = true
    await flushPromises()
    expectSameMenus(menus, 'user')
    expect(menus.editor.findAll('[data-menu-path]').slice(0, 4).map(row => row.attributes('data-menu-path'))).toEqual(['/custom/cards', '/available-channels', '/profile', '/keys'])
  })
})
