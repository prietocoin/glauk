package domain

import (
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

const (
	// DefaultListLimit conserva el tope histórico de 100 filas cuando el
	// cliente no solicita paginación explícita.
	DefaultListLimit = 100
	// MaxListLimit protege PostgreSQL de páginas arbitrariamente grandes.
	MaxListLimit = 100
)

// ListFilters contiene los parámetros de GET /api/v1/comprobantes.
type ListFilters struct {
	Socio       string `query:"socio"`
	Rol         string `query:"rol"`
	FechaInicio string `query:"fechaInicio"`
	FechaFin    string `query:"fechaFin"`
	Hash        string `query:"hash"`
	Orden       string `query:"orden"`
	Limit       int    `query:"limit"`
	Offset      int    `query:"offset"`
}

// Normalize limpia los filtros, aplica el orden histórico por defecto y acota
// la paginación al rango soportado por la consulta.
func (f *ListFilters) Normalize() {
	f.Socio = strings.TrimSpace(f.Socio)
	f.Rol = strings.TrimSpace(f.Rol)
	f.FechaInicio = strings.TrimSpace(f.FechaInicio)
	f.FechaFin = strings.TrimSpace(f.FechaFin)
	f.Hash = strings.TrimSpace(f.Hash)
	f.Orden = strings.TrimSpace(f.Orden)
	if f.Orden != "fecha_asc" && f.Orden != "fecha_desc" {
		f.Orden = "fecha_desc"
	}
	if f.Limit <= 0 {
		f.Limit = DefaultListLimit
	}
	if f.Limit > MaxListLimit {
		f.Limit = MaxListLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// Validate exige fechas ISO cuando vienen informadas.
func (f ListFilters) Validate() error {
	if err := validateDate("fechaInicio", f.FechaInicio); err != nil {
		return err
	}
	return validateDate("fechaFin", f.FechaFin)
}

// OrderAsc indica si la consulta debe ordenar de forma ascendente.
func (f ListFilters) OrderAsc() bool {
	return f.Orden == "fecha_asc"
}

func validateDate(field, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(dateLayout, value); err != nil {
		return Validationf("%s debe tener formato YYYY-MM-DD", field)
	}
	return nil
}
