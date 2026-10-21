# Changelog

Todos los cambios notables de Glauk se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el proyecto usa versionado semántico. Las fechas están en UTC.

## [Unreleased]

## [1.0.0] - 2026-10-21

Primera versión de producción del módulo unificado. Consolida **Tasas**, **Perfiles**, **Comprobantes** y **Reportes** en un solo binario Go, un solo contenedor y una sola base de datos, sin microservicios, Redis, BullMQ ni Chromium.

### Added

- **Consolidación de los cuatro dominios:** Tasas (cálculo, render y publicación), Perfiles (directorio, actividad y plantilla), Comprobantes (auditoría, liquidación y administración) y Reportes (canvas, validación, idempotencia y envío por WhatsApp).
- **Motor de despliegue distroless:** imagen final `gcr.io/distroless/static-debian12:nonroot`, sin shell, sin compiladores y con el binario estático como único ejecutable.
- **Codec superset de `monedas`:** lectura compatible con las formas históricas de Tasas, Perfiles y Comprobantes; escritura canónica del superset en la primera escritura de cada socio.
- **Reportes con canvas en navegador:** el panel genera el JPEG localmente (1080×1350, calidad 0.90); el backend valida firma, MIME, tamaño, dimensiones y píxeles antes de enviar. No se renderiza HTML ni se ejecuta Chromium en el servidor.
- **Aislamiento hexagonal de Reportes:** el dominio depende de puertos, no de pgx, SQL, Fiber ni otros dominios; los adaptadores viven en `internal/reportes/adapters`.
- **Auditoría pre-cutover:** `scripts/audit-pre-cutover.sql` y `scripts/audit-db.ps1` con seis bloques de solo lectura para duplicados, JIDs, `monedas`, actividad, lotes y comprobantes huérfanos.
- **Operación y observabilidad:** `/health` y `/ready` con timeout de 2 s, logs `slog` estructurados, `X-Request-Id`, redacción de claves/JIDs/teléfonos y reporte de eventos de negocio en las tablas de auditoría.
- **Documentación maestra:** `SUPER-README.md`, `README.md` sincronizado, CHANGELOG, índice de migraciones y avisos de deprecación de los módulos legados.
- **Frontends modulares:** shells HTML pequeños y módulos ES/CSS dentro del presupuesto de líneas, con service workers y assets locales.

### Changed

- El proyecto pasa de tres servicios legados a un **monolito modular** con dependencias explícitas y un único composition root en `cmd/server`.
- El esquema de base de datos es aditivo: `impactos_raw.eliminado_en` y sus índices se aplican con una migración idempotente; no hay migración destructiva ni reescritura masiva de `monedas`.
- La actividad del socio usa `mostrar.tasas` como fuente de verdad; la columna `activo` solo puede apagar, nunca encender.
- Los placeholders `NOMBRE_JID` dejan de generarse; el default es vacío y Reportes rechaza JIDs inválidos antes de llamar a Evolution.
- La publicación de tasas y los reportes respetan los límites únicos de body (20 MiB), imagen (10 MiB) y timeouts de Evolution (25 s).
- La consola administrativa exige `ADMIN_API_KEY`; si falta, responde `503 ADMIN_DISABLED` en lugar de aceptar una clave por defecto.

### Compatibility

- **64 alias legados** se mantienen sobre la API canónica `/api/v1/*` para no romper los paneles ni automatizaciones existentes.
- Cada alias responde con `Deprecation: true`, `Sunset: Sat, 31 Dec 2027 23:59:59 GMT` y `Link: <sucesor>; rel="successor-version"`.
- La ruta retirada `/api/tasas/n8n-webhook` conserva un `410 ROUTE_REMOVED` explícito con enlace a `/api/v1/tasas/guardar-lote`.
- Los envelopes de error conservan los campos que esperan los clientes legados y añaden el código estable de la API v1.
- Los alias se retiran después de 30 días sin uso y nunca durante el cutover.

### Security

- Comparación de claves administrativas y de API en tiempo constante con `crypto/subtle`; una clave vacía deshabilita la superficie correspondiente.
- Cabeceras de seguridad en todas las respuestas: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, `Cross-Origin-Opener-Policy` y HSTS solo sobre HTTPS.
- Mitigación de path traversal con decodificación de URL, rechazo de segmentos `..`, `filepath.Clean`, verificación de prefijo y `403`; las rutas API desconocidas no caen al fallback SPA.
- Aislamiento de secretos: `.env` no se copia a la imagen, las claves se cargan como secretos del orquestador y los logs no incluyen cuerpos, base64, captions, cabeceras ni credenciales.
- Límite de body, validación de JPEG, rate limit de Reportes (60/min por IP, ráfaga 10) e idempotencia en memoria con TTL de 10 minutos.
- CSP de paneles sin CDNs y con `frame-ancestors 'none'`; `reportes` no ejecuta SQL ni importa pgx.

### Migrations

- `internal/comprobantes/migrations/0001_impactos_soft_delete.sql` agrega `impactos_raw.eliminado_en TIMESTAMPTZ NULL` y los índices `impactos_raw_eliminado_en_idx` e `impactos_raw_vigentes_idx` con `IF NOT EXISTS`.
- La migración es idempotente y compatible con el servicio viejo; se aplica con backup verificado y no borra filas.
- La auditoría pre-cutover es de solo lectura y se ejecuta en una transacción `READ ONLY`; el operador revisa cada hallazgo antes de abrir tráfico real.
- El cutover, rollback, resolución de JIDs y conflictos de monedas están documentados en el Capítulo 17 del README.

### Documentation

- `SUPER-README.md` completo con capítulos de monitoreo, seguridad, apéndices, glosario financiero, FAQ operativo y ADRs.
- `README.md` sincronizado con `SUPER-README.md` para que GitHub muestre la documentación completa en la portada.
- `migrations/README.md` como índice de migraciones para el operador.
- `DEPRECATED.md` en `Tasas/`, `Perfiles/` y `Comprobantes/`, con la migración al módulo unificado y la prohibición de mantenimiento futuro fuera de `Core/`.
