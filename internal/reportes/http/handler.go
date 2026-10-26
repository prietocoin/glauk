package http

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/reportes"
)

// Handler traduce Fiber al dominio de reportes. No conoce adaptadores ni
// servicios de otros dominios: solo el módulo que se inyecta desde cmd/server.
type Handler struct {
	module *reportes.Module
}

// NewHandler construye la capa HTTP sobre el módulo ya ensamblado.
func NewHandler(module *reportes.Module) *Handler {
	return &Handler{module: module}
}

// Filtros responde GET /api/v1/reportes/filtros con el envelope canónico.
func (h *Handler) Filtros(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	if !h.valido() {
		return fallo(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE",
			"el servicio de reportes no está configurado")
	}
	filtros, err := h.module.Filtros(c.UserContext(), c.Query("rol"))
	if err != nil {
		return responderError(c, err, false)
	}
	return escribirFiltros(c, filtros)
}

// FiltrosLegacy conserva el envelope plano que consume api.js.
func (h *Handler) FiltrosLegacy(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	if !h.valido() {
		return falloLegacy(c, fiber.StatusServiceUnavailable,
			"el servicio de reportes no está configurado")
	}
	filtros, err := h.module.Filtros(c.UserContext(), c.Query("rol"))
	if err != nil {
		return responderError(c, err, true)
	}
	return escribirFiltrosLegacy(c, filtros)
}

// EnviarImagen responde POST /api/v1/reportes/imagen.
func (h *Handler) EnviarImagen(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	var req EnviarImagenRequest
	if err := c.BodyParser(&req); err != nil {
		return fallo(c, fiber.StatusBadRequest, "INVALID_JSON", "payload JSON inválido")
	}
	return h.imagen(c, req, false)
}

// EnviarImagenLegacy responde los alias históricos de envío de imagen.
func (h *Handler) EnviarImagenLegacy(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	var req EnviarImagenRequest
	if err := c.BodyParser(&req); err != nil {
		return falloLegacy(c, fiber.StatusBadRequest, "payload JSON inválido")
	}
	return h.imagen(c, req, true)
}

// EnviarTexto responde POST /api/v1/reportes/texto.
func (h *Handler) EnviarTexto(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	var req EnviarTextoRequest
	if err := c.BodyParser(&req); err != nil {
		return fallo(c, fiber.StatusBadRequest, "INVALID_JSON", "payload JSON inválido")
	}
	if !h.valido() {
		return fallo(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE",
			"el servicio de reportes no está configurado")
	}
	destino := strings.TrimSpace(req.RemoteJID)
	if destino == "" {
		destino = strings.TrimSpace(req.Socio)
	}
	key := h.clave(c, destino, "", req.Texto)
	return h.enviar(c, key, false, func(ctx context.Context) (reportes.Resultado, error) {
		return h.module.Service.EnviarTexto(ctx, req.Solicitud())
	})
}

func (h *Handler) imagen(c *fiber.Ctx, req EnviarImagenRequest, legacy bool) error {
	if !h.valido() {
		return fallo(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE",
			"el servicio de reportes no está configurado")
	}
	destino := strings.TrimSpace(req.RemoteJID)
	if destino == "" {
		destino = strings.TrimSpace(req.Socio)
	}
	key := h.clave(c, destino, req.Caption, req.Base64)
	return h.enviar(c, key, legacy, func(ctx context.Context) (reportes.Resultado, error) {
		return h.module.Service.EnviarImagen(ctx, req.Solicitud())
	})
}
