package adapters

import (
	comprobantesservice "glauk/internal/comprobantes/service"
	"glauk/internal/platform/evolution"
	"glauk/internal/reportes"
)

// Assertions de compilacion: si cambia una firma del dominio, el fallo aparece
// aqui y no en el primer request de produccion.
var (
	_ reportes.SociosProvider      = (*SociosPerfiles)(nil)
	_ reportes.MovimientosProvider = (*MovimientosComprobantes)(nil)
	_ reportes.WhatsAppSender      = (*WhatsAppEvolution)(nil)
)

// New ensambla el modulo de reportes con los adaptadores reales.
//
// Solo cmd/server llama a esta funcion: el dominio y su capa HTTP reciben
// interfaces y nunca construyen dependencias externas por su cuenta. El
// modulo conserva la idempotencia y el rate limiting en memoria del proceso.
func New(perfilesReader SocioReader, comprobantesReader comprobantesservice.ReportReader,
	evo *evolution.Client, cfg reportes.Config) *reportes.Module {
	service := reportes.NewService(
		NewSociosPerfiles(perfilesReader),
		NewMovimientosComprobantes(comprobantesReader),
		NewWhatsAppEvolution(evo),
		cfg,
	)
	return reportes.NewModule(service)
}
