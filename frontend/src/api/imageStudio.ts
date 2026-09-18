import { buildGatewayUrl } from './client'

export interface StudioImage {
  url?: string
  preview_url?: string
  revised_prompt?: string
  width?: number
  height?: number
  size?: string
  quality?: string
}

export interface StudioTask {
  id: string
  status: 'processing' | 'completed' | 'failed'
  created_at: number
  expires_at: number
  request?: { prompt: string; model: string; size?: string; aspect_ratio?: string; quality?: string; n?: number }
  result?: { data?: StudioImage[]; size?: string; quality?: string }
  error?: { message?: string }
}

async function request<T>(key: string, path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(buildGatewayUrl(path), {
    ...init,
    headers: { ...init.headers, Authorization: `Bearer ${key}` },
    cache: 'no-store',
  })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error?.message || body.message || `HTTP ${response.status}`)
  }
  return response.json() as Promise<T>
}

export async function listStudioModels(key: string, signal?: AbortSignal): Promise<string[]> {
  const result = await request<{ data?: Array<{ id?: string }> }>(key, '/v1/models', { signal })
  // Never invent a fallback model: every choice must be returned for this key.
  return [...new Set((result.data || []).map(item => item.id || '').filter(id =>
    /^(gpt-image(?:-|$)|grok-imagine-image(?:-|$)|grok-.*image)/i.test(id),
  ))]
}

export function listStudioTasks(key: string, signal?: AbortSignal) {
  return request<{ data: StudioTask[]; enabled: boolean }>(key, '/v1/images/tasks', { signal })
}

export interface StudioInput {
  model: string
  prompt: string
  ratio: string
  resolution: string
  quality: string
  count: number
  width?: number
  height?: number
  reference?: File
}

export function studioCapabilities(model: string) {
  const grok = /^grok-/i.test(model)
  const flexibleSize = /^gpt-image-2(?:[.-]|$)/i.test(model)
  return { quality: !grok, resolutions: flexibleSize ? ['1K', '2K', '4K'] : ['1K'], grok }
}

export const studioRatios = ['1:1', '5:4', '9:16', '21:9', '16:9', '3:2', '4:3', '4:5', '3:4', '2:3']

export function studioDimensions(ratio: string, resolution: string, width?: number, height?: number): string {
  if (ratio === 'custom') {
    if (!Number.isSafeInteger(width) || !Number.isSafeInteger(height) || width! <= 0 || height! <= 0) throw new Error('Invalid image dimensions')
    return `${width}x${height}`
  }
  if (!studioRatios.includes(ratio)) throw new Error('Invalid aspect ratio')
  const square = resolution === '4K' ? 2880 : resolution === '2K' ? 2048 : 1024
  const long = resolution === '4K' ? 3840 : resolution === '2K' ? 2048 : 1536
  if (ratio === '1:1') return `${square}x${square}`
  const [w, h] = ratio.split(':').map(Number) as [number, number]
  const gcd = (a: number, b: number): number => b ? gcd(b, a % b) : a
  const divisor = gcd(w, h)
  // Keep the exact ratio and align preset sizes to 16 px. Custom input is never rounded.
  const unit = Math.max(16, Math.floor(long / Math.max(w / divisor, h / divisor) / 16) * 16)
  return `${w / divisor * unit}x${h / divisor * unit}`
}

export function submitStudioTask(key: string, input: StudioInput) {
  const caps = studioCapabilities(input.model)
  const fields: Record<string, string> = { model: input.model, prompt: input.prompt, n: String(input.count), response_format: 'b64_json' }
  const size = studioDimensions(input.ratio, input.resolution, input.width, input.height)
  if (caps.grok && input.ratio !== 'custom') fields.aspect_ratio = input.ratio
  else fields.size = size
  if (!caps.grok) fields.quality = input.quality
  if (input.reference) {
    const body = new FormData()
    Object.entries(fields).forEach(([name, value]) => body.append(name, value))
    body.append('image', input.reference)
    return request<StudioTask>(key, '/v1/images/edits/async', { method: 'POST', body })
  }
  return request<StudioTask>(key, '/v1/images/generations/async', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...fields, n: input.count }),
  })
}

export async function fetchStudioImage(key: string, path: string, signal?: AbortSignal): Promise<Blob> {
  // Credentials only travel to our authenticated asset endpoint, never upstream/CDN URLs.
  if (!/^\/v1\/images\/assets\/[0-9a-f-]{36}$/i.test(path)) throw new Error('Invalid image asset URL')
  const response = await fetch(buildGatewayUrl(path), { headers: { Authorization: `Bearer ${key}` }, cache: 'no-store', signal })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  return response.blob()
}
