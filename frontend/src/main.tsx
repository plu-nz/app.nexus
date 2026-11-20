import { createEffect, createResource, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { render } from 'solid-js/web';
import './style.css';
import { appsApi, audio, display, errorText, panel, power, timer, type TVApp } from './api';
import { locale, localeNames, locales, setLocale, t } from './i18n';
import { focusFirst, handleKey, trackFocus } from './nav';
import { applyTheme, gtkTheme, loadTheme, setThemeChoice, themeChoice } from './theme';

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

/** Stepper is a − value + control; the remote cannot drag a slider. */
function Stepper(props: { value: string; down: string; up: string; disabled: boolean; onStep: (delta: number) => void }) {
  return (
    <>
      <button class="square" data-nav aria-label={props.down} disabled={props.disabled} onClick={() => props.onStep(-1)}>
        −
      </button>
      <span class="value">{props.value}</span>
      <button class="square" data-nav aria-label={props.up} disabled={props.disabled} onClick={() => props.onStep(1)}>
        +
      </button>
    </>
  );
}

const timerChoices = [0, 15, 30, 60, 120];

function Panel() {
  const [apps, { refetch: refetchApps }] = createResource(() => appsApi.list());
  const [volume, { refetch: refetchVolume, mutate: setVolume }] = createResource(() => orNull(audio.volume()));
  const [outputs, { refetch: refetchOutputs }] = createResource(() => orNull(audio.outputs()));
  const [brightness, { refetch: refetchBrightness, mutate: setBrightness }] = createResource(() => orNull(display.brightness()));
  const [sleep, { refetch: refetchSleep }] = createResource(() => timer.state());
  const [armed, setArmed] = createSignal<'restart' | 'powerOff'>();
  let armTimer: ReturnType<typeof setTimeout> | undefined;

  const focusApps = () => queueMicrotask(() => focusFirst('.app'));

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
      say('');
      setArmed(undefined);
      refetchVolume();
      refetchOutputs();
      refetchBrightness();
      refetchSleep();
      void Promise.resolve(refetchApps()).then(focusApps);
    }),
  );

  const poll = setInterval(refetchSleep, 30_000);
  onCleanup(() => clearInterval(poll));

  const onKey = (e: KeyboardEvent) => {
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
  const nextOutput = async () => {
    const list = outputs();
    if (!list?.length) return;
    const i = list.findIndex((o) => o.default);
    await attempt(() => audio.setOutput(list[(i + 1) % list.length].id));
    refetchOutputs();
    refetchVolume();
  };

  const stepBrightness = async (dir: number) => {
    const b = brightness();
    if (b == null) return;
    const percent = clamp(b + dir * 10, 1, 100);
    setBrightness(percent);
    if (!(await attempt(() => display.setBrightness(percent)))) refetchBrightness();
  };

  const chooseTimer = async (minutes: number) => {
    await attempt(() => (minutes ? timer.start(minutes) : timer.cancel()));
    refetchSleep();
  };

  const sleepNote = () => {
    const s = sleep();
    if (s?.error) return t('suspendFailed', { error: s.error });
    if (s?.minutes) return t('sleepsIn', { n: s.remaining });
    return '';
  };

  /** confirm runs a power action on its second press within a few seconds. */
  const confirm = (action: 'restart' | 'powerOff') => {
    clearTimeout(armTimer);
    if (armed() !== action) {
      setArmed(action);
      armTimer = setTimeout(() => setArmed(undefined), 4000);
      return;
    }
    setArmed(undefined);
    void attempt(action === 'restart' ? power.restart : power.powerOff);
  };

  const themeLabel = () => {
    if (themeChoice() === 'nexus') return 'Nexus';
    const g = gtkTheme();
    return g?.name ? t('systemNamed', { name: g.name }) : t('system');
  };

  return (
    <>
      <div class="scrim" onClick={() => void panel.hide()} />
      <main class="panel">
        <header>
          <div class="brand">
            PLUNZ <b>+</b> <span>Nexus</span>
          </div>
          <p class="status" role="status">
            {status()}
          </p>
          <Clock />
          <button class="square close" data-nav aria-label={t('close')} onClick={() => void panel.hide()}>
            ×
          </button>
        </header>

        <section class="apps" aria-label={t('apps')}>
          <For each={apps() ?? []}>{(app) => <AppTile app={app} />}</For>
          <Show when={!apps.loading && !apps()?.length}>
            <p class="empty">{t('empty')}</p>
          </Show>
        </section>

        <section class="controls">
          <div class="group">
            <h2>{t('volume')}</h2>
            <div class="row">
              <Stepper
                value={volume() ? (volume()!.muted ? t('muted') : `${volume()!.percent}%`) : t('unavailable')}
                down={t('volumeDown')}
                up={t('volumeUp')}
                disabled={!volume()}
                onStep={stepVolume}
              />
              <button data-nav classList={{ selected: volume()?.muted }} disabled={!volume()} onClick={toggleMute}>
                {volume()?.muted ? t('unmute') : t('mute')}
              </button>
            </div>
          </div>

          <div class="group">
            <h2>{t('output')}</h2>
            <div class="row">
              <button class="output" data-nav disabled={(outputs()?.length ?? 0) < 2} onClick={nextOutput}>
                {currentOutput()?.name ?? t('unavailable')}
              </button>
            </div>
          </div>

          <div class="group">
            <h2>{t('brightness')}</h2>
            <div class="row">
              <Stepper
                value={brightness() != null ? `${brightness()}%` : t('unavailable')}
                down={t('brightnessDown')}
                up={t('brightnessUp')}
                disabled={brightness() == null}
                onStep={stepBrightness}
              />
            </div>
          </div>

          <div class="group">
            <h2>
              {t('sleep')}
              <Show when={sleepNote()}>
                <span class="note" classList={{ error: Boolean(sleep()?.error) }}>
                  {' · '}
                  {sleepNote()}
                </span>
              </Show>
            </h2>
            <div class="row">
              <For each={timerChoices}>
                {(m) => (
                  <button class="chip" data-nav classList={{ selected: (sleep()?.minutes ?? 0) === m }} onClick={() => chooseTimer(m)}>
                    {m === 0 ? t('off') : m < 60 ? t('minutes', { n: m }) : t('hours', { n: m / 60 })}
                  </button>
                )}
              </For>
            </div>
          </div>
        </section>

        <footer>
          <button data-nav onClick={() => setLocale(locales[(locales.indexOf(locale()) + 1) % locales.length])}>
            {t('language')}: {localeNames[locale()]}
          </button>
          <button data-nav onClick={() => setThemeChoice(themeChoice() === 'gtk' ? 'nexus' : 'gtk')}>
            {t('theme')}: {themeLabel()}
          </button>
          <button data-nav onClick={() => refetchApps()}>
            {t('refresh')}
          </button>
          <span class="spacer" />
          <button data-nav onClick={() => void attempt(power.suspend)}>
            {t('suspend')}
          </button>
          <button data-nav classList={{ armed: armed() === 'restart' }} onClick={() => confirm('restart')}>
            {armed() === 'restart' ? t('confirm') : t('restart')}
          </button>
          <button data-nav classList={{ armed: armed() === 'powerOff' }} onClick={() => confirm('powerOff')}>
            {armed() === 'powerOff' ? t('confirm') : t('powerOff')}
          </button>
        </footer>
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
