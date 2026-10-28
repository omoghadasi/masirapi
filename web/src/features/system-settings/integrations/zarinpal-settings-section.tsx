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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useSystemOptions, getOptionValue } from '../hooks/use-system-options'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const ZARINPAL_AMOUNT_UNITS = ['toman', 'rial'] as const

const createZarinpalSchema = (t: (key: string) => string) =>
  z.object({
    enabled: z.boolean(),
    merchant_id: z.string(),
    sandbox: z.boolean(),
    amount_unit: z.enum(ZARINPAL_AMOUNT_UNITS),
    min_topup: z.number().int().min(1, t('Value must be at least 1')),
  })

type ZarinpalFormValues = z.infer<ReturnType<typeof createZarinpalSchema>>

const ZARINPAL_DEFAULT_OPTIONS = {
  'zarinpal_setting.enabled': false,
  'zarinpal_setting.merchant_id': '',
  'zarinpal_setting.sandbox': false,
  'zarinpal_setting.amount_unit': 'toman',
  'zarinpal_setting.min_topup': 1,
}

type ZarinpalOptionKey = keyof typeof ZARINPAL_DEFAULT_OPTIONS

function toFormValues(
  options: Record<ZarinpalOptionKey, string | number | boolean>
): ZarinpalFormValues {
  return {
    enabled: options['zarinpal_setting.enabled'] === true,
    merchant_id: String(options['zarinpal_setting.merchant_id']),
    sandbox: options['zarinpal_setting.sandbox'] === true,
    amount_unit:
      options['zarinpal_setting.amount_unit'] === 'rial' ? 'rial' : 'toman',
    min_topup: Number(options['zarinpal_setting.min_topup']) || 1,
  }
}

export function ZarinpalSettingsSection() {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const { data } = useSystemOptions()
  const defaultValues = toFormValues(
    getOptionValue(data?.data, ZARINPAL_DEFAULT_OPTIONS)
  )
  const zarinpalSchema = createZarinpalSchema(t)

  const form = useForm<ZarinpalFormValues>({
    resolver: zodResolver(zarinpalSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: ZarinpalFormValues) => {
    const merchantID = values.merchant_id.trim()
    const updates: Array<{ key: string; value: string | number | boolean }> = [
      { key: 'zarinpal_setting.enabled', value: values.enabled },
      { key: 'zarinpal_setting.merchant_id', value: merchantID },
      { key: 'zarinpal_setting.sandbox', value: values.sandbox },
      { key: 'zarinpal_setting.amount_unit', value: values.amount_unit },
      { key: 'zarinpal_setting.min_topup', value: values.min_topup },
    ]

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('ZarinPal Gateway')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save ZarinPal settings'
          />

          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable ZarinPal')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Requires the payment compliance confirmation and a merchant ID.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='merchant_id'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('ZarinPal Merchant ID')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Enter the merchant ID from the panel')}
                    autoComplete='off'
                    {...field}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'The gateway credential used to request and verify payments.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='sandbox'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Use ZarinPal sandbox')}</FormLabel>
                  <FormDescription>
                    {t('Routes payments to the sandbox gateway for testing.')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='amount_unit'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Payment amount unit')}</FormLabel>
                <Select
                  items={[
                    { value: 'toman', label: t('Toman') },
                    { value: 'rial', label: t('Rial') },
                  ]}
                  value={field.value}
                  onValueChange={field.onChange}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder={t('Select amount unit')} />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem value='toman'>{t('Toman')}</SelectItem>
                      <SelectItem value='rial'>{t('Rial')}</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormDescription>
                  {t(
                    'The unit of the top-up price. ZarinPal always bills in Rial, so amounts in Toman are multiplied by ten.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='min_topup'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Minimum top-up amount')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    {...safeNumberFieldProps(field)}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Smallest recharge amount accepted by this gateway, in the unit used by the wallet form.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
