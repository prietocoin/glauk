import { get, post, patch, del } from '../../shared/js/http.js';

const BASE = '/api/v1/perfiles';

export const listar = () => get(BASE);
export const activos = () => get(`${BASE}/activos`);
export const guardar = (payload) => post(`${BASE}/guardar`, payload).then((data) => data?.data || data);
export const toggle = (nombre, activo) => patch(`${BASE}/${encodeURIComponent(nombre)}/estado`, { activo });
export const eliminar = (nombre) => del(`${BASE}/${encodeURIComponent(nombre)}`);
export const apagarTodos = () => post(`${BASE}/apagar-todos`, {});
export const memorizar = () => post(`${BASE}/plantilla/memorizar`, {});
export const restaurar = () => post(`${BASE}/plantilla/restaurar`, {});

export const api = { listar, activos, guardar, toggle, eliminar, apagarTodos, memorizar, restaurar };
