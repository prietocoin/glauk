package calc

import (
	"encoding/json"
	"testing"
	"time"

	"glauk/internal/comprobantes/domain"
)

func testPtr(value float64) *float64 { return &value }

func TestSignedDivision(t *testing.T) {
	if got := SignedDivision(1, 100, 0); got != 100 {
		t.Fatalf("tasa cero = %v", got)
	}
	if got := SignedDivision(-1, 100, 2); got != -50 {
		t.Fatalf("división con signo = %v", got)
	}
}

func TestCalculateAmountsLive(t *testing.T) {
	row := SourceRow{TasasLote: []byte(`{"COP":4000,"USDT":1,"ARS":1000}`)}
	cfg1 := SocioConfig{Active: true, Moneda: "USDT",
		Conf: CurrencyConfig{Porcentaje: Spread{Deposito: 2, Pago: 3}}}
	cfg2 := SocioConfig{Active: true, Moneda: "ARS",
		Conf: CurrencyConfig{Porcentaje: Spread{Deposito: 0, Pago: 3}}}
	t1, m1, me1, t2, m2, me2 := CalculateAmounts(
		row, 400000, "COP", cfg1, cfg2, "D", 1, -1, true)
	if t1 != 4080 || t2 != 4 || m1 != 400000.0/4080 || me1 != 400000.0/4080 ||
		m2 != -100000 || me2 != -100 {
		t.Fatalf("live = %v %v %v %v %v %v", t1, m1, me1, t2, m2, me2)
	}
	_, _, _, t2, m2, me2 = CalculateAmounts(
		row, 400000, "COP", cfg1, cfg2, "D", 1, -1, false)
	if t2 != 1 || m2 != 0 || me2 != 0 {
		t.Fatalf("sin socio2 = %v %v %v", t2, m2, me2)
	}
}

func TestProjectRowLive(t *testing.T) {
	socio1 := []byte(`{"nombre":"FUNDA","rol":"SOCIO","moneda_base":"USDT",` +
		`"monedas":{"COP":{"tipo":"D","polaridad":"+",` +
		`"porcentaje":{"deposito":"2","pago":"3"}}}}`)
	socio2 := []byte(`{"nombre":"SOCIO2","moneda_base":"ARS",` +
		`"monedas":{"COP":{"tipo":"P","polaridad":"-"}}}`)
	fundda := Profile{Nombre: "FUNDDA", Monedas: map[string]CurrencyConfig{
		"COP": {Tipo: "D", Polaridad: "+"},
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

// TestProjectRowLiquidado verifica que una fila ya liquidada se siga proyectando
// en vivo: el socio sale de comprobantes_liq, pero el signo y las equivalencias
// se recalculan con su contrato actual, no con los importes almacenados.
func TestProjectRowLiquidado(t *testing.T) {
	perfil1 := []byte(`{"nombre":"FUNDA","rol":"SOCIO","moneda_base":"USDT",` +
		`"monedas":{"COP":{"tipo":"D","polar_D":"-","porcentaje":{"deposito":2}}}}`)
	row := SourceRow{
		HashLargo: "h2", Monto: -100, Moneda: "COP", EstaLiquidado: true,
		Socio1: socioPtr(" FUNDA "), TipoOp1: socioPtr("D-COP"),
		Monto1: testPtr(100), Tasa1: testPtr(4000), ME1: testPtr(100),
		Monto2: testPtr(-50), Tasa2: testPtr(1), ME2: testPtr(-50),
		RolSocio1: "SOCIO", MonedaSocio1: "USDT", MonedaSocio2: "USDT",
		LoteTasaHistorico: socioPtr("T049"),
		TasasLote:         []byte(`{"COP":4000,"USDT":1}`),
		Socio1Row:         perfil1,
	}
	got := ProjectRow(row, Profile{})
	if got.NombreSocio1 != "FUNDA" || got.NombreSocio2 != nil || got.TipoOp1 != "D-COP" {
		t.Fatalf("cabecera = %+v", got)
	}
	if got.Monto != 100 || got.Tasa1 != 4080 || got.Monto2 != 0 || got.ME2 != 0 ||
		got.LoteTasaAsignado != "T049" {
		t.Fatalf("proyección viva = %+v", got)
	}
	if got.Monto1 >= 0 || got.ME1 != got.Monto1 {
		t.Fatalf("el signo debe seguir polar_D=- del perfil: %+v", got)
	}
	if got.Monto1 == 100 || got.ME1 == 100 {
		t.Fatalf("no debe reutilizar el importe almacenado: %+v", got)
	}
}

// TestProjectRowDepositoFUNDDAPolaridadNegativa exige que FUNDDA como socio 2
// se proyecte con su polar_D, incluso si comprobantes_liq guardó el signo
// opuesto. Los valores persistidos no bloquean la lectura viva.
func TestProjectRowDepositoFUNDDAPolaridadNegativa(t *testing.T) {
	perfil1 := []byte(`{"nombre":"SOCIO_A","rol":"SOCIO","moneda_base":"USDT",` +
		`"monedas":{"COP":{"tipo":"D","polar_D":"+","porcentaje":{"deposito":2}}}}`)
	perfil2 := []byte(`{"nombre":"FUNDDA","rol":"ADMIN","moneda_base":"USDT",` +
		`"monedas":{"COP":{"tipo":"D","polar_D":"-","porcentaje":{"deposito":0}}}}`)
	row := SourceRow{
		HashLargo: "h3", Monto: 400000, Moneda: "COP", EstaLiquidado: true,
		Socio1: socioPtr("SOCIO_A"), Socio2: socioPtr("FUNDDA"),
		TipoOp1: socioPtr("D-COP"), TipoOp2: socioPtr("D-COP"),
		Monto1: testPtr(98.04), Tasa1: testPtr(4080), ME1: testPtr(98.04),
		Monto2: testPtr(200), Tasa2: testPtr(4000), ME2: testPtr(200),
		RolSocio1: "SOCIO", MonedaSocio1: "USDT", MonedaSocio2: "USDT",
		LoteTasa:  socioPtr("T052"),
		TasasLote: []byte(`{"COP":4000,"USDT":1}`),
		Socio1Row: perfil1, Socio2Row: perfil2,
	}
	got := ProjectRow(row, Profile{})
	if got.NombreSocio2 == nil || *got.NombreSocio2 != "FUNDDA" {
		t.Fatalf("socio 2 = %+v", got)
	}
	if got.Monto1 <= 0 || got.Monto2 >= 0 {
		t.Fatalf("FUNDDA debe proyectarse negativa por polar_D=-: %+v", got)
	}
	if got.Monto2 != -100 || got.ME2 != -100 || got.Tasa2 != 4000 {
		t.Fatalf("importe vivo de FUNDDA = %+v", got)
	}
	if got.M2Socio != got.Monto2 || got.M2Socio == 200 {
		t.Fatalf("m_socio debe seguir la proyección viva: %+v", got)
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
	if operationLetter(SourceRow{RolSocio1: "proveedor"}, conf, "X", "", "USDT", "COP") != "P" ||
		operationLetter(SourceRow{RolSocio1: "PROVEEDOR", TipoOp1: socioPtr("D-COP")}, conf, "X", "", "USDT", "COP") != "D" ||
		operationLetter(SourceRow{RolSocio1: "PROVEEDOR", TipoOp1: socioPtr("A-COP")}, conf, "X", "", "USDT", "COP") != "P" ||
		operationLetter(SourceRow{RolSocio1: "SOCIO", TipoOp1: socioPtr("P-COP")}, conf, "X", "", "USDT", "COP") != "P" ||
		operationLetter(SourceRow{RolSocio1: "SOCIO", MonedaSocio1: "COP"}, conf, "FUNDA", "", "COP", "COP") != "A" ||
		operationLetter(SourceRow{RolSocio1: "SOCIO", MonedaSocio1: "COP"}, CurrencyConfig{}, "GENERAL", "", "COP", "COP") != "D" {
		t.Fatal("operationLetter")
	}
	if domain.NormalizeCurrency(" usdt ", "") != "USDT" {
		t.Fatal("normalización de divisa")
	}
}

func profileJSON(t *testing.T, profile Profile) []byte {
	t.Helper()
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("json de perfil: %v", err)
	}
	return raw
}

// TestProjectRowLiveParidadConSnapshot exige que la vista previa de una fila no
// liquidada use la misma polaridad y el mismo ancla que el snapshot congelado.
func TestProjectRowLiveParidadConSnapshot(t *testing.T) {
	fundda := Profile{Nombre: "FUNDDA", MonedaBase: "USDT",
		Monedas: map[string]CurrencyConfig{
			"COP": {Tipo: "D", Polaridad: "+", Porcentaje: Spread{Deposito: 2, Pago: 3}},
		}}
	socio1 := Profile{Nombre: "SOCIO_HEREDA", MonedaBase: "USDT", Herencia: true,
		Monedas: map[string]CurrencyConfig{"COP": {Polaridad: "-"}}}
	socio2 := Profile{Nombre: "SOCIO_ANCLA", MonedaBase: "USDT",
		Monedas: map[string]CurrencyConfig{"COP": {Polaridad: "+"}}}
	lote := RateLot{IDTasa: "T052", Tasas: map[string]float64{"COP": 1000, "USDT": 1}}
	want := CalcularSnapshotFinanciero(
		Raw{HashLargo: "h4", Monto: 1000000, Moneda: "COP", TipoManual: "D", IDTasa: "T052"},
		socio1, socio2, lote, fundda,
	)

	row := SourceRow{
		HashLargo: "h4", Monto: 1000000, Moneda: "COP",
		FBSocio1: "SOCIO_HEREDA", FBSocio2: socioPtr("SOCIO_ANCLA"),
		RolSocio1: "SOCIO", MonedaSocio1: "USDT", MonedaSocio2: "USDT",
		TasasLote: []byte(`{"COP":1000,"USDT":1}`),
		Socio1Row: profileJSON(t, socio1), Socio2Row: profileJSON(t, socio2),
	}
	got := ProjectRow(row, fundda)
	if TruncateAmount(got.Monto1) != want.Monto1 || TruncateAmount(got.ME1) != want.ME1 ||
		TruncateAmount(got.Monto2) != want.Monto2 || TruncateAmount(got.ME2) != want.ME2 {
		t.Fatalf("live != snapshot: got=%+v want=%+v", got, want)
	}
	if got.Tasa1 != want.Tasa1 || got.Tasa2 != want.Tasa2 {
		t.Fatalf("tasas live != snapshot: got=%+v want=%+v", got, want)
	}
	if got.Monto1 >= 0 || got.Monto2 <= 0 {
		t.Fatalf("el ancla de herencia debe oponer los signos: %+v", got)
	}
}
