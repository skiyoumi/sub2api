import { describe, expect, it } from 'vitest'
import type { ChannelModelPricing } from '@/api/admin/channels'
import { emptyGroupPricing, groupPricingFromAPI, groupPricingToAPI } from '../groupsModelPricing'

function savedPricing(disabled?: boolean): ChannelModelPricing {
  return {
    platform: 'openai', models: ['gpt-5.4'], billing_mode: 'token',
    ...(disabled === undefined ? {} : { long_context_pricing_disabled: disabled }),
    input_price: 2.5e-6, output_price: 15e-6, cache_write_price: null,
    cache_write_1h_price: null, reasoning_effort_multipliers: null,
    cache_read_price: 0.25e-6, image_input_price: null, image_output_price: null,
    per_request_price: null, intervals: [], time_pricing: null,
  }
}

describe('group model long-context pricing', () => {
  it('keeps long-context behavior for new entries and saved entries without the override', () => {
    expect(emptyGroupPricing().long_context_pricing_disabled).toBe(false)
    expect(groupPricingFromAPI([savedPricing()])[0].long_context_pricing_disabled).toBe(false)
  })

  it('preserves a disabled model across loading, saving and reloading', () => {
    const original = savedPricing(true)
    const form = groupPricingFromAPI([original])
    expect(form[0].long_context_pricing_disabled).toBe(true)
    expect(form[0].input_price).toBe(2.5)
    const saved = groupPricingToAPI(form, 'openai')
    expect(saved).toEqual([original])
    expect(groupPricingFromAPI(saved)[0].long_context_pricing_disabled).toBe(true)
  })

  it('can restore inherited long-context pricing without changing custom rates', () => {
    const form = groupPricingFromAPI([savedPricing(true)])
    form[0].long_context_pricing_disabled = false
    expect(groupPricingToAPI(form, 'openai')).toEqual([savedPricing(false)])
  })

  it('allows disabling a wildcard model without entering any custom prices', () => {
    const form = { ...emptyGroupPricing(), models: ['gpt-5.4*'], long_context_pricing_disabled: true }
    const [saved] = groupPricingToAPI([form], 'openai')
    expect(saved.models).toEqual(['gpt-5.4*'])
    expect(saved.long_context_pricing_disabled).toBe(true)
    expect(saved.input_price).toBeNull()
    expect(saved.output_price).toBeNull()
    expect(saved.cache_read_price).toBeNull()
    expect(groupPricingFromAPI([saved])[0].long_context_pricing_disabled).toBe(true)
  })
})
