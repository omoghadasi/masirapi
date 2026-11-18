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

// The zibal_setting.* option keys, mapped to the flat PaymentFormValues field
// names the shared payment form uses.
export interface ZibalSettingsValues {
  ZibalEnabled: boolean
  ZibalMerchantID: string
  ZibalSandbox: boolean
  ZibalAmountUnit: string
  ZibalMinTopUp: number
}

export const ZIBAL_OPTION_KEYS = {
  ZibalEnabled: 'zibal_setting.enabled',
  ZibalMerchantID: 'zibal_setting.merchant_id',
  ZibalSandbox: 'zibal_setting.sandbox',
  ZibalAmountUnit: 'zibal_setting.amount_unit',
  ZibalMinTopUp: 'zibal_setting.min_topup',
} as const satisfies Record<keyof ZibalSettingsValues, string>

interface Props {
  values: ZibalSettingsValues
  onValueChange: <K extends keyof ZibalSettingsValues>(
    key: K,
    value: ZibalSettingsValues[K]
  ) => void
}

export function ZibalSettingsSection({ values, onValueChange }: Props) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4 pt-4'>
      <div>
        <h3 className='text-lg font-medium'>{t('Zibal Gateway')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Requires the payment compliance confirmation and a merchant ID unless test mode is on.'
          )}
        </p>
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <SettingsSwitchField
          checked={values.ZibalEnabled}
          onCheckedChange={(v) => onValueChange('ZibalEnabled', v)}
          label={t('Enable Zibal')}
          className='py-0'
        />
        <SettingsSwitchField
          checked={values.ZibalSandbox}
          onCheckedChange={(v) => onValueChange('ZibalSandbox', v)}
          label={t('Use Zibal test mode')}
          description={t(
            "Routes payments to Zibal's built-in test merchant for testing."
          )}
          className='py-0'
        />
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label htmlFor='zibal-merchant-id'>{t('Zibal Merchant ID')}</Label>
          <Input
            id='zibal-merchant-id'
            type='password'
            placeholder={t('Enter the merchant ID from the panel')}
            autoComplete='off'
            value={values.ZibalMerchantID}
            onChange={(event) =>
              onValueChange('ZibalMerchantID', event.target.value)
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
            value={values.ZibalAmountUnit}
            onValueChange={(v) =>
              onValueChange('ZibalAmountUnit', v ?? 'toman')
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
              'The unit of the top-up price. Zibal always bills in Rial, so amounts in Toman are multiplied by ten.'
            )}
          </p>
        </div>

        <div className='grid gap-1.5'>
          <Label htmlFor='zibal-min-topup'>{t('Minimum top-up amount')}</Label>
          <Input
            id='zibal-min-topup'
            type='number'
            min={1}
            value={
              Number.isFinite(values.ZibalMinTopUp) ? values.ZibalMinTopUp : ''
            }
            onChange={(event) => {
              const next = event.target.valueAsNumber
              if (Number.isFinite(next)) {
                onValueChange('ZibalMinTopUp', next)
              }
            }}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Smallest recharge amount accepted by this gateway, in the unit used by the wallet form. Zibal requires at least 1,000 Rial.'
            )}
          </p>
        </div>
      </div>
    </div>
  )
}
