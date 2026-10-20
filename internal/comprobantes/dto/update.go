package dto

import "glauk/internal/comprobantes/domain"

// UpdateInput es el payload de PUT /api/v1/comprobantes/:hashLargo.
//
// Incluye todos los alias que el panel envió históricamente. Los métodos
// resuelven la precedencia en un único lugar para evitar reglas divergentes.
type UpdateInput struct {
	Monto            OptionalFloat `json:"monto"`
	Moneda           string        `json:"moneda"`
	MonedaRecibo     string        `json:"moneda_recibo"`
	Banco            string        `json:"banco"`
	Referencia       string        `json:"referencia"`
	Titular          string        `json:"titular"`
	IDTasa           string        `json:"id_tasa"`
	LoteTasa         string        `json:"lote_tasa"`
	LoteTasaAsignado string        `json:"lote_tasa_asignado"`
	NombreSocio1     string        `json:"nombre_socio_1"`
	Socio1           string        `json:"socio_1"`
	Socio1Legacy     string        `json:"socio1"`
	NombreSocio2     string        `json:"nombre_socio_2"`
	Socio2           string        `json:"socio_2"`
	Socio2Legacy     string        `json:"socio2"`
	TipoManual       string        `json:"tipo_manual"`
	TipoOp1          string        `json:"tipo_op1"`
	TipoOp           string        `json:"tipo_op"`
	Monto1           OptionalFloat `json:"monto_1"`
	Monto2           OptionalFloat `json:"monto_2"`
	M1Socio          OptionalFloat `json:"m1_socio"`
	M2Socio          OptionalFloat `json:"m2_socio"`
	Estado           *string       `json:"estado"`
	Conteo           *int          `json:"conteo"`
	Timestamp        *int64        `json:"timestamp"`
}

// Lote resuelve los alias de lote usados por el frontend.
func (i UpdateInput) Lote() string {
	return firstNonEmpty(i.LoteTasaAsignado, i.LoteTasa, i.IDTasa, DefaultLoteTasa)
}

// Socio1Nombre resuelve los alias del socio 1.
func (i UpdateInput) Socio1Nombre() string {
	return firstNonEmpty(i.NombreSocio1, i.Socio1, i.Socio1Legacy, DefaultSocio)
}

// Socio2Nombre resuelve los alias del socio 2.
func (i UpdateInput) Socio2Nombre() string {
	return firstNonEmpty(i.NombreSocio2, i.Socio2, i.Socio2Legacy, DefaultSocio)
}

// MonedaValor resuelve los alias de moneda.
func (i UpdateInput) MonedaValor() string {
	return firstNonEmpty(i.Moneda, i.MonedaRecibo)
}

// TipoValor resuelve los alias de naturaleza de la operación.
func (i UpdateInput) TipoValor() string {
	return firstNonEmpty(i.TipoManual, i.TipoOp1, i.TipoOp, DefaultTipo)
}

// Monto1Valor devuelve el override manual del socio 1, si existe.
func (i UpdateInput) Monto1Valor() (float64, bool) {
	if i.Monto1.Set {
		return i.Monto1.Value, true
	}
	if i.M1Socio.Set {
		return i.M1Socio.Value, true
	}
	return 0, false
}

// Monto2Valor devuelve el override manual del socio 2, si existe.
func (i UpdateInput) Monto2Valor() (float64, bool) {
	if i.Monto2.Set {
		return i.Monto2.Value, true
	}
	if i.M2Socio.Set {
		return i.M2Socio.Value, true
	}
	return 0, false
}

// NormalizedCurrency devuelve la divisa que se persistirá en comprobantes_raw.
func (i UpdateInput) NormalizedCurrency() string {
	return domain.NormalizeCurrency(i.MonedaValor(), "COP")
}

// QueueUpdate adapta el payload compartido a los campos raw de cola_fb.
func (i UpdateInput) QueueUpdate() domain.QueueUpdate {
	estado := "PROCESADO"
	if i.Estado != nil && domain.FirstNonEmpty(*i.Estado) != "" {
		estado = domain.FirstNonEmpty(*i.Estado)
	}
	conteo := 1
	if i.Conteo != nil {
		conteo = *i.Conteo
	}
	return domain.QueueUpdate{
		NombreSocio1: optionalString(domain.FirstNonEmpty(i.NombreSocio1, i.Socio1, i.Socio1Legacy)),
		NombreSocio2: optionalString(domain.FirstNonEmpty(i.NombreSocio2, i.Socio2, i.Socio2Legacy)),
		Estado:       &estado,
		Conteo:       &conteo,
		Timestamp:    i.Timestamp,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
