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
import { describe, expect, it } from 'vitest'

import { Button } from '@/components/ui/button'
import { ButtonGroup } from '@/components/ui/button-group'

// A right-to-left row reverses the children, so physical corner and border
// utilities land on the wrong side of the group.
const PHYSICAL_INLINE = ['rounded-l', 'rounded-r', 'border-l-0', 'border-r-0']

function renderGroup(orientation?: 'horizontal' | 'vertical') {
  const { container } = render(
    <div dir='rtl'>
      <ButtonGroup aria-label='گروه دکمه' orientation={orientation}>
        <Button>اول</Button>
        <Button>دوم</Button>
        <Button>سوم</Button>
      </ButtonGroup>
    </div>
  )

  const group = container.querySelector('[data-slot="button-group"]')
  if (!group) {
    throw new Error('button group did not render')
  }
  return group.className
}

describe('button group direction', () => {
  it('trims corners and the shared border on the inline axis of a horizontal group', () => {
    const className = renderGroup('horizontal')

    expect(className).toContain('rounded-e-none')
    expect(className).toContain('rounded-e-lg!')
    expect(className).toContain('rounded-s-none')
    expect(className).toContain('border-s-0')
  })

  it('never uses physical left or right corners, which would break under right-to-left', () => {
    for (const orientation of ['horizontal', 'vertical'] as const) {
      const className = renderGroup(orientation)
      for (const physical of PHYSICAL_INLINE) {
        expect(className).not.toContain(physical)
      }
    }
  })

  it('keeps a vertical stack on the block axis, which direction cannot flip', () => {
    const className = renderGroup('vertical')

    expect(className).toContain('rounded-b-none')
    expect(className).toContain('rounded-b-lg!')
    expect(className).toContain('rounded-t-none')
    expect(className).toContain('border-t-0')
    expect(className).not.toContain('rounded-s-')
    expect(className).not.toContain('border-s-0')
  })
})
