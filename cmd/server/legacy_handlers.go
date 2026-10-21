package main

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	perfileshttp "glauk/internal/perfiles/http"
	perfilesservice "glauk/internal/perfiles/service"
	tasasdomain "glauk/internal/tasas/domain"
	tasashttp "glauk/internal/tasas/http"
	"glauk/internal/tasas/render"
	"glauk/internal/tasas/service"
	"glauk/internal/tasas/store"
)

// legacyGuardarPerfil traduce el payload histórico de /api/socios/config.
func legacyGuardarPerfil(deps perfileshttp.Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req perfilesservice.LegacyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false, "error": "payload JSON inválido", "code": "JSON_INVALIDO",
			})
		}
		perfil, err := deps.Perfiles.GuardarLegacy(c.UserContext(), req)
		if err != nil {
			return falloPerfil(c, err)
		}
		return c.JSON(fiber.Map{"success": true, "data": perfil})
	}
}

func falloPerfil(c *fiber.Ctx, err error) error {
	status, code := fiber.StatusInternalServerError, "ERROR_INTERNO"
	switch {
	case errors.Is(err, perfilesservice.ErrNombreRequerido):
		status, code = fiber.StatusBadRequest, "NOMBRE_REQUERIDO"
	case errors.Is(err, perfilesservice.ErrEntidadProtegida):
		status, code = fiber.StatusBadRequest, "ENTIDAD_PROTEGIDA"
	case errors.Is(err, pgx.ErrNoRows):
		status, code = fiber.StatusNotFound, "NO_ENCONTRADO"
	}
	return c.Status(status).JSON(fiber.Map{"success": false, "error": err.Error(), "code": code})
}

// publicarAlias implementa /api/tasas/publicar: publica el lote indicado, o el
// vigente, y guarda el cuerpo si llega un lote nuevo sin id_tasa.
func publicarAlias(deps *tasashttp.Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			IDTasa     string         `json:"id_tasa"`
			Tasas      map[string]any `json:"tasas"`
			ModoPrueba bool           `json:"modoPrueba"`
		}
		if err := c.BodyParser(&body); err != nil {
			return tasashttp.Fail(c, fiber.StatusBadRequest, "cuerpo de publicación inválido")
		}
		var publicacion service.Publicacion
		var err error
		switch {
		case strings.TrimSpace(body.IDTasa) != "":
			publicacion, err = deps.Service.PublicarLote(c.UserContext(), body.IDTasa, body.ModoPrueba)
		case len(body.Tasas) > 0:
			var lote tasasdomain.Lote
			lote, err = deps.Service.GuardarLote(c.UserContext(), body.Tasas)
			if err == nil {
				publicacion, err = deps.Service.PublicarLote(c.UserContext(), lote.IDTasa, body.ModoPrueba)
			}
		default:
			publicacion, err = deps.Service.Publicar(c.UserContext(), body.ModoPrueba)
		}
		if err != nil {
			return tasashttp.ErrorDeDominio(c, err)
		}
		return tasashttp.OKFlat(c, fiber.Map{
			"id_tasa": publicacion.IDTasa, "procesados": publicacion.Procesados,
			"detalle": publicacion.Detalle,
		})
	}
}

// reenviarAlias implementa /api/tasas/reenviar. GENERAL significa todos los
// socios activos, igual que el contrato histórico.
func reenviarAlias(deps *tasashttp.Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			IDTasa     string `json:"id_tasa"`
			Socio      string `json:"socio"`
			ModoPrueba bool   `json:"modoPrueba"`
		}
		if err := c.BodyParser(&body); err != nil {
			return tasashttp.Fail(c, fiber.StatusBadRequest, "cuerpo de reenvío inválido")
		}
		var publicacion service.Publicacion
		var err error
		if strings.TrimSpace(body.IDTasa) != "" {
			publicacion, err = deps.Service.PublicarLote(c.UserContext(), body.IDTasa, body.ModoPrueba)
		} else {
			nombre := body.Socio
			if strings.EqualFold(strings.TrimSpace(nombre), "GENERAL") {
				nombre = ""
			}
			publicacion, err = deps.Service.Reenviar(c.UserContext(), nombre, body.ModoPrueba)
		}
		if err != nil {
			return tasashttp.ErrorDeDominio(c, err)
		}
		return tasashttp.OKFlat(c, fiber.Map{
			"id_tasa": publicacion.IDTasa, "procesados": publicacion.Procesados,
			"detalle": publicacion.Detalle,
		})
	}
}

// previewImage conserva /api/preview-image/:identificador. Sin parámetro usa
// GENERAL, como el nombre legado del panel.
func previewImage(deps *tasashttp.Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nombre := strings.TrimSpace(c.Params("identificador"))
		if nombre == "" {
			nombre = "GENERAL"
		}
		cartelera, err := deps.Service.CarteleraDe(c.UserContext(), nombre)
		if errors.Is(err, store.ErrNotFound) {
			return tasashttp.SinSocio(c)
		}
		if err != nil {
			return tasashttp.SinImagen(c)
		}
		png, err := render.GenerarCartelera(cartelera)
		if err != nil {
			return tasashttp.SinImagen(c)
		}
		return c.Type("png").Send(png)
	}
}
