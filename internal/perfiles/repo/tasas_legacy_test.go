package repo

import "testing"

func TestSentenciaSocioTasasColumnasOpcionales(t *testing.T) {
	monedas := []byte(`{"COP":{"activo":true}}`)
	rol := "ADMIN"
	sentencia, args, err := SentenciaSocioTasas(
		map[string]bool{"monedas": true, "modo_prueba": true, "rol": true},
		ActualizacionSocioTasas{Monedas: monedas, Rol: &rol},
		"Ana",
	)
	if err != nil {
		t.Fatal(err)
	}
	quiero := `UPDATE perfiles_glaukov SET monedas = $1, modo_prueba = COALESCE($2, modo_prueba), rol = COALESCE($3, rol) WHERE LOWER(nombre) = LOWER($4) RETURNING *`
	if sentencia != quiero {
		t.Fatalf("sentencia = %q, quiero %q", sentencia, quiero)
	}
	if len(args) != 4 || string(args[0].([]byte)) != string(monedas) || args[3] != "Ana" {
		t.Fatalf("args = %#v", args)
	}
	if puntero, ok := args[1].(*bool); !ok || puntero != nil {
		t.Fatalf("modo_prueba = %#v", args[1])
	}
	if puntero, ok := args[2].(*string); !ok || *puntero != rol {
		t.Fatalf("rol = %#v", args[2])
	}
}

func TestSentenciaSocioTasasIncluyeActivo(t *testing.T) {
	activo := true
	sentencia, args, err := SentenciaSocioTasas(
		map[string]bool{"monedas": true, "activo": true},
		ActualizacionSocioTasas{Monedas: []byte(`{}`), Activo: &activo},
		"Ana",
	)
	if err != nil {
		t.Fatal(err)
	}
	quiero := `UPDATE perfiles_glaukov SET monedas = $1, activo = COALESCE($2, activo) WHERE LOWER(nombre) = LOWER($3) RETURNING *`
	if sentencia != quiero {
		t.Fatalf("sentencia = %q, quiero %q", sentencia, quiero)
	}
	if len(args) != 3 || args[2] != "Ana" {
		t.Fatalf("args = %#v", args)
	}
}
