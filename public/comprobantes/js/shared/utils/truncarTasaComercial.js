/**
 * @file truncarTasaComercial.js
 * @description Truncado comercial de precisión para divisas en glauk2.
 * Formatea valores numéricos aplicando truncado a 2 decimales para valores <= 99.99
 * y truncado a entero para valores superiores.
 *
 * @param {number|string} valor - Valor numérico o string a truncar.
 * @returns {number} Valor truncado según la regla comercial.
 */
export function truncarTasaComercial(valor) {
  const num = Math.abs(parseFloat(valor) || 0);
  if (num === 0) return 1.0;
  if (num > 99.99) {
    return Math.trunc(num);
  }
  return Math.trunc((num + 0.0000001) * 100) / 100;
}
