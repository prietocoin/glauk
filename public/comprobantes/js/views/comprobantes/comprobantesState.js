// =================================================================
// ARCHIVO: comprobantesState.js
// RESPONSABILIDAD: Estado reactivo y sincronización con AteneaAPI
// =================================================================

export const comprobantesState = {
  items: [],
  socios: [],
  directorio: [],
  saldoAnteriorReporte: 0.00,

  // Filtros de Auditoría
  filtroRol: '',
  filtroSocio: '',
  filtroFechaInicio: '',
  filtroFechaFin: '',
  filtroDesdeHash: '',
  filtroHastaHash: '',
  filtroHash: '',
  filtroOrden: 'fecha_desc',
  soloDuplicados: false,

  // Modales
  modalEdicionAbierto: false,
  itemEdicion: null,
  modalVistaPreviaAbierto: false,
  enviandoReporte: false,

  async init() {
    await this.cargarDirectorio();
    await this.cargarSocios();
    await this.cargarComprobantes();
  },

  async cargarComprobantes() {
    try {
      const filtros = {
        socio: this.filtroSocio,
        rol: this.filtroRol,
        fechaInicio: this.filtroFechaInicio,
        fechaFin: this.filtroFechaFin,
        desdeHash: this.filtroDesdeHash,
        hastaHash: this.filtroHastaHash,
        hash: this.filtroHash,
        soloDuplicados: this.soloDuplicados
      };

      if (this.filtroSocio) {
        const socioNorm = this.filtroSocio.trim().toUpperCase();
        const reg = (this.directorio || []).find(d => d.nombre && d.nombre.trim().toUpperCase() === socioNorm);
        if (reg) {
          const valSaldo = reg.saldo_inicial ?? reg.saldo_anterior ?? 0;
          this.saldoAnteriorReporte = parseFloat(valSaldo) || 0;
        }
      }

      this.items = await window.AteneaAPI.getComprobantes(filtros);
    } catch (err) {
      console.error('[Glaukov Comprobantes ❌]', err);
    }
  },

  async cargarSocios() {
    try {
      const res = await window.AteneaAPI.getSocios();
      this.socios = Array.isArray(res) ? res.map(s => s.nombre || s) : [];
    } catch (err) {
      console.error('[Glaukov Socios ❌]', err);
    }
  },

  async cargarDirectorio() {
    try {
      this.directorio = await window.AteneaAPI.getDirectorio();
    } catch (err) {
      console.error('[Glaukov Directorio ❌]', err);
    }
  }
};
