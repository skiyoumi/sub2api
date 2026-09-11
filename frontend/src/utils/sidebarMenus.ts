import type { CustomMenuItem } from '@/types'
import type { RegisteredFeatureFlag } from '@/utils/featureFlags'

export type SidebarSection = 'user' | 'admin'
export type SidebarMenuFeature = RegisteredFeatureFlag | 'opsMonitoring' | 'adminPayment' | 'batchImage'
export type SidebarMenuIcon =
  | 'BatchImageIcon'
  | 'BellIcon'
  | 'ChannelIcon'
  | 'ChartIcon'
  | 'CogIcon'
  | 'CreditCardIcon'
  | 'DashboardIcon'
  | 'FolderIcon'
  | 'GiftIcon'
  | 'GlobeIcon'
  | 'KeyIcon'
  | 'OrderIcon'
  | 'OrderListIcon'
  | 'PluginIcon'
  | 'PriceTagIcon'
  | 'RechargeSubscriptionIcon'
  | 'ServerIcon'
  | 'ShieldIcon'
  | 'SignalIcon'
  | 'TicketIcon'
  | 'UserIcon'
  | 'UsersIcon'

export interface SidebarMenuEntry {
  path: string
  labelKey?: string
  label?: string
  icon?: SidebarMenuIcon
  badgeKey?: string
  custom?: boolean
  customItem?: CustomMenuItem
  children?: SidebarMenuEntry[]
  expandOnly?: boolean
  hideInSimpleMode?: boolean
  simpleModeOnly?: boolean
  userOnly?: boolean
  feature?: SidebarMenuFeature
}

export interface SidebarMenuContext {
  isAdmin: boolean
  isSimpleMode: boolean
  backendModeEnabled: boolean
  features: Partial<Record<SidebarMenuFeature, boolean>>
}

// Shared by the sidebar and its order editor, including groups and visibility rules.
const builtInSidebarMenus: Record<SidebarSection, SidebarMenuEntry[]> = {
  user: [
    { path: '/dashboard', labelKey: 'nav.dashboard', icon: 'DashboardIcon', userOnly: true },
    { path: '/keys', labelKey: 'nav.apiKeys', icon: 'KeyIcon' },
    { path: '/batch-image', labelKey: 'nav.batchImage', icon: 'BatchImageIcon', hideInSimpleMode: true, feature: 'batchImage' },
    { path: '/usage', labelKey: 'nav.usage', icon: 'ChartIcon', hideInSimpleMode: true },
    { path: '/available-channels', labelKey: 'nav.availableChannels', icon: 'ChannelIcon', hideInSimpleMode: true, feature: 'availableChannels' },
    { path: '/model-pricing', labelKey: 'nav.modelPricing', icon: 'PriceTagIcon', hideInSimpleMode: true },
    { path: '/monitor', labelKey: 'nav.channelStatus', icon: 'SignalIcon', feature: 'channelMonitor' },
    { path: '/subscriptions', labelKey: 'nav.mySubscriptions', icon: 'CreditCardIcon', hideInSimpleMode: true },
    { path: '/purchase', labelKey: 'nav.buySubscription', badgeKey: 'nav.limitedTime', icon: 'RechargeSubscriptionIcon', hideInSimpleMode: true, feature: 'payment' },
    { path: '/orders', labelKey: 'nav.myOrders', icon: 'OrderListIcon', hideInSimpleMode: true, feature: 'payment' },
    { path: '/redeem', labelKey: 'nav.redeem', icon: 'GiftIcon', hideInSimpleMode: true },
    { path: '/affiliate', labelKey: 'nav.affiliate', icon: 'UsersIcon', hideInSimpleMode: true, feature: 'affiliate' },
    { path: '/profile', labelKey: 'nav.profile', icon: 'UserIcon' },
  ],
  admin: [
    { path: '/admin/dashboard', labelKey: 'nav.dashboard', icon: 'DashboardIcon' },
    { path: '/admin/ops', labelKey: 'nav.ops', icon: 'ChartIcon', feature: 'opsMonitoring' },
    { path: '/admin/users', labelKey: 'nav.users', icon: 'UsersIcon', hideInSimpleMode: true },
    { path: '/admin/groups', labelKey: 'nav.groups', icon: 'FolderIcon' },
    { path: '/admin/cc-switch-defaults', labelKey: 'nav.ccSwitchDefaults', icon: 'FolderIcon', hideInSimpleMode: true },
    { path: '/admin/model-pricing', labelKey: 'nav.modelPricingManagement', icon: 'PriceTagIcon', hideInSimpleMode: true },
    {
      path: '/admin/channels',
      labelKey: 'nav.channelManagement',
      icon: 'ChannelIcon',
      hideInSimpleMode: true,
      expandOnly: true,
      children: [
        { path: '/admin/channels/pricing', labelKey: 'nav.channelPricing', icon: 'PriceTagIcon' },
        { path: '/admin/channels/monitor', labelKey: 'nav.channelMonitor', icon: 'SignalIcon', feature: 'channelMonitor' },
      ],
    },
    { path: '/admin/subscriptions', labelKey: 'nav.subscriptions', icon: 'CreditCardIcon', hideInSimpleMode: true },
    { path: '/admin/accounts', labelKey: 'nav.accounts', icon: 'GlobeIcon' },
    { path: '/admin/plugins', labelKey: 'nav.plugins', icon: 'PluginIcon', feature: 'pluginManagement' },
    { path: '/admin/announcements', labelKey: 'nav.announcements', icon: 'BellIcon' },
    { path: '/admin/proxies', labelKey: 'nav.proxies', icon: 'ServerIcon' },
    {
      path: '/admin/security-audit',
      labelKey: 'nav.securityAudit',
      icon: 'ShieldIcon',
      expandOnly: true,
      feature: 'riskControl',
      children: [
        { path: '/admin/risk-control', labelKey: 'nav.contentModeration', icon: 'ShieldIcon' },
        { path: '/admin/prompt-audit', labelKey: 'nav.promptAudit', icon: 'ShieldIcon' },
      ],
    },
    { path: '/admin/redeem', labelKey: 'nav.redeemCodes', icon: 'TicketIcon', hideInSimpleMode: true },
    { path: '/admin/promo-codes', labelKey: 'nav.promoCodes', icon: 'GiftIcon', hideInSimpleMode: true },
    {
      path: '/admin/affiliates',
      labelKey: 'nav.affiliateManagement',
      icon: 'UsersIcon',
      hideInSimpleMode: true,
      expandOnly: true,
      feature: 'affiliate',
      children: [
        { path: '/admin/affiliates/invites', labelKey: 'nav.affiliateInviteRecords', icon: 'UsersIcon' },
        { path: '/admin/affiliates/rebates', labelKey: 'nav.affiliateRebateRecords', icon: 'OrderIcon' },
        { path: '/admin/affiliates/transfers', labelKey: 'nav.affiliateTransferRecords', icon: 'CreditCardIcon' },
      ],
    },
    {
      path: '/admin/orders',
      labelKey: 'nav.orderManagement',
      icon: 'OrderIcon',
      hideInSimpleMode: true,
      expandOnly: true,
      feature: 'adminPayment',
      children: [
        { path: '/admin/orders/dashboard', labelKey: 'nav.paymentDashboard', icon: 'ChartIcon' },
        { path: '/admin/orders', labelKey: 'nav.orderManagement', icon: 'OrderIcon' },
        { path: '/admin/orders/plans', labelKey: 'nav.paymentPlans', icon: 'CreditCardIcon' },
      ],
    },
    { path: '/admin/usage', labelKey: 'nav.usage', icon: 'ChartIcon' },
    { path: '/admin/audit-logs', labelKey: 'nav.auditLogs', icon: 'ShieldIcon', hideInSimpleMode: true },
    { path: '/keys', labelKey: 'nav.apiKeys', icon: 'KeyIcon', simpleModeOnly: true },
    { path: '/admin/settings', labelKey: 'nav.settings', icon: 'CogIcon' },
  ],
}

export function getSidebarMenuEntries(section: SidebarSection, customItems: readonly CustomMenuItem[]): SidebarMenuEntry[] {
  return [
    ...builtInSidebarMenus[section],
    ...customItems
      .filter(item => item.visibility === section && item.id)
      .sort((a, b) => a.sort_order - b.sort_order)
      .map(item => ({ path: `/custom/${item.id}`, label: item.label, custom: true, customItem: item })),
  ]
}

export function isSidebarSectionVisible(section: SidebarSection, context: SidebarMenuContext): boolean {
  if (section === 'admin') return context.isAdmin
  return context.isAdmin ? !context.isSimpleMode : !context.backendModeEnabled
}

export function getVisibleSidebarMenuEntries(
  section: SidebarSection, customItems: readonly CustomMenuItem[], context: SidebarMenuContext,
): SidebarMenuEntry[] {
  if (!isSidebarSectionVisible(section, context)) return []
  const filter = (items: SidebarMenuEntry[]): SidebarMenuEntry[] => items
    .filter(item => !(item.userOnly && context.isAdmin)
      && !(item.hideInSimpleMode && context.isSimpleMode)
      && !(item.simpleModeOnly && !context.isSimpleMode)
      && !(item.feature && context.features[item.feature] === false))
    .map(item => item.children ? { ...item, children: filter(item.children) } : item)
  return filter(getSidebarMenuEntries(section, customItems))
}
