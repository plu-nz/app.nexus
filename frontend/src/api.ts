import { Call, Events } from '@wailsio/runtime';

/** The Go services, called by name (main.<Service>.<Method>). */
const call = <T = void>(method: string, ...args: unknown[]) => Call.ByName(`main.${method}`, ...args) as Promise<T>;

export type TVApp = { name: string; command: string; icon: string };
export type TimerState = { minutes: number; remaining: number; error: string };
export type Volume = { percent: number; muted: boolean };
export type Output = { id: number; name: string; default: boolean };

export type Category = 'video' | 'games';

/** A catalog app with its install state. */
export type StoreItem = TVApp & {
  id: string;
  category: Category;
  source: 'flathub' | 'external';
  website?: string;
  installed: boolean;
  /** Installed for all users: it opens, but only the system can change it. */
  system: boolean;
  update: boolean;
};

export type StoreAction = 'install' | 'update' | 'remove';

/** One ref of an operation; flatpak reports no percentage. */
export type StoreStep = { action: 'install' | 'update' | 'uninstall'; kind: 'app' | 'runtime'; id: string; n: number };

export type StoreJob = {
  app: string;
  action: StoreAction;
  state: 'queued' | 'running' | 'done' | 'failed' | 'canceled';
  step?: StoreStep;
  error?: string;
};

export type AppInfo = { downloadSize: number; installedSize: number; runtime: string; needsRuntime: boolean };

export const storeApi = {
  available: () => call<boolean>('StoreService.Available'),
  items: () => call<StoreItem[]>('StoreService.Items'),
  info: (id: string) => call<AppInfo>('StoreService.Info', id),
  install: (id: string) => call('StoreService.Install', id),
  update: (id: string) => call('StoreService.Update', id),
  remove: (id: string) => call('StoreService.Remove', id),
  cancel: (id: string) => call('StoreService.Cancel', id),
  jobs: () => call<StoreJob[]>('StoreService.Jobs'),
  /** onJob hears every change to an operation. */
  onJob: (fn: (job: StoreJob) => void) => Events.On('store', (e) => fn(e.data as StoreJob)),
  /** onAppsChanged runs after an operation changed what is installed. */
  onAppsChanged: (fn: () => void) => Events.On('apps', fn),
};

export const panel = {
  hide: () => call('PanelService.Hide'),
  /** setExpanded makes room for the store when Nexus is a normal window. */
  setExpanded: (expanded: boolean) => call('PanelService.SetExpanded', expanded),
  /** onShown runs every time the panel appears. */
  onShown: (fn: () => void) => Events.On('shown', fn),
  /** onHidden conceals the panel before the window is unmapped. */
  onHidden: (fn: () => void) => Events.On('hidden', fn),
};

export const appsApi = {
  list: () => call<TVApp[]>('AppsService.List'),
  launch: (command: string) => call('AppsService.Launch', command),
  closeAll: () => call<number>('AppsService.CloseAll'),
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
