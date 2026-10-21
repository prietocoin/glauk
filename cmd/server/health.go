package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"glauk/internal/platform/evolution"
	"glauk/internal/platform/httpx"
	"glauk/internal/platform/postgres"
)

const healthPingTimeout = 2 * time.Second

// newHealthHandlers conecta /health y /ready con PostgreSQL y Evolution sin
// exponer detalles de conexión al cliente.
func newHealthHandlers(pool *pgxpool.Pool, evo *evolution.Client,
	instance string, started time.Time) (fiber.Handler, fiber.Handler) {
	deps := httpx.HealthDeps{
		Database: func(ctx context.Context) error {
			return postgres.Ping(ctx, pool, healthPingTimeout)
		},
		Evolution: func() (string, bool) {
			if evo == nil {
				return instance, true
			}
			return instance, evo.Simulated()
		},
		Version: "dev",
		Started: started,
	}
	return httpx.Health(deps), httpx.Ready(deps)
}
