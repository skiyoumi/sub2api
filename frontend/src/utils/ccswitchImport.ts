export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.5'
export const GROK_CC_SWITCH_MODEL = 'grok-4.5'

export type CcSwitchApp = 'claude' | 'codex' | 'gemini' | 'opencode' | 'grokbuild'

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

export function antigravityEndpoint(baseUrl: string): string {
  return `${withoutV1Endpoint(baseUrl)}/antigravity`
}

function withV1Endpoint(baseUrl: string): string {
  const normalizedBaseUrl = baseUrl.replace(/\/+$/, '')
  return /\/v1$/i.test(normalizedBaseUrl) ? normalizedBaseUrl : `${normalizedBaseUrl}/v1`
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const endpointBaseUrl = input.endpointBaseUrl || input.baseUrl
  const endpoint = input.app === 'claude'
    ? withoutV1Endpoint(endpointBaseUrl)
    : input.app === 'codex' || input.app === 'grokbuild'
      ? withV1Endpoint(endpointBaseUrl)
      : endpointBaseUrl.replace(/\/+$/, '')
  // opencode imports are branded as 'modelscube'; other apps keep the
  // provider name entered by the user.
  const name = input.app === 'opencode' ? 'modelscube' : input.providerName
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', input.app],
    ['model', input.model],
    ['name', name],
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
