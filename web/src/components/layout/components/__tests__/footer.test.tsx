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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { STATUS_QUERY_KEY } from '@/lib/status-query'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { Footer } from '../footer'

async function renderFooter(footerHtml?: string) {
  useSystemConfigStore.getState().setConfig({
    systemName: 'Test Site',
    footerHtml,
  })
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  queryClient.setQueryData(STATUS_QUERY_KEY, {})
  const router = createRouter({
    routeTree: createRootRoute({ component: Footer }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return screen.findByRole('link', { name: 'Enamad Trust Seal' })
}

afterEach(() => {
  cleanup()
  useSystemConfigStore.getState().setConfig({ footerHtml: undefined })
})

describe('footer Enamad seal', () => {
  it('links to the Enamad trust seal from the default footer', async () => {
    const seal = await renderFooter()

    expect(seal).toHaveAttribute(
      'href',
      'https://trustseal.enamad.ir/?id=8450885&Code=mzh8m4XChho5LEyKeaOpIaDeG6a9OBAg'
    )
    expect(seal).toHaveAttribute('target', '_blank')
    expect(seal).toHaveAttribute('rel', 'noopener')
    expect(seal).toHaveAttribute('referrerpolicy', 'origin')
    expect(
      screen.getByRole('img', { name: 'Enamad Trust Seal' })
    ).toHaveAttribute(
      'src',
      'https://trustseal.enamad.ir/logo.aspx?id=8450885&Code=mzh8m4XChho5LEyKeaOpIaDeG6a9OBAg'
    )
  })

  it('keeps the seal when a custom footer HTML is configured', async () => {
    const seal = await renderFooter('<p>Custom footer</p>')

    expect(screen.getByText('Custom footer')).toBeVisible()
    expect(seal).toBeVisible()
  })
})
