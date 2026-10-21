import { qs, el, clear, renderOptions } from '../../../shared/js/dom.js';
import { formatDate, formatDateInput, formatMoney, formatNumber } from '../../../shared/js/format.js';
import { generarReporteJPEG, descargarBlob } from '../../../shared/js/reporte-canvas.js';
import { idempotencyKey } from '../../../shared/js/http.js';

export function crearVistaReportes(ctx) {
  let mounted = false;
  let blob = null;
  let base64 = '';
  let clave = '';
  let filas = [];

  function setSocios(socios) {
    renderOptions(qs('#reporte-socio'), (socios || []).map((socio) => socio.nombre), 'Seleccione un socio…');
  }

  function capturar() {
    return {
      socio: qs('#reporte-socio')?.value || '',
      desde: qs('#reporte-desde')?.value || '',
      hasta: qs('#reporte-hasta')?.value || '',
      lote: qs('#reporte-lote')?.value.trim() || '',
      caption: qs('#reporte-caption')?.value.trim() || ''
    };
  }

  function filaReporte(item) {
    return el('tr', {},
      el('td', {}, formatDate(item.fecha_hora_comprobante)),
      el('td', {}, item.nombre_socio_1 || '—'),
      el('td', {}, item.tipo_op1 || '—'),
      el('td', {}, formatMoney(item.monto, item.moneda)),
      el('td', {}, formatMoney(item.me1, 'USDT'))
    );
  }

  function construirReporte(meta, items) {
    const host = qs('#reporte-render');
    if (!host) return;
    clear(host);
    const logo = el('img', { src: '../shared/brand/logo-fundablock.png', alt: 'FUNDABLOCK', style: 'width:88px;height:auto' });
    host.append(el('header', {},
      el('div', { class: 'row' }, logo, el('div', {}, el('h2', {}, 'FUNDABLOCK'), el('p', {}, 'Glaukov Engine · Atenea'))),
      el('div', { style: 'text-align:right' }, el('h1', {}, 'REPORTE'), el('small', {}, formatDate(new Date(), true)))
    ));
    host.append(el('div', { class: 'reporte-meta' },
      el('div', {}, el('strong', {}, 'Socio'), el('div', {}, meta.socio || 'Todos')),
      el('div', {}, el('strong', {}, 'Periodo'), el('div', {}, `${meta.desde || '—'} / ${meta.hasta || '—'}`)),
      el('div', {}, el('strong', {}, 'Lote'), el('div', {}, meta.lote || 'Vigente'))
    ));
    const table = el('table', {}, el('thead', {}, el('tr', {},
      ['Fecha', 'Socio', 'Operación', 'Monto', 'ME1'].map((title) => el('th', {}, title))
    )));
    const tbody = el('tbody');
    for (const item of items.slice(0, 18)) tbody.append(filaReporte(item));
    if (!items.length) tbody.append(el('tr', {}, el('td', { colspan: '5', class: 'center' }, 'Sin movimientos en el periodo.')));
    table.append(tbody);
    host.append(table);
    const total = items.reduce((sum, item) => sum + Number(item.me1 || 0), 0);
    host.append(el('div', { class: 'reporte-total' }, `ME1 total: ${formatNumber(total, 2)}`));
    host.append(el('div', { class: 'reporte-foot' },
      el('span', {}, `${items.length} operaciones · ${items.length > 18 ? 'vista parcial de 18' : 'listado completo'}`),
      el('span', {}, 'Documento generado localmente en el navegador.')
    ));
  }

  async function cargar() {
    const meta = capturar();
    if (!meta.socio) {
      ctx.notify.error('Seleccione un socio para construir el reporte.');
      return [];
    }
    const params = {
      socio: meta.socio, fechaInicio: meta.desde, fechaFin: meta.hasta, orden: 'fecha_desc'
    };
    let items = await ctx.api.listar(params);
    if (meta.lote) {
      items = items.filter((item) => String(item.lote_tasa_asignado || '').toUpperCase() === meta.lote.toUpperCase());
    }
    filas = items;
    construirReporte(meta, items);
    return items;
  }

  async function generar() {
    const meta = capturar();
    if (!meta.socio) {
      ctx.notify.error('Seleccione un socio.');
      return;
    }
    try {
      qs('#reporte-badge').textContent = 'Generando…';
      await cargar();
      const resultado = await generarReporteJPEG({ elemento: qs('#reporte-render') });
      blob = resultado.blob;
      base64 = resultado.base64;
      clave = idempotencyKey();
      const image = el('img', { src: `data:image/jpeg;base64,${base64}`, alt: 'Vista previa del reporte' });
      qs('#reporte-preview').replaceChildren(image);
      qs('#btn-enviar-reporte').disabled = false;
      qs('#btn-descargar-reporte').disabled = false;
      qs('#reporte-badge').textContent = `${Math.round(blob.size / 1024)} KiB`;
      qs('#reporte-resultado').textContent = `${filas.length} movimientos incluidos.`;
      ctx.notify.success('JPEG generado localmente.');
    } catch (err) {
      qs('#reporte-badge').textContent = 'Error';
      ctx.notify.error(err.message || 'No se pudo generar el reporte.');
    }
  }

  async function enviar() {
    if (!base64) {
      await generar();
      if (!base64) return;
    }
    const meta = capturar();
    const button = qs('#btn-enviar-reporte');
    button.disabled = true;
    try {
      const data = await ctx.api.enviarImagen({
        socio: meta.socio,
        caption: meta.caption || `Reporte de ${meta.socio}`,
        base64,
        mimeType: 'image/jpeg',
        fileName: `reporte-${meta.socio}.jpg`
      }, clave);
      const result = data?.data || data;
      qs('#reporte-resultado').textContent = `Enviado · ${result?.messageId || 'sin ID'}${result?.simulated ? ' · simulado' : ''}`;
      qs('#reporte-badge').textContent = 'Enviado';
      clave = idempotencyKey();
      ctx.notify.success('Reporte enviado por WhatsApp.');
    } catch (err) {
      qs('#reporte-resultado').textContent = err.message || 'No se pudo enviar el reporte.';
      ctx.notify.error(err.message || 'No se pudo enviar el reporte.');
    } finally {
      button.disabled = !base64;
    }
  }

  function descargar() {
    if (blob) descargarBlob(blob, `reporte-${capturar().socio || 'glauk'}.jpg`);
  }

  function mount() {
    if (mounted) return;
    mounted = true;
    qs('#btn-generar-reporte')?.addEventListener('click', generar);
    qs('#btn-enviar-reporte')?.addEventListener('click', enviar);
    qs('#btn-descargar-reporte')?.addEventListener('click', descargar);
    qs('#reporte-form')?.addEventListener('submit', (event) => { event.preventDefault(); generar(); });
  }

  function show() {
    if (qs('#reporte-desde') && !qs('#reporte-desde').value) qs('#reporte-desde').value = formatDateInput(new Date());
    if (qs('#reporte-hasta') && !qs('#reporte-hasta').value) qs('#reporte-hasta').value = formatDateInput(new Date());
  }

  return { mount, show, setSocios, generar };
}
