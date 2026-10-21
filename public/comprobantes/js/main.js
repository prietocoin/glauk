import { qs, qsa } from '../../shared/js/dom.js';
import { success, error } from '../../shared/js/toast.js';
import { api } from './api.js';
import { crearVistaComprobantes } from './views/comprobantes.js';
import { crearVistaDirectorio } from './views/directorio.js';
import { crearVistaReportes } from './views/reportes.js';
import { crearVistaAdmin } from './views/admin.js';

const state = {
  view: 'comprobantes', items: [], socios: [], query: '', page: 1,
  filters: {}, report: { socio: '', desde: '', hasta: '', lote: '', caption: '' }
};

const ctx = { api, state, notify: { success, error }, navigate };

const views = {
  comprobantes: crearVistaComprobantes(ctx),
  directorio: crearVistaDirectorio(ctx),
  reportes: crearVistaReportes(ctx),
  admin: crearVistaAdmin(ctx)
};

function navigate(view) {
  state.view = view;
  qsa('[data-view]').forEach((tab) => tab.classList.toggle('active', tab.dataset.view === view));
  qsa('.view').forEach((section) => section.classList.toggle('hidden', section.id !== `view-${view}`));
  views[view]?.show?.();
}

async function cargarDirectorio() {
  try {
    state.socios = await api.directorio();
    views.directorio.setSocios(state.socios);
    views.reportes.setSocios(state.socios);
  } catch (err) {
    error(err.message || 'No se pudo cargar el directorio.');
  }
}

function registrarServiceWorker() {
  if (!('serviceWorker' in navigator)) return;
  const script = new URL('../sw.js', import.meta.url);
  const scope = new URL('../', import.meta.url).pathname;
  navigator.serviceWorker.register(script, { scope }).catch(() => { /* offline opcional */ });
}

async function boot() {
  qsa('[data-view]').forEach((tab) => tab.addEventListener('click', () => navigate(tab.dataset.view)));
  views.comprobantes.mount();
  views.directorio.mount();
  views.reportes.mount();
  views.admin.mount();
  registrarServiceWorker();
  await cargarDirectorio();
  await views.comprobantes.show();
  navigate('comprobantes');
}

ctx.reloadDirectorio = cargarDirectorio;
boot().catch((err) => error(err.message || 'No se pudo iniciar Atenea.'));
