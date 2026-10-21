# Migraciones de referencia

Este directorio es el índice para el operador. La aplicación **no ejecuta migraciones automáticamente** al arrancar; cada migración se aplica con `psql`, después de un backup verificado y dentro de la ventana de cutover.

| Orden | Migración | Propósito | Idempotencia |
|---:|---|---|---|
| 0001 | [`internal/comprobantes/migrations/0001_impactos_soft_delete.sql`](../internal/comprobantes/migrations/0001_impactos_soft_delete.sql) | Agrega `impactos_raw.eliminado_en` y sus índices para el borrado lógico. | `ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`. |

## Aplicación

Desde `Core/`:

```powershell
psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 `
  -f internal/comprobantes/migrations/0001_impactos_soft_delete.sql
```

La migración es compatible con el servicio viejo: agrega una columna nullable y un índice, sin borrar ni reescribir filas. `CREATE INDEX` puede tomar un bloqueo breve; aplicarla en ventana de mantenimiento y con backup verificado.

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
```

Las migraciones nuevas se agregan a la tabla en orden y deben ser idempotentes. El seed `scripts/local-db-init.sql` es solo para el stack local y no sustituye este índice.
