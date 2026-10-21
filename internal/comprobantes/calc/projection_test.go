package calc

import (
	"testing"
	"time"

	"glauk/internal/comprobantes/domain"
)

func testPtr(value float64) *float64 { return &value }

func TestSignedDivisionYFrozenAmounts(t *testing.T) {
	if got := SignedDivision(1, 100, 0); got != 100 {
		t.Fatalf("tasa cero = %v", got)
	}
	if got := SignedDivision(-1, 100, 2); got != -50 {
		t.Fatalf("división con signo = %v", got)
	}
	row := SourceRow{
		Monto1: testPtr(-120.5), ME1: testPtr(-60.25), Tasa1: testPtr(0),
		Monto2: testPtr(40), ME2: testPtr(20), Tasa2: testPtr(4000),
	}
	t1, m1, me1, t2, m2, me2 := FrozenAmounts(row, -1, 1)
	if t1 != 1 || m1 != -120.5 || me1 != -60.25 || t2 != 4000 || m2 != 40 || me2 != 20 {
		t.Fatalf("frozen = %v %v %v %v %v %v", t1, m1, me1, t2, m2, me2)
	}
	t1, m1, me1, t2, m2, me2 = FrozenAmounts(SourceRow{}, 1, -1)
	if t1 != 1 || t2 != 1 || m1 != 0 || me1 != 0 || m2 != 0 || me2 != 0 {
		t.Fatalf("frozen vacío = %v %v %v %v %v %v", t1, m1, me1, t2, m2, me2)
	}
}

func TestCalculateAmountsLiveYCongelado(t *testing.T) {
	row := SourceRow{TasasLote: []byte(`{"COP":4000,"USDT":1,"ARS":1000}`)}
	conf1 := CurrencyConfig{Porcentaje: Spread{Deposito: 2, Pago: 3}}
	conf2 := CurrencyConfig{Porcentaje: Spread{Deposito: 0, Pago: 3}}
	t1, m1, me1, t2, m2, me2 := CalculateAmounts(
		row, false, 400000, "COP", "USDT", "ARS", conf1, conf2, "D", 1, -1, true)
	if t1 != 4080 || t2 != 4 || m1 != 400000.0/4080 || me1 != 400000.0/4080 ||
		m2 != -100000 || me2 != -100 {
		t.Fatalf("live = %v %v %v %v %v %v", t1, m1, me1, t2, m2, me2)
	}
	_, _, _, t2, m2, me2 = CalculateAmounts(
		row, false, 400000, "COP", "USDT", "ARS", conf1, conf2, "D", 1, -1, false)
	if t2 != 4 || m2 != 0 || me2 != 0 {
		t.Fatalf("sin socio2 = %v %v %v", t2, m2, me2)
	}
	frozen := SourceRow{Monto1: testPtr(10), ME1: testPtr(5), Tasa1: testPtr(4000)}
	t1, m1, me1, _, m2, me2 = CalculateAmounts(
		frozen, true, 999, "COP", "USDT", "USDT", conf1, conf2, "D", 1, -1, false)
	if t1 != 4000 || m1 != 10 || me1 != 5 || m2 != 0 || me2 != 0 {
		t.Fatalf("congelado = %v %v %v %v %v", t1, m1, me1, m2, me2)
	}
}

func TestProjectRowLive(t *testing.T) {
	socio1 := []byte(`{"nombre":"FUNDA","rol":"SOCIO","moneda_base":"USDT",` +
		`"monedas":{"COP":{"tipo":"D","polar_D":"+","polar_P":"-",` +
		`"porcentaje":{"deposito":"2","pago":"3"}}}}`)
	socio2 := []byte(`{"nombre":"SOCIO2","moneda_base":"ARS",` +
		`"monedas":{"COP":{"tipo":"P","polar_D":"-","polar_P":"+"}}}`)
	fundda := Profile{Nombre: "FUNDDA", Monedas: map[string]CurrencyConfig{
		"COP": {Tipo: "D", PolarD: "+", PolarP: "-"},
	}}
	row := SourceRow{
		HashLargo: "h1", HashCorto: "c1", FechaHora: time.Unix(10, 0),
		Monto: 400000, Moneda: "cop", Banco: "B", Titular: "T",
		Referencia: "R", ProcesadoIA: true, URLImagen: "u",
		FBSocio1: " FUNDA ", FBSocio2: socioPtr(" SOCIO2 "),
		RolSocio1: "SOCIO", MonedaSocio1: "usdt", MonedaSocio2: "ars",
		TasasLote: []byte(`{"COP":4000,"USDT":1,"ARS":1000}`),
		Socio1Row: socio1, Socio2Row: socio2, LoteTasa: socioPtr("T052"),
	}
	got := ProjectRow(row, fundda)
	if got.Monto != 400000 || got.Moneda != "cop" || got.NombreSocio1 != "FUNDA" ||
		got.NombreSocio2 == nil || *got.NombreSocio2 != "SOCIO2" {
		t.Fatalf("cabecera = %+v", got)
	}
	if got.TipoOp1 != "D-COP" || got.Tasa1 != 4080 || got.Monto1 != 400000.0/4080 ||
		got.ME1 != 400000.0/4080 || got.Tasa2 != 4 || got.Monto2 != -100000 ||
		got.ME2 != -100 || got.LoteTasaAsignado != "T052" {
		t.Fatalf("montos = %+v", got)
	}
	if got.M1Socio != got.Monto1 || got.M2Socio != got.Monto2 || got.TimestampMsg != nil {
		t.Fatalf("proyección = %+v", got)
	}
}

func TestProjectRowLiquidado(t *testing.T) {
	row := SourceRow{
		HashLargo: "h2", Monto: -100, Moneda: "COP", EstaLiquidado: true,
		Socio1: socioPtr(" FUNDA "), TipoOp1: socioPtr("P-COP"),
		Monto1: testPtr(100), Tasa1: testPtr(4000), ME1: testPtr(100),
		Monto2: testPtr(-50), Tasa2: testPtr(1), ME2: testPtr(-50),
		RolSocio1: "PROVEEDOR", LoteTasaHistorico: socioPtr("T049"),
	}
	got := ProjectRow(row, Profile{})
	if got.NombreSocio1 != "FUNDA" || got.NombreSocio2 != nil || got.TipoOp1 != "P-COP" {
		t.Fatalf("cabecera = %+v", got)
	}
	if got.Monto != 100 || got.Tasa1 != 4000 || got.Monto1 != -100 || got.ME1 != -100 ||
		got.Tasa2 != 1 || got.Monto2 != -50 || got.ME2 != -50 || got.LoteTasaAsignado != "T049" {
		t.Fatalf("congelado = %+v", got)
	}
}

func TestProjectionHelpers(t *testing.T) {
	if savedLetter(nil) != "" || savedLetter(socioPtr("A-COP")) != "A" {
		t.Fatal("savedLetter")
	}
	if assignedLot(SourceRow{}) != "T001" ||
		assignedLot(SourceRow{LoteTasaHistorico: socioPtr(" T049 ")}) != "T049" ||
		assignedLot(SourceRow{LoteTasa: socioPtr("T052"), LoteTasaHistorico: socioPtr("T049")}) != "T052" {
		t.Fatal("assignedLot")
	}
	if resolveSocio1(SourceRow{}, true) != "GENERAL" ||
		resolveSocio1(SourceRow{Socio1: socioPtr(" "), FBSocio1: " FALLBACK "}, true) != "GENERAL" ||
		resolveSocio1(SourceRow{FBSocio1: " FALLBACK "}, false) != "FALLBACK" {
		t.Fatal("resolveSocio1")
	}
	if resolveSocio2(SourceRow{FBSocio2: socioPtr(" ")}, false) != nil ||
		resolveSocio2(SourceRow{Socio2: socioPtr(" DOS "), FBSocio2: socioPtr("FB")}, true) == nil {
		t.Fatal("resolveSocio2")
	}
	conf := CurrencyConfig{Tipo: "P"}
	if operationLetter(SourceRow{RolSocio1: "proveedor"}, conf, "X", "USDT", "COP") != "P" ||
		operationLetter(SourceRow{RolSocio1: "PROVEEDOR", TipoOp1: socioPtr("D-COP")}, conf, "X", "USDT", "COP") != "D" ||
		operationLetter(SourceRow{RolSocio1: "PROVEEDOR", TipoOp1: socioPtr("A-COP")}, conf, "X", "USDT", "COP") != "P" ||
		operationLetter(SourceRow{RolSocio1: "SOCIO", TipoOp1: socioPtr("P-COP")}, conf, "X", "USDT", "COP") != "P" ||
		operationLetter(SourceRow{RolSocio1: "SOCIO", MonedaSocio1: "COP"}, conf, "FUNDA", "COP", "COP") != "A" ||
		operationLetter(SourceRow{RolSocio1: "SOCIO", MonedaSocio1: "COP"}, CurrencyConfig{}, "GENERAL", "COP", "COP") != "D" {
		t.Fatal("operationLetter")
	}
	if domain.NormalizeCurrency(" usdt ", "") != "USDT" {
		t.Fatal("normalización de divisa")
	}
}
