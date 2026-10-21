/**
 * =================================================================
 * @file comprobantesAcciones.js
 * @description Modales de auditoría, edición, borrado y WhatsApp (ESModule).
 * =================================================================
 */

export const comprobantesAcciones = {
  // 1. ABRIR MODAL CON PARSEO ROBUSTO DE FECHA
  abrirModalEdicion(item) {
    if (!item) return;

    let dateInput = item.fecha_hora_input || '';

    // Si no viene en fecha_hora_input, parseamos desde el timestamp o string ISO
    if (!dateInput) {
      const ts = item.timestamp || item.timestamp_comprobante;
      if (ts) {
        const d = typeof ts === 'number' ? new Date(ts * 1000) : new Date(ts);
        if (!isNaN(d.getTime())) {
          const tzOffset = d.getTimezoneOffset() * 60000;
          dateInput = (new Date(d.getTime() - tzOffset)).toISOString().slice(0, 16);
        }
      }
    }

    const loteSeleccionado = item.id_tasa || item.lote_tasa || item.lote_tasa_asignado || 'T052';

    // Se asigna la copia al estado
    this.itemEdicion = {
      ...item,
      id_tasa: loteSeleccionado,
      lote_tasa: loteSeleccionado,
      lote_tasa_asignado: loteSeleccionado,
      tipo_manual: item.tipo_manual || item.tipo_op || item.tipo_op_1 || 'D',
      fecha_hora_input: dateInput
    };

    // Sincronización de ambas banderas de visibilidad para compatibilidad
    this.modalAbierto = true;
    this.modalEdicionAbierto = true;
  },

  // Alias para botones viejos
  abrirModal(item) {
    this.abrirModalEdicion(item);
  },

  cerrarModalEdicion() {
    this.modalAbierto = false;
    this.modalEdicionAbierto = false;
    this.itemEdicion = null;
  },

  // 2. GUARDAR CAMBIOS (Mapeado a 'guardarCambios' y 'guardarEdicionComprobante')
  async guardarCambios() {
    if (!this.itemEdicion || !this.itemEdicion.hash_largo) return;
    try {
      if (this.itemEdicion.fecha_hora_input) {
        const ts = Math.floor(new Date(this.itemEdicion.fecha_hora_input).getTime() / 1000);
        if (!isNaN(ts) && ts > 0) this.itemEdicion.timestamp = ts;
      }

      const payload = {
        ...this.itemEdicion,
        id_tasa: this.itemEdicion.id_tasa || this.itemEdicion.lote_tasa_asignado || 'T052',
        lote_tasa: this.itemEdicion.id_tasa || this.itemEdicion.lote_tasa_asignado || 'T052'
      };

      if (window.AteneaAPI && typeof window.AteneaAPI.actualizarComprobante === 'function') {
        await window.AteneaAPI.actualizarComprobante(payload.hash_largo, payload);
      }

      this.cerrarModalEdicion();
      if (typeof this.cargarComprobantes === 'function') {
        await this.cargarComprobantes();
      }
    } catch (err) {
      alert('Error al guardar comprobante: ' + err.message);
    }
  },

  async guardarEdicionComprobante() {
    await this.guardarCambios();
  },

  // 3. ELIMINAR COMPROBANTE
  async eliminarComprobante(hashLargo) {
    if (!confirm('¿Deseas eliminar este comprobante de la base de datos?')) return;
    try {
      if (window.AteneaAPI && typeof window.AteneaAPI.eliminarComprobante === 'function') {
        await window.AteneaAPI.eliminarComprobante(hashLargo);
      }
      this.cerrarModalEdicion();
      if (typeof this.cargarComprobantes === 'function') {
        await this.cargarComprobantes();
      }
    } catch (err) {
      alert('Error al eliminar comprobante: ' + err.message);
    }
  },

  // 4. ENVÍO WHATSAPP
  async enviarReporteWhatsApp() {
    if (!this.jidSocioActual) {
      alert('El socio seleccionado no posee un Remote JID o ID Grupo de WhatsApp registrado.');
      return;
    }

    this.enviandoReporte = true;
    try {
      await window.AteneaAPI.enviarWhatsApp({
        socio: this.filtroSocio,
        remoteJid: this.jidSocioActual,
        saldoAnterior: this.saldoAnteriorReporte,
        movimiento: this.totalMovimientoFiltrado,
        nuevoSaldo: this.nuevoSaldoTotalCalculado,
        moneda: this.monedaSocioDominante,
        comprobantes: this.comprobantesProcesadosYOrdenados
      });
      alert(`✅ Reporte enviado a WhatsApp (${this.jidSocioActual}) con éxito.`);
    } catch (err) {
      alert('⚠️ Error enviando reporte: ' + err.message);
    } finally {
      this.enviandoReporte = false;
    }
  }
};
