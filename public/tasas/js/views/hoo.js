import { qs, clear, el } from '../../../shared/js/dom.js';

export function crearVistaHoo(ctx) {
  let mounted = false;
  let ultimo = {};

  function render(borrador = ultimo) {
    ultimo = borrador || {};
    const pre = qs('#hoo-estado');
    const textarea = qs('#hoo-json');
    if (pre) pre.textContent = Object.keys(ultimo).length ? 'Borrador recibido. Revise y aplique.' : 'Sin consultar.';
    if (textarea) textarea.value = JSON.stringify(ultimo, null, 2);
  }

  async function consultar() {
    try {
      const borrador = await ctx.api.hoo();
      render(borrador);
      ctx.notify.success('Borrador HOO actualizado.');
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo consultar HOO.');
    }
  }

  function aplicar() {
    try {
      const borrador = JSON.parse(qs('#hoo-json')?.value || '{}');
      ctx.state.borrador = { ...ctx.state.borrador, ...borrador };
      ctx.monitor?.aplicarBorrador?.(borrador);
      ctx.navigate('monitor');
      ctx.notify.success('Borrador aplicado al monitor.');
    } catch {
      ctx.notify.error('El borrador no es un JSON válido.');
    }
  }

  function show() {
    if (Object.keys(ultimo).length) render(ultimo);
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#btn-hoo-actualizar')?.addEventListener('click', consultar);
    qs('#btn-hoo-aplicar')?.addEventListener('click', aplicar);
  }

  return { mount, show, render, consultar };
}
