package domain

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
