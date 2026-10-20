package calculator

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"glauk/internal/tasas/domain"
)

// golden es el fixture generado ejecutando el servicio Node original.
type golden struct {
	Casos []struct {
		Nombre         string           `json:"nombre"`
		InstanteMillis int64            `json:"instante_millis"`
		Perfil         domain.Perfil    `json:"perfil"`
		LoteActual     domain.Lote      `json:"lote_actual"`
		LoteAnterior   *domain.Lote     `json:"lote_anterior"`
		Esperado       domain.Cartelera `json:"esperado"`
	} `json:"casos"`
	Precision []struct {
		Entrada  float64  `json:"entrada"`
		Esperado *float64 `json:"esperado"`
	} `json:"precision"`
	Tiempo []struct {
		Millis int64  `json:"millis"`
		Fecha  string `json:"fecha"`
		Hora   string `json:"hora"`
	} `json:"tiempo"`
}

// loadGolden lee el fixture y aborta si no se puede interpretar.
func loadGolden(t *testing.T) golden {
	t.Helper()
	raw, err := os.ReadFile("testdata/golden_calculator.json")
	if err != nil {
		t.Fatalf("no se pudo leer el fixture dorado: %v", err)
	}
	var file golden
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("fixture dorado inválido: %v", err)
	}
	return file
}

// TestGolden valida el cálculo contra la salida del servicio original, con una
// tolerancia de 1e-9 que absorbe la última cifra de las funciones matemáticas.
func TestGolden(t *testing.T) {
	file := loadGolden(t)
	carteleras, fallos := 0, 0

	for _, caso := range file.Casos {
		previous := domain.Lote{}
		if caso.LoteAnterior != nil {
			previous = *caso.LoteAnterior
		}
		got := Build(caso.Perfil, caso.LoteActual, previous, time.UnixMilli(caso.InstanteMillis))
		if diff := cmp.Diff(caso.Esperado, got, cmpopts.EquateApprox(1e-9, 1e-9), cmpopts.EquateEmpty()); diff != "" {
			fallos++
			t.Errorf("cartelera %q (-esperado +obtenido):\n%s", caso.Nombre, diff)
			continue
		}
		carteleras++
		t.Logf("  %-38s %2d tarjetas  %s %s  base=%-5s lote=%s", caso.Nombre, len(got.Tarjetas), got.Fecha, got.Hora, got.MonedaBase, got.LoteTasa)
	}

	precisiones := 0
	for _, caso := range file.Precision {
		got := ApplyRatePrecision(caso.Entrada)
		ok := math.IsNaN(got)
		if caso.Esperado != nil {
			ok = math.Abs(got-*caso.Esperado) <= 1e-9
		}
		if ok {
			precisiones++
			continue
		}
		fallos++
		t.Errorf("precisión(%v) = %v, se esperaba %v", caso.Entrada, got, caso.Esperado)
	}

	tiempos := 0
	for _, caso := range file.Tiempo {
		fecha, hora := CaracasStamp(time.UnixMilli(caso.Millis))
		if fecha == caso.Fecha && hora == caso.Hora {
			tiempos++
			continue
		}
		fallos++
		t.Errorf("CaracasStamp(%d) = (%q, %q), se esperaba (%q, %q)", caso.Millis, fecha, hora, caso.Fecha, caso.Hora)
	}

	t.Logf("resumen: %d/%d carteleras | %d/%d valores de precisión | %d/%d instantes horarios | %d fallos",
		carteleras, len(file.Casos), precisiones, len(file.Precision), tiempos, len(file.Tiempo), fallos)
}
