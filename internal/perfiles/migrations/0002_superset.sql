-- 0002_superset.sql — auditoría de referencia, NO se ejecuta automáticamente.
--
-- El codec de Go ya escribe el superset completo en la primera escritura de cada
-- socio. Esta migración es deliberadamente no destructiva: solo consulta. Una
-- normalización masiva queda para la fase 12, con backup, comparación de
-- fixtures y ventana de mantenimiento. Nunca debe ejecutarse desde un request.

BEGIN;

-- 1. Conflictos H-01: porcentaje y el campo legacy difieren. La lectura ya
-- resuelve a favor de porcentaje; esta consulta lista lo que se reconciliará en
-- la próxima escritura del socio.
SELECT
  p.nombre,
  m.key AS moneda,
  m.value->'porcentaje'->>'deposito' AS porcentaje_deposito,
  m.value->>'deposito' AS legacy_deposito,
  m.value->'porcentaje'->>'pago' AS porcentaje_pago,
  m.value->>'pago' AS legacy_pago
FROM perfiles_glaukov AS p
CROSS JOIN LATERAL jsonb_each(COALESCE(p.monedas, '{}'::jsonb)) AS m
WHERE m.value ? 'porcentaje'
  AND (
    (m.value ? 'deposito' AND (m.value->'porcentaje'->>'deposito') IS DISTINCT FROM (m.value->>'deposito'))
    OR
    (m.value ? 'pago' AND (m.value->'porcentaje'->>'pago') IS DISTINCT FROM (m.value->>'pago'))
  )
ORDER BY p.nombre, m.key;

-- 2. Cobertura del superset por moneda. Las filas con false se completarán en la
-- siguiente escritura; no se actualizan aquí.
SELECT
  p.nombre,
  m.key AS moneda,
  NOT (m.value ? 'activo') AS falta_activo,
  NOT (m.value ? 'porcentaje') AS falta_porcentaje,
  NOT (m.value ? 'def') AS falta_def,
  NOT (m.value ? 'dep_op') AS falta_dep_op,
  NOT (m.value ? 'pago_op') AS falta_pago_op,
  NOT (m.value ? 'tipo') AS falta_tipo,
  NOT (m.value ? 'polar_D') AS falta_polar_D,
  NOT (m.value ? 'polar_P') AS falta_polar_P,
  NOT (m.value ? 'polaridad') AS falta_polaridad
FROM perfiles_glaukov AS p
CROSS JOIN LATERAL jsonb_each(COALESCE(p.monedas, '{}'::jsonb)) AS m
WHERE NOT (m.value @> '{"activo": true}'::jsonb)
   OR NOT (m.value ? 'def')
   OR NOT (m.value ? 'tipo')
ORDER BY p.nombre, m.key;

-- 3. Sin UPDATE por defecto: cualquier normalización masiva requiere el plan de
-- fase 12. El repositorio es el único escritor y emite el superset al guardar.

COMMIT;
