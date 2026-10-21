import { qs, qsa } from '../../shared/js/dom.js';
import { success, error } from '../../shared/js/toast.js';
import { api } from './api.js';
import { crearVistaDirectorio } from './views/directorio.js';
import { crearVistaEditor } from './views/editor.js';
import { crearVistaPlantilla } from './views/plantilla.js';

const state = {
  view: 'directorio', socios: [], selected: null, query: '', filter: 'todos'
};

const ctx = { api, state, notify: { success, error }, navigate, reloadSocios };

const views = {
  directorio: crearVistaDirectorio(ctx),
  editor: crearVistaEditor(ctx),
  plantilla: crearVistaPlantilla(ctx)
};

function navigate(view) {
  state.view = view;
  qsa('[data-view]').forEach((tab) => tab.classList.toggle('active', tab.dataset.view === view));
  qsa('.view').forEach((section) => section.classList.toggle('hidden', section.id !== `view-${view}`));
  views[view]?.show?.();
}

async function cargarSocios() {
  state.socios = await api.listar();
  views.directorio.setSocios(state.socios);
  return state.socios;
}

async function reloadSocios() {
  try {
    await cargarSocios();
    renderCount();
    if (state.view === 'directorio') views.directorio.render();
    if (state.view === 'plantilla') views.plantilla.render();
  } catch (err) {
    error(err.message || 'No se pudo cargar el directorio.');
  }
}

function renderCount() {
  const total = state.socios.length;
  const activos = state.socios.filter((socio) => socio.activo).length;
  qs('#perfiles-count').textContent = `${total} socios · ${activos} activos`;
  views.directorio.setSocios(state.socios);
}

function registrarServiceWorker() {
  if (!('serviceWorker' in navigator)) return;
  const script = new URL('../sw.js', import.meta.url);
  const scope = new URL('../', import.meta.url).pathname;
  navigator.serviceWorker.register(script, { scope }).catch(() => { /* offline opcional */ });
}

async function boot() {
  qsa('[data-view]').forEach((tab) => tab.addEventListener('click', () => navigate(tab.dataset.view)));
  qs('#btn-refrescar')?.addEventListener('click', reloadSocios);
  qs('#btn-nuevo')?.addEventListener('click', () => {
    views.editor.open(null);
    navigate('editor');
  });
  views.directorio.mount();
  views.editor.mount();
  views.plantilla.mount();
  registrarServiceWorker();
  await reloadSocios();
  navigate('directorio');
}

ctx.editSocio = (socio) => {
  views.editor.open(socio);
  navigate('editor');
};
ctx.closeEditor = () => navigate('directorio');

boot().catch((err) => error(err.message || 'No se pudo iniciar PerfilHub.'));
