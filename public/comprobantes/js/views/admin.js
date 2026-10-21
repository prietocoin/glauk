import { qs, el, escapeHTML } from '../../../shared/js/dom.js';
import { saveAdminKey, clearAdminKey, hasAdminKey } from '../../../shared/js/auth.js';
import { abrirModal } from '../components/modal.js';
import { renderTabla, celda, boton } from '../components/tabla.js';

export function crearVistaAdmin(ctx) {
  let mounted = false;

  function estado() {
    const badge = qs('#admin-estado');
    if (!badge) return;
    badge.textContent = hasAdminKey() ? 'Clave en sesión' : 'Sin autenticar';
    badge.className = `badge ${hasAdminKey() ? 'badge-green' : 'badge-red'}`;
    const input = qs('#admin-key');
    if (input) input.placeholder = hasAdminKey() ? 'Clave guardada en esta sesión' : 'Ingrese X-Admin-Key';
  }

  function acciones(item) {
    const wrap = el('div', { class: 'table-actions' });
    wrap.append(
      boton('Editar', () => editar(item), { className: 'btn btn-secondary btn-sm' }),
      boton('Eliminar', () => eliminar(item), { className: 'btn btn-danger btn-sm' })
    );
    return wrap;
  }

  async function cargar() {
    if (!hasAdminKey()) {
      ctx.notify.error('Guarde la clave administrativa para consultar la cola.');
      return;
    }
    const body = qs('#admin-tabla');
    if (body) body.append(el('tr', {}, el('td', { colspan: '6', class: 'muted' }, 'Consultando cola…')));
    try {
      const items = await ctx.api.cola();
      renderTabla(body, items, {
        colspan: 6, vacio: 'La cola administrativa está vacía.', render: (item) => el('tr', {},
          celda(item.hash_largo?.slice(0, 18) || '—', { className: 'mono', titulo: item.hash_largo || '' }),
          celda(item.nombre_socio_1 || '—'),
          celda(item.nombre_socio_2 || '—'),
          celda(item.estado || '—'),
          celda(item.conteo ?? '—'),
          celda(acciones(item))
        )
      });
    } catch (err) {
      renderTabla(body, [], { colspan: 6, vacio: err.message || 'No se pudo consultar la cola.' });
      ctx.notify.error(err.message || 'No se pudo consultar la cola.');
    }
  }

  function editar(item) {
    const contenido = `<div class="grid cols-2">
      <label class="field"><span class="label">Hash</span><input class="input" value="${escapeHTML(item.hash_largo || '')}" disabled></label>
      <label class="field"><span class="label">Estado</span><input class="input" id="admin-edit-estado" value="${escapeHTML(item.estado || '')}"></label>
      <label class="field"><span class="label">Socio 1</span><input class="input" id="admin-edit-socio1" value="${escapeHTML(item.nombre_socio_1 || '')}"></label>
      <label class="field"><span class="label">Socio 2</span><input class="input" id="admin-edit-socio2" value="${escapeHTML(item.nombre_socio_2 || '')}"></label>
      <label class="field"><span class="label">Conteo</span><input class="input" id="admin-edit-conteo" type="number" value="${Number(item.conteo || 0)}"></label>
    </div>`;
    const modal = abrirModal({
      titulo: 'Editar registro de cola', contenido,
      acciones: [
        { label: 'Cancelar', onClick: () => modal?.close() },
        { label: 'Guardar', className: 'btn-success', onClick: () => guardar(item, modal) }
      ]
    });
  }

  async function guardar(item, modal) {
    const payload = {
      estado: qs('#admin-edit-estado')?.value.trim() || '',
      nombre_socio_1: qs('#admin-edit-socio1')?.value.trim() || '',
      nombre_socio_2: qs('#admin-edit-socio2')?.value.trim() || '',
      conteo: Number(qs('#admin-edit-conteo')?.value || 0)
    };
    try {
      await ctx.api.actualizarCola(item.hash_largo, payload);
      ctx.notify.success('Registro de cola actualizado.');
      modal?.close();
      await cargar();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo actualizar el registro.');
    }
  }

  async function eliminar(item) {
    if (!globalThis.confirm('¿Eliminar las tablas raw de este registro?')) return;
    try {
      await ctx.api.eliminarCola(item.hash_largo);
      ctx.notify.success('Registro eliminado.');
      await cargar();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo eliminar el registro.');
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#btn-admin-key')?.addEventListener('click', () => {
      saveAdminKey(qs('#admin-key')?.value);
      estado();
      ctx.notify.success('Clave administrativa guardada en sessionStorage.');
    });
    qs('#btn-admin-clear')?.addEventListener('click', () => {
      clearAdminKey();
      if (qs('#admin-key')) qs('#admin-key').value = '';
      estado();
      ctx.notify.success('Clave administrativa olvidada.');
    });
    qs('#btn-admin-refrescar')?.addEventListener('click', cargar);
  }

  function show() { estado(); }

  return { mount, show, cargar };
}
