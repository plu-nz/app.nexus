import { Call, Events } from '@wailsio/runtime';

/** The Go services, called by name (main.<Service>.<Method>). */
const call = <T = void>(method: string, ...args: unknown[]) => Call.ByName(`main.${method}`, ...args) as Promise<T>;

export type TVApp = { name: string; command: string; icon: string };
export type TimerState = { minutes: number; remaining: number; error: string };
export type Volume = { percent: number; muted: boolean };
export type Output = { id: number; name: string; default: boolean };

export const panel = {
  hide: () => call('PanelService.Hide'),
  /** onShown runs every time the panel appears. */
  onShown: (fn: () => void) => Events.On('shown', fn),
};

export const appsApi = {
  list: () => call<TVApp[]>('AppsService.List'),
  launch: (command: string) => call('AppsService.Launch', command),
};

export const timer = {
  start: (minutes: number) => call('TimerService.Start', minutes),
  cancel: () => call('TimerService.Cancel'),
  state: () => call<TimerState>('TimerService.State'),
  suspendNow: () => call('TimerService.SuspendNow'),
};

export const display = {
  brightness: () => call<number>('DisplayService.Brightness'),
  setBrightness: (percent: number) => call('DisplayService.SetBrightness', percent),
};

export const audio = {
  volume: () => call<Volume>('AudioService.Volume'),
  setVolume: (percent: number) => call('AudioService.SetVolume', percent),
  toggleMute: () => call('AudioService.ToggleMute'),
  outputs: () => call<Output[]>('AudioService.Outputs'),
  setOutput: (id: number) => call('AudioService.SetOutput', id),
};

/** errorText turns a failed call into a message for the status line. */
export function errorText(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) return String((err as { message: unknown }).message);
  return String(err);
}
