/**
 * @file hooApiService.js
 * @description Cliente atómico para capturar borradores de tasas enviados por n8n / Hoo API.
 * Normaliza automáticamente todas las claves de divisas a MAYÚSCULAS.
 */

/**
 * Conecta con la API backend de Hoo y retorna el borrador de cotizaciones listo para publicar.
 * @returns {Promise<Object|null>} Objeto de tasas normalizado o null si falla.
 */
export async function capturarBorradorHoo() {
  try {
    const response = await fetch('/api/tasas/hoo-borrador');
    if (!response.ok) throw new Error('No se pudo conectar con la API de Hoo');

    const res = await response.json();
    const rawRates = res?.borrador || res?.rates || res?.rates_draft;
    if (!rawRates) {
      alert('No hay un borrador reciente enviado por n8n / Hoo API.');
      return null;
    }
    const ratesNormalizadas = {};

    Object.keys(rawRates).forEach(k => {
      ratesNormalizadas[k.toUpperCase().trim()] = rawRates[k];
    });

    alert('✅ Borrador capturado e inyectado con éxito.');
    return ratesNormalizadas;
  } catch (err) {
    console.error('[hooApiService ❌ Error al capturar Hoo]:', err);
    alert('Error conectando con la API de Hoo: ' + err.message);
    return null;
  }
}
