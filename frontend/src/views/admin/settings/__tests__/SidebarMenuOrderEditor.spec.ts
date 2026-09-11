import { afterEach, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { VueDraggable } from 'vue-draggable-plus'
import type { SidebarMenuOrder } from '@/types'
import SidebarMenuOrderEditor from '../SidebarMenuOrderEditor.vue'

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAdmin: true, isSimpleMode: false }),
  useAppStore: () => ({ backendModeEnabled: false }),
  useAdminSettingsStore: () => ({ fetch: vi.fn() }),
}))
vi.mock('@/utils/featureFlags', () => ({ FeatureFlags: {}, makeSidebarFlag: () => () => true }))
vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({ canUseBatchImage: { value: false }, refreshBatchImageAccess: vi.fn() }),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const wrappers: ReturnType<typeof mount>[] = []
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

it('applies drag results to the selected section and preserves the other section', async () => {
  const wrapper = mount(SidebarMenuOrderEditor, { props: {
    modelValue: { user: ['/profile'], admin: ['/admin/settings'] },
    customItems: [{ id: 'cards', label: 'Card recharge', url: 'https://example.com', icon_svg: '', visibility: 'user', sort_order: 0 }],
    'onUpdate:modelValue': (order: SidebarMenuOrder) => { void wrapper.setProps({ modelValue: order }) },
  } })
  wrappers.push(wrapper)
  const draggable = wrapper.findComponent(VueDraggable)
  const items = [...draggable.props('modelValue')]
  const custom = items.pop()
  items.splice(1, 0, custom)
  draggable.vm.$emit('update:modelValue', items)
  await wrapper.vm.$nextTick()
  expect(wrapper.findAll('[data-menu-path]').slice(0, 2).map(row => row.attributes('data-menu-path'))).toEqual(['/profile', '/custom/cards'])
  expect(wrapper.props('modelValue').admin).toEqual(['/admin/settings'])
  expect(wrapper.get('[data-menu-path="/profile"] [data-direction="up"]').attributes('disabled')).toBeDefined()
  const savedUserOrder = [...wrapper.props('modelValue').user ?? []]
  await wrapper.get('[data-sidebar-section="admin"]').trigger('click')
  expect(wrapper.find('[data-menu-path="/custom/cards"]').exists()).toBe(false)
  expect(wrapper.findAll('[data-menu-path]')[0].attributes('data-menu-path')).toBe('/admin/settings')
  await wrapper.get('[data-reset-menu-order]').trigger('click')
  expect(wrapper.props('modelValue').admin).toEqual([])
  expect(wrapper.props('modelValue').user).toEqual(savedUserOrder)
})
