package calc

import "testing"

func TestDefaultRatesAreIndependentCopies(t *testing.T) {
	first := DefaultRates()
	second := DefaultRates()
	first["USD"] = 99
	if second["USD"] != 1 {
		t.Fatalf("DefaultRates compartió el mapa: %v", second)
	}
}

func TestMergeRatesNormalizesAndOverrides(t *testing.T) {
	merged := MergeRates(map[string]float64{" cop ": 4000, "PYUSD": 1.5})
	if merged["COP"] != 4000 || merged["PYUSD"] != 1.5 {
		t.Fatalf("merge inesperado: %v", merged)
	}
	if merged["USD"] != 1 || merged["USDT"] != 1 || merged["ECU"] != 1 || merged["PAN"] != 1 {
		t.Fatalf("defaults perdidos: %v", merged)
	}
}

func TestDecodeRatesJSONBAndString(t *testing.T) {
	jsonb := DecodeRates([]byte(`{"cop":"4000","usdt":1}`))
	if jsonb["COP"] != 4000 || jsonb["USDT"] != 1 {
		t.Fatalf("jsonb inesperado: %v", jsonb)
	}
	encoded := DecodeRates([]byte(`"{\"ARS\":1250.5}"`))
	if encoded["ARS"] != 1250.5 {
		t.Fatalf("string JSON inesperado: %v", encoded)
	}
	invalid := DecodeRates([]byte(`{`))
	if invalid == nil || len(invalid) != 0 {
		t.Fatalf("json inválido debe devolver mapa vacío: %v", invalid)
	}
}

func TestSnapshotRateFallbacks(t *testing.T) {
	rates := map[string]float64{"COP": 4000, "CERO": 0, "NAN": 0}
	if got := SnapshotRate(rates, "COP"); got != 4000 {
		t.Fatalf("COP = %v", got)
	}
	if got := SnapshotRate(rates, "CERO"); got != 1 {
		t.Fatalf("cero = %v", got)
	}
	if got := SnapshotRate(rates, "FALTA"); got != 1 {
		t.Fatalf("falta = %v", got)
	}
}
