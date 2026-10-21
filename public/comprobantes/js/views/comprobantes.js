import { qs, clear, el, escapeHTML } from '../../../shared/js/dom.js';
import { formatDate, formatMoney, truncate } from '../../../shared/js/format.js';
import { abrirModal } from '../components/modal.js';
import { renderTabla, celda, boton } from '../components/tabla.js';

const PAGE_SIZE = 25;

export function crearVistaComprobantes(ctx) {
  let mounted = false;

  function filtros() {
    return {
      socio: qs('#filtro-socio')?.value.trim() || '',
      rol: qs('#filtro-rol')?.value.trim() || '',
      hash: qs('#filtro-hash')?.value.trim() || '',
      fechaInicio: qs('#filtro-desde')?.value || '',
      fechaFin: qs('#filtro-hasta')?.value || '',
      orden: qs('#filtro-orden')?.value || 'fecha_desc'
    };
  }

  async function cargar() {
    const body = qs('#comprobantes-tabla');
    if (body) body.append(el('tr', {}, el('td', { colspan: '7', class: 'muted' }, 'Cargando movimientos…')));
    try {
      ctx.state.filters = filtros();
      ctx.state.items = await ctx.api.listar(ctx.state.filters);
      ctx.state.page = 1;
      render();
    } catch (err) {
      clear(body);
      body.append(el('tr', {}, el('td', { colspan: '7', class: 'muted' }, err.message || 'No se pudieron cargar los comprobantes.')));
      ctx.notify.error(err.message || 'No se pudieron cargar los comprobantes.');
    }
  }

  function renderKpis() {
    const items = ctx.state.items || [];
    const procesados = items.filter((item) => item.procesado_ia).length;
    const total = items.reduce((sum, item) => sum + Number(item.monto || 0), 0);
    const socios = new Set(items.map((item) => item.nombre_socio_1).filter(Boolean)).size;
    const host = qs('#comprobantes-kpis');
    if (!host) return;
    clear(host);
    const kpis = [
      ['Total', items.length], ['Procesados IA', procesados], ['Pendientes', items.length - procesados],
      ['Monto agregado', formatMoney(total)], ['Socios', socios]
    ];
    for (const [label, value] of kpis) {
      host.append(el('div', { class: 'kpi' }, el('div', { class: 'kpi-label' }, label), el('div', { class: 'kpi-value' }, String(value))));
    }
    const count = qs('#comprobantes-count');
    if (count) count.textContent = `${items.length} registros`;
  }

  function acciones(item) {
    const wrap = el('div', { class: 'table-actions' });
    wrap.append(
      boton('Editar', () => abrirEdicion(item), { className: 'btn btn-secondary btn-sm' }),
      boton('Releer', () => releer(item), { className: 'btn btn-ghost btn-sm' }),
      boton('Eliminar', () => eliminar(item), { className: 'btn btn-danger btn-sm' })
    );
    return wrap;
  }

  function render() {
    renderKpis();
    const items = (ctx.state.items || []).slice(0, ctx.state.page * PAGE_SIZE);
    renderTabla(qs('#comprobantes-tabla'), items, {
      colspan: 7, vacio: 'No hay comprobantes para estos filtros.', render: (item) => el('tr', {},
        celda(formatDate(item.fecha_hora_comprobante, true)),
        celda(`${item.nombre_socio_1 || '—'}${item.nombre_socio_2 ? ` / ${item.nombre_socio_2}` : ''}`),
        celda(`${item.tipo_op1 || '—'} · ${item.moneda_socio_1 || item.moneda || ''}`),
        celda(formatMoney(item.monto, item.moneda)),
        celda(truncate(item.hash_largo || item.hash_corto, 18), { className: 'mono', titulo: item.hash_largo || '' }),
        celda(el('span', { class: `badge ${item.procesado_ia ? 'badge-green' : 'badge-amber'}` }, item.procesado_ia ? 'Procesado' : 'Pendiente')),
        celda(acciones(item))
      )
    });
    const total = ctx.state.items?.length || 0;
    if (qs('#comprobantes-pie')) qs('#comprobantes-pie').textContent = `${items.length} de ${total} movimientos`;
    qs('#btn-mas')?.classList.toggle('hidden', items.length >= total);
    const estado = qs('#filtros-estado');
    if (estado) {
      const activos = Object.entries(ctx.state.filters || {}).filter(([, value]) => value).length;
      estado.textContent = activos ? `${activos} filtro(s) aplicados.` : 'Sin filtros aplicados.';
    }
  }

  function abrirEdicion(item) {
    const contenido = `<div class="grid cols-2">
      <label class="field"><span class="label">Monto</span><input class="input" id="edit-monto" type="number" step="0.01" value="${Number(item.monto || 0)}"></label>
      <label class="field"><span class="label">Moneda</span><input class="input" id="edit-moneda" value="${escapeHTML(item.moneda || '')}"></label>
      <label class="field"><span class="label">Banco</span><input class="input" id="edit-banco" value="${escapeHTML(item.banco || '')}"></label>
      <label class="field"><span class="label">Titular</span><input class="input" id="edit-titular" value="${escapeHTML(item.titular || '')}"></label>
      <label class="field"><span class="label">Referencia</span><input class="input" id="edit-referencia" value="${escapeHTML(item.referencia || '')}"></label>
      <label class="field"><span class="label">Socio 1</span><input class="input" id="edit-socio1" value="${escapeHTML(item.nombre_socio_1 || '')}"></label>
      <label class="field"><span class="label">Socio 2</span><input class="input" id="edit-socio2" value="${escapeHTML(item.nombre_socio_2 || '')}"></label>
      <label class="field"><span class="label">Tipo operación</span><input class="input" id="edit-tipo" value="${escapeHTML(item.tipo_op1 || '')}"></label>
    </div>`;
    const modal = abrirModal({
      titulo: `Editar ${truncate(item.hash_largo, 20)}`, contenido,
      acciones: [
        { label: 'Cancelar', onClick: () => modal?.close() },
        { label: 'Guardar cambios', className: 'btn-success', onClick: () => guardarEdicion(item, modal) }
      ]
    });
  }

  async function guardarEdicion(item, modal) {
    const leer = (id) => qs(id)?.value.trim() ?? '';
    const payload = {
      monto: Number(leer('#edit-monto') || 0),
      moneda: leer('#edit-moneda'),
      banco: leer('#edit-banco'),
      titular: leer('#edit-titular'),
      referencia: leer('#edit-referencia'),
      nombre_socio_1: leer('#edit-socio1'),
      nombre_socio_2: leer('#edit-socio2'),
      tipo_op1: leer('#edit-tipo'),
      lote_tasa_asignado: item.lote_tasa_asignado || ''
    };
    try {
      await ctx.api.actualizar(item.hash_largo, payload);
      ctx.notify.success('Comprobante actualizado.');
      modal?.close();
      await cargar();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo actualizar el comprobante.');
    }
  }

  async function releer(item) {
    if (!globalThis.confirm('¿Marcar este comprobante para reprocesamiento IA?')) return;
    try {
      await ctx.api.releer(item.hash_largo);
      ctx.notify.success('Re-lectura IA marcada.');
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo marcar la re-lectura.');
    }
  }

  async function eliminar(item) {
    if (!globalThis.confirm('¿Eliminar este comprobante? La acción no se puede deshacer.')) return;
    try {
      await ctx.api.eliminar(item.hash_largo);
      ctx.notify.success('Comprobante eliminado.');
      await cargar();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo eliminar el comprobante.');
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#filtros-form')?.addEventListener('submit', (event) => { event.preventDefault(); cargar(); });
    qs('#btn-refrescar')?.addEventListener('click', cargar);
    qs('#btn-limpiar-filtros')?.addEventListener('click', () => {
      qs('#filtros-form')?.reset();
      cargar();
    });
    qs('#btn-mas')?.addEventListener('click', () => { ctx.state.page += 1; render(); });
  }

  async function show() {
    if (!ctx.state.items.length) await cargar();
    else render();
  }

  return { mount, show, cargar, render };
}
