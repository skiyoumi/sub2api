import { describe, expect, it } from 'vitest'
import { antigravityEndpoint, buildCcSwitchImportDeeplink } from '../ccswitchImport'

describe('Antigravity CC Switch endpoint', () => {
  it.each([
    ['https://api.example.com', 'https://api.example.com/antigravity'],
    ['https://api.example.com/', 'https://api.example.com/antigravity'],
    ['https://api.example.com///', 'https://api.example.com/antigravity'],
    ['https://api.example.com/sub2api/', 'https://api.example.com/sub2api/antigravity']
  ])('joins the platform path onto %s for both clients', (baseUrl, endpoint) => {
    for (const app of ['claude', 'gemini'] as const) {
      const url = new URL(buildCcSwitchImportDeeplink({
        baseUrl, endpointBaseUrl: antigravityEndpoint(baseUrl), app, providerName: 'Sub2API',
        apiKey: 'sk-test', usageScript: 'return true', model: 'model-main'
      }))
      expect(url.searchParams.get('endpoint')).toBe(endpoint)
      expect(url.searchParams.get('app')).toBe(app)
      expect(url.searchParams.get('homepage')).toBe(baseUrl)
      expect(url.searchParams.get('model')).toBe('model-main')
    }
  })
})
