package adapters

import (
	"context"

	comprobantesservice "glauk/internal/comprobantes/service"
	"glauk/internal/reportes"
)

// MovimientosComprobantes adapta el puerto de auditoría de Comprobantes al
// contrato que consume reportes.
type MovimientosComprobantes struct {
	reader comprobantesservice.ReportReader
}

// NewMovimientosComprobantes construye el adaptador sobre el lector inyectado.
func NewMovimientosComprobantes(reader comprobantesservice.ReportReader) *MovimientosComprobantes {
	return &MovimientosComprobantes{reader: reader}
}

// Filtros proyecta roles, entidades y hashes sin exponer tipos de Comprobantes.
func (a *MovimientosComprobantes) Filtros(ctx context.Context, rol string) (reportes.Filtros, error) {
	filtros, err := a.reader.FiltrosReporte(ctx, rol)
	if err != nil {
		return reportes.Filtros{}, err
	}
	return reportes.Filtros{
		Roles:     sinNulos(filtros.Roles),
		Entidades: sinNulos(filtros.Entidades),
		Hashes:    sinNulos(filtros.Hashes),
	}, nil
}

// Movimientos traduce la consulta y proyecta cada fila al modelo de reportes.
func (a *MovimientosComprobantes) Movimientos(ctx context.Context,
	q reportes.ConsultaMovimientos) ([]reportes.Movimiento, error) {
	rows, err := a.reader.MovimientosReporte(ctx, consultaDe(q))
	if err != nil {
		return nil, err
	}
	out := make([]reportes.Movimiento, 0, len(rows))
	for _, row := range rows {
		out = append(out, reportes.Movimiento{
			HashLargo: row.HashLargo, HashCorto: row.HashCorto, Fecha: row.Fecha,
			Socio: row.Socio, Rol: row.Rol, Moneda: row.Moneda, Monto: row.Monto,
			TipoOp: row.TipoOp, Banco: row.Banco, Referencia: row.Referencia,
			Estado: row.Estado,
		})
	}
	return out, nil
}

// Resumen proyecta el agregado de Comprobantes al modelo de reportes.
func (a *MovimientosComprobantes) Resumen(ctx context.Context,
	q reportes.ConsultaMovimientos) (reportes.Resumen, error) {
	resumen, err := a.reader.ResumenReporte(ctx, consultaDe(q))
	if err != nil {
		return reportes.Resumen{}, err
	}
	porMoneda := resumen.PorMoneda
	if porMoneda == nil {
		porMoneda = map[string]float64{}
	}
	return reportes.Resumen{
		Total: resumen.Total, MontoTotal: resumen.MontoTotal,
		PorMoneda: porMoneda, Desde: resumen.Desde, Hasta: resumen.Hasta,
	}, nil
}

func consultaDe(q reportes.ConsultaMovimientos) comprobantesservice.ReportQuery {
	return comprobantesservice.ReportQuery{
		Rol: q.Rol, Entidad: q.Entidad, Hash: q.Hash,
		Desde: q.Desde, Hasta: q.Hasta, Orden: q.Orden, Limite: q.Limite,
	}
}

func sinNulos(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
