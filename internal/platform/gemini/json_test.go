package gemini

import (
	"testing"
)

func TestParseJSONConMarkdown(t *testing.T) {
	text := "Listo:\n```json\n{\"monto\":\"1.234,56\",\"moneda\":\"usd\"," +
		"\"banco\":\" Banco X \",\"referencia\":\"REF-1\",\"titular\":\"Ana  Pérez\"}\n```"
	result, err := ParseJSON(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Monto != 1234.56 {
		t.Fatalf("monto = %v", result.Monto)
	}
	if result.Banco != "Banco X" || result.Titular != "Ana Pérez" {
		t.Fatalf("resultado = %+v", result)
	}
	normalized := result.Normalize()
	if normalized.Moneda != "USD" {
		t.Fatalf("moneda = %s", normalized.Moneda)
	}
}

func TestFromMapAlias(t *testing.T) {
	result := FromMap(map[string]any{
		"importe": 10.5, "currency": "eur", "bank": "B",
		"ref": "R", "beneficiario": "Titular",
	})
	if result.Monto != 10.5 || result.Moneda != "eur" || result.Banco != "B" ||
		result.Referencia != "R" || result.Titular != "Titular" {
		t.Fatalf("resultado = %+v", result)
	}
}

func TestParseAmountFormatos(t *testing.T) {
	cases := map[string]float64{
		"1.234,56":   1234.56,
		"1,234.56":   1234.56,
		"1,234":      1234,
		"1.234":      1234,
		"12,50":      12.5,
		"12.50":      12.5,
		"12.5":       12.5,
		"$ 1.500,00": 1500,
		"abc":        0,
		"":           0,
	}
	for input, want := range cases {
		if got := parseAmount(input); got != want {
			t.Errorf("parseAmount(%q) = %v, want %v", input, got, want)
		}
	}
	if got := parseAmount(1500.0); got != 1500 {
		t.Errorf("parseAmount(float) = %v", got)
	}
}

func TestParseJSONSinObjeto(t *testing.T) {
	if _, err := ParseJSON("sin json"); err == nil {
		t.Fatal("se esperaba error")
	}
}
