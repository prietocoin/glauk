package http

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/reportes"
)

func (h *Handler) enviar(c *fiber.Ctx, key string, legacy bool,
	run func(context.Context) (reportes.Resultado, error)) error {
	if !h.permitir(c) {
		if legacy {
			return falloLegacy(c, fiber.StatusTooManyRequests,
				"demasiadas solicitudes; intente más tarde")
		}
		return fallo(c, fiber.StatusTooManyRequests, "RATE_LIMITED",
			"demasiadas solicitudes; intente más tarde")
	}
	store := h.module.Idempotency
	if store == nil {
		result, err := run(c.UserContext())
		if err != nil {
			return responderError(c, err, legacy)
		}
		return h.responder(c, result, legacy)
	}

	previo, owner, err := store.Begin(key)
	switch {
	case errors.Is(err, reportes.ErrEnvioEnCurso):
		return fallo(c, fiber.StatusConflict, "ENVIO_EN_CURSO",
			"ya hay un envío en curso para esta clave")
	case errors.Is(err, reportes.ErrIdempotenciaLlena):
		if legacy {
			return falloLegacy(c, fiber.StatusTooManyRequests,
				"demasiadas solicitudes; intente más tarde")
		}
		return fallo(c, fiber.StatusTooManyRequests, "RATE_LIMITED",
			"demasiadas solicitudes; intente más tarde")
	case err != nil:
		return responderError(c, err, legacy)
	case previo != nil:
		previo.Duplicado = true
		return h.responder(c, *previo, legacy)
	case !owner:
		return fallo(c, fiber.StatusConflict, "ENVIO_EN_CURSO",
			"ya hay un envío en curso para esta clave")
	}

	result, err := run(c.UserContext())
	if err != nil {
		store.Abort(key)
		return responderError(c, err, legacy)
	}
	store.Commit(key, result)
	return h.responder(c, result, legacy)
}

func (h *Handler) responder(c *fiber.Ctx, result reportes.Resultado, legacy bool) error {
	if legacy {
		return escribirImagenLegacy(c, result)
	}
	return escribirResultado(c, result)
}

func (h *Handler) permitir(c *fiber.Ctx) bool {
	if h.module == nil || h.module.RateLimiter == nil {
		return true
	}
	if h.module.RateLimiter.Allow(c.IP()) {
		return true
	}
	if segundos := h.module.RateLimiter.RetryAfter(c.IP()); segundos > 0 {
		c.Set("Retry-After", strconv.Itoa(segundos))
	}
	return false
}

func (h *Handler) clave(c *fiber.Ctx, destino, caption, contenido string) string {
	if h.module == nil || h.module.Idempotency == nil {
		return ""
	}
	return h.module.Idempotency.Key(c.Get("Idempotency-Key"), destino, caption, contenido)
}

func (h *Handler) valido() bool {
	return h != nil && h.module != nil && h.module.Service != nil
}
