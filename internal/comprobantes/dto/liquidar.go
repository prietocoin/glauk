package dto

import (
	"strings"

	"glauk/internal/comprobantes/domain"
)

const (
	DefaultLoteTasa = "T052"
	DefaultSocio    = "GENERAL"
	DefaultTipoOp   = "D-USDT"
	DefaultTipo     = "P"
)

// LiquidarInput es el payload de POST /api/v1/comprobantes/liquidar.
type LiquidarInput struct {
	HashLargo string        `json:"hash_largo"`
	Socio1    string        `json:"socio_1"`
	TipoOp1   string        `json:"tipo_op1"`
	Monto1    OptionalFloat `json:"monto_1"`
	Tasa1     OptionalFloat `json:"tasa_1"`
	ME1       OptionalFloat `json:"me1"`
	Socio2    *string       `json:"socio_2"`
	TipoOp2   string        `json:"tipo_op2"`
	Monto2    OptionalFloat `json:"monto_2"`
	Tasa2     OptionalFloat `json:"tasa_2"`
	ME2       OptionalFloat `json:"me2"`
	LoteTasa  string        `json:"lote_tasa"`
}

// ToLiquidacion normaliza el payload con los defaults del legado.
func (i LiquidarInput) ToLiquidacion() domain.Liquidacion {
	socio2 := DefaultSocio
	if i.Socio2 != nil && strings.TrimSpace(*i.Socio2) != "" {
		socio2 = strings.TrimSpace(*i.Socio2)
	}
	tipo1 := firstNonEmpty(i.TipoOp1, DefaultTipoOp)
	tipo2 := firstNonEmpty(i.TipoOp2, tipo1)

	return domain.Liquidacion{
		HashLargo: strings.TrimSpace(i.HashLargo),
		Socio1:    firstNonEmpty(i.Socio1, DefaultSocio),
		TipoOp1:   tipo1,
		Monto1:    i.Monto1.Or(0),
		Tasa1:     i.Tasa1.Or(1),
		ME1:       i.ME1.Or(0),
		Socio2:    &socio2,
		TipoOp2:   tipo2,
		Monto2:    i.Monto2.Or(0),
		Tasa2:     i.Tasa2.Or(1),
		ME2:       i.ME2.Or(0),
		LoteTasa:  firstNonEmpty(i.LoteTasa, DefaultLoteTasa),
	}
}

func firstNonEmpty(values ...string) string {
	return domain.FirstNonEmpty(values...)
}
