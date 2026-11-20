import { createEffect, createResource, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { render } from 'solid-js/web';
import './style.css';
import { appsApi, audio, display, errorText, panel, timer, type TVApp } from './api';
import { locale, t } from './i18n';
import { focusFirst, handleKey, trackFocus } from './nav';
import { applyTheme, loadTheme } from './theme';

loadTheme();
trackFocus();

const [status, setStatus] = createSignal('');
let statusTimer: ReturnType<typeof setTimeout> | undefined;

/** say shows a message in the header for a few seconds. */
function say(message: string) {
  setStatus(message);
  clearTimeout(statusTimer);
  if (message) statusTimer = setTimeout(() => setStatus(''), 6000);
}

/** attempt runs a call and shows its error; it reports whether it worked. */
async function attempt(work: () => Promise<unknown>): Promise<boolean> {
  try {
    await work();
    return true;
  } catch (err) {
    say(errorText(err));
    return false;
  }
}

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
    <button class="app" data-nav onClick={open}>
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
  const [revealed, setRevealed] = createSignal(false);

  const focusApps = () => queueMicrotask(() => focusFirst('.app'));
  let volumeButton: HTMLButtonElement | undefined;
  let outputButton: HTMLButtonElement | undefined;
  let brightnessButton: HTMLButtonElement | undefined;
  let sleepButton: HTMLButtonElement | undefined;
  let cancelSuspendButton: HTMLButtonElement | undefined;
  let suspendButton: HTMLButtonElement | undefined;
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
      refetchVolume();
      refetchOutputs();
      refetchBrightness();
      refetchSleep();
      void Promise.resolve(refetchApps()).then(focusApps);
    }),
  );
  onCleanup(panel.onHidden(cancelReveal));

  const poll = setInterval(refetchSleep, 30_000);
  onCleanup(() => clearInterval(poll));

  const onKey = (e: KeyboardEvent) => {
    if (confirmSuspend()) {
      if (e.key === 'Escape' || e.key === 'Backspace') {
        e.preventDefault();
        closeSuspendDialog();
      } else if (e.key.startsWith('Arrow')) {
        e.preventDefault();
        if (e.key === 'ArrowLeft') cancelSuspendButton?.focus();
        if (e.key === 'ArrowRight') suspendButton?.focus();
      } else if (e.key === 'Tab') {
        e.preventDefault();
        (document.activeElement === cancelSuspendButton ? suspendButton : cancelSuspendButton)?.focus();
      }
      return;
    }
    const focused = document.activeElement;
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
    if (e.key === 'Escape' || e.key === 'Backspace') {
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
    queueMicrotask(() => cancelSuspendButton?.focus());
  };

  const closeSuspendDialog = () => {
    setConfirmSuspend(false);
    queueMicrotask(() => sleepButton?.focus());
  };

  const suspendNow = async () => {
    setConfirmSuspend(false);
    await sleepQueue;
    await attempt(timer.suspendNow);
    refetchSleep();
  };

  return (
    <>
      <div class="scrim" classList={{ revealed: revealed() }} />
      <main class="panel" classList={{ revealed: revealed() }}>
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
          <For each={apps() ?? []}>{(app) => <AppTile app={app} />}</For>
          <Show when={!apps.loading && !apps()?.length}>
            <p class="empty">{t('empty')}</p>
          </Show>
          <ControlTile
            label={t('volume')}
            value={volume() ? (volume()!.muted ? t('muted') : `${volume()!.percent}%`) : t('unavailable')}
            disabled={!volume()}
            description={t('volumeHint')}
            ref={(el) => (volumeButton = el)}
            onClick={toggleMute}
          />
          <ControlTile
            label={t('output')}
            value={currentOutput()?.name ?? t('unavailable')}
            disabled={(outputs()?.length ?? 0) < 2}
            description={t('outputHint')}
            ref={(el) => (outputButton = el)}
            onClick={() => void stepOutput(1)}
          />
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
                <button data-nav ref={cancelSuspendButton} onClick={closeSuspendDialog}>{t('cancel')}</button>
                <button data-nav ref={suspendButton} onClick={() => void suspendNow()}>{t('suspend')}</button>
              </div>
            </div>
          </div>
        </Show>
      </main>
    </>
  );
}

function App() {
  createEffect(applyTheme);
  createEffect(() => (document.documentElement.lang = locale()));
  return <Panel />;
}

render(() => <App />, document.getElementById('root')!);
