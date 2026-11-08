import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')
const menuSource = readFileSync(resolve(dirname(componentPath), '../../utils/sidebarMenus.ts'), 'utf8')
const composableSource = readFileSync(resolve(dirname(componentPath), '../../composables/useSidebarMenus.ts'), 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar collapsible groups', () => {
  it('lets the user collapse a group even while a child route is active', () => {
    // The expand state must come from the user's override first, falling back
    // to the active-route heuristic only when the user has not clicked yet.
    expect(componentSource).toContain('const groupExpandOverrides = ref<Map<string, boolean>>(new Map())')
    expect(componentSource).not.toContain('expandedGroups.value.has(item.path) || isGroupActive(item)')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar subscription feature flag', () => {
  it('gates the My Subscriptions entry behind the subscription public-settings flag', () => {
    expect(composableSource).toContain('Object.entries(FeatureFlags).map(([name, flag]) => [name, makeSidebarFlag(flag)])')
    expect(menuSource).toMatch(/path: '\/subscriptions'[^\n]*feature: 'subscription'/)
  })

  it('also hides the admin Subscription Management entry on recharge-only sites', () => {
    expect(menuSource).toMatch(/path: '\/admin\/subscriptions'[^\n]*feature: 'subscription'/)
  })

  it('derives the purchase entry label from the site billing mode', () => {
    expect(composableSource).toContain("import { resolveSiteBillingMode } from '@/utils/siteBillingMode'")
    expect(composableSource).toMatch(/case 'recharge_only':\s*return 'nav\.recharge'/)
    expect(composableSource).toMatch(/case 'subscription_only':\s*return 'nav\.subscribe'/)
    expect(composableSource).toContain("item.path === '/purchase' ? { ...item, labelKey: purchaseLabelKey.value } : item")
  })
})

describe('AppSidebar MCP entry', () => {
  it('uses the MCP logo and shows the entry only when the deployment enables MCP', () => {
    expect(componentSource).toContain("render: () => h(Icon, { name: 'mcp' })")
    expect(composableSource).toContain('Object.entries(FeatureFlags).map(([name, flag]) => [name, makeSidebarFlag(flag)])')
    expect(menuSource).toMatch(/path: '\/settings\/mcp'[^\n]*icon: 'MCPIcon', feature: 'mcp'/)
    expect(menuSource).toMatch(/path: '\/settings\/mcp'[^\n]*feature: 'mcp', simpleModeOnly: true/)
    expect(menuSource).not.toMatch(/path: '\/settings\/mcp'[^\n]*PluginIcon/)
  })
})
