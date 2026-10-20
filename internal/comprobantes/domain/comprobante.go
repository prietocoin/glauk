package domain

import "time"

// Comprobante es la proyección de auditoría que consume el panel.
//
// Los importes pueden ser positivos o negativos: el signo ya incorpora la
// polaridad del contrato y nunca debe recalcularse en el frontend.
type Comprobante struct {
	HashLargo            string    `json:"hash_largo"`
	HashCorto            string    `json:"hash_corto"`
	FechaHoraComprobante time.Time `json:"fecha_hora_comprobante"`
	TimestampMsg         *int64    `json:"timestamp_msg"`
	Monto                float64   `json:"monto"`
	Moneda               string    `json:"moneda"`
	Banco                string    `json:"banco"`
	Titular              string    `json:"titular"`
	Referencia           string    `json:"referencia"`
	ProcesadoIA          bool      `json:"procesado_ia"`
	URLImagen            string    `json:"url_imagen"`
	Caption              *string   `json:"caption"`

	NombreSocio1 string  `json:"nombre_socio_1"`
	MonedaSocio1 string  `json:"moneda_socio_1"`
	TipoOp1      string  `json:"tipo_op1"`
	Monto1       float64 `json:"monto_1"`
	Tasa1        float64 `json:"tasa_1"`
	ME1          float64 `json:"me1"`

	NombreSocio2 *string `json:"nombre_socio_2"`
	MonedaSocio2 string  `json:"moneda_socio_2"`
	TipoOp2      string  `json:"tipo_op2"`
	Monto2       float64 `json:"monto_2"`
	Tasa2        float64 `json:"tasa_2"`
	ME2          float64 `json:"me2"`

	M1Socio          float64 `json:"m1_socio"`
	M2Socio          float64 `json:"m2_socio"`
	LoteTasaAsignado string  `json:"lote_tasa_asignado"`
}
