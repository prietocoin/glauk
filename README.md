# Glauk — SUPER-README

Documento maestro de arquitectura, operación y contratos del monolito modular **Glauk**.

| Campo | Valor |
|---|---|
| Versión | 1.0.0 (producción) |
| Fecha | 2026-10-21 |
| Estado | Fase 16 — release `v1.0.0` |
| Módulo | `glauk` |
| API | `v1` |
| Tag | `v1.0.0` / `f16-cleanup` |
| Fuente normativa | `../PLAN_CONSOLIDACION_GLAUK.md` |

> Documento maestro de la versión 1.0.0. Describe el estado implementado, no un diseño futuro. Los capítulos 18–20 cierran la documentación operativa; los ADRs y el glosario se conservan en los apéndices.

## 1. Portada y estado del proyecto

Glauk es un monolito modular en Go que consolida **Tasas**, **Perfiles**, **Comprobantes** y **Reportes** en un solo binario, un solo contenedor y una sola base PostgreSQL. La versión 1.0.0 es la release de cutover: mantiene 64 alias de compatibilidad, aplica una migración aditiva para el borrado lógico de impactos y sustituye los servicios legados por el módulo unificado.

- **Alcance:** API canónica `/api/v1/*`, paneles Tasas/Perfiles/Comprobantes, generación de reportes en canvas, envío por Evolution, migraciones y runbook de cutover.
- **Runtime:** imagen distroless `static-debian12:nonroot`, binario estático, sin shell, sin Redis, sin BullMQ y sin Chromium.
- **Calidad:** presupuesto de líneas, arquitectura hexagonal de Reportes, pruebas de contrato, alias con fecha de retiro y documentación sincronizada.
- **Compatibilidad:** los alias responden con `Deprecation: true`, `Sunset: Sat, 31 Dec 2027 23:59:59 GMT` y un enlace al sucesor; `/api/tasas/n8n-webhook` responde `410 ROUTE_REMOVED`.
- **Documentación relacionada:** [CHANGELOG.md](CHANGELOG.md), [migrations/README.md](migrations/README.md), [docs/openapi.yaml](docs/openapi.yaml) y el plan normativo `../PLAN_CONSOLIDACION_GLAUK.md`.
- **Estado de los módulos legados:** `Tasas/`, `Perfiles/` y `Comprobantes/` quedan deprecados; el mantenimiento futuro se realiza exclusivamente en este repositorio. Ver los `DEPRECATED.md` en cada carpeta de origen.

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
| `PUBLIC_DIR` | path | `./public` | N | httpx | Raíz de los frontends, de los vendors locales y del redirect de `/`. En el contenedor es `/app/public`. |
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
| `DEFAULT_TENANT` | enum | `comprobantes` | N | httpx | `tasas`, `perfiles`, `comprobantes`, `reportes` o `none`. Con `comprobantes`, `/` abre la UI unificada Glauk2. Con `none`, `/` sirve un redirect a `/comprobantes/`. |
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

> Fase 6 implementada. El dominio Tasas se migró desde `Tasas/internal/modules/tasas` sin cambiar su comportamiento observable: los fixtures dorados del calculador siguen pasando, las trece rutas conservan sus contratos y la publicación aplica H-02 antes de enviar una sola cartelera.

### 6.1 Estructura

| Ruta | Responsabilidad |
|---|---|
| `internal/tasas/domain/` | `Perfil`, `Moneda`, `Lote`, `TarjetaPais` y `Cartelera`. No conoce SQL, HTTP ni render. |
| `internal/tasas/calculator/` | Cálculo puro de carteleras, triangulación, precisión y tendencia. Recibe el instante por parámetro. |
| `internal/tasas/store/` | Única puerta a `perfiles_glaukov`, `tasas_glaukov` y `notificaciones_tasas`. Recibe un `Querier`; no abre pools. |
| `internal/tasas/service/` | Casos de uso: publicar, publicar por ID, reenviar, cartelera, envío, concurrencia y administración. |
| `internal/tasas/http/` | Traduce Fiber a servicio y devuelve los contratos JSON/PNG. No escribe SQL ni abre clientes. |
| `internal/tasas/hoo/` | Contrato y cliente del borrador externo HOO. |
| `internal/tasas/render/` | PNG en memoria con `gg`, assets embebidos de banderas, fondo, logo y fuente. |
| `internal/tasas/calculator/testdata/golden_calculator.json` | Fixture congelado del servicio Node, ejecutado por `TestGolden`. |

Las dependencias externas entran por `platform/postgres` y `platform/evolution`. Tasas no crea un `pgxpool` ni un cliente HTTP de Evolution propios: el pool y el `Sender` se inyectan desde `cmd/server`.

### 6.2 Actividad del socio (H-02)

`mostrar->>'tasas'` es la fuente de verdad. La columna `activo` es un interruptor opcional que **solo puede apagar**; nunca enciende a un socio que Perfiles apagó.

```text
publicar(socio) =
    mostrar.tasas == true
    AND (activo inexistente OR activo IS NULL OR activo == true)
```

| `mostrar.tasas` | `activo` | ¿Se publica? |
|---|---|---|
| ausente | cualquiera | No |
| `false` | `true` | No |
| `true` | ausente / `NULL` | Sí |
| `true` | `true` | Sí |
| `true` | `false` | No |

La decisión vive en `service.activosDe`/`activoParaTasas` y tiene una tabla de verdad en `service_test.go`. El store solo transporta `mostrar` y `activo`; no decide por su cuenta.

### 6.3 Cálculo y triangulación

- La moneda base del socio se resuelve con `BaseRate`. Las divisas de referencia (`USD`, `USDT`, etc.) valen `1`; una cotización ausente, cero o no numérica también cae a `1` para evitar divisiones imposibles.
- Cada divisa configurada genera una tarjeta si no está apagada, si el lote la cotiza o es de referencia, y si la tasa normalizada no es cero.
- La triangulación es `CrossRate = tasa / base`: expresa cada divisa en unidades de la moneda nativa del socio.
- Compra y venta se calculan con los márgenes del perfil, en valor absoluto:
  - `compra = CrossRate × (1 + deposito/100)`
  - `venta = CrossRate / (1 + pago/100)`
- La regla de precisión redondea a ocho decimales y luego trunca por magnitud: entero por encima de `99.99`, dos decimales desde `10` y resolución fina por debajo. El comportamiento del original, incluido su defecto conocido en valores ínfimos, se conserva para no alterar ninguna cartelera.
- La tendencia compara la tasa cruzada actual con la del lote anterior: `up`, `down` o `equal`. Con un solo lote, el anterior es el mismo.
- La fecha y hora se publican con el locale histórico de Caracas (`17-oct`, `12:00`), usando un desplazamiento fijo UTC-4 y sin depender de tzdata del sistema.
- Las tarjetas se ordenan por código para que el resultado no dependa del orden del mapa.

### 6.4 Rutas canónicas (13)

| # | Método | Ruta | Entrada | Salida 200 | Errores |
|---:|---|---|---|---|---|
| 1 | `GET` | `/api/v1/tasas/monitor` | — | `{success,id_tasa,tasas}` | 500 |
| 2 | `GET` | `/api/v1/tasas/socios` | — | `{success,socios[]}` | 500 |
| 3 | `GET` | `/api/v1/tasas/hoo-borrador` | — | `{success,borrador}` | 500 |
| 4 | `PUT` | `/api/v1/tasas/socios/:socio` | `SocioUpdate` | `{success,socio}` | 400/404/500 |
| 5 | `POST` | `/api/v1/tasas/guardar-lote` | `{tasas}` | `{success,id_tasa,registro}` | 400/500 |
| 6 | `POST` | `/api/v1/tasas/publicar-lote` | `{modo_prueba_global?}` | `{success,id_tasa,procesados,detalle[]}` | 400/500 |
| 7 | `POST` | `/api/v1/tasas/notificar-remesas` | vacío | `{success,notificacion:{id_tasa}}` | 500 |
| 8 | `GET` | `/api/v1/tasas/calcular/:socio` | — | `{success,datos}` | 404/500 |
| 9 | `GET` | `/api/v1/tasas/render/:socio` | — | `image/png` | 404/500 texto |
| 10 | `POST` | `/api/v1/tasas/render` | `{socio}` | `image/png` | 404/500 texto |
| 11 | `POST` | `/api/v1/tasas/disparar` | `{socio,modo_prueba?}` | `{success,socio,evolutionResponse}` | 400/404/500 |
| 12 | `POST` | `/api/v1/tasas/disparar-lote` | `{modo_prueba_global?}` | Igual que #6 | 400/500 |
| 13 | `DELETE` | `/api/v1/tasas/lote/:id_tasa` | — | `{success,message}` | 400/404/500 |

`GET /render/:socio` y `POST /render` son dos formas de entrar al mismo handler. `POST /disparar-lote` comparte exactamente la lógica de `POST /publicar-lote`, como en el servicio Node.

### 6.5 Contratos JSON

**#1 Monitor**

```json
{"success":true,"id_tasa":"T052","tasas":{"USD":1,"USDT":1,"ARS":1250.5,"EUR":0.92}}
```

Sin lotes: `{"success":true,"id_tasa":"T000","tasas":{}}`. Nunca responde 404.

**#2 Socios**

```json
{
  "success": true,
  "socios": [
    {
      "nombre": "FUNDABLOCK",
      "id_grupo": "120363012345678901@g.us",
      "moneda_base": "USDT",
      "monedas": {
        "ARS": {"activo": true, "porcentaje": {"deposito": 2.5, "pago": 1.5}}
      },
      "activo": true,
      "modo_prueba": false,
      "rol": "SOCIO"
    }
  ]
}
```

`monedas` nunca es `null`; si el perfil no tiene, se devuelve `{}`. Un rol vacío se normaliza a `SOCIO`.

**#3 HOO**

```json
{"success":true,"borrador":{"ID":"...","USD":1,"ARS":1250.5,"CREATED_AT":"2026-10-20T12:00:00.000Z"}}
```

El objeto completo tiene los veinte campos de `hoo/mapeo.go`. Si el origen no está configurado: `500 {"success":false,"error":"origen HOO no configurado"}`.

**#4 Guardar socio**

```json
{
  "monedas": {
    "ARS": {"activo": true, "porcentaje": {"deposito": 2.5, "pago": 1.5}}
  },
  "activo": true,
  "modo_prueba": false,
  "rol": "SOCIO"
}
```

Todos los campos son opcionales; un campo ausente **no se escribe**. `monedas` se escribe con el codec superset. Respuesta:

```json
{"success":true,"socio":{"nombre":"FUNDABLOCK","monedas":{},"activo":true,"modo_prueba":false,"rol":"SOCIO"}}
```

404 si el socio no existe. El legado respondía éxito con un objeto vacío.

**#5 Guardar lote**

```json
{"tasas":{"USD":1,"USDT":1,"ARS":1250.5,"CREATED_AT":"2026-10-20T12:00:00.000Z"}}
```

Respuesta:

```json
{"success":true,"id_tasa":"T053","registro":{"id_tasa":"T053","tasas":{"USD":1,"USDT":1,"ARS":1250.5}}}
```

`tasas` es obligatorio y no vacío. Si falta `CREATED_AT`, se sella en UTC con formato `2006-01-02T15:04:05.000Z`; el ID se deriva del último lote (`T###`).

**#6 / #12 Publicar**

```json
{"modo_prueba_global":false}
```

Respuesta:

```json
{
  "success": true,
  "id_tasa": "T053",
  "procesados": 3,
  "detalle": [
    {"socio":"FUNDABLOCK","status":"ok"},
    {"socio":"GENERAL","status":"error","error":"evolution api: estado 401"}
  ]
}
```

`procesados` cuenta solo socios activos según H-02. Los errores parciales no convierten la respuesta en 500: el lote se publicó para el resto y `detalle` es la fuente de verdad. Sin lotes: `400 {"success":false,"error":"No hay lotes guardados en tasas_glaukov para publicar."}`.

**#7 Notificar remesas**

Sin cuerpo. Inserta la notificación del lote vigente y devuelve:

```json
{"success":true,"notificacion":{"id_tasa":"T053"}}
```

**#8 Calcular**

```json
{
  "success": true,
  "datos": {
    "nombre_socio": "FUNDABLOCK",
    "moneda_base": "USDT",
    "lote_tasa": "T053",
    "fecha_actualizacion": "20-oct",
    "hora_actualizacion": "12:00",
    "tarjetas_paises": [
      {"code":"ARS","pais":"Argentina","nombre_pais":"Argentina","bandera":"ar","compra":1250.5,"venta":1262.5,"trend":"up"}
    ]
  }
}
```

Un socio inexistente responde `404 {"success":false,"error":"Socio no encontrado"}`.

**#9 / #10 Render**

`GET /render/:socio` y `POST /render` con `{"socio":"FUNDABLOCK"}` devuelven `image/png` dibujado en memoria. No hay archivo temporal ni servicio externo. Errores en texto plano:

- `404 Socio no encontrado`
- `500 Error generando PNG`

**#11 Disparar**

```json
{"socio":"FUNDABLOCK","modo_prueba":true}
```

`modo_prueba` ausente toma el valor guardado del socio. Respuesta:

```json
{"success":true,"socio":"FUNDABLOCK","evolutionResponse":{"status":"SENT","messageId":"3EB0..."}}
```

**#13 Eliminar lote**

```json
{"success":true,"message":"Lote T053 eliminado correctamente."}
```

404 si no existe; 400 si el ID está vacío.

### 6.6 Publicación, modo prueba y Evolution

- `Publicar` lee el lote vigente y el anterior con una sola consulta (`ORDER BY id DESC LIMIT 2`), carga los socios y aplica H-02 antes de abrir workers.
- El pool de publicación usa cinco workers y un plazo independiente de 30 segundos por socio. Los resultados se recolectan por índice y se ordenan por nombre.
- En modo prueba, el destino es siempre `JID_GRUPO_PRUEBA`; sin ese valor el envío falla en vez de inventar un grupo.
- En modo oficial se usa `id_grupo` del socio. Si el socio no tiene grupo, se cae al JID de pruebas para no publicar en un destino vacío.
- El PNG se envía como `image/png`, base64 puro (sin prefijo `data:`) y `fileName: cartelera.png`.
- El pie conserva el prefijo histórico: `🚀 [OFICIAL]` o `🧪 [PRUEBA]`, seguido de `Cartelera *socio* — Lote: T###`.
- El envío usa `platform/evolution.Client.SendMedia`; sus reintentos, timeout y simulación son responsabilidad de ese adaptador.

### 6.7 Render PNG

`render.GenerarCartelera` compone la imagen completa en memoria con `fogleman/gg`. Las banderas, el fondo, el logo y la fuente `Inter-Bold.ttf` están embebidos con `go:embed`; no se leen archivos del filesystem en runtime. La misma cartelera calculada alimenta `/calcular`, `/render` y la publicación, de modo que no existan dos definiciones de lo que se muestra.

### 6.8 Errores

| Condición | Respuesta |
|---|---|
| Cuerpo JSON inválido o `tasas` vacío | `400 {success:false,error}` |
| `id_tasa` vacío al borrar/publicar | `400 {success:false,error}` |
| Sin lotes para publicar | `400` con el texto histórico |
| Socio inexistente | `404 {success:false,error}` en JSON; texto plano en render |
| Lote inexistente | `404 {success:false,error}` |
| Fallo de PostgreSQL, HOO o Evolution no parcial | `500 {success:false,error}` |
| Fallo de dibujo | `500 Error generando PNG` |
| Error de un socio durante publicación | `200` con `detalle[].status = "error"` |

### 6.9 Verificación

```powershell
$env:CGO_ENABLED = "1"   # -race requiere cgo y gcc en PATH (WinLibs en Windows)
go test ./internal/tasas/... -race -v
go test ./internal/tasas/calculator -run TestGolden -v
```

`routes_test.go` comprueba las trece rutas con `app.GetRoutes()` y ejecuta un smoke de cada endpoint. `service_test.go` fija la tabla de verdad H-02, el filtro de publicación, los errores parciales y la elección de destino. `calculator_test.go` ejecuta el fixture dorado completo contra `golden_calculator.json`.

## 7. Dominio Perfiles

> Fase 7 implementada. `internal/perfiles` es el **único escritor** de `perfiles_glaukov` (D-12). Tasas, Comprobantes y Reportes pueden leer la tabla a través de sus propios puertos, pero no ejecutan `INSERT`, `UPDATE` ni `DELETE` sobre ella.

### 7.1 Mapa de archivos

| Capa | Archivos | Responsabilidad |
|---|---|---|
| Modelo | `model/perfil.go`, `model/mostrar.go`, `model/monedas.go`, `model/conflictos.go`, `model/requests.go`, `model/aliases.go` | Contrato JSON, codec tolerante, defaults y alias. |
| Servicio | `service/service.go`, `service/params.go`, `service/actividad.go`, `service/plantilla.go`, `service/compat.go`, `service/errors.go`, `service/ports.go` | Casos de uso, H-01/H-02/H-03, protección de matrices. |
| Repositorio | `repo/reader.go`, `repo/writer.go`, `repo/sql.go`, `repo/scan.go`, `repo/plantilla.go`, `repo/monedas_codec.go`, `repo/querier.go` | SQL, escaneo y serialización superset. No abre pools. |
| HTTP | `http/routes.go`, `http/deps.go`, `http/handler_read.go`, `http/handler_write.go`, `http/handler_plantilla.go`, `http/response.go` | Ocho rutas canónicas y envelopes de compatibilidad. |
| Migraciones | `migrations/0001_init.sql`, `migrations/0002_superset.sql` | Referencia; no se ejecutan automáticamente. |

`repo.Querier` es una interfaz local compatible con `*pgxpool.Pool` de `platform/postgres`. El servicio depende de `Lector` y `Escritor` (puertos pequeños) para poder probar las reglas sin PostgreSQL.

La fachada histórica de Tasas conserva su endpoint de actualización de socios, pero ya no construye SQL de escritura: `internal/perfiles/repo/tasas_legacy.go` es el adaptador de compatibilidad que ejecuta el `UPDATE` con las columnas opcionales. `tasas/store` solo codifica `monedas` y descubre las columnas existentes. Así se preserva el contrato de Fase 6 sin reintroducir un segundo escritor de `perfiles_glaukov`.

### 7.2 Esquema de `perfiles_glaukov`

| Columna | Tipo | Contrato |
|---|---|---|
| `id_grupo` | `text` | JID del grupo. Un alta sin JID guarda `''` y emite `WARN`; **nunca** se genera `nombre+"_JID"` (H-03). |
| `nombre` | `text` | Identidad canónica. El índice único es `UPPER(BTRIM(nombre))`. |
| `rol` | `text` | Default `SOCIO`. |
| `moneda_base` | `text` | Default `USDT`. |
| `saldo_inicial` | `numeric` | Se lee y serializa como string para conservar el contrato de `node-pg`. |
| `mostrar` | `jsonb` | `{"tasas": boolean, "dashboard": boolean}`. `mostrar.tasas` manda para H-02. |
| `monedas` | `jsonb` | Mapa ISO → superset de §7.4. |
| `herencia` | `boolean` | Indica que el socio usa las reglas de la matriz. |
| `recordar_mostrar` | `jsonb` | Copia temporal de `mostrar` para memorizar/restaurar. La crea `0001_init.sql`. |

`0001_init.sql` añade `recordar_mostrar` y el índice único. `0002_superset.sql` solo audita conflictos y cobertura: la normalización masiva queda para la fase 12 con backup y fixtures. Ninguna migración se ejecuta desde un request.

### 7.3 Actividad H-02

`mostrar.tasas` es la fuente de verdad. La columna legada `activo`, si existiera, **solo puede apagar**; un `true` no enciende a un socio apagado en Perfiles. El campo público `activo` se deriva y es de solo lectura: `GuardarRequest` no lo declara y cualquier valor enviado se ignora.

| `mostrar.tasas` | Columna `activo` | Resultado |
|---|---|---|
| `true` | `nil` (no existe) | `true` |
| `true` | `true` | `true` |
| `true` | `false` | `false` |
| `false` | `nil`, `true` o `false` | `false` |

`service.Actividad` implementa la tabla y `service.ActividadDePerfil` la aplica al leer. El repositorio actual no escribe la columna legada: `Toggle` solo modifica `mostrar.tasas`.

### 7.4 Codec `monedas`

El codec lee las tres formas históricas y siempre escribe el superset completo:

| Origen | Forma de entrada | Claves que aporta |
|---|---|---|
| Tasas | `porcentaje: {deposito, pago}` | `porcentaje` |
| Perfiles | `def`, `deposito`, `dep_op`, `pago`, `pago_op` | operadores y valores planos |
| Comprobantes | `tipo`, `polar_D`, `polar_P`, `polaridad`, `porcentaje` | tipo y polaridades |

Al leer:

1. Si falta `def`/`tipo`, se usa `"D"`.
2. Si falta `activo`, se asume `true`.
3. Si falta un operador, se deriva de su polaridad; si tampoco existe, `dep_op`/`polar_D` usan `"+"` y `pago_op`/`polar_P` usan `"-"`.
4. Si `porcentaje` y `deposito`/`pago` discrepan, **gana `porcentaje`** y se registra un `WARN` con el socio, el código ISO, el campo y ambos valores. El conflicto también queda disponible en `model.Conflictos` para auditoría.

Al escribir, `repo.encodeMonedas` normaliza el código ISO a mayúsculas y emite estas once claves por moneda:

```json
{
  "activo": true,
  "porcentaje": {"deposito": 2.5, "pago": 1.5},
  "def": "D",
  "deposito": 2.5,
  "dep_op": "+",
  "pago": 1.5,
  "pago_op": "-",
  "tipo": "D",
  "polar_D": "+",
  "polar_P": "-",
  "polaridad": "+"
}
```

### 7.5 Endpoints canónicos

`http.Register` monta exactamente estas ocho rutas. Los alias históricos se registran en `cmd/server` reutilizando el mismo `Deps`; no crean un segundo escritor.

| Método | Ruta | Éxito | Error |
|---|---|---|---|
| `GET` | `/api/v1/perfiles` | `200` + array raíz de `Perfil`. | `{success:false,error,code}` |
| `GET` | `/api/v1/perfiles/activos` | `200` + array raíz de nombres. | `{success:false,error,code}` |
| `POST` | `/api/v1/perfiles/guardar` | `200` + `{success:true,data:Perfil}`. | `{success:false,error,code}` |
| `POST` | `/api/v1/perfiles/apagar-todos` | `200` + `{success:true,message}`. | `{success:false,error,code}` |
| `POST` | `/api/v1/perfiles/plantilla/memorizar` | `200` + `{success:true,message}`. | `{success:false,error,code}` |
| `POST` | `/api/v1/perfiles/plantilla/restaurar` | `200` + `{success:true,message}`. | `{success:false,error,code}` |
| `PATCH` | `/api/v1/perfiles/:nombre/estado` | `200` + `{success:true,data:Perfil}`. Body `{"activo":boolean}`. | `{success:false,error,code}` |
| `DELETE` | `/api/v1/perfiles/:nombre` | `200` + `{success:true}`. | `{success:false,error,code}` |

Códigos de error: `JSON_INVALIDO`, `NOMBRE_REQUERIDO`, `ENTIDAD_PROTEGIDA`, `NO_ENCONTRADO` y `ERROR_INTERNO`. Los errores de persistencia no exponen SQL.

`GuardarRequest` acepta alias como `socio`/`nombre_socio`, `jid`/`grupo`, `moneda`/`monedaBase` y `saldo`. El campo canónico siempre gana sobre el alias. `saldo_inicial` acepta número o cadena numérica.

### 7.6 Plantilla y protecciones

- `GENERAL` no puede eliminarse ni apagarse por `PATCH`; la petición se ignora y se fuerza activo.
- `FUNDDA` no puede eliminarse. Es la matriz que consume el modo `herencia`.
- `apagar-todos` mantiene `GENERAL` activo y apaga `mostrar.tasas` del resto.
- `memorizar` copia `mostrar` a `recordar_mostrar`; `restaurar` recupera la copia y fuerza `GENERAL` activo.
- Un `id_grupo` ausente se persiste como `""` con `WARN`; no se inventa ningún JID.

### 7.7 Verificación

```powershell
$env:CGO_ENABLED = "1"   # -race requiere cgo y gcc en PATH
gofmt -l .
go vet ./...
go test ./internal/perfiles/... -race -v -count=1
go test ./...
```

Auditoría D-12 (debe devolver `escrituras_fuera_de_perfiles=0`):

```powershell
$files = Get-ChildItem . -Recurse -File -Filter *.go |
  Where-Object { $_.FullName -notmatch '\\internal\\perfiles\\' }
$files | Select-String -Pattern '(?i)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+perfiles_glaukov\b'
```

Los tests cubren las tres formas de `monedas`, round-trip al superset, conflicto con `WARN`, tabla H-02, H-03 sin `_JID`, protección de matrices, ocho rutas HTTP y el upsert con código normalizado.

## 8. Dominio Comprobantes

> Fase 8 implementada. `internal/comprobantes` es el único dueño de `comprobantes_raw`, `comprobantes_liq`, `impactos_raw` y de la consola `cola_fb`. Tasas y Perfiles se leen, nunca se escriben desde este dominio (D-12).

### 8.1 Mapa de archivos

| Capa | Archivos | Responsabilidad |
|---|---|---|
| Dominio | `domain/comprobante.go`, `domain/liquidacion.go`, `domain/lote_tasa.go`, `domain/filters.go`, `domain/admin.go`, `domain/report.go`, `domain/money.go`, `domain/errors.go` | Entidades, filtros, errores y contratos del puerto de Reportes. |
| DTO | `dto/optional_float.go`, `dto/list_filters.go`, `dto/liquidar.go`, `dto/update.go`, `dto/liquidacion.go`, `dto/api_response.go` | Payloads, alias legacy y distinción ausente/cero. |
| Cálculo | `calc/snapshot.go`, `calc/profile.go`, `calc/amounts.go`, `calc/polarity.go`, `calc/precision.go`, `calc/rates.go`, `calc/projection.go`, `calc/source_row.go` | Snapshot congelado, herencia, spread, truncado y proyección. |
| Persistencia | `store/repository_list.go`, `store/sql_list.go`, `store/row_list.go`, `store/repository_mutate.go`, `store/sql_mutate.go`, `store/repository_mutation_read.go`, `store/repository_fundda.go`, `store/rates.go`, `store/admin.go`, `store/report_reader.go`, `store/sql_report.go` | SQL, transacciones y escaneo. |
| Servicio | `service/list.go`, `service/rates.go`, `service/snapshot.go`, `service/mutate.go`, `service/admin.go`, `service/report_reader.go`, `service/ports.go` | Casos de uso y coordinación transaccional. |
| HTTP | `http/routes.go`, `http/deps.go`, `http/handler_list.go`, `http/handler_tasas.go`, `http/handler_mutate.go`, `http/handler_admin.go`, `http/response.go` | Siete rutas canónicas y tres rutas admin. |
| Migración | `migrations/0001_impactos_soft_delete.sql` | Columna `eliminado_en` e índice parcial de impactos vigentes. |

`store.DB` es una interfaz local compatible con `*pgxpool.Pool` y con `pgxmock`. Las mutaciones usan `platform/postgres.WithTx`; por eso `WithTx` acepta el puerto `postgres.Beginner` y conserva el rollback ante error o pánico.

### 8.2 Modelo de datos

| Tabla | Uso en Comprobantes |
|---|---|
| `comprobantes_raw` | Fila base extraída de WhatsApp: monto, moneda, banco, titular, referencia, imagen y estado IA. `PUT` la edita; `DELETE` la elimina. |
| `comprobantes_liq` | Snapshot congelado por `hash_largo`. Es la fuente de verdad de importes, tasas y polaridad una vez liquidado. |
| `impactos_raw` | Evidencia cruda de los mensajes. El listado toma el primer impacto por hash. El borrado de un comprobante la marca con `eliminado_en = NOW()`; no destruye la evidencia. |
| `cola_fb` | Cola raw de la consola administrativa. Máximo 100 filas por consulta, orden `timestamp DESC`. |
| `comprobantes_fb` | Tabla raw histórica; se elimina junto con la cola en la misma transacción. |
| `comprobantes_auditados_fb` | Tabla raw auditada; misma regla transaccional que `comprobantes_fb`. |
| `tasas_glaukov` | Solo lectura. Se usa el lote asignado o, si no existe, el más reciente. |
| `perfiles_glaukov` | Solo lectura. Aporta socio, rol, moneda base, `herencia` y configuración `monedas`. |

`migrations/0001_impactos_soft_delete.sql` es idempotente y **no** se ejecuta desde un request. Debe aplicarse antes de desplegar el binario porque el listado y los filtros de Reportes excluyen `eliminado_en IS NOT NULL`.

### 8.3 Snapshot congelado

Entrada: `hash_largo`, monto, moneda, tipo manual (`D`, `P`, `A`) e ID de lote. El motor lee el lote y los perfiles dentro de la misma transacción del `PUT`; `Liquidar` recibe el snapshot ya calculado por el panel y lo persiste idempotentemente.

Defaults de tasas: `USD=1`, `USDT=1`, `PYUSD=1.2`, `ECU=1`, `PAN=1`. El JSONB del lote se superpone; una tasa `0`, `NaN` o `Inf` cae a `1`.

Precisión:

| Valor | Regla |
|---|---|
| Monto | Dos decimales por truncado con epsilon `1e-7`, conservando signo. |
| Tasa comercial (proyección) | Dos decimales; si el valor absoluto supera `99.99`, se trunca a entero; `0`/inválida usa `1`. |
| Tasa snapshot | Precisión adaptativa según magnitud; conserva el comportamiento del legado. |
| Spread | `D`/`A`: `1 + deposito%`; `P`: `1 - pago%`. Los porcentajes se usan en valor absoluto. |

Polaridad:

| Caso | Signo |
|---|---|
| `A` | Socio 1 `+`; socio 2 `-`. |
| `D` | `polaridad = "+"` → `+`; en otro caso `-`. |
| `P` | `polaridad = "+"` → `-`; en otro caso `+`. |
| Herencia | Si exactamente un lado tiene `herencia=true`, su signo se invierte para quedar opuesto al lado no heredante. Si ambos heredan, no hay override. |
| `FUNDDA` | Cuando un lado hereda y existe `FUNDDA`, se usan las `monedas` de `FUNDDA`; si falta la divisa, el contrato por defecto es `tipo=D`, `polar_D="+"`, `polar_P="-"`. |
| `GENERAL` o nombre vacío | Lado inactivo: monto `0`, tasa `1`, ME `0`. |

Ejemplos dorados:

| Caso | Entrada | Resultado |
|---|---|---|
| D estándar | `400000 COP`, tasa `4000`, polaridad `+` | `100 / 4000 / 100`; socio 2 `0 / 1 / 0`. |
| A misma moneda | `1500 USDT`, tasa `1` | `1500 / 1 / 1500`; tipo `A-USDT`. |
| D negativa | `400000 COP`, tasa `4000`, polaridad `-` | `-100 / 4000 / -100`. |
| Herencia FUNDDA | `1000000 COP`, tasa `1000`, depósito `2%`, socio 2 ancla `+` | Socio 1 `-980.39 / 1020 / -980.39`; socio 2 `1000 / 1000 / 1000`. |
| P con pago | `1000000 COP`, tasa `1000`, pago `3%` | `-1030.92 / 970 / -1030.92`. |

### 8.4 Proyección del listado

`calc.ProjectRow` parte del monto absoluto truncado, resuelve socios e idioma de operación y, si la fila está liquidada, reaplica la polaridad sobre las magnitudes congeladas. Si no está liquidada, calcula en vivo con el lote histórico asignado. El tipo visible siempre es `letra-divisa` (`D-COP`, `A-USDT`, `P-COP`). `GENERAL` se serializa como `nombre_socio_2: null`; un array vacío se serializa como `[]`, nunca `null`.

### 8.5 Endpoints canónicos

| Método | Ruta | Contrato |
|---|---|---|
| `GET` | `/api/v1/comprobantes` | Array raíz. Filtros `socio`, `rol`, `fechaInicio`, `fechaFin`, `hash`, `orden`. Fechas inválidas → `400`. Orden inválido → `fecha_desc`. Límite 100. |
| `POST` | `/api/v1/comprobantes/liquidar` | `{success,message,data}` con snapshot. `hash_largo` obligatorio; valores no finitos → `400`. |
| `POST` | `/api/v1/comprobantes/:hashLargo/releer` | Marca `estado_ia='RE-PROCESANDO'` y `procesado_ia=false`. No llama a Gemini. |
| `PUT` | `/api/v1/comprobantes/:hashLargo` | Edita `comprobantes_raw`, recalcula snapshot y hace upsert en `comprobantes_liq` dentro de una transacción. `404` si no existe. |
| `DELETE` | `/api/v1/comprobantes/:hashLargo` | Borra `comprobantes_liq`, marca `impactos_raw.eliminado_en` y borra `comprobantes_raw` en la misma transacción. |
| `GET` | `/api/v1/comprobantes/tasas/ultimas` | Lote vigente o fallback `T001`; nunca `404`. |
| `GET` | `/api/v1/comprobantes/tasas/historial` | Últimos 20 lotes; array posiblemente vacío. |

`PUT` acepta los alias `socio1`/`socio_1`/`nombre_socio_1`, `socio2`/`socio_2`/`nombre_socio_2`, `moneda`/`moneda_recibo`, `id_tasa`/`lote_tasa`/`lote_tasa_asignado`, `tipo_manual`/`tipo_op1`/`tipo_op`, `monto_1`/`m1_socio` y `monto_2`/`m2_socio`. El campo canónico gana sobre el alias.

`OptionalFloat` distingue un campo omitido de un cero explícito. Acepta número JSON o cadena simple con coma de miles (`"1,500.50"`); rechaza `NaN`, `Inf` y el formato local completo (`"1.500,50"`).

### 8.6 Consola administrativa

| Método | Ruta | Entrada | Salida |
|---|---|---|---|
| `GET` | `/api/v1/admin/cola` | `X-Admin-Key` | `{success:true,data:[...]}` con `Cache-Control: no-store`. |
| `PUT` | `/api/v1/admin/cola/:hashLargo` | `X-Admin-Key` + `UpdateInput` | `{success:true,data}`. `nombre_socio_1/2` se reemplazan; `estado`/`conteo` aplican defaults legacy y `timestamp` se conserva si se omite. |
| `DELETE` | `/api/v1/admin/cola/:hashLargo` | `X-Admin-Key` | `{success:true,message}`. Borra las tres tablas raw en una transacción. |

H-06: `ADMIN_API_KEY` vacía → `503 ADMIN_DISABLED` en todas las rutas admin. Clave incorrecta → `401 UNAUTHORIZED`. La comparación usa `platform/httpx.VerifyAdmin` y `subtle.ConstantTimeCompare`; nunca se acepta por query string ni por URL. La clave hardcodeada `ATENEA` fue eliminada. Los datos de cola pueden contener PII y no se cachean.

### 8.7 SQL e índices

- `store/sql_list.go` conserva el CTE legacy: primer y segundo impacto, `comprobantes_liq`, perfiles por nombre/JID y lote histórico. Añade `eliminado_en IS NULL` al CTE de impactos.
- `store/sql_mutate.go` fija el `UPSERT` de `comprobantes_liq`, el `UPDATE` de `comprobantes_raw`, el marcado lógico de impactos y el borrado base.
- `store/repository_mutation_read.go` usa `SELECT ... FOR UPDATE` antes de recalcular.
- `store/sql_report.go` comparte el `FROM` de Reportes para movimientos y resumen, siempre con impactos vigentes.
- La migración crea un índice parcial `LOWER(BTRIM(hash_largo)) WHERE eliminado_en IS NULL`. Se recomienda mantener además índices sobre `comprobantes_raw(creado_en)`, `comprobantes_liq(LOWER(BTRIM(hash_largo)))` y `impactos_raw(LOWER(BTRIM(hash_largo)))`.

### 8.8 Puerto de Reportes

`service.ReportReader` es una interfaz propia de Comprobantes, no de Reportes:

```go
type ReportReader interface {
    FiltrosReporte(ctx context.Context, rol string) (ReportFilters, error)
    MovimientosReporte(ctx context.Context, q ReportQuery) ([]ReportRow, error)
    ResumenReporte(ctx context.Context, q ReportQuery) (ReportSummary, error)
}
```

`store.Repository` implementa las tres consultas en `store/report_reader.go`; el adaptador de `internal/reportes` solo traduce tipos. Reportes no ejecuta SQL ni conoce `comprobantes_liq`. `FiltrosReporte` devuelve roles, entidades y hasta 100 hashes recientes; `MovimientosReporte` filtra por rol, entidad, hash y fechas inclusivas; `ResumenReporte` agrupa por moneda y devuelve total, monto, rango y mapa por divisa.

### 8.9 Errores y validaciones

| Código HTTP | Código estable | Causa |
|---|---|---|
| `400` | `VALIDATION_ERROR` | JSON inválido, hash vacío, fecha inválida, valor no finito. |
| `401` | `UNAUTHORIZED` | Clave admin incorrecta. |
| `404` | `NOT_FOUND` | Comprobante o fila de cola inexistente. |
| `500` | `INTERNAL_ERROR` | Fallo de PostgreSQL u otro error no clasificado; no se expone SQL. |
| `503` | `ADMIN_DISABLED` | `ADMIN_API_KEY` vacía. |

### 8.10 Verificación

```powershell
$env:CGO_ENABLED = "1"   # -race requiere cgo y gcc en PATH
gofmt -l .
go vet ./...
go test ./internal/comprobantes/... -race -v -count=1
go test ./...
```

Tests obligatorios cubiertos:

- Golden de snapshot para D/A, polaridad `+`/`-`, herencia y `FUNDDA`.
- `OptionalFloat` ausente vs cero, coma de miles y rechazo de `NaN`/`Inf`.
- Alias de `PUT`: `socio1`, `moneda_recibo`, `id_tasa`.
- `DELETE` marca `impactos_raw` dentro de la misma transacción y hace rollback si falla el marcado.
- Admin con `ADMIN_API_KEY` vacía → `503`; clave incorrecta → `401`; clave válida → `200` con `no-store`.
- Listado con fechas inválidas → `400`.
- Smoke de las siete rutas canónicas y de las tres rutas admin.

Límites: archivos de producción entre 30 y 180 líneas; tests ≤220 líneas. `gofmt -l .` debe quedar vacío.

## 9. Dominio Reportes

> Fase 9 implementada. `internal/reportes` es un dominio hexagonal puro: no conoce PostgreSQL, HTTP, Fiber, Evolution, perfiles, comprobantes ni tasas.

### 9.1 Arquitectura hexagonal

El paquete raíz `internal/reportes/` solo importa biblioteca estándar permitida: `context`, `time`, `errors`, `strings`, `regexp`, `fmt`, `encoding/base64`, `bytes`, `image`, `image/jpeg`, `crypto/sha256` y `sync`. Las dependencias externas se declaran como puertos y se satisfacen en adaptadores cableados desde `cmd/server`.

| Archivo | Responsabilidad |
|---|---|
| `ports.go` | Puertos `SociosProvider`, `MovimientosProvider`, `WhatsAppSender` y DTOs puros. |
| `models.go` | Casos de uso `SolicitudImagen`, `SolicitudTexto` y `Resultado`. |
| `errors.go` | Errores centinela del dominio. |
| `config.go` | Límites de imagen y TTL de idempotencia. |
| `service.go` | Orquestación de filtros, envío de imagen y envío de texto. |
| `target.go` | Resolución de destino, validación y enmascarado de JID. |
| `image.go` | Limpieza de data URL, validación JPEG y sanitización de nombre. |
| `idempotency.go` | Reserva de claves en memoria con TTL y límite de entradas. |
| `ratelimit.go` | Token bucket por IP. |
| `module.go` | Anclaje del servicio y del estado de proceso. |

### 9.2 Puertos y casos de uso

- `SociosProvider.SocioPorNombre` / `Socios`: única vía para consultar el directorio.
- `MovimientosProvider.Filtros` / `Movimientos` / `Resumen`: filas y agregados ya proyectados, sin columnas internas.
- `WhatsAppSender.EnviarImagen` / `EnviarTexto`: única salida hacia mensajería.

`Service.Filtros` delega en `MovimientosProvider`. `Service.EnviarImagen` resuelve el destino, valida el JPEG, recorta el caption a 1024 caracteres, sanitiza `fileName` y envía `image/jpeg`. `Service.EnviarTexto` recorta a 4096 caracteres. Cualquier fallo del sender se envuelve con `ErrEnvioFallido`, conservando la causa con `errors.Join`.

### 9.3 Flujo de validación JPEG

1. El navegador rasteriza el reporte en un canvas de 1080×1350 y envía el JPEG en base64. El backend nunca recibe HTML ni ejecuta Chromium/Puppeteer.
2. `LimpiarDataURL` acepta base64 puro o `data:image/jpeg;base64,...`; un MIME distinto o un data URL sin `;base64` devuelve `ErrImagenFormato`.
3. Se decodifica base64 y se normaliza a `StdEncoding` sin re-encodear el JPEG.
4. Se aplica `ImageMaxBytes` (10 MiB por defecto) antes de decodificar dimensiones.
5. Se exige la firma binaria `FF D8 FF` y `format == "jpeg"`; PNG, WebP, SVG, texto y polyglots se rechazan con `ErrImagenFormato`.
6. `image.DecodeConfig` lee solo la cabecera: mínimo 200 px por lado, máximo 4096 px por lado y máximo 16 000 000 de píxeles. Los incumplimientos devuelven `ErrImagenDimensiones`.
7. El nombre se limpia de rutas y caracteres de control y se limita a 120 runas. El base64 validado es el que se envía a Evolution.

### 9.4 Reglas de JID

Regex normativa:

```text
^(?:[0-9]{5,25}|[0-9]{5,25}-[0-9]{5,25})@(g\.us|s\.whatsapp\.net|lid)$
```

- `remoteJid` válido tiene prioridad absoluta sobre `socio`; si viene presente, no se consulta el directorio.
- Sin `socio` ni `remoteJid`: `ErrDestinoRequerido`.
- Socio inexistente: `ErrSocioNoEncontrado`.
- `id_grupo` vacío o placeholder terminado en `_JID`: `ErrJIDNoDisponible`.
- JID con formato corrupto (`NOMBRE_JID`, `+549...`, `foo@g.us`, espacios o símbolos): `ErrJIDInvalido`.
- `@c.us` se normaliza a `@s.whatsapp.net`; mayúsculas y el prefijo `whatsapp:` se toleran.
- Los errores usan `maskJID` y nunca exponen el JID completo.

### 9.5 Idempotencia y rate limiting

`IdempotencyStore` usa `sync.Mutex`, TTL de 10 minutos por defecto y un máximo de 1000 entradas. La clave es `Idempotency-Key` si cumple `^[A-Za-z0-9_-]{8,64}$`; si no, `sha256(destino + "\x00" + caption + "\x00" + base64)`. `Begin` reserva la clave antes del envío, `Commit` guarda el `Resultado` y `Abort` libera el fallo. Una clave en curso responde `ErrEnvioEnCurso`; el almacén lleno responde `ErrIdempotenciaLlena`.

`RateLimiter` aplica un token bucket por IP de 60/minuto con ráfaga 10. El handler traduce el rechazo a `429 RATE_LIMITED` con `Retry-After`; `GET /filtros` no se limita.

### 9.6 Verificación del dominio

```powershell
$env:CGO_ENABLED = "1"
go test ./internal/reportes -race -v -count=1
go list -deps ./internal/reportes
```

`architecture_test.go` parsea los archivos de producción con `go/parser` y `go/ast` y falla si aparece un import fuera de la lista cerrada. Los tests cubren `remoteJid` sobre `socio`, `@c.us`, `@g.us`, `@lid`, placeholders, JPEG válido, PNG/SVG/WebP, exceso de bytes, dimensiones, ausencia de base64, texto vacío y fallo del sender.

### 9.7 Cliente unificado de Evolution (`platform/evolution`)

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

### 9.8 Reintentos y errores

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

### 9.9 Simulación

La simulación se activa **únicamente** cuando `EVOLUTION_API_URL` queda vacía (`Client.Simulated() == true`). En ese modo no se abre ninguna conexión y la API key no se utiliza.

- `Status` devuelto: `SIMULATED`.
- `MessageID`: `sim-` + los primeros 8 bytes de `SHA-256(kind|number|content)` en hexadecimal.
- El identificador es determinista para el mismo `kind`, `number` y contenido, lo que permite probar idempotencia y reintentos sin red.
- En producción el arranque rechaza `EVOLUTION_API_URL` vacía; la simulación nunca puede activarse por accidente en ese entorno.

### 9.10 Verificación del cliente Evolution

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

El frontend de construcción de reportes y la superficie HTTP del dominio se completan en las fases siguientes.

## 10. Frontends y routing estático

> `internal/platform/httpx` resuelve host/prefijo, sirve assets con caché diferenciada y aplica el SPA fallback. La entrada principal es la UI **Glauk2 Engine** importada de `glauk2`, servida desde `public/comprobantes/`; agrupa Comprobantes, Cartelera de Tasas, Directorio y Dashboard en una sola shell Alpine.js. La shell **TasasHub Engine** de `public/tasas/` es la migración 1:1 del panel tasasgo: monitor global, checklist de socios, borrador HOO, remesas, cartelera y PWA, dividida en módulos ES con CSS local y sin CDN. Los shells de `perfiles` y `reportes` se conservan como compatibilidad de tenant.

### 10.1 Tabla de tenants

| Tenant | Hosts por defecto | Prefijo path | Shell | API que consume |
|---|---|---|---|---|
| `tasas` | `tasas.localhost` | `/tasas` | `public/tasas/index.html` | `/api/v1/tasas/*` |
| `perfiles` | `perfiles.localhost` | `/perfiles` | `public/perfiles/index.html` | `/api/v1/perfiles/*` |
| `comprobantes` | `comprobantes.localhost` | `/comprobantes` | `public/comprobantes/index.html` (**Glauk2 Engine**) | `/api/v1/comprobantes/*`, `/api/v1/reportes/*`, `/api/v1/tasas/*`, `/api/v1/perfiles/*`, alias `/api/comprobantes`, `/api/directorio`, `/api/tasas/*`, `/api/socios/*` |
| `reportes` | `reportes.localhost` | `/reportes` | `public/reportes/index.html` | `/api/v1/reportes/*` |

`DEFAULT_TENANT=comprobantes` hace que `/` sirva directamente la UI Glauk2. Los hosts reales se configuran con `TENANT_*_HOSTS` (CSV, sin puerto). Se admite un wildcard de un solo nivel: `*.glauk.com` coincide con `tasas.glauk.com` pero no con `a.b.glauk.com`.

### 10.2 Resolución

1. Si `TRUSTED_PROXY_HEADER` está configurado, se usa el primer valor de esa cabecera (`X-Forwarded-Host`) en lugar de `Host`.
2. Se quita el puerto, se normaliza a minúsculas y se compara con `TENANT_*_HOSTS`.
3. Un match de host gana y no se evalúa el path.
4. `HOST_ROUTING_MODE=host`: un host desconocido responde `404`, incluso en `/`; nunca sirve el panel de otro tenant.
5. Sin match de host, se prueba el primer segmento del path contra `/tasas`, `/perfiles`, `/comprobantes` y `/reportes`.
6. Si no hay match y la ruta es `/` o el fallback SPA, se usa `DEFAULT_TENANT`.
7. `DEFAULT_TENANT=none` sirve `public/index.html` solo en `/`; ese archivo redirige a `/comprobantes/`. El resto responde `404`.
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
| `GET /` sin match, `DEFAULT_TENANT=none` | `public/index.html` (redirect a `/comprobantes/`) | `no-cache` |
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
- `index.html` se sirve con la CSP de paneles descrita en §19.3. Glauk2 requiere `script-src 'unsafe-eval'` (Alpine.js), imágenes HTTPS (comprobantes en R2) y `frame-src` para los modales externos de Tasas/Perfiles. La API no recibe CSP porque no devuelve HTML.
- Glauk2 usa `<base href="/comprobantes/">`; sus assets son relativos y el mismo shell funciona en `/` (tenant por defecto) y en `/comprobantes/`.
- El service worker de Glauk2 se registra desde `js/pwa.js` con el scope del tenant y **no** intercepta `/api/`; las mutaciones y los reportes nunca quedan atrapados offline.
- `/shared/*` es común a todos los paneles; los assets de cada tenant viven bajo `public/<tenant>/`.

### 10.6 Estructura modular (Fase 11)

```text
public/
├── index.html                  redirect a /comprobantes/ (DEFAULT_TENANT=none)
├── robots.txt                  Disallow: /api/
├── shared/
│   ├── brand/                  logo FUNDABLOCK y favicon local
│   ├── css/                    base.css, components.css, print.css (shells de compatibilidad)
│   ├── js/                     http, dom, format, toast, auth, reporte-canvas (shells de compatibilidad)
│   └── vendor/
│       ├── html2canvas.min.js  html2canvas 1.4.1 (MIT, copia local)
│       └── glauk2/             tailwind.css compilado y alpine.min.js vendorizado
├── comprobantes/               Glauk2 Engine (UI unificada)
│   ├── index.html              shell con vistas Comprobantes/Cartelera/Directorio/Dashboard
│   ├── js/appBootstrap.js      composición reactiva de los estados
│   ├── js/modules/             estados y servicios por dominio
│   ├── js/shared/              constantes, router y utilidades
│   ├── js/pwa.js               registro externo del service worker (sin inline)
│   ├── manifest.json           PWA de Glauk2
│   ├── sw.js                   service worker pass-through
│   └── logo-fundablock.png     icono PWA
├── tasas/                      TasasHub Engine (monitor, checklist, HOO, PWA; módulos ES y CSS local)
├── perfiles/                   shell de compatibilidad
└── reportes/                   shell de compatibilidad
```

La shell Glauk2 contiene la estructura y los puntos de montaje; la navegación, el estado y las llamadas viven en módulos ES. Los estados se combinan preservando descriptores para que los getters de Alpine (`comprobantes`, `movimientoFiltradoTotal`, `directorioFiltrado`) sigan siendo reactivos. No hay scripts inline: incluso el registro del service worker vive en `js/pwa.js`.

Los límites son 180 líneas por HTML, 250 por módulo JS/CSS y exclusión total de `shared/vendor/` del presupuesto. La UI importada de `glauk2` bajo `public/comprobantes/` está exenta del presupuesto de líneas para conservar paridad visual 1:1; no se recorta ni se parte artificialmente. El resto de los shells mantiene los límites.

### 10.7 Política de assets y CSP

- Glauk2 usa `<base href="/comprobantes/">` y assets relativos (`./js/...`, `/shared/vendor/glauk2/...`). La misma shell funciona en modo host (`/`) y en modo path (`/comprobantes/`).
- Las llamadas de negocio siguen siendo absolutas y canónicas: `/api/v1/comprobantes/*`, `/api/v1/reportes/*`, `/api/v1/tasas/*`, `/api/v1/perfiles/*` y `/api/v1/admin/*`. Glauk2 usa además los alias legacy `/api/comprobantes`, `/api/directorio`, `/api/tasas/*` y `/api/socios/*` para mantener la paridad funcional.
- No hay dependencias de CDN en tiempo de ejecución. Tailwind se compiló localmente en `public/shared/vendor/glauk2/tailwind.css`; Alpine y `html2canvas` están vendorizados. Las fuentes, el logo y el favicon se sirven desde `public/shared/`.
- El redirect `public/index.html` solo se sirve en `/` cuando `DEFAULT_TENANT=none`; apunta a `/comprobantes/` y no requiere JavaScript.
- `public/robots.txt` bloquea explícitamente `/api/`.

### 10.8 Service workers y aislamiento

Glauk2 registra el service worker desde un módulo externo, sin script inline:

```js
const script = new URL('../sw.js', import.meta.url);
navigator.serviceWorker.register(script);
```

El scope efectivo es `/comprobantes/`; en modo host la página `/` sigue funcionando, pero el worker controla solo ese prefijo. `public/comprobantes/sw.js` hace `skipWaiting`, `clients.claim` y un `fetch` pass-through deliberado: no cachea el shell ni las respuestas de `/api/`. Los datos contables, las mutaciones y los envíos de reportes siempre van a la red. `perfiles` y `reportes` conservan su propio `sw.js`; TasasHub registra el suyo con el mismo comportamiento pass-through.

### 10.9 Pipeline de canvas

Glauk2 rasteriza el documento visible `#reporte-html-documento` con el `html2canvas` vendorizado y envía el JPEG a `POST /api/v1/reportes/imagen`:

| Regla | Implementación |
|---|---|
| Documento | `#reporte-html-documento` dentro del modal Reportes |
| Fondo | `#FFFFFF`, sin transparencia |
| Escala | `2` (1600 × 2000 px para el documento de 800 × 1000) |
| Formato | `canvas.toDataURL('image/jpeg', 0.92)` |
| Transporte | JSON con `base64` (data URL), `socio`, `caption`, `fileName` y `mimeType` |
| Destino | `socio` seleccionado; vacío se normaliza a `GENERAL` |
| Reintento | el botón se deshabilita mientras `generandoImagen` está activo |

El backend valida la firma JPEG, aplica idempotencia por cabecera `Idempotency-Key` o por hash estable del contenido y responde con `simulated` cuando Evolution no está configurado. La UI muestra el resultado sin dejar el modal en estado de carga. El backend recibe únicamente la imagen; nunca HTML.

### 10.10 Verificación

```powershell
$env:CGO_ENABLED = "1"
go test ./internal/platform/httpx -race -v
.\scripts\check-file-length.ps1
```

Cubre host routing, path routing, prioridad del host, `/api/*` en JSON, traversal con `403`, asset inexistente con `404`, SPA fallback y las políticas de caché. `check-file-length.ps1` aplica los límites al resto del repositorio y excluye `public/shared/vendor/` y la UI importada de `public/comprobantes/`; el redirect `public/index.html` queda dentro del límite de 120 líneas.

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

### 11.8 Matriz canónica vs alias

La superficie canónica es la única documentada en `docs/openapi.yaml`. Los alias existen solo para compatibilidad y siempre responden con `Deprecation`, `Sunset` y `Link`.

| Dominio | Método y ruta canónica | Alias legacy |
|---|---|---|
| Health | `GET /health` | — |
| Health | `GET /ready` | — |
| Tasas | `GET /api/v1/tasas/monitor` | `GET /api/tasas/monitor` |
| Tasas | `GET /api/v1/tasas/socios` | `GET /api/tasas/socios` |
| Tasas | `GET /api/v1/tasas/hoo-borrador` | `GET /api/tasas/hoo-borrador`, `GET /api/tasas/fetch-hoo` |
| Tasas | `PUT /api/v1/tasas/socios/:socio` | `PUT /api/tasas/socios/:socio` |
| Tasas | `POST /api/v1/tasas/guardar-lote` | `POST /api/tasas/guardar-lote` |
| Tasas | `POST /api/v1/tasas/publicar-lote` | `POST /api/tasas/publicar-lote`, `POST /api/tasas/publicar`, `POST /api/tasas/reenviar` |
| Tasas | `POST /api/v1/tasas/notificar-remesas` | `POST /api/tasas/notificar-remesas` |
| Tasas | `GET /api/v1/tasas/calcular/:socio` | `GET /api/tasas/calcular/:socio` |
| Tasas | `GET /api/v1/tasas/render/:socio` | `GET /api/tasas/render/:socio`, `GET /api/preview-image/:identificador?` |
| Tasas | `POST /api/v1/tasas/render` | `POST /api/tasas/render` |
| Tasas | `POST /api/v1/tasas/disparar` | `POST /api/tasas/disparar` |
| Tasas | `POST /api/v1/tasas/disparar-lote` | `POST /api/tasas/disparar-lote` |
| Tasas | `DELETE /api/v1/tasas/lote/:id_tasa` | `DELETE /api/tasas/lote/:id_tasa` |
| Perfiles | `GET /api/v1/perfiles` | `GET /api/directorio`, `GET /api/socios`, `GET /api/v1/atenea/directorio`, `GET /api/v1/atenea/socios` |
| Perfiles | `GET /api/v1/perfiles/activos` | — |
| Perfiles | `POST /api/v1/perfiles/guardar` | `POST /api/socios/config`, `POST /api/v1/atenea/socios/config` |
| Perfiles | `POST /api/v1/perfiles/apagar-todos` | `PATCH /api/socios/desactivar-todos`, `PATCH /api/v1/atenea/socios/desactivar-todos` |
| Perfiles | `POST /api/v1/perfiles/plantilla/memorizar` | `POST /api/socios/guardar-vigentes`, `POST /api/v1/atenea/socios/guardar-vigentes` |
| Perfiles | `POST /api/v1/perfiles/plantilla/restaurar` | `POST /api/socios/restaurar-vigentes`, `POST /api/v1/atenea/socios/restaurar-vigentes` |
| Perfiles | `PATCH /api/v1/perfiles/:nombre/estado` | `PATCH /api/socios/:nombre/estado`, `PATCH /api/v1/atenea/socios/:nombre/estado` |
| Perfiles | `DELETE /api/v1/perfiles/:nombre` | `DELETE /api/directorio/:nombre`, `DELETE /api/v1/atenea/directorio/:nombre` |
| Comprobantes | `GET /api/v1/comprobantes` | `GET /api/comprobantes`, `GET /api/cola`, `GET /api/reportes`, `GET /api/reportes/operaciones`, `GET /api/v1/atenea/comprobantes`, `GET /api/v1/atenea/cola`, `GET /api/v1/atenea/reportes`, `GET /api/v1/atenea/reportes/operaciones` |
| Comprobantes | `POST /api/v1/comprobantes/liquidar` | `POST /api/comprobantes/liquidar`, `POST /api/v1/atenea/comprobantes/liquidar` |
| Comprobantes | `POST /api/v1/comprobantes/:hashLargo/releer` | `POST /api/comprobantes/:hashLargo/releer`, `POST /api/v1/atenea/comprobantes/:hashLargo/releer` |
| Comprobantes | `PUT /api/v1/comprobantes/:hashLargo` | `PUT /api/comprobantes/:hashLargo`, `PUT /api/v1/atenea/comprobantes/:hashLargo` |
| Comprobantes | `DELETE /api/v1/comprobantes/:hashLargo` | `DELETE /api/comprobantes/:hashLargo`, `DELETE /api/v1/atenea/comprobantes/:hashLargo` |
| Comprobantes | `GET /api/v1/comprobantes/tasas/ultimas` | `GET /api/tasas/ultimas`, `GET /api/v1/atenea/tasas/ultimas` |
| Comprobantes | `GET /api/v1/comprobantes/tasas/historial` | `GET /api/tasas/historial`, `GET /api/v1/atenea/tasas/historial` |
| Reportes | `GET /api/v1/reportes/filtros` | `GET /api/reportes/filtros`, `GET /api/v1/atenea/reportes/filtros` |
| Reportes | `POST /api/v1/reportes/imagen` | `POST /api/reportes/enviar-whatsapp`, `POST /api/whatsapp/enviar-media`, `POST /api/v1/atenea/reportes/enviar-whatsapp`, `POST /api/v1/atenea/whatsapp/enviar-media` |
| Reportes | `POST /api/v1/reportes/texto` | — |
| Admin | `GET /api/v1/admin/cola` | `GET /api/admin/cola`, `GET /api/v1/atenea/admin/cola` |
| Admin | `PUT /api/v1/admin/cola/:hashLargo` | `PUT /api/admin/cola/:hashLargo` |
| Admin | `DELETE /api/v1/admin/cola/:hashLargo` | `DELETE /api/admin/cola/:hashLargo` (acepta `{claveAdmin}`) |

`GET /api/reportes` nunca pertenece al dominio Reportes: es el listado de Comprobantes. Las rutas específicas `/api/reportes/filtros` y `/api/reportes/enviar-whatsapp` se registran antes que el alias general para conservar esa precedencia.

### 11.9 Política de deprecación y rutas retiradas

- `Sunset` es `Sat, 31 Dec 2027 23:59:59 GMT` para todos los alias.
- `Deprecation: true` y `Link: <sucesor>; rel="successor-version"` acompañan siempre al alias, incluso cuando el guard responde `401` o `503`.
- Un alias no cambia de semántica ni de envelope; el contador `alias_usage_total` permite decidir su retiro.
- Los alias `/api/v1/atenea/*` son los primeros candidatos a retiro.
- `POST /api/tasas/n8n-webhook` responde `410 GONE` con `code: ROUTE_REMOVED` y `Link` a `/api/v1/tasas/guardar-lote`.
- `/disparar` solo acepta `POST`; cualquier otro método responde `405 METHOD_NOT_ALLOWED` con `Allow: POST`.
- Un `/api/*` desconocido responde `404` JSON con `{success:false,code:"NOT_FOUND"}` y nunca cae en el SPA.

### 11.10 OpenAPI y verificación de rutas

La especificación está en [`docs/openapi.yaml`](docs/openapi.yaml). Es OpenAPI 3.1, documenta únicamente `/health`, `/ready` y las 36 rutas canónicas, e incluye los esquemas `ApiKeyAuth` y `AdminKeyAuth`. Los alias no se publican como endpoints de la API v1.

```powershell
npx --yes @redocly/cli@latest lint docs/openapi.yaml
.\scripts\check-routes.ps1
go test ./cmd/server -race -v -count=1
```

`TestNoDuplicateRoutes` recorre `app.GetRoutes(true)` y falla ante cualquier par `Method + Path` repetido. `TestRouteManifest` imprime la superficie para `check-routes.ps1`; el script compara las 36 canónicas, los 64 alias y la ruta retirada contra la matriz de §5, y falla si una firma cambia o desaparece.

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

La seguridad operativa y el hardening se detallan en el **Capítulo 19**. En resumen: admin protegido por `ADMIN_API_KEY`, mutaciones opcionalmente protegidas por `SECURITY_API_KEY`, comparación en tiempo constante, CORS restringido, cabeceras de seguridad, límites de body/imagen, rate limit, idempotencia, secretos fuera de la imagen y runtime distroless nonroot. El checklist de despliegue y la respuesta a fuga de claves están en §19.9 y §19.10.

## 14. Operación

El despliegue de Glauk es un único binario y un único contenedor. La imagen final es `distroless`, se ejecuta como usuario `nonroot` y solo contiene `/app/glauk` y `/app/public`; el código fuente, las dependencias de compilación y los archivos `.env` quedan exclusivamente en el stage de build.

### 14.1 Dockerfile explicado línea a línea

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/glauk ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/glauk /app/glauk
COPY --from=build /src/public /app/public
ENV PORT=3000
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/app/glauk"]
```

| Línea / bloque | Por qué existe |
|---|---|
| `# syntax=docker/dockerfile:1` | Activa BuildKit y fija la sintaxis del Dockerfile. No forma parte de la imagen. |
| `FROM golang:1.27.2-alpine AS build` | Toolchain reproducible para compilar; el stage se llama `build` y no llega al runtime. |
| `WORKDIR /src` | Directorio único del build; todas las rutas relativas cuelgan de aquí. |
| `COPY go.mod go.sum ./` | Copia solo los manifiestos primero para aprovechar la caché de capas. |
| `RUN go mod download` | Descarga las dependencias en una capa separada; si el código cambia pero `go.mod`/`go.sum` no, se reutiliza. |
| `COPY . .` | Copia el contexto a `/src` solo durante el build. `.dockerignore` excluye `.git`, `.env`, binarios, logs y `coverage.out`. |
| `ARG TARGETOS=linux` | Permite que Buildx sobrescriba el sistema operativo destino; por defecto, Linux. |
| `ARG TARGETARCH=amd64` | Permite compilar para `amd64`, `arm64`, etc. sin editar el archivo. |
| `RUN CGO_ENABLED=0 ... go build ...` | Genera un binario estático, sin dependencias de libc, con `-trimpath` (rutas reproducibles) y `-ldflags="-s -w"` (símbolos y depuración eliminados). La salida es `/out/glauk` desde `./cmd/server`. |
| `FROM gcr.io/distroless/static-debian12:nonroot` | Runtime mínimo: sin shell, sin gestor de paquetes, sin compiladores y con el usuario `nonroot` predefinido. |
| `WORKDIR /app` | El binario y los estáticos viven en `/app`; el `PUBLIC_DIR` por defecto (`./public`) resuelve a `/app/public`. |
| `COPY --from=build /out/glauk /app/glauk` | Único ejecutable de la imagen final. |
| `COPY --from=build /src/public /app/public` | Recursos estáticos servidos por el binario; no hay servidor web adicional. |
| `ENV PORT=3000` | Puerto por defecto de la aplicación dentro del contenedor. |
| `EXPOSE 3000` | Documenta el puerto; el mapeo real lo hace Easypanel o `docker run -p`. |
| `USER nonroot:nonroot` | El proceso no corre como root; el usuario existe en la imagen distroless. |
| `ENTRYPOINT ["/app/glauk"]` | Ejecución directa sin shell, con señales y `SIGTERM` llegando al proceso Go. |

La imagen final **no** contiene `go.mod`, `go.sum`, `cmd/`, `internal/`, `.env`, tests, compiladores ni un shell. No se puede usar `docker exec ... sh` para depurar; el diagnóstico se hace por logs, `/health`, `/ready` y el estado del contenedor. No se define un `HEALTHCHECK` dentro de la imagen porque distroless no incluye `curl` ni `wget`; el healthcheck HTTP lo ejecuta Easypanel o el smoke test externo.

### 14.2 Build y ejecución local

```powershell
docker build -t glauk:local .
docker image inspect glauk:local --format '{{.Size}}'
docker run --rm --name glauk -p 3000:3000 --env-file .env glauk:local
```

`DATABASE_URL` debe ser accesible desde dentro del contenedor. En Windows/macOS, `localhost` dentro del contenedor no es el host: usar el nombre del servicio de Compose o `host.docker.internal` si se ejecuta con `docker run`. El archivo `.env` se pasa en tiempo de ejecución y nunca se copia a la imagen.

Stack local completo:

```powershell
docker compose up -d --build
.\scripts\smoke.ps1
docker compose logs -f glauk
docker compose down
```

El compose levanta `glauk` en `3000:3000` y `postgres:16-alpine` en una red `db` marcada como `internal`; Postgres no publica puertos. `glauk` se conecta a `db` y a una red `edge` para conservar salida a Evolution si se configura. El volumen `glauk_pgdata` persiste la base local. `scripts/local-db-init.sql` crea el mínimo de tablas y datos para que el smoke valide monitor y filtros en una base nueva; **es solo un seed de desarrollo**, no una migración de producción. `docker compose down -v` elimina también ese volumen y vuelve a ejecutar el seed en el siguiente arranque.

### 14.3 Easypanel

| Elemento | Valor |
|---|---|
| Tipo de servicio | Aplicación desde Dockerfile. |
| Repositorio / contexto | Repositorio Glauk, directorio `Core`, Dockerfile `Dockerfile`. |
| Puerto interno | `3000`. Easypanel publica 80/443 a través de su proxy TLS. |
| Dominio | Dominio del panel o de la API; TLS termina en el proxy. No se necesita certificado dentro del contenedor. |
| Healthcheck | `GET /health`, intervalo 5 s, timeout 5 s, 3 reintentos, periodo de gracia inicial 15 s. |
| Réplicas | **1**. La idempotencia de Reportes y el rate limit viven en memoria del proceso. |
| Reinicio | `unless-stopped` o la política equivalente de Easypanel. |
| Zona horaria | `TZ=UTC`. |
| Persistencia | Ninguna en el contenedor: el binario es stateless. PostgreSQL es un servicio gestionado o un contenedor separado, nunca expuesto públicamente. |

Variables mínimas de producción (los secretos se cargan como *secrets* de Easypanel, no en la imagen):

| Variable | Valor / uso |
|---|---|
| `APP_ENV` | `production`. |
| `PORT` | `3000`. |
| `PUBLIC_DIR` | `/app/public` (explícito; el default también resuelve ahí por el `WORKDIR`). |
| `DATABASE_URL` | URL de PostgreSQL gestionado. **Secreto.** Tiene prioridad sobre `DB_*`. |
| `EVOLUTION_API_URL` | URL de Evolution. Si queda vacía, el envío entra en simulación. |
| `EVOLUTION_API_KEY` | Clave de Evolution. **Secreto.** |
| `EVOLUTION_INSTANCE` | Instancia de Evolution, por defecto `Glauk`. |
| `TZ` | `UTC`. |
| `LOG_LEVEL` | `info`; subir a `debug` solo de forma temporal. |
| `LOG_FORMAT` | `json` en producción para que el visor pueda filtrar campos. |
| `SHUTDOWN_TIMEOUT` | `15s`; el proxy debe permitir al menos ese margen antes de matar el contenedor. |
| `TRUSTED_PROXY_HEADER` | `X-Forwarded-Host` detrás del proxy de Easypanel. |
| `CORS_ALLOWED_ORIGINS` | Origen exacto del panel; evitar `*` si el panel usa cookies o credenciales. |
| `SECURITY_API_KEY` | Opcional; si se define, protege las mutaciones con `X-API-Key`. **Secreto.** |
| `ADMIN_API_KEY` | Opcional; si queda vacía, la consola administrativa responde 503. **Secreto.** |

No se debe definir `AUTHENTICATION_API_KEY` junto con `EVOLUTION_API_KEY`: el alias legado existe solo para migración y el arranque falla si ambas difieren. La lista completa y sus defaults está en `.env.example` y en el Capítulo 5.

### 14.4 Health y ready

| Sonda | Contrato | Interpretación |
|---|---|---|
| `GET /health` | 200 con `status:"ok"`, `service:"glauk"`, `checks.database.status:"ok"`, `checks.evolution.status`, `uptime_s` y `version`. Si el ping a PostgreSQL falla, responde 503 con `status:"degraded"` y `database.status:"error"`. | Sonda operativa completa. Incluye base de datos y modo de Evolution; `Cache-Control: no-store`. |
| `GET /ready` | 200 con `status:"ok"` cuando el ping a PostgreSQL funciona; 503 cuando no. No incluye `version` ni `instance`. | Sonda de disponibilidad para enrutar tráfico. |

Ambas usan un timeout acotado de 2 s para el ping. Si `/health` falla:

1. Revisar logs del contenedor y la conectividad a `DATABASE_URL`.
2. Confirmar que PostgreSQL acepta conexiones y que el pool no está agotado.
3. No reiniciar en bucle si la base está caída: el proceso está sano pero degradado.
4. Si solo `checks.evolution` está en `simulated`, no es un fallo: `EVOLUTION_API_URL` está vacía y los envíos se simulan.
5. Después de corregir, esperar a que `/ready` responda 200 antes de reabrir tráfico.

El smoke test valida además cabeceras de seguridad: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Permissions-Policy` y `Cross-Origin-Opener-Policy: same-origin`. `Strict-Transport-Security` solo se exige cuando `BaseUrl` usa HTTPS, porque el middleware lo emite únicamente sobre TLS.

### 14.5 Logs

El logger usa texto o JSON según `LOG_FORMAT`. En producción usar `json`. Cada request registra, sin cuerpos ni cabeceras sensibles:

- `request_id`: mismo valor que la cabecera `X-Request-Id` de la respuesta.
- `method`, `path`, `status`, `duration_ms`, `bytes`.
- `tenant`, cuando el enrutamiento por host lo resolvió.

Consultas útiles:

```powershell
docker compose logs -f glauk
docker compose logs glauk | Select-String '"status":5'
docker compose logs glauk | Select-String 'request_id'
```

En Easypanel, usar el visor de logs y filtrar por `status`, `path` o `request_id`. Para un incidente, conservar el `request_id` del panel; permite correlacionar la respuesta con la línea exacta sin exponer payloads. Nunca se registran bases64, captions, JIDs completos, API keys ni el cuerpo de los errores de PostgreSQL.

### 14.6 Métricas mínimas

En la fase 1 no hay Prometheus; se usan las métricas del proveedor, los logs estructurados y las sondas:

| Métrica | Cómo obtenerla | Umbral de atención |
|---|---|---|
| Latencia de `/health` | `duration_ms` del log o el smoke test | p95 > 1 s de forma sostenida. |
| Errores HTTP 5xx | Logs filtrados por `status:5xx` | Cualquier aumento sostenido sobre la línea base. |
| Envíos de Reportes | Eventos de envío y respuestas de Evolution | Aumento de fallos o `EVOLUTION_UNAVAILABLE`. |
| Latencia de PostgreSQL | `checks.database.latency_ms` en `/health` | Cerca del timeout de 2 s. |
| Reinicios del contenedor | Estado de Easypanel / `docker ps` | Más de un reinicio no planificado. |
| Tamaño de imagen | `docker image inspect glauk:local --format '{{.Size}}'` | Crecimiento anómalo tras un cambio de base. |

El `uptime_s` de `/health` ayuda a distinguir un reinicio reciente de una caída de base. El healthcheck de Easypanel no sustituye una alerta de latencia: un 200 lento sigue siendo un problema de operación.

### 14.7 Escalado y limitaciones

La arquitectura actual es de **una réplica**:

- `IdempotencyStore` y `RateLimiter` de Reportes son estado en memoria del proceso. Con dos réplicas, un doble clic puede generar dos envíos si cae en instancias distintas, y el límite de 60/minuto se aplica por instancia, no de forma global.
- El apagado ordenado espera hasta `SHUTDOWN_TIMEOUT` (15 s) para terminar envíos en curso. El orquestador debe dar más margen; el compose usa `stop_grace_period: 20s`.
- No hay sesiones de usuario ni estado de aplicación en disco. Escalar horizontalmente solo es seguro después de mover idempotencia y rate limit a un almacén compartido (PostgreSQL o Redis) y ajustar el TTL/límite global.

Hasta entonces, mantener **1 réplica**. Si se necesita más capacidad, primero separar lectura/escritura o migrar el estado compartido; no basta con subir el número de réplicas en Easypanel.

### 14.8 Rollback y mantenimiento

**Imagen:** volver a desplegar la imagen o el commit anterior. En Easypanel, seleccionar el deployment previo; en local:

```powershell
git checkout f13-ci
docker compose up -d --build
```

El tag de esta fase es `f14-docker`. El rollback de aplicación no requiere tocar DNS ni TLS: el dominio sigue apuntando al mismo servicio y el proxy sigue terminando HTTPS.

**Base de datos:** hacer backup antes de cualquier cambio de esquema y desplegar primero una versión compatible. Si el rollback de código no es suficiente, restaurar el backup con el servicio detenido y volver a ejecutar el smoke. En local, `docker compose down` conserva `glauk_pgdata`; `docker compose down -v` lo destruye y no debe usarse como rollback de producción.

**Ventana de mantenimiento:** con una sola réplica hay un corte breve. Cerrar el tráfico en el proxy, esperar el apagado ordenado, ejecutar backup, desplegar, esperar `/ready` 200 y solo entonces reabrir. El smoke test debe pasar antes de declarar el despliegue terminado.

## 15. Pruebas y calidad

La calidad de Glauk se verifica con pruebas ejecutables y puertas de CI, no con revisión manual. Las puertas obligatorias son `gofmt`, `go vet`, `go test -race`, el presupuesto de líneas, el aislamiento arquitectónico, la matriz de rutas y los umbrales de cobertura de §6.3.

### 15.1 Pirámide de pruebas

| Nivel | Qué verifica | Implementación | Se ejecuta en CI |
|---|---|---|---|
| Unitario de dominio | Reglas financieras, polaridad, perfiles, filtros y estados. | `internal/comprobantes/calc`, `internal/comprobantes/domain`, `internal/tasas/calculator`, `internal/perfiles/model`. | Sí, con `go test`. |
| Servicio y casos de uso | Orquestación, errores públicos, defaults y transacciones simuladas. | `internal/*/service`, `internal/*/store`. | Sí, con dobles y `pgxmock`. |
| Contract test de adaptadores | Puertos de Reportes, Evolution y proyecciones entre dominios. | `internal/reportes/adapters`, `internal/platform/evolution`. | Sí, con fakes y `httptest`. |
| HTTP y routing | Envelopes, códigos de error, aliases, guards y colisiones de rutas. | `internal/*/http`, `cmd/server`, `scripts/check-routes.ps1`. | Sí, con `app.Test`. |
| Integración PostgreSQL | SQL, escaneo, transacciones y rollback. | `internal/*/store` con `pgxmock`; Postgres efímero para verificación manual. | Sí la parte mockeada. |
| E2E manual del canvas | Pipeline de imagen y descarga en navegador. | Capítulo 10; fixtures JPEG y prueba manual por tenant. | No. |

La cobertura por paquete es local: Go instrumenta el paquete cuyas pruebas se ejecutan, no los paquetes que lo consumen desde otro test. Por eso un handler puede aparecer cubierto por su prueba HTTP y no por la del dominio que lo llama.

### 15.2 Matriz de pruebas

| Área | Archivos principales | Casos clave |
|---|---|---|
| Configuración | `platform/config/config_test.go` | Precedencia, defaults, errores agregados, DSN y timeouts. |
| Evolution | `platform/evolution/*_test.go` | 2xx, 4xx, 429, 5xx, timeout, reintentos, simulación y redacción de API key. |
| HTTP compartido | `platform/httpx/*_test.go` | Envelopes, errores, health, estáticos, host/path y traversal. |
| Postgres | `platform/postgres/*_test.go` | Pool, ping, timeouts, `WithTx`, commit, rollback y pánico. |
| Tasas | `tasas/calculator/calculator_test.go`, `tasas/render/*_test.go`, `tasas/service/service_test.go`, `tasas/store/*_test.go`, `tasas/http/routes_test.go` | Golden de carteleras, triangulación, PNG, modo prueba, lotes, socios y aliases. |
| Perfiles | `perfiles/model/*_test.go`, `perfiles/repo/*_test.go`, `perfiles/service/service_test.go`, `perfiles/http/routes_test.go` | Tres formas del codec `monedas`, round-trip, actividad H-02, defaults y rutas. |
| Comprobantes | `comprobantes/calc/*_test.go`, `comprobantes/domain/domain_test.go`, `comprobantes/service/service_test.go`, `comprobantes/http/routes_test.go` | Snapshot congelado, D/A, polaridad, herencia, proyección, filtros y mutaciones. |
| Reportes | `reportes/service_test.go`, `reportes/image_test.go`, `reportes/target_test.go`, `reportes/state_test.go`, `reportes/http/*_test.go`, `reportes/adapters/*_test.go` | Puertos, JPEG, JID, idempotencia, rate limit, aliases legacy, Evolution y arquitectura. |
| Rutas del servidor | `cmd/server/routes_test.go`, `routes_aliases_test.go`, `routes_manifest_test.go` | Registro, alias, manifest y ausencia de colisiones. |

### 15.3 Golden fixtures

- **Tasas:** `internal/tasas/calculator/testdata/golden_calculator.json` se generó desde el servicio Node original con `testdata/gen_golden_calculator.mjs`. Congela 18 carteleras, 37 valores de precisión y 13 instantes horarios. `TestGolden` compara con tolerancia `1e-9` y `cmpopts.EquateEmpty`; la fuente de verdad es el comportamiento legacy, no el resultado actual.
- **Comprobantes:** `internal/comprobantes/calc/snapshot_test.go` congela cinco casos: depósito estándar, abono en la misma moneda, polaridad negativa, herencia de FUNDDA y pago con spread. La comparación es exacta con `cmp.Diff`; `snapshot.go` exige un fixture nuevo ante cualquier cambio de regla.
- **No se actualiza un fixture para que pase una prueba.** Primero se identifica si el cambio es una regla nueva deliberada; si lo es, se agrega el caso y se documenta. Si no, se corrige el código.

Reproducir un fallo dorado:

```powershell
go test ./internal/tasas/calculator -run TestGolden -v
go test ./internal/comprobantes/calc -run TestCalcularSnapshotFinancieroGolden -v
```

### 15.4 Arquitectura y presupuesto

Además del test `internal/reportes/architecture_test.go`, que cierra los imports del dominio a la lista permitida, hay dos auditores de repositorio:

```powershell
.\scripts\check-file-length.ps1
.\scripts\check-architecture.ps1
.\scripts\check-routes.ps1
```

`check-file-length.ps1` evalúa todo `Core/` y falla si:

| Tipo | Límite |
|---|---|
| Go de producción | 30 a 200 líneas; objetivo SRP 30 a 180. |
| Go de pruebas (`*_test.go`) | Máximo 220 líneas. |
| HTML | Máximo 180 líneas. |
| JavaScript y CSS | Máximo 250 líneas, excluyendo `public/shared/vendor/`. |
| UI Glauk2 (`public/comprobantes/**/*.html`, `*.js`) | Exenta: paridad visual 1:1 con `glauk2`; no se recorta. |

`check-architecture.ps1` falla con código 1 si detecta: imports de pgx, perfiles, comprobantes, tasas, Postgres o Evolution desde `reportes` raíz/http; lectura de entorno fuera de `platform/config`; construcción de `pgxpool` fuera de `cmd/server` o `platform/postgres`; escrituras a `perfiles_glaukov` fuera de `internal/perfiles`; imports de dominios desde `platform/*`; o SQL en el dominio `reportes` (los adaptadores están excluidos). Los tests no se escanean como producción.

### 15.5 Cobertura

Los umbrales de §6.3 se cumplen. La cobertura global es ponderada por sentencias: **2.520 de 3.476 = 72,5 %**. No se persigue 100 %; se priorizan las rutas financieras y los contratos públicos.

| Paquete | Cobertura | Puerta |
|---|---:|---|
| `internal/reportes` | 95,7 % | ≥ 80 % ✅ |
| `internal/reportes/http` | 88,0 % | Subárbol de Reportes ✅ |
| `internal/reportes/adapters` | 96,4 % | Subárbol de Reportes ✅ |
| `internal/tasas/calculator` | 94,8 % | ≥ 75 % ✅ |
| `internal/comprobantes/calc` | 95,3 % | ≥ 75 % ✅ |
| **Global** | **72,5 %** | **≥ 70 % ✅** |

Cobertura obtenida por paquete:

| Paquete | Sentencias | Cubiertas | % |
|---|---:|---:|---:|
| `glauk/cmd/server` | 290 | 154 | 53,1 |
| `glauk/internal/comprobantes/calc` | 236 | 225 | 95,3 |
| `glauk/internal/comprobantes/domain` | 48 | 48 | 100,0 |
| `glauk/internal/comprobantes/dto` | 61 | 41 | 67,2 |
| `glauk/internal/comprobantes/http` | 95 | 70 | 73,7 |
| `glauk/internal/comprobantes/service` | 129 | 67 | 51,9 |
| `glauk/internal/comprobantes/store` | 222 | 14 | 6,3 |
| `glauk/internal/perfiles/http` | 56 | 44 | 78,6 |
| `glauk/internal/perfiles/model` | 120 | 109 | 90,8 |
| `glauk/internal/perfiles/repo` | 116 | 86 | 74,1 |
| `glauk/internal/perfiles/service` | 69 | 52 | 75,4 |
| `glauk/internal/platform/config` | 273 | 194 | 71,1 |
| `glauk/internal/platform/evolution` | 133 | 118 | 88,7 |
| `glauk/internal/platform/httpx` | 301 | 240 | 79,7 |
| `glauk/internal/platform/logging` | 39 | 37 | 94,9 |
| `glauk/internal/platform/postgres` | 65 | 56 | 86,2 |
| `glauk/internal/reportes` | 207 | 198 | 95,7 |
| `glauk/internal/reportes/adapters` | 56 | 54 | 96,4 |
| `glauk/internal/reportes/http` | 133 | 117 | 88,0 |
| `glauk/internal/tasas/calculator` | 96 | 91 | 94,8 |
| `glauk/internal/tasas/hoo` | 46 | 42 | 91,3 |
| `glauk/internal/tasas/http` | 111 | 75 | 67,6 |
| `glauk/internal/tasas/render` | 266 | 237 | 89,1 |
| `glauk/internal/tasas/service` | 184 | 59 | 32,1 |
| `glauk/internal/tasas/store` | 124 | 92 | 74,2 |

`internal/tasas/domain` no tiene archivos de prueba propios; se ejercita a través de `calculator` y `store`. Los paquetes sin umbral específico no bloquean la fase, pero cualquier nueva ruta crítica debe añadir pruebas en su propio paquete para que la cobertura local no dependa de otro binario de test.

### 15.6 Comandos de verificación

Ejecutar desde `Core/`:

```powershell
gofmt -l .
go vet ./...
.\scripts\check-file-length.ps1
.\scripts\check-architecture.ps1
.\scripts\check-routes.ps1
go test ./... -race -count=1
go test ./... -coverprofile coverage.out
go tool cover -func coverage.out
```

En Windows, `-race` requiere `CGO_ENABLED=1` y un compilador GCC disponible en `PATH`; el repositorio no versiona ese compilador. `coverage.out` está ignorado por Git. Para revisar un paquete concreto:

```powershell
go test ./internal/reportes -cover
go test ./internal/comprobantes/calc -cover
go test ./internal/tasas/calculator -cover
```

La fase se considera verde solo si la batería de CI termina con código 0, los umbrales de §15.5 se mantienen y el árbol de trabajo queda limpio después de eliminar artefactos temporales.

## 16. Runbook de incidentes

| Incidente | Primer control | Escalado |
|---|---|---|
| Evolution caída o 502 | Verificar `EVOLUTION_API_URL`, clave e instancia; ejecutar `evoprobe`. | Si el proveedor está caído, pausar envíos y comunicar; no reintentar en bucle. |
| Base lenta o `/ready` 503 | Revisar `checks.database.latency_ms`, pool y consultas activas. | Aplicar el plan de recuperación de PostgreSQL y validar backup antes de reiniciar. |
| JID inválido o placeholder | Confirmar el perfil y el valor `id_grupo`; no inventar un número. | Corregir en Perfiles y reenviar con una nueva clave de idempotencia. |
| Imagen rechazada | Revisar firma JPEG, MIME, dimensiones y tamaño. | Regenerar desde el canvas; no relajar límites sin análisis de memoria. |
| Doble envío | Revisar `Idempotency-Key`, `duplicate` y `messageId`. | Si se duplicó, registrar el incidente y evitar reintentos manuales. |
| Fuga de secreto | Rotar la clave y reiniciar; revisar `401`/`403`. | Invalidar la clave vieja, conservar evidencia y abrir incidente. |

El detalle de diagnóstico, mensajes y acciones está en §20.2; la respuesta específica a una fuga de clave está en §19.10.

## 17. Migración y compatibilidad

Glauk se incorpora sobre datos heredados sin una migración masiva ni destructiva. La regla de esta fase es **auditar primero, corregir con revisión humana y recién después abrir tráfico real**. El servicio viejo se mantiene disponible en modo lectura durante siete días y el esquema nuevo es aditivo: una columna nullable y sus índices.

### 17.1 Estado y contrato de compatibilidad

| Elemento | Estado en el cutover |
|---|---|
| Código | Un solo módulo Go y un solo binario; las fases 1–16 están ejecutadas y la release es `v1.0.0`. Los módulos legados quedan deprecados y no reciben mantenimiento. |
| Tag de fase | `v1.0.0` (y `f16-cleanup`); el tag histórico `f15-cutover` se conserva. |
| Rutas | Las rutas canónicas y sus alias están documentados en el Capítulo 11; los alias se retiran después de 30 días sin uso, no durante el cutover. |
| Base de datos | `internal/comprobantes/migrations/0001_impactos_soft_delete.sql` es la única migración de referencia. Es idempotente y no borra filas. |
| Migraciones automáticas | **No existen**. El binario no aplica DDL al arrancar; el operador ejecuta `psql` con backup verificado. |
| `monedas` | El codec acepta las tres formas y escribe el superset en la primera escritura de cada socio; no se reescribe masivamente en fase 1. |
| `id_grupo` | El placeholder `NOMBRE_JID` está prohibido; el default nuevo es vacío y Reportes rechaza JIDs inválidos antes de enviar. |
| `old/` y `pgxmock` | `old/` de los repositorios legados se conserva como referencia de rollback y no se despliega; el mantenimiento vive solo en `Core/`. `pgxmock` sigue mientras la suite no use exclusivamente PostgreSQL efímero. |

Índice de migraciones y comandos: [`migrations/README.md`](migrations/README.md).

### 17.2 Auditoría pre-cutover

El SQL [`scripts/audit-pre-cutover.sql`](scripts/audit-pre-cutover.sql) reúne las consultas obligatorias de §1.7.4, incluida la validación estricta de JID de §3.4, y añade la discrepancia de actividad de §1.7.3. Se ejecuta en una transacción `READ ONLY`; no contiene `INSERT`, `UPDATE`, `DELETE` ni DDL.

Ejecución directa:

```powershell
psql "$env:DATABASE_URL" -v ON_ERROR_STOP=1 -f .\scripts\audit-pre-cutover.sql
```

Reporte tabular con alerta y códigos de salida:

```powershell
.\scripts\audit-db.ps1 -Dsn $env:DATABASE_URL
.\scripts\audit-db.ps1 -Dsn $env:DATABASE_URL -Detailed -CsvPath .\audit-pre-cutover.csv
```

Si no hay `psql` o DSN, el script no inventa resultados: cae en **simulación / parseo seguro**, muestra qué valida cada bloque y sale con código 2. `-Simulate` fuerza ese modo y sale con 0. Códigos: `0` auditoría limpia o simulación explícita, `1` hallazgos que requieren acción humana, `2` auditoría no ejecutada por falta de herramienta/DSN, `3` error de ejecución. Un bloque con error no se considera aprobado.

| Bloque | Qué valida | Criterio de aprobación |
|---:|---|---|
| 01 | Duplicados de `UPPER(BTRIM(nombre))` en `perfiles_glaukov`. | Cero duplicados sin decisión documentada; el índice único debe poder crearse. |
| 02 | JIDs vacíos, placeholders `*_JID`, legado `@c.us` o formato inválido. | Cada fila corregida a mano y revalidada con el regex de §3.4; no se inventan números. |
| 03 | Formas `porcentaje`, `deposito` y `tipo` de `monedas`, y cuántas coexisten. | Cada conflicto resuelto por superset; `porcentaje` manda si los valores difieren. |
| 04 | Discrepancia entre `mostrar.tasas` y `activo`. | Cada fila revisada y aprobada; `mostrar.tasas` es canónico y `activo` solo apaga. Si la columna no existe, se ignora. |
| 05 | `tasas_glaukov.created_at` nulo o `id_tasa` fuera de `^T[0-9]{3}$`. | Lote corregido con backup antes de publicar. |
| 06 | `comprobantes_raw` sin fila equivalente en `comprobantes_liq`. | Huérfano liquidado o justificado; la evidencia cruda no se borra. |

El script `audit-db.ps1` ejecuta cada bloque dentro de `BEGIN TRANSACTION READ ONLY` y con `statement_timeout` acotado. `-Detailed` imprime las filas encontradas; `-CsvPath` guarda el resumen tabular para adjuntarlo a `DECISIONES.md`. Las filas y los JIDs son datos operativos: el reporte se comparte por canal restringido, no en tickets públicos.

### 17.3 Checklist de cutover

| Momento | Acción | Evidencia |
|---|---|---|
| D−7 | Congelar cambios de esquema, copiar producción a staging y ejecutar la auditoría completa. | Reporte CSV y `DECISIONES.md` con dueño de cada corrección. |
| D−7 | Corregir JIDs y resolver duplicados, conflictos de monedas, actividad y huérfanos. | Segunda auditoría sin hallazgos no aprobados. |
| D−7 | Probar con copia de producción y Evolution real usando `JID_GRUPO_PRUEBA`: una cartelera y un reporte con `messageId`. | Imagen, caption, destino y respuesta de Evolution verificados. |
| D−1 | Aplicar la migración idempotente en staging y luego en producción, con backup previo. | `migrations/README.md` y consultas de verificación de columna/índices. |
| D−1 | Definir ventana, TTL de DNS bajo y responsables de rollback. | Runbook y contactos. |
| D0 | Bajar **escrituras** de los servicios viejos; verificar que no quedan colas ni workers Node/Redis/BullMQ. | Servicios viejos en modo lectura. |
| D0 | Desplegar Glauk con el tag aprobado, verificar `/ready` 200 y ejecutar `scripts/smoke.ps1`. | Smoke verde y logs JSON sin secretos. |
| D0 | Cambiar DNS/proxy al nuevo servicio y mantener el anterior caliente. | Tráfico nuevo en Glauk y old service disponible. |
| D0 a D+72 h | Monitorear 5xx, latencia, envíos, duplicados, JIDs inválidos y los cuatro paneles. | Sin errores nuevos ni duplicados; métricas dentro de la línea base. |
| D+7 d | Confirmar estabilidad, archivar evidencia y recién entonces planificar la retirada de los servicios viejos. | Sign-off de operación y negocio. |

La aceptación del cutover es: **72 horas sin errores 5xx nuevos, sin envíos duplicados y con los cuatro paneles operativos**. Un hallazgo de auditoría no aprobado es un bloqueo, no una advertencia.

### 17.4 Rollback inmediato

El rollback de aplicación no requiere restaurar datos ni tocar el esquema:

1. Devolver DNS/proxy al servicio viejo, que se mantuvo caliente y en modo lectura.
2. Reactivar sus escrituras solo si el negocio lo aprueba; el esquema nuevo es compatible y la columna `eliminado_en` es nullable.
3. En Easypanel, volver al deployment anterior; en local, desplegar el tag anterior (`f14-docker` o la imagen previa) y no borrar el volumen.
4. Conservar el contenedor Glauk detenido con sus logs para diagnosticar; no ejecutar `docker compose down -v` ni borrar tablas.
5. Restaurar `glauk-pre.dump` **solo** si hubo una escritura incompatible, con aprobación explícita y el servicio detenido. La migración de fase 15 no es destructiva.

El objetivo es DNS en menos de 15 minutos, imagen anterior en menos de 10 minutos y restauración de base solo como último recurso. Un rollback de datos debe ser independiente del rollback de código; si no lo es, se replanifica.

### 17.5 Criterios para resolver hallazgos

- **Duplicados de nombre:** decidir con negocio cuál es el perfil canónico y corregir el nombre o fusionar la información; no borrar filas automáticamente. El resultado debe permitir el índice único por nombre canónico.
- **JIDs:** corregir `id_grupo` en Perfiles con una persona responsable. Un placeholder `NOMBRE_JID` se reemplaza por vacío o por un JID verificado; nunca se inventa un número. `@c.us` se normaliza al formato actual. Se vuelve a ejecutar el bloque 02 hasta que no queden inválidos no aprobados.
- **Monedas:** el superset es la forma de escritura. Si `porcentaje.deposito` y `deposito` difieren, gana `porcentaje` porque Tasas es el motor de cálculo; registrar el conflicto en `DECISIONES.md`. No se hace una migración masiva en fase 1: cada socio queda normalizado en su primera escritura.
- **Actividad:** `mostrar.tasas` manda. `activo` solo puede apagar; una fila donde `activo` intente encender un perfil apagado se revisa y se aprueba o se corrige. Si `activo` no existe, no se inventa.
- **Lotes:** `id_tasa` debe cumplir `^T[0-9]{3}$` y `created_at` no puede ser nulo. Cualquier corrección se hace con backup y se verifica con una lectura posterior.
- **Huérfanos de liquidación:** liquidar el comprobante o documentar por qué no aplica. No se elimina `comprobantes_raw`: es la evidencia de origen.

### 17.6 Lecciones del legado

- No confiar en defaults silenciosos: `monedas` tenía tres formas y `activo` podía contradecir a `mostrar.tasas`.
- No compartir JSONB sin un codec único: el superset se escribe en un solo dominio y los demás solo leen.
- No hardcodear claves ni JIDs; los secretos viven fuera de la imagen y los placeholders se rechazan.
- No ejecutar migraciones destructivas sin backup verificado y prueba de restauración en staging.
- No retirar los servicios viejos el mismo día: siete días de lectura y 72 horas de métricas son la red de seguridad.

## 18. Monitoreo, métricas y observabilidad

Glauk no incorpora Prometheus, OpenTelemetry ni un agente externo en la versión 1.0.0. La observabilidad se apoya en tres fuentes que ya forman parte del binario: las sondas HTTP, los logs estructurados de `slog` y la auditoría persistente de las tablas de negocio. Esta decisión mantiene la imagen distroless pequeña y evita una dependencia operativa que todavía no aporta valor con una sola réplica.

### 18.1 Sondas `/health` y `/ready`

| Sonda | Contrato | Código | Uso |
|---|---|---|---|
| `GET /health` | `status`, `service:"glauk"`, `checks.database.status`, `checks.database.latency_ms`, `checks.evolution.status`, `checks.evolution.instance`, `uptime_s` y `version`. | `200` si PostgreSQL responde; `503` con `status:"degraded"` y `checks.database.status:"error"` si falla. | Sonda operativa completa y healthcheck de Easypanel. |
| `GET /ready` | `status`, `service`, `checks.database`, `checks.evolution.status` (sin `instance`) y `uptime_s`. No incluye `version`. | `200` solo cuando PostgreSQL responde; `503` en caso contrario. | Decidir si el proxy puede enviar tráfico. |

Ambas rutas:

- Ejecutan el ping a PostgreSQL con un timeout máximo de **2 segundos** (`healthTimeout`).
- Devuelven `Cache-Control: no-store`; el resultado nunca se cachea.
- Exponen únicamente estado, latencia, modo de Evolution y, en `/health`, la versión; nunca DSN, usuario, contraseña, host interno ni rutas del sistema.
- Se registran antes de las rutas de dominio y son públicas: no requieren `X-API-Key` ni `X-Admin-Key`.

Ejemplos:

```powershell
Invoke-RestMethod http://localhost:3000/health | ConvertTo-Json -Depth 5
Invoke-RestMethod http://localhost:3000/ready  | ConvertTo-Json -Depth 5
```

Si `/health` devuelve 503:

1. Revisar `checks.database.status` y `latency_ms`. Un `error` confirma que el ping falló; una latencia cercana a 2000 ms indica saturación del pool o de PostgreSQL.
2. Verificar conectividad, credenciales y `DB_MAX_CONNS`. No reiniciar en bucle si la base está caída: el proceso puede estar sano pero degradado.
3. Confirmar que PostgreSQL acepta conexiones y que no se agotó el pool por consultas lentas.
4. Si solo `checks.evolution.status` es `simulated`, no es un fallo: `EVOLUTION_API_URL` está vacía y los envíos se simulan.
5. Después de corregir, esperar `/ready` 200 antes de reabrir tráfico.

### 18.2 Logs estructurados con `slog`

`LOG_FORMAT` selecciona el handler: `json` en producción y `text` en desarrollo; `auto` elige JSON cuando `APP_ENV=production` y texto en el resto. `LOG_LEVEL` acepta `debug`, `info`, `warn` y `error`. El logger se construye una sola vez en `cmd/server` y se inyecta; el dominio no usa `slog.Default()`.

Cada request produce una línea con nombre de evento `http_request` y los campos obligatorios de §2.8:

| Campo | Origen | Ejemplo |
|---|---|---|
| `request_id` | Cabecera `X-Request-Id` validada (UUID/ULID) o ID hexadecimal de 16 bytes generado por el servidor. | `9f2c1e7a4b8d4c1e` |
| `method` | Método HTTP. | `POST` |
| `path` | Ruta sin query string. | `/api/v1/reportes/imagen` |
| `status` | Código final, incluido el de un error controlado. | `502` |
| `duration_ms` | Duración total en milisegundos. | `1840` |
| `bytes` | Tamaño del cuerpo de respuesta. | `812345` |
| `tenant` | Tenant resuelto por host o path; vacío si no aplica. | `reportes` |

El nivel se deriva del estado: `2xx` → `info`, `4xx` → `warn`, `5xx` → `error`. El middleware **no** registra cuerpos, query strings, cabeceras, cookies, `Authorization`, `X-API-Key`, `X-Admin-Key`, base64, captions ni la API key de Evolution. Los errores 5xx se registran con `request_id`, método, ruta y causa; el cliente recibe un mensaje estable sin la causa interna.

El `request_id` se devuelve en la cabecera `X-Request-Id` y permite correlacionar la respuesta del panel con la línea de log exacta. Conservarlo es el primer paso de cualquier incidente.

### 18.3 Auditoría de eventos de negocio

La auditoría de negocio no depende de un servicio externo. Vive en las tablas que ya son fuente de verdad y se correlaciona con el `request_id` del log HTTP:

| Evento | Evidencia persistente | Qué permite reconstruir |
|---|---|---|
| Alta, edición, activación o plantilla de socio | `perfiles_glaukov` | Nombre, JID, `mostrar.tasas`, monedas y estado. |
| Lote de tasas | `tasas_glaukov` (`id_tasa`, `tasas`, `created_at`) | Valores publicados y momento de vigencia. |
| Cálculo o liquidación | `comprobantes_liq` y snapshot congelado | Tasas, polaridad, montos, herencia y lote asignado. |
| Borrado lógico de comprobante | `impactos_raw.eliminado_en` | Evidencia cruda conservada fuera del listado. |
| Cola y proyección | `cola_fb`, `comprobantes_fb`, `comprobantes_auditados_fb` | Estado del procesamiento y datos auditados. |
| Envío de reporte | Respuesta con `messageId` y `request_id`; no hay tabla propia de envíos en v1.0.0. | Correlación con Evolution y con el request HTTP. |

No se registra el base64, el caption completo, el cuerpo de un comprobante ni una API key. Si en el futuro se necesita retención histórica de envíos, se crea una tabla propia de Reportes (`reportes_envios`) con `request_id`, socio, JID hasheado, `message_id`, bytes, estado y fecha; Reportes no escribe en tablas de Comprobantes.

### 18.4 Métricas mínimas y alertas

Con una réplica, las métricas del proveedor, los logs y las sondas son suficientes:

| Métrica | Cómo obtenerla | Umbral de atención |
|---|---|---|
| Disponibilidad | `/ready` 200/503 y healthcheck de Easypanel. | Cualquier 503 sostenido más de 2 minutos. |
| Latencia HTTP | `duration_ms` de `http_request`. | p95 > 2 s en rutas de lectura o > 30 s en render/envío. |
| Errores 5xx | Logs filtrados por `status` ≥ 500. | Aumento sostenido sobre la línea base o un 5xx nuevo reproducible. |
| Latencia de PostgreSQL | `checks.database.latency_ms` y logs de error. | Cerca del timeout de 2 s. |
| Envíos | Respuestas `messageId`, errores `EVOLUTION_UNAVAILABLE`, logs de Reportes. | Fallos repetidos o ausencia de `messageId`. |
| Duplicados | `duplicate:true` en la respuesta de Reportes y repetición de `messageId`. | Cualquier duplicado real; revisar la clave de idempotencia. |
| Reinicios | Estado de Easypanel o `docker ps`. | Más de un reinicio no planificado por hora. |
| Tamaño de imagen | `docker image inspect glauk:local --format '{{.Size}}'`. | Crecimiento anómalo tras cambiar la base o dependencias. |
| Uso de alias | `path` de `http_request` con prefijos `/api/` no canónicos. | Aumento de rutas con `Deprecation` próximo al `Sunset`. |

Consultas útiles:

```powershell
docker compose logs -f glauk
docker compose logs glauk | Select-String '"status":5'
docker compose logs glauk | Select-String '"request_id":"9f2c1e7a4b8d4c1e"'
docker compose logs glauk | Select-String '"path":"/api/v1/reportes'
```

En Easypanel, filtrar por `status`, `path` o `request_id` en el visor. Para un incidente, guardar la línea completa y el `request_id`; no pegar payloads ni JIDs completos en tickets públicos. La retención de logs la define la plataforma; se recomienda conservar al menos 30 días en caliente y 90 días en archivo, con acceso restringido.

### 18.5 Diagnóstico rápido

| Señal | Interpretación | Acción |
|---|---|---|
| `/health` 503, `/ready` 503 | PostgreSQL no responde. | Revisar base, credenciales, pool y red. No reiniciar en bucle. |
| `/health` 200, `/ready` 200, Evolution `simulated` | Modo simulación activo. | Configurar `EVOLUTION_API_URL` y `EVOLUTION_API_KEY` si se espera envío real. |
| `status:502` en `/api/v1/reportes/*` | Evolution no responde o rechaza. | Revisar URL, clave, instancia y `evoprobe`; no reintentar a ciegas. |
| `status:401` en mutaciones | `SECURITY_API_KEY` configurada y cabecera ausente o incorrecta. | Enviar `X-API-Key`; rotar si se sospecha fuga. |
| `status:503` en admin | `ADMIN_API_KEY` vacía. | Configurar el secreto y reiniciar el servicio. |
| `status:404` en `/api/...` | Ruta desconocida; no es un fallback SPA. | Revisar el manifiesto de rutas y `check-routes.ps1`. |

## 19. Seguridad, hardening y conformidad

La seguridad de Glauk se apoya en un perímetro TLS, una superficie HTTP pequeña, secretos fuera de la imagen y un runtime mínimo. Esta sección describe el comportamiento implementado, las recomendaciones de despliegue y los límites que el operador debe conocer.

### 19.1 Modelo de amenazas y límites de confianza

| Amenaza (STRIDE) | Superficie | Control implementado |
|---|---|---|
| Spoofing | Admin y mutaciones. | `ADMIN_API_KEY` obligatoria para admin; `SECURITY_API_KEY` opcional para mutaciones; comparación en tiempo constante. |
| Tampering | Body, rutas y assets. | Validación de entrada, límites de body, rutas canónicas congeladas y path traversal guard. |
| Repudiation | Operaciones de negocio. | Auditoría en tablas, `request_id`, logs estructurados y `messageId` de Evolution. |
| Information disclosure | Logs, errores y health. | Redacción, envelopes estables, health sin DSN y headers restrictivos. |
| Denial of service | Reportes, imágenes y admin. | BodyLimit, límites de imagen, rate limit, timeouts y una réplica controlada. |
| Elevation of privilege | Admin y secretos. | Clave admin fuera del código, sin default, comparación constante y `503` si falta. |

Límites de confianza: el navegador y el proxy TLS están fuera del contenedor; PostgreSQL y Evolution son dependencias de red; `PUBLIC_DIR` es contenido controlado por el despliegue y no un directorio de uploads de usuario. Las URLs de Evolution y HOO son configuración del operador, no entrada arbitraria del usuario.

### 19.2 Autenticación y autorización

| Superficie | Sin clave | Con `SECURITY_API_KEY` | Con `ADMIN_API_KEY` |
|---|---|---|---|
| `GET` de lectura | Abierta | Abierta | Abierta |
| Mutaciones de dominio | Abiertas por compatibilidad | Exige `X-API-Key` | No aplica |
| Admin | `503 ADMIN_DISABLED` | No aplica | Exige `X-Admin-Key` o cookie `admin_key` |
| `/health`, `/ready` | Abiertos | Abiertos | Abiertos |
| Estáticos y SPA | Abiertos | Abiertos | Abiertos |

Reglas:

- `VerifyAdmin` usa `crypto/subtle.ConstantTimeCompare`. Si la clave configurada está vacía, devuelve `false`; `RequireAdminKey` responde `503 ADMIN_DISABLED` en vez de aceptar cualquier valor.
- `RequireAPIKey` compara `X-API-Key` en tiempo constante cuando `SECURITY_API_KEY` está definida. Si está vacía, conserva el comportamiento histórico de mutaciones abiertas; en producción se recomienda definirla y migrar los paneles para enviarla.
- La consola admin acepta `X-Admin-Key` o la cookie `admin_key`. Preferir la cabecera y usarla solo sobre HTTPS; no poner la clave en query strings ni en logs.
- Las claves deben ser secretos de alta entropía, rotables y distintos entre entornos. Nunca se hardcodean en el repositorio ni se reutilizan entre staging y producción.
- `AUTHENTICATION_API_KEY` es un alias legado de `EVOLUTION_API_KEY`; si ambas están definidas y difieren, el arranque falla. No se debe usar como mecanismo de autenticación de Glauk.

### 19.3 Cabeceras, CSP y CORS

Todas las respuestas reciben:

```text
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Permissions-Policy: camera=(), microphone=(), geolocation=()
Cross-Origin-Opener-Policy: same-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains   (solo HTTPS)
```

Los HTML de los paneles reciben además:

```text
default-src 'self';
img-src 'self' data: blob: https:;
script-src 'self' 'unsafe-eval';
style-src 'self' 'unsafe-inline';
font-src 'self' data:;
connect-src 'self';
frame-src https://tasas.jairokov.com https://perfiles.jairokov.com;
frame-ancestors 'none';
base-uri 'self';
form-action 'self'
```

No hay CDNs ni scripts inline. `script-src 'unsafe-eval'` es necesario porque Alpine.js evalúa expresiones declarativas y `html2canvas` las usa internamente. `img-src https:` permite los comprobantes alojados fuera del origen; `frame-src` se limita a los visores de Tasas/Perfiles. `style-src 'unsafe-inline'` se conserva porque Alpine y Tailwind aplican estilos calculados. Cualquier cambio de CSP debe probar la UI Glauk2 completa y los shells de compatibilidad. La API no sirve HTML y no necesita CSP.

CORS:

- `CORS_ALLOWED_ORIGINS` es una lista CSV. El default `*` es solo para desarrollo/staging; en `production` el arranque rechaza `*`.
- Se permiten `GET,POST,PUT,PATCH,DELETE,OPTIONS` y las cabeceras `Origin,Content-Type,Accept,X-Admin-Key,X-API-Key,X-Request-Id,Idempotency-Key`.
- Se expone `X-Request-Id` para correlación.
- No se habilita `Access-Control-Allow-Credentials` con `*`; los orígenes de producción deben ser exactos.
- Los preflight `OPTIONS` no ejecutan lógica de dominio.

### 19.4 Path traversal, estáticos y aislamiento de API

El orden de registro es parte del contrato: rutas canónicas → alias → `/health` → `/api` 404 → estáticos → SPA → 404 global. Esto garantiza que un endpoint API desconocido **nunca** caiga al `index.html`.

`RegisterStatic` aplica `traversalGuard` antes de servir archivos:

1. Toma `OriginalURL` y `Path`, elimina la query y decodifica la URL.
2. Rechaza NUL, barras invertidas y cualquier segmento exactamente igual a `..`.
3. Normaliza con `path.Clean` y construye la ruta con `filepath.FromSlash`.
4. Verifica con `filepath.Rel` que el resultado permanezca dentro de `PUBLIC_DIR`; si no, responde `403 FORBIDDEN`.
5. Solo si el archivo existe y no es directorio lo sirve; si el tenant lo permite, hace fallback a su `index.html` para rutas SPA sin extensión.

Los assets compartidos (`/shared/*`, `/robots.txt`) se sirven desde la raíz. Los paneles se aíslan por tenant/host o por path según `HOST_ROUTING_MODE`; un tenant desconocido no se resuelve a otro panel. No hay uploads a disco: las imágenes de Reportes llegan en base64 y se validan en memoria.

### 19.5 Entradas, límites y abuso

| Vector | Defensa |
|---|---|
| Body gigante | `BODY_LIMIT_BYTES=20971520` (20 MiB) en Fiber; rechazo temprano por `Content-Length`. |
| Base64 bomba | Decodificación acotada por `REPORT_IMAGE_MAX_BYTES=10485760` y `image.DecodeConfig` sin descomprimir el bitmap completo. |
| Imagen no JPEG | Firma binaria `FF D8 FF`, MIME `image/jpeg` y decodificador JPEG; otro formato → `IMAGEN_FORMATO`. |
| Dimensiones abusivas | `REPORT_IMAGE_MIN_DIMENSION=200`, `MAX_DIMENSION=4096` y `MAX_PIXELS=16000000`. |
| Fuerza bruta admin | Clave de alta entropía, comparación constante, TLS y rate limit del proxy. |
| Envío masivo | `REPORT_RATE_LIMIT_PER_MINUTE=60`, ráfaga 10, token bucket por IP; `429 RATE_LIMITED` con `Retry-After`. |
| Doble envío | `Idempotency-Key` o SHA-256 del contenido, reserva antes de enviar, TTL 10 min y límite de 1000 entradas. |
| Path traversal | Guard de segmentos, `filepath.Clean`, `filepath.Rel` y `403`. |
| XSS en caption | El caption se envía a WhatsApp, no se renderiza en HTML del servidor; el frontend usa `textContent`/escape. |
| SSRF | `EVOLUTION_API_URL` y `HOO_GLOBAL_RATES_URL` son configuración del operador, no URLs de usuario. |
| Fuga de secretos | Logger sin cuerpos/cabeceras, `MaskKey`, `MaskJID`, `MaskPhone` y secretos del orquestador. |
| Inyección SQL | `pgx` con parámetros; Reportes no ejecuta SQL y los adaptadores están aislados. |

La idempotencia y el rate limit viven en memoria del proceso. Con más de una réplica, un doble clic puede caer en instancias distintas; por eso el despliegue objetivo sigue siendo **una réplica**. Escalar exige mover ambos estados a un almacén compartido y ajustar la configuración del proxy para no agrupar a todos los clientes bajo su IP.

### 19.6 Secretos y rotación

| Secreto | Uso | Rotación |
|---|---|---|
| `ADMIN_API_KEY` | Consola administrativa. | Crear nueva clave, actualizar el secreto, reiniciar, verificar 200 con la nueva y retirar la anterior. |
| `SECURITY_API_KEY` | Mutaciones cuando está habilitada. | Igual que admin; actualizar paneles y automatizaciones antes de retirar la anterior. |
| `EVOLUTION_API_KEY` | Envío de imágenes/texto. | Rotar en Evolution y en el servicio; verificar un envío simulado y uno real de prueba. |
| `DATABASE_URL` / `DB_PASSWORD` | PostgreSQL. | Rotar en la base y en el secreto; probar `/ready` antes de reabrir tráfico. |

Reglas:

- `.env` no se versiona ni se copia a la imagen; `.env.example` solo contiene placeholders.
- En producción, las claves se cargan como secretos de Easypanel. No se pasan por argumentos de línea de comandos ni se escriben en logs.
- `LOG_FORMAT=json` y `LOG_LEVEL=info` son el default operativo. Subir a `debug` solo de forma temporal y con revisión de lo que se registra.
- Ante una fuga: rotar el secreto, reiniciar el servicio, invalidar la clave vieja en el proveedor, revisar logs por `401`/`403`, revisar accesos a la base y documentar el incidente. No borrar evidencia.

### 19.7 Distroless y ejecución nonroot

La imagen final es `gcr.io/distroless/static-debian12:nonroot`:

- No tiene shell, gestor de paquetes, compilador, `curl` ni `wget`.
- El único ejecutable es `/app/glauk`; los estáticos viven en `/app/public`.
- El build usa `CGO_ENABLED=0`, `-trimpath` y `-ldflags="-s -w"`; el binario es estático y no depende de libc.
- El proceso corre como `nonroot:nonroot` y no requiere privilegios.
- No hay volúmenes persistentes: el binario es stateless y la base es un servicio externo.
- `.dockerignore` excluye `.git`, `.env`, binarios, logs, `coverage.out` y `docs/`.
- No se define `HEALTHCHECK` dentro de la imagen porque no hay cliente HTTP; el healthcheck lo hace Easypanel contra `/health`.
- Siempre que la plataforma lo permita, montar el rootfs como read-only y limitar capabilities; la aplicación no escribe en disco.

No se puede entrar con `docker exec ... sh`; el diagnóstico se hace por logs, `/health`, `/ready`, estado del contenedor y el deployment anterior.

### 19.8 Conformidad y protección de datos

Glauk trata datos operativos de socios, JIDs, comprobantes e imágenes. Las medidas implementadas son: minimización en logs, acceso restringido a secretos, TLS en el proxy, autenticación admin, envelopes sin causas internas, borrado lógico de impactos y backups con acceso controlado.

Recomendaciones de cumplimiento:

- Mantener una base legal y una política de retención para comprobantes, imágenes y logs; Glauk no decide plazos legales por sí solo.
- Restringir el acceso a la base, a los backups y al visor de logs por rol.
- Usar HTTPS obligatorio y HSTS en producción; no exponer PostgreSQL ni el puerto interno a Internet.
- Documentar cada acceso administrativo y conservar el `request_id` de las operaciones relevantes.
- Probar la restauración de backups, no solo su creación.
- Revisar dependencias y la imagen base ante cada release; el runtime distroless reduce la superficie, pero no sustituye el escaneo de vulnerabilidades.

Esta sección describe controles técnicos; no constituye una certificación legal ni una auditoría de cumplimiento externa.

### 19.9 Checklist de despliegue seguro

- [ ] `APP_ENV=production`, TLS terminado en el proxy y HSTS activo.
- [ ] `DATABASE_URL` con `sslmode=require` o superior; `DB_ALLOW_INSECURE` solo con excepción documentada.
- [ ] `ADMIN_API_KEY` definida como secreto y distinta de la de staging.
- [ ] `SECURITY_API_KEY` definida si los paneles ya envían `X-API-Key`.
- [ ] `CORS_ALLOWED_ORIGINS` con los hosts exactos, nunca `*`.
- [ ] `EVOLUTION_API_URL`, `EVOLUTION_API_KEY` e instancia verificadas con un grupo de prueba.
- [ ] `LOG_FORMAT=json`, `LOG_LEVEL=info`, sin `.env` dentro de la imagen.
- [ ] Una réplica; idempotencia y rate limit en memoria documentados.
- [ ] `/health` y `/ready` responden 200; smoke test verde.
- [ ] Backup reciente y restauración de prueba validada.
- [ ] Rootfs read-only/capabilities limitadas si la plataforma lo permite.

### 19.10 Respuesta a fuga de clave

1. Aislar el servicio o rotar el secreto afectado; no esperar a que termine la investigación.
2. Invalidar la clave vieja en el proveedor correspondiente (Evolution, base o proxy).
3. Reiniciar con la nueva clave y verificar `/ready`, admin y un envío de prueba.
4. Buscar en logs `401`, `403`, rutas admin y `request_id` sospechosos; revisar accesos a la base.
5. Conservar evidencia y abrir el incidente con impacto, detección, mitigación y causa raíz.
6. Si la clave estuvo en un repositorio o imagen, rotarla aunque el commit se haya revertido.

## 20. Apéndices

### 20.1 Glosario de términos financieros y operativos

| Término | Definición |
|---|---|
| **Socio** | Entidad o persona con un perfil, monedas, porcentajes y un JID de WhatsApp. Puede estar activo o apagado. |
| **Perfil** | Fila de `perfiles_glaukov` que representa a un socio y su configuración. |
| **Perfil GENERAL** | Perfil matriz usado cuando no se especifica un socio; no tiene JID propio. |
| **Lote** | Conjunto de tasas identificado por `id_tasa` (formato `T###`) y `created_at`. |
| **Tasa vigente** | Lote más reciente por `created_at`/`id` que se usa para calcular o publicar. |
| **Histórico** | Lotes anteriores que se conservan para auditoría y reportes. |
| **Cartelera** | Imagen PNG que resume tasas de uno o más socios; se genera en el servidor de Tasas. |
| **Snapshot congelado** | Copia de tasas y configuración fijada al liquidar un comprobante; no cambia si luego se publica otro lote. |
| **D (depósito)** | Operación en la que el cliente entrega moneda; la polaridad puede sumar o restar según el contrato. |
| **A (abono)** | Operación de abono/pago; tiene regla de polaridad propia y puede heredar configuración de FUNDDA. |
| **P (pago)** | Tipo de operación de pago que usa `polar_P` y el porcentaje de pago. |
| **Polaridad** | Signo (`+`/`-`) que se aplica a un lado de la operación; el frontend no debe recalcularlo. |
| **Porcentaje de depósito** | Margen aplicado al lado de depósito; en el superset vive en `porcentaje.deposito`. |
| **Porcentaje de pago** | Margen aplicado al lado de pago; en el superset vive en `porcentaje.pago`. |
| **Spread** | Diferencia entre tasas de compra/venta o margen aplicado en la liquidación. |
| **Triangulación** | Cálculo de una tasa cruzada usando una moneda base cuando no hay par directo. |
| **Moneda base** | Divisa de referencia del socio; por defecto `USDT` en los puntos que lo requieren. |
| **HOO** | Origen externo de tasas globales (`HOO_GLOBAL_RATES_URL`) con timeout y fallback. |
| **Comprobante** | Evidencia cruda de una operación recibida desde WhatsApp, almacenada en `comprobantes_raw`. |
| **Liquidación** | Resultado financiero calculado y congelado en `comprobantes_liq`. |
| **Impacto** | Registro de la evidencia cruda de WhatsApp en `impactos_raw`; el borrado es lógico. |
| **Hash largo / hash corto** | Identificadores del comprobante usados para deduplicación y enlaces. |
| **JID** | Identificador de WhatsApp (`numero@s.whatsapp.net`, grupo `...@g.us` o `...@lid`). |
| **Instancia** | Nombre de la instancia de Evolution que envía los mensajes; por defecto `Glauk`. |
| **Tenant** | Superficie/panel resuelto por host o path: `tasas`, `perfiles`, `comprobantes` o `reportes`. |
| **Envelope** | Sobre JSON de respuesta con `success`, `data` o `error`, `code` y `requestId`. |
| **Puerto (hexagonal)** | Interfaz que define lo que Reportes necesita; los adaptadores conectan con otros dominios. |
| **Alias** | Ruta histórica que conserva compatibilidad y responde con `Deprecation`/`Sunset`. |
| **Canónico** | Ruta `/api/v1/*` que es la fuente de verdad y el sucesor de los alias. |
| **Superset** | Forma de `monedas` que contiene las claves de los tres consumidores y se escribe de forma canónica. |
| **Modo prueba** | Configuración que envía a `JID_GRUPO_PRUEBA` en lugar de a los socios reales. |

### 20.2 Troubleshooting operativo y FAQ

| Síntoma | Diagnóstico | Acción |
|---|---|---|
| `/health` 503 | `checks.database.status:"error"` o latencia alta. | Revisar PostgreSQL, credenciales, pool y red; no reiniciar en bucle. |
| `/ready` 503 con `/health` 200 | El ping de base falla en la sonda de tráfico. | Revisar el mismo pool y el proxy; esperar 200 antes de reabrir. |
| `/health` 200 y Evolution `simulated` | No hay `EVOLUTION_API_URL`. | Configurar URL/clave si se espera envío real. |
| Envío 502 `EVOLUTION_UNAVAILABLE` | Evolution caída, URL/clave/instancia incorrecta o timeout. | Probar `evoprobe`; revisar `EVOLUTION_*`; no reintentar a ciegas. |
| `JID_INVALIDO` | `id_grupo` no cumple el regex o quedó un placeholder. | Corregir en Perfiles; usar un JID verificado; nunca inventar un número. |
| `JID_NO_DISPONIBLE` | Perfil sin JID. | Completar el perfil o usar `JID_GRUPO_PRUEBA` en modo prueba. |
| `IMAGEN_FORMATO` | Base64 no es JPEG o falta la firma. | Regenerar con canvas; verificar MIME y prefijo data URL. |
| `IMAGEN_MUY_GRANDE` / `PAYLOAD_TOO_LARGE` | Supera 10 MiB de imagen o 20 MiB de body. | Reducir calidad/tamaño; no subir límites sin revisar memoria. |
| `IMAGEN_DIMENSIONES` | Fuera de 200–4096 px o >16 MP. | Ajustar el canvas a 1080×1350. |
| `429 RATE_LIMITED` | Más de 60 envíos/minuto o ráfaga >10. | Esperar `Retry-After`; revisar automatizaciones. |
| `409 CONFLICT` / `ENVIO_EN_CURSO` | La misma clave de idempotencia está en curso. | Reutilizar la clave y esperar; no generar una nueva para el mismo envío. |
| Respuesta `duplicate:true` | El envío ya se había completado. | No reenviar; verificar `messageId`. |
| Panel en blanco | Asset 404, `PUBLIC_DIR` incorrecto, host/path mode o CSP. | Revisar Network, `PUBLIC_DIR`, tenant y CSP. |
| Alias 404 inesperado | Alias no registrado o ruta retirada. | Ejecutar `check-routes.ps1`; revisar el orden de registro. |
| Admin 503 | `ADMIN_API_KEY` vacía. | Configurar el secreto y reiniciar. |
| Admin 401 | `X-Admin-Key` o cookie incorrecta. | Verificar la clave activa; rotar si hay duda. |
| `docker exec ... sh` falla | La imagen es distroless. | Usar logs, `/health`, `/ready` y el estado del contenedor. |
| Migración falla | Permisos, conexión o migración ya aplicada. | Verificar `psql`, backup, `information_schema` e índices; no editar datos a mano sin aprobación. |
| Backup no restaura | Dump incompleto o versión incompatible. | Probar en staging con la misma versión; documentar el resultado. |

Preguntas frecuentes:

**¿Por qué no hay Prometheus?** Porque la operación objetivo es un solo contenedor y una sola réplica; los logs, las sondas y la base cubren las métricas mínimas. Añadir Prometheus ahora duplicaría infraestructura sin mejorar la detección.

**¿Por qué el reporte se genera en el navegador?** Para evitar Chromium/Puppeteer en el servidor, reducir la imagen y mantener el render determinista. El backend valida el JPEG y lo envía.

**¿Puedo escalar a dos réplicas?** No con la idempotencia y el rate limit actuales en memoria. Primero hay que moverlos a un almacén compartido y ajustar el límite global.

**¿Qué pasa si cambio `monedas` a mano?** El codec tolera las formas históricas, pero la escritura canónica es el superset. Una edición manual puede perder claves; usar la API de Perfiles y auditar el resultado.

**¿Los alias son permanentes?** No. Tienen `Deprecation` y `Sunset` (31/12/2027); se retiran tras 30 días sin uso y nunca durante el cutover.

**¿Dónde está el changelog?** En [`CHANGELOG.md`](CHANGELOG.md), con la versión `1.0.0` y el detalle de compatibilidad, migraciones y seguridad.

### 20.3 Apéndices técnicos

#### 20.3.1 Configuración y secretos

La plantilla completa es [`.env.example`](.env.example). Las variables sensibles (`DATABASE_URL`, `DB_PASSWORD`, `EVOLUTION_API_KEY`, `ADMIN_API_KEY`, `SECURITY_API_KEY`) se cargan como secretos del orquestador; `.env` es solo para desarrollo local y nunca se copia a la imagen. La matriz completa, precedencia y validaciones están en el Capítulo 5.

#### 20.3.2 Catálogo de tablas

| Tabla | Dueño | Uso |
|---|---|---|
| `perfiles_glaukov` | Perfiles | Socios, `monedas`, `mostrar`, actividad y JID. |
| `tasas_glaukov` | Tasas | Lotes y valores de tasas. |
| `comprobantes_raw` | Comprobantes | Evidencia cruda de comprobantes. |
| `comprobantes_liq` | Comprobantes | Snapshot financiero congelado. |
| `impactos_raw` | Comprobantes | Evidencia de WhatsApp con borrado lógico. |
| `cola_fb`, `comprobantes_fb`, `comprobantes_auditados_fb` | Comprobantes | Cola y proyecciones de auditoría. |

`reportes` no tiene tablas propias y no ejecuta SQL; consume puertos de Perfiles y Comprobantes. El detalle de columnas, JSONB, índices y migraciones está en el Capítulo 12 y en [`migrations/README.md`](migrations/README.md).

#### 20.3.3 Rutas, contratos y OpenAPI

- Matriz canónica y alias: Capítulo 11.
- Manifiesto OpenAPI: [`docs/openapi.yaml`](docs/openapi.yaml).
- Verificación de superficie: `.\scripts\check-routes.ps1`.
- Una ruta API desconocida responde `404 NOT_FOUND`; nunca cae al SPA.

#### 20.3.4 Códigos de error frecuentes

| Código | HTTP | Significado |
|---|---:|---|
| `INVALID_JSON` / `VALIDATION_ERROR` | 400 | Cuerpo o campo inválido. |
| `DESTINO_REQUERIDO`, `IMAGEN_REQUERIDA`, `TEXTO_REQUERIDO` | 400 | Falta un dato obligatorio de Reportes. |
| `UNAUTHORIZED` | 401 | Clave admin/API ausente o inválida. |
| `FORBIDDEN` | 403 | Ruta fuera de `PUBLIC_DIR` o tenant no permitido. |
| `SOCIO_NO_ENCONTRADO`, `NOT_FOUND` | 404 | Recurso o ruta inexistente. |
| `CONFLICT`, `ENVIO_EN_CURSO` | 409 | Envío duplicado en curso. |
| `IMAGEN_MUY_GRANDE`, `PAYLOAD_TOO_LARGE` | 413 | Límite de imagen o body. |
| `IMAGEN_FORMATO` | 415 | No es un JPEG válido. |
| `IMAGEN_DIMENSIONES`, `JID_NO_DISPONIBLE`, `JID_INVALIDO` | 422 | Regla de negocio de Reportes. |
| `RATE_LIMITED` | 429 | Límite de envíos. |
| `INTERNAL_ERROR` | 500 | Error no controlado. |
| `EVOLUTION_UNAVAILABLE` | 502 | Evolution caída o rechazo. |
| `ADMIN_DISABLED` | 503 | Falta `ADMIN_API_KEY`. |
| `TIMEOUT` | 504 | Timeout de contexto o dependencia. |

#### 20.3.5 Changelog, auditoría y evidencia

- Versión y cambios: [`CHANGELOG.md`](CHANGELOG.md).
- Migración idempotente: [`internal/comprobantes/migrations/0001_impactos_soft_delete.sql`](internal/comprobantes/migrations/0001_impactos_soft_delete.sql).
- Auditoría pre-cutover: [`scripts/audit-pre-cutover.sql`](scripts/audit-pre-cutover.sql) y [`scripts/audit-db.ps1`](scripts/audit-db.ps1).
- Smoke HTTP: [`scripts/smoke.ps1`](scripts/smoke.ps1).
- Documento normativo: `../PLAN_CONSOLIDACION_GLAUK.md`.

#### 20.3.6 Trazabilidad de hallazgos H-01…H-14

| ID | Hallazgo | Resolución | Evidencia |
|---|---|---|---|
| H-01 | Tres formas de `monedas`. | Codec superset y escritura canónica. | `monedas_test.go`. |
| H-02 | Actividad divergente. | `mostrar.tasas` canónico; `activo` solo apaga. | `service_test.go`. |
| H-03 | JID placeholder. | Validación estricta, default vacío y auditoría. | `target_test.go`. |
| H-04 | `pgx` dual. | Pin `v5.10.0`. | `go.mod`. |
| H-05 | Go dual. | `go 1.27.2`. | `go.mod` y Dockerfile. |
| H-06 | Clave admin hardcodeada. | `ADMIN_API_KEY` + tiempo constante. | `admin_test.go`. |
| H-07 | Envelopes distintos. | Envelope canónico con compatibilidad. | `response_test.go`. |
| H-08 | Límites dispares. | Body 20 MiB, imagen 10 MiB, Evolution 25 s. | `config_test.go`. |
| H-09 | Colisión `/api/reportes`. | Listado legacy + dominio en `/api/v1/reportes`. | `TestNoDuplicateRoutes`. |
| H-10 | Rutas faltantes. | Matriz de alias y adaptadores. | Contract tests. |
| H-11 | `activo` fuera de decisión. | H-02 + auditoría de conjuntos. | Test de publicación. |
| H-12 | Sin autenticación en mutaciones. | `SECURITY_API_KEY` opcional y admin obligatorio. | Tests 401/503. |
| H-13 | Archivos gigantes. | Shell + módulos; presupuesto de líneas. | `check-file-length.ps1`. |
| H-14 | `BodyLimit` ausente. | `BODY_LIMIT_BYTES` explícito. | `config_test.go`. |

### 20.4 ADRs

Las seis decisiones estructurales se resumen aquí. El contexto ejecutivo y las alternativas descartadas están en el plan de consolidación, §0.1.

| ADR | Decisión | Alternativas | Consecuencia |
|---|---|---|---|
| ADR-0001 | Un módulo Go y un binario con cuatro dominios aislados. | Cuatro microservicios. | Menos coste operativo, un pool y un despliegue; el aislamiento se garantiza con paquetes y tests. |
| ADR-0002 | `pgx v5.10.0` y `pgxmock` mientras la suite no migre a PostgreSQL efímero. | Actualizar a v5.11 y romper mocks. | Build reproducible; la retirada de `pgxmock` queda como deuda explícita. |
| ADR-0003 | Reportes hexagonal, sin SQL ni pgx. | Reportes con repositorio propio. | El dominio es testeable y no conoce la base; los adaptadores concentran la integración. |
| ADR-0004 | Canvas en navegador; el backend valida JPEG. | Chromium/Puppeteer en el servidor. | Imagen distroless pequeña, sin navegador; el render depende del navegador del panel. |
| ADR-0005 | Codec superset de `monedas` y escritura canónica. | Migración masiva o un solo formato. | Compatibilidad con los tres consumidores sin reescritura destructiva. |
| ADR-0006 | Rutas canónicas `/api/v1/*` + 64 alias con `Deprecation`/`Sunset`. | Romper rutas legacy de inmediato. | Los paneles siguen funcionando; la deuda de alias tiene fecha y métrica de uso. |
