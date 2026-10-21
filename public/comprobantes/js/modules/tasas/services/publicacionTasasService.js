/**
 * @file publicacionTasasService.js
 * @description Servicio para publicar borradores y reenviar lotes de tasas de forma masiva.
 * Maneja las alertas de confirmación para evitar envíos accidentales a producción.
 */

/**
 * Publica el borrador capturado como tasa oficial de mercado.
 * @param {Object} borradorRates - Mapa de tasas capturado.
 * @param {boolean} modoPrueba - Si está activo el modo prueba.
 * @returns {Promise<string|null>} ID de la tasa creada o null si se cancela/falla.
 */
export async function publicarBorradorTasa(borradorRates, modoPrueba) {
  if (!borradorRates || Object.keys(borradorRates).length === 0) {
    alert('No hay borrador capturado para publicar.');
    return null;
  }

  const msg = modoPrueba
    ? '🧪 MODO PRUEBA ACTIVADO\n¿Deseas publicar el borrador y enviarlo AL GRUPO DE PRUEBAS?'
    : '🚀 MODO PRODUCCIÓN ACTIVADO\n¿Deseas publicar el borrador como tasa oficial a TODOS LOS SOCIOS?';

  if (!confirm(msg)) return null;

  try {
    const response = await fetch('/api/tasas/publicar', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tasas: borradorRates,
        modoPrueba
      })
    });

    const res = await response.json();
    if (!response.ok || !res.success) throw new Error(res.error || 'Error al publicar la tasa');

    const idTasa = res?.id_tasa || '';
    alert(`Tasa oficial ${idTasa} publicada. ${modoPrueba ? '🧪 Enviado al grupo de pruebas.' : '🚀 Enviado a producción.'}`);
    return idTasa;
  } catch (err) {
    console.error('[publicacionTasasService ❌ Error al publicar]:', err);
    alert('Error al publicar tasa: ' + err.message);
    return null;
  }
}

/**
 * Reenvía el lote de tasas activo a todos los socios o al grupo de pruebas.
 * @param {string} loteActivo - ID del lote activo.
 * @param {boolean} modoPrueba - Si se debe reenviar en modo prueba.
 */
export async function reenviarLoteCompleto(loteActivo, modoPrueba) {
  if (!loteActivo) {
    alert('No hay un lote activo cargado.');
    return;
  }

  const msg = modoPrueba
    ? `🧪 MODO PRUEBA ACTIVADO\n¿Reenviar notificaciones del lote ${loteActivo} AL GRUPO DE PRUEBAS?`
    : `🚀 MODO PRODUCCIÓN ACTIVADO\n¿Reenviar notificaciones del lote ${loteActivo} a TODOS LOS SOCIOS?`;

  if (!confirm(msg)) return;

  try {
    const response = await fetch('/api/tasas/reenviar', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id_tasa: loteActivo,
        socio: 'GENERAL',
        modoPrueba
      })
    });

    const res = await response.json();
    if (!response.ok || !res.success) throw new Error(res.error || 'Error al reenviar la tasa');

    alert(`Reenvío activado para la tasa ${loteActivo} ${modoPrueba ? '(🧪 Grupo de Prueba)' : '(🚀 Producción)'}.`);
  } catch (err) {
    console.error('[publicacionTasasService ❌ Error al reenviar]:', err);
    alert('Error al reenviar tasa: ' + err.message);
  }
}
