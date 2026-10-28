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
import i18n, { type InitOptions } from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import { convertDetectedLanguage } from './languages'
import en from './locales/en.json'
import fa from './locales/fa.json'
import fr from './locales/fr.json'
import ja from './locales/ja.json'
import ru from './locales/ru.json'
import vi from './locales/vi.json'
import zhTW from './locales/zh-TW.json'
import zhCN from './locales/zh.json'

export const resources = {
  en,
  zhCN,
  fr,
  ru,
  ja,
  vi,
  zhTW,
  fa,
} as const

/**
 * Interface language for a visitor who has not chosen one. The browser's own
 * language is deliberately not consulted (`order` has no `navigator` entry):
 * this deployment serves Persian first, and a choice made with the language
 * switcher is cached in `localStorage` and wins.
 */
const DEFAULT_INTERFACE_LANGUAGE = 'fa'

/**
 * `LanguageDetector` whose empty result (first visit, nothing cached) resolves
 * to the default language above. The detector itself cannot express a default,
 * and `lng` is not an option: i18next prefers it over the detector, which would
 * make a stored choice stop working after a reload.
 */
export class DefaultLanguageDetector extends LanguageDetector {
  detect() {
    const detected = super.detect()
    // An empty result is an empty array on i18next v19.5+ and `null` before
    // that, so neither `??` nor a truthiness check alone is enough.
    const nothingDetected = Array.isArray(detected)
      ? detected.length === 0
      : !detected
    return nothingDetected ? DEFAULT_INTERFACE_LANGUAGE : detected
  }
}

/**
 * Options for the app's i18n instance. Exported so tests can build a fresh
 * instance: `i18n` is a module singleton and cannot be re-initialized.
 */
export const i18nOptions: InitOptions = {
  resources,
  // Partial locales (see PARTIAL_INTERFACE_LANGUAGES) rely on this per-key
  // English fallback for keys they do not translate yet.
  fallbackLng: 'en',
  supportedLngs: ['en', 'zhCN', 'fr', 'ru', 'ja', 'vi', 'zhTW', 'fa'],
  load: 'currentOnly',
  nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
  debug: import.meta.env.DEV,
  interpolation: {
    escapeValue: false, // not needed for react as it escapes by default
  },
  detection: {
    order: ['localStorage'],
    caches: ['localStorage'],
    // Browsers report `zh-CN`/`zh-TW`/`zh`/`fa-IR`; map them onto our
    // `zhCN`/`zhTW`/`fa` codes (other codes pass through for normal
    // supportedLngs matching).
    convertDetectedLanguage,
  },
}

i18n
  .use(DefaultLanguageDetector)
  .use(initReactI18next)
  .init(i18nOptions)

export default i18n
