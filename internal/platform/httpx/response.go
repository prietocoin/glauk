package httpx

import "github.com/gofiber/fiber/v2"

// Envelope is the canonical contract for new endpoints and every error.
type Envelope struct {
	Success   bool   `json:"success"`
	Data      any    `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// OK writes a successful canonical envelope with HTTP 200.
func OK(c *fiber.Ctx, data any) error {
	return c.JSON(Envelope{Success: true, Data: data, RequestID: RequestIDFrom(c)})
}

// Created writes a successful canonical envelope with HTTP 201.
func Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).
		JSON(Envelope{Success: true, Data: data, RequestID: RequestIDFrom(c)})
}

// Fail writes the canonical error envelope and stamps the request ID.
func Fail(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(Envelope{
		Success:   false,
		Error:     message,
		Code:      code,
		RequestID: RequestIDFrom(c),
	})
}

// FailCode is the explicit form used when the stable catalog code is known.
func FailCode(c *fiber.Ctx, status int, code, message string) error {
	return Fail(c, status, code, message)
}

// OKFlat preserves legacy panels that read domain fields from the root.
func OKFlat(c *fiber.Ctx, fields map[string]any) error {
	body := make(map[string]any, len(fields)+2)
	for key, value := range fields {
		body[key] = value
	}
	body["success"] = true
	if id := RequestIDFrom(c); id != "" {
		body["requestId"] = id
	}
	return c.JSON(body)
}
