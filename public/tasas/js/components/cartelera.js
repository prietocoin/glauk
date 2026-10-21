import { qs, el } from '../../../shared/js/dom.js';
import { descargarBlob } from '../../../shared/js/reporte-canvas.js';

export function abrirCartelera(ctx, socio) {
  const root = qs('#modal-root');
  if (!root) return;
  const modal = el('div', { class: 'modal open', role: 'dialog', 'aria-modal': 'true', 'aria-label': `Cartelera de ${socio.nombre}` });
  const card = el('div', { class: 'modal-card' });
  const head = el('div', { class: 'modal-head' },
    el('h3', {}, `Cartelera · ${socio.nombre}`),
    el('button', { class: 'modal-close', type: 'button', 'aria-label': 'Cerrar' }, '×')
  );
  const body = el('div', { class: 'modal-body' }, el('div', { class: 'empty' }, 'Generando PNG…'));
  const foot = el('div', { class: 'modal-foot' });
  const download = el('button', { class: 'btn btn-success', type: 'button', disabled: true }, 'Descargar PNG');
  const close = el('button', { class: 'btn btn-secondary', type: 'button' }, 'Cerrar');
  foot.append(close, download);
  card.append(head, body, foot);
  modal.append(card);
  root.replaceChildren(modal);

  let url = '';
  let blob = null;
  const cerrar = () => {
    if (url) URL.revokeObjectURL(url);
    modal.remove();
  };
  head.querySelector('.modal-close').addEventListener('click', cerrar);
  close.addEventListener('click', cerrar);
  modal.addEventListener('click', (event) => { if (event.target === modal) cerrar(); });
  download.addEventListener('click', () => {
    if (blob) descargarBlob(blob, `cartelera-${socio.nombre.replace(/[^a-z0-9]/gi, '_')}.png`);
  });

  ctx.api.renderCartelera(socio.nombre).then((response) => {
    blob = response;
    url = URL.createObjectURL(blob);
    const image = el('img', { src: url, alt: `Cartelera de ${socio.nombre}` });
    body.replaceChildren(el('div', { class: 'preview-frame' }, image));
    download.disabled = false;
  }).catch((err) => {
    body.replaceChildren(el('div', { class: 'empty' }, err.message || 'No se pudo generar la cartelera.'));
    ctx.notify.error(err.message || 'No se pudo generar la cartelera.');
  });
}
