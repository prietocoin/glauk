package httpx

import (
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"glauk/internal/platform/config"
)

// NewApp builds the single Fiber application for the REST API. Domain routes
// are registered later by the composition root.
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
	allowed := cfg.HTTP.AllowedOrigins
	return cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return originAllowed(origin, allowed)
		},
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		// Authorization y X-Requested-With son el contrato de glauk-ui;
		// las demas cabeceras conservan la compatibilidad de los paneles previos.
		AllowHeaders:  "Content-Type,Authorization,X-Requested-With,Accept,X-Request-Id,Idempotency-Key,X-Admin-Key,X-API-Key",
		ExposeHeaders: "X-Request-Id",
		MaxAge:        600,
	}
}

// originAllowed accepts exact origins and the deployment pattern
// "http://localhost:*", which Fiber's static matcher cannot express.
func originAllowed(origin string, allowed []string) bool {
	origin = strings.ToLower(strings.TrimSpace(origin))
	if origin == "" {
		return false
	}
	for _, pattern := range allowed {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "*" || pattern == origin {
			return true
		}
		if strings.HasSuffix(pattern, ":*") && anyPortMatch(origin, strings.TrimSuffix(pattern, ":*")) {
			return true
		}
	}
	return false
}

func anyPortMatch(origin, base string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return false
	}
	expected, err := url.Parse(base)
	if err != nil || expected.Scheme == "" || expected.Hostname() == "" {
		return false
	}
	if u.Scheme != expected.Scheme || u.Hostname() != expected.Hostname() {
		return false
	}
	return expected.Port() == "" || expected.Port() == u.Port()
}
