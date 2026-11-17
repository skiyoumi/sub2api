import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { ModelPricingConfig } from '@/api/modelPricing'
import ModelPricingManagementView from '@/views/admin/ModelPricingManagementView.vue'

const { getConfig, updateConfig } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn()
}))

vi.mock('@/api/modelPricing', () => ({
  getModelPricingConfig: getConfig,
  updateModelPricingConfig: updateConfig
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key })
}))

function makeGroup(id: number): ModelPlazaGroup {
  return {
    id, name: `Group ${id}`, description: '', platform: 'openai',
    subscription_type: 'standard', rate_multiplier: 1,
    peak_rate_enabled: false, peak_start: '', peak_end: '', peak_rate_multiplier: 1,
    is_exclusive: false, image_rate_independent: false, image_rate_multiplier: 1,
    video_rate_independent: false, video_rate_multiplier: 1,
    long_context_pricing_enabled: false,
    models: [{ name: `model-${id}`, platform: 'openai', pricing: null, official_pricing: null }]
  }
}

const groups = Array.from({ length: 8 }, (_, index) => makeGroup(index + 1))
const selectedGroups = () => groups.map(group => ({
  group_id: group.id, models: group.models.map(model => model.name)
}))

async function mountPage(config: ModelPricingConfig, availableGroups = groups) {
  getConfig.mockResolvedValue({ config, groups: availableGroups })
  const wrapper = mount(ModelPricingManagementView, {
    global: {
      stubs: {
        AppLayout: defineComponent({ template: '<main><slot /></main>' }),
        Icon: true, Toggle: true, LoadingSpinner: true, PlatformIcon: true
      }
    }
  })
  await flushPromises()
  return wrapper
}

describe('ModelPricingManagementView selected group count', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    updateConfig.mockImplementation(async (config: ModelPricingConfig) => config)
  })

  it('shows eight selected groups when eight checkboxes are checked', async () => {
    const wrapper = await mountPage({ enabled: true, description: '', groups: selectedGroups() })
    expect(wrapper.findAll('aside input:checked')).toHaveLength(8)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['8', '8'])
    wrapper.unmount()
  })

  it.each([
    { name: 'an unavailable saved group', extra: { group_id: 99, models: ['old-model'] } },
    { name: 'a duplicate saved group', extra: { group_id: 1, models: ['model-1'] } }
  ])('keeps the count at eight despite $name', async ({ extra }) => {
    const wrapper = await mountPage({ enabled: true, description: '', groups: [extra, ...selectedGroups()] })
    expect(wrapper.findAll('aside input:checked')).toHaveLength(8)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['8', '8'])
    expect(wrapper.find('h3').text()).toBe('Group 1')
    await wrapper.find('aside input[type="checkbox"]').setValue(false)
    expect(wrapper.findAll('aside input:checked')).toHaveLength(7)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['7', '7'])
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(updateConfig).toHaveBeenCalledWith({ enabled: true, description: '', groups: selectedGroups().slice(1) })
    wrapper.unmount()
  })

  it('keeps the total across group searches', async () => {
    const wrapper = await mountPage({ enabled: true, description: '', groups: selectedGroups() })
    await wrapper.find('aside input[type="text"], aside input:not([type])').setValue('Group 1')
    expect(wrapper.findAll('aside input:checked')).toHaveLength(1)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['8', '8'])
    wrapper.unmount()
  })

  it('merges distinct models from duplicate records and reconciles the save response', async () => {
    const group = makeGroup(1)
    group.models.push({ name: 'another-model', platform: 'openai', pricing: null, official_pricing: null })
    const config = {
      enabled: true, description: '', groups: [
        { group_id: 1, models: ['model-1'] },
        { group_id: 1, models: ['another-model', 'model-1'] }
      ]
    }
    const wrapper = await mountPage(config, [group])
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['1', '2'])
    updateConfig.mockResolvedValue({ ...config, groups: [...config.groups, { group_id: 99, models: ['old-model'] }] })
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(updateConfig).toHaveBeenCalledWith({
      enabled: true, description: '', groups: [{ group_id: 1, models: ['model-1', 'another-model'] }]
    })
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['1', '2'])
    wrapper.unmount()
  })

  it('updates the count when selecting eight groups and clearing the last selected model', async () => {
    const wrapper = await mountPage({ enabled: true, description: '', groups: [] })
    for (const checkbox of wrapper.findAll('aside input[type="checkbox"]')) await checkbox.setValue(true)
    expect(wrapper.findAll('aside input:checked')).toHaveLength(8)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['8', '8'])
    await wrapper.find('main label input[type="checkbox"]').setValue(false)
    expect(wrapper.findAll('aside input:checked')).toHaveLength(7)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['7', '7'])
    wrapper.unmount()
  })

  it('excludes empty groups and unavailable models from the selection', async () => {
    const emptyGroup = { ...makeGroup(9), models: [] }
    const wrapper = await mountPage({
      enabled: true, description: '',
      groups: [...selectedGroups(), { group_id: 9, models: [] }, { group_id: 1, models: ['removed-model'] }]
    }, [...groups, emptyGroup])
    expect(wrapper.findAll('aside input:checked')).toHaveLength(8)
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['8', '8'])
    const emptyCheckbox = wrapper.find('input[aria-label="Group 9"]')
    expect(emptyCheckbox.attributes('disabled')).toBeDefined()
    await wrapper.findAll('aside button')[8].trigger('click')
    await wrapper.findAll('main button').find(button => button.text() === 'admin.modelPricing.selectAllModels')!.trigger('click')
    expect(wrapper.findAll('strong').map(item => item.text())).toEqual(['8', '8'])
    wrapper.unmount()
  })
})
