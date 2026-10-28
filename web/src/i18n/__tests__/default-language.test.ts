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
import i18next from 'i18next'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { DefaultLanguageDetector, i18nOptions } from '@/i18n/config'

async function freshI18n() {
  const instance = i18next.createInstance()
  await instance.use(DefaultLanguageDetector).init({ ...i18nOptions })
  return instance
}

describe('default interface language', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('is Persian for a visitor who has not chosen a language', async () => {
    const i18n = await freshI18n()
    expect(i18n.language).toBe('fa')
    expect(i18n.t('Sign in')).toBe('ورود')
  })

  it('keeps the language the visitor chose', async () => {
    localStorage.setItem('i18nextLng', 'zhCN')
    const i18n = await freshI18n()
    expect(i18n.language).toBe('zhCN')
    expect(i18n.t('Sign in')).not.toBe('ورود')
  })

  it('does not let the browser language override the default', async () => {
    vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['fr-FR', 'fr'])
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('fr-FR')
    const i18n = await freshI18n()
    expect(i18n.language).toBe('fa')
  })

  it('shows the English text for keys Persian does not translate yet', async () => {
    const i18n = await freshI18n()
    const missing = 'A key that no locale file defines'
    expect(i18n.t(missing)).toBe(missing)
  })
})
