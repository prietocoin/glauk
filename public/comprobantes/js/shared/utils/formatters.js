/**
 * @file formatters.js
 * @description Utilidades puras de precisión matemática y formateo para montos y tasas.
 */

/**
 * Ajusta la precisión decimal de un monto truncando a 2 decimales sin errores de coma flotante.
 * @param {number|string} val - Monto numérico o string.
 * @returns {number}
 */
export function aplicarPrecisionMonto(val) {
  if (!val || isNaN(val)) return 0;
  const num = parseFloat(val);
  const signo = num < 0 ? -1 : 1;
  const vRound = Math.round(Math.abs(num) * 1e8) / 1e8;
  return signo * (Math.trunc((vRound + 1e-7) * 100) / 100);
}

/**
 * Aplica formato localizado (es-ES) con exactamente 2 decimales a un monto.
 * @param {number|string} val
 * @returns {string}
 */
export function formatMonto(val) {
  const num = aplicarPrecisionMonto(val);
  return num.toLocaleString('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

/**
 * Aplica la regla de truncamiento de significancia según la magnitud de la tasa.
 * @param {number|string} val
 * @returns {number}
 */
export function aplicarReglaPrecisionTasa(val) {
  if (!val || isNaN(val)) return 0;
  const num = Math.abs(parseFloat(val));
  const vRound = Math.round(num * 1e8) / 1e8;
  if (vRound === 0) return 0;

  const signo = parseFloat(val) < 0 ? -1 : 1;
  if (vRound > 99.99) return signo * Math.trunc(vRound);
  if (vRound >= 10.0) return signo * (Math.trunc((vRound + 1e-7) * 100) / 100);

  const factor = Math.pow(10, 2 - Math.floor(Math.log10(vRound)));
  return signo * (Math.trunc((vRound + 1e-7) * factor) / factor);
}
