import { describe, expect, it } from 'vitest'
import { buildCcSwitchImportDeeplink, withoutV1Endpoint, type CcSwitchApp } from '@/utils/ccswitchImport'

function paramsFromDeeplink(deeplink: string): URLSearchParams {
  return new URLSearchParams(deeplink.split('?')[1] || '')
}

const baseInput = {
  baseUrl: 'https://api.example.com/v1',
  providerName: 'Sub2API',
  apiKey: 'sk-test',
  usageScript: 'return true',
  model: 'model-main',
}

describe('ccswitchImport utils', () => {
  it.each([
    ['https://api.example.com/v1', 'https://api.example.com'],
    ['https://api.example.com/v1/', 'https://api.example.com'],
    ['https://api.example.com/', 'https://api.example.com'],
  ])('removes a trailing v1 from Claude base URL %s', (input, expected) => {
    expect(withoutV1Endpoint(input)).toBe(expected)
  })

  it.each(['claude', 'codex', 'gemini', 'opencode'] as CcSwitchApp[])('imports the selected %s application and model', (app) => {
    const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({ ...baseInput, app }))
    expect(params.get('app')).toBe(app)
    expect(params.get('model')).toBe(baseInput.model)
    expect(params.get('endpoint')).toBe(app === 'claude' ? 'https://api.example.com' : baseInput.baseUrl)
    expect(atob(params.get('usageScript') || '')).toBe(baseInput.usageScript)
  })

  it('adds Claude family model parameters only for Claude imports', () => {
    const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({
      ...baseInput,
      app: 'claude',
      haikuModel: 'haiku',
      sonnetModel: 'sonnet',
      opusModel: 'opus',
    }))
    expect(params.get('haikuModel')).toBe('haiku')
    expect(params.get('sonnetModel')).toBe('sonnet')
    expect(params.get('opusModel')).toBe('opus')
  })

  it('supports a platform-specific endpoint without changing the provider homepage', () => {
    const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({
      ...baseInput,
      app: 'claude',
      endpointBaseUrl: 'https://api.example.com/antigravity',
    }))
    expect(params.get('homepage')).toBe(baseInput.baseUrl)
    expect(params.get('endpoint')).toBe('https://api.example.com/antigravity')
  })
})
