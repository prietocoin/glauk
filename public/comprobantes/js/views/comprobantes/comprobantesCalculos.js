// =================================================================
// ARCHIVO: comprobantesCalculos.js
// RESPONSABILIDAD: Lógica matemática de tasas y volúmenes ME1 / ME2
// =================================================================

export const comprobantesCalculos = {
  obtenerTasaSocioCalculada(c, numSocio = 1) {
    if (!c) return 1.0;
    const propTasa = numSocio === 2 ? (c.tasa_2 || c.tasa_socio_2) : (c.tasa_1 || c.tasa_socio_1 || c.tasa_socio);
    const valTasa = parseFloat(propTasa || c.tasa_base || c.tasa);
    return (!isNaN(valTasa) && valTasa !== 0) ? valTasa : 1.0;
  },

  obtenerME1(c) {
    if (!c) return 0;
    if (c.m1_socio !== undefined && c.m1_socio !== null && c.m1_socio !== '') {
      return Math.abs(parseFloat(c.m1_socio) || 0);
    }
    if (c.me1 !== undefined && c.me1 !== null && c.me1 !== '') {
      return Math.abs(parseFloat(c.me1) || 0);
    }

    const montoOrigen = Math.abs(parseFloat(c.monto || c.monto_origen || 0));
    const tasa = Math.abs(this.obtenerTasaSocioCalculada(c, 1));
    const moneda = String(c.moneda || c.moneda_comprobante || 'USDT').toUpperCase().trim();

    let equivalente = ['USD', 'USDT', 'PYUSD'].includes(moneda)
      ? (tasa > 0 ? (montoOrigen * tasa) : montoOrigen)
      : (tasa > 0 ? (montoOrigen / tasa) : montoOrigen);

    return parseFloat(Math.abs(equivalente).toFixed(2));
  },

  obtenerME2(c) {
    if (!c) return 0;
    if (c.m2_socio !== undefined && c.m2_socio !== null && c.m2_socio !== '') {
      return Math.abs(parseFloat(c.m2_socio) || 0);
    }
    if (c.me2 !== undefined && c.me2 !== null && c.me2 !== '') {
      return Math.abs(parseFloat(c.me2) || 0);
    }

    const montoOrigen = Math.abs(parseFloat(c.monto || c.monto_origen || 0));
    const tasa = Math.abs(this.obtenerTasaSocioCalculada(c, 2));
    const moneda = String(c.moneda || c.moneda_comprobante || 'USDT').toUpperCase().trim();

    let equivalente = ['USD', 'USDT', 'PYUSD'].includes(moneda)
      ? (tasa > 0 ? (montoOrigen * tasa) : montoOrigen)
      : (tasa > 0 ? (montoOrigen / tasa) : montoOrigen);

    return parseFloat(Math.abs(equivalente).toFixed(2));
  },

 obtenerMontoSocioCalculado(c) {
    if (!c) return 0;
    if (this.filtroSocio) {
      const socioNorm = this.filtroSocio.trim().toUpperCase();
      const s1 = String(c.nombre_socio_1 || c.socio_1 || c.socio || '').trim().toUpperCase();
      const s2 = String(c.nombre_socio_2 || c.socio_2 || '').trim().toUpperCase();

      if (socioNorm === s2) return this.obtenerME2(c);
      if (socioNorm === s1) return this.obtenerME1(c);
      if (c.monto_socio_final !== undefined && c.monto_socio_final !== null) {
        return Math.abs(parseFloat(c.monto_socio_final) || 0);
      }
    }
    return this.obtenerME1(c);
  }
};
