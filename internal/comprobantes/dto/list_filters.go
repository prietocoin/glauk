package dto

import "glauk/internal/comprobantes/domain"

// ListFilters es el contrato de query de GET /api/v1/comprobantes.
//
// Se mantiene como alias para que Fiber parsee exactamente los mismos campos
// que valida el dominio, sin duplicar nombres ni defaults.
//
// Parámetros aceptados:
//
//	socio       búsqueda por nombre; vacío = todos
//	rol         filtro por rol; vacío = todos
//	fechaInicio YYYY-MM-DD inclusiva; formato inválido = 400
//	fechaFin    YYYY-MM-DD inclusiva; formato inválido = 400
//	hash        coincidencia por hash largo o corto
//	orden       fecha_asc | fecha_desc; cualquier otro valor usa fecha_desc
//	limit       filas por página (default 100, máximo 100)
//	offset      filas omitidas desde el inicio (mínimo 0)
type ListFilters = domain.ListFilters

// NormalizeListFilters limpia y valida las fechas antes de tocar PostgreSQL.
// El handler llama a esta función una sola vez para que el store reciba un
// filtro reproducible y nunca tenga que interpretar strings de fecha.
//
// La validación es deliberadamente estricta: el panel puede enviar fechas
// vacías, pero cualquier valor no vacío debe cumplir YYYY-MM-DD. Así un error
// de tipeo se convierte en 400 y no en una consulta silenciosamente vacía.
func NormalizeListFilters(filters *ListFilters) error {
	filters.Normalize()
	return filters.Validate()
}
