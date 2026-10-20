package httpx

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
)

// VerifyAdmin compares the supplied and expected keys in constant time. An
// empty configured key disables admin access instead of accepting anything.
func VerifyAdmin(provided, expected string) bool {
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// RequireAdminKey protects the administrative console.
func RequireAdminKey(expected string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if expected == "" {
			return Fail(c, fiber.StatusServiceUnavailable, "ADMIN_DISABLED",
				"la consola administrativa no está configurada")
		}
		provided := c.Get("X-Admin-Key")
		if provided == "" {
			provided = c.Cookies("admin_key")
		}
		if !VerifyAdmin(provided, expected) {
			return Fail(c, fiber.StatusUnauthorized, "UNAUTHORIZED",
				"clave administrativa inválida")
		}
		return c.Next()
	}
}
