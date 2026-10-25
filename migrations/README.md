# Migraciones de referencia

Este directorio es el índice para el operador. La aplicación **no ejecuta migraciones automáticamente** al arrancar; cada migración se aplica con `psql`, después de un backup verificado y dentro de la ventana de cutover.

| Orden | Migración | Propósito | Idempotencia |
|---:|---|---|---|
| 0001 | [`internal/comprobantes/migrations/0001_impactos_soft_delete.sql`](../internal/comprobantes/migrations/0001_impactos_soft_delete.sql) | Agrega `impactos_raw.eliminado_en` y sus índices para el borrado lógico. | `ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`. |
| 0002 | [`internal/ingesta/migrations/0001_impactos_dedupe.sql`](../internal/ingesta/migrations/0001_impactos_dedupe.sql) | Marca duplicados históricos y crea el índice único parcial que hace atómico el `INSERT ... ON CONFLICT` de impactos. | `CREATE UNIQUE INDEX IF NOT EXISTS`; el marcado de duplicados es idempotente porque solo toca `eliminado_en IS NULL`. |
| 0003 | [`internal/tasas/migrations/0001_lote_id_unico.sql`](../internal/tasas/migrations/0001_lote_id_unico.sql) | Crea el índice único de `tasas_glaukov.id_tasa` como respaldo del advisory lock de creación de lotes. | `CREATE UNIQUE INDEX IF NOT EXISTS`. |
| 0004 | [`internal/comprobantes/migrations/0002_list_indexes.sql`](../internal/comprobantes/migrations/0002_list_indexes.sql) | Índices funcionales para el listado: hash normalizado, instancia/fecha, socios, perfiles y lote histórico. | `CREATE INDEX IF NOT EXISTS`. |

## Aplicación

Desde `Core/`:

```powershell
psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 `
  -f internal/comprobantes/migrations/0001_impactos_soft_delete.sql

psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 `
  -f internal/ingesta/migrations/0001_impactos_dedupe.sql

psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 `
  -f internal/tasas/migrations/0001_lote_id_unico.sql

psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 `
  -f internal/comprobantes/migrations/0002_list_indexes.sql
```

La migración `0002` es obligatoria antes de desplegar el `INSERT ... ON CONFLICT` de impactos: sin el índice único, PostgreSQL responde `42P10`. La `0003` exige resolver antes cualquier `T###` duplicado; el bloque de verificación del archivo lista los conflictos.

La migración `0001` es compatible con el servicio viejo: agrega una columna nullable y un índice, sin borrar ni reescribir filas. La `0002` conserva los duplicados como evidencia con `eliminado_en` y crea el índice único parcial. La `0003` solo agrega el índice único de lotes. `CREATE INDEX` puede tomar un bloqueo breve; aplicarlas en ventana de mantenimiento y con backup verificado.

## Verificación posterior

```sql
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'impactos_raw'
  AND column_name = 'eliminado_en';

SELECT indexname
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'impactos_raw'
  AND indexname IN ('impactos_raw_eliminado_en_idx', 'impactos_raw_vigentes_idx');

SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND indexname IN ('ux_impactos_raw_dedupe', 'ux_tasas_glaukov_id_tasa');

SELECT indexname
FROM pg_indexes
WHERE schemaname = 'public'
  AND indexname IN (
    'comprobantes_raw_creado_en_idx',
    'comprobantes_raw_instancia_creado_idx',
    'comprobantes_raw_hash_lower_idx',
    'comprobantes_liq_hash_lower_idx',
    'impactos_raw_vigentes_hash_id_idx',
    'perfiles_glaukov_id_grupo_lower_idx',
    'tasas_glaukov_created_id_idx',
    'comprobantes_liq_socio1_lower_idx',
    'comprobantes_liq_socio2_lower_idx'
  );
```

Las migraciones nuevas se agregan a la tabla en orden y deben ser idempotentes. El seed `scripts/local-db-init.sql` es solo para el stack local y no sustituye este índice.
