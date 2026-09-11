import type { CustomMenuItem, SidebarMenuOrder } from '@/types'

export type SidebarSection = 'user' | 'admin'

export interface SidebarMenuEntry {
  path: string
  labelKey?: string
  label?: string
  custom?: boolean
  simpleModeOnly?: boolean
}

// Top-level sidebar entries. Groups move as a unit; access is still decided by AppSidebar.
export const builtInSidebarMenus: Record<SidebarSection, SidebarMenuEntry[]> = {
  user: [
    { path: '/dashboard', labelKey: 'nav.dashboard' },
    { path: '/keys', labelKey: 'nav.apiKeys' },
    { path: '/batch-image', labelKey: 'nav.batchImage' },
    { path: '/usage', labelKey: 'nav.usage' },
    { path: '/available-channels', labelKey: 'nav.availableChannels' },
    { path: '/model-pricing', labelKey: 'nav.modelPricing' },
    { path: '/monitor', labelKey: 'nav.channelStatus' },
    { path: '/subscriptions', labelKey: 'nav.mySubscriptions' },
    { path: '/purchase', labelKey: 'nav.buySubscription' },
    { path: '/orders', labelKey: 'nav.myOrders' },
    { path: '/redeem', labelKey: 'nav.redeem' },
    { path: '/affiliate', labelKey: 'nav.affiliate' },
    { path: '/profile', labelKey: 'nav.profile' },
  ],
  admin: [
    { path: '/admin/dashboard', labelKey: 'nav.dashboard' },
    { path: '/admin/ops', labelKey: 'nav.ops' },
    { path: '/admin/users', labelKey: 'nav.users' },
    { path: '/admin/groups', labelKey: 'nav.groups' },
    { path: '/admin/cc-switch-defaults', labelKey: 'nav.ccSwitchDefaults' },
    { path: '/admin/model-pricing', labelKey: 'nav.modelPricingManagement' },
    { path: '/admin/channels', labelKey: 'nav.channelManagement' },
    { path: '/admin/subscriptions', labelKey: 'nav.subscriptions' },
    { path: '/admin/accounts', labelKey: 'nav.accounts' },
    { path: '/admin/plugins', labelKey: 'nav.plugins' },
    { path: '/admin/announcements', labelKey: 'nav.announcements' },
    { path: '/admin/proxies', labelKey: 'nav.proxies' },
    { path: '/admin/security-audit', labelKey: 'nav.securityAudit' },
    { path: '/admin/redeem', labelKey: 'nav.redeemCodes' },
    { path: '/admin/promo-codes', labelKey: 'nav.promoCodes' },
    { path: '/admin/affiliates', labelKey: 'nav.affiliateManagement' },
    { path: '/admin/orders', labelKey: 'nav.orderManagement' },
    { path: '/admin/usage', labelKey: 'nav.usage' },
    { path: '/admin/audit-logs', labelKey: 'nav.auditLogs' },
    { path: '/keys', labelKey: 'nav.apiKeys', simpleModeOnly: true },
    { path: '/admin/settings', labelKey: 'nav.settings' },
  ],
}

// Reorder only the supplied items. Stale/hidden paths never create navigation entries,
// and newly added items retain their default order after the saved entries.
export function orderSidebarItems<T extends { path: string }>(items: readonly T[], order?: readonly string[]): T[] {
  const remaining = new Map(items.map(item => [item.path, item]))
  const result: T[] = []
  for (const path of order ?? []) {
    const item = remaining.get(path)
    if (!item) continue
    result.push(item)
    remaining.delete(path)
  }
  return [...result, ...remaining.values()]
}

export function getSidebarMenuEntries(section: SidebarSection, customItems: readonly CustomMenuItem[]): SidebarMenuEntry[] {
  return [
    ...builtInSidebarMenus[section],
    ...customItems
      .filter(item => item.visibility === section && item.id)
      .sort((a, b) => a.sort_order - b.sort_order)
      .map(item => ({ path: `/custom/${item.id}`, label: item.label, custom: true })),
  ]
}

export function normalizeSidebarMenuOrder(order: SidebarMenuOrder, customItems: readonly CustomMenuItem[]): SidebarMenuOrder {
  const paths = (section: SidebarSection) => order[section]?.length
    ? orderSidebarItems(getSidebarMenuEntries(section, customItems), order[section]).map(item => item.path)
    : []
  return { user: paths('user'), admin: paths('admin') }
}
