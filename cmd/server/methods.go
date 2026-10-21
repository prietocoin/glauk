package main

import (
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/httpx"
)

// methodNotAllowed cierra el contrato de 405 de la seccion 5.12. Fiber no lo
// sintetiza por si solo: si la ruta existe para otro metodo, este middleware
// responde JSON con Allow en lugar de dejar que el request caiga en el 404.
func methodNotAllowed(routes []fiber.Route) fiber.Handler {
	permitidos := map[string]map[string]bool{}
	for _, route := range routes {
		if permitidos[route.Path] == nil {
			permitidos[route.Path] = map[string]bool{}
		}
		permitidos[route.Path][route.Method] = true
	}
	return func(c *fiber.Ctx) error {
		if c.Path() == "/disparar" {
			if c.Method() == fiber.MethodPost {
				return c.Next()
			}
			return responder405(c, []string{fiber.MethodPost})
		}
		if !strings.HasPrefix(c.Path(), "/api/") {
			return c.Next()
		}
		for patron, metodos := range permitidos {
			if !coincidePatron(patron, c.Path()) || metodos[c.Method()] {
				continue
			}
			allow := make([]string, 0, len(metodos))
			for metodo := range metodos {
				allow = append(allow, metodo)
			}
			return responder405(c, allow)
		}
		return c.Next()
	}
}

func responder405(c *fiber.Ctx, allow []string) error {
	sort.Strings(allow)
	c.Set("Allow", strings.Join(allow, ", "))
	return httpx.Fail(c, fiber.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
		"metodo no permitido para esta ruta")
}

// coincidePatron soporta parametros :nombre y el comodin final *. No pretender
// reemplazar al router: solo evita falsos 405 en las rutas registradas.
func coincidePatron(patron, ruta string) bool {
	if patron == ruta {
		return true
	}
	partesPatron := strings.Split(strings.Trim(patron, "/"), "/")
	partesRuta := strings.Split(strings.Trim(ruta, "/"), "/")
	if len(partesPatron) != len(partesRuta) {
		return false
	}
	for i, parte := range partesPatron {
		if parte == "*" || strings.HasPrefix(parte, "*") {
			return true
		}
		if strings.HasPrefix(parte, ":") {
			continue
		}
		if parte != partesRuta[i] {
			return false
		}
	}
	return true
}
