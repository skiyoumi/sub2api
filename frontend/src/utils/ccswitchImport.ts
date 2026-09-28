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

/**
 * Balance query CC Switch runs against the imported provider. CC Switch fills
 * `{{baseUrl}}` with the provider's base URL as stored. Users may edit it
 * afterwards, so the URL
 * strips an existing `/v1` instead of blindly appending one (`/v1/v1/usage`
 * is a 404 and CC Switch shows "query failed").
 */
export const CC_SWITCH_USAGE_SCRIPT = `({
    request: {
      url: "{{baseUrl}}".replace(/\\/+$/, "").replace(/\\/v1$/, "") + "/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`

function withV1Endpoint(baseUrl: string): string {
  const normalizedBaseUrl = baseUrl.replace(/\/+$/, '')
  return /\/v1$/i.test(normalizedBaseUrl) ? normalizedBaseUrl : `${normalizedBaseUrl}/v1`
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const endpointBaseUrl = input.endpointBaseUrl || input.baseUrl
  // CC Switch's Codex provider appends the OpenAI-compatible path itself.
  const endpoint = input.app === 'claude'
    ? withoutV1Endpoint(endpointBaseUrl)
    : input.app === 'grokbuild'
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
