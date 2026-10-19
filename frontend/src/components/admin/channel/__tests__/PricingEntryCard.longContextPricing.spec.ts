import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PricingEntryCard from '../PricingEntryCard.vue'
import { emptyGroupPricing } from '@/views/admin/groupsModelPricing'

vi.mock('@/api/admin/channels', () => ({
  default: { getModelDefaultPricing: vi.fn() },
}))

vi.mock('vue-i18n', async importOriginal => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))

const selector = '[data-testid="disable-long-context-pricing"]'

describe('PricingEntryCard long-context pricing override', () => {
  it('shows the switch only for group token cards', () => {
    const hidden = shallowMount(PricingEntryCard, { props: { entry: emptyGroupPricing() } })
    expect(hidden.find(selector).exists()).toBe(false)
    const shown = shallowMount(PricingEntryCard, {
      props: { entry: emptyGroupPricing(), showLongContextPricingToggle: true },
    })
    expect(shown.find(selector).exists()).toBe(true)
    const requestCard = shallowMount(PricingEntryCard, {
      props: { entry: { ...emptyGroupPricing(), billing_mode: 'per_request' }, showLongContextPricingToggle: true },
    })
    expect(requestCard.find(selector).exists()).toBe(false)
  })

  it('emits the model override while preserving prices and model selection', async () => {
    const entry = { ...emptyGroupPricing(), models: ['gpt-5.4'], input_price: 2.5 }
    const wrapper = shallowMount(PricingEntryCard, { props: { entry, showLongContextPricingToggle: true } })
    await wrapper.find(selector).setValue(true)
    expect(wrapper.emitted('update')?.[0]?.[0]).toEqual({ ...entry, long_context_pricing_disabled: true })
    expect(entry.long_context_pricing_disabled).toBe(false)
  })

  it('loads the saved switch and allows enabling long-context pricing again', async () => {
    const entry = { ...emptyGroupPricing(), models: ['gpt-5.4'], long_context_pricing_disabled: true }
    const wrapper = shallowMount(PricingEntryCard, { props: { entry, showLongContextPricingToggle: true } })
    expect((wrapper.find(selector).element as HTMLInputElement).checked).toBe(true)
    await wrapper.find(selector).setValue(false)
    expect(wrapper.emitted('update')?.[0]?.[0]).toEqual({ ...entry, long_context_pricing_disabled: false })
  })
})
