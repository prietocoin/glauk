package service

import (
	"log/slog"
	"strings"

	"glauk/internal/perfiles/model"
	"glauk/internal/perfiles/repo"
)

// NormalizarNombre replica UPPER(BTRIM(nombre)) de PostgreSQL.
func NormalizarNombre(nombre string) string {
	return strings.ToUpper(strings.TrimSpace(nombre))
}

// buildParams aplica los defaults del contrato y elimina cualquier placeholder.
// Un id_grupo ausente queda vacío y produce WARN; nunca se sintetiza un JID.
func buildParams(nombre string, req model.GuardarRequest, log *slog.Logger) repo.UpsertParams {
	idGrupo := strings.TrimSpace(req.IDGrupo)
	if idGrupo == "" && log != nil {
		log.Warn("id_grupo vacío: el alta queda pendiente de JID", "socio", nombre)
	}
	monedas := req.Monedas
	if monedas == nil {
		monedas = model.Monedas{}
	}
	return repo.UpsertParams{
		IDGrupo:      idGrupo,
		Nombre:       nombre,
		Rol:          fallback(req.Rol, "SOCIO"),
		MonedaBase:   strings.ToUpper(fallback(req.MonedaBase, "USDT")),
		SaldoInicial: fallback(req.SaldoInicial.String(), "0"),
		Mostrar:      model.MostrarDe(req.Mostrar),
		Monedas:      monedas,
		Herencia:     req.Herencia,
	}
}

// advertirMonedas emite un WARN por cada valor legacy que porcentaje descartó.
func advertirMonedas(nombre string, monedas model.Monedas, log *slog.Logger) {
	if log == nil {
		return
	}
	for _, conflicto := range monedas.Conflictos() {
		log.Warn("monedas: porcentaje gana sobre el valor legacy",
			"socio", nombre,
			"moneda", conflicto.Moneda,
			"campo", conflicto.Campo,
			"descartado", conflicto.Descartado,
			"conservado", conflicto.Conservado,
		)
	}
}

func fallback(valor, defecto string) string {
	if strings.TrimSpace(valor) == "" {
		return defecto
	}
	return strings.TrimSpace(valor)
}
