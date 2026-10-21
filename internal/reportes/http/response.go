package http

import (
	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/httpx"
	"glauk/internal/reportes"
)

func escribirFiltros(c *fiber.Ctx, f reportes.Filtros) error {
	return httpx.OK(c, fiber.Map{
		"roles": sinNulos(f.Roles), "entidades": sinNulos(f.Entidades),
		"hashes": sinNulos(f.Hashes),
	})
}

func escribirResultado(c *fiber.Ctx, r reportes.Resultado) error {
	return httpx.OK(c, resultadoDe(r))
}

func escribirFiltrosLegacy(c *fiber.Ctx, f reportes.Filtros) error {
	return httpx.OKFlat(c, map[string]any{
		"roles": sinNulos(f.Roles), "entidades": sinNulos(f.Entidades),
		"hashes": sinNulos(f.Hashes),
	})
}

func escribirImagenLegacy(c *fiber.Ctx, r reportes.Resultado) error {
	return httpx.OKFlat(c, map[string]any{
		"message": "Reporte enviado con éxito.",
		"jid":     r.JID, "remoteJid": r.JID,
		"data": map[string]any{
			"messageId": r.MessageID, "status": r.Estado,
			"simulated": r.Simulado, "duplicate": r.Duplicado,
			"bytes": r.Bytes,
		},
	})
}

func sinNulos(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
