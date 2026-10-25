-- P-03: unicidad de los identificadores de lote.
--
-- El código Core serializa la lectura del último lote y el INSERT con un
-- advisory lock. Este índice es la red de seguridad para cualquier escritor
-- externo (por ejemplo, el servicio legado) que no participe de ese lock.
--
-- Antes de aplicarlo, verificar duplicados. Si existen, resolverlos a mano:
-- no hay una regla automática que decida qué lote histórico conservar.
--
--   SELECT lower(btrim(id_tasa)) AS id, count(*)
--   FROM tasas_glaukov
--   WHERE id_tasa IS NOT NULL
--   GROUP BY 1
--   HAVING count(*) > 1;
--
-- La comparación es case-insensitive y sin espacios porque el resto del
-- dominio busca los lotes con LOWER(id_tasa).

CREATE UNIQUE INDEX IF NOT EXISTS ux_tasas_glaukov_id_tasa
    ON tasas_glaukov (lower(btrim(id_tasa)))
    WHERE id_tasa IS NOT NULL;
