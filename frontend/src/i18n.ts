export type Locale = 'en' | 'pt' | 'es';
const locales: Locale[] = ['en', 'pt', 'es'];

const en = {
  apps: 'Apps',
  controls: 'Apps and controls',
  empty: 'No compatible apps found',
  opening: 'Opening {name}…',
  volume: 'Volume',
  volumeHint: 'Up and Down adjust volume. Enter mutes or unmutes.',
  muted: 'Muted',
  output: 'Output',
  outputHint: 'Up and Down switch audio outputs. Enter selects the next output.',
  brightness: 'Brightness',
  brightnessHint: 'Up and Down adjust brightness. Enter increases it.',
  unavailable: 'Unavailable',
  sleep: 'Sleep timer',
  sleepHint: 'Up and Down change the timer by 15 minutes. Enter asks to suspend now.',
  suspendNow: 'Suspend now?',
  suspendConfirm: 'The computer will suspend immediately.',
  suspend: 'Suspend',
  cancel: 'Cancel',
  failed: 'Failed',
  off: 'Off',
  minutes: '{n} min',
  hours: '{n} h',
  sleepsIn: 'Suspends in {n} min',
  suspendFailed: 'Could not suspend: {error}',
};

type Key = keyof typeof en;

const pt: Record<Key, string> = {
  apps: 'Apps',
  controls: 'Apps e controles',
  empty: 'Nenhum app compatível encontrado',
  opening: 'Abrindo {name}…',
  volume: 'Volume',
  volumeHint: 'Para cima e para baixo ajustam o volume. Enter ativa ou desativa o mudo.',
  muted: 'Mudo',
  output: 'Saída de áudio',
  outputHint: 'Para cima e para baixo trocam a saída de áudio. Enter seleciona a próxima saída.',
  brightness: 'Brilho',
  brightnessHint: 'Para cima e para baixo ajustam o brilho. Enter aumenta o brilho.',
  unavailable: 'Indisponível',
  sleep: 'Timer para suspender',
  sleepHint: 'Para cima e para baixo mudam o timer em 15 minutos. Enter pede confirmação para suspender agora.',
  suspendNow: 'Suspender agora?',
  suspendConfirm: 'O computador será suspenso imediatamente.',
  suspend: 'Suspender',
  cancel: 'Cancelar',
  failed: 'Falhou',
  off: 'Desligado',
  minutes: '{n} min',
  hours: '{n} h',
  sleepsIn: 'Suspende em {n} min',
  suspendFailed: 'Não foi possível suspender: {error}',
};

const es: Record<Key, string> = {
  apps: 'Apps',
  controls: 'Apps y controles',
  empty: 'No se encontraron apps compatibles',
  opening: 'Abriendo {name}…',
  volume: 'Volumen',
  volumeHint: 'Arriba y abajo ajustan el volumen. Enter activa o desactiva el silencio.',
  muted: 'Silenciado',
  output: 'Salida de audio',
  outputHint: 'Arriba y abajo cambian la salida de audio. Enter selecciona la siguiente salida.',
  brightness: 'Brillo',
  brightnessHint: 'Arriba y abajo ajustan el brillo. Enter lo aumenta.',
  unavailable: 'No disponible',
  sleep: 'Temporizador de suspensión',
  sleepHint: 'Arriba y abajo cambian el temporizador en 15 minutos. Enter pide confirmación para suspender ahora.',
  suspendNow: '¿Suspender ahora?',
  suspendConfirm: 'El equipo se suspenderá inmediatamente.',
  suspend: 'Suspender',
  cancel: 'Cancelar',
  failed: 'Falló',
  off: 'Apagado',
  minutes: '{n} min',
  hours: '{n} h',
  sleepsIn: 'Se suspende en {n} min',
  suspendFailed: 'No se pudo suspender: {error}',
};

const words: Record<Locale, Record<Key, string>> = { en, pt, es };

function initialLocale(): Locale {
  const lang = navigator.language.slice(0, 2) as Locale;
  return locales.includes(lang) ? lang : 'en';
}

const selectedLocale = initialLocale();
export const locale = () => selectedLocale;

/** t returns the string for key in the current locale, filling {vars}. */
export function t(key: Key, vars: Record<string, string | number> = {}): string {
  return words[locale()][key].replace(/\{(\w+)\}/g, (_, name) => String(vars[name] ?? ''));
}
