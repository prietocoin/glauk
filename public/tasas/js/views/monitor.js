import { qs, el, clear } from '../../../shared/js/dom.js';
import { formatNumber } from '../../../shared/js/format.js';

const MONEDAS = [
  'ARS', 'BOB', 'BRL', 'CAD', 'CLP', 'COP', 'CRC', 'DOP', 'EUR',
  'MXN', 'PAN', 'PEN', 'PYG', 'VES', 'USD', 'USDT', 'PYUSD', 'ECU'
];

const DEFAULTS = { USD: 1, USDT: 1, PYUSD: 1.2, ECU: 1, PAN: 1 };

export function crearVistaMonitor(ctx) {
  let mounted = false;

  function valores() {
    return { ...DEFAULTS, ...(ctx.state.monitor.tasas || {}), ...(ctx.state.borrador || {}) };
  }

  function render() {
    const grid = qs('#tasas-grid');
    if (!grid) return;
    clear(grid);
    const actuales = ctx.state.monitor.tasas || {};
    const borrador = valores();
    for (const code of MONEDAS) {
      const produccion = actuales[code] ?? DEFAULTS[code] ?? 0;
      const input = el('input', {
        class: 'input', type: 'number', step: 'any', value: borrador[code] ?? 0,
        'aria-label': `Nueva tasa ${code}`
      });
      input.addEventListener('input', () => {
        ctx.state.borrador[code] = Number.parseFloat(input.value) || 0;
      });
      const card = el('div', { class: 'panel stack' },
        el('div', { class: 'row between' },
          el('strong', { class: 'mono' }, code),
          el('span', { class: 'tiny muted' }, `Prod. ${formatNumber(produccion, 4)}`)
        ),
        el('label', { class: 'field' }, el('span', { class: 'label' }, 'Nuevo lote'), input)
      );
      grid.append(card);
    }
    const badge = qs('#lote-badge');
    if (badge) badge.textContent = ctx.state.monitor.id_tasa || 'T000';
  }

  async function show() {
    try {
      const data = await ctx.api.monitor();
      ctx.state.monitor = { id_tasa: data.id_tasa || 'T000', tasas: data.tasas || {} };
      if (!Object.keys(ctx.state.borrador).length) ctx.state.borrador = { ...ctx.state.monitor.tasas };
      render();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo cargar el monitor.');
    }
  }

  async function guardar() {
    try {
      await ctx.api.guardarLote(valores());
      ctx.notify.success('Lote guardado en tasas_glaukov.');
      await show();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo guardar el lote.');
    }
  }

  async function publicar() {
    if (!globalThis.confirm('¿Publicar el lote vigente a todos los socios activos?')) return;
    try {
      const data = await ctx.api.publicarLote(ctx.state.test);
      ctx.notify.success(`Lote ${data.id_tasa || ''} publicado (${data.procesados ?? 0} socios).`);
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo publicar el lote.');
    }
  }

  async function traerHoo() {
    try {
      const borrador = await ctx.api.hoo();
      aplicarBorrador(borrador);
      ctx.notify.success('Borrador HOO aplicado al monitor.');
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo consultar HOO.');
    }
  }

  function aplicarBorrador(borrador) {
    ctx.state.borrador = { ...ctx.state.borrador, ...borrador };
    render();
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#btn-refresh-monitor')?.addEventListener('click', show);
    qs('#btn-guardar-lote')?.addEventListener('click', guardar);
    qs('#btn-publicar-lote')?.addEventListener('click', publicar);
    qs('#btn-traer-hoo')?.addEventListener('click', traerHoo);
    qs('#btn-remesas')?.addEventListener('click', async () => {
      try {
        const data = await ctx.api.notificarRemesas();
        ctx.notify.success(`Remesas notificadas para ${data.notificacion?.id_tasa || 'el lote vigente'}.`);
      } catch (err) {
        ctx.notify.error(err.message || 'No se pudo notificar remesas.');
      }
    });
    qs('#modo-prueba')?.addEventListener('change', (event) => {
      ctx.state.test = event.target.checked;
    });
  }

  return { mount, show, render, aplicarBorrador };
}
