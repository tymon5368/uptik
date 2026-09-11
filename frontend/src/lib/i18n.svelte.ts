import { getLocale, setLocale, locales } from '$lib/paraglide/runtime.js';

export type SupportedLocale = 'en' | 'de' | 'ja' | 'ar' | 'fr' | 'vi';

export interface LocaleInfo {
  code: SupportedLocale;
  label: string;
  nativeLabel: string;
  country: string;
  dir: 'ltr' | 'rtl';
}

export const LOCALES_METADATA: Record<SupportedLocale, LocaleInfo> = {
  en: { code: 'en', label: 'English', nativeLabel: 'English (US)', country: 'USA, UK, Global', dir: 'ltr' },
  de: { code: 'de', label: 'German', nativeLabel: 'Deutsch', country: 'Germany, Switzerland, Austria', dir: 'ltr' },
  ja: { code: 'ja', label: 'Japanese', nativeLabel: '日本語', country: 'Japan', dir: 'ltr' },
  ar: { code: 'ar', label: 'Arabic', nativeLabel: 'العربية', country: 'UAE, Saudi Arabia, Qatar', dir: 'rtl' },
  fr: { code: 'fr', label: 'French', nativeLabel: 'Français', country: 'France, Switzerland, Canada', dir: 'ltr' },
  vi: { code: 'vi', label: 'Vietnamese', nativeLabel: 'Tiếng Việt', country: 'Vietnam', dir: 'ltr' }
};

export const SUPPORTED_LOCALES_LIST = Object.values(LOCALES_METADATA);

class I18nState {
  current = $state<SupportedLocale>('en');

  constructor() {
    try {
      const loc = getLocale() as SupportedLocale;
      if (loc in LOCALES_METADATA) {
        this.current = loc;
      }
    } catch {
      this.current = 'en';
    }
  }

  set(locale: SupportedLocale) {
    if (locale in LOCALES_METADATA) {
      setLocale(locale as any, { reload: false });
      this.current = locale;
      const meta = LOCALES_METADATA[locale];
      if (typeof document !== 'undefined') {
        document.documentElement.lang = locale;
        document.documentElement.dir = meta.dir;
      }
    }
  }

  get isRtl(): boolean {
    return LOCALES_METADATA[this.current]?.dir === 'rtl';
  }

  get info(): LocaleInfo {
    return LOCALES_METADATA[this.current] || LOCALES_METADATA.en;
  }
}

export const i18n = new I18nState();
