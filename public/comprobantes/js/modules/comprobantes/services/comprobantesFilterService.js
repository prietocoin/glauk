/**
 * =================================================================
 * @file comprobantesFilterService.js
 * @path public/js/modules/comprobantes/services/comprobantesFilterService.js
 * @description Servicio atómico de cortafuegos (Filtro x2 estricto) y ordenamiento para comprobantes_final.
 * =================================================================
 */

export function limpiarFiltro(val) {
  if (!val) return '';
  const str = String(val).trim().toUpperCase();
  return (str === 'TODOS' || str === 'TODOS LOS SOCIOS' || str === 'GENERAL') ? '' : val;
}

export function filtrarComprobantesAtómico(listaBase = [], directorio = [], filtros = {}) {
  if (!Array.isArray(listaBase) || listaBase.length === 0) return [];

  let resultado = [...listaBase];

  // 🟢 0. CORTAFUEGOS OBLIGATORIO x2 (Descartar registros 1x / basura unipersonal)
  resultado = resultado.filter(item => {
    if (!item) return false;

    // Si viene la bandera directa desde comprobantes_final, la evaluamos
    if (typeof item.es_x2 === 'boolean') {
      if (!item.es_x2) return false;
    }

    const s2 = String(item.nombre_socio_2 || item.socio_2 || '').trim().toUpperCase();
    return Boolean(s2 && s2 !== 'GENERAL' && s2 !== 'NO DEFINIDO' && s2 !== 'S/D');
  });

  const socioBuscado = limpiarFiltro(filtros.filtroSocio).toUpperCase();

  // 1. CORTAFUEGOS DE SOCIO (Socio 1 o Socio 2)
  if (socioBuscado) {
    const sociosValidos = new Set([socioBuscado]);
    if (Array.isArray(directorio) && directorio.length > 0) {
      directorio.forEach(d => {
        const padre = String(d.padre || d.herencia || '').trim().toUpperCase();
        const nombre = String(d.nombre || '').trim().toUpperCase();
        if (padre === socioBuscado || nombre === socioBuscado) {
          if (d.nombre) sociosValidos.add(String(d.nombre).trim().toUpperCase());
        }
      });
    }

    resultado = resultado.filter(item => {
      const s1 = String(item.nombre_socio_1 || item.socio_1 || '').trim().toUpperCase();
      const s2 = String(item.nombre_socio_2 || item.socio_2 || '').trim().toUpperCase();
      return sociosValidos.has(s1) || sociosValidos.has(s2);
    });
  }

  // 2. FILTRADO POR FECHA INICIO
  if (filtros.filtroFechaInicio) {
    const startObj = new Date(String(filtros.filtroFechaInicio).trim() + 'T00:00:00');
    if (!isNaN(startObj.getTime())) {
      const startMs = startObj.getTime();

      resultado = resultado.filter(item => {
        const rawDate = item.fecha_hora_iso || item.fecha_fmt || item.fecha_hora_comprobante || item.fecha_hora_input || item.created_at;
        if (!rawDate) return true;
        const itemMs = new Date(rawDate).getTime();
        return !isNaN(itemMs) ? itemMs >= startMs : true;
      });
    }
  }

  // 3. FILTRADO POR FECHA FIN
  if (filtros.filtroFechaFin) {
    const endObj = new Date(String(filtros.filtroFechaFin).trim() + 'T23:59:59');
    if (!isNaN(endObj.getTime())) {
      const endMs = endObj.getTime();

      resultado = resultado.filter(item => {
        const rawDate = item.fecha_hora_iso || item.fecha_fmt || item.fecha_hora_comprobante || item.fecha_hora_input || item.created_at;
        if (!rawDate) return true;
        const itemMs = new Date(rawDate).getTime();
        return !isNaN(itemMs) ? itemMs <= endMs : true;
      });
    }
  }

  // 4. ORDENAMIENTO COMPLETO
  const criterioOrden = String(filtros.filtroOrden || 'fecha_desc').toLowerCase();

  return [...resultado].sort((a, b) => {
    // A) Orden por Monto
    if (criterioOrden === 'monto_desc') {
      return (parseFloat(b.monto) || 0) - (parseFloat(a.monto) || 0);
    }
    if (criterioOrden === 'monto_asc') {
      return (parseFloat(a.monto) || 0) - (parseFloat(b.monto) || 0);
    }

    // B) Orden por Fecha
    const dateA = a.fecha_hora_iso || a.fecha_fmt || a.fecha_hora_comprobante || a.fecha_hora_input || 0;
    const dateB = b.fecha_hora_iso || b.fecha_fmt || b.fecha_hora_comprobante || b.fecha_hora_input || 0;

    const msA = new Date(dateA).getTime() || 0;
    const msB = new Date(dateB).getTime() || 0;

    if (criterioOrden === 'fecha_asc') {
      return msA - msB; // Antiguos primero
    } else {
      return msB - msA; // Recientes primero (Default)
    }
  });
}
