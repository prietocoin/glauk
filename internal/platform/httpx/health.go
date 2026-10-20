package httpx

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

const healthTimeout = 2 * time.Second

type HealthDeps struct {
	Database  func(context.Context) error
	Evolution func() (instance string, simulated bool)
	Version   string
	Started   time.Time
}

func Health(deps HealthDeps) fiber.Handler { return healthHandler(deps, true) }
func Ready(deps HealthDeps) fiber.Handler  { return healthHandler(deps, false) }
func healthHandler(deps HealthDeps, detailed bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		status, database := fiber.StatusServiceUnavailable, map[string]any{"status": "unavailable"}
		checks := make(map[string]any, 2)
		if deps.Database != nil {
			start := time.Now()
			ctx, cancel := context.WithTimeout(c.UserContext(), healthTimeout)
			err := deps.Database(ctx)
			cancel()
			database["latency_ms"] = time.Since(start).Milliseconds()
			if err != nil {
				database["status"] = "error"
			} else {
				database["status"], status = "ok", fiber.StatusOK
			}
		}
		checks["database"] = database
		if deps.Evolution != nil {
			instance, simulated := deps.Evolution()
			evolution := map[string]any{"status": "ok"}
			if simulated {
				evolution["status"] = "simulated"
			}
			if detailed {
				evolution["instance"] = instance
			}
			checks["evolution"] = evolution
		}
		report := map[string]any{"status": "ok", "service": "glauk", "checks": checks,
			"uptime_s": int64(time.Since(deps.Started).Seconds())}
		if status != fiber.StatusOK {
			report["status"] = "degraded"
		}
		if detailed {
			report["version"] = deps.Version
		}
		c.Set("Cache-Control", "no-store")
		return c.Status(status).JSON(report)
	}
}
