package store

import (
	"github.com/jackc/pgx/v5"

	"glauk/internal/comprobantes/calc"
)

// ScanRows materializa las 34 columnas del listado más el total del universo
// filtrado (COUNT(*) OVER()) en filas crudas.
func ScanRows(rows pgx.Rows) ([]calc.SourceRow, int, error) {
	defer rows.Close()
	items := make([]calc.SourceRow, 0)
	total := 0
	for rows.Next() {
		var row calc.SourceRow
		var totalRegistros int64
		err := rows.Scan(
			&row.HashLargo, &row.HashCorto, &row.FechaHora, &row.TimestampMsg,
			&row.Monto, &row.Moneda, &row.Banco, &row.Titular, &row.Referencia,
			&row.ProcesadoIA, &row.URLImagen, &row.Caption, &row.EstaLiquidado,
			&row.Socio1, &row.TipoOp1, &row.Monto1, &row.Tasa1, &row.ME1,
			&row.Socio2, &row.TipoOp2, &row.Monto2, &row.Tasa2, &row.ME2,
			&row.LoteTasa, &row.LoteTasaHistorico, &row.TasasLote,
			&row.MonedaSocio1, &row.MonedaSocio2, &row.Socio1Row, &row.Socio2Row,
			&row.FBSocio1, &row.FBSocio2, &row.RolSocio1, &row.RolSocio2,
			&totalRegistros,
		)
		if err != nil {
			return nil, 0, err
		}
		total = int(totalRegistros)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
