import { createSignal } from 'solid-js';

export type Locale = 'en' | 'pt' | 'es';
export const locales: Locale[] = ['en', 'pt', 'es'];
export const localeNames: Record<Locale, string> = { en: 'English', pt: 'Português', es: 'Español' };

const en = {
  apps: 'Apps',
  empty: 'No compatible apps found',
  refresh: 'Refresh apps',
  close: 'Close',
  opening: 'Opening {name}…',
  volume: 'Volume',
  volumeDown: 'Volume down',
  volumeUp: 'Volume up',
  mute: 'Mute',
  unmute: 'Unmute',
  muted: 'Muted',
  output: 'Output',
  brightness: 'Brightness',
  brightnessDown: 'Brightness down',
  brightnessUp: 'Brightness up',
  unavailable: 'Unavailable',
  sleep: 'Sleep timer',
  off: 'Off',
  minutes: '{n} min',
  hours: '{n} h',
  sleepsIn: 'Suspends in {n} min',
  suspendFailed: 'Could not suspend: {error}',
  language: 'Language',
  theme: 'Theme',
  system: 'System',
  systemNamed: 'System ({name})',
  suspend: 'Suspend',
  restart: 'Restart',
  powerOff: 'Power off',
  confirm: 'Press again to confirm',
};

type Key = keyof typeof en;

const pt: Record<Key, string> = {
  apps: 'Apps',
  empty: 'Nenhum app compatível encontrado',
  refresh: 'Atualizar apps',
  close: 'Fechar',
  opening: 'Abrindo {name}…',
  volume: 'Volume',
  volumeDown: 'Diminuir volume',
  volumeUp: 'Aumentar volume',
  mute: 'Silenciar',
  unmute: 'Ativar som',
  muted: 'Mudo',
  output: 'Saída de áudio',
  brightness: 'Brilho',
  brightnessDown: 'Diminuir brilho',
  brightnessUp: 'Aumentar brilho',
  unavailable: 'Indisponível',
  sleep: 'Timer para suspender',
  off: 'Desligado',
  minutes: '{n} min',
  hours: '{n} h',
  sleepsIn: 'Suspende em {n} min',
  suspendFailed: 'Não foi possível suspender: {error}',
  language: 'Idioma',
  theme: 'Tema',
  system: 'Sistema',
  systemNamed: 'Sistema ({name})',
  suspend: 'Suspender',
  restart: 'Reiniciar',
  powerOff: 'Desligar',
  confirm: 'Aperte de novo para confirmar',
};

const es: Record<Key, string> = {
  apps: 'Apps',
  empty: 'No se encontraron apps compatibles',
  refresh: 'Actualizar apps',
  close: 'Cerrar',
  opening: 'Abriendo {name}…',
  volume: 'Volumen',
  volumeDown: 'Bajar volumen',
  volumeUp: 'Subir volumen',
  mute: 'Silenciar',
  unmute: 'Activar sonido',
  muted: 'Silenciado',
  output: 'Salida de audio',
  brightness: 'Brillo',
  brightnessDown: 'Bajar brillo',
  brightnessUp: 'Subir brillo',
  unavailable: 'No disponible',
  sleep: 'Temporizador de suspensión',
  off: 'Apagado',
  minutes: '{n} min',
  hours: '{n} h',
  sleepsIn: 'Se suspende en {n} min',
  suspendFailed: 'No se pudo suspender: {error}',
  language: 'Idioma',
  theme: 'Tema',
  system: 'Sistema',
  systemNamed: 'Sistema ({name})',
  suspend: 'Suspender',
  restart: 'Reiniciar',
  powerOff: 'Apagar',
  confirm: 'Pulsa otra vez para confirmar',
};

const words: Record<Locale, Record<Key, string>> = { en, pt, es };

function initialLocale(): Locale {
  try {
    const saved = localStorage.getItem('nexus_locale');
    if (locales.includes(saved as Locale)) return saved as Locale;
  } catch {
    // Storage can be unavailable; fall back to the system language.
  }
  const lang = navigator.language.slice(0, 2) as Locale;
  return locales.includes(lang) ? lang : 'en';
}

const [locale, setLocaleSignal] = createSignal<Locale>(initialLocale());
export { locale };

export function setLocale(l: Locale) {
  try {
    localStorage.setItem('nexus_locale', l);
  } catch {
    // The choice then lasts until Nexus restarts.
  }
  setLocaleSignal(l);
}

/** t returns the string for key in the current locale, filling {vars}. */
export function t(key: Key, vars: Record<string, string | number> = {}): string {
  return words[locale()][key].replace(/\{(\w+)\}/g, (_, name) => String(vars[name] ?? ''));
}
