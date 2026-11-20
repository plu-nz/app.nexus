import { createSignal } from 'solid-js';
import { errorText } from './api';

const [status, setStatus] = createSignal('');
let statusTimer: ReturnType<typeof setTimeout> | undefined;

export { status };

/** say shows a message in the header for a few seconds. */
export function say(message: string) {
  setStatus(message);
  clearTimeout(statusTimer);
  if (message) statusTimer = setTimeout(() => setStatus(''), 6000);
}

/** attempt runs a call and shows its error; it reports whether it worked. */
export async function attempt(work: () => Promise<unknown>): Promise<boolean> {
  try {
    await work();
    return true;
  } catch (err) {
    say(errorText(err));
    return false;
  }
}
