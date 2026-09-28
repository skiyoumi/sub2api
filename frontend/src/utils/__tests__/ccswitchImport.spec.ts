import { describe, expect, it } from 'vitest'
import {
  CC_SWITCH_USAGE_SCRIPT,
  buildCcSwitchImportDeeplink,
  withoutV1Endpoint,
  type CcSwitchApp
} from '@/utils/ccswitchImport'

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

  it.each(['claude', 'codex', 'gemini', 'opencode', 'grokbuild'] as CcSwitchApp[])('imports the selected %s application and model', (app) => {
    const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({ ...baseInput, app }))
    expect(params.get('app')).toBe(app)
    expect(params.get('model')).toBe(baseInput.model)
    expect(params.get('endpoint')).toBe(app === 'claude' ? 'https://api.example.com' : baseInput.baseUrl)
    expect(atob(params.get('usageScript') || '')).toBe(baseInput.usageScript)
  })

  it('keeps Codex imports on the configured endpoint', () => {
    for (const [baseUrl, endpoint] of [
      ['https://api.example.com', 'https://api.example.com'],
      ['https://api.example.com/', 'https://api.example.com'],
      ['https://api.example.com/v1', 'https://api.example.com/v1'],
      ['https://api.example.com/v1/', 'https://api.example.com/v1']
    ]) {
      const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({ ...baseInput, baseUrl, app: 'codex' }))
      expect(params.get('endpoint')).toBe(endpoint)
    }
  })

  it('imports Grok Build with exactly one /v1 suffix', () => {
    for (const baseUrl of ['https://api.example.com', 'https://api.example.com/', 'https://api.example.com/v1', 'https://api.example.com/v1/']) {
      const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({ ...baseInput, baseUrl, app: 'grokbuild' }))
      expect(params.get('endpoint')).toBe('https://api.example.com/v1')
    }
  })

  it('uses modelscube as the import name for opencode', () => {
    const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({ ...baseInput, app: 'opencode' }))
    expect(params.get('name')).toBe('modelscube')
  })

  it.each(['claude', 'codex', 'gemini', 'grokbuild'] as CcSwitchApp[])('keeps the provider name for %s imports', (app) => {
    const params = paramsFromDeeplink(buildCcSwitchImportDeeplink({ ...baseInput, app }))
    expect(params.get('name')).toBe(baseInput.providerName)
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

describe('CC Switch usage script', () => {
  // Mirrors CC Switch: substitute the template vars as text, evaluate, read request.url.
  function usageUrlFor(baseUrl: string): string {
    const script = CC_SWITCH_USAGE_SCRIPT.split('{{baseUrl}}').join(baseUrl).split('{{apiKey}}').join('sk-test')
    // eslint-disable-next-line no-new-func
    const config = new Function(`return ${script}`)() as { request: { url: string } }
    return config.request.url
  }

  it.each([
    'https://api.example.com',
    'https://api.example.com/',
    'https://api.example.com/v1',
    'https://api.example.com/v1/'
  ])('queries exactly one /v1/usage for base URL %s', (baseUrl) => {
    expect(usageUrlFor(baseUrl)).toBe('https://api.example.com/v1/usage')
  })

  it('works against the endpoint stored for each selected application', () => {
    for (const app of ['claude', 'codex', 'grokbuild', 'gemini'] as CcSwitchApp[]) {
      const endpoint = paramsFromDeeplink(
        buildCcSwitchImportDeeplink({
          baseUrl: 'https://api.example.com',
          app,
          providerName: 'Sub2API',
          apiKey: 'sk-test',
          usageScript: CC_SWITCH_USAGE_SCRIPT,
          model: 'model-main'
        })
      ).get('endpoint') as string
      expect(usageUrlFor(endpoint)).toBe('https://api.example.com/v1/usage')
    }
  })
})
