package service

import (
	"context"
	"strings"

	"glauk/internal/perfiles/model"
)

// LegacyRequest es la forma que usaban /api/socios/config y /api/directorio.
//
// El panel canónico ya envía nombre, id_grupo y moneda_base, pero los alias
// históricos siguen existiendo en cmd/server. Se traducen aquí, una sola vez,
// para que los handlers no dupliquen reglas de defaults ni de H-01.
type LegacyRequest struct {
	Nombre       string              `json:"nombre"`
	Socio        string              `json:"socio"`
	IDGrupo      string              `json:"id_grupo"`
	JID          string              `json:"jid"`
	Rol          string              `json:"rol"`
	MonedaBase   string              `json:"moneda_base"`
	Moneda       string              `json:"moneda"`
	SaldoInicial model.FlexibleFloat `json:"saldo_inicial"`
	Saldo        model.FlexibleFloat `json:"saldo"`
	Mostrar      *model.Mostrar      `json:"mostrar"`
	Monedas      model.Monedas       `json:"monedas"`
	Herencia     bool                `json:"herencia"`
}

// Canonico reduce el payload legacy al GuardarRequest del dominio.
func (l LegacyRequest) Canonico() model.GuardarRequest {
	return model.GuardarRequest{
		Nombre:       primeroLegacy(l.Nombre, l.Socio),
		IDGrupo:      primeroLegacy(l.IDGrupo, l.JID),
		Rol:          l.Rol,
		MonedaBase:   primeroLegacy(l.MonedaBase, l.Moneda),
		SaldoInicial: primeroSaldo(l.SaldoInicial, l.Saldo),
		Mostrar:      l.Mostrar,
		Monedas:      l.Monedas,
		Herencia:     l.Herencia,
	}
}

// GuardarLegacy conserva el contrato de /api/socios/config y /api/directorio.
func (s *Service) GuardarLegacy(ctx context.Context, legacy LegacyRequest) (model.Perfil, error) {
	return s.Guardar(ctx, legacy.Canonico())
}

func primeroSaldo(canonico, alias model.FlexibleFloat) model.FlexibleFloat {
	if canonico.String() != "0" || alias.String() == "0" {
		return canonico
	}
	return alias
}

func primeroLegacy(valores ...string) string {
	for _, valor := range valores {
		if strings.TrimSpace(valor) != "" {
			return valor
		}
	}
	return ""
}
