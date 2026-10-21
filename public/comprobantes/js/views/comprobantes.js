// =================================================================
// ARCHIVO: comprobantes.js (Ensamblador Principal)
// UBICACIÓN: public/js/views/comprobantes.js
// RESPONSABILIDAD: Unifica sub-módulos manteniendo la reactividad
// =================================================================

import { comprobantesState } from './comprobantes/comprobantesState.js';
import { comprobantesCalculos } from './comprobantes/comprobantesCalculos.js';
import { comprobantesGetters } from './comprobantes/comprobantesGetters.js';
import { comprobantesAcciones } from './comprobantes/comprobantesAcciones.js';
import { comprobantesFormatters } from './comprobantes/comprobantesFormatters.js';

export function comprobantesView() {
  // 1. Unificar Estado, Cálculos, Acciones y Formateadores
  const view = {
    ...comprobantesState,
    ...comprobantesCalculos,
    ...comprobantesAcciones,
    ...comprobantesFormatters
  };

  // 2. Copiar los Getters preservando su naturaleza reactiva para Alpine.js
  // (Evita que el spread operator '...' los convierta en arrays vacíos estáticos)
  Object.defineProperties(view, Object.getOwnPropertyDescriptors(comprobantesGetters));

  return view;
}

// 3. Registro global para Alpine.js
if (typeof window !== 'undefined') {
  window.comprobantesView = comprobantesView;
}
