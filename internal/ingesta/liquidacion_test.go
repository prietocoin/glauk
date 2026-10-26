package ingesta

import (
	"context"
	"strings"
	"testing"
)

func TestGroupsOfRechazaGruposIguales(t *testing.T) {
	_, _, err := groupsOf([]string{"120363000000000001@g.us", " 120363000000000001@g.us "})
	if err == nil || !strings.Contains(err.Error(), "dos grupos distintos") {
		t.Fatalf("error = %v, esperado rechazo de grupos iguales", err)
	}
}

func TestGroupsOfRechazaPlaceholders(t *testing.T) {
	_, _, err := groupsOf([]string{"GENERAL", "NO DEFINIDO", ""})
	if err == nil || !strings.Contains(err.Error(), "dos grupos distintos") {
		t.Fatalf("error = %v, esperado rechazo de placeholders", err)
	}
}

func TestGroupsOfAceptaDosGruposDistintos(t *testing.T) {
	primero, segundo, err := groupsOf([]string{"grupo-a", "grupo-b", "grupo-a"})
	if err != nil {
		t.Fatalf("groupsOf: %v", err)
	}
	if primero != "grupo-a" || segundo != "grupo-b" {
		t.Fatalf("grupos = %q, %q", primero, segundo)
	}
}

func TestPerfilRechazaPlaceholderSinConsultarDB(t *testing.T) {
	adapter := &LiquidadorAdapter{}
	if _, err := adapter.perfil(context.Background(), "GENERAL"); err == nil {
		t.Fatal("GENERAL no debe resolver un perfil")
	}
	if _, err := adapter.perfil(context.Background(), "NO DEFINIDO"); err == nil {
		t.Fatal("NO DEFINIDO no debe resolver un perfil")
	}
}
