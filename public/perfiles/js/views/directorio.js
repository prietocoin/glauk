import { qs, clear, el, escapeHTML } from '../../../shared/js/dom.js';

export function crearVistaDirectorio(ctx) {
  let mounted = false;

  function setSocios(socios) {
    ctx.state.socios = socios || [];
    if (ctx.state.view === 'directorio') render();
  }

  function visibles() {
    const query = (ctx.state.query || '').toLowerCase().trim();
    return ctx.state.socios.filter((socio) => {
      const activo = !!socio.activo;
      if (ctx.state.filter === 'activo' && !activo) return false;
      if (ctx.state.filter === 'inactivo' && activo) return false;
      if (!query) return true;
      return [socio.nombre, socio.rol, socio.id_grupo, socio.moneda_base]
        .some((value) => String(value || '').toLowerCase().includes(query));
    });
  }

  function resumen() {
    const total = ctx.state.socios.length;
    const activos = ctx.state.socios.filter((socio) => socio.activo).length;
    const monedas = ctx.state.socios.reduce((sum, socio) => sum + Object.keys(socio.monedas || {}).length, 0);
    const grupos = new Set(ctx.state.socios.map((socio) => socio.id_grupo).filter(Boolean)).size;
    const host = qs('#perfiles-resumen');
    if (!host) return;
    clear(host);
    for (const [label, value] of [['Socios', total], ['Activos', activos], ['Inactivos', total - activos], ['Reglas de moneda', monedas], ['Grupos', grupos]]) {
      host.append(el('div', { class: 'kpi' }, el('div', { class: 'kpi-label' }, label), el('div', { class: 'kpi-value' }, String(value))));
    }
  }

  function fila(socio) {
    const monedas = Object.keys(socio.monedas || {}).filter((code) => socio.monedas[code]?.activo !== false);
    const acciones = el('div', { class: 'table-actions' });
    const editar = el('button', { class: 'btn btn-secondary btn-sm', type: 'button' }, 'Editar');
    editar.addEventListener('click', () => ctx.editSocio(socio));
    acciones.append(editar);
    if (socio.nombre !== 'GENERAL') {
      const toggle = el('button', { class: `btn btn-sm ${socio.activo ? 'btn-warning' : 'btn-success'}`, type: 'button' }, socio.activo ? 'Apagar' : 'Activar');
      toggle.addEventListener('click', () => cambiarEstado(socio));
      const borrar = el('button', { class: 'btn btn-danger btn-sm', type: 'button' }, 'Eliminar');
      borrar.addEventListener('click', () => eliminar(socio));
      acciones.append(toggle, borrar);
    }
    return el('tr', {},
      el('td', {}, el('strong', {}, socio.nombre || '—'), el('div', { class: 'tiny muted mono' }, escapeHTML(socio.id_grupo || ''))),
      el('td', {}, socio.rol || '—'),
      el('td', {}, socio.moneda_base || 'USDT'),
      el('td', {}, el('span', { class: 'badge' }, `${monedas.length} activas`), ' ', el('span', { class: 'tiny muted' }, monedas.slice(0, 4).join(', '))),
      el('td', {}, el('span', { class: `badge ${socio.activo ? 'badge-green' : 'badge-red'}` }, socio.activo ? 'Activo' : 'Inactivo')),
      el('td', {}, acciones)
    );
  }

  function render() {
    resumen();
    const body = qs('#perfiles-tabla');
    if (!body) return;
    clear(body);
    const items = visibles();
    if (!items.length) {
      body.append(el('tr', {}, el('td', { colspan: '6', class: 'empty' }, 'No hay socios que coincidan con el filtro.')));
      return;
    }
    for (const socio of items) body.append(fila(socio));
  }

  async function cambiarEstado(socio) {
    try {
      await ctx.api.toggle(socio.nombre, !socio.activo);
      ctx.notify.success(`${socio.nombre} ${socio.activo ? 'apagado' : 'activado'}.`);
      await ctx.reloadSocios();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo cambiar el estado.');
    }
  }

  async function eliminar(socio) {
    if (!globalThis.confirm(`¿Eliminar el perfil de ${socio.nombre}? Esta acción no se puede deshacer.`)) return;
    try {
      await ctx.api.eliminar(socio.nombre);
      ctx.notify.success(`Perfil ${socio.nombre} eliminado.`);
      await ctx.reloadSocios();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo eliminar el perfil.');
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#perfiles-busqueda')?.addEventListener('input', (event) => {
      ctx.state.query = event.target.value;
      render();
    });
    qs('#perfiles-filtro')?.addEventListener('change', (event) => {
      ctx.state.filter = event.target.value;
      render();
    });
  }

  function show() { render(); }

  return { mount, show, setSocios, render };
}
