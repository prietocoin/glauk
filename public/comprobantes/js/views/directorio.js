import { qs } from '../../../shared/js/dom.js';
import { celda } from '../components/tabla.js';

export function crearVistaDirectorio(ctx) {
  let mounted = false;

  function setSocios(socios) {
    ctx.state.socios = socios || [];
    if (ctx.state.view === 'directorio') render();
  }

  function visibles() {
    const query = (ctx.state.query || '').toLowerCase().trim();
    if (!query) return ctx.state.socios;
    return ctx.state.socios.filter((socio) =>
      [socio.nombre, socio.rol, socio.moneda_base, socio.id_grupo]
        .some((value) => String(value || '').toLowerCase().includes(query))
    );
  }

  function render() {
    const items = visibles();
    const body = qs('#directorio-tabla');
    if (!body) return;
    body.replaceChildren();
    if (!items.length) {
      const row = document.createElement('tr');
      const cell = document.createElement('td');
      cell.colSpan = 5;
      cell.className = 'empty';
      cell.textContent = 'No hay socios que coincidan.';
      row.append(cell);
      body.append(row);
      return;
    }
    for (const socio of items) {
      const monedas = Object.keys(socio.monedas || {}).filter((code) => socio.monedas[code]?.activo !== false);
      const badge = document.createElement('span');
      badge.className = `badge ${socio.activo ? 'badge-green' : 'badge-red'}`;
      badge.textContent = socio.activo ? 'Activo' : 'Inactivo';
      const row = document.createElement('tr');
      row.append(
        celda(socio.nombre || '—'),
        celda(socio.rol || '—'),
        celda(socio.moneda_base || 'USDT'),
        celda(`${monedas.length} · ${monedas.slice(0, 4).join(', ')}`),
        celda(badge)
      );
      body.append(row);
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#directorio-busqueda')?.addEventListener('input', (event) => {
      ctx.state.query = event.target.value;
      render();
    });
    qs('#btn-refrescar-directorio')?.addEventListener('click', () => ctx.reloadDirectorio());
  }

  function show() { render(); }

  return { mount, show, setSocios, render };
}
