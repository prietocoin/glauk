import { qs, clear, el } from '../../../shared/js/dom.js';
import { formatDate } from '../../../shared/js/format.js';

export function crearVistaLotes(ctx) {
  let mounted = false;

  async function cargar() {
    const body = qs('#lotes-tabla');
    if (!body) return;
    clear(body);
    body.append(el('tr', {}, el('td', { colspan: '3', class: 'muted' }, 'Cargando lotes…')));
    try {
      const lotes = await ctx.api.historial();
      clear(body);
      if (!Array.isArray(lotes) || lotes.length === 0) {
        body.append(el('tr', {}, el('td', { colspan: '3', class: 'empty' }, 'No hay lotes registrados.')));
        return;
      }
      for (const lote of lotes) {
        const id = lote.id_tasa || lote.id || '—';
        const tasas = lote.tasas || {};
        const fecha = lote.created_at || lote.fecha || lote.timestamp;
        const button = el('button', { class: 'btn btn-danger btn-sm', type: 'button' }, 'Eliminar');
        button.addEventListener('click', () => eliminar(id));
        body.append(el('tr', {},
          el('td', { class: 'mono' }, id),
          el('td', {}, String(Object.keys(tasas).length || 0)),
          el('td', {}, formatDate(fecha, true)),
          el('td', {}, button)
        ));
      }
    } catch (err) {
      clear(body);
      body.append(el('tr', {}, el('td', { colspan: '3', class: 'muted' }, err.message || 'No se pudo cargar el historial.')));
    }
  }

  async function eliminar(id) {
    if (!id || !globalThis.confirm(`¿Eliminar el lote ${id}?`)) return;
    try {
      await ctx.api.eliminarLote(id);
      ctx.notify.success(`Lote ${id} eliminado.`);
      await cargar();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo eliminar el lote.');
    }
  }

  function show() {
    const input = qs('#lote-id');
    if (input && !input.value) input.value = ctx.state.monitor.id_tasa || 'T000';
    cargar();
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#btn-refresh-lotes')?.addEventListener('click', cargar);
    qs('#btn-eliminar-lote')?.addEventListener('click', () => eliminar(qs('#lote-id')?.value.trim()));
  }

  return { mount, show, cargar };
}
