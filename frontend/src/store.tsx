import { createEffect, createResource, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { appsApi, storeApi, type Category, type StoreItem, type StoreJob } from './api';
import { locale, t, type Key } from './i18n';
import { backKeys, focusFirst, handleKey } from './nav';
import { attempt, say, status } from './status';

/** Operations by app ID. They outlive the store screen, so they are kept here. */
const [jobs, setJobs] = createSignal<Record<string, StoreJob>>({});
/** App names by ID, for messages about operations that end after the store closed. */
const names: Record<string, string> = {};

const ended = new Set(['done', 'failed', 'canceled']);
const doneKeys: Record<StoreJob['action'], Key> = { install: 'installDone', update: 'updateDone', remove: 'removeDone' };
const failedKeys: Record<StoreJob['action'], Key> = { install: 'installFailed', update: 'updateFailed', remove: 'removeFailed' };

storeApi.onJob((job) => {
  setJobs((all) => {
    const next = { ...all };
    if (ended.has(job.state)) delete next[job.app];
    else next[job.app] = job;
    return next;
  });
  const name = names[job.app] ?? job.app;
  if (job.state === 'done') say(t(doneKeys[job.action], { name }));
  if (job.state === 'failed') say(t(failedKeys[job.action], { name, error: job.error ?? '' }));
  if (job.state === 'canceled') say(t('canceledJob'));
});

const categories: (Category | 'all')[] = ['all', 'video', 'games'];
const categoryKeys: Record<Category | 'all', Key> = { all: 'allApps', video: 'video', games: 'games' };

function formatSize(bytes: number): string {
  const gigabytes = bytes >= 1e9;
  return new Intl.NumberFormat(locale(), {
    style: 'unit',
    unit: gigabytes ? 'gigabyte' : 'megabyte',
    maximumFractionDigits: gigabytes ? 1 : 0,
  }).format(bytes / (gigabytes ? 1e9 : 1e6));
}

const host = (url = '') => url.replace(/^https?:\/\//, '').replace(/\/$/, '');

/** progressText describes an operation in progress. */
function progressText(item: StoreItem, job: StoreJob): string {
  if (job.state === 'queued') return t('stateQueued');
  const name = item.name;
  if (job.action === 'remove') return t('stepRemove', { name });
  const update = job.action === 'update';
  if (!job.step) return t(update ? 'stateUpdating' : 'stateInstalling');
  if (job.step.kind === 'runtime') return t(update ? 'stepRuntimeUpdate' : 'stepRuntime', { n: job.step.n });
  return t(update ? 'stepAppUpdate' : 'stepApp', { name });
}

function stateText(item: StoreItem): string {
  const job = jobs()[item.id];
  if (job) return progressText(item, job);
  if (item.source === 'external' && !item.installed) return t('stateExternal', { site: host(item.website) });
  if (item.system) return t('stateSystem');
  if (item.update) return t('stateUpdate');
  return t(item.installed ? 'stateInstalled' : 'stateNotInstalled');
}

function AppIcon(props: { item: StoreItem }) {
  return (
    <Show when={props.item.icon} fallback={<span class="glyph">{props.item.name.slice(0, 1)}</span>}>
      <img src={props.item.icon} alt="" />
    </Show>
  );
}

/** Store lists the catalog and installs, updates, and removes its apps. */
export function Store(props: { onClose: () => void }) {
  const [category, setCategory] = createSignal<Category | 'all'>('all');
  const [available] = createResource(() => storeApi.available());
  const [items, { refetch }] = createResource(async () => {
    const list = await storeApi.items();
    for (const item of list) names[item.id] = item.name;
    return list;
  });
  const [selectedId, setSelectedId] = createSignal<string>();
  const [confirmRemove, setConfirmRemove] = createSignal(false);

  const visible = () => (items() ?? []).filter((item) => category() === 'all' || item.category === category());
  const selected = () => items()?.find((item) => item.id === selectedId());
  const needsInfo = () => {
    const item = selected();
    return item && item.source === 'flathub' && !item.installed && available() ? item.id : undefined;
  };
  const [info] = createResource(needsInfo, (id) => storeApi.info(id).catch(() => undefined));

  // Sync with operations already running, then follow what is installed.
  void storeApi.jobs().then((list) => setJobs(Object.fromEntries(list.map((job) => [job.app, job]))));
  onCleanup(storeApi.onAppsChanged(() => void refetch()));

  let placed = false;
  createEffect(() => {
    if (placed || items.loading) return;
    placed = true;
    queueMicrotask(() => focusFirst('[data-store-card]'));
  });

  // A button can vanish under focus, such as Cancel when an install ends.
  createEffect(() => {
    jobs();
    items();
    queueMicrotask(() => {
      const el = document.activeElement as HTMLElement | null;
      if (!el || el === document.body || !el.isConnected) focusFirst();
    });
  });

  const openDetail = (id: string) => {
    setSelectedId(id);
    queueMicrotask(() => focusFirst('[data-detail-primary]'));
  };

  const closeDetail = () => {
    const id = selectedId();
    setConfirmRemove(false);
    setSelectedId(undefined);
    queueMicrotask(() => document.querySelector<HTMLElement>(`[data-store-card="${id}"]`)?.focus());
  };

  const askRemove = () => {
    setConfirmRemove(true);
    queueMicrotask(() => focusFirst('[data-remove-cancel]'));
  };

  const closeRemove = () => {
    setConfirmRemove(false);
    queueMicrotask(() => focusFirst('[data-detail-remove]'));
  };

  const run = (work: () => Promise<unknown>) => void attempt(work);

  const remove = (item: StoreItem) => {
    setConfirmRemove(false);
    run(() => storeApi.remove(item.id));
    queueMicrotask(() => focusFirst('[data-detail-primary]'));
  };

  const open = (item: StoreItem) => {
    say(t('opening', { name: item.name }));
    run(() => appsApi.launch(item.command));
  };

  const onKey = (e: KeyboardEvent) => {
    if (backKeys.has(e.key)) {
      e.preventDefault();
      if (confirmRemove()) closeRemove();
      else if (selectedId()) closeDetail();
      else props.onClose();
      return;
    }
    handleKey(e);
  };
  onMount(() => document.addEventListener('keydown', onKey));
  onCleanup(() => document.removeEventListener('keydown', onKey));

  return (
    <div class="store">
      <header>
        <div class="brand">
          PLUNZ <b>+</b> <span>{t('moreApps')}</span>
        </div>
        <p class="status" role="status">
          {status()}
        </p>
      </header>

      <Show when={available() === false}>
        <p class="store-note">{t('storeUnavailable')}</p>
      </Show>

      <nav class="store-tabs" aria-label={t('moreApps')}>
        <For each={categories}>
          {(c) => (
            <button data-nav classList={{ active: category() === c }} aria-pressed={category() === c} onClick={() => setCategory(c)}>
              {t(categoryKeys[c])}
            </button>
          )}
        </For>
      </nav>

      <Show when={!items.loading} fallback={<p class="store-note">{t('loading')}</p>}>
        <section class="store-grid">
          <For each={visible()}>
            {(item) => (
              <button class="store-card" data-nav data-store-card={item.id} classList={{ busy: !!jobs()[item.id] }} onClick={() => openDetail(item.id)}>
                <AppIcon item={item} />
                <span class="store-card-text">
                  <span class="store-card-name">{item.name}</span>
                  <span class="store-card-state" classList={{ highlight: item.update && !jobs()[item.id] }}>{stateText(item)}</span>
                </span>
                <Show when={jobs()[item.id]}>
                  <span class="progress" aria-hidden="true" />
                </Show>
              </button>
            )}
          </For>
        </section>
      </Show>

      <Show when={selected()}>
        {(item) => {
          const job = () => jobs()[item().id];
          const flathub = () => item().source === 'flathub';
          const own = () => item().installed && !item().system && flathub();
          return (
            <div class="dialog-backdrop">
              <div class="dialog store-detail" data-nav-scope role="dialog" aria-modal="true" aria-labelledby="detail-title" aria-describedby="detail-description">
                <div class="store-detail-head">
                  <AppIcon item={item()} />
                  <div>
                    <h2 id="detail-title">{item().name}</h2>
                    <p class="store-detail-state">{stateText(item())}</p>
                  </div>
                </div>
                <p id="detail-description">{t(`desc_${item().id}` as Key)}</p>
                <Show when={job()}>
                  <span class="progress wide" aria-hidden="true" />
                </Show>
                <Show when={!item().installed && info()}>
                  {(details) => (
                    <p class="store-detail-note">
                      {t('sizes', { download: formatSize(details().downloadSize), installed: formatSize(details().installedSize) })}
                      <Show when={details().needsRuntime}> {t('needsRuntime')}</Show>
                    </p>
                  )}
                </Show>
                <Show when={item().system}>
                  <p class="store-detail-note">{t('systemNote')}</p>
                </Show>
                <Show when={!flathub() && !item().installed}>
                  <p class="store-detail-note">{t('externalNote', { name: item().name, site: host(item().website) })}</p>
                </Show>
                <div class="dialog-actions">
                  <Show when={job()} fallback={
                    <>
                      <Show when={item().installed}>
                        <button data-nav data-detail-primary onClick={() => open(item())}>{t('open')}</button>
                      </Show>
                      <Show when={own() && item().update}>
                        <button data-nav onClick={() => run(() => storeApi.update(item().id))}>{t('update')}</button>
                      </Show>
                      <Show when={own()}>
                        <button class="danger-button" data-nav data-detail-remove onClick={askRemove}>{t('remove')}</button>
                      </Show>
                      <Show when={flathub() && !item().installed && available()}>
                        <button data-nav data-detail-primary onClick={() => run(() => storeApi.install(item().id))}>{t('install')}</button>
                      </Show>
                    </>
                  }>
                    <button data-nav data-detail-primary onClick={() => run(() => storeApi.cancel(item().id))}>{t('cancel')}</button>
                  </Show>
                  <button data-nav onClick={closeDetail}>{t('close')}</button>
                </div>
              </div>
              <Show when={confirmRemove()}>
                <div class="dialog-backdrop">
                  <div class="dialog" data-nav-scope role="dialog" aria-modal="true" aria-labelledby="remove-title" aria-describedby="remove-description">
                    <h2 id="remove-title">{t('removeTitle', { name: item().name })}</h2>
                    <p id="remove-description">{t('removeConfirm')}</p>
                    <div class="dialog-actions">
                      <button data-nav data-remove-cancel onClick={closeRemove}>{t('cancel')}</button>
                      <button class="danger-button" data-nav onClick={() => remove(item())}>{t('remove')}</button>
                    </div>
                  </div>
                </div>
              </Show>
            </div>
          );
        }}
      </Show>
    </div>
  );
}
