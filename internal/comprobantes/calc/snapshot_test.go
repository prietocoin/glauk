package calc

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"glauk/internal/comprobantes/domain"
)

func socioPtr(name string) *string { return &name }

func goldenLiq(hash, socio1, tipo string, monto1, tasa1, me1 float64,
	socio2 *string, monto2, tasa2, me2 float64, lote string) domain.Liquidacion {
	return domain.Liquidacion{
		HashLargo: hash, Socio1: socio1, TipoOp1: tipo,
		Monto1: monto1, Tasa1: tasa1, ME1: me1,
		Socio2: socio2, TipoOp2: tipo,
		Monto2: monto2, Tasa2: tasa2, ME2: me2, LoteTasa: lote,
	}
}

func TestCalcularSnapshotFinancieroGolden(t *testing.T) {
	fundda := Profile{
		Nombre: "FUNDDA", MonedaBase: "USDT",
		Monedas: map[string]CurrencyConfig{
			"COP": {Polaridad: "+", Porcentaje: Spread{Deposito: 2, Pago: 3}},
		},
	}
	cases := []struct {
		name string
		raw  Raw
		s1   Profile
		s2   Profile
		lote RateLot
		fund Profile
		want domain.Liquidacion
	}{
		{
			name: "deposito_estandar",
			raw:  Raw{HashLargo: "h1", Monto: 400000, Moneda: "COP", TipoManual: "D", IDTasa: "T052"},
			s1: Profile{Nombre: "FUNDDA", MonedaBase: "USDT", Monedas: map[string]CurrencyConfig{
				"COP": {Polaridad: "+"},
			}},
			s2:   Profile{Nombre: "GENERAL", MonedaBase: "USDT"},
			lote: RateLot{IDTasa: "T052", Tasas: map[string]float64{"COP": 4000, "USDT": 1}},
			want: goldenLiq("h1", "FUNDDA", "D-COP", 100, 4000, 100, nil, 0, 1, 0, "T052"),
		},
		{
			name: "abono_misma_moneda",
			raw:  Raw{HashLargo: "h2", Monto: 1500, Moneda: "USDT", TipoManual: "D", IDTasa: "T052"},
			s1: Profile{Nombre: "FUNDABLOCK", MonedaBase: "USDT", Monedas: map[string]CurrencyConfig{
				"USDT": {Polaridad: "+"},
			}},
			s2:   Profile{Nombre: "GENERAL", MonedaBase: "USDT"},
			lote: RateLot{IDTasa: "T052", Tasas: map[string]float64{"USDT": 1}},
			want: goldenLiq("h2", "FUNDABLOCK", "A-USDT", 1500, 1, 1500, nil, 0, 1, 0, "T052"),
		},
		{
			name: "polaridad_negativa",
			raw:  Raw{HashLargo: "h3", Monto: 400000, Moneda: "COP", TipoManual: "D", IDTasa: "T052"},
			s1: Profile{Nombre: "SOCIO_NEG", MonedaBase: "USDT", Monedas: map[string]CurrencyConfig{
				"COP": {Polaridad: "-"},
			}},
			s2:   Profile{Nombre: "GENERAL", MonedaBase: "USDT"},
			lote: RateLot{IDTasa: "T052", Tasas: map[string]float64{"COP": 4000, "USDT": 1}},
			want: goldenLiq("h3", "SOCIO_NEG", "D-COP", -100, 4000, -100, nil, 0, 1, 0, "T052"),
		},
		{
			name: "herencia_fundda",
			raw:  Raw{HashLargo: "h4", Monto: 1000000, Moneda: "COP", TipoManual: "D", IDTasa: "T052"},
			s1: Profile{Nombre: "SOCIO_HEREDA", MonedaBase: "USDT", Herencia: true,
				Monedas: map[string]CurrencyConfig{"COP": {Polaridad: "-"}}},
			s2: Profile{Nombre: "SOCIO_ANCLA", MonedaBase: "USDT",
				Monedas: map[string]CurrencyConfig{"COP": {Polaridad: "+"}}},
			lote: RateLot{IDTasa: "T052", Tasas: map[string]float64{"COP": 1000, "USDT": 1}},
			fund: fundda,
			want: goldenLiq("h4", "SOCIO_HEREDA", "D-COP", -980.39, 1020, -980.39,
				socioPtr("SOCIO_ANCLA"), 1000, 1000, 1000, "T052"),
		},
		{
			name: "pago_con_spread",
			raw:  Raw{HashLargo: "h5", Monto: 1000000, Moneda: "COP", TipoManual: "P", IDTasa: "T052"},
			s1: Profile{Nombre: "SOCIO_PAGO", MonedaBase: "USDT", Monedas: map[string]CurrencyConfig{
				"COP": {Polaridad: "+", Porcentaje: Spread{Deposito: 2, Pago: 3}},
			}},
			s2:   Profile{Nombre: "GENERAL", MonedaBase: "USDT"},
			lote: RateLot{IDTasa: "T052", Tasas: map[string]float64{"COP": 1000, "USDT": 1}},
			want: goldenLiq("h5", "SOCIO_PAGO", "P-COP", -1030.92, 970, -1030.92, nil, 0, 1, 0, "T052"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CalcularSnapshotFinanciero(tc.raw, tc.s1, tc.s2, tc.lote, tc.fund)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("snapshot distinto (-want +got):\n%s", diff)
			}
		})
	}
}
