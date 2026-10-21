/**
 * @file tasasState.js
 * @description Estado y acciones del módulo de Tasas de Mercado y Hoo API.
 */

import { calcularTasaEnVivo, obtenerClaseTalla } from '../utils/calculoTasaEnVivo.js';
import { obtenerTasasVigentes } from '../services/tasasMercadoService.js';
import { despacharTasaIndividual } from '../services/despachoTasaIndividualService.js';
import { capturarBorradorHoo } from '../services/hooApiService.js';
import { publicarBorradorTasa, reenviarLoteCompleto } from '../services/publicacionTasasService.js';

export function createTasasState() {
  return {
    modoPruebaActivo: false,
    loteActivo: '',
    tasasProduccion: {},
    borradorCapturado: {},
    historialTasas: [],

    async cargarTasasMercado() {
      const t = await obtenerTasasVigentes();
      this.loteActivo = t?.id_tasa || '';
      this.tasasProduccion = t?.tasas || {};
    },
    async conectarHooAPI() {
      const b = await capturarBorradorHoo();
      if (b) this.borradorCapturado = b;
    },
    async publicarTasaOficial() {
      const id = await publicarBorradorTasa(this.borradorCapturado, this.modoPruebaActivo);
      if (id) this.loteActivo = id;
    },
    async reenviarTasaActual() {
      await reenviarLoteCompleto(this.loteActivo, this.modoPruebaActivo);
    },
    async enviarTasaIndividual(socio, fPrueba = null) {
      await despacharTasaIndividual(this.loteActivo, socio, fPrueba, this.modoPruebaActivo);
    },
    calcularTasaEnVivo(code, pct, esResta) {
      return calcularTasaEnVivo(this.tasasProduccion[code], pct, esResta);
    },
    getTallaClass(socio) {
      return obtenerClaseTalla(socio);
    }
  };
}
