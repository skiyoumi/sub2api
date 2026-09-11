import type { CustomMenuItem, SidebarMenuOrder } from '@/types'
import { getSidebarMenuEntries, type SidebarSection } from '@/utils/sidebarMenus'

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

// Replace only visible slots, retaining saved positions for disabled or inaccessible entries.
export function reorderVisibleSidebarItems(allItems: readonly { path: string }[], visibleItems: readonly { path: string }[]): string[] {
  const moved = new Set(visibleItems.map(item => item.path))
  let index = 0
  return allItems.map(item => moved.has(item.path) ? visibleItems[index++].path : item.path)
}

export function normalizeSidebarMenuOrder(order: SidebarMenuOrder, customItems: readonly CustomMenuItem[]): SidebarMenuOrder {
  const paths = (section: SidebarSection) => order[section]?.length
    ? orderSidebarItems(getSidebarMenuEntries(section, customItems), order[section]).map(item => item.path)
    : []
  return { user: paths('user'), admin: paths('admin') }
}
