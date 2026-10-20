// GENERADOR DE FIXTURES DORADOS (herramienta temporal, no forma parte del proyecto).
//
// Ejecuta el calculador REAL de Node sobre una matriz de casos borde y escribe el
// resultado como JSON. El puerto en Go debe reproducirlo exactamente.
//
// El reloj se congela por caso para que fecha_actualizacion/hora_actualizacion
// sean reproducibles: sin esto, el fixture caducaría en el minuto siguiente.
//
// Uso: node gen_golden_calculator.mjs <raiz-proyecto-node> <ruta-salida.json>
//
// Ejemplo, desde la carpeta tasasgo/ :
//   node internal/modules/tasas/calculator/testdata/gen_golden_calculator.mjs ../old internal/modules/tasas/calculator/testdata/golden_calculator.json
import { pathToFileURL } from 'node:url';
import { writeFileSync } from 'node:fs';

const [root, outPath] = process.argv.slice(2);
const { calcularTasasCartelera } = await import(
  pathToFileURL(`${root}/src/modules/tasas/services/calculator.service.js`).href
);
const { aplicarReglaPrecisionTasa } = await import(pathToFileURL(`${root}/src/utils/formatters.js`).href);

// ---------------------------------------------------------------- reloj congelado
let FIXED = Date.parse('2026-10-18T02:35:00Z');
const RealDate = Date;
class FrozenDate extends RealDate {
  constructor(...args) {
    super(...(args.length ? args : [FIXED]));
  }
  static now() {
    return FIXED;
  }
}
globalThis.Date = FrozenDate;

const dateOpts = { timeZone: 'America/Caracas', day: '2-digit', month: 'short' };
const hourOpts = { timeZone: 'America/Caracas', hour: '2-digit', minute: '2-digit', hour12: false };

// Inserta las claves ordenadas para que el orden de inserción del JSON coincida
// con el orden alfabético que emitirá el puerto en Go.
function sorted(obj) {
  if (obj === null || typeof obj !== 'object' || Array.isArray(obj)) return obj;
  const out = {};
  for (const key of Object.keys(obj).sort()) out[key] = obj[key];
  return out;
}

const CATALOGO = ['ARS', 'BOB', 'BRL', 'CAD', 'CLP', 'COP', 'CRC', 'DOP', 'EUR', 'MXN', 'PEN', 'PYG', 'USD', 'USDT', 'PYUSD', 'VES', 'ECU', 'PAN'];
const TASAS = {
  ARS: 1450.5, BOB: 6.96, BRL: 5.42, CAD: 1.36, CLP: 945, COP: 4050, CRC: 520, DOP: 58.5,
  EUR: 0.92, MXN: 17.15, PEN: 3.745, PYG: 7300, USD: 1, USDT: 1, PYUSD: 1, VES: 36.5, ECU: 1, PAN: 1,
};

const I = {
  catalogo: Date.parse('2026-10-18T02:35:00Z'),
  basePen: Date.parse('2026-03-01T00:10:00Z'),
  tendencias: Date.parse('2026-01-01T04:00:00Z'),
  simple: Date.parse('2026-06-15T04:05:00Z'),
  otroMes: Date.parse('2026-12-31T23:59:00Z'),
};

// Margenes variados para cubrir el valor absoluto y los factores 0 y >1.
const monedasCatalogo = {};
CATALOGO.forEach((code, i) => {
  monedasCatalogo[code] = { activo: true, porcentaje: { deposito: (i % 7) - 3, pago: i % 5 } };
});

const perfil = (nombre, monedaBase, monedas, idGrupo) => ({
  nombre,
  id_grupo: idGrupo,
  moneda_base: monedaBase,
  monedas: sorted(monedas),
});

const casos = [
  {
    nombre: 'catalogo_completo_18_monedas',
    instante_millis: I.catalogo,
    perfil: perfil('Glaukov', 'USDT', monedasCatalogo, '120363421142957552@g.us'),
    lote_actual: { id_tasa: 'T059', tasas: sorted(TASAS) },
    lote_anterior: { id_tasa: 'T058', tasas: sorted({ ...TASAS, COP: 4000, EUR: 0.95 }) },
  },
  {
    nombre: 'base_pen_triangulada',
    instante_millis: I.basePen,
    perfil: perfil('Socio Peru', 'PEN', {
      PEN: { activo: true, porcentaje: { deposito: 2, pago: 2 } },
      USD: { activo: true, porcentaje: { deposito: 1.5, pago: 1.5 } },
      COP: { activo: true, porcentaje: { deposito: -3, pago: 4 } },
      EUR: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      ARS: { activo: true, porcentaje: { deposito: 5, pago: 5 } },
    }, ''),
    lote_actual: { id_tasa: 'T100', tasas: sorted(TASAS) },
    lote_anterior: { id_tasa: 'T099', tasas: sorted(TASAS) },
  },
  {
    nombre: 'tendencias_up_down_equal_ausente',
    instante_millis: I.tendencias,
    perfil: perfil('Socio Tendencia', 'USDT', {
      BOB: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      EUR: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      USD: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T010', tasas: sorted({ BOB: 6.96, COP: 4050, EUR: 0.92, USD: 1 }) },
    lote_anterior: { id_tasa: 'T009', tasas: sorted({ COP: 4000, EUR: 0.95, USD: 1 }) },
  },
  {
    nombre: 'sin_lote_anterior',
    instante_millis: I.simple,
    perfil: perfil('Socio Nuevo', 'USDT', {
      COP: { activo: true, porcentaje: { deposito: 1, pago: 2 } },
      EUR: { activo: true, porcentaje: { deposito: 1, pago: 2 } },
    }),
    lote_actual: { id_tasa: 'T001', tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'activo_omitido_y_activo_false',
    instante_millis: I.simple,
    perfil: perfil('Socio Discreto', 'USDT', {
      EUR: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PEN: { porcentaje: { deposito: 0, pago: 0 } },
      COP: { activo: false, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T020', tasas: sorted(TASAS) },
    lote_anterior: { id_tasa: 'T019', tasas: sorted(TASAS) },
  },
  {
    nombre: 'ausentes_no_fijas_se_descartan',
    instante_millis: I.simple,
    perfil: perfil('Socio Parcial', 'USDT', {
      BOB: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      CLP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T030', tasas: sorted({ COP: 4050 }) },
    lote_anterior: { id_tasa: 'T029', tasas: sorted({ COP: 4000 }) },
  },
  {
    nombre: 'fijas_ausentes_valen_1',
    instante_millis: I.simple,
    perfil: perfil('Socio Fijas', 'USDT', {
      ECU: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PAN: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PYUSD: { activo: true, porcentaje: { deposito: 3, pago: 3 } },
      USD: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      USDT: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T040', tasas: sorted({}) },
    lote_anterior: null,
  },
  {
    nombre: 'cero_numerico_vs_cero_string',
    instante_millis: I.simple,
    perfil: perfil('Socio Cero', 'USDT', {
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PEN: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      VES: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T050', tasas: sorted({ COP: 4050, PEN: '0', VES: 0 }) },
    lote_anterior: null,
  },
  {
    nombre: 'margen_negativo_y_alto',
    instante_millis: I.simple,
    perfil: perfil('Socio Margen', 'PEN', {
      COP: { activo: true, porcentaje: { deposito: -7.5, pago: 12.25 } },
      PYG: { activo: true, porcentaje: { deposito: -100, pago: 50 } },
    }),
    lote_actual: { id_tasa: 'T060', tasas: sorted(TASAS) },
    lote_anterior: { id_tasa: 'T059', tasas: sorted(TASAS) },
  },
  {
    nombre: 'rangos_de_precision',
    instante_millis: I.otroMes,
    perfil: perfil('Socio Rangos', 'PEN', {
      CLP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      EUR: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PYG: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T070', tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'rate_ruidoso_muchos_decimales',
    instante_millis: I.simple,
    perfil: perfil('Socio Ruido', 'USD', {
      COP: { activo: true, porcentaje: { deposito: 2.5, pago: 7.5 } },
      PEN: { activo: true, porcentaje: { deposito: 1, pago: 1 } },
    }),
    lote_actual: { id_tasa: 'T080', tasas: sorted({ COP: 4050.0000000001, PEN: 3.7489999999999996 }) },
    lote_anterior: null,
  },
  {
    nombre: 'rate_string_con_separador_de_miles',
    instante_millis: I.simple,
    perfil: perfil('Socio String', 'USDT', {
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PEN: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T090', tasas: sorted({ COP: '4,050.50', PEN: '  3.745 ' }) },
    lote_anterior: null,
  },
  {
    nombre: 'clave_en_minuscula',
    instante_millis: I.simple,
    perfil: perfil('Socio Minuscula', 'USDT', {
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      pen: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T110', tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'config_null_y_config_vacio',
    instante_millis: I.simple,
    perfil: perfil('Socio Config', 'USDT', {
      COP: null,
      EUR: {},
      PEN: { activo: true, porcentaje: { deposito: 1, pago: 1 } },
    }),
    lote_actual: { id_tasa: 'T120', tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'monedas_vacio',
    instante_millis: I.simple,
    perfil: perfil('Socio Sin Monedas', 'USDT', {}),
    lote_actual: { id_tasa: 'T130', tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'sin_metadatos_usa_defaults',
    instante_millis: I.simple,
    perfil: { nombre: 'Socio Defaults', monedas: sorted({ COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } } }) },
    lote_actual: { tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'moneda_base_en_minuscula',
    instante_millis: I.simple,
    perfil: perfil('Socio Base Minuscula', 'pen', {
      COP: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      PEN: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T150', tasas: sorted(TASAS) },
    lote_anterior: null,
  },
  {
    nombre: 'moneda_fuera_de_catalogo',
    instante_millis: I.simple,
    perfil: perfil('Socio Exotico', 'USDT', {
      XAU: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
      ZZZ: { activo: true, porcentaje: { deposito: 0, pago: 0 } },
    }),
    lote_actual: { id_tasa: 'T140', tasas: sorted({ XAU: 2400.5, ZZZ: 2 }) },
    lote_anterior: null,
  },
];

const precisionInputs = [
  0, 0.5, -0.5, 1e-9, 1.5e-8, 1e-8, 9e-8, 1e-7, 0.00012345678, 0.0099999999, 0.01, 0.099999999,
  0.1, 0.2456, 0.99999999, 1.00000001, 2.5, 3.7489999999999996, 9.99, 9.995, 9.999, 9.9999,
  9.999999999, 10, 10.001, 10.004999999, 12.3456, 99.99, 99.995, 99.999999, 100, 100.5,
  1234.5678, 4050.505, 7300.0000001, -123.4567, -99.995,
];

const tiempoInstantes = [
  '2026-01-15T12:00:00Z', '2026-02-15T12:00:00Z', '2026-03-01T00:10:00Z', '2026-04-15T12:00:00Z',
  '2026-05-15T12:00:00Z', '2026-06-15T04:05:00Z', '2026-07-15T12:00:00Z', '2026-08-15T12:00:00Z',
  '2026-09-15T12:00:00Z', '2026-10-18T02:35:00Z', '2026-11-15T12:00:00Z', '2026-12-31T23:59:00Z',
  '2026-01-01T04:00:00Z',
];

const salida = {
  generado_por: 'node calculator.service.js (caracterizacion del comportamiento actual)',
  origen: 'old/src/modules/tasas/services/calculator.service.js + old/src/utils/formatters.js',
  nota: 'Cada caso congela el reloj con su instante_millis, de modo que fecha_actualizacion y hora_actualizacion son reproducibles. El orden de tarjetas_paises sigue el del objeto monedas tal como lo inserta este generador (claves ordenadas).',
  casos: [],
  precision: [],
  tiempo: [],
};

for (const caso of casos) {
  FIXED = caso.instante_millis;
  const esperado = calcularTasasCartelera(caso.perfil, caso.lote_actual, caso.lote_anterior);
  salida.casos.push({
    nombre: caso.nombre,
    instante_millis: caso.instante_millis,
    perfil: caso.perfil,
    lote_actual: caso.lote_actual,
    lote_anterior: caso.lote_anterior,
    esperado,
  });
}

for (const entrada of precisionInputs) {
  salida.precision.push({ entrada, esperado: aplicarReglaPrecisionTasa(entrada) });
}

for (const iso of tiempoInstantes) {
  FIXED = Date.parse(iso);
  const d = new Date();
  salida.tiempo.push({
    millis: FIXED,
    fecha: d.toLocaleDateString('es-VE', dateOpts).replace('.', ''),
    hora: d.toLocaleTimeString('es-VE', hourOpts),
  });
}

writeFileSync(outPath, `${JSON.stringify(salida, null, 2)}\n`, 'utf8');

// Autoverificación: las fechas de los casos deben corresponder a su instante.
let avisos = 0;
for (const caso of salida.casos) {
  FIXED = caso.instante_millis;
  const d = new Date();
  const fecha = d.toLocaleDateString('es-VE', dateOpts).replace('.', '');
  const hora = d.toLocaleTimeString('es-VE', hourOpts);
  if (fecha !== caso.esperado.fecha_actualizacion || hora !== caso.esperado.hora_actualizacion) {
    avisos++;
    console.error(`AVISO: ${caso.nombre} fecha/hora no reproducible (${fecha} ${hora})`);
  }
}
console.log(`casos=${salida.casos.length} precision=${salida.precision.length} tiempo=${salida.tiempo.length} avisos=${avisos}`);
