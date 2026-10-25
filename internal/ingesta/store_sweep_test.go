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
// saneamiento y barrido solo ejecutan UPDATE, así que Query/QueryRow no se usan.
type execFake struct {
	query string
	args  []any
	tag   pgconn.CommandTag
	err   error
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
