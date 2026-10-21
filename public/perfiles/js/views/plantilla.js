import { qs } from '../../../shared/js/dom.js';

export function crearVistaPlantilla(ctx) {
  let mounted = false;

  function render() {
    const total = ctx.state.socios.length;
    const activos = ctx.state.socios.filter((socio) => socio.activo).length;
    const host = qs('#plantilla-estado');
    if (host) {
      host.textContent = total
        ? `${activos} de ${total} socios publican tasas. GENERAL siempre permanece activo.`
        : 'Aún no hay socios cargados.';
    }
  }

  async function ejecutar(operacion, mensaje) {
    try {
      await operacion();
      ctx.notify.success(mensaje);
      await ctx.reloadSocios();
      render();
    } catch (err) {
      ctx.notify.error(err.message || 'No se pudo actualizar la plantilla.');
    }
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#btn-memorizar')?.addEventListener('click', () => ejecutar(ctx.api.memorizar, 'Plantilla memorizada.'));
    qs('#btn-restaurar')?.addEventListener('click', () => ejecutar(ctx.api.restaurar, 'Plantilla restaurada.'));
    qs('#btn-apagar-todos')?.addEventListener('click', () => {
      if (!globalThis.confirm('¿Apagar todos los socios? GENERAL permanecerá activo.')) return;
      ejecutar(ctx.api.apagarTodos, 'Socios apagados; GENERAL permanece activo.');
    });
  }

  function show() { render(); }

  return { mount, show, render };
}
