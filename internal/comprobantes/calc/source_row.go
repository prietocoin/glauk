package calc

import "time"

// SourceRow es la fila cruda del query de auditoría.
//
// El store la escanea sin decidir reglas; esta capa la interpreta. Los
// apuntadores conservan la diferencia entre NULL y cero de PostgreSQL.
type SourceRow struct {
	HashLargo         string
	HashCorto         string
	FechaHora         time.Time
	TimestampMsg      *int64
	Monto             float64
	Moneda            string
	Banco             string
	Titular           string
	Referencia        string
	ProcesadoIA       bool
	URLImagen         string
	Caption           *string
	EstaLiquidado     bool
	Socio1            *string
	TipoOp1           *string
	Monto1            *float64
	Tasa1             *float64
	ME1               *float64
	Socio2            *string
	TipoOp2           *string
	Monto2            *float64
	Tasa2             *float64
	ME2               *float64
	LoteTasa          *string
	LoteTasaHistorico *string
	TasasLote         []byte
	MonedaSocio1      string
	MonedaSocio2      string
	Socio1Row         []byte
	Socio2Row         []byte
	FBSocio1          string
	FBSocio2          *string
	RolSocio1         string
	RolSocio2         string
}
