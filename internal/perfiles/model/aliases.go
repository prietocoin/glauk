package model

import (
	"encoding/json"
	"strings"
)

// UnmarshalJSON acepta los alias históricos del panel sin relajar las reglas de
// negocio: solo traduce claves antes de entregar el GuardarRequest normalizado.
// Un campo canónico siempre gana sobre su alias, y activo sigue siendo ignorado.
func (r *GuardarRequest) UnmarshalJSON(data []byte) error {
	type base GuardarRequest
	var normal base
	if err := json.Unmarshal(data, &normal); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	aliasString(raw, &normal.Nombre, "socio", "nombre_socio", "nombreSocio")
	aliasString(raw, &normal.IDGrupo, "jid", "grupo", "idGrupo")
	aliasString(raw, &normal.MonedaBase, "moneda", "monedaBase")
	aliasString(raw, &normal.SaldoInicial, "saldo", "saldoInicial")
	*r = GuardarRequest(normal)
	return nil
}

// aliasString solo completa un destino vacío. Así una clave canónica presente
// con cadena vacía no es sobrescrita por un alias con un valor distinto.
func aliasString(raw map[string]json.RawMessage, destino any, claves ...string) {
	switch actual := destino.(type) {
	case *string:
		if strings.TrimSpace(*actual) != "" {
			return
		}
	case *FlexibleFloat:
		if strings.TrimSpace(string(*actual)) != "" {
			return
		}
	}
	for _, clave := range claves {
		valor, ok := raw[clave]
		if !ok {
			continue
		}
		if json.Unmarshal(valor, destino) == nil {
			return
		}
	}
}
