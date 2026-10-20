package httpx

import (
	"io"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"glauk/internal/platform/config"
)

// NewApp builds the single Fiber application. Domain routes and static
// handlers are registered later by the composition root, preserving the
// normative order API -> health -> API 404 -> static -> SPA.
func NewApp(cfg *config.Config, log *slog.Logger) *fiber.App {
	if cfg == nil {
		panic("httpx: configuración nil")
	}
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	app := fiber.New(fiberConfig(cfg, log))

	app.Use(RequestID())
	app.Use(recover.New(recover.Config{EnableStackTrace: false}))
	app.Use(AccessLog(log))
	app.Use(cors.New(corsConfig(cfg)))
	app.Use(SecurityHeaders())
	return app
}

func fiberConfig(cfg *config.Config, log *slog.Logger) fiber.Config {
	return fiber.Config{
		AppName:               "Glauk",
		BodyLimit:             cfg.HTTP.BodyLimit,
		ReadTimeout:           cfg.HTTP.ReadTimeout,
		WriteTimeout:          cfg.HTTP.WriteTimeout,
		IdleTimeout:           cfg.HTTP.IdleTimeout,
		DisableStartupMessage: cfg.AppEnv == "production",
		ErrorHandler:          ErrorHandler(log),
	}
}

func corsConfig(cfg *config.Config) cors.Config {
	return cors.Config{
		AllowOrigins:  joinCSV(cfg.HTTP.AllowedOrigins),
		AllowMethods:  "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:  "Origin,Content-Type,Accept,X-Admin-Key,X-API-Key,X-Request-Id,Idempotency-Key",
		ExposeHeaders: "X-Request-Id",
		MaxAge:        600,
	}
}

func joinCSV(values []string) string {
	if len(values) == 0 {
		return "*"
	}
	return strings.Join(values, ",")
}
