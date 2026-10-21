-- Auditoria pre-cutover de Glauk (Fase 15, 7.16).
-- SOLO LECTURA: este archivo no contiene INSERT, UPDATE, DELETE ni DDL.
-- Uso: psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/audit-pre-cutover.sql
-- Cada bloque devuelve unicamente las filas que requieren revision humana.

BEGIN TRANSACTION READ ONLY;
SET LOCAL statement_timeout = '30s';

-- ============================================================
-- BLOQUE 01 | Duplicados por nombre canonico
-- VALIDA: UPPER(BTRIM(nombre)) repetido; el indice unico fallaria.
-- ACCION: resolver duplicados con revision humana.
-- ============================================================
SELECT UPPER(BTRIM(nombre)) AS canonico,
       COUNT(*) AS cantidad,
       STRING_AGG(COALESCE(nombre, '(sin nombre)'), ' | ' ORDER BY nombre) AS nombres
FROM perfiles_glaukov
GROUP BY UPPER(BTRIM(nombre))
HAVING COUNT(*) > 1
ORDER BY cantidad DESC, canonico;

-- ============================================================
-- BLOQUE 02 | JIDs invalidos, vacios o placeholder
-- VALIDA: formato estricto de 3.4; @c.us se reporta como legado y *_JID como placeholder.
-- ACCION: corregir JIDs a mano; no inventar numeros.
-- ============================================================
WITH normalizados AS (
    SELECT nombre, id_grupo,
           REGEXP_REPLACE(LOWER(BTRIM(id_grupo)), '^whatsapp:', '') AS jid
    FROM perfiles_glaukov
), clasificados AS (
    SELECT nombre, id_grupo, jid,
           CASE
               WHEN jid IS NULL OR jid = '' THEN 'vacio'
               WHEN RIGHT(jid, 4) = '_jid' THEN 'placeholder'
               WHEN jid ~ '^([0-9]{5,25}|[0-9]{5,25}-[0-9]{5,25})@(g\.us|s\.whatsapp\.net|lid)$'
                   THEN 'ok'
               WHEN jid ~ '^([0-9]{5,25}|[0-9]{5,25}-[0-9]{5,25})@c\.us$'
                   THEN 'legacy_c_us'
               ELSE 'invalido'
           END AS estado
    FROM normalizados
)
SELECT nombre, id_grupo, jid AS jid_normalizado, estado
FROM clasificados
WHERE estado <> 'ok'
ORDER BY estado, nombre;

-- ============================================================
-- BLOQUE 03 | Formas de monedas y conflictos
-- VALIDA: presencia de porcentaje, deposito y tipo; cuantas formas coexisten.
-- ACCION: resolver por superset; porcentaje manda.
-- ============================================================
SELECT nombre,
       COALESCE(monedas ? 'porcentaje', FALSE) AS porcentaje,
       COALESCE(monedas ? 'deposito', FALSE) AS deposito,
       COALESCE(monedas ? 'tipo', FALSE) AS tipo,
       CASE
           WHEN monedas IS NULL THEN 'null'
           WHEN monedas ? 'porcentaje' THEN 'tasas'
           WHEN monedas ? 'deposito' THEN 'perfiles'
           WHEN monedas ? 'tipo' THEN 'comprobantes'
           ELSE 'desconocida'
       END AS forma,
       (COALESCE((monedas ? 'porcentaje')::int, 0) +
        COALESCE((monedas ? 'deposito')::int, 0) +
        COALESCE((monedas ? 'tipo')::int, 0)) AS formas_presentes
FROM perfiles_glaukov
ORDER BY formas_presentes DESC, nombre;

-- ============================================================
-- BLOQUE 04 | Actividad: mostrar.tasas contra activo
-- VALIDA: discrepancias; activo solo puede apagar y se ignora si la columna no existe.
-- ACCION: aprobar actividad; mostrar.tasas manda.
-- ============================================================
WITH base AS (
    SELECT nombre, mostrar, to_jsonb(p) AS fila
    FROM perfiles_glaukov p
), parsed AS (
    SELECT nombre,
           CASE LOWER(BTRIM(mostrar->>'tasas'))
               WHEN 'true' THEN TRUE WHEN 't' THEN TRUE WHEN '1' THEN TRUE
               WHEN 'false' THEN FALSE WHEN 'f' THEN FALSE WHEN '0' THEN FALSE
               ELSE NULL
           END AS mostrar_tasas,
           CASE LOWER(BTRIM(fila->>'activo'))
               WHEN 'true' THEN TRUE WHEN 't' THEN TRUE WHEN '1' THEN TRUE
               WHEN 'false' THEN FALSE WHEN 'f' THEN FALSE WHEN '0' THEN FALSE
               ELSE NULL
           END AS activo,
           (fila ? 'activo') AS tiene_activo,
           mostrar->>'tasas' AS mostrar_raw,
           fila->>'activo' AS activo_raw
    FROM base
)
-- Equivale a COALESCE((mostrar->>'tasas')::boolean, true), sin abortar por un valor no booleano.
SELECT nombre,
       COALESCE(mostrar_tasas, TRUE) AS mostrar_tasas,
       activo,
       tiene_activo,
       mostrar_raw,
       activo_raw
FROM parsed
WHERE tiene_activo
  AND COALESCE(mostrar_tasas, TRUE) <> COALESCE(activo, TRUE)
ORDER BY nombre;

-- ============================================================
-- BLOQUE 05 | Lotes de tasas con created_at nulo o id_tasa no canonico
-- VALIDA: created_at nulo o id_tasa fuera de ^T[0-9]{3}$.
-- ACCION: corregir el lote con backup.
-- ============================================================
SELECT id, id_tasa, created_at
FROM tasas_glaukov
WHERE created_at IS NULL
   OR id_tasa IS NULL
   OR id_tasa !~ '^T[0-9]{3}$'
ORDER BY id DESC;

-- ============================================================
-- BLOQUE 06 | Comprobantes huerfanos de liquidacion
-- VALIDA: comprobantes_raw sin fila equivalente en comprobantes_liq.
-- ACCION: liquidar o justificar el huerfano.
-- ============================================================
SELECT c.hash_largo
FROM comprobantes_raw c
LEFT JOIN comprobantes_liq l
  ON LOWER(TRIM(l.hash_largo)) = LOWER(TRIM(c.hash_largo))
WHERE l.hash_largo IS NULL
ORDER BY c.hash_largo;

COMMIT;
