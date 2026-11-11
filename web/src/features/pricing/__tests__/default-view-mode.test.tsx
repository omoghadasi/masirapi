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
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'

import { useFilters } from '../hooks/use-filters'

function ViewModeFixture() {
  const filters = useFilters([])
  return (
    <div>
      <span data-testid='view-mode'>{filters.viewMode}</span>
      <button type='button' onClick={() => filters.setViewMode('card')}>
        card
      </button>
      <button type='button' onClick={() => filters.setViewMode('table')}>
        table
      </button>
    </div>
  )
}

async function renderPricingFilters(initialEntry: string) {
  const root = createRootRoute()
  const pricing = createRoute({
    getParentRoute: () => root,
    path: '/pricing',
  })
  const index = createRoute({
    getParentRoute: () => pricing,
    path: '/',
    component: ViewModeFixture,
    validateSearch: (search: Record<string, unknown>) => search,
  })
  const router = createRouter({
    routeTree: root.addChildren([pricing.addChildren([index])]),
    history: createMemoryHistory({ initialEntries: [initialEntry] }),
  })
  render(<RouterProvider router={router} />)
  await screen.findByTestId('view-mode')
  return router
}

afterEach(() => {
  cleanup()
})

describe('pricing default view mode', () => {
  it('defaults the model listing to the table view', async () => {
    await renderPricingFilters('/pricing')
    expect(screen.getByTestId('view-mode')).toHaveTextContent('table')
  })

  it('keeps an explicit card view choice from the URL', async () => {
    await renderPricingFilters('/pricing?view=card')
    expect(screen.getByTestId('view-mode')).toHaveTextContent('card')
  })

  it('switches between card and table views', async () => {
    const user = userEvent.setup()
    await renderPricingFilters('/pricing')

    await user.click(screen.getByRole('button', { name: 'card' }))
    expect(screen.getByTestId('view-mode')).toHaveTextContent('card')

    await user.click(screen.getByRole('button', { name: 'table' }))
    expect(screen.getByTestId('view-mode')).toHaveTextContent('table')
  })
})
