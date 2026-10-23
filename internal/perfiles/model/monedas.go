package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Monedas es el mapa de reglas por código ISO que vive en perfiles_glaukov.
type Monedas map[string]Moneda

// Porcentajes es la forma canónica que consume el motor de Tasas.
type Porcentajes struct {
	Deposito float64 `json:"deposito"`
	Pago     float64 `json:"pago"`
}

// Moneda es el superset compatible con Tasas, Perfiles y Comprobantes.
// Al leer acepta las tres formas históricas y deriva las claves que falten. Al
// escribir emite siempre todas las claves, de modo que ningún consumidor pierda
// la semántica que ya usaba; los conflictos quedan para el WARN del llamador.
type Moneda struct {
	Activo             bool        `json:"activo"`
	Porcentaje         Porcentajes `json:"porcentaje"`
	Def                string      `json:"def"`
	Deposito           float64     `json:"deposito"`
	DepOp              string      `json:"dep_op"`
	Pago               float64     `json:"pago"`
	PagoOp             string      `json:"pago_op"`
	Tipo               string      `json:"tipo"`
	PolarD             string      `json:"polar_D"`
	PolarP             string      `json:"polar_P"`
	Polaridad          string      `json:"polaridad"`
	conflictos         []Conflicto
	porcentajePresente bool
}

type porcentajeEntrada struct {
	Deposito *FlexibleFloat `json:"deposito"`
	Pago     *FlexibleFloat `json:"pago"`
}

type monedaEntrada struct {
	Activo     *bool              `json:"activo"`
	Porcentaje *porcentajeEntrada `json:"porcentaje"`
	Def        string             `json:"def"`
	Deposito   *FlexibleFloat     `json:"deposito"`
	DepOp      string             `json:"dep_op"`
	Pago       *FlexibleFloat     `json:"pago"`
	PagoOp     string             `json:"pago_op"`
	Tipo       string             `json:"tipo"`
	PolarD     string             `json:"polar_D"`
	PolarP     string             `json:"polar_P"`
	Polaridad  string             `json:"polaridad"`
}

// UnmarshalJSON tolera las formas de Tasas, Perfiles y Comprobantes.
func (m *Moneda) UnmarshalJSON(data []byte) error {
	var e monedaEntrada
	if err := json.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("moneda inválida: %w", err)
	}
	deposito, pago := numeroDe(e.Deposito), numeroDe(e.Pago)
	var conflictos []Conflicto
	if e.Porcentaje != nil {
		deposito, pago = aplicarPorcentaje(e, deposito, pago, &conflictos)
	}
	def, tipo := e.Def, e.Tipo
	if def == "" {
		def = primero(tipo, "D")
	}
	if tipo == "" {
		tipo = def
	}
	// polaridad es la forma histórica de una sola polaridad: describía el
	// depósito y el pago era su contrapartida. Se usa solo si no llegó la
	// separación canónica polar_D / polar_P.
	depOp := primero(e.DepOp, e.PolarD, e.Polaridad, "+")
	pagoOp := primero(e.PagoOp, e.PolarP, contraparte(depOp))
	activo := true
	if e.Activo != nil {
		activo = *e.Activo
	}
	*m = Moneda{
		Activo: activo, Porcentaje: Porcentajes{Deposito: deposito, Pago: pago},
		Def: def, Deposito: deposito, DepOp: depOp, Pago: pago, PagoOp: pagoOp,
		Tipo: tipo, PolarD: primero(e.PolarD, depOp, "+"),
		// polaridad es el alias legacy de polar_D: si ambos llegan, manda el
		// canónico para que el JSONB no quede con dos verdades distintas.
		PolarP: primero(e.PolarP, pagoOp, "-"), Polaridad: primero(e.PolarD, e.Polaridad, depOp, "+"),
		conflictos: conflictos, porcentajePresente: e.Porcentaje != nil,
	}
	return nil
}

// aplicarPorcentaje da prioridad a la forma de Tasas y registra las diferencias.
func aplicarPorcentaje(e monedaEntrada, deposito, pago float64, conflictos *[]Conflicto) (float64, float64) {
	if e.Porcentaje.Deposito != nil {
		nuevo := numeroDe(e.Porcentaje.Deposito)
		if e.Deposito != nil && nuevo != deposito {
			*conflictos = append(*conflictos, Conflicto{Campo: "deposito", Descartado: deposito, Conservado: nuevo})
		}
		deposito = nuevo
	}
	if e.Porcentaje.Pago != nil {
		nuevo := numeroDe(e.Porcentaje.Pago)
		if e.Pago != nil && nuevo != pago {
			*conflictos = append(*conflictos, Conflicto{Campo: "pago", Descartado: pago, Conservado: nuevo})
		}
		pago = nuevo
	}
	return deposito, pago
}

// MarshalJSON emite el superset completo, sin las marcas internas de conflicto.
func (m Moneda) MarshalJSON() ([]byte, error) {
	type salida Moneda
	return json.Marshal(salida(m))
}

func numeroDe(valor *FlexibleFloat) float64 {
	if valor == nil {
		return 0
	}
	return valor.Float64()
}

func primero(valores ...string) string {
	for _, valor := range valores {
		if trimmed := strings.TrimSpace(valor); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// contraparte devuelve el signo opuesto. Se usa para migrar la polaridad
// histórica (una sola para depósito y pago) sin inventar una dirección nueva.
func contraparte(signo string) string {
	if strings.TrimSpace(signo) == "-" {
		return "+"
	}
	return "-"
}
