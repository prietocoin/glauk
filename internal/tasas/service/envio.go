package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"glauk/internal/platform/evolution"
	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/render"
	"glauk/internal/tasas/store"
)

// Sender es la única operación de Evolution que Tasas necesita. La implementa
// platform/evolution.Client; el servicio no conoce HTTP, reintentos ni credenciales.
type Sender interface {
	SendMedia(ctx context.Context, in evolution.MediaInput) (evolution.Receipt, error)
}

// enviarCartelera renderiza el PNG, lo codifica en base64 puro y lo despacha al
// destino que corresponda. El pie distingue prueba de oficial, igual que el panel.
func (s *Service) enviarCartelera(
	ctx context.Context,
	socio store.Socio,
	cartelera domain.Cartelera,
	modoPrueba bool,
) (evolution.Receipt, error) {
	if s.sender == nil {
		return evolution.Receipt{}, fmt.Errorf("cliente de Evolution no configurado")
	}
	destino, err := s.destino(socio, modoPrueba)
	if err != nil {
		return evolution.Receipt{}, err
	}
	png, err := render.GenerarCartelera(cartelera)
	if err != nil {
		return evolution.Receipt{}, err
	}
	prefijo := "🚀 [OFICIAL]"
	if modoPrueba {
		prefijo = "🧪 [PRUEBA]"
	}
	return s.sender.SendMedia(ctx, evolution.MediaInput{
		Number:    destino,
		MediaType: "image",
		MimeType:  "image/png",
		FileName:  "cartelera.png",
		Media:     base64.StdEncoding.EncodeToString(png),
		Caption:   fmt.Sprintf("%s Cartelera *%s* — Lote: %s", prefijo, cartelera.NombreSocio, cartelera.LoteTasa),
	})
}

// destino elige el grupo real del socio o el grupo de pruebas. En modo prueba
// siempre va al JID_GRUPO_PRUEBA; fuera de prueba, un socio sin grupo también cae
// allí para no publicar en un destino inventado.
func (s *Service) destino(socio store.Socio, modoPrueba bool) (string, error) {
	if !modoPrueba && socio.IDGrupo != "" {
		return socio.IDGrupo, nil
	}
	return s.grupoDePrueba()
}

// grupoDePrueba falla de forma explícita cuando falta configuración: publicar en
// un grupo equivocado es peor que no publicar.
func (s *Service) grupoDePrueba() (string, error) {
	if s.jidPrueba == "" {
		return "", fmt.Errorf("JID_GRUPO_PRUEBA sin configurar")
	}
	return s.jidPrueba, nil
}
