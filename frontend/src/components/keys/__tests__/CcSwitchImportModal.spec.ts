import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CcSwitchImportModal from '../CcSwitchImportModal.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: { count?: number }) => params?.count == null ? key : `${key}:${params.count}` }),
}))

const BaseDialogStub = {
  props: ['show'],
  emits: ['close'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
}

describe('CcSwitchImportModal', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: [{ id: 'gpt-5.5' }, { id: 'gpt-5-mini' }] }),
    }))
  })

  it('loads models from the public base URL and applies the group default', async () => {
    const wrapper = mount(CcSwitchImportModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://api.example.com/v1/',
        providerName: 'Example - OpenAI',
        platform: 'openai',
        defaults: { codex: 'gpt-5-mini' },
      },
      global: { stubs: { BaseDialog: BaseDialogStub } },
    })
    await flushPromises()

    expect(fetch).toHaveBeenCalledWith('https://api.example.com/v1/models', {
      headers: { Authorization: 'Bearer sk-test' },
    })
    expect((wrapper.find('select').element as HTMLSelectElement).value).toBe('gpt-5-mini')

    await wrapper.findAll('button').at(-1)!.trigger('click')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({ app: 'codex', model: 'gpt-5-mini' })
  })
})
