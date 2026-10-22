package httpx

import (
	"crypto/subtle"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/config"
	"glauk/internal/platform/logging"
)

const (
	requestIDKey = "request_id"
	tenantKey    = "tenant"
	requestIDHdr = "X-Request-Id"
)

// RequestID validates an incoming UUID/ULID or generates a 16-byte hex ID.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Get(requestIDHdr))
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Locals(requestIDKey, id)
		c.Set(requestIDHdr, id)
		return c.Next()
	}
}

// RequestIDFrom returns the ID stored by RequestID.
func RequestIDFrom(c *fiber.Ctx) string {
	id, _ := c.Locals(requestIDKey).(string)
	return id
}

// AccessLog emits only operational metadata; bodies, headers and query
// strings with credentials never reach the logger.
func AccessLog(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		if log == nil {
			return err
		}
		status := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else if status < 400 {
				status = fiber.StatusInternalServerError
			}
		}
		tenant, _ := c.Locals(tenantKey).(string)
		logging.Request{
			RequestID: RequestIDFrom(c),
			Method:    c.Method(),
			Path:      c.Path(),
			Status:    status,
			Duration:  time.Since(start),
			Bytes:     int64(len(c.Response().Body())),
			Tenant:    tenant,
		}.Log(log)
		return err
	}
}

// SecurityHeaders applies the baseline policy to every API response.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Cross-Origin-Opener-Policy", "same-origin")
		if c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	}
}

// RequireAPIKey protects mutations when SECURITY_API_KEY is configured. An
// empty key keeps historical read/write behavior for existing panels.
func RequireAPIKey(cfg *config.Config) fiber.Handler {
	expected := ""
	if cfg != nil {
		expected = strings.TrimSpace(cfg.HTTP.SecurityAPIKey)
	}
	return func(c *fiber.Ctx) error {
		if expected == "" {
			return c.Next()
		}
		provided := c.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			return Fail(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "clave de API inválida")
		}
		return c.Next()
	}
}
