import { messages, type Locale, type MessageKey } from '../../../../packages/i18n/generated/messages';

export type { Locale, MessageKey };
export type Translate = (key: MessageKey) => string;

export function detectLocale(): Locale {
  const lang = navigator.language.toLowerCase();
  if (lang.startsWith('en')) {
    return 'en';
  }
  if (lang.startsWith('es')) {
    return 'es';
  }
  return 'pt-BR';
}

export function translator(locale: Locale): Translate {
  return (key) => messages[locale][key];
}
