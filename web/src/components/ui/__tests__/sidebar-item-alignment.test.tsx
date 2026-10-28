/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { afterEach, describe, expect, it } from 'vitest'

import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
} from '@/components/ui/sidebar'

function renderMenuItem(ui: ReactElement) {
  return render(
    <SidebarProvider defaultOpen>
      <SidebarMenu>
        <SidebarMenuItem>{ui}</SidebarMenuItem>
      </SidebarMenu>
    </SidebarProvider>
  )
}

describe('sidebar item alignment', () => {
  afterEach(() => {
    document.documentElement.removeAttribute('dir')
  })

  it('aligns menu button labels with the inline start, not physical left', () => {
    document.documentElement.setAttribute('dir', 'rtl')
    const { getByRole } = renderMenuItem(
      <SidebarMenuButton tooltip='پروفایل'>پروفایل</SidebarMenuButton>
    )

    const button = getByRole('button')
    expect(button.classList.contains('text-start')).toBe(true)
    expect(button.classList.contains('text-left')).toBe(false)
    expect(button.className).not.toContain('pr-8')
  })

  it('draws the sub-menu guide line on the inline start, not physical left', () => {
    document.documentElement.setAttribute('dir', 'rtl')
    const { container } = render(
      <SidebarProvider defaultOpen>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton>کنسول</SidebarMenuButton>
            <SidebarMenuSub>
              <SidebarMenuSubItem>
                <SidebarMenuSubButton>توکن‌ها</SidebarMenuSubButton>
              </SidebarMenuSubItem>
            </SidebarMenuSub>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarProvider>
    )

    const sub = container.querySelector('[data-slot="sidebar-menu-sub"]')
    expect(sub).not.toBeNull()
    expect(sub?.classList.contains('border-s')).toBe(true)
    expect(sub?.classList.contains('border-l')).toBe(false)
  })
})
