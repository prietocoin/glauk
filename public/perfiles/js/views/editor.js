import { qs, clear, el } from '../../../shared/js/dom.js';
import { toNumber } from '../../../shared/js/format.js';

const MONEDAS = [
  'ARS', 'BOB', 'BRL', 'CAD', 'CLP', 'COP', 'CRC', 'DOP', 'EUR',
  'MXN', 'PAN', 'PEN', 'PYG', 'VES', 'USD', 'USDT', 'PYUSD', 'ECU'
];

export function crearVistaEditor(ctx) {
  let mounted = false;
  let editing = null;

  function open(socio) {
    editing = socio || null;
    qs('#editor-titulo').textContent = socio ? `Editar ${socio.nombre}` : 'Nuevo socio';
    qs('#editor-badge').textContent = socio ? 'Edición' : 'Nuevo';
    qs('#editor-nombre').value = socio?.nombre || '';
    qs('#editor-nombre').readOnly = !!socio;
    qs('#editor-grupo').value = socio?.id_grupo || '';
    qs('#editor-rol').value = socio?.rol || 'SOCIO';
    qs('#editor-base').value = socio?.moneda_base || 'USDT';
    qs('#editor-saldo').value = socio?.saldo_inicial ?? '0';
    qs('#editor-herencia').checked = !!socio?.herencia;
    renderMonedas(socio?.monedas || {});
  }

  function renderMonedas(monedas) {
    const grid = qs('#monedas-grid');
    if (!grid) return;
    clear(grid);
    for (const code of MONEDAS) {
      const regla = monedas[code] || {};
      const porcentaje = regla.porcentaje || {};
      const activo = el('input', { type: 'checkbox', dataset: { activo: code } });
      activo.checked = regla.activo !== false;
      const deposito = el('input', { class: 'input', type: 'number', step: '0.01', dataset: { deposito: code } });
      deposito.value = toNumber(porcentaje.deposito ?? regla.deposito);
      const pago = el('input', { class: 'input', type: 'number', step: '0.01', dataset: { pago: code } });
      pago.value = toNumber(porcentaje.pago ?? regla.pago);
      grid.append(el('div', { class: 'panel stack' },
        el('label', { class: 'check' }, activo, el('strong', { class: 'mono' }, code)),
        el('div', { class: 'grid cols-2' },
          el('label', { class: 'field' }, el('span', { class: 'label' }, 'Depósito'), deposito),
          el('label', { class: 'field' }, el('span', { class: 'label' }, 'Pago'), pago)
        )
      ));
    }
  }

  function leerMonedas() {
    const grid = qs('#monedas-grid');
    const monedas = {};
    for (const code of MONEDAS) {
      const activo = grid.querySelector(`[data-activo="${code}"]`)?.checked ?? false;
      const deposito = toNumber(grid.querySelector(`[data-deposito="${code}"]`)?.value);
      const pago = toNumber(grid.querySelector(`[data-pago="${code}"]`)?.value);
      monedas[code] = { activo, porcentaje: { deposito, pago } };
    }
    return monedas;
  }

  async function guardar(event) {
    event.preventDefault();
    const nombre = qs('#editor-nombre').value.trim();
    if (!nombre) {
      ctx.notify.error('El nombre del socio es obligatorio.');
      return;
    }
    const payload = {
      nombre,
      id_grupo: qs('#editor-grupo').value.trim(),
      rol: qs('#editor-rol').value.trim() || 'SOCIO',
      moneda_base: qs('#editor-base').value.trim().toUpperCase() || 'USDT',
      saldo_inicial: String(toNumber(qs('#editor-saldo').value)),
      herencia: qs('#editor-herencia').checked,
      mostrar: editing?.mostrar || { tasas: true, dashboard: true },
      monedas: leerMonedas()
    };
    try {
      await ctx.api.guardar(payload);
      ctx.notify.success(`Perfil ${nombre} guardado.`);
      await ctx.reloadSocios();
      ctx.closeEditor();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo guardar el perfil.');
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#perfil-form')?.addEventListener('submit', guardar);
    qs('#btn-cancelar-editor')?.addEventListener('click', () => ctx.closeEditor());
  }

  function show() {
    if (!editing) open(null);
  }

  return { mount, show, open };
}
