// =================================================================
// ARCHIVO: comprobantesFormatters.js
// RESPONSABILIDAD: Formateadores de moneda, tasas e insignias
// =================================================================

export const comprobantesFormatters = {
  formatMonto(val) {
    if (val === null || val === undefined || isNaN(val) || val === '') return '0.00';
    const num = Math.abs(parseFloat(val));
    if (num === 0) return '0.00';
    const parts = num.toFixed(2).split('.');
    return `${Number(parts[0]).toLocaleString('en-US')}.${parts[1]}`;
  },

  formatTasa(val) {
    if (val === null || val === undefined || isNaN(val) || val === '') return '-';
    const num = Math.abs(parseFloat(val));
    if (isNaN(num) || num === 0) return '0';
    const vRound = Math.round(num * 1e8) / 1e8;

    if (vRound > 99.99) {
      return Math.trunc(vRound).toLocaleString('en-US');
    } else if (vRound >= 10.0) {
      const resNum = Math.trunc((vRound + 0.0000001) * 100) / 100;
      return resNum.toFixed(2).replace(/\.?0+$/, "");
    } else {
      const magnitud = Math.floor(Math.log10(vRound));
      const factor = Math.pow(10, 2 - magnitud);
      const resNum = Math.trunc((vRound + 0.0000001) * factor) / factor;
      return resNum.toString();
    }
  },

  formatFechaVE(ts) {
    if (!ts) return '-';
    const date = new Date(ts * 1000);
    return date.toLocaleString('es-VE', {
      timeZone: 'America/Caracas',
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
    });
  },

  claseInsignia(codigo) {
    const mapa = {
      'D': 'bg-cyan-500/20 text-cyan-300 border-cyan-500/40',
      'P': 'bg-amber-500/20 text-amber-300 border-amber-500/40',
      'A': 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40',
      'C': 'bg-purple-500/20 text-purple-300 border-purple-500/40'
    };
    return mapa[codigo] || 'bg-slate-500/20 text-slate-300 border-slate-500/40';
  }
};
