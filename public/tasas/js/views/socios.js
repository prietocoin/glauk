import { qs, qsa, clear, el, escapeHTML } from '../../../shared/js/dom.js';
import { formatPercent, toNumber } from '../../../shared/js/format.js';
import { abrirCartelera } from '../components/cartelera.js';

const MONEDAS = [
  'ARS', 'BOB', 'BRL', 'CAD', 'CLP', 'COP', 'CRC', 'DOP', 'EUR',
  'MXN', 'PAN', 'PEN', 'PYG', 'VES', 'USD', 'USDT', 'PYUSD', 'ECU'
];

export function crearVistaSocios(ctx) {
  let mounted = false;

  function setSocios(socios) {
    ctx.state.socios = socios || [];
    const select = qs('#socio-select');
    if (!select) return;
    clear(select);
    select.append(el('option', { value: '' }, 'Seleccione un socio…'));
    for (const socio of ctx.state.socios) {
      select.append(el('option', { value: socio.nombre }, socio.nombre));
    }
    if (!ctx.state.selected && ctx.state.socios[0]) select.value = ctx.state.socios[0].nombre;
    if (select.value) seleccionar(select.value);
  }

  function seleccionar(nombre) {
    const socio = ctx.state.socios.find((item) => item.nombre === nombre) || null;
    ctx.state.selected = socio;
    render();
  }

  function render() {
    const host = qs('#socio-detalle');
    const estado = qs('#socio-estado');
    if (!host) return;
    const socio = ctx.state.selected;
    if (!socio) {
      host.innerHTML = '<div class="empty">Seleccione un socio para editar sus monedas.</div>';
      if (estado) estado.textContent = 'Sin socio seleccionado.';
      return;
    }
    if (estado) estado.textContent = `${socio.rol || 'SOCIO'} · ${socio.moneda_base || 'USDT'} · ${socio.id_grupo || 'sin JID'}`;
    const monedas = socio.monedas || {};
    const tarjetas = MONEDAS.map((code) => {
      const regla = monedas[code] || {};
      const activo = regla.activo !== false;
      const porcentaje = regla.porcentaje || {};
      return `<div class="panel stack" data-moneda="${code}">
        <label class="check"><input type="checkbox" data-activo="${code}" ${activo ? 'checked' : ''}> <strong class="mono">${code}</strong></label>
        <div class="grid cols-2">
          <label class="field"><span class="label">Depósito</span><input class="input" type="number" step="0.01" data-dep="${code}" value="${toNumber(porcentaje.deposito ?? regla.deposito)}"></label>
          <label class="field"><span class="label">Pago</span><input class="input" type="number" step="0.01" data-pago="${code}" value="${toNumber(porcentaje.pago ?? regla.pago)}"></label>
        </div>
        <small class="tiny muted">${activo ? formatPercent(porcentaje.deposito ?? 0) : 'Desactivada'}</small>
      </div>`;
    }).join('');
    host.innerHTML = `<div class="grid cols-3">${tarjetas}</div>`;
    host.dataset.socio = escapeHTML(socio.nombre);
  }

  function recolectar() {
    const socio = ctx.state.selected;
    if (!socio) return null;
    const monedas = {};
    for (const code of MONEDAS) {
      const activo = qs(`[data-activo="${code}"]`)?.checked ?? false;
      const deposito = toNumber(qs(`[data-dep="${code}"]`)?.value);
      const pago = toNumber(qs(`[data-pago="${code}"]`)?.value);
      monedas[code] = { activo, porcentaje: { deposito, pago } };
    }
    return { monedas };
  }

  async function guardar() {
    const socio = ctx.state.selected;
    const cambios = recolectar();
    if (!socio || !cambios) return;
    try {
      const data = await ctx.api.guardarSocio(socio.nombre, cambios);
      const actualizado = data.socio || { ...socio, ...cambios };
      ctx.state.socios = ctx.state.socios.map((item) => item.nombre === socio.nombre ? actualizado : item);
      ctx.state.selected = actualizado;
      render();
      ctx.notify.success('Cambios del socio guardados.');
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudieron guardar los cambios.');
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#socio-select')?.addEventListener('change', (event) => seleccionar(event.target.value));
    qs('#btn-guardar-socio')?.addEventListener('click', guardar);
    qs('#btn-cartelera')?.addEventListener('click', () => {
      if (ctx.state.selected) abrirCartelera(ctx, ctx.state.selected);
      else ctx.notify.error('Seleccione un socio.');
    });
    qs('#btn-disparar')?.addEventListener('click', async () => {
      if (!ctx.state.selected) return;
      try {
        await ctx.api.disparar(ctx.state.selected.nombre, ctx.state.test);
        ctx.notify.success(`Cartelera disparada a ${ctx.state.selected.nombre}.`);
      } catch (err) {
        ctx.notify.error(err.message || 'No se pudo disparar la cartelera.');
      }
    });
  }

  function show() {
    if (!ctx.state.selected && ctx.state.socios[0]) ctx.state.selected = ctx.state.socios[0];
    const select = qs('#socio-select');
    if (select && ctx.state.selected) select.value = ctx.state.selected.nombre;
    render();
  }

  return { mount, show, setSocios, render, seleccionar };
}
