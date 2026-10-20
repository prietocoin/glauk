package logging

import (
	"strings"
	"testing"
)

func TestMaskKey(t *testing.T) {
	raw := "supersecretkey"
	got := MaskKey(raw)
	if got != "supe****ey" {
		t.Fatalf("MaskKey = %q", got)
	}
	if strings.Contains(got, raw) {
		t.Fatal("la máscara revela la clave completa")
	}
	if MaskKey("short") != "****" {
		t.Fatal("las claves cortas deben ocultarse por completo")
	}
}

func TestMaskJID(t *testing.T) {
	got := MaskJID("5491123456789@g.us")
	if got != "54911****@g.us" {
		t.Fatalf("MaskJID = %q", got)
	}
	if strings.Contains(got, "5491123456789") {
		t.Fatal("la máscara revela el JID completo")
	}
	if got := MaskJID("123@g.us"); got != "****@g.us" {
		t.Fatalf("JID corto = %q", got)
	}
	if got := MaskJID("sin-arroba"); got != "****" {
		t.Fatalf("JID inválido = %q", got)
	}
}

func TestMaskPhone(t *testing.T) {
	got := MaskPhone("+5491123456789")
	if got != "+549****89" {
		t.Fatalf("MaskPhone = %q", got)
	}
	if strings.Contains(got, "1123456") {
		t.Fatal("la máscara revela el teléfono completo")
	}
	if MaskPhone("123") != "****" {
		t.Fatal("los teléfonos cortos deben ocultarse")
	}
}
