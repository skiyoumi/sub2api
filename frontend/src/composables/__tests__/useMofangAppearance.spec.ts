import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useMofangAppearance } from '../useMofangAppearance'

function mediaQuery(initial: boolean) {
  let matches = initial
  const handlers = new Set<(event: MediaQueryListEvent) => void>()
  const media = {
    get matches() { return matches }, media: '', onchange: null,
    addEventListener: vi.fn((_type, handler) => handlers.add(handler)),
    removeEventListener: vi.fn((_type, handler) => handlers.delete(handler)),
    addListener: vi.fn(), removeListener: vi.fn(), dispatchEvent: vi.fn(),
  } as unknown as MediaQueryList
  return { media, set(value: boolean) { matches = value; handlers.forEach(handler => handler({ matches: value } as MediaQueryListEvent)) } }
}
describe('Mofang appearance', () => {
  let dark = mediaQuery(false)
  let motion = mediaQuery(false)
  beforeEach(() => {
    localStorage.clear(); document.documentElement.classList.remove('dark')
    dark = mediaQuery(false); motion = mediaQuery(false)
    vi.spyOn(window, 'matchMedia').mockImplementation(query => query.includes('reduced-motion') ? motion.media : dark.media)
  })
  afterEach(() => vi.restoreAllMocks())
  function createView() {
    return mount(defineComponent({ setup: useMofangAppearance, template: '<div />' }))
  }
  it('follows the system until a user chooses a theme', () => {
    const view = createView()
    expect(view.vm.theme).toBe('system')
    dark.set(true)
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    view.vm.setTheme('light'); dark.set(false); dark.set(true)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
    view.vm.setTheme('system')
    expect(localStorage.getItem('theme')).toBeNull()
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    view.unmount()
  })
  it('uses the existing site preference and supports reduced motion', () => {
    localStorage.setItem('theme', 'dark')
    const view = createView()
    expect(view.vm.isDark).toBe(true)
    expect(view.vm.animate).toBe(true)
    motion.set(true)
    expect(view.vm.animate).toBe(false)
    view.unmount()
  })
  it('removes system and storage subscriptions when leaving the new pages', () => {
    const view = createView(); view.unmount()
    expect(dark.media.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function))
    expect(motion.media.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function))
  })
})
