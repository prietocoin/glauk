# Glauk — SUPER-README

Documento maestro de arquitectura, operación y contratos del monolito modular **Glauk**.

| Campo | Valor |
|---|---|
| Versión | 0.4.0 (mensajería) |
| Fecha | 2026-10-20 |
| Estado | Fase 4 — `platform/evolution` |
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

### 9.1 Cliente unificado de Evolution (`platform/evolution`)

> Fase 4 implementada. Este paquete es el único adaptador de salida hacia Evolution API. No conoce JIDs, PostgreSQL ni reglas de dominio: recibe un `number` opaco y un payload ya validado por el dominio.

Contrato:

| Operación | Endpoint | Cuerpo |
|---|---|---|
| Media | `POST {EVOLUTION_API_URL}/message/sendMedia/{EVOLUTION_INSTANCE}` | `number`, `mediatype`, `mimetype`, `caption`, `media`, `fileName` opcional. |
| Texto | `POST {EVOLUTION_API_URL}/message/sendText/{EVOLUTION_INSTANCE}` | `number`, `text`. |

- `SendMedia` exige `number` y `media` no vacíos; `mediatype` y `mimetype` son opcionales y usan `image` y `image/jpeg` cuando faltan.
- `SendText` exige `number` y `text` no vacíos.
- No se valida ni normaliza el `number`: eso pertenece al dominio. El cliente no acepta JIDs ni consulta la base.
- La única cabecera de autenticación es `apikey: <EVOLUTION_API_KEY>`. Nunca se envían `Authorization` ni `Global-Api-Key`.
- `Content-Type: application/json`; el cuerpo se serializa una sola vez antes de los reintentos.
- `EVOLUTION_TIMEOUT` se aplica con `http.Client.Timeout` y además mediante el `context.Context` del request.

Recibo:

- `Receipt.Status` es `SENT` por defecto. Si Evolution devuelve `status`, se conserva.
- `Receipt.MessageID` se extrae de `key.id`; también se aceptan `messageId` o `id` de nivel superior para variantes de versión.
- `Receipt.Raw` conserva una copia del cuerpo recibido, con la clave redactada si apareciera.
- Un `200` con cuerpo vacío, `null` o no JSON no es error: devuelve `Status: "SENT"` y `MessageID` vacío.

### 9.2 Reintentos y errores

| Condición | ¿Reintenta? | Espera |
|---|---|---|
| Error de red o timeout | Sí | Exponencial con jitter ±20 %. |
| `408`, `425`, `429` | Sí | Exponencial; `Retry-After` tiene prioridad. |
| `500`, `502`, `503`, `504` y demás `5xx` | Sí | Exponencial con jitter ±20 %. |
| `400`, `401`, `403`, `404`, `422` | No | Error inmediato. |
| Contexto cancelado | No | Se devuelve `ctx.Err()` sin esperar el backoff. |

- `EVOLUTION_MAX_RETRIES` es el número de reintentos después del primer intento; por defecto `2`.
- `EVOLUTION_RETRY_BASE` es la base; por defecto `300ms`. Si el cliente recibe una base no positiva, usa `250ms`.
- La espera exponencial es `base × 2^(attempt-1)`, con el exponente acotado para no desbordar.
- El jitter multiplica el cálculo exponencial por un valor entre `0.8` y `1.2`.
- `Retry-After` se interpreta en segundos y tiene prioridad sobre el cálculo exponencial.
- Toda espera, incluido `Retry-After`, se corta en `5s` y es cancelable por el contexto.
- Los cuerpos de respuesta se leen con un límite de `1 MiB`.
- `APIError` expone `StatusCode`, `Body`, `Retryable` y `RetryAfter` (segundos). `Body` se recorta a 500 caracteres e incluye `...` cuando se truncó.
- La `apikey` se elimina del cuerpo antes de construir `APIError`; nunca aparece en `err.Error()` ni en logs. El paquete no registra credenciales.

### 9.3 Simulación

La simulación se activa **únicamente** cuando `EVOLUTION_API_URL` queda vacía (`Client.Simulated() == true`). En ese modo no se abre ninguna conexión y la API key no se utiliza.

- `Status` devuelto: `SIMULATED`.
- `MessageID`: `sim-` + los primeros 8 bytes de `SHA-256(kind|number|content)` en hexadecimal.
- El identificador es determinista para el mismo `kind`, `number` y contenido, lo que permite probar idempotencia y reintentos sin red.
- En producción el arranque rechaza `EVOLUTION_API_URL` vacía; la simulación nunca puede activarse por accidente en ese entorno.

### 9.4 Verificación

```powershell
$env:CGO_ENABLED = "1"   # -race requiere cgo y gcc en PATH
go test ./internal/platform/evolution -race -v
```

Casos cubiertos:

- `200` con `key.id` y `200` con cuerpo vacío o no estructurado.
- `400/401/403/404/422` no reintentables y con cuerpo acotado.
- `429` con `Retry-After`, reintento y espera respetada.
- `408/425/500/502/503/504` reintentados hasta `MaxRetries`.
- Timeout de contexto, cancelación durante el backoff y timeout de red.
- Simulación sin red, determinista y con validación de campos mínimos.
- La API key no aparece en el error aunque el servidor la refleje en el cuerpo.

Los detalles de generación de carteleras, JPG, idempotencia de reportes y endpoints de dominio se completan en las fases de Reportes.

## 10. Frontends y routing estático

> Fase 5 implementada: `internal/platform/httpx` resuelve host/prefijo, sirve assets con caché diferenciada y aplica el SPA fallback. Los shells reales de cada panel se incorporan en las fases de frontend; el contrato de aislamiento ya está activo.

### 10.1 Tabla de tenants

| Tenant | Hosts por defecto | Prefijo path | Shell | API que consume |
|---|---|---|---|---|
| `tasas` | `tasas.localhost` | `/tasas` | `public/tasas/index.html` | `/api/v1/tasas/*` |
| `perfiles` | `perfiles.localhost` | `/perfiles` | `public/perfiles/index.html` | `/api/v1/perfiles/*` |
| `comprobantes` | `comprobantes.localhost` | `/comprobantes` | `public/comprobantes/index.html` | `/api/v1/comprobantes/*`, `/api/v1/reportes/*` |
| `reportes` | `reportes.localhost` | `/reportes` | `public/reportes/index.html` | `/api/v1/reportes/*` |

Los hosts reales se configuran con `TENANT_*_HOSTS` (CSV, sin puerto). Se admite un wildcard de un solo nivel: `*.glauk.com` coincide con `tasas.glauk.com` pero no con `a.b.glauk.com`.

### 10.2 Resolución

1. Si `TRUSTED_PROXY_HEADER` está configurado, se usa el primer valor de esa cabecera (`X-Forwarded-Host`) en lugar de `Host`.
2. Se quita el puerto, se normaliza a minúsculas y se compara con `TENANT_*_HOSTS`.
3. Un match de host gana y no se evalúa el path.
4. `HOST_ROUTING_MODE=host`: un host desconocido responde `404`, incluso en `/`; nunca sirve el panel de otro tenant.
5. Sin match de host, se prueba el primer segmento del path contra `/tasas`, `/perfiles`, `/comprobantes` y `/reportes`.
6. Si no hay match y la ruta es `/` o el fallback SPA, se usa `DEFAULT_TENANT`.
7. `DEFAULT_TENANT=none` sirve `public/index.html` solo en `/`; el resto responde `404`.
8. `HOST_ROUTING_MODE=path` ignora los hosts; `host` ignora los prefijos; `both` acepta ambos y el host tiene prioridad.

### 10.3 Precedencia de rutas

```text
1. /health, /ready
2. API canónica /api/v1/*
3. Alias legacy específicos y generales
4. /api y /api/* desconocidos -> 404 JSON (nunca SPA)
5. /robots.txt y /favicon.ico
6. Assets compartidos /shared/*
7. Assets del tenant (con o sin hash)
8. SPA fallback del tenant para rutas sin extensión
9. 404 global
```

`RegisterStatic` se invoca después de registrar toda la API. Instala el guard de `/api` antes del handler estático, por lo que una ruta de API desconocida no puede caer al `index.html`.

### 10.4 Reglas de servido y caché

| Ruta | Respuesta | `Cache-Control` |
|---|---|---|
| `GET /` con host tenant | `index.html` del tenant | `no-cache` |
| `GET /` sin match, `DEFAULT_TENANT=comprobantes` | Shell del tenant por defecto | `no-cache` |
| `GET /` sin match, `DEFAULT_TENANT=none` | `public/index.html` (lanzador) | `no-cache` |
| `GET /tasas/` | `public/tasas/index.html` | `no-cache` |
| `GET /tasas/main.a1b2c3d4.js` | Asset con hash | `public, max-age=31536000, immutable` |
| `GET /tasas/main.js` | Asset sin hash | `no-cache` |
| `GET /shared/js/http.js` | Asset compartido | `public, max-age=3600` |
| `GET /manifest.json`, `/sw.js` | Del tenant | `no-cache` |
| `GET /api/inexistente` | Envelope `NOT_FOUND` | `no-store` |
| `GET /assets/..%2f..%2fetc/passwd` | Envelope `FORBIDDEN` | `no-store` |
| `GET /ruta-sin-extension` | `index.html` del tenant | `no-cache` |
| `GET /ruta.js` inexistente | Envelope `NOT_FOUND` | `no-store` |

### 10.5 Aislamiento y seguridad

- **Anti-traversal:** se inspeccionan `OriginalURL` y `Path`, se decodifica `%2e%2e`, y se rechazan segmentos `..`, barras invertidas y bytes NUL con `403 FORBIDDEN`.
- Después de `filepath.Clean`, `filepath.Rel` verifica que el archivo resuelto permanezca dentro de `PUBLIC_DIR/<tenant>` o `PUBLIC_DIR/shared`.
- Un asset inexistente **con extensión** responde `404`; nunca se sustituye por el SPA. Una ruta sin extensión sí usa el `index.html` del tenant.
- `index.html` se sirve con CSP de paneles (`default-src 'self'`, `img-src 'self' data: blob:`, `style-src 'self' 'unsafe-inline'`). La API no recibe CSP porque no devuelve HTML.
- Todos los paneles usan rutas relativas para que el mismo shell funcione en host y en path.
- El service worker se registra con el scope del tenant y **no** intercepta `/api/`; las mutaciones y los reportes nunca quedan atrapados offline.
- `/shared/*` es común a todos los paneles; los assets de cada tenant viven bajo `public/<tenant>/`.

### 10.6 Verificación

```powershell
$env:CGO_ENABLED = "1"
go test ./internal/platform/httpx -race -v
```

Cubre host routing, path routing, prioridad del host, `/api/*` en JSON, traversal con `403`, asset inexistente con `404`, SPA fallback y las cuatro políticas de caché.

## 11. Rutas y contratos HTTP

> Fase 5 implementada: envelopes, traducción de errores, request ID, cabeceras de seguridad y guards de API/admin. Las rutas de dominio se registran en las fases 6–12 manteniendo el orden normativo.

### 11.1 Envelope canónico

Éxito:

```json
{
  "success": true,
  "data": {},
  "requestId": "9f2c1e7a4b8d4c1e"
}
```

Error:

```json
{
  "success": false,
  "error": "mensaje legible para el operador",
  "code": "CODIGO_ESTABLE",
  "requestId": "9f2c1e7a4b8d4c1e"
}
```

`Envelope` expone `success`, `data`, `error`, `code`, `message` y `requestId`. Los helpers son `OK`, `Created`, `Fail` y `FailCode`; `OKFlat` conserva los contratos legacy que leen campos en la raíz.

### 11.2 Compatibilidad por dominio

| Dominio | Éxito | Error |
|---|---|---|
| Tasas | `{success:true, ...campos}` plano (`OKFlat`) | `{success:false, error, code, requestId}` |
| Perfiles | Array raíz o `{success:true, data}` | `{success:false, error, code, requestId}` |
| Comprobantes | Array raíz o `APIResponse{success,message,data}` | `APIResponse{success:false,error}` + `code` |
| Reportes canónico | `Envelope` | `Envelope` |
| Reportes legacy | `{success,message,jid,remoteJid,data}` | `{success:false,error,code}` |
| Admin | `{success,data}` | `{success:false,error,code,requestId}` |

Los errores siempre usan el envelope canónico, aunque el endpoint de éxito conserve su forma histórica.

### 11.3 Catálogo de errores

| Código | HTTP | Significado |
|---|---:|---|
| `INVALID_JSON` | 400 | Cuerpo no parseable. |
| `VALIDATION_ERROR` | 400 | Campo inválido. |
| `DESTINO_REQUERIDO` | 400 | Falta `socio` y `remoteJid`. |
| `IMAGEN_REQUERIDA` | 400 | Falta `base64`. |
| `TEXTO_REQUERIDO` | 400 | Falta `texto`. |
| `UNAUTHORIZED` | 401 | Clave inválida o ausente. |
| `FORBIDDEN` | 403 | Ruta fuera de `PUBLIC_DIR` o tenant no permitido. |
| `SOCIO_NO_ENCONTRADO` | 404 | Socio inexistente. |
| `NOT_FOUND` | 404 | Recurso o ruta inexistente. |
| `METHOD_NOT_ALLOWED` | 405 | Método no soportado. |
| `CONFLICT` | 409 | Conflicto de estado. |
| `IMAGEN_MUY_GRANDE` | 413 | Imagen mayor a `REPORT_IMAGE_MAX_BYTES`. |
| `PAYLOAD_TOO_LARGE` | 413 | Cuerpo mayor a `BODY_LIMIT_BYTES`. |
| `IMAGEN_FORMATO` | 415 | No es un JPEG válido. |
| `IMAGEN_DIMENSIONES` | 422 | Dimensiones o píxeles fuera de rango. |
| `JID_NO_DISPONIBLE` | 422 | Perfil sin JID. |
| `JID_INVALIDO` | 422 | JID con formato inválido. |
| `RATE_LIMITED` | 429 | Límite de envíos. |
| `INTERNAL` | 500 | Error no controlado. El detalle se registra, nunca se devuelve. |
| `EVOLUTION_UNAVAILABLE` | 502 | Evolution no responde o rechaza. |
| `ADMIN_DISABLED` | 503 | `ADMIN_API_KEY` no configurada. |
| `TIMEOUT` | 504 | Contexto o dependencia excedida. |

`ErrorHandler` traduce `*HTTPError`, `*fiber.Error` y `context.DeadlineExceeded`. Cualquier error desconocido produce `500 INTERNAL` con el mensaje fijo `error interno del servidor`; la causa se registra con `request_id`, método y ruta.

### 11.4 Request ID, seguridad y guards

- `RequestID` acepta un UUID o ULID válido en `X-Request-Id`; si falta o es inválido genera 16 bytes criptográficos en hex. El valor se guarda en `Locals` y se devuelve en la respuesta.
- `AccessLog` registra `request_id`, método, ruta, estado, duración, bytes y tenant. Nunca registra cuerpo, query con tokens, `X-API-Key`, `X-Admin-Key` ni credenciales.
- `SecurityHeaders` aplica `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Permissions-Policy` y `Cross-Origin-Opener-Policy`; añade HSTS solo sobre HTTPS.
- `RequireAPIKey` deja pasar si `SECURITY_API_KEY` está vacía (compatibilidad) y exige `X-API-Key` en tiempo constante si está definida.
- `RequireAdminKey` responde `503 ADMIN_DISABLED` si `ADMIN_API_KEY` está vacía, `401 UNAUTHORIZED` si la clave es incorrecta y delega si es correcta. Acepta `X-Admin-Key` o la cookie `admin_key`.
- `VerifyAdmin` usa exclusivamente `subtle.ConstantTimeCompare`; nunca registra la clave recibida.

### 11.5 Orden de registro

```text
/health, /ready
API canónica /api/v1/*
alias legacy específicos
alias legacy generales
/api/v1/atenea/*
/api y /api/* desconocidos -> 404 JSON
assets estáticos (shared y tenant)
SPA fallback del tenant
404 global
```

El orden es parte del contrato. Un `404` de API nunca se delega al SPA, y el host tiene prioridad sobre el prefijo de path.

### 11.6 Health y readiness

```json
{
  "status": "ok",
  "service": "glauk",
  "version": "1.0.0",
  "uptime_s": 1234,
  "checks": {
    "database": {"status": "ok", "latency_ms": 2},
    "evolution": {"status": "simulated", "instance": "Glauk"}
  }
}
```

- `/health` devuelve `200` si la base responde y `503` si no. Incluye versión e instancia de Evolution.
- `/ready` es equivalente pero omite `version` y la instancia; mantiene `database.status` y `evolution.status`.
- Ambos son públicos, usan `Cache-Control: no-store` y nunca exponen DSN, usuario, contraseña, claves, rutas del sistema ni detalles internos del error.

### 11.7 Verificación

```powershell
$env:CGO_ENABLED = "1"
go test ./internal/platform/httpx -race -v
```

Casos cubiertos: mapeo de `HTTPError`, `fiber.Error` y timeout; ocultamiento de la causa en `500`; validación y generación de request IDs; admin deshabilitado, inválido y correcto; envelopes; health/ready; routing por host y path; `/api/*` JSON; traversal `403`; asset inexistente `404`; SPA fallback y caché.

## 12. Base de datos

> Fase 3 implementada: pool único, ping de arranque, `statement_timeout` y transacciones seguras. El catálogo completo de tablas, ownership, JSONB, índices, migraciones y backup/restore se completa en las fases de dominio.

### 12.1 Pool único y ownership

- El proceso crea **un solo** `*pgxpool.Pool` en `cmd/server` mediante `postgres.New(ctx, cfg.DB)` y lo inyecta por constructor.
- Ningún paquete de dominio construye pools, usa `pgx.Conn` suelto ni `database/sql`. Los repositorios reciben el pool; los servicios no lo conocen.
- No hay `var db *pgxpool.Pool`, `init()` con efectos ni estado global mutable.
- El pool se cierra una sola vez durante el apagado ordenado, después de dejar de aceptar peticiones.

`postgres.New` aplica esta política:

| Parámetro | Origen / valor | Efecto |
|---|---|---|
| `MaxConns` | `DB_MAX_CONNS` | Tope de conexiones del proceso. |
| `MinConns` | `DB_MIN_CONNS` | Mínimo que el pool intenta mantener; 0 es válido. |
| `MaxConnLifetime` | `DB_CONN_MAX_LIFETIME` | Reciclaje de conexiones. |
| `MaxConnIdleTime` | `DB_CONN_MAX_IDLE_TIME` | Cierre de conexiones ociosas. |
| `HealthCheckPeriod` | fijo `30s` | Frecuencia de revisión del pool. |
| `ConnectTimeout` | `DB_CONNECT_TIMEOUT` | Cota del ping inicial y de cada intento de conexión. |
| `application_name` | fijo `glauk` | Identificación en `pg_stat_activity`. |
| `statement_timeout` | `DB_STATEMENT_TIMEOUT` en ms | Límite aplicado por el servidor PostgreSQL. |

**Fail-fast:** `New` valida los parámetros, parsea el DSN con `pgxpool.ParseConfig`, crea el pool y ejecuta un `Ping` obligatorio acotado por `DB_CONNECT_TIMEOUT`. Si el parseo, la configuración o el ping fallan, cierra el pool y devuelve error antes de aceptar tráfico. Un error de credenciales no puede convertirse en un `500` en la primera petición de un usuario.

Los errores de parseo no reproducen el DSN ni la contraseña. La precedencia `DATABASE_URL` → `DB_*` se define en el Capítulo 5.

### 12.2 Timeouts de sentencia

`statement_timeout` se envía como `RuntimeParams["statement_timeout"]` en milisegundos; por ejemplo, `DB_STATEMENT_TIMEOUT=10s` se traduce a `"10000"`. Se aplica en el servidor y no depende del cliente.

Además, cada consulta recibe el `context.Context` heredado del request (`c.UserContext()`), que ya tiene deadline. El timeout del servidor protege contra consultas olvidadas; el contexto del request permite cancelar antes y liberar la conexión. No se permite `0`: el mínimo validado es `1ms`.

### 12.3 Transaccionalidad

Toda escritura de más de una sentencia usa `postgres.WithTx`. Las lecturas no abren transacción explícita.

```go
err := postgres.WithTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
    // todas las sentencias de la unidad de trabajo
    return nil
})
```

Garantías:

1. `BeginTx` recibe las `pgx.TxOptions` del llamador.
2. Si `fn` devuelve error, se ejecuta `Rollback` y se propaga el error original.
3. Si `fn` entra en pánico, se ejecuta `Rollback`, se libera la transacción y el pánico se vuelve a lanzar.
4. El rollback usa `context.WithoutCancel` con una cota propia de 5 s: un request cancelado no deja la transacción abierta.
5. Tras un `Commit` exitoso, el `Rollback` diferido devuelve `pgx.ErrTxClosed`; se ignora a propósito.
6. Si `Commit` falla, se intenta rollback y se devuelve el error de commit.
7. Ningún repositorio hace commit, rollback ni deja transacciones abiertas por su cuenta.
8. No se usan transacciones anidadas ni `context.Background()` dentro de repositorios.

### 12.4 Health

`postgres.Ping(ctx, pool, timeout)` deriva un contexto con timeout y ejecuta `pool.Ping`. `httpx.Health` lo consume con un timeout corto (`healthTimeout`, 2 s) y devuelve `200` si la base responde o `503` si no. La respuesta solo expone `database.status` y `latency_ms`; nunca incluye DSN, usuario, contraseña, host interno ni rutas del sistema. `/ready` mantiene el mismo criterio sin revelar la versión ni la instancia de Evolution.

### 12.5 Verificación

```powershell
$env:CGO_ENABLED = "1"   # -race requiere cgo y gcc en PATH
go test ./internal/platform/postgres -race -v
```

Casos cubiertos:

- Tuning del pool y `RuntimeParams` (`application_name`, `statement_timeout`).
- Rechazo de timeouts no positivos, `MaxConns` inválido y `MinConns > MaxConns`.
- DSN inválido sin filtrar credenciales.
- Fallo rápido con base inalcanzable, dentro de `DB_CONNECT_TIMEOUT`.
- `WithTx`: commit, rollback por error, rollback por pánico, fallo de `BeginTx` y pool nil.
- `Ping`: deadline acotado y rechazo de timeout no positivo.

### 12.6 Catálogo pendiente

Las tablas, columnas, JSONB `monedas`/`mostrar`, ownership por dominio, índices, migraciones idempotentes, consultas de auditoría y backup/restore se documentan al implementar cada dominio. Regla vigente: `reportes` no ejecuta SQL; solo consume puertos de los demás dominios.

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
