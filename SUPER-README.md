# Glauk — SUPER-README

Documento maestro de arquitectura, operación y contratos del monolito modular **Glauk**.

| Campo | Valor |
|---|---|
| Versión | 0.1.0 (esqueleto) |
| Fecha | 2026-10-20 |
| Estado | Fase 1 — andamiaje del módulo |
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

Matriz completa de variables, precedencia, secretos, validación, ejemplos por entorno y rotación.

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
