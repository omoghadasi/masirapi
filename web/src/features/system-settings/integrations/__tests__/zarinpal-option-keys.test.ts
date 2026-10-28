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
import assert from 'node:assert/strict'

import { describe, test } from 'vitest'

import { ZARINPAL_OPTION_KEYS } from '../zarinpal-settings-section'

describe('ZarinPal payment gateway option keys', () => {
  // The contract with setting/payment_zarinpal.go: the "zarinpal_setting"
  // config is serialized through its struct json tags, so these exact
  // suffixes are what the backend persists and reads back.
  test('maps every form field to its persisted zarinpal_setting option', () => {
    assert.deepEqual(ZARINPAL_OPTION_KEYS, {
      ZarinpalEnabled: 'zarinpal_setting.enabled',
      ZarinpalMerchantID: 'zarinpal_setting.merchant_id',
      ZarinpalSandbox: 'zarinpal_setting.sandbox',
      ZarinpalAmountUnit: 'zarinpal_setting.amount_unit',
      ZarinpalMinTopUp: 'zarinpal_setting.min_topup',
    })
  })

  test('option keys are unique and namespaced', () => {
    const keys = Object.values(ZARINPAL_OPTION_KEYS)
    assert.equal(new Set(keys).size, keys.length)
    for (const key of keys) {
      assert.match(key, /^zarinpal_setting\.[a-z_]+$/)
    }
  })
})
