package reportes

import (
	"errors"
	"testing"
)

func TestNormalizarJIDValidos(t *testing.T) {
	cases := map[string]string{
		"120363012345678901@g.us":      "120363012345678901@g.us",
		"5491123456789@s.whatsapp.net": "5491123456789@s.whatsapp.net",
		"12345@lid":                    "12345@lid",
		"12345-67890@g.us":             "12345-67890@g.us",
		" 12345@C.US ":                 "12345@s.whatsapp.net",
		"whatsapp:12345@g.us":          "12345@g.us",
	}
	for raw, want := range cases {
		got, err := NormalizarJID(raw)
		if err != nil {
			t.Errorf("NormalizarJID(%q): %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizarJID(%q) = %q, quiero %q", raw, got, want)
		}
	}
}

func TestNormalizarJIDInvalidos(t *testing.T) {
	invalid := []string{
		"", "NOMBRE_JID", "*_JID", "+5491123456789@s.whatsapp.net",
		"foo@g.us", "1234@g.us", "12345678901234567890123456@g.us",
		"123 45@g.us", "12345@foo", "12345-@g.us", "12345-67890-12345@g.us",
		"12345@g.us!",
	}
	for _, raw := range invalid {
		if got, err := NormalizarJID(raw); !errors.Is(err, ErrJIDInvalido) {
			t.Errorf("NormalizarJID(%q) = %q, %v; quiero ErrJIDInvalido", raw, got, err)
		}
	}
}

func TestEsPlaceholderJID(t *testing.T) {
	placeholders := []string{"NOMBRE_JID", "FUNDA_JID", "X_JID", "*_JID"}
	for _, raw := range placeholders {
		if !EsPlaceholderJID(raw) {
			t.Errorf("EsPlaceholderJID(%q) = false", raw)
		}
	}
	if EsPlaceholderJID("120363012345678901@g.us") {
		t.Fatal("un JID válido no es placeholder")
	}
}

func TestMaskJID(t *testing.T) {
	cases := map[string]string{
		"1234567890@s.whatsapp.net": "123***@s.whatsapp.net",
		"12345@g.us":                "123***@g.us",
		"":                          "***",
		"abc":                       "***",
	}
	for raw, want := range cases {
		if got := maskJID(raw); got != want {
			t.Errorf("maskJID(%q) = %q, quiero %q", raw, got, want)
		}
	}
}

func TestNormalizarJIDNoDevuelveSinteticos(t *testing.T) {
	for _, raw := range []string{"", "NOMBRE_JID", "54911", "00000"} {
		got, err := NormalizarJID(raw)
		if err == nil {
			t.Errorf("NormalizarJID(%q) = %q, quiero error", raw, got)
		}
	}
}
