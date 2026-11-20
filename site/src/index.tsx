import { render } from 'solid-js/web';
import { createEffect, createSignal, For, onCleanup } from 'solid-js';
import { LinkButton, Theme } from '@plunz/storybook';
import { copy, detectLocale, readPreference, savePreference, type Copy, type Locale, type Preference } from './locale';
import '@plunz/storybook/styles.css';
import './style.css';

const repo = 'https://github.com/plu-nz/app.nexus';
const archive = 'nexus-linux-x86_64.tar.gz';
const download = `${repo}/releases/latest/download/${archive}`;

function Icon(props: { class?: string }) {
  return <svg class={props.class} viewBox="0 0 128 128" aria-hidden="true">
    <rect width="128" height="128" rx="28" fill="#1c1c1c" />
    <rect x="20" y="24" width="88" height="62" rx="8" fill="none" stroke="#a6e3a1" stroke-width="7" />
    <rect x="31" y="60" width="66" height="15" rx="4" fill="#a6e3a1" />
    <circle cx="44" cy="67.5" r="3.5" fill="#1c1c1c" /><circle cx="58" cy="67.5" r="3.5" fill="#1c1c1c" /><circle cx="72" cy="67.5" r="3.5" fill="#1c1c1c" /><circle cx="86" cy="67.5" r="3.5" fill="#1c1c1c" />
    <path d="M30 100c0-6 4-9 9-9h50c5 0 9 3 9 9v6H30z" fill="#cdd6f4" />
  </svg>;
}

function Arrow(props: { direction?: 'down' | 'up' }) {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path d={props.direction === 'up' ? 'M7 17 17 7M8 7h9v9' : 'M12 4v16m-6-6 6 6 6-6'} stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
  </svg>;
}

/** An illustration of the Nexus panel over a playing video. */
function Screen(props: { t: Copy }) {
  return <figure class="tv" aria-hidden="true">
    <div class="screen">
      <div class="mock-video" />
      <div class="mock-panel">
        <div class="mock-head"><span class="mock-brand">PLUNZ <b>+</b> <span>Nexus</span></span><span class="mock-clock">21:30</span></div>
        <div class="mock-tiles">
          <span class="tile focused"><svg viewBox="0 0 128 128"><rect x="22" y="28" width="84" height="54" rx="8" fill="none" stroke="currentColor" stroke-width="9" /><path d="M56 44v22l18-11z" fill="currentColor" /></svg><span>Lounge</span></span>
          <span class="tile"><span class="glyph">V</span><span>VacuumTube</span></span>
          <span class="tile control"><b>65%</b><span>{props.t.mockVolume}</span></span>
          <span class="tile control"><b>TV</b><span>{props.t.mockOutput}</span></span>
          <span class="tile control"><b>80%</b><span>{props.t.mockBrightness}</span></span>
          <span class="tile control"><b>{props.t.mockSleepValue}</b><span>{props.t.mockSleep}</span></span>
        </div>
      </div>
    </div>
    <div class="stand" />
  </figure>;
}

function CodeLine(props: { code: string; t: Copy }) {
  const [copied, setCopied] = createSignal(false);
  let timer: number | undefined;
  onCleanup(() => clearTimeout(timer));
  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(props.code);
      setCopied(true);
      clearTimeout(timer);
      timer = window.setTimeout(() => setCopied(false), 1600);
    } catch { /* The visitor can still select the text. */ }
  };
  return <div class="code"><code>{props.code}</code><button type="button" onClick={copyCode} aria-live="polite">{copied() ? props.t.copied : props.t.copy}</button></div>;
}

function App() {
  const [preference, setPreference] = createSignal<Preference>(readPreference());
  const [systemLocale, setSystemLocale] = createSignal<Locale>(detectLocale());
  const locale = (): Locale => preference() === 'system' ? systemLocale() : preference() as Locale;
  const t = () => copy[locale()];
  const changeLanguage = (value: Preference) => { setPreference(value); savePreference(value); };
  const refreshSystem = () => setSystemLocale(detectLocale());
  const refreshPreference = () => setPreference(readPreference());
  window.addEventListener('languagechange', refreshSystem);
  window.addEventListener('pageshow', refreshPreference);
  onCleanup(() => {
    window.removeEventListener('languagechange', refreshSystem);
    window.removeEventListener('pageshow', refreshPreference);
  });
  createEffect(() => {
    document.documentElement.lang = locale();
    document.title = t().title;
    document.querySelector('meta[name="description"]')?.setAttribute('content', t().description);
  });

  return <Theme accent="green" shape="square">
    <a class="skip-link" href="#main">{t().skip}</a>
    <header class="site-header shell">
      <a class="brand" href="/"><Icon class="brand-icon" /><span>Nexus</span></a>
      <nav aria-label="Nexus"><a href="#idea">{t().navIdea}</a><a href="#features">{t().navFeatures}</a><a href="#install">{t().navInstall}</a></nav>
      <label class="language-control"><span>{t().language}</span><select aria-label={t().language} value={preference()} onChange={event => changeLanguage(event.currentTarget.value as Preference)}><option value="system">{t().system}</option><option value="en">English</option><option value="pt">Português</option><option value="es">Español</option></select></label>
    </header>

    <main id="main">
      <section class="hero shell" aria-labelledby="hero-title">
        <div class="hero-copy">
          <p class="eyebrow">{t().eyebrow}</p>
          <h1 id="hero-title">{t().headlineStart}<br /><em>{t().headlineEnd}</em></h1>
          <p class="lead">{t().lead}</p>
          <div class="actions">
            <LinkButton href={download}>{t().download} <Arrow /></LinkButton>
            <LinkButton href={repo} variant="outline">{t().source}</LinkButton>
          </div>
          <p class="download-meta">{t().downloadMeta}</p>
        </div>
        <Screen t={t()} />
      </section>

      <section class="idea" id="idea" aria-labelledby="idea-title"><div class="shell">
        <p class="eyebrow">{t().ideaEyebrow}</p>
        <h2 id="idea-title">{t().ideaTitle}</h2>
        <p class="idea-body">{t().ideaBody}</p>
        <div class="pillars"><For each={t().pillars}>{(pillar, index) => <article><span class="pillar-no">0{index() + 1}</span><h3>{pillar.title}</h3><p>{pillar.body}</p></article>}</For></div>
      </div></section>

      <section class="features shell" id="features" aria-labelledby="features-title">
        <div>
          <p class="eyebrow">{t().featuresEyebrow}</p>
          <h2 id="features-title">{t().featuresTitle}</h2>
          <ul class="feature-list"><For each={t().features}>{feature => <li>{feature}</li>}</For></ul>
        </div>
        <div class="controls">
          <h3>{t().controlsTitle}</h3>
          <dl><For each={t().controls}>{([key, action]) => <div><dt><kbd>{key}</kbd></dt><dd>{action}</dd></div>}</For></dl>
        </div>
      </section>

      <section class="install" id="install" aria-labelledby="install-title"><div class="shell">
        <div class="install-head">
          <div><p class="eyebrow">{t().installEyebrow}</p><h2 id="install-title">{t().installTitle}</h2></div>
          <div><p>{t().installLead}</p><LinkButton href={download}>{t().download} <Arrow /></LinkButton></div>
        </div>
        <ol class="steps"><For each={t().steps}>{(step, index) => <li><span class="step-no">{index() + 1}</span><div><h3>{step.title}</h3><CodeLine code={step.code} t={t()} /></div></li>}</For></ol>
        <p class="install-note">{t().installNote}</p>
        <p class="install-links"><a href={`${download}.sha256`}>{t().checksum}</a><a href={`${repo}/releases`}>{t().releases}</a><a href={`${repo}#install`}>{t().fromSource}</a></p>
      </div></section>

      <section class="pair shell" aria-label={t().pairEyebrow}>
        <p class="eyebrow">{t().pairEyebrow}</p>
        <p>{t().pairBody}</p>
        <a href="https://lounge.plu.nz/">{t().pairLink} <Arrow direction="up" /></a>
      </section>
    </main>

    <footer class="site-footer shell">
      <p class="disclaimer">{t().disclaimer}</p>
      <div class="footer-links">
        <a href="https://www.plu.nz/">{t().partOf}</a>
        <a href="https://www.plu.nz/privacy">{t().privacy}</a>
        <a href="https://www.plu.nz/terms">{t().terms}</a>
        <a href={repo}>GitHub <Arrow direction="up" /></a>
      </div>
    </footer>
  </Theme>;
}

render(() => <App />, document.getElementById('root')!);
