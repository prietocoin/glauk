/**
 * @file comprobantesState.js
 * Estado reactivo Alpine.js con Paginación Real Servidor + Buffer Incremental desde comprobantes_final.
 */

import {
  formatMonto, formatTasa, obtenerME1, obtenerME2,
  obtenerTasaSocioCalculada, claseInsignia, formatDiaHora
} from '../utils/comprobantesFormatters.js';

import {
  calcularSociosPendientesConsolidado,
  calcularMovimientoFiltradoTotal
} from '../services/saldosCalculatorService.js';

import { fetchComprobantesPaginados, eliminarComprobantePorHash } from '../services/comprobantesApiService.js';
import { crearAccionesModal } from '../services/comprobantesModalActions.js';
import { prepararEdicionComprobante, obtenerInsigniaRol } from '../services/comprobantesMapperService.js';
import { obtenerSaldoAnteriorSocio, obtenerSujetoAuditado } from '../services/comprobantesSaldosService.js';

function obtenerRangoMesActualCaracas() {
  const timeZone = 'America/Caracas';
  const ahora = new Date();
  const fmt = new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit' });

  const primerDia = new Date(ahora.getFullYear(), ahora.getMonth(), 1);
  const fechaInicio = fmt.format(primerDia);

  const ultimoDia = new Date(ahora.getFullYear(), ahora.getMonth() + 1, 0);
  const fechaFin = fmt.format(ultimoDia);

  return { fechaInicio, fechaFin };
}

export function comprobantesState() {
  const { fechaInicio: inicioMes, fechaFin: finMes } = obtenerRangoMesActualCaracas();

  const state = {
    paginaAPI: 1,
    limitAPI: 50,
    todosLosItems: [],
    items: [],
    directorio: [],
    cargando: false,
    hayMasDatos: true,

    tamanoLote: 20,
    capMaximoDOM: 300,

    modalAbierto: false,
    itemEdicion: null,
    loteActivo: 'T070',

    filtroRol: '',
    filtroSocio: '',
    filtroFechaInicio: inicioMes,
    filtroFechaFin: finMes,
    filtroDesdeHash: '',
    filtroHastaHash: '',
    filtroOrden: 'fecha_desc',
    filtroHash: '',
    saldoAnterior: 0,

    obtenerSujetoAuditado,
    obtenerInsigniaRol,

    get comprobantes() {
      return this.items;
    },

    get movimientoFiltradoTotal() {
      return calcularMovimientoFiltradoTotal(this.items, this.filtroSocio);
    },

    get saldoActualTotal() {
      return (parseFloat(this.saldoAnterior) || 0) + this.movimientoFiltradoTotal;
    },

    get sociosPendientesConsolidado() {
      return calcularSociosPendientesConsolidado(this.directorio, this.items, {
        fechaInicio: this.filtroFechaInicio, fechaFin: this.filtroFechaFin
      });
    },

    init() {
      this.$nextTick(() => {
        this.escucharActualizacionesGlobales();
        this.resetearYRefrescar();
        this.iniciarEscuchadoresScroll();
      });
    },

    escucharActualizacionesGlobales() {
      window.addEventListener('comprobante-actualizado', (evt) => {
        const payload = evt.detail;
        if (!payload || !payload.hash_largo) return;

        const targetHash = String(payload.hash_largo).trim().toLowerCase();

        const idxTodos = this.todosLosItems.findIndex(i => String(i.hash_largo || i.id).trim().toLowerCase() === targetHash);
        if (idxTodos !== -1) {
          this.todosLosItems[idxTodos] = prepararEdicionComprobante({
            ...this.todosLosItems[idxTodos],
            ...payload
          }, this.loteActivo);
        }

        const idxVisibles = this.items.findIndex(i => String(i.hash_largo || i.id).trim().toLowerCase() === targetHash);
        if (idxVisibles !== -1) {
          this.items[idxVisibles] = prepararEdicionComprobante({
            ...this.items[idxVisibles],
            ...payload
          }, this.loteActivo);

          this.items = [...this.items];
        } else if (idxTodos === -1) {
          this.resetearYRefrescar();
        }
      });
    },

    actualizarSocioSeleccionado() {
      this.saldoAnterior = obtenerSaldoAnteriorSocio(this.directorio, this.filtroSocio);
      this.resetearYRefrescar();
    },

    async cargarComprobantes(silencioso = false) {
      return this.resetearYRefrescar();
    },

    async resetearYRefrescar() {
      this.items = [];
      this.todosLosItems = [];
      this.paginaAPI = 1;
      this.hayMasDatos = true;
      this.cargando = false;
      return this.cargarSiguientePagina();
    },

    async cargarSiguientePagina() {
      if (this.cargando || !this.hayMasDatos) return;
      this.cargando = true;

      try {
        if (this.items.length + this.tamanoLote > this.todosLosItems.length) {
          const params = {
            page: this.paginaAPI,
            limit: this.limitAPI,
            socio: this.filtroSocio,
            rol: this.filtroRol,
            fechaInicio: this.filtroFechaInicio,
            fechaFin: this.filtroFechaFin,
            desdeHash: this.filtroDesdeHash,
            hastaHash: this.filtroHastaHash,
            orden: this.filtroOrden,
            hash: this.filtroHash
          };

          const raw = await fetchComprobantesPaginados(params);
          const lista = Array.isArray(raw) ? raw : (raw?.datos || raw?.comprobantes || raw?.data || []);

          if (lista.length === 0) {
            this.hayMasDatos = false;
          } else {
            const mapeados = lista.map(item => prepararEdicionComprobante(item, this.loteActivo));

            const idsExistentes = new Set(this.todosLosItems.map(i => String(i.id || i.hash_largo)));
            const nuevosUnicos = mapeados.filter(i => !idsExistentes.has(String(i.id || i.hash_largo)));

            this.todosLosItems = [...this.todosLosItems, ...nuevosUnicos];
            this.paginaAPI++;

            if (lista.length < this.limitAPI) {
              this.hayMasDatos = false;
            }
          }
        }

        const proximoLimite = Math.min(
          this.items.length + this.tamanoLote,
          this.capMaximoDOM,
          this.todosLosItems.length
        );

        this.items = [...this.todosLosItems.slice(0, proximoLimite)];

        if (this.items.length >= this.capMaximoDOM) {
          this.hayMasDatos = false;
        }

      } catch (err) {
        console.error('❌ Error al cargar lote de comprobantes:', err);
      } finally {
        setTimeout(() => {
          this.cargando = false;
        }, 100);
      }
    },

    iniciarEscuchadoresScroll() {
      this.$nextTick(() => {
        const intentarCargar = () => {
          if (this.cargando || !this.hayMasDatos) return;

          const sentinel = document.getElementById('sentinel-scroll');
          if (!sentinel) return;

          const rect = sentinel.getBoundingClientRect();
          const vh = window.innerHeight || document.documentElement.clientHeight;

          if (rect.top <= vh + 600) {
            this.cargarSiguientePagina();
          }
        };

        document.addEventListener('scroll', intentarCargar, { capture: true, passive: true });
        window.addEventListener('scroll', intentarCargar, { passive: true });
        document.addEventListener('wheel', intentarCargar, { capture: true, passive: true });
        document.addEventListener('touchmove', intentarCargar, { capture: true, passive: true });

        setInterval(intentarCargar, 250);

        const sentinel = document.getElementById('sentinel-scroll');
        if (sentinel) {
          const observer = new IntersectionObserver((entries) => {
            if (entries[0].isIntersecting) {
              intentarCargar();
            }
          }, { rootMargin: '400px' });

          observer.observe(sentinel);
        }
      });
    },

    async enviarReporteViaEvolution(socio = '') {
      const documento = document.getElementById('reporte-html-documento');
      if (!documento) {
        alert('No se encontró el documento del reporte.');
        return;
      }
      if (typeof html2canvas !== 'function') {
        alert('El generador de imágenes no está disponible.');
        return;
      }

      try {
        const canvas = await html2canvas(documento, {
          backgroundColor: '#ffffff',
          scale: 2,
          useCORS: true,
          logging: false
        });
        const base64 = canvas.toDataURL('image/jpeg', 0.92);
        const response = await fetch('/api/v1/reportes/imagen', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            socio: String(socio || '').trim() || 'GENERAL',
            caption: 'Estado de cuenta FUNDABLOCK',
            fileName: `estado-cuenta-${Date.now()}.jpg`,
            mimeType: 'image/jpeg',
            base64
          })
        });
        const res = await response.json().catch(() => ({}));
        if (!response.ok || res.success === false) {
          throw new Error(res.error || res.message || `HTTP ${response.status}`);
        }
        const detalle = res.data || res;
        alert(detalle.simulated
          ? '🧪 Reporte generado en modo simulación (Evolution no configurado).'
          : '✅ Reporte enviado por Evolution.');
      } catch (err) {
        console.error('[comprobantesState ❌ Error al enviar reporte]:', err);
        alert('❌ Error al enviar el reporte: ' + err.message);
      }
    },

    async eliminarComprobante(identificador) {
      if (!identificador) return;

      if (!confirm('¿Confirmas que deseas eliminar este comprobante?')) return;

      const idStr = String(identificador).trim();

      try {
        await eliminarComprobantePorHash(idStr);
      } catch (err) {
        console.error('❌ Error al eliminar:', err);
      } finally {
        this.items = this.items.filter(item => String(item.id || item.hash_largo || '') !== idStr);
        this.todosLosItems = this.todosLosItems.filter(item => String(item.id || item.hash_largo || '') !== idStr);
      }
    },

    formatMonto, formatTasa, obtenerME1, obtenerME2, obtenerTasaSocioCalculada, claseInsignia, formatDiaHora
  };

  Object.assign(state, crearAccionesModal(state));

  return state;
}
