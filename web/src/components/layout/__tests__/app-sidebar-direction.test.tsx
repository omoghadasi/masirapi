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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { AppSidebar } from '@/components/layout/components/app-sidebar'
import { SidebarProvider } from '@/components/ui/sidebar'
import { DirectionProvider, useDirection } from '@/context/direction-provider'
import { LayoutProvider } from '@/context/layout-provider'

vi.mock('@/hooks/use-sidebar-view', () => ({
  useSidebarView: () => ({ key: '__root', view: null, navGroups: [] }),
}))

function DirectionControls() {
  const { dir, setDir } = useDirection()
  return (
    <>
      <output aria-label='Direction'>{dir}</output>
      <button type='button' onClick={() => setDir('rtl')}>
        RTL
      </button>
      <button type='button' onClick={() => setDir('ltr')}>
        LTR
      </button>
    </>
  )
}

function renderSidebar() {
  const result = render(
    <DirectionProvider>
      <LayoutProvider>
        <SidebarProvider defaultOpen>
          <AppSidebar />
          <DirectionControls />
        </SidebarProvider>
      </LayoutProvider>
    </DirectionProvider>
  )

  return {
    ...result,
    sidebarSide: () =>
      result.container.querySelector('[data-slot="sidebar-container"]'),
  }
}

describe('AppSidebar direction', () => {
  afterEach(() => {
    document.cookie = 'dir=; path=/; max-age=0'
    document.documentElement.removeAttribute('dir')
  })

  it('docks the sidebar on the left while the page is left-to-right', () => {
    const { sidebarSide } = renderSidebar()

    expect(screen.getByLabelText('Direction')).toHaveTextContent('ltr')
    expect(sidebarSide()).toHaveAttribute('data-side', 'left')
  })

  it('moves the sidebar to the right when RTL is selected in the settings, and back on LTR', async () => {
    const user = userEvent.setup()
    const { sidebarSide } = renderSidebar()

    await user.click(screen.getByRole('button', { name: 'RTL' }))
    expect(screen.getByLabelText('Direction')).toHaveTextContent('rtl')
    expect(sidebarSide()).toHaveAttribute('data-side', 'right')

    await user.click(screen.getByRole('button', { name: 'LTR' }))
    expect(sidebarSide()).toHaveAttribute('data-side', 'left')
  })
})
