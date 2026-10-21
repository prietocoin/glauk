/**
 * @file saldosCalculatorService.js
 * @description Servicio matemático puro para cálculo algebraico de saldos y movimientos desde comprobantes_final.
 */
import { filtrarComprobantesPorRango } from '../utils/saldosFilterUtils.js';

/**
 * Suma o resta los movimientos de un socio en específico considerando su polaridad.
 */
export function calcularMovimientoFiltradoTotal(comprobantes = [], socioTarget = '') {
  if (!Array.isArray(comprobantes)) return 0;
  const target = (socioTarget || '').trim().toUpperCase();

  return comprobantes.reduce((sum, item) => {
    const s1 = (item.nombre_socio_1 || item.socio_1 || '').trim().toUpperCase();
    const s2 = (item.nombre_socio_2 || item.socio_2 || '').trim().toUpperCase();

    let val = 0;
    let pol = '+';

    if (target && s2 === target && s1 !== target) {
      val = parseFloat(item.m2_socio ?? item.monto_2 ?? 0) || 0;
      pol = item.polaridad_2 || item.grupo_2?.polaridad || '-';
    } else {
      val = parseFloat(item.m1_socio ?? item.monto_1 ?? item.monto ?? 0) || 0;
      pol = item.polaridad_1 || item.grupo_1?.polaridad || '+';
    }

    const signo = pol === '-' ? -1 : 1;
    return sum + (val * signo);
  }, 0);
}

/**
 * Resumen consolidado de socios con saldos pendientes activos.
 */
export function calcularSociosPendientesConsolidado(directorio = [], comprobantes = [], filtros = {}) {
  if (!Array.isArray(directorio)) return [];

  const compFiltrados = filtrarComprobantesPorRango(comprobantes, filtros);

  return directorio
    .filter(socio => {
      let mostrarObj = socio.mostrar;
      if (typeof mostrarObj === 'string') {
        try { mostrarObj = JSON.parse(mostrarObj); } catch (e) { mostrarObj = {}; }
      }
      return (mostrarObj?.dashboard ?? true) !== false;
    })
    .map(socio => {
      const nombreUpper = (socio.nombre || '').trim().toUpperCase();
      const saldoBase = parseFloat(socio.saldo_inicial ?? socio.saldo_anterior) || 0;

      const movimientoHistorico = compFiltrados.reduce((acc, item) => {
        const s1 = (item.nombre_socio_1 || item.socio_1 || '').trim().toUpperCase();
        const s2 = (item.nombre_socio_2 || item.socio_2 || '').trim().toUpperCase();

        if (s1 === nombreUpper) {
          const val1 = parseFloat(item.m1_socio ?? item.monto_1 ?? item.monto) || 0;
          const pol1 = item.polaridad_1 || item.grupo_1?.polaridad || '+';
          const signo1 = pol1 === '-' ? -1 : 1;
          return acc + (val1 * signo1);
        } else if (s2 === nombreUpper) {
          const val2 = parseFloat(item.m2_socio ?? item.monto_2) || 0;
          const pol2 = item.polaridad_2 || item.grupo_2?.polaridad || '-';
          const signo2 = pol2 === '-' ? -1 : 1;
          return acc + (val2 * signo2);
        }
        return acc;
      }, 0);

      const saldoFinal = Math.trunc((saldoBase + movimientoHistorico + 0.0000001) * 100) / 100;

      return {
        nombre: socio.nombre,
        moneda: (socio.moneda_base || socio.moneda_socio || 'USDT').toUpperCase(),
        saldoBase,
        movimientoHistorico,
        saldoFinal
      };
    })
    .filter(s => Math.abs(s.saldoFinal) >= 0.01);
}
