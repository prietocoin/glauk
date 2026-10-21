/**
 * @file comprobantesApiService.js
 * @description Servicio atómico de red para el CRUD de comprobantes contra el
 * backend Go unificado. Mantiene la paginación y los filtros de la interfaz
 * glauk2 sobre el contrato /api/v1/comprobantes.
 */

const API_URL = '/api/comprobantes';

function obtenerIdentificadorValido(identificador, fallbackPayload = {}) {
  let target = '';

  if (typeof identificador === 'object' && identificador !== null) {
    target = identificador.hash_largo ||
             identificador.id ||
             identificador._id ||
             identificador.hash ||
             identificador.referencia || '';
  } else {
    target = identificador;
  }

  if (!target) {
    target = fallbackPayload?.hash_largo ||
             fallbackPayload?.hash ||
             fallbackPayload?.id ||
             fallbackPayload?._id ||
             fallbackPayload?.referencia || '';
  }

  return String(target || '').trim();
}

function normalizarFecha(valor) {
  const fecha = new Date(valor || 0);
  return isNaN(fecha.getTime()) ? 0 : fecha.getTime();
}

function aplicarRangoHash(lista, desdeHash, hastaHash) {
  if (!desdeHash && !hastaHash) return lista;
  const hashes = lista.map(item => String(item.hash_largo || item.id || ''));
  const desde = desdeHash ? hashes.indexOf(String(desdeHash)) : 0;
  const hasta = hastaHash ? hashes.indexOf(String(hastaHash)) : hashes.length - 1;
  if (desde === -1 || hasta === -1) return lista;
  const inicio = Math.min(desde, hasta);
  const fin = Math.max(desde, hasta);
  return lista.slice(inicio, fin + 1);
}

function ordenarLista(lista, orden) {
  const copia = [...lista];
  switch (orden) {
    case 'fecha_asc':
      return copia.sort((a, b) => normalizarFecha(a.fecha_hora_comprobante || a.timestamp_msg) - normalizarFecha(b.fecha_hora_comprobante || b.timestamp_msg));
    case 'monto_asc':
      return copia.sort((a, b) => (parseFloat(a.monto) || 0) - (parseFloat(b.monto) || 0));
    case 'monto_desc':
      return copia.sort((a, b) => (parseFloat(b.monto) || 0) - (parseFloat(a.monto) || 0));
    case 'fecha_desc':
    default:
      return copia.sort((a, b) => normalizarFecha(b.fecha_hora_comprobante || b.timestamp_msg) - normalizarFecha(a.fecha_hora_comprobante || a.timestamp_msg));
  }
}

/**
 * Carga los comprobantes y emula la paginación de la API v2 sobre el contrato
 * canónico. El backend devuelve como máximo 100 filas por consulta; el frontend
 * aplica rango de hashes, orden y página para conservar el scroll incremental.
 */
export async function fetchComprobantesPaginados(params = {}) {
  try {
    const {
      page = 1,
      limit = 20,
      socio = '',
      rol = '',
      fechaInicio = '',
      fechaFin = '',
      desdeHash = '',
      hastaHash = '',
      orden = 'fecha_desc',
      hash = ''
    } = params;

    const url = new URL(API_URL, window.location.origin);
    if (socio && socio.toUpperCase() !== 'TODOS' && socio.toUpperCase() !== 'TODOS LOS SOCIOS') {
      url.searchParams.append('socio', socio.trim());
    }
    if (rol && rol.toUpperCase() !== 'TODOS') url.searchParams.append('rol', rol.trim());
    if (fechaInicio) url.searchParams.append('fechaInicio', fechaInicio.trim());
    if (fechaFin) url.searchParams.append('fechaFin', fechaFin.trim());
    if (hash) url.searchParams.append('hash', hash.trim());

    const res = await fetch(url.toString());

    const contentType = res.headers.get('content-type');
    if (!contentType || !contentType.includes('application/json')) {
      console.warn(`[comprobantesApiService ⚠️ Warning]: La API respondió con formato no JSON (${res.status}).`);
      return [];
    }

    if (!res.ok) throw new Error(`HTTP Error: ${res.status}`);

    const json = await res.json();
    const lista = json?.comprobantes || json?.data || (Array.isArray(json) ? json : []);

    const filtrada = aplicarRangoHash(Array.isArray(lista) ? lista : [], desdeHash, hastaHash);
    const ordenada = ordenarLista(filtrada, orden);
    const inicio = Math.max(0, (Number(page) - 1) * Number(limit));

    return ordenada.slice(inicio, inicio + Number(limit));
  } catch (err) {
    console.error('[comprobantesApiService ❌ Fetch Error]:', err.message);
    return [];
  }
}

/**
 * Guarda cambios de un comprobante editado (PUT directo al contrato canónico).
 */
export async function guardarCambiosComprobante(hashLargo, payload = {}, loteActivo = 'T070') {
  try {
    const idValido = obtenerIdentificadorValido(hashLargo, payload);
    if (!idValido) {
      throw new Error('No se proporcionó un hash o ID válido para actualizar');
    }

    const dataPayload = (typeof hashLargo === 'object' && hashLargo !== null) ? hashLargo : payload;
    const body = { ...dataPayload, loteActivo };
    if (!body.lote_tasa_asignado && !body.lote_tasa) body.lote_tasa_asignado = loteActivo;

    const res = await fetch(`${API_URL}/${encodeURIComponent(idValido)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });

    if (!res.ok) throw new Error(`HTTP Error: ${res.status}`);
    return await res.json().catch(() => true);
  } catch (err) {
    console.error('[comprobantesApiService ❌ Save Error]:', err.message);
    throw err;
  }
}

/**
 * Solicita relectura con IA.
 */
export async function solicitarRelecturaIA(hashLargo) {
  try {
    const idValido = obtenerIdentificadorValido(hashLargo);
    if (!idValido) {
      throw new Error('No se proporcionó un hash o ID válido para releer con IA');
    }

    const res = await fetch(`${API_URL}/${encodeURIComponent(idValido)}/releer`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });

    if (!res.ok) throw new Error(`HTTP Error: ${res.status}`);
    return await res.json().catch(() => true);
  } catch (err) {
    console.error('[comprobantesApiService ❌ IA Re-read Error]:', err.message);
    throw err;
  }
}

/**
 * Elimina un comprobante por su hash o ID.
 */
export async function eliminarComprobantePorHash(hashLargo) {
  try {
    const idValido = obtenerIdentificadorValido(hashLargo);
    if (!idValido) {
      throw new Error('No se proporcionó un hash o ID válido para eliminar');
    }

    const res = await fetch(`${API_URL}/${encodeURIComponent(idValido)}`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' }
    });

    if (!res.ok) throw new Error(`HTTP Error: ${res.status}`);
    return true;
  } catch (err) {
    console.error('[comprobantesApiService ❌ Delete Error]:', err.message);
    throw err;
  }
}

export {
  guardarCambiosComprobante as actualizarComprobanteApi,
  solicitarRelecturaIA as releerIaApi
};
