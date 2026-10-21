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

> Fases 5 y 11 implementadas: `internal/platform/httpx` resuelve host/prefijo, sirve assets con caché diferenciada y aplica el SPA fallback. Los cuatro paneles usan shells mínimos, módulos ES atómicos y assets locales; el render de reportes ocurre en el navegador.

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

### 10.6 Estructura modular (Fase 11)

```text
public/
├── index.html                  lanzador neutro (DEFAULT_TENANT=none)
├── robots.txt                  Disallow: /api/
├── shared/
│   ├── brand/                  logo FUNDABLOCK y favicon local
│   ├── css/                    base.css, components.css, print.css
│   ├── js/                     http, dom, format, toast, auth, reporte-canvas
│   └── vendor/                 html2canvas 1.4.1 (MIT, copia local)
├── tasas/                      shell, manifest.json, sw.js, js/{main,api,views,components}
├── perfiles/                   shell, manifest.json, sw.js, js/{main,api,views}
├── comprobantes/               shell, manifest.json, sw.js, js/{main,api,views,components}
└── reportes/                   shell, manifest.json, sw.js, js/{main,api,views,components}
```

Cada shell contiene solo estructura y puntos de montaje. La navegación, el estado y las llamadas viven en módulos ES; los componentes reutilizables se importan con rutas relativas. Los límites son 180 líneas por HTML (120 para el lanzador), 250 por módulo JS/CSS y exclusión total de `shared/vendor/` del presupuesto. Los formularios y acciones se enlazan con `addEventListener`; no se usan scripts inline.

### 10.7 Política de assets y CSP

- Todos los assets de un tenant se referencian de forma relativa: `../shared/css/base.css`, `./js/main.js`, `../shared/vendor/html2canvas.min.js`. La misma shell funciona en modo host (`/`) y en modo path (`/tasas/`, `/perfiles/`, `/comprobantes/`, `/reportes/`).
- Las llamadas de negocio siguen siendo absolutas y canónicas: `/api/v1/tasas/*`, `/api/v1/perfiles/*`, `/api/v1/comprobantes/*`, `/api/v1/reportes/*` y `/api/v1/admin/*`.
- No hay dependencias de CDN en tiempo de ejecución. Las fuentes, el logo, el favicon y `html2canvas` se sirven desde `public/shared/`. El vendor se valida por versión y licencia, no por longitud.
- El lanzador `public/index.html` solo se sirve en `/` cuando `DEFAULT_TENANT=none`; enlaza a los cuatro portales sin JavaScript.
- `public/robots.txt` bloquea explícitamente `/api/`.

### 10.8 Service workers y aislamiento

Cada SPA registra `../sw.js` resolviendo su URL con `import.meta.url` y declara el scope del tenant:

```js
const script = new URL('../sw.js', import.meta.url);
const scope = new URL('../', import.meta.url).pathname;
navigator.serviceWorker.register(script, { scope });
```

En modo path el scope es `/tasas/`, `/perfiles/`, `/comprobantes/` o `/reportes/`; en modo host es `/`. El handler `fetch` solo atiende `GET` del mismo origen y sale inmediatamente para `/api` y `/api/*`: las mutaciones, la sesión y los envíos de reportes nunca se cachean ni se sirven desde una copia offline. El shell y los assets estáticos se precargan; una navegación sin red usa el `index.html` del tenant.

### 10.9 Pipeline de canvas

`public/shared/js/reporte-canvas.js` implementa `generarReporteJPEG` con el contrato de §3.6.1:

| Regla | Implementación |
|---|---|
| Lienzo lógico | `1080 × 1350` px |
| Escala física | `Math.min(devicePixelRatio, 2)` |
| Fondo | `#FFFFFF`, sin transparencia |
| Formato | `canvas.toBlob('image/jpeg', quality)` |
| Calidad inicial | `0.90` |
| Límite de tamaño | `8 * 1024 * 1024` bytes |
| Reducción | baja de `0.05` hasta `0.60`; si aún excede, error visible |
| Retorno | `{ blob, base64 }` |

El panel construye el reporte en un `article.reporte-sheet` fuera de pantalla, espera `document.fonts.ready`, prepara las imágenes locales y rasteriza con el `html2canvas` vendorizado. El backend recibe únicamente el JPEG en base64; nunca HTML. Los envíos usan `Idempotency-Key` generado en el navegador y conservado mientras el reintento no confirme éxito.

### 10.10 Verificación

```powershell
$env:CGO_ENABLED = "1"
go test ./internal/platform/httpx -race -v

Get-ChildItem ".\public" -Recurse -Include *.html,*.js,*.css |
  Where-Object { $_.FullName -notmatch '\\vendor\\' } |
  ForEach-Object {
    $n = (Get-Content $_.FullName).Count
    $limit = if ($_.Extension -eq '.html') { 180 } else { 250 }
    if ($n -gt $limit) { Write-Error "$($_.FullName): $n > $limit" }
  }
```

Cubre host routing, path routing, prioridad del host, `/api/*` en JSON, traversal con `403`, asset inexistente con `404`, SPA fallback, las cuatro políticas de caché y el presupuesto de líneas de los frontends. La auditoría del lanzador usa el mismo script con límite `120` para `public/index.html`.

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
