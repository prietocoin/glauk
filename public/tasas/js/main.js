import { qs, qsa, on } from '../../shared/js/dom.js';
import { success, error, info } from '../../shared/js/toast.js';
import { formatMoney } from '../../shared/js/format.js';
import { api } from './api.js';
import { crearVistaMonitor } from './views/monitor.js';
import { crearVistaLotes } from './views/lotes.js';
import { crearVistaSocios } from './views/socios.js';
import { crearVistaHoo } from './views/hoo.js';

const state = {
  view: 'monitor', socios: [], selected: null, monitor: { id_tasa: 'T000', tasas: {} },
  borrador: {}, test: true
};

const ctx = {
  api, state,
  notify: { success, error, info },
  formatMoney,
  navigate,
  openCartelera: null
};

const views = {
  monitor: crearVistaMonitor(ctx),
  lotes: crearVistaLotes(ctx),
  socios: crearVistaSocios(ctx),
  hoo: crearVistaHoo(ctx)
};
ctx.monitor = views.monitor;
ctx.sociosView = views.socios;

function navigate(view) {
  state.view = view;
  qsa('[data-view]').forEach((tab) => tab.classList.toggle('active', tab.dataset.view === view));
  qsa('.view').forEach((section) => section.classList.toggle('hidden', section.id !== `view-${view}`));
  views[view]?.show?.();
}

async function cargarSocios() {
  try {
    state.socios = await api.socios();
    views.socios.setSocios(state.socios);
  } catch (err) {
    error(err.message || 'No se pudieron cargar los socios.');
  }
}

function registrarServiceWorker() {
  if (!('serviceWorker' in navigator)) return;
  const script = new URL('../sw.js', import.meta.url);
  const scope = new URL('../', import.meta.url).pathname;
  navigator.serviceWorker.register(script, { scope }).catch(() => {
    /* el panel sigue funcionando sin modo offline */
  });
}

function instalarPWA() {
  let prompt = null;
  const button = qs('#btn-install');
  globalThis.addEventListener('beforeinstallprompt', (event) => {
    event.preventDefault();
    prompt = event;
    button?.classList.remove('hidden');
  });
  button?.addEventListener('click', async () => {
    if (!prompt) return;
    prompt.prompt();
    await prompt.userChoice;
    prompt = null;
    button.classList.add('hidden');
  });
}

async function boot() {
  qsa('[data-view]').forEach((tab) => tab.addEventListener('click', () => navigate(tab.dataset.view)));
  qs('#btn-ir-lotes')?.addEventListener('click', () => navigate('lotes'));
  views.monitor.mount();
  views.lotes.mount();
  views.socios.mount();
  views.hoo.mount();
  instalarPWA();
  registrarServiceWorker();
  await Promise.all([views.monitor.show(), cargarSocios()]);
  navigate('monitor');
}

on(document, 'click', '[data-navigate]', (_event, target) => navigate(target.dataset.navigate));
boot().catch((err) => error(err.message || 'No se pudo iniciar TasasHub.'));
