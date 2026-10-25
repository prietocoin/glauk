package ingesta

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// execFake implementa el contrato DB que usa el repositorio. Los métodos de
// saneamiento y barrido solo ejecutan UPDATE/DELETE, así que Query/QueryRow no
// se usan.
type execCall struct {
	query string
	args  []any
}

type execFake struct {
	query string
	args  []any
	tag   pgconn.CommandTag
	err   error
	calls []execCall
	tags  []pgconn.CommandTag
	errs  []error
}

func (f *execFake) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (f *execFake) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (f *execFake) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	f.query = query
	f.args = args
	f.calls = append(f.calls, execCall{query: query, args: args})
	index := len(f.calls) - 1
	if index < len(f.errs) && f.errs[index] != nil {
		return pgconn.CommandTag{}, f.errs[index]
	}
	if index < len(f.tags) {
		return f.tags[index], nil
	}
	return f.tag, f.err
}

func TestSweepStaleJobsMarcaFallo(t *testing.T) {
	db := &execFake{tag: pgconn.NewCommandTag("UPDATE 13")}
	repo := NewRepository(db, testLogger())

	got, err := repo.SweepStaleJobs(context.Background(), 15*time.Minute)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if got != 13 {
		t.Fatalf("barridos = %d, want 13", got)
	}
	if !strings.Contains(db.query, "estado = 'FALLO'") ||
		!strings.Contains(db.query, "estado = 'EN_COLA'") ||
		!strings.Contains(db.query, "timestamp_msg") {
		t.Fatalf("consulta inesperada:\n%s", db.query)
	}
	if len(db.args) != 1 || db.args[0] != int64(900) {
		t.Fatalf("argumentos = %#v, want [900]", db.args)
	}
}

func TestSweepStaleJobsRechazaVentanaInvalida(t *testing.T) {
	repo := NewRepository(&execFake{}, testLogger())
	if _, err := repo.SweepStaleJobs(context.Background(), 0); err == nil {
		t.Fatal("se esperaba error con ventana cero")
	}
}

func TestSweepOrphansMarcaImpactosYPurgaComprobantes(t *testing.T) {
	db := &execFake{
		tags: []pgconn.CommandTag{
			pgconn.NewCommandTag("UPDATE 4"),
			pgconn.NewCommandTag("DELETE 3"),
		},
	}
	repo := NewRepository(db, testLogger())

	got, err := repo.SweepOrphans(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("sweep huérfanos: %v", err)
	}
	if got.Impactos != 4 || got.Comprobantes != 3 {
		t.Fatalf("resultado = %+v, want {4 3}", got)
	}
	if len(db.calls) != 2 {
		t.Fatalf("sentencias ejecutadas = %d, want 2", len(db.calls))
	}

	impactos := strings.Join(strings.Fields(db.calls[0].query), " ")
	for _, want := range []string{
		"UPDATE impactos_raw",
		"SET eliminado_en = NOW()",
		"comprobantes_raw",
		"socio_2 IS NOT NULL",
		"socio_2) <> ''",
		"socio_2)) <> 'GENERAL'",
	} {
		if !strings.Contains(impactos, want) {
			t.Fatalf("barrido de impactos: falta %q\n%s", want, impactos)
		}
	}

	purga := strings.Join(strings.Fields(db.calls[1].query), " ")
	for _, want := range []string{
		"DELETE FROM comprobantes_raw",
		"impactos_raw",
		"eliminado_en IS NULL",
		"socio_2 IS NOT NULL",
	} {
		if !strings.Contains(purga, want) {
			t.Fatalf("purga de comprobantes: falta %q\n%s", want, purga)
		}
	}
	if strings.Contains(purga, "DELETE FROM comprobantes_liq") {
		t.Fatal("el barrido no debe borrar el snapshot canónico comprobantes_liq")
	}

	for _, call := range db.calls {
		if len(call.args) != 1 || call.args[0] != int64(86400) {
			t.Fatalf("argumentos = %#v, want [86400]", call.args)
		}
	}
}

func TestSweepOrphansRechazaVentanaInvalida(t *testing.T) {
	repo := NewRepository(&execFake{}, testLogger())
	if _, err := repo.SweepOrphans(context.Background(), 0); err == nil {
		t.Fatal("se esperaba error con ventana cero")
	}
}

func TestSweepOrphansNoPurgaSiFallaElMarcado(t *testing.T) {
	db := &execFake{errs: []error{context.DeadlineExceeded}}
	repo := NewRepository(db, testLogger())

	if _, err := repo.SweepOrphans(context.Background(), 24*time.Hour); err == nil {
		t.Fatal("se esperaba error al marcar impactos")
	}
	if len(db.calls) != 1 {
		t.Fatalf("sentencias ejecutadas = %d, want 1", len(db.calls))
	}
}

func TestSweepNonPositivePurgaLiquidacionYComprobante(t *testing.T) {
	db := &execFake{
		tags: []pgconn.CommandTag{
			pgconn.NewCommandTag("UPDATE 2"),
			pgconn.NewCommandTag("DELETE 5"),
			pgconn.NewCommandTag("DELETE 4"),
		},
	}
	repo := NewRepository(db, testLogger())

	got, err := repo.SweepNonPositive(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("sweep montos no positivos: %v", err)
	}
	if got.Impactos != 2 || got.Liquidaciones != 5 || got.Comprobantes != 4 {
		t.Fatalf("resultado = %+v, want {2 5 4}", got)
	}
	if len(db.calls) != 3 {
		t.Fatalf("sentencias ejecutadas = %d, want 3", len(db.calls))
	}

	impactos := strings.Join(strings.Fields(db.calls[0].query), " ")
	for _, want := range []string{
		"UPDATE impactos_raw",
		"SET eliminado_en = NOW()",
		"c.monto <= 0",
		"c.creado_en < NOW()",
	} {
		if !strings.Contains(impactos, want) {
			t.Fatalf("marcado de impactos: falta %q\n%s", want, impactos)
		}
	}

	liquidaciones := strings.Join(strings.Fields(db.calls[1].query), " ")
	for _, want := range []string{
		"DELETE FROM comprobantes_liq",
		"USING comprobantes_raw",
		"c.monto <= 0",
		"c.creado_en < NOW()",
	} {
		if !strings.Contains(liquidaciones, want) {
			t.Fatalf("purga de liquidaciones: falta %q\n%s", want, liquidaciones)
		}
	}

	comprobantes := strings.Join(strings.Fields(db.calls[2].query), " ")
	for _, want := range []string{
		"DELETE FROM comprobantes_raw",
		"monto <= 0",
		"creado_en < NOW()",
	} {
		if !strings.Contains(comprobantes, want) {
			t.Fatalf("purga de comprobantes: falta %q\n%s", want, comprobantes)
		}
	}

	for _, call := range db.calls {
		if len(call.args) != 1 || call.args[0] != int64(86400) {
			t.Fatalf("argumentos = %#v, want [86400]", call.args)
		}
	}
}

func TestSweepNonPositiveRechazaVentanaInvalida(t *testing.T) {
	repo := NewRepository(&execFake{}, testLogger())
	if _, err := repo.SweepNonPositive(context.Background(), 0); err == nil {
		t.Fatal("se esperaba error con ventana cero")
	}
}

func TestSweepNonPositiveNoPurgaSiFallaElMarcado(t *testing.T) {
	db := &execFake{errs: []error{context.DeadlineExceeded}}
	repo := NewRepository(db, testLogger())

	if _, err := repo.SweepNonPositive(context.Background(), 24*time.Hour); err == nil {
		t.Fatal("se esperaba error al marcar impactos")
	}
	if len(db.calls) != 1 {
		t.Fatalf("sentencias ejecutadas = %d, want 1", len(db.calls))
	}
}

func TestNormalizeRegistrosHash(t *testing.T) {
	db := &execFake{tag: pgconn.NewCommandTag("UPDATE 7")}
	repo := NewRepository(db, testLogger())

	got, err := repo.NormalizeRegistrosHash(context.Background())
	if err != nil {
		t.Fatalf("normalizar: %v", err)
	}
	if got != 7 {
		t.Fatalf("normalizados = %d, want 7", got)
	}
	if !strings.Contains(db.query, "lower(btrim(r.hash_largo))") ||
		!strings.Contains(db.query, "NOT EXISTS") {
		t.Fatalf("consulta inesperada:\n%s", db.query)
	}
}
