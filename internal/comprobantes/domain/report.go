package domain

import (
	"strings"
	"time"
)

// ReportFilters alimenta el modal del módulo Reportes.
type ReportFilters struct {
	Roles     []string
	Entidades []string
	Hashes    []string
}

// ReportQuery es el filtro normalizado de movimientos y resumen.
type ReportQuery struct {
	Rol     string
	Entidad string
	Hash    string
	Desde   time.Time
	Hasta   time.Time
	Orden   string
	Limite  int
}

// Normalize limpia el filtro y fija orden y límite seguros.
func (q *ReportQuery) Normalize() {
	q.Rol = strings.TrimSpace(q.Rol)
	if strings.EqualFold(q.Rol, "TODOS") {
		q.Rol = ""
	}
	q.Entidad = strings.TrimSpace(q.Entidad)
	q.Hash = strings.TrimSpace(q.Hash)
	q.Orden = strings.TrimSpace(q.Orden)
	if q.Orden != "fecha_asc" && q.Orden != "fecha_desc" {
		q.Orden = "fecha_desc"
	}
	if q.Limite <= 0 {
		q.Limite = 100
	}
	if q.Limite > 500 {
		q.Limite = 500
	}
}

// ReportRow es una fila de auditoría ya proyectada para Reportes.
type ReportRow struct {
	HashLargo  string
	HashCorto  string
	Fecha      time.Time
	Socio      string
	Rol        string
	Moneda     string
	Monto      float64
	TipoOp     string
	Banco      string
	Referencia string
	Estado     string
}

// ReportSummary es el agregado que puede aparecer en un reporte de texto.
type ReportSummary struct {
	Total      int
	MontoTotal float64
	PorMoneda  map[string]float64
	Desde      time.Time
	Hasta      time.Time
}
