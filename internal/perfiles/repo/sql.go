package repo

// perfilColumns normaliza NULLs para el escaneo estricto de Go y fija el orden
// que consume scanPerfil. Ninguna otra capa enumera columnas de perfiles_glaukov.
const perfilColumns = `
  COALESCE(id_grupo::text, '') AS id_grupo,
  COALESCE(nombre::text, '') AS nombre,
  COALESCE(rol::text, 'SOCIO') AS rol,
  COALESCE(moneda_base::text, 'USDT') AS moneda_base,
  COALESCE(saldo_inicial::text, '0') AS saldo_inicial,
  COALESCE(mostrar, '{}'::jsonb) AS mostrar,
  COALESCE(monedas, '{}'::jsonb) AS monedas,
  COALESCE(herencia, false) AS herencia`

const returningPerfil = ` RETURNING ` + perfilColumns

// ListarSQL devuelve el directorio completo ordenado por nombre.
const ListarSQL = `SELECT ` + perfilColumns + ` FROM perfiles_glaukov ORDER BY nombre ASC`

// PorNombreSQL busca por identidad canónica sin distinguir mayúsculas ni espacios.
const PorNombreSQL = `SELECT ` + perfilColumns + `
FROM perfiles_glaukov
WHERE UPPER(BTRIM(nombre)) = UPPER(BTRIM($1))`

// ActivosSQL aplica H-02 sobre mostrar. Una clave tasas ausente cuenta como
// true, igual que COALESCE((mostrar->>'tasas')::boolean, true) del plan; solo un
// false explícito se excluye. La columna opcional activo solo podría apagar y el
// contrato de esta fase la trata como derivada, sin escribirla.
const ActivosSQL = `
SELECT DISTINCT BTRIM(nombre) AS nombre
FROM perfiles_glaukov
WHERE nombre IS NOT NULL
  AND BTRIM(nombre) <> ''
  AND (
    mostrar @> '{"tasas": true}'::jsonb
    OR NOT (COALESCE(mostrar, '{}'::jsonb) ? 'tasas')
  )
ORDER BY nombre ASC`

// UpsertSQL es atómico y requiere el índice único sobre UPPER(BTRIM(nombre)).
// monedas se pasa ya codificado por el codec superset de repo/monedas_codec.go.
const UpsertSQL = `
INSERT INTO perfiles_glaukov (id_grupo, nombre, rol, moneda_base, saldo_inicial, mostrar, monedas, herencia)
VALUES ($1, UPPER(BTRIM($2)), $3, $4, $5::text::numeric, $6::jsonb, $7::jsonb, $8)
ON CONFLICT ((UPPER(BTRIM(nombre)))) DO UPDATE SET
  id_grupo = EXCLUDED.id_grupo,
  rol = EXCLUDED.rol,
  moneda_base = EXCLUDED.moneda_base,
  saldo_inicial = EXCLUDED.saldo_inicial,
  mostrar = EXCLUDED.mostrar,
  monedas = EXCLUDED.monedas,
  herencia = EXCLUDED.herencia` + returningPerfil

// ToggleSQL escribe únicamente mostrar.tasas. No toca la columna legada activo:
// el campo público se deriva con H-02 después de leer la fila resultante.
const ToggleSQL = `
UPDATE perfiles_glaukov
SET mostrar = jsonb_set(
  COALESCE(mostrar, '{"tasas": true, "dashboard": true}'::jsonb),
  '{tasas}', to_jsonb($1::boolean))
WHERE UPPER(BTRIM(nombre)) = UPPER(BTRIM($2))` + returningPerfil

// NormalizarMonedasSQL persiste la forma canónica de una fila histórica. El
// tercer argumento es el JSONB tal como se leyó: la condición evita pisar una
// escritura concurrente entre el SELECT del lector y este UPDATE. Si la fila ya
// cambió, afecta 0 filas y el lector devuelve igualmente la vista canonicalizada
// en memoria, porque el siguiente guardado del socio persistirá el superset.
const NormalizarMonedasSQL = `
UPDATE perfiles_glaukov
SET monedas = $1::jsonb
WHERE UPPER(BTRIM(nombre)) = UPPER(BTRIM($2))
  AND COALESCE(monedas, '{}'::jsonb) = $3::jsonb`

// DeleteSQL devuelve el control de filas afectadas al writer.
const DeleteSQL = `DELETE FROM perfiles_glaukov WHERE UPPER(BTRIM(nombre)) = UPPER(BTRIM($1))`

// ApagarTodosSQL desactiva tasas en todos los socios salvo GENERAL, la matriz.
const ApagarTodosSQL = `
UPDATE perfiles_glaukov
SET mostrar = jsonb_set(
  COALESCE(mostrar, '{}'::jsonb),
  '{tasas}', to_jsonb(UPPER(BTRIM(nombre)) = 'GENERAL'))
WHERE nombre IS NOT NULL`

// MemorizarSQL congela el estado vigente sin alterar la visibilidad actual.
const MemorizarSQL = `UPDATE perfiles_glaukov SET recordar_mostrar = mostrar`

// RestaurarSQL recupera la plantilla y fuerza GENERAL activo, como el legado.
const RestaurarSQL = `
UPDATE perfiles_glaukov
SET mostrar = CASE
  WHEN UPPER(BTRIM(nombre)) = 'GENERAL'
    THEN jsonb_set(COALESCE(recordar_mostrar, '{}'::jsonb), '{tasas}', 'true'::jsonb)
  ELSE COALESCE(recordar_mostrar, '{"tasas": false, "dashboard": false}'::jsonb)
END`
