import { createEffect, createResource, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { render } from 'solid-js/web';
import './style.css';
import { appsApi, audio, display, errorText, panel, storeApi, timer, type TVApp } from './api';
import { locale, t } from './i18n';
import { backKeys, focusFirst, handleKey, trackFocus } from './nav';
import { attempt, say, status } from './status';
import { Store } from './store';
import { applyTheme, loadTheme } from './theme';

loadTheme();
trackFocus();


/** view is what the panel shows: the row, or the full-screen store. */
const [view, setView] = createSignal<'panel' | 'store'>('panel');

const clamp = (v: number, min: number, max: number) => Math.min(max, Math.max(min, v));

/** orNull turns an unavailable control into null instead of an error. */
const orNull = <T,>(p: Promise<T>) => p.catch(() => null);

function Clock() {
  const [now, setNow] = createSignal(new Date());
  const id = setInterval(() => setNow(new Date()), 10_000);
  onCleanup(() => clearInterval(id));
  return <span class="clock">{new Intl.DateTimeFormat(locale(), { hour: '2-digit', minute: '2-digit' }).format(now())}</span>;
}

function AppTile(props: { app: TVApp }) {
  const open = () => {
    say(t('opening', { name: props.app.name }));
    void attempt(() => appsApi.launch(props.app.command));
  };
  return (
    <button
      class="app"
      data-nav
      data-app-tile
      data-command={props.app.command}
      aria-description={props.app.open ? t('closeAppHint') : undefined}
      onClick={open}
    >
      <Show when={props.app.icon} fallback={<span class="glyph">{props.app.name.slice(0, 1)}</span>}>
        <img src={props.app.icon} alt="" />
      </Show>
      <span>{props.app.name}</span>
    </button>
  );
}

function ControlTile(props: {
  label: string;
  value: string;
  disabled?: boolean;
  description?: string;
  onClick: () => void;
  ref?: (element: HTMLButtonElement) => void;
}) {
  return (
    <button
      class="app control-tile"
      data-nav
      ref={props.ref}
      disabled={props.disabled}
      aria-label={`${props.label}: ${props.value}`}
      aria-description={props.description}
      onClick={props.onClick}
    >
      <span class="tile-value">{props.value}</span>
      <span>{props.label}</span>
      <span class="tile-hint">↑ ↓</span>
    </button>
  );
}

function formatMinutes(minutes: number): string {
  if (!minutes) return t('off');
  if (minutes < 60) return t('minutes', { n: minutes });
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest ? `${t('hours', { n: hours })} ${t('minutes', { n: rest })}` : t('hours', { n: hours });
}

function Panel() {
  const [apps, { refetch: refetchApps }] = createResource(() => appsApi.list());
  const [volume, { refetch: refetchVolume, mutate: setVolume }] = createResource(() => orNull(audio.volume()));
  const [outputs, { refetch: refetchOutputs, mutate: setOutputs }] = createResource(() => orNull(audio.outputs()));
  const [brightness, { refetch: refetchBrightness, mutate: setBrightness }] = createResource(() => orNull(display.brightness()));
  const [sleep, { refetch: refetchSleep, mutate: setSleep }] = createResource(() => timer.state());
  const [confirmSuspend, setConfirmSuspend] = createSignal(false);
  const [confirmCloseApps, setConfirmCloseApps] = createSignal(false);
  const [showCloseApps, setShowCloseApps] = createSignal(false);
  /** The open app whose Close button shows under its tile. */
  const [closeFor, setCloseFor] = createSignal<string | null>(null);
  const [revealed, setRevealed] = createSignal(false);

  const firstAppButton = () => document.querySelector<HTMLButtonElement>('[data-app-tile]');
  const appTile = (command: string) =>
    [...document.querySelectorAll<HTMLButtonElement>('[data-app-tile]')].find((el) => el.dataset.command === command);
  const focusApps = () => queueMicrotask(() => focusFirst('[data-app-tile]'));
  let volumeButton: HTMLButtonElement | undefined;
  let outputButton: HTMLButtonElement | undefined;
  let brightnessButton: HTMLButtonElement | undefined;
  let sleepButton: HTMLButtonElement | undefined;
  let closeAppsButton: HTMLButtonElement | undefined;
  let storeButton: HTMLButtonElement | undefined;
  let cancelDialogButton: HTMLButtonElement | undefined;
  let confirmDialogButton: HTMLButtonElement | undefined;
  let outputQueue = Promise.resolve();
  let outputRevision = 0;
  let sleepQueue = Promise.resolve();
  let sleepRevision = 0;

  let revealFrame = 0;
  const cancelReveal = () => {
    cancelAnimationFrame(revealFrame);
    setRevealed(false);
  };
  onCleanup(cancelReveal);

  // Focus the first app once the list first arrives.
  let placed = false;
  createEffect(() => {
    if (placed || apps.loading) return;
    placed = true;
    focusApps();
  });

  // Every time the panel appears: fresh state, focus on the apps.
  onCleanup(
    panel.onShown(() => {
      // Wait for the hidden first frame before starting the one opening transition.
      cancelReveal();
      revealFrame = requestAnimationFrame(() => {
        revealFrame = requestAnimationFrame(() => setRevealed(true));
      });
      say('');
      setConfirmSuspend(false);
      setConfirmCloseApps(false);
      setShowCloseApps(false);
      setCloseFor(null);
      refetchVolume();
      refetchOutputs();
      refetchBrightness();
      refetchSleep();
      void Promise.resolve(refetchApps()).then(focusApps);
    }),
  );
  onCleanup(panel.onHidden(() => {
    cancelReveal();
    setShowCloseApps(false);
    setCloseFor(null);
    if (view() === 'store') {
      setView('panel');
      void panel.setExpanded(false);
    }
  }));
  // An install or removal changes the row.
  onCleanup(storeApi.onAppsChanged(() => void refetchApps()));

  const openStore = () => {
    say('');
    setShowCloseApps(false);
    setCloseFor(null);
    setView('store');
    void panel.setExpanded(true);
  };

  const closeStore = () => {
    setView('panel');
    void panel.setExpanded(false);
    queueMicrotask(() => storeButton?.focus());
  };

  const poll = setInterval(refetchSleep, 30_000);
  onCleanup(() => clearInterval(poll));

  const onKey = (e: KeyboardEvent) => {
    // The store handles its own keys.
    if (view() !== 'panel') return;
    if (confirmSuspend() || confirmCloseApps()) {
      if (backKeys.has(e.key)) {
        e.preventDefault();
        closeDialog();
      } else if (e.key.startsWith('Arrow')) {
        e.preventDefault();
        if (e.key === 'ArrowLeft') cancelDialogButton?.focus();
        if (e.key === 'ArrowRight') confirmDialogButton?.focus();
      } else if (e.key === 'Tab') {
        e.preventDefault();
        (document.activeElement === cancelDialogButton ? confirmDialogButton : cancelDialogButton)?.focus();
      }
      return;
    }
    const focused = document.activeElement;
    const modified = e.altKey || e.ctrlKey || e.metaKey;
    // Down on an open app offers to close only that app.
    if (e.key === 'ArrowDown' && !modified && focused instanceof HTMLElement && focused.hasAttribute('data-app-tile')) {
      const command = focused.dataset.command;
      if (command && apps()?.find((app) => app.command === command)?.open) {
        e.preventDefault();
        setCloseFor(command);
        queueMicrotask(() => document.querySelector<HTMLButtonElement>('[data-app-close]')?.focus());
        return;
      }
    }
    if (focused instanceof HTMLElement && focused.hasAttribute('data-app-close')) {
      if (e.key === 'ArrowUp' || e.key === 'ArrowLeft' || e.key === 'ArrowRight' || backKeys.has(e.key)) {
        e.preventDefault();
        const command = closeFor();
        setCloseFor(null);
        if (command) queueMicrotask(() => appTile(command)?.focus());
        return;
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        return;
      }
    }
    if (e.key === 'ArrowLeft' && !e.altKey && !e.ctrlKey && !e.metaKey && focused === firstAppButton()) {
      e.preventDefault();
      setShowCloseApps(true);
      queueMicrotask(() => closeAppsButton?.focus());
      return;
    }
    if (focused === closeAppsButton &&
        (e.key === 'ArrowRight' || e.key === 'ArrowUp' || e.key === 'ArrowDown' || backKeys.has(e.key))) {
      e.preventDefault();
      setShowCloseApps(false);
      queueMicrotask(() => firstAppButton()?.focus());
      return;
    }
    if ((e.key === 'ArrowUp' || e.key === 'ArrowDown') &&
        (focused === volumeButton || focused === outputButton || focused === brightnessButton || focused === sleepButton)) {
      e.preventDefault();
      const dir = e.key === 'ArrowUp' ? 1 : -1;
      if (focused === volumeButton) void stepVolume(dir);
      if (focused === outputButton) void stepOutput(dir);
      if (focused === brightnessButton) void stepBrightness(dir);
      if (focused === sleepButton) void stepSleep(dir);
      return;
    }
    if (handleKey(e)) return;
    if (backKeys.has(e.key)) {
      e.preventDefault();
      void panel.hide();
    }
  };
  onMount(() => document.addEventListener('keydown', onKey));
  onCleanup(() => document.removeEventListener('keydown', onKey));

  const stepVolume = async (dir: number) => {
    const v = volume();
    if (!v) return;
    const percent = clamp(v.percent + dir * 5, 0, 100);
    setVolume({ ...v, percent });
    if (!(await attempt(() => audio.setVolume(percent)))) refetchVolume();
  };

  const toggleMute = async () => {
    await attempt(() => audio.toggleMute());
    refetchVolume();
  };

  const currentOutput = () => outputs()?.find((o) => o.default);
  const stepOutput = (dir: number) => {
    const list = outputs();
    if (!list || list.length < 2) return;
    const i = list.findIndex((o) => o.default);
    const next = list[(i + dir + list.length) % list.length];
    setOutputs(list.map((o) => ({ ...o, default: o.id === next.id })));
    const revision = ++outputRevision;
    outputQueue = outputQueue.then(async () => {
      const ok = await attempt(() => audio.setOutput(next.id));
      if (revision === outputRevision) {
        if (!ok) refetchOutputs();
        refetchVolume();
      }
    });
  };

  const stepBrightness = async (dir: number) => {
    const b = brightness();
    if (b == null) return;
    const percent = clamp(b + dir * 10, 1, 100);
    setBrightness(percent);
    if (!(await attempt(() => display.setBrightness(percent)))) refetchBrightness();
  };

  const stepSleep = (dir: number) => {
    const current = sleep();
    if (!current) return;
    const minutes = clamp(current.remaining + dir * 15, 0, 240);
    if (minutes === current.remaining) return;
    setSleep({ minutes, remaining: minutes, error: '' });
    const revision = ++sleepRevision;
    sleepQueue = sleepQueue.then(async () => {
      const ok = await attempt(() => (minutes ? timer.start(minutes) : timer.cancel()));
      if (!ok && revision === sleepRevision) refetchSleep();
    });
  };

  const openSuspendDialog = () => {
    setConfirmSuspend(true);
    queueMicrotask(() => cancelDialogButton?.focus());
  };

  const openCloseAppsDialog = () => {
    setConfirmCloseApps(true);
    queueMicrotask(() => cancelDialogButton?.focus());
  };

  const closeDialog = () => {
    const fromCloseApps = confirmCloseApps();
    setConfirmSuspend(false);
    setConfirmCloseApps(false);
    if (fromCloseApps) setShowCloseApps(false);
    queueMicrotask(() => (fromCloseApps ? firstAppButton() : sleepButton)?.focus());
  };

  const suspendNow = async () => {
    setConfirmSuspend(false);
    await sleepQueue;
    await attempt(timer.suspendNow);
    refetchSleep();
  };

  /** After windows were asked to close: read which apps are open again. */
  const refreshOpenApps = () =>
    setTimeout(() => {
      const command = (document.activeElement as HTMLElement | null)?.dataset.command;
      void Promise.resolve(refetchApps()).then(() => {
        // The row is rebuilt: keep the remote on the same app.
        if (command && (document.activeElement === document.body || !document.activeElement)) appTile(command)?.focus();
      });
    }, 1500);

  const closeApp = async (app: TVApp) => {
    setCloseFor(null);
    appTile(app.command)?.focus();
    try {
      const count = await appsApi.close(app.command);
      say(t(count ? 'closingApp' : 'appNotOpen', { name: app.name }));
    } catch (err) {
      say(errorText(err));
    }
    refreshOpenApps();
  };

  const closeAllApps = async () => {
    setConfirmCloseApps(false);
    setShowCloseApps(false);
    firstAppButton()?.focus();
    try {
      const count = await appsApi.closeAll();
      say(t(count ? 'closingWindows' : 'noOpenApps', { n: count }));
    } catch (err) {
      say(errorText(err));
    }
    refreshOpenApps();
  };

  return (
    <>
      <div class="scrim" classList={{ revealed: revealed(), behind: view() !== 'panel' }} />
      <main class="panel" classList={{ revealed: revealed(), behind: view() !== 'panel' }}>
        <header>
          <div class="brand">
            PLUNZ <b>+</b> <span>Nexus</span>
          </div>
          <p class="status" role="status">
            {status()}
          </p>
          <Clock />
        </header>

        <section class="tiles" aria-label={t('controls')}>
          <div class="app-strip" data-nav-row>
            <Show when={showCloseApps()}>
              <button class="app close-all-tile" data-nav ref={closeAppsButton} aria-description={t('closeAppsHint')} onClick={openCloseAppsDialog} onFocusOut={() => {
                if (!confirmCloseApps()) setShowCloseApps(false);
              }}>
                <span class="close-all-icon" aria-hidden="true">×</span>
                <span>{t('closeApps')}</span>
              </button>
            </Show>
            <For each={apps() ?? []}>
              {(app) => (
                <div class="app-slot">
                  <AppTile app={app} />
                  <Show when={closeFor() === app.command}>
                    <button
                      class="app-close"
                      data-nav
                      data-app-close
                      aria-label={t('closeAppNamed', { name: app.name })}
                      onClick={() => void closeApp(app)}
                      onFocusOut={() => {
                        if (closeFor() === app.command) setCloseFor(null);
                      }}
                    >
                      <span aria-hidden="true">×</span>
                      <span>{t('closeApp')}</span>
                    </button>
                  </Show>
                </div>
              )}
            </For>
            <button class="app store-tile" data-nav ref={storeButton} aria-description={t('moreAppsHint')} onClick={openStore}>
              <span class="store-icon" aria-hidden="true">+</span>
              <span>{t('moreApps')}</span>
            </button>
          </div>
          <ControlTile
            label={t('volume')}
            value={volume() ? (volume()!.muted ? t('muted') : `${volume()!.percent}%`) : t('unavailable')}
            disabled={!volume()}
            description={t('volumeHint')}
            ref={(el) => (volumeButton = el)}
            onClick={toggleMute}
          />
          <Show when={(outputs()?.length ?? 0) > 1}>
            <ControlTile
              label={t('output')}
              value={currentOutput()?.name ?? t('unavailable')}
              description={t('outputHint')}
              ref={(el) => (outputButton = el)}
              onClick={() => void stepOutput(1)}
            />
          </Show>
          <ControlTile
            label={t('brightness')}
            value={brightness() != null ? `${brightness()}%` : t('unavailable')}
            disabled={brightness() == null}
            description={t('brightnessHint')}
            ref={(el) => (brightnessButton = el)}
            onClick={() => void stepBrightness(1)}
          />
          <ControlTile
            label={t('sleep')}
            value={sleep()?.error ? t('failed') : formatMinutes(sleep()?.remaining ?? 0)}
            description={sleep()?.error ? t('suspendFailed', { error: sleep()!.error }) : t('sleepHint')}
            ref={(el) => (sleepButton = el)}
            onClick={openSuspendDialog}
          />
        </section>
        <Show when={confirmSuspend()}>
          <div class="dialog-backdrop">
            <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="suspend-title" aria-describedby="suspend-description">
              <h2 id="suspend-title">{t('suspendNow')}</h2>
              <p id="suspend-description">{t('suspendConfirm')}</p>
              <div class="dialog-actions">
                <button data-nav ref={cancelDialogButton} onClick={closeDialog}>{t('cancel')}</button>
                <button data-nav ref={confirmDialogButton} onClick={() => void suspendNow()}>{t('suspend')}</button>
              </div>
            </div>
          </div>
        </Show>
        <Show when={confirmCloseApps()}>
          <div class="dialog-backdrop">
            <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="close-apps-title" aria-describedby="close-apps-description">
              <h2 id="close-apps-title">{t('closeAppsConfirmTitle')}</h2>
              <p id="close-apps-description">{t('closeAppsConfirm')}</p>
              <div class="dialog-actions">
                <button data-nav ref={cancelDialogButton} onClick={closeDialog}>{t('cancel')}</button>
                <button class="danger-button" data-nav ref={confirmDialogButton} onClick={() => void closeAllApps()}>{t('closeApps')}</button>
              </div>
            </div>
          </div>
        </Show>
      </main>
      <Show when={view() === 'store'}>
        <Store onClose={closeStore} />
      </Show>
    </>
  );
}

function App() {
  createEffect(applyTheme);
  createEffect(() => (document.documentElement.lang = locale()));
  return <Panel />;
}

render(() => <App />, document.getElementById('root')!);
