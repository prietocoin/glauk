# Glauk — SUPER-README

Documento maestro de arquitectura, operación y contratos del monolito modular **Glauk**.

| Campo | Valor |
|---|---|
| Versión | 0.2.0 (configuración y logging) |
| Fecha | 2026-10-20 |
| Estado | Fase 2 — `platform/config` y `platform/logging` |
| Módulo | `glauk` |
| API | `v1` |
| Fuente normativa | `../PLAN_CONSOLIDACION_GLAUK.md` |

> Este documento se completa de forma incremental en cada fase del runbook. Los títulos siguientes son el índice maestro normativo; ninguna sección se considera cerrada hasta cumplir sus puertas de calidad.

## 1. Portada y estado del proyecto

Versión, fecha, commit, alcance, licencia, autoría, enlaces a ADRs y al plan de consolidación.

## 2. Índice y convenciones

Anchors, glosario mínimo, notación RFC 2119 y cómo leer las matrices de contratos.

## 3. Visión y arquitectura

Contexto, diagrama de contenedores, módulo único, cuatro dominios, reglas de dependencia y por qué no microservicios.

## 4. Arranque rápido

Requisitos, `.env`, PostgreSQL local, `go run`, Docker, modo simulación, primer request y solución de problemas.

## 5. Configuración

> Fase 2 implementada. `platform/config` es el único paquete autorizado a leer el entorno; el resto recibe `*config.Config` por inyección.

### 5.1 Fuente única y precedencia

Solo `internal/platform/config` importa `os` para leer variables y `godotenv`. Cualquier otro paquete que necesite un valor lo recibe a través de `*config.Config`.

Orden de resolución:

```text
1. Variable de proceso (Easypanel / docker run / shell)
2. Archivo .env del directorio de trabajo o del directorio del ejecutable,
   SOLO si APP_ENV != production
3. Default declarado en la matriz
```

Reglas específicas:

1. **Base de datos:** si `DATABASE_URL` no está vacía se usa tal cual. `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` y `DB_SSLMODE` solo construyen el DSN cuando `DATABASE_URL` está vacía. Las variables de pool y timeouts sí se aplican siempre.
2. **DSN compuesto:** se construye con `net/url`, escapando usuario y contraseña. Una contraseña con `@`, `:` o `/` no rompe la URL. En producción se exige `DB_PASSWORD` cuando el DSN es compuesto.
3. **Claves de Evolution:** `EVOLUTION_API_KEY` gana sobre `AUTHENTICATION_API_KEY`. Si ambas están definidas y difieren, `Load()` falla nombrando las dos variables, sin imprimir sus valores.
4. **Instancia:** `EVOLUTION_INSTANCE` vacía equivale a `Glauk`; nunca se usa el hostname ni el nombre de la base.
5. **`.env`:** se busca en el directorio de trabajo y en el del ejecutable. En `production` no se lee, aunque el archivo exista. El archivo `.env` no se versiona; `.env.example` es la plantilla.
6. **APP_ENV:** se resuelve antes de cargar `.env`; en `production` las validaciones estrictas se aplican aunque exista un `.env` local.

### 5.2 Matriz consolidada de variables

Leyenda de obligatoriedad: **S** = siempre; **P** = solo en `APP_ENV=production`; **C** = condicional (si se usa la capacidad); **N** = nunca (tiene default). El dominio indica quién consume el valor, no quién lo lee.

| Variable | Tipo | Default | Oblig. | Dominio | Regla / efecto |
|---|---|---:|---:|---|---|
| `APP_ENV` | string | `development` | N | global | `development`, `staging`, `production`. En `production` no se carga `.env` y se activan validaciones estrictas. |
| `PORT` | int | `3000` | N | global | 1–65535. Un solo puerto para los cuatro dominios. |
| `PUBLIC_DIR` | path | `./public` | N | httpx | Raíz de los cuatro frontends y del lanzador `index.html`. En el contenedor es `/app/public`. |
| `LOG_LEVEL` | string | `info` | N | logging | `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | string | `auto` | N | logging | `json` en producción, `text` en desarrollo; `auto` decide. |
| `DATABASE_URL` | url | — | P/C | postgres | `postgres://` o `postgresql://`. Si está presente, tiene prioridad total sobre `DB_*`. |
| `DB_HOST` | string | `localhost` | N | postgres | Solo se usa si `DATABASE_URL` está vacío. |
| `DB_PORT` | int | `5432` | N | postgres | 1–65535. |
| `DB_USER` | string | `postgres` | N | postgres | Solo DSN compuesto. |
| `DB_PASSWORD` | secret | `""` | P/C | postgres | Obligatoria en producción si se compone el DSN; nunca se serializa ni se registra. |
| `DB_NAME` | string | `automat` | N | postgres | Base histórica de los tres servicios. |
| `DB_SSLMODE` | string | `disable` | N | postgres | `disable`, `require`, `verify-ca`, `verify-full`. En producción se exige `require` o superior salvo `DB_ALLOW_INSECURE=true`. |
| `DB_ALLOW_INSECURE` | bool | `false` | N | postgres | Permite `sslmode=disable` en producción con un `WARN` explícito. |
| `DB_MAX_CONNS` | int | `20` | N | postgres | ≥ `DB_MIN_CONNS`. |
| `DB_MIN_CONNS` | int | `2` | N | postgres | ≥ 0 y ≤ `DB_MAX_CONNS`. |
| `DB_CONN_MAX_LIFETIME` | duration | `1h` | N | postgres | Formato Go (`30m`, `1h`). |
| `DB_CONN_MAX_IDLE_TIME` | duration | `30m` | N | postgres | Formato Go. |
| `DB_CONNECT_TIMEOUT` | duration | `10s` | N | postgres | Incluye ping de arranque. |
| `DB_STATEMENT_TIMEOUT` | duration | `10s` | N | postgres | Se aplica como `RuntimeParams["statement_timeout"]`. |
| `EVOLUTION_API_URL` | url | `""` | C | evolution | Vacío = modo simulación. Si se define, debe ser `http(s)://host` y se normaliza sin `/` final. |
| `EVOLUTION_API_KEY` | secret | `""` | C | evolution | Clave canónica. Si hay URL, debe existir (o su alias). |
| `AUTHENTICATION_API_KEY` | secret | `""` | N | evolution | Alias legado de `EVOLUTION_API_KEY`. Si ambos difieren, el arranque falla. |
| `EVOLUTION_INSTANCE` | string | `Glauk` | N | evolution | Instancia oficial. Case-sensitive; por defecto `Glauk`. |
| `EVOLUTION_TIMEOUT` | duration | `25s` | N | evolution | Timeout por intento. |
| `EVOLUTION_MAX_RETRIES` | int | `2` | N | evolution | 0–5. |
| `EVOLUTION_RETRY_BASE` | duration | `300ms` | N | evolution | Backoff exponencial con jitter. |
| `ADMIN_API_KEY` | secret | `""` | N | comprobantes | Si está vacía, las rutas admin responden `503 ADMIN_DISABLED`. Nunca hay clave por defecto. |
| `SECURITY_API_KEY` | secret | `""` | N | httpx | Si está vacía, las mutaciones quedan abiertas por compatibilidad. Si está definida, se exige `X-API-Key`. |
| `CORS_ALLOWED_ORIGINS` | csv | `*` | N | httpx | En producción debe ser una lista explícita; `*` es error. |
| `TRUSTED_PROXY_HEADER` | string | `""` | N | httpx | `X-Forwarded-Host` o `X-Forwarded-Proto`; vacío = no confiar. |
| `HOST_ROUTING_MODE` | string | `both` | N | httpx | `host`, `path`, `both`. Controla el aislamiento de frontends. |
| `DEFAULT_TENANT` | enum | `comprobantes` | N | httpx | `tasas`, `perfiles`, `comprobantes`, `reportes` o `none`. `none` habilita el lanzador `public/index.html`. |
| `TENANT_TASAS_HOSTS` | csv | `tasas.localhost` | N | httpx | Hosts (sin puerto) que sirven el panel de Tasas. |
| `TENANT_PERFILES_HOSTS` | csv | `perfiles.localhost` | N | httpx | Hosts del panel de Perfiles. |
| `TENANT_COMPROBANTES_HOSTS` | csv | `comprobantes.localhost` | N | httpx | Hosts del panel de Comprobantes. |
| `TENANT_REPORTES_HOSTS` | csv | `reportes.localhost` | N | httpx | Hosts del panel de Reportes. |
| `BODY_LIMIT_BYTES` | bytes | `20971520` | N | httpx | Acepta `20MiB`, `20MB` o entero. `MB` y `MiB` equivalen a `1 << 20`. |
| `HTTP_READ_TIMEOUT` | duration | `15s` | N | httpx | Lectura de cabeceras/cuerpo. |
| `HTTP_WRITE_TIMEOUT` | duration | `90s` | N | httpx | Cota del ciclo completo. |
| `HTTP_IDLE_TIMEOUT` | duration | `60s` | N | httpx | Keep-alive. |
| `SHUTDOWN_TIMEOUT` | duration | `15s` | N | cmd/server | Apagado ordenado. |
| `REPORT_IMAGE_MAX_BYTES` | bytes | `10485760` | N | reportes | Bytes ya decodificados. |
| `REPORT_IMAGE_MIN_DIMENSION` | int | `200` | N | reportes | Lado mínimo en px. |
| `REPORT_IMAGE_MAX_DIMENSION` | int | `4096` | N | reportes | Lado máximo en px. |
| `REPORT_IMAGE_MAX_PIXELS` | int | `16000000` | N | reportes | Ancho × alto. |
| `REPORT_IDEMPOTENCY_TTL` | duration | `10m` | N | reportes | Ventana de deduplicación. |
| `REPORT_RATE_LIMIT_PER_MINUTE` | int | `60` | N | reportes | 0 desactiva; no recomendado en producción. |
| `JID_GRUPO_PRUEBA` | string | `""` | N | tasas | Grupo destino cuando `modo_prueba` está activo. Si está vacío y se pide modo prueba, se responde `400`. |
| `HOO_GLOBAL_RATES_URL` | url | `https://hoo.jairokov.com/global-rates` | N | tasas | Origen del borrador HOO. |
| `HOO_TIMEOUT` | duration | `10s` | N | tasas | Timeout del cliente HOO. |
| `ENABLE_PPROF` | bool | `false` | N | cmd/server | Solo diagnóstico; nunca en producción pública. |
| `TZ` | string | `UTC` | N | global | Zonas explícitas para sellos de fecha. |

### 5.3 Validación fail-fast

`config.Load()` lee el entorno una sola vez, aplica defaults y valida el conjunto completo. No aborta en el primer fallo: acumula todos los errores en `config.Errors` y devuelve un único error multilínea. Ejemplo:

```text
configuración inválida (4 errores):
  - PORT debe estar entre 1 y 65535
  - DATABASE_URL es obligatoria en production (o defina DB_HOST/DB_USER/DB_NAME)
  - production exige sslmode=require o superior (o DB_ALLOW_INSECURE=true)
  - EVOLUTION_API_URL es obligatoria en production
```

| Regla | development | staging | production |
|---|---|---|---|
| Fuente de base de datos | defaults locales | obligatoria | obligatoria |
| `DB_PASSWORD` con DSN compuesto | opcional | opcional | obligatoria |
| `sslmode=disable` | permitido | permitido con `WARN` | error salvo `DB_ALLOW_INSECURE=true` |
| `EVOLUTION_API_URL` y clave | ambos o ninguno | ambos o ninguno | ambos o ninguno |
| `EVOLUTION_API_URL` vacía | simulación permitida | simulación permitida con `WARN` | error: no se permite simular |
| `ADMIN_API_KEY` vacía | admin en `503` | admin en `503` + `WARN` | admin en `503` + `WARN`; no aborta |
| `SECURITY_API_KEY` vacía | permitido | `WARN` | `WARN`; no aborta |
| `CORS_ALLOWED_ORIGINS="*"` | permitido | `WARN` | error |
| `DEFAULT_TENANT` o `HOST_ROUTING_MODE` inválido | error | error | error |
| `ENABLE_PPROF=true` | permitido | permitido | error |
| `PORT` fuera de 1–65535 | error | error | error |
| `DB_MIN_CONNS > DB_MAX_CONNS` | error | error | error |

Reglas adicionales:

- Duraciones y tamaños deben ser positivos. `20MiB`, `20MB` y `20971520` producen el mismo valor.
- `EVOLUTION_MAX_RETRIES` debe estar entre 0 y 5.
- `HOO_GLOBAL_RATES_URL` y `EVOLUTION_API_URL` deben ser URLs `http(s)` con host.
- `ADMIN_API_KEY` ausente no aborta: la consola admin es una capacidad opcional. Una URL de Evolution sin clave sí aborta: un servicio que cree estar enviando y no puede autenticarse es peor que un servicio detenido.

### 5.4 Política de secretos y logs

- **Nunca** se registran `Authorization`, `apikey`, `X-API-Key`, `X-Admin-Key`, `DATABASE_URL` completa, `media`/`base64` ni cuerpos de comprobantes.
- `logging.MaskKey` conserva los primeros 4 y últimos 2 caracteres; una clave de 8 caracteres o menos se reduce a `****`.
- `logging.MaskJID("5491123456789@g.us")` devuelve exactamente `54911****@g.us`.
- `logging.MaskPhone` conserva un prefijo y sufijo cortos; los números cortos se reducen a `****`.
- Los errores de configuración nombran variables, nunca valores. `ValidateDSN` no incluye la URL cruda en el error.
- Los logs de request solo llevan `request_id`, `method`, `path`, `status`, `duration_ms`, `bytes` y `tenant`. No incluyen cuerpos, query strings sensibles ni cabeceras de autenticación.
- `LOG_FORMAT=auto` usa JSON en producción y texto en desarrollo. El logger se inyecta; no se usa `slog.Default()` en código de dominio.

### 5.5 Implementación y verificación

| Archivo | Propósito |
|---|---|
| `internal/platform/config/config.go` | Structs inmutables y `Load()`. |
| `config/env.go` | Helpers tipados y parseo de bytes/duraciones/CSV. |
| `config/dsn.go` | DSN compuesto escapado y `ValidateDSN`. |
| `config/validate.go` | `Errors` agregado y validación core/DB/Evolution/HTTP. |
| `config/validate_rules.go` | Reglas de reportes, HOO y producción. |
| `config/load.go` | `build()`: lectura única, defaults y normalización. |
| `config/readers.go` | Adaptadores que acumulan errores de parseo. |
| `config/dotenv.go` | Precedencia proceso → `.env` → default. |
| `config/config_test.go` | Matriz de precedencia, DSN, validación y parseo. |
| `internal/platform/logging/logger.go` | Handler JSON/text y nivel configurable. |
| `logging/request.go` | Campos operativos de request sin secretos. |
| `logging/redact.go` | `MaskKey`, `MaskJID`, `MaskPhone`. |

Verificación:

```powershell
$env:CGO_ENABLED = "1"   # -race requiere cgo y un gcc en PATH (WinLibs en Windows)
go test ./internal/platform/config ./internal/platform/logging -race -v
```

Todos los archivos de producción respetan 30–180 líneas y los tests ≤220 líneas. `gofmt -l .` y `go vet ./...` deben quedar limpios antes de cada commit.


## 6. Dominio Tasas

Modelo, 13 endpoints, cálculo, triangulación, render PNG, HOO, publicación, modo prueba, errores y ejemplos.

## 7. Dominio Perfiles

Modelo, codec superset de `monedas`, actividad H-02, endpoints, plantilla, compatibilidad y migración.

## 8. Dominio Comprobantes

Auditoría, snapshot congelado, reglas financieras, endpoints, consola admin, SQL, proyección y errores.

## 9. Dominio Reportes

Puertos, flujo canvas/JPEG, validación, JID, idempotencia, rate limit, endpoints, ejemplos y límites.

## 10. Frontends

Estructura de cada panel, módulos, service workers, CSP, canvas, despliegue y convenciones JS/CSS.

## 11. Rutas y contratos

Matriz completa de rutas, alias, envelopes, códigos de error, OpenAPI y política de deprecación.

## 12. Base de datos

Tablas, ownership, columnas, JSONB, índices, migraciones, consultas de auditoría y backup/restore.

## 13. Seguridad

Autenticación, CORS, headers, rate limit, secretos, amenazas, checklist de despliegue y respuesta a incidentes.

## 14. Operación

Docker, Easypanel, health, logs, métricas, escalado, rollback y ventanas de mantenimiento.

## 15. Pruebas y calidad

Pirámide, matriz de tests, golden fixtures, contract tests, arquitectura, cobertura y CI.

## 16. Runbook de incidentes

Evolution caída, base lenta, JID inválido, imagen rechazada, doble envío y fuga de secreto.

## 17. Migración y compatibilidad

Fases, alias, deprecación, cutover, rollback y lecciones del legado.

## 18. ADRs

Índice y resumen de las seis decisiones; enlaces a `docs/adr/*.md`.

## 19. Glosario

Socio, perfil, lote, cartelera, snapshot, JID, instancia, tenant, envelope y puerto.

## 20. Apéndices

Catálogo SQL, `.env.example`, ejemplos curl, códigos, changelog y trazabilidad H-01…H-14.
