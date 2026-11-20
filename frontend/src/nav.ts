/**
 * Spatial navigation for remotes: the arrow keys move focus to the nearest
 * [data-nav] element in that direction.
 */
export type Direction = 'left' | 'right' | 'up' | 'down';

const selector = '[data-nav]:not(:disabled)';

export function focusables(root: ParentNode = document): HTMLElement[] {
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

export function move(dir: Direction) {
  const all = focusables();
  const current = document.activeElement as HTMLElement | null;
  if (!current || !all.includes(current)) {
    all[0]?.focus();
    return;
  }
  const a = current.getBoundingClientRect();
  let best: HTMLElement | undefined;
  let bestScore = Infinity;
  for (const el of all) {
    if (el === current) continue;
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
  best?.focus();
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
