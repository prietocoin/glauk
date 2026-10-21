import { clear, el } from '../../../shared/js/dom.js';

export function renderTabla(body, filas, { colspan = 6, vacio = 'Sin registros.', render } = {}) {
  if (!body) return;
  clear(body);
  if (!filas || filas.length === 0) {
    body.append(el('tr', {}, el('td', { colspan: String(colspan), class: 'empty' }, vacio)));
    return;
  }
  for (const item of filas) body.append(render ? render(item) : fila(item));
}

export function fila(item, columnas = []) {
  const tr = el('tr');
  for (const columna of columnas) {
    const value = typeof columna === 'function' ? columna(item) : item[columna];
    tr.append(el('td', {}, value instanceof Node ? value : String(value ?? '—')));
  }
  return tr;
}

export function celda(valor, { className = '', titulo = '' } = {}) {
  const td = el('td', { class: className, title: titulo });
  if (valor instanceof Node) td.append(valor);
  else td.textContent = valor ?? '—';
  return td;
}

export function boton(texto, onClick, { className = 'btn btn-secondary btn-sm', disabled = false } = {}) {
  const button = el('button', { class: className, type: 'button', disabled }, texto);
  button.addEventListener('click', () => onClick(button));
  return button;
}
