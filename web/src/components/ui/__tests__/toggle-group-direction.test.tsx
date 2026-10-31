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

import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

// A right-to-left row reverses the children, so physical corner and border
// utilities land on the wrong side of the segmented control.
const PHYSICAL_INLINE = /(rounded|border)-(tl|tr|bl|br|l|r)\b/

function renderGroup(orientation?: 'horizontal' | 'vertical') {
  const { container } = render(
    <div dir='rtl'>
      <ToggleGroup
        defaultValue={['standard']}
        variant='outline'
        size='sm'
        spacing={0}
        orientation={orientation}
        aria-label='حالت نمایش قیمت'
      >
        <ToggleGroupItem value='standard'>استاندارد</ToggleGroupItem>
        <ToggleGroupItem value='recharge'>قیمت شارژ</ToggleGroupItem>
      </ToggleGroup>
    </div>
  )

  const group = container.querySelector('[data-slot="toggle-group"]')
  const item = container.querySelector('[data-slot="toggle-group-item"]')
  if (!group || !item) {
    throw new Error('toggle group did not render')
  }
  return `${group.className} ${item.className}`
}

describe('toggle group direction', () => {
  it('trims corners and the shared border on the inline axis', () => {
    const classNames = renderGroup('horizontal')

    expect(classNames).toContain('first:rounded-s-lg')
    expect(classNames).toContain('last:rounded-e-lg')
    expect(classNames).toContain('border-s-0')
    expect(classNames).toContain('first:border-s')
  })

  it('never uses physical left or right corners, which would break under right-to-left', () => {
    for (const orientation of ['horizontal', 'vertical'] as const) {
      const offenders = renderGroup(orientation)
        .split(/\s+/)
        .filter((cls) => PHYSICAL_INLINE.test(cls))
      expect(offenders).toEqual([])
    }
  })

  it('keeps a vertical stack on the block axis, which direction cannot flip', () => {
    const classNames = renderGroup('vertical')

    expect(classNames).toContain('first:rounded-t-lg')
    expect(classNames).toContain('last:rounded-b-lg')
    expect(classNames).toContain('border-t-0')
    expect(classNames).toContain('first:border-t')
  })
})
