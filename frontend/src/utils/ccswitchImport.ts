export type CcSwitchApp = 'claude' | 'codex' | 'gemini' | 'opencode'

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  endpointBaseUrl?: string
  app: CcSwitchApp
  providerName: string
  apiKey: string
  usageScript: string
  model: string
  haikuModel?: string
  sonnetModel?: string
  opusModel?: string
}

export function withoutV1Endpoint(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '').replace(/\/v1$/i, '')
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const endpointBaseUrl = input.endpointBaseUrl || input.baseUrl
  const endpoint = input.app === 'claude'
    ? withoutV1Endpoint(endpointBaseUrl)
    : endpointBaseUrl.replace(/\/+$/, '')
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', input.app],
    ['model', input.model],
    ['name', input.providerName],
    ['homepage', input.baseUrl],
    ['endpoint', endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(input.usageScript)],
    ['usageAutoInterval', '30']
  ]

  if (input.app === 'claude') {
    if (input.haikuModel) entries.push(['haikuModel', input.haikuModel])
    if (input.sonnetModel) entries.push(['sonnetModel', input.sonnetModel])
    if (input.opusModel) entries.push(['opusModel', input.opusModel])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
