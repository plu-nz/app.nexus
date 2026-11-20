import { createSignal } from 'solid-js';
import { Call, Events } from '@wailsio/runtime';

/** GtkTheme is the desktop's GTK theme; colors are CSS values, empty when undefined. */
export type GtkTheme = {
  name: string;
  dark: boolean;
  background: string;
  surface: string;
  text: string;
  accent: string;
  onAccent: string;
  border: string;
  subtle: string;
  error: string;
  success: string;
  warning: string;
};

const [gtkTheme, setGtkTheme] = createSignal<GtkTheme>();
export { gtkTheme };

/** followsGtk reports whether the panel takes its colors from the GTK theme. */
export function followsGtk(): boolean {
  return Boolean(gtkTheme()?.background);
}

const mix = (color: string, percent: number, base: string) => `color-mix(in srgb, ${color} ${percent}%, ${base})`;

/**
 * gtkTokens maps a GTK theme onto the panel's tokens. Colors the theme does
 * not define are mixed from its background and text.
 */
function gtkTokens(g: GtkTheme): Record<string, string> {
  const bg = g.background;
  const text = g.text || (g.dark ? '#ffffff' : '#000000');
  return {
    '--plz-bg': bg,
    '--plz-surface': g.surface && g.surface !== bg ? g.surface : mix(text, 6, bg),
    '--plz-raised': mix(text, 14, bg),
    '--plz-line': mix(text, 24, bg),
    '--plz-text': text,
    '--plz-muted': mix(text, 72, bg),
    '--plz-accent': g.accent,
    '--plz-on-accent': g.onAccent,
    '--plz-danger': g.error,
  };
}

let applied: string[] = [];

/** applyTheme sets the tokens on the root element; the stylesheet holds the Nexus defaults. */
export function applyTheme() {
  const root = document.documentElement;
  for (const name of applied) root.style.removeProperty(name);
  applied = [];
  const g = gtkTheme();
  if (followsGtk() && g) {
    for (const [name, value] of Object.entries(gtkTokens(g))) {
      if (!value) continue;
      root.style.setProperty(name, value);
      applied.push(name);
    }
  }
  root.style.colorScheme = followsGtk() && g && !g.dark ? 'light' : 'dark';
}

/** loadTheme reads the GTK theme now and again whenever the desktop changes it. */
export function loadTheme() {
  const load = () =>
    (Call.ByName('main.ThemeService.GTK') as Promise<GtkTheme>).then((g) => setGtkTheme(g?.background ? g : undefined)).catch(() => undefined);
  Events.On('theme', load);
  void load();
}
