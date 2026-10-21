import { get, post, put, del, idempotencyKey } from '../../shared/js/http.js';
import { adminHeaders } from '../../shared/js/auth.js';

const BASE = '/api/v1/comprobantes';

export const listar = (params = {}) => get(BASE, { params });
export const actualizar = (hash, payload) => put(`${BASE}/${encodeURIComponent(hash)}`, payload);
export const eliminar = (hash) => del(`${BASE}/${encodeURIComponent(hash)}`);
export const releer = (hash) => post(`${BASE}/${encodeURIComponent(hash)}/releer`, {});
export const liquidar = (payload) => post(`${BASE}/liquidar`, payload);
export const historialTasas = () => get(`${BASE}/tasas/historial`);

export const directorio = () => get('/api/v1/perfiles');
export const filtrosReporte = (rol = '') =>
  get('/api/v1/reportes/filtros', { params: { rol } }).then((data) => data?.data || data || {});

export const cola = () => get('/api/v1/admin/cola', { headers: adminHeaders() }).then((data) => data?.data || []);
export const actualizarCola = (hash, payload) =>
  put(`/api/v1/admin/cola/${encodeURIComponent(hash)}`, payload, { headers: adminHeaders() });
export const eliminarCola = (hash) =>
  del(`/api/v1/admin/cola/${encodeURIComponent(hash)}`, { headers: adminHeaders() });

export function enviarImagen(payload, key = idempotencyKey()) {
  return post('/api/v1/reportes/imagen', payload, { idempotencyKey: key, timeout: 30000 })
    .then((data) => ({ ...data, _idempotencyKey: key }));
}

export const enviarTexto = (payload, key = idempotencyKey()) =>
  post('/api/v1/reportes/texto', payload, { idempotencyKey: key, timeout: 30000 });

export const api = {
  listar, actualizar, eliminar, releer, liquidar, historialTasas, directorio,
  filtrosReporte, cola, actualizarCola, eliminarCola, enviarImagen, enviarTexto
};
