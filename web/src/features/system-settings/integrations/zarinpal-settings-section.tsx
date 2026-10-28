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
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { SettingsSwitchField } from '../components/settings-form-layout'

// The zarinpal_setting.* option keys, mapped to the flat PaymentFormValues
// field names the shared payment form uses.
export interface ZarinpalSettingsValues {
  ZarinpalEnabled: boolean
  ZarinpalMerchantID: string
  ZarinpalSandbox: boolean
  ZarinpalAmountUnit: string
  ZarinpalMinTopUp: number
}

export const ZARINPAL_OPTION_KEYS = {
  ZarinpalEnabled: 'zarinpal_setting.enabled',
  ZarinpalMerchantID: 'zarinpal_setting.merchant_id',
  ZarinpalSandbox: 'zarinpal_setting.sandbox',
  ZarinpalAmountUnit: 'zarinpal_setting.amount_unit',
  ZarinpalMinTopUp: 'zarinpal_setting.min_topup',
} as const satisfies Record<keyof ZarinpalSettingsValues, string>

interface Props {
  values: ZarinpalSettingsValues
  onValueChange: <K extends keyof ZarinpalSettingsValues>(
    key: K,
    value: ZarinpalSettingsValues[K]
  ) => void
}

export function ZarinpalSettingsSection({ values, onValueChange }: Props) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4 pt-4'>
      <div>
        <h3 className='text-lg font-medium'>{t('ZarinPal Gateway')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t('Requires the payment compliance confirmation and a merchant ID.')}
        </p>
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <SettingsSwitchField
          checked={values.ZarinpalEnabled}
          onCheckedChange={(v) => onValueChange('ZarinpalEnabled', v)}
          label={t('Enable ZarinPal')}
          className='py-0'
        />
        <SettingsSwitchField
          checked={values.ZarinpalSandbox}
          onCheckedChange={(v) => onValueChange('ZarinpalSandbox', v)}
          label={t('Use ZarinPal sandbox')}
          description={t('Routes payments to the sandbox gateway for testing.')}
          className='py-0'
        />
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label htmlFor='zarinpal-merchant-id'>
            {t('ZarinPal Merchant ID')}
          </Label>
          <Input
            id='zarinpal-merchant-id'
            type='password'
            placeholder={t('Enter the merchant ID from the panel')}
            autoComplete='off'
            value={values.ZarinpalMerchantID}
            onChange={(event) =>
              onValueChange('ZarinpalMerchantID', event.target.value)
            }
          />
          <p className='text-muted-foreground text-xs'>
            {t('The gateway credential used to request and verify payments.')}
          </p>
        </div>

        <div className='grid gap-1.5'>
          <Label>{t('Payment amount unit')}</Label>
          <Select
            items={[
              { value: 'toman', label: t('Toman') },
              { value: 'rial', label: t('Rial') },
            ]}
            value={values.ZarinpalAmountUnit}
            onValueChange={(v) =>
              onValueChange('ZarinpalAmountUnit', v ?? 'toman')
            }
          >
            <SelectTrigger>
              <SelectValue placeholder={t('Select amount unit')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                <SelectItem value='toman'>{t('Toman')}</SelectItem>
                <SelectItem value='rial'>{t('Rial')}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
          <p className='text-muted-foreground text-xs'>
            {t(
              'The unit of the top-up price. ZarinPal always bills in Rial, so amounts in Toman are multiplied by ten.'
            )}
          </p>
        </div>

        <div className='grid gap-1.5'>
          <Label htmlFor='zarinpal-min-topup'>
            {t('Minimum top-up amount')}
          </Label>
          <Input
            id='zarinpal-min-topup'
            type='number'
            min={1}
            value={
              Number.isFinite(values.ZarinpalMinTopUp)
                ? values.ZarinpalMinTopUp
                : ''
            }
            onChange={(event) => {
              const next = event.target.valueAsNumber
              if (Number.isFinite(next)) {
                onValueChange('ZarinpalMinTopUp', next)
              }
            }}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Smallest recharge amount accepted by this gateway, in the unit used by the wallet form.'
            )}
          </p>
        </div>
      </div>
    </div>
  )
}
