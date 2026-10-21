/**
 * @file directorioView.js
 * @description Controlador Alpine.js para la vista del directorio de socios en glauk2.
 */
import { obtenerDirectorioNormalizado } from '../services/directorioService.js';
import { alternarEstadoSocioWA } from '../services/socioEstadoService.js';
import { alternarHerenciaSocio } from '../services/socioHerenciaService.js';

export function directorioView() {
  return {
    socios: [],
    busqueda: '',
    loading: false,

    async cargar() {
      this.loading = true;
      try {
        this.socios = await obtenerDirectorioNormalizado();
      } catch (err) {
        console.error('[directorioView ❌ Error al cargar]:', err.message);
      } finally {
        this.loading = false;
      }
    },

    get filtrados() {
      if (!this.busqueda.trim()) return this.socios;
      const q = this.busqueda.toLowerCase();

      return this.socios.filter(s =>
        (s.nombre && s.nombre.toLowerCase().includes(q)) ||
        ((s.rol || s.roles) && (s.rol || s.roles).toLowerCase().includes(q)) ||
        ((s.id_grupo || s.whatsapp) && (s.id_grupo || s.whatsapp).toLowerCase().includes(q))
      );
    },

    async toggleEstado(socio) {
      await alternarEstadoSocioWA(socio);
    },

    async toggleHerencia(socio) {
      await alternarHerenciaSocio(socio);
    }
  };
}
