-- 0001_init.sql — referencia de esquema, NO se ejecuta automáticamente.
--
-- La tabla perfiles_glaukov es compartida con Tasas y con la base histórica de
-- Glauk. Este dominio no la crea: solo documenta el contrato que consumen el
-- repositorio y el codec. Columnas esperadas:
--
--   id_grupo       text      JID del grupo; puede quedar '' si el alta no lo trae
--   nombre         text      identidad del socio, único por UPPER(BTRIM(nombre))
--   rol            text      SOCIO por defecto
--   moneda_base    text      USDT por defecto
--   saldo_inicial  numeric   se lee como text para conservar el contrato node-pg
--   mostrar        jsonb     {tasas:boolean, dashboard:boolean}
--   monedas        jsonb     mapa ISO -> superset de reglas
--   herencia       boolean   usa las reglas de la matriz FUNDDA/GENERAL
--
-- Antes de aplicar el índice, verificar duplicados:
--   SELECT UPPER(BTRIM(nombre)), count(*)
--   FROM perfiles_glaukov
--   GROUP BY 1 HAVING count(*) > 1;
-- Si hay duplicados, resolverlos primero: el índice único fallará.

BEGIN;

ALTER TABLE perfiles_glaukov
  ADD COLUMN IF NOT EXISTS recordar_mostrar JSONB;

CREATE UNIQUE INDEX IF NOT EXISTS ux_perfiles_glaukov_nombre
  ON perfiles_glaukov (UPPER(BTRIM(nombre)));

COMMIT;
