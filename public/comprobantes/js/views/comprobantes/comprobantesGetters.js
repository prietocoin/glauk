export const comprobantesGetters = {
  get comprobantesProcesadosYOrdenados() {
    let list = Array.isArray(this.items) ? [...this.items] : [];
    if (!list.length) return [];

    // 1. Filtro por Rol
    if (this.filtroRol && this.filtroRol !== 'TODOS' && this.filtroRol !== '') {
      const r = this.filtroRol.toUpperCase();
      list = list.filter(i => (i.rol || i.rol_socio_1 || '').toUpperCase() === r);
    }

    // 2. Filtro por Socio / Entidad (Socio 1 o Socio 2)
    if (this.filtroSocio && this.filtroSocio !== 'TODOS' && this.filtroSocio !== '') {
      const s = this.filtroSocio.toLowerCase();
      list = list.filter(i => {
        const s1 = (i.nombre_socio_1 || i.socio_1 || i.socio || '').toLowerCase();
        const s2 = (i.nombre_socio_2 || i.socio_2 || '').toLowerCase();
        return s1.includes(s) || s2.includes(s);
      });
    }

    // 3. Filtro por Rango de Fechas
    if (this.filtroFechaInicio) {
      list = list.filter(i => (i.fecha_hora_comprobante || i.fecha) >= this.filtroFechaInicio);
    }
    if (this.filtroFechaFin) {
      list = list.filter(i => (i.fecha_hora_comprobante || i.fecha) <= this.filtroFechaFin);
    }

    // 4. Búsqueda por Texto (Hash, Titular o Banco)
    if (this.filtroHash && typeof this.filtroHash === 'string' && this.filtroHash.trim() !== '') {
      const q = this.filtroHash.trim().toLowerCase();
      list = list.filter(i =>
        (i.hash_largo || '').toLowerCase().includes(q) ||
        (i.hash_corto || '').toLowerCase().includes(q) ||
        (i.titular || '').toLowerCase().includes(q) ||
        (i.banco || '').toLowerCase().includes(q)
      );
    }

    // 5. Ordenamiento
    const orden = this.filtroOrden || 'fecha_desc';
    list.sort((a, b) => {
      const fA = new Date(a.fecha_hora_comprobante || a.fecha || 0);
      const fB = new Date(b.fecha_hora_comprobante || b.fecha || 0);
      if (orden === 'fecha_desc') return fB - fA;
      if (orden === 'fecha_asc') return fA - fB;
      if (orden === 'monto_desc') return (Number(b.monto || b.me1) || 0) - (Number(a.monto || a.me1) || 0);
      if (orden === 'monto_asc') return (Number(a.monto || a.me1) || 0) - (Number(b.monto || b.me1) || 0);
      return 0;
    });

    return list;
  }
};
