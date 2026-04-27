import locale from './locale.json';

type LocaleKey = keyof typeof locale;

const translations: Record<string, string> = locale;

export function t(key: string, params?: Record<string, string | number>): string {
  let val = translations[key];
  if (val === undefined) {
    return key;
  }
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      val = val.replace(`{${k}}`, String(v));
    }
  }
  return val;
}
