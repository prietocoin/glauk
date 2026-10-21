import { get, post, put, del, request } from '../../shared/js/http.js';

const BASE = '/api/v1/tasas';

export const monitor = () => get(`${BASE}/monitor`);
export const socios = () => get(`${BASE}/socios`).then((data) => data.socios || []);
export const historial = () => get('/api/v1/comprobantes/tasas/historial');
export const hoo = () => get(`${BASE}/hoo-borrador`).then((data) => data.borrador || {});

export const guardarLote = (tasas) => post(`${BASE}/guardar-lote`, { tasas });
export const publicarLote = (modoPrueba) => post(`${BASE}/publicar-lote`, { modo_prueba_global: !!modoPrueba });
export const notificarRemesas = () => post(`${BASE}/notificar-remesas`, {});
export const eliminarLote = (id) => del(`${BASE}/lote/${encodeURIComponent(id)}`);

export const guardarSocio = (nombre, cambios) =>
  put(`${BASE}/socios/${encodeURIComponent(nombre)}`, cambios);
export const disparar = (socio, modoPrueba) =>
  post(`${BASE}/disparar`, { socio, modo_prueba: !!modoPrueba });

export const renderCartelera = (socio) =>
  request(`${BASE}/render/${encodeURIComponent(socio)}`, { responseType: 'blob', timeout: 30000 });

export const api = {
  monitor, socios, historial, hoo, guardarLote, publicarLote,
  notificarRemesas, eliminarLote, guardarSocio, disparar, renderCartelera
};
