/**
 * @file comprobantesLecturaService.js
 * @path public/js/modules/comprobantes/services/comprobantesLecturaService.js
 * @description Servicio atómico de lectura HTTP para comprobantes_final con mapeo plano reactivo.
 */

export async function obtenerComprobantes(params = {}) {
  try {
    const query = new URLSearchParams();

    if (params.socio && params.socio.toUpperCase() !== 'TODOS' && params.socio.toUpperCase() !== 'TODOS LOS SOCIOS') {
      query.append('socio', params.socio.trim());
    }

    if (params.rol && params.rol.toUpperCase() !== 'TODOS') query.append('rol', params.rol.trim());

    // Soporte transparente para filtroFechaInicio o filtroFecha
    const fInicio = params.fechaInicio || params.fecha;
    if (fInicio) query.append('fechaInicio', fInicio.trim());
    if (params.fechaFin) query.append('fechaFin', params.fechaFin.trim());

    if (params.desdeHash) query.append('desdeHash', params.desdeHash.trim());
    if (params.hastaHash) query.append('hastaHash', params.hastaHash.trim());
    if (params.hash) query.append('hash', params.hash.trim());
    if (params.orden) query.append('orden', params.orden.trim());

    // La lectura usa el alias unificado del backend (API v1).
    const response = await fetch(`/api/comprobantes?${query.toString()}`);
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);

    const rawData = await response.json();
    const lista = Array.isArray(rawData) ? rawData : (rawData.data || [rawData]);

    // 🟢 MAPEO DIRECTO DESDE COMPROBANTES_FINAL
    return lista.map((item) => {
      const c = item.comprobante || item;
      const g1 = c.grupo_1 || {};
      const g2 = c.grupo_2 || {};

      return {
        ...c,
        id: c.hash_largo || c.id,
        hash_largo: c.hash_largo || c.id,
        hash_corto: c.hash_corto || (c.hash_largo ? c.hash_largo.substring(0, 8) : ''),
        lote_tasa: c.lote || c.lote_tasa_asignado,
        lote_tasa_asignado: c.lote_tasa_asignado || c.lote,
        monto: c.monto,
        moneda: c.moneda,
        banco: c.banco,
        titular: c.titular,
        referencia: c.referencia,
        tipo_op1: c.tipo || c.tipo_manual,
        naturaleza: c.tipo || c.tipo_manual,
        link_img: c.url_r2_comprobante || c.link_img,

        // Mapeo Plano Grupo 1 (prioriza columna plana de comprobantes_final)
        nombre_socio_1: c.nombre_socio_1 || c.socio_1 || g1.nombre_1 || g1.grupo,
        socio_1: c.socio_1 || c.nombre_socio_1 || g1.nombre_1 || g1.grupo,
        rol_1: c.rol_1 || g1.rol || 'SOCIO',
        tasa_1: c.tasa_1 ?? g1.tasa_1 ?? g1.tasa ?? 0,
        polaridad_1: c.polaridad_1 || g1.polaridad_1 || g1.polaridad || '+',
        monto_1: c.monto_1 ?? c.m1_socio ?? g1.monto_1 ?? g1.monto ?? 0,
        m1_socio: c.m1_socio ?? c.monto_1 ?? g1.monto_1 ?? g1.monto ?? 0,
        moneda_base_socio1: c.moneda_base_socio1 || g1.moneda_base_1 || g1.moneda_base || 'USDT',
        me1: c.me1 ?? g1.me_1 ?? g1.me ?? 0,

        // Mapeo Plano Grupo 2 (prioriza columna plana de comprobantes_final)
        nombre_socio_2: c.nombre_socio_2 || c.socio_2 || g2.nombre_2 || g2.grupo,
        socio_2: c.socio_2 || c.nombre_socio_2 || g2.nombre_2 || g2.grupo,
        rol_2: c.rol_2 || g2.rol || 'SOCIO',
        tasa_2: c.tasa_2 ?? g2.tasa_2 ?? g2.tasa ?? 0,
        polaridad_2: c.polaridad_2 || g2.polaridad_2 || g2.polaridad || '-',
        monto_2: c.monto_2 ?? c.m2_socio ?? g2.monto_2 ?? g2.monto ?? 0,
        m2_socio: c.m2_socio ?? c.monto_2 ?? g2.monto_2 ?? g2.monto ?? 0,
        moneda_base_socio2: c.moneda_base_socio2 || g2.moneda_base_2 || g2.moneda_base || 'USDT',
        me2: c.me2 ?? g2.me_2 ?? g2.me ?? 0
      };
    });
  } catch (error) {
    console.error('[comprobantesLecturaService ❌ Error al obtener comprobantes]:', error);
    return [];
  }
}
