package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"glauk/internal/platform/postgres"
	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// ErrTasasInvalidas rechaza un lote sin tasas antes de llegar a PostgreSQL.
var ErrTasasInvalidas = errors.New("Estructura de tasas inválida.")

const (
	// loteLockKey serializa la lectura del último lote y la inserción del
	// siguiente. Es una constante estable: todos los procesos Core compiten por
	// el mismo advisory lock de transacción.
	loteLockKey int64 = 20261024

	// maxIntentosLote cubre el caso de un escritor externo (el servicio legado)
	// que no toma el advisory lock: el índice único lo rechaza y se reintenta
	// con el identificador siguiente.
	maxIntentosLote = 3
)

// selloFecha reproduce new Date().toISOString(): milisegundos y zona Z.
const selloFecha = "2006-01-02T15:04:05.000Z"

// GuardarLote inserta un lote nuevo sin publicarlo. La lectura del último lote
// y el INSERT ocurren en una misma transacción protegida por un advisory lock,
// de modo que dos publicaciones concurrentes no pueden calcular el mismo T###.
// Si un escritor externo gana la carrera, el índice único de id_tasa produce
// 23505 y la operación se reintenta con el siguiente identificador.
func (s *Service) GuardarLote(ctx context.Context, tasas map[string]any) (domain.Lote, error) {
	if len(tasas) == 0 {
		return domain.Lote{}, ErrTasasInvalidas
	}
	if sello, ok := tasas["CREATED_AT"].(string); !ok || sello == "" {
		tasas["CREATED_AT"] = s.now().UTC().Format(selloFecha)
	}

	beginner, ok := s.pool.(postgres.Beginner)
	if !ok {
		return domain.Lote{}, errors.New("tasas: el pool no implementa transacciones")
	}

	var ultimoError error
	for intento := 0; intento < maxIntentosLote; intento++ {
		var lote domain.Lote
		err := postgres.WithTx(ctx, beginner, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx,
				`SELECT pg_advisory_xact_lock($1)`, loteLockKey); err != nil {
				return err
			}
			lotes, err := store.RecentLotes(ctx, tx)
			if err != nil {
				return err
			}
			ultimo := ""
			if len(lotes) > 0 {
				ultimo = lotes[0].IDTasa
			}
			lote, err = store.InsertLote(ctx, tx, store.NextLoteID(ultimo), tasas)
			return err
		})
		if err == nil {
			return lote, nil
		}
		if !esConflictoUnico(err) {
			return domain.Lote{}, err
		}
		ultimoError = err
	}
	return domain.Lote{}, ultimoError
}

// esConflictoUnico reconoce la violación del índice único de id_tasa. La cadena
// de errores pasa por store.queryOne, que envuelve con %w, por eso se usa As.
func esConflictoUnico(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// EliminarLote borra un lote concreto e informa si existía.
func (s *Service) EliminarLote(ctx context.Context, idTasa string) (bool, error) {
	idTasa = strings.TrimSpace(idTasa)
	if idTasa == "" {
		return false, ErrLoteInvalido
	}
	return store.DeleteLote(ctx, s.pool, idTasa)
}
