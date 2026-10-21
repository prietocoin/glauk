import { qs, el } from '../../../shared/js/dom.js';

let abierto = null;

export function abrirModal({ titulo, contenido, acciones = [], ancho = '' }) {
  cerrarModal();
  const root = qs('#modal-root');
  if (!root) return null;
  const modal = el('div', { class: 'modal open', role: 'dialog', 'aria-modal': 'true', 'aria-label': titulo });
  const card = el('div', { class: `modal-card ${ancho}`.trim() });
  const head = el('div', { class: 'modal-head' },
    el('h3', {}, titulo),
    el('button', { class: 'modal-close', type: 'button', 'aria-label': 'Cerrar' }, '×')
  );
  const body = el('div', { class: 'modal-body' });
  if (contenido instanceof Node) body.append(contenido);
  else body.innerHTML = contenido || '';
  const foot = el('div', { class: 'modal-foot' });
  for (const accion of acciones) {
    const button = el('button', {
      class: `btn ${accion.className || 'btn-secondary'}`,
      type: 'button', disabled: !!accion.disabled
    }, accion.label);
    button.addEventListener('click', () => accion.onClick?.(button));
    foot.append(button);
  }
  card.append(head, body, foot);
  modal.append(card);
  root.replaceChildren(modal);
  const close = () => {
    modal.remove();
    if (abierto === modal) abierto = null;
    document.removeEventListener('keydown', onKey);
  };
  const onKey = (event) => { if (event.key === 'Escape') close(); };
  head.querySelector('.modal-close').addEventListener('click', close);
  modal.addEventListener('click', (event) => { if (event.target === modal) close(); });
  document.addEventListener('keydown', onKey);
  abierto = modal;
  return { modal, body, close };
}

export function cerrarModal() {
  abierto?.remove();
  abierto = null;
}
