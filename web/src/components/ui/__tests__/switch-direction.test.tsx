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
import { describe, expect, it } from 'vitest'

import { Switch } from '@/components/ui/switch'

describe('switch direction', () => {
  it('keeps the thumb track left-to-right on a right-to-left page', () => {
    const { container } = render(
      <div dir='rtl'>
        <Switch defaultChecked />
      </div>
    )

    const root = container.querySelector('[data-slot="switch"]')
    expect(root).toHaveAttribute('dir', 'ltr')
    expect(root).toHaveAttribute('data-checked')
  })

  it('lets a caller override the track direction', () => {
    const { container } = render(
      <div dir='rtl'>
        <Switch dir='rtl' />
      </div>
    )

    expect(container.querySelector('[data-slot="switch"]')).toHaveAttribute(
      'dir',
      'rtl'
    )
  })

  it('moves the checked state to the thumb while toggling', async () => {
    const user = userEvent.setup()
    const { container } = render(<Switch />)

    const root = screen.getByRole('switch')
    const thumb = () => container.querySelector('[data-slot="switch-thumb"]')
    expect(thumb()).toHaveAttribute('data-unchecked')

    await user.click(root)
    expect(root).toBeChecked()
    expect(thumb()).toHaveAttribute('data-checked')
  })
})
