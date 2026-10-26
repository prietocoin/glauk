package domain

import (
	"math"
	"strings"
)

// Liquidacion es el snapshot financiero persistido en comprobantes_liq.
//
// Cada edición del comprobante lo reemplaza dentro de una transacción con el
// contrato vigente de los socios. La proyección de lectura vuelve a aplicar ese
// contrato en vivo, de modo que un cambio de polaridad o comisión se refleje sin
// migrar filas históricas.
type Liquidacion struct {
	HashLargo string  `json:"hash_largo"`
	Socio1    string  `json:"socio_1"`
	TipoOp1   string  `json:"tipo_op1"`
	Monto1    float64 `json:"monto_1"`
	Tasa1     float64 `json:"tasa_1"`
	ME1       float64 `json:"me1"`
	Socio2    *string `json:"socio_2"`
	TipoOp2   string  `json:"tipo_op2"`
	Monto2    float64 `json:"monto_2"`
	Tasa2     float64 `json:"tasa_2"`
	ME2       float64 `json:"me2"`
	LoteTasa  string  `json:"lote_tasa"`
}

// HasSocio2 indica si el snapshot involucra un segundo socio real.
//
// GENERAL se representa con nil en socio_2; por eso no alcanza con comprobar
// el apuntador: un string vacío tampoco cuenta como segundo socio. El método
// existe para que los adaptadores no repitan esta comparación.
func (l Liquidacion) HasSocio2() bool {
	return l.Socio2 != nil && *l.Socio2 != ""
}

var acentosSocio = strings.NewReplacer(
	"Á", "A", "À", "A", "Ä", "A", "Â", "A",
	"É", "E", "È", "E", "Ë", "E", "Ê", "E",
	"Í", "I", "Ì", "I", "Ï", "I", "Î", "I",
	"Ó", "O", "Ò", "O", "Ö", "O", "Ô", "O",
	"Ú", "U", "Ù", "U", "Ü", "U", "Û", "U",
	"Ñ", "N",
)

// CanonicalSocio normaliza un nombre para comparar identidades sin depender
// de mayúsculas, espacios ni acentos. No reemplaza la identidad canónica por
// ID: es la comparación mínima mientras el esquema solo persiste nombres.
func CanonicalSocio(name string) string {
	return acentosSocio.Replace(strings.ToUpper(strings.TrimSpace(name)))
}

// IsPlaceholderSocio identifica valores que no representan una contraparte.
func IsPlaceholderSocio(name string) bool {
	switch CanonicalSocio(name) {
	case "", "GENERAL", "NO DEFINIDO", "S/D", "N/A", "NULL", "UNDEFINED", "-":
		return true
	}
	return false
}

// IsRealSocio indica si el nombre puede participar en un binomio 2x.
func IsRealSocio(name string) bool {
	return !IsPlaceholderSocio(name)
}

// Validate exige un snapshot 2x contablemente utilizable.
//
// Los importes firmados pueden ser positivos o negativos: lo que se valida es
// su magnitud. Una liquidación que trunca cualquiera de los dos equivalentes a
// 0.00 no tiene impacto contable real y se rechaza.
func (l Liquidacion) Validate() error {
	if strings.TrimSpace(l.HashLargo) == "" {
		return Validationf("hash_largo es obligatorio")
	}
	socio1 := CanonicalSocio(l.Socio1)
	if !IsRealSocio(socio1) {
		return Validationf("socio_1 debe ser un socio real, no %q", l.Socio1)
	}
	if l.Socio2 == nil || !IsRealSocio(*l.Socio2) {
		return Validationf("socio_2 debe ser un socio real")
	}
	socio2 := CanonicalSocio(*l.Socio2)
	if socio1 == socio2 {
		return Validationf("socio_1 y socio_2 deben ser distintos")
	}

	values := []float64{
		l.Monto1, l.Tasa1, l.ME1,
		l.Monto2, l.Tasa2, l.ME2,
	}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return Validationf("la liquidación contiene valores no finitos")
		}
	}
	if l.Tasa1 <= 0 || l.Tasa2 <= 0 {
		return Validationf("las tasas de la liquidación deben ser positivas")
	}
	if math.Abs(l.Monto1) == 0 || math.Abs(l.Monto2) == 0 {
		return Validationf("ambos lados de la liquidación deben tener importe")
	}
	if math.Abs(l.ME1) < 0.01 || math.Abs(l.ME2) < 0.01 {
		return Validationf("ambos equivalentes deben tener impacto contable de al menos 0.01")
	}
	if strings.TrimSpace(l.LoteTasa) == "" {
		return Validationf("lote_tasa es obligatorio")
	}
	return nil
}
