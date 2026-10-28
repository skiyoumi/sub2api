import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MofangLoginView from '../MofangLoginView.vue'

const mocks = vi.hoisted(() => ({
  settings: vi.fn(), login: vi.fn(), login2FA: vi.fn(), passkey: vi.fn(), push: vi.fn(),
  success: vi.fn(), error: vi.fn(), warning: vi.fn(), query: {} as Record<string, string>,
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push, currentRoute: { value: { query: mocks.query } } }) }))
vi.mock('vue-i18n', () => ({
  createI18n: () => ({ global: { t: (key: string) => key } }),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@/stores', () => ({
  useAuthStore: () => ({ login: mocks.login, login2FA: mocks.login2FA, loginWithPasskey: mocks.passkey }),
  useAppStore: () => ({ showSuccess: mocks.success, showError: mocks.error, showWarning: mocks.warning }),
}))
vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: unknown[]) => mocks.settings(...args),
  isTotp2FARequired: (response: { requires_2fa?: boolean }) => response?.requires_2fa === true,
  isWeChatWebOAuthEnabled: () => false,
  buildOAuthLoginStartURL: vi.fn(), startOAuthLogin: vi.fn(),
}))
vi.mock('@/utils/oauthAffiliate', () => ({ clearAllAffiliateReferralCodes: vi.fn() }))
const settings = {
  registration_enabled: true, turnstile_enabled: false, tencent_captcha_enabled: false,
  aliyun_captcha_enabled: false, backend_mode_enabled: false, password_reset_enabled: true,
  passkey_enabled: false, github_oauth_enabled: false, google_oauth_enabled: false,
  linuxdo_oauth_enabled: false, dingtalk_oauth_enabled: false, oidc_oauth_enabled: false,
  login_agreement_enabled: false, login_agreement_documents: [],
}
const agreement = defineComponent({
  name: 'LoginAgreementPrompt', emits: ['accept', 'reject', 'open'],
  template: '<button type="button" data-accept @click="$emit(\'accept\')">Accept</button>',
})
const totp = defineComponent({
  name: 'TotpLoginModal', props: ['tempToken', 'userEmailMasked'], emits: ['verify', 'cancel'],
  methods: { setVerifying() {}, setError() {} }, template: '<div data-totp />',
})
function createView() {
  return mount(MofangLoginView, { global: { stubs: {
    MofangShell: { template: '<div><slot /></div>' }, MofangLoginStory: true, Icon: true,
    RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
    LoginAgreementPrompt: agreement, TotpLoginModal: totp, TurnstileWidget: true,
    EmailOAuthButtons: true, LinuxDoOAuthSection: true, WechatOAuthSection: true,
    DingTalkOAuthSection: true, OidcOAuthSection: true,
  } } })
}
async function fillCredentials(wrapper: ReturnType<typeof createView>) {
  await wrapper.get('#email').setValue('developer@example.com')
  await wrapper.get('#password').setValue('correct-password')
}
describe('Mofang replacement login', () => {
  beforeEach(() => {
    vi.clearAllMocks(); localStorage.clear(); sessionStorage.clear()
    delete mocks.query.redirect
    mocks.settings.mockResolvedValue(settings)
    mocks.login.mockResolvedValue({ access_token: 'test-token' })
  })
  it('does not enable authentication until settings load', () => {
    mocks.settings.mockReturnValue(new Promise(() => {}))
    const wrapper = createView()
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('submits credentials to the existing store and honors the destination', async () => {
    mocks.query.redirect = '/keys'
    const wrapper = createView(); await flushPromises(); await fillCredentials(wrapper)
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(mocks.login).toHaveBeenCalledWith(expect.objectContaining({ email: 'developer@example.com', password: 'correct-password' }))
    expect(mocks.push).toHaveBeenCalledWith('/keys')
    expect(mocks.success).toHaveBeenCalledWith('auth.loginSuccess')
    wrapper.unmount()
  })
  it('keeps 2FA in the login flow and completes its redirect', async () => {
    mocks.login.mockResolvedValue({ requires_2fa: true, temp_token: 'pending-totp', user_email_masked: 'd***@example.com' })
    const wrapper = createView(); await flushPromises(); await fillCredentials(wrapper)
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(wrapper.getComponent(totp).props('tempToken')).toBe('pending-totp')
    expect(mocks.push).not.toHaveBeenCalled()
    wrapper.getComponent(totp).vm.$emit('verify', '123456'); await flushPromises()
    expect(mocks.login2FA).toHaveBeenCalledWith('pending-totp', '123456')
    expect(mocks.push).toHaveBeenCalledWith('/dashboard')
    wrapper.unmount()
  })
  it('blocks authentication until the current agreement is accepted', async () => {
    mocks.settings.mockResolvedValue({ ...settings, login_agreement_enabled: true, login_agreement_mode: 'checkbox', login_agreement_revision: 'revision-2', login_agreement_documents: [{ id: 'terms', title: 'Terms' }] })
    const wrapper = createView(); await flushPromises()
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-accept]').trigger('click'); await fillCredentials(wrapper)
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(mocks.login).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
  it('shows authentication failures and allows a retry', async () => {
    mocks.login.mockRejectedValue(new Error('Incorrect password'))
    const wrapper = createView(); await flushPromises(); await fillCredentials(wrapper)
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(mocks.error).toHaveBeenCalled()
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeUndefined()
    expect(mocks.push).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it.each([['closed registration', { registration_enabled: false }], ['backend mode', { backend_mode_enabled: true }]])('hides registration for %s', async (_name, overrides) => {
    mocks.settings.mockResolvedValue({ ...settings, ...overrides })
    const wrapper = createView(); await flushPromises()
    expect(wrapper.find('a[href="/register"]').exists()).toBe(false)
    wrapper.unmount()
  })
  it('keeps registration and password recovery when enabled', async () => {
    const wrapper = createView(); await flushPromises()
    expect(wrapper.find('a[href="/register"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/forgot-password"]').exists()).toBe(true)
    wrapper.unmount()
  })
  it('keeps third-party providers disabled in backend mode', async () => {
    mocks.settings.mockResolvedValue({ ...settings, backend_mode_enabled: true, github_oauth_enabled: true })
    const wrapper = createView(); await flushPromises()
    expect(wrapper.findComponent({ name: 'EmailOAuthButtons' }).exists()).toBe(false)
    wrapper.unmount()
  })
  it('requires the configured captcha proof before login', async () => {
    mocks.settings.mockResolvedValue({ ...settings, turnstile_enabled: true, turnstile_site_key: 'site-key' })
    const wrapper = createView(); await flushPromises(); await fillCredentials(wrapper)
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(mocks.login).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
