import { describe, expect, it } from 'vitest'
import type { CustomMenuItem } from '@/types'
import { normalizeSidebarMenuOrder, orderSidebarItems, reorderVisibleSidebarItems } from '../sidebarMenuOrder'
import { getSidebarMenuEntries } from '../sidebarMenus'

describe('sidebar menu ordering', () => {
  it('ignores duplicate and obsolete paths and appends new entries without mutating input', () => {
    const items = [{ path: '/keys' }, { path: '/custom/cards' }, { path: '/profile' }]
    const result = orderSidebarItems(items, ['/custom/cards', '/removed', '/custom/cards', '/keys'])
    expect(result).toEqual([items[1], items[0], items[2]])
    expect(items.map(item => item.path)).toEqual(['/keys', '/custom/cards', '/profile'])
  })

  it('removes deleted or moved custom entries from the saved section and includes newly added menus', () => {
    const custom: CustomMenuItem[] = [
      { id: 'moved', label: 'Moved', icon_svg: '', url: 'https://example.com', visibility: 'admin', sort_order: 0 },
      { id: 'new', label: 'New', icon_svg: '', url: 'https://example.com', visibility: 'user', sort_order: 1 },
    ]
    const order = normalizeSidebarMenuOrder({ user: ['/profile', '/custom/moved', '/custom/deleted'], admin: ['/admin/settings'] }, custom)
    expect(order.user?.[0]).toBe('/profile')
    expect(order.user).not.toContain('/custom/moved')
    expect(order.user).not.toContain('/custom/deleted')
    expect(order.user).toContain('/custom/new')
    expect(order.admin).toContain('/custom/moved')
    expect(order.admin).not.toContain('/custom/new')
    expect(getSidebarMenuEntries('admin', custom).filter(item => item.custom).map(item => item.path)).toEqual(['/custom/moved'])
  })
})

it('retains hidden menu positions when visible entries are reordered', () => {
  const items = ['/keys', '/batch-image', '/custom/cards', '/purchase'].map(path => ({ path }))
  const visible = [items[3], items[0], items[2]]
  expect(reorderVisibleSidebarItems(items, visible)).toEqual(['/purchase', '/batch-image', '/keys', '/custom/cards'])
})
