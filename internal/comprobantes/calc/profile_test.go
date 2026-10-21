package calc

import (
	"encoding/json"
	"math"
	"testing"
)

func TestDecimalAceptaNumerosCadenasYBasura(t *testing.T) {
	cases := map[string]float64{
		`{"deposito":12.5,"pago":"3.25"}`: 12.5,
		`{"deposito":"","pago":null}`:     0,
		`{"deposito":"abc","pago":-4}`:    0,
	}
	for raw, want := range cases {
		var spread Spread
		if err := json.Unmarshal([]byte(raw), &spread); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
		if float64(spread.Deposito) != want {
			t.Fatalf("%s depósito = %v", raw, spread.Deposito)
		}
	}
}

func TestDecodeProfileToleraVacioEInvalido(t *testing.T) {
	if got := DecodeProfile(nil); got.Nombre != "" || got.Monedas != nil {
		t.Fatalf("perfil vacío = %+v", got)
	}
	if got := DecodeProfile([]byte(`{`)); got.Nombre != "" {
		t.Fatalf("perfil inválido = %+v", got)
	}
	got := DecodeProfile([]byte(`{"nombre":"FUNDA","herencia":true,` +
		`"monedas":{"COP":{"tipo":"D","polar_D":"+","polar_P":"-"}}}`))
	if got.Nombre != "FUNDA" || !got.Herencia || got.Monedas["COP"].Tipo != "D" {
		t.Fatalf("perfil válido = %+v", got)
	}
}

func TestEffectiveCurrencyHerenciaYDefault(t *testing.T) {
	fundda := Profile{Monedas: map[string]CurrencyConfig{
		"COP": {Tipo: "A", PolarD: "-", PolarP: "+"},
	}}
	socio := Profile{Monedas: map[string]CurrencyConfig{
		"COP": {Tipo: "D", PolarD: "+", PolarP: "-"},
	}}
	if got := EffectiveCurrency(socio, fundda, "COP"); got.Tipo != "D" {
		t.Fatalf("propia = %+v", got)
	}
	hereda := Profile{Herencia: true, Monedas: map[string]CurrencyConfig{}}
	if got := EffectiveCurrency(hereda, fundda, "COP"); got.Tipo != "A" || got.PolarD != "-" {
		t.Fatalf("heredada = %+v", got)
	}
	if got := EffectiveCurrency(socio, fundda, "ARS"); got.Tipo != "D" || got.PolarD != "+" || got.PolarP != "-" {
		t.Fatalf("default = %+v", got)
	}
}

func TestResolveSocioConfigRamas(t *testing.T) {
	if got := ResolveSocioConfig(Profile{Nombre: "GENERAL"}, "COP", Profile{}); got.Active || got.Moneda != "USDT" {
		t.Fatalf("general = %+v", got)
	}
	fundda := Profile{Nombre: "FUNDDA", Monedas: map[string]CurrencyConfig{
		"COP": {Polaridad: "-"},
	}}
	hereda := Profile{Nombre: "HIJO", MonedaBase: "usdt", Herencia: true}
	got := ResolveSocioConfig(hereda, "COP", fundda)
	if !got.Active || !got.HereDa || got.Conf.Polaridad != "-" || got.Moneda != "USDT" {
		t.Fatalf("herencia = %+v", got)
	}
	got = ResolveSocioConfig(Profile{Nombre: "SIN", MonedaBase: ""}, "COP", Profile{})
	if got.Conf.Polaridad != "+" || got.Moneda != "USDT" {
		t.Fatalf("default = %+v", got)
	}
	got = ResolveSocioConfig(Profile{Nombre: "VACIA", Monedas: map[string]CurrencyConfig{
		"COP": {Polaridad: " "},
	}}, "COP", Profile{})
	if got.Conf.Polaridad != "+" {
		t.Fatalf("polaridad vacía = %+v", got)
	}
}

func TestTruncateCommercialRateYAbsPercent(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{0, 1},
		{math.NaN(), 1},
		{math.Inf(1), 1},
		{100.9, 100},
		{1.234, 1.23},
		{-50.555, -50.55},
	}
	for _, tc := range cases {
		if got := TruncateCommercialRate(tc.in); got != tc.want {
			t.Fatalf("TruncateCommercialRate(%v) = %v, quiero %v", tc.in, got, tc.want)
		}
	}
	if got := AbsPercent(Decimal(-7.5)); got != 7.5 {
		t.Fatalf("AbsPercent = %v", got)
	}
}

func TestRateOrValueOrTasaOrOne(t *testing.T) {
	rates := map[string]float64{"COP": 4000, "CERO": 0}
	if RateOr(rates, "COP") != 4000 || RateOr(rates, "CERO") != 1 || RateOr(rates, "FALTA") != 1 {
		t.Fatal("RateOr")
	}
	nan := math.NaN()
	if ValueOr(nil, 7) != 7 || ValueOr(&nan, 8) != 8 || ValueOr(testPtr(-2), 0) != -2 {
		t.Fatal("ValueOr")
	}
	if TasaOrOne(nil) != 1 || TasaOrOne(testPtr(0)) != 1 || TasaOrOne(testPtr(4000)) != 4000 {
		t.Fatal("TasaOrOne")
	}
}

func TestNaturalSignYOperationNature(t *testing.T) {
	conf := CurrencyConfig{Polaridad: "+"}
	cases := []struct {
		cfg  SocioConfig
		tipo string
		want int
	}{
		{SocioConfig{Active: false}, "D", 1},
		{SocioConfig{Active: true, Conf: conf}, "A", 1},
		{SocioConfig{Active: true, Conf: conf}, "D", 1},
		{SocioConfig{Active: true, Conf: CurrencyConfig{Polaridad: "-"}}, "D", -1},
		{SocioConfig{Active: true, Conf: conf}, "P", -1},
		{SocioConfig{Active: true, Conf: CurrencyConfig{Polaridad: "-"}}, "P", 1},
	}
	for _, tc := range cases {
		if got := NaturalSign(tc.cfg, tc.tipo); got != tc.want {
			t.Fatalf("NaturalSign(%+v, %s) = %d", tc.cfg, tc.tipo, got)
		}
	}
	if OperationNature("") != "D" || OperationNature(" pago ") != "P" || OperationNature("abono") != "A" {
		t.Fatal("OperationNature")
	}
}
