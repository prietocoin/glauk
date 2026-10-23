package calc

import "testing"

func TestContieneIndicadorAbono(t *testing.T) {
	casos := []struct {
		caption string
		quiero  bool
	}{
		{"", false},
		{"pago nequi", false},
		{"transferencia recibida", false},
		{"abono a capital", true},
		{"ABONO", true},
		{"Abonaré mañana", true},
		{"abonar cuota", true},
		{"  abona  ", true},
	}
	for _, caso := range casos {
		if got := contieneIndicadorAbono(caso.caption); got != caso.quiero {
			t.Fatalf("contieneIndicadorAbono(%q) = %v, quiero %v", caso.caption, got, caso.quiero)
		}
	}
}

func TestNatureForAbonoIRISCOP(t *testing.T) {
	contratoD := CurrencyConfig{Tipo: "D", PolarD: "-", PolarP: "+"}
	contratoP := CurrencyConfig{Tipo: "P", PolarD: "+", PolarP: "-"}
	casos := []struct {
		nombre  string
		tipo    string
		socio1  string
		socio2  string
		divisa  string
		caption string
		conf    CurrencyConfig
		quiero  string
	}{
		{"iris sin caption revierte a D", "A", "IRIS", "GENERAL", "COP", "pago nequi", contratoD, "D"},
		{"iris sin caption revierte a P", "A", "IRIS", "GENERAL", "COP", "", contratoP, "P"},
		{"iris con caption conserva A", "A", "IRIS", "GENERAL", "COP", "abono a capital", contratoD, "A"},
		{"iris como socio 2", "A", "GENERAL", "IRIS", "COP", "pago nequi", contratoD, "D"},
		{"iris con caption como socio 2", "A", "GENERAL", "IRIS", "COP", "ABONO recibido", contratoD, "A"},
		{"iris en otra divisa conserva A", "A", "IRIS", "GENERAL", "USDT", "pago nequi", contratoD, "A"},
		{"otro socio en COP conserva A", "A", "FUNDABLOCK", "GENERAL", "COP", "pago nequi", contratoD, "A"},
		{"tipo no A no se toca", "D", "IRIS", "GENERAL", "COP", "pago nequi", contratoD, "D"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			got := NatureForAbono(caso.tipo, caso.socio1, caso.socio2, caso.divisa, caso.caption, caso.conf)
			if got != caso.quiero {
				t.Fatalf("NatureForAbono = %q, quiero %q", got, caso.quiero)
			}
		})
	}
}

// TestSnapshotIRISCOPSinCaptionUsaContrato es la regresión del abono automático:
// la coincidencia de moneda base ya no convierte el pago en A; se aplica el tipo
// contractual y su polar_D.
func TestSnapshotIRISCOPSinCaptionUsaContrato(t *testing.T) {
	iris := Profile{
		Nombre: "IRIS", MonedaBase: "COP",
		Monedas: map[string]CurrencyConfig{
			"COP": {Tipo: "D", PolarD: "-", PolarP: "+"},
		},
	}
	lote := RateLot{IDTasa: "T052", Tasas: map[string]float64{"COP": 4000, "USDT": 1}}

	sinCaption := CalcularSnapshotFinanciero(Raw{
		HashLargo: "iris-1", Monto: 400000, Moneda: "COP",
		TipoManual: "A", Caption: "pago nequi", IDTasa: "T052",
	}, iris, Profile{}, lote, Profile{})

	if sinCaption.TipoOp1 != "D-COP" || sinCaption.TipoOp2 != "D-COP" {
		t.Fatalf("naturaleza = %q/%q, quiero D-COP", sinCaption.TipoOp1, sinCaption.TipoOp2)
	}
	if sinCaption.Monto1 >= 0 {
		t.Fatalf("debe aplicar polar_D=- del contrato, monto1 = %v", sinCaption.Monto1)
	}

	conCaption := CalcularSnapshotFinanciero(Raw{
		HashLargo: "iris-2", Monto: 400000, Moneda: "COP",
		TipoManual: "A", Caption: "abono a capital", IDTasa: "T052",
	}, iris, Profile{}, lote, Profile{})

	if conCaption.TipoOp1 != "A-COP" || conCaption.TipoOp2 != "A-COP" {
		t.Fatalf("naturaleza = %q/%q, quiero A-COP", conCaption.TipoOp1, conCaption.TipoOp2)
	}
	if conCaption.Monto1 <= 0 {
		t.Fatalf("el abono explícito debe usar signos de abono, monto1 = %v", conCaption.Monto1)
	}
}

// TestProjectRowIRISCOPCaptionGobiernaLaNaturaleza cubre la lectura viva: un A
// guardado en comprobantes_liq o inferido por moneda base no sobrevive si el
// caption no declara abono.
func TestProjectRowIRISCOPCaptionGobiernaLaNaturaleza(t *testing.T) {
	perfil := []byte(`{"nombre":"IRIS","rol":"SOCIO","moneda_base":"COP",` +
		`"monedas":{"COP":{"tipo":"D","polar_D":"-","polar_P":"+","porcentaje":{"deposito":0,"pago":0}}}}`)
	base := SourceRow{
		HashLargo: "iris-p", Monto: 400000, Moneda: "COP",
		FBSocio1: "IRIS", MonedaSocio1: "COP", RolSocio1: "SOCIO",
		TasasLote: []byte(`{"COP":4000,"USDT":1}`),
		Socio1Row: perfil,
	}
	caption := func(value string) *string { return &value }

	casos := []struct {
		nombre     string
		tipoOp1    *string
		caption    *string
		quieroTipo string
		quieroPos  bool
	}{
		{"pago sin caption no es abono", nil, caption("pago nequi"), "D-COP", false},
		{"caption vacío no es abono", nil, caption(""), "D-COP", false},
		{"abono explícito se conserva", caption("A-COP"), caption("abono a capital"), "A-COP", true},
		{"P guardado con caption no se promueve", caption("P-COP"), caption("abono a capital"), "D-COP", false},
		{"A guardado sin caption se revierte", caption("A-COP"), caption("transferencia"), "D-COP", false},
		{"A guardado con caption se conserva", caption("A-COP"), caption("ABONO"), "A-COP", true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			row := base
			row.TipoOp1 = caso.tipoOp1
			row.Caption = caso.caption
			got := ProjectRow(row, Profile{})
			if got.TipoOp1 != caso.quieroTipo {
				t.Fatalf("tipo = %q, quiero %q", got.TipoOp1, caso.quieroTipo)
			}
			if (got.Monto1 > 0) != caso.quieroPos {
				t.Fatalf("monto1 = %v, signo inesperado", got.Monto1)
			}
		})
	}
}
