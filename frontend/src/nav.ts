/**
 * Spatial navigation for remotes: the arrow keys move focus to the nearest
 * [data-nav] element in that direction.
 */
export type Direction = 'left' | 'right' | 'up' | 'down';

const selector = '[data-nav]:not(:disabled)';

/** The keys remotes and keyboards send for Back. */
export const backKeys = new Set(['Escape', 'Backspace', 'GoBack', 'BrowserBack', 'XF86Back', 'Back']);

/**
 * scope is where focus may go: the last [data-nav-scope] element, such as
 * an open dialog, else the whole page.
 */
function scope(): ParentNode {
  const scopes = document.querySelectorAll<HTMLElement>('[data-nav-scope]');
  return scopes[scopes.length - 1] ?? document;
}

export function focusables(root: ParentNode = scope()): HTMLElement[] {
  return [...root.querySelectorAll<HTMLElement>(selector)].filter((el) => {
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  });
}

/** focusFirst focuses the first element of the selector, else any focusable. */
export function focusFirst(preferred?: string) {
  const el = (preferred && document.querySelector<HTMLElement>(preferred)) || focusables()[0];
  el?.focus();
}

/** gap is the distance between two ranges on one axis, 0 when they overlap. */
function gap(a1: number, a2: number, b1: number, b2: number) {
  return Math.max(0, b1 - a2, a1 - b2);
}

/**
 * clipped reports whether an element is scrolled out of its [data-nav-row],
 * a strip that scrolls sideways on its own.
 */
function clipped(el: HTMLElement): boolean {
  const row = el.closest<HTMLElement>('[data-nav-row]');
  if (!row) return false;
  const r = row.getBoundingClientRect();
  const b = el.getBoundingClientRect();
  return b.right <= r.left + 1 || b.left >= r.right - 1;
}

/** show focuses an element and scrolls its row to it. */
function show(el: HTMLElement) {
  el.focus();
  el.scrollIntoView({ block: 'nearest', inline: 'nearest' });
}

export function move(dir: Direction) {
  const all = focusables();
  const current = document.activeElement as HTMLElement | null;
  if (!current || !all.includes(current)) {
    if (all[0]) show(all[0]);
    return;
  }
  // Within a strip, Left and Right walk it in order, scrolling as they go.
  const row = current.closest<HTMLElement>('[data-nav-row]');
  if (row && (dir === 'left' || dir === 'right')) {
    const members = all.filter((el) => row.contains(el));
    const next = members[members.indexOf(current) + (dir === 'right' ? 1 : -1)];
    if (next) {
      show(next);
      return;
    }
  }
  const a = current.getBoundingClientRect();
  let best: HTMLElement | undefined;
  let bestScore = Infinity;
  for (const el of all) {
    if (el === current || clipped(el)) continue;
    const b = el.getBoundingClientRect();
    let along: number;
    let across: number;
    let centers: number;
    if (dir === 'left' || dir === 'right') {
      along = dir === 'right' ? b.left - a.right : a.left - b.right;
      across = gap(a.top, a.bottom, b.top, b.bottom);
      centers = Math.abs(a.top + a.height / 2 - (b.top + b.height / 2));
    } else {
      along = dir === 'down' ? b.top - a.bottom : a.top - b.bottom;
      across = gap(a.left, a.right, b.left, b.right);
      centers = Math.abs(a.left + a.width / 2 - (b.left + b.width / 2));
    }
    // Only elements wholly in that direction, with a little tolerance.
    if (along < -2) continue;
    const score = along + across * 4 + centers * 0.05;
    if (score < bestScore) {
      bestScore = score;
      best = el;
    }
  }
  if (best) show(best);
}

/**
 * trackFocus mirrors focus in a .focused class. Style that class, not
 * :focus: WebKit drops :focus while the window itself is unfocused, and a
 * TV panel must always show where the remote is.
 */
export function trackFocus() {
  let current: HTMLElement | undefined;
  document.addEventListener('focusin', (e) => {
    const el = e.target as HTMLElement;
    if (el === current) return;
    current?.classList.remove('focused');
    current = el;
    el.classList.add('focused');
  });
}

const keys: Record<string, Direction> = { ArrowLeft: 'left', ArrowRight: 'right', ArrowUp: 'up', ArrowDown: 'down' };

/** handleKey moves focus for an arrow key and reports whether it did. */
export function handleKey(e: KeyboardEvent): boolean {
  const dir = keys[e.key];
  if (!dir || e.altKey || e.ctrlKey || e.metaKey) return false;
  e.preventDefault();
  move(dir);
  return true;
}
