import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchStudioImage, listStudioModels, studioDimensions, studioRatios, submitStudioTask } from '../imageStudio'

vi.mock('../client', () => ({ buildGatewayUrl: (path: string) => `https://gateway.test${path}` }))
afterEach(() => vi.unstubAllGlobals())

describe('image studio gateway calls', () => {
  it('sends 4K landscape and high quality exactly as selected', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ id: '4k' }) })
    vi.stubGlobal('fetch', fetcher)
    await submitStudioTask('key', { model: 'gpt-image-2', prompt: 'A portrait beside a window', ratio: '16:9', resolution: '4K', quality: 'high', count: 1 })
    expect(JSON.parse(fetcher.mock.calls[0]![1].body)).toMatchObject({ model: 'gpt-image-2', size: '3840x2160', quality: 'high' })
  })
  it('preserves all preset ratios at each resolution, including portrait and ultrawide', () => {
    for (const ratio of studioRatios) for (const resolution of ['1K', '2K', '4K']) {
      const [w, h] = studioDimensions(ratio, resolution).split('x').map(Number)
      const [rw, rh] = ratio.split(':').map(Number)
      expect(w! / h!).toBeCloseTo(rw! / rh!, 10)
      expect(w! % 16).toBe(0)
      expect(h! % 16).toBe(0)
    }
  })

  it('sends the exact custom dimensions and rejects invalid dimensions without a request', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ id: 'custom' }) })
    vi.stubGlobal('fetch', fetcher)
    const input = { model: 'gpt-image-2', prompt: 'Cat', ratio: 'custom', resolution: '4K', quality: 'high', count: 1, width: 1234, height: 987 }
    await submitStudioTask('key', input)
    expect(JSON.parse(fetcher.mock.calls[0]![1].body)).toMatchObject({ size: '1234x987', prompt: 'Cat', quality: 'high' })
    for (const width of [0, -1, 12.5, NaN, Infinity]) expect(() => submitStudioTask('key', { ...input, width })).toThrow('Invalid image dimensions')
    expect(fetcher).toHaveBeenCalledTimes(1)
  })
  it('uses the selected key and only models returned for that key', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'gpt-image-2' }, { id: 'gpt-5.5' }, { id: 'gpt-image-2' }] }) })
    vi.stubGlobal('fetch', fetcher)
    expect(await listStudioModels('key-A')).toEqual(['gpt-image-2'])
    expect(fetcher).toHaveBeenCalledWith('https://gateway.test/v1/models', expect.objectContaining({ headers: { Authorization: 'Bearer key-A' } }))
    fetcher.mockResolvedValue({ ok: true, json: async () => ({ data: [] }) })
    expect(await listStudioModels('key-B')).toEqual([])
  })

  it('never sends credentials to image URLs outside the authenticated asset route', async () => {
    const fetcher = vi.fn()
    vi.stubGlobal('fetch', fetcher)
    for (const url of ['https://external.test/image.png', '/v1/images/assets/../../keys', '//external.test/file']) {
      await expect(fetchStudioImage('secret', url)).rejects.toThrow('Invalid image asset URL')
    }
    expect(fetcher).not.toHaveBeenCalled()
  })

  it('submits reference uploads through the async edit endpoint with key authentication', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ id: 'task-1' }) })
    vi.stubGlobal('fetch', fetcher)
    const file = new File(['image'], 'reference.png', { type: 'image/png' })
    await submitStudioTask('key-B', { model: 'gpt-image-2', prompt: 'Moss', ratio: '2:3', resolution: '1K', quality: 'auto', count: 4, reference: file })
    const [url, options] = fetcher.mock.calls[0]
    expect(url).toBe('https://gateway.test/v1/images/edits/async')
    expect(options.headers.Authorization).toBe('Bearer key-B')
    expect(options.body.get('image')).toBe(file)
    expect(options.body.get('size')).toBe('1024x1536')
    expect(options.body.get('n')).toBe('4')
  })
})
