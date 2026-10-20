package httpx

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// HTTPError carries a stable public code and message through the handler stack.
type HTTPError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *HTTPError) Error() string { return e.Message }
func (e *HTTPError) Unwrap() error { return e.Err }

// ErrorHandler converts known errors into envelopes. Unknown failures are
// logged with their cause and returned as a fixed INTERNAL message.
func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			status := httpErr.Status
			if status < 400 || status > 599 {
				status = fiber.StatusInternalServerError
			}
			code := httpErr.Code
			if code == "" {
				code = codeForStatus(status)
			}
			message := httpErr.Message
			if message == "" && status >= 500 {
				message = "error interno del servidor"
			}
			if status >= 500 && log != nil {
				log.Error("error http", "method", c.Method(), "path", c.Path(),
					"request_id", RequestIDFrom(c), "status", status, "error", err)
			}
			return Fail(c, status, code, message)
		}

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			if fiberErr.Code >= 500 && log != nil {
				log.Error("error http", "method", c.Method(), "path", c.Path(),
					"request_id", RequestIDFrom(c), "status", fiberErr.Code, "error", err)
			}
			return Fail(c, fiberErr.Code, codeForStatus(fiberErr.Code), fiberErr.Message)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return Fail(c, fiber.StatusGatewayTimeout, "TIMEOUT",
				"la operación excedió el tiempo máximo")
		}

		if log != nil {
			log.Error("error no controlado", "method", c.Method(), "path", c.Path(),
				"request_id", RequestIDFrom(c), "error", err)
		}
		return Fail(c, fiber.StatusInternalServerError, "INTERNAL",
			"error interno del servidor")
	}
}

func codeForStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "VALIDATION_ERROR"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case fiber.StatusConflict:
		return "CONFLICT"
	case fiber.StatusRequestEntityTooLarge:
		return "PAYLOAD_TOO_LARGE"
	case fiber.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	case fiber.StatusTooManyRequests:
		return "RATE_LIMITED"
	case fiber.StatusBadGateway:
		return "EVOLUTION_UNAVAILABLE"
	case fiber.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case fiber.StatusGatewayTimeout:
		return "TIMEOUT"
	default:
		if status >= 500 {
			return "INTERNAL"
		}
		return "NOT_FOUND"
	}
}
