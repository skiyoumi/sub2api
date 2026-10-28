import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import MofangHomeView from '../MofangHomeView.vue'

const { appStore, authStore } = vi.hoisted(() => ({
  appStore: { cachedPublicSettings: {} as Record<string, unknown>, siteName: 'Sub2API', siteLogo: '', docUrl: '', publicSettingsLoaded: true, fetchPublicSettings: vi.fn() },
  authStore: { isAuthenticated: false, isAdmin: false },
}))
vi.mock('@/stores', () => ({ useAppStore: () => appStore, useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('vue-i18n', async importOriginal => ({
  ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }),
}))
function createView(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = settings
  return mount(MofangHomeView, { global: { stubs: {
    RouterLink: RouterLinkStub, Icon: true, MofangShell: { template: '<div><slot /></div>' }, MofangScene: true, MofangModels: true,
  } } })
}
describe('Mofang replacement home configuration', () => {
  beforeEach(() => { authStore.isAuthenticated = false; authStore.isAdmin = false; appStore.fetchPublicSettings.mockClear() })
  it('preserves administrator-provided custom home HTML', () => {
    const view = createView({ home_content: '<section id="configured-home">Custom home</section>' })
    expect(view.get('#configured-home').text()).toBe('Custom home')
    expect(view.find('#mofang-home-title').exists()).toBe(false)
  })
  it('preserves custom home URLs', () => {
    const view = createView({ home_content: ' https://example.com/home ' })
    expect(view.get('iframe').attributes('src')).toBe('https://example.com/home')
  })
  it('honors compact mode without bringing back the old page', () => {
    const view = createView({ compact_home_enabled: true })
    expect(view.find('#mofang-home-title').exists()).toBe(true)
    expect(view.find('.mf-feature-grid').exists()).toBe(false)
  })
  it.each([
    [false, false, '/login'], [true, false, '/dashboard'], [true, true, '/admin/dashboard'],
  ])('sends authenticated=%s admin=%s to %s', (authenticated, admin, expected) => {
    authStore.isAuthenticated = authenticated; authStore.isAdmin = admin
    const view = createView()
    expect(view.findAllComponents(RouterLinkStub).some(link => link.props('to') === expected)).toBe(true)
  })
  it('only exposes the model plaza when the public settings allow it', () => {
    const hidden = createView({ model_plaza_enabled: true, model_plaza_require_auth: true })
    expect(hidden.findAllComponents(RouterLinkStub).some(link => link.props('to') === '/model-plaza')).toBe(false)
    const visible = createView({ model_plaza_enabled: true, model_plaza_require_auth: false })
    expect(visible.findAllComponents(RouterLinkStub).some(link => link.props('to') === '/model-plaza')).toBe(true)
  })
  it('uses a configured documentation URL and rejects unsafe URLs', () => {
    const visible = createView({ doc_url: 'https://docs.example.com' })
    expect(visible.get('a[href="https://docs.example.com/"]').attributes('rel')).toBe('noopener noreferrer')
    const hidden = createView({ doc_url: 'javascript:alert(1)' })
    expect(hidden.find('a[target="_blank"]').exists()).toBe(false)
  })
})
