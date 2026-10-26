package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/platform/postgres"
)

type sqlExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Tx expone las operaciones que el servicio debe ejecutar en una transacción.
type Tx interface {
	LockComprobante(ctx context.Context, hash string) (RawComprobante, error)
	LiquidacionActual(ctx context.Context, hash string) (domain.Liquidacion, bool, error)
	UpdateComprobante(ctx context.Context, hash string, data RawUpdate) error
	UpsertLiquidacion(ctx context.Context, liq domain.Liquidacion) error
	DeleteComprobante(ctx context.Context, hash string) error
	RateLotByID(ctx context.Context, id string) (calcRateLot, error)
	LatestRateLot(ctx context.Context) (calcRateLot, error)
	ProfileByName(ctx context.Context, name string) (calcProfile, error)
	Fundda(ctx context.Context) (calcProfile, error)
}

// InTx abre una transacción con platform/postgres y entrega un store acotado.
func (r *Repository) InTx(ctx context.Context, fn func(Tx) error) error {
	return postgres.WithTx(ctx, r.db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(&txStore{tx: tx})
	})
}

type txStore struct {
	tx pgx.Tx
}

// Liquidar ejecuta el upsert atómico de una liquidación ya validada.
func (r *Repository) Liquidar(ctx context.Context, liq domain.Liquidacion) error {
	return upsertLiquidacion(ctx, r.db, liq)
}

// UpsertLiquidacion congela el snapshot dentro de la transacción activa.
func (t *txStore) UpsertLiquidacion(ctx context.Context, liq domain.Liquidacion) error {
	return upsertLiquidacion(ctx, t.tx, liq)
}

func upsertLiquidacion(ctx context.Context, q sqlExecutor, liq domain.Liquidacion) error {
	if err := liq.Validate(); err != nil {
		return err
	}
	tag, err := q.Exec(ctx, upsertLiquidacionSQL,
		liq.HashLargo, liq.Socio1, liq.TipoOp1, liq.Monto1, liq.Tasa1, liq.ME1,
		liq.Socio2, liq.TipoOp2, liq.Monto2, liq.Tasa2, liq.ME2, liq.LoteTasa,
	)
	if err != nil {
		return fmt.Errorf("guardar liquidación: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrComprobanteNotFound
	}
	return nil
}

// DeleteComprobante fija el orden canónico de locks de Comprobantes:
// comprobantes_raw -> comprobantes_liq -> impactos_raw. El bloqueo de la fila
// base debe ocurrir primero para que un Update concurrente no tome el lock de
// la liquidación y luego espere el de la fila base (deadlock DB-19).
func (t *txStore) DeleteComprobante(ctx context.Context, hash string) error {
	if _, err := t.LockComprobante(ctx, hash); err != nil {
		return err
	}
	if _, err := t.tx.Exec(ctx, deleteLiquidacionSQL, hash); err != nil {
		return fmt.Errorf("eliminar liquidación: %w", err)
	}
	if _, err := t.tx.Exec(ctx, markImpactosSQL, hash); err != nil {
		return fmt.Errorf("marcar impactos: %w", err)
	}
	tag, err := t.tx.Exec(ctx, deleteComprobanteSQL, hash)
	if err != nil {
		return fmt.Errorf("eliminar comprobante: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrComprobanteNotFound
	}
	return nil
}

// MarkReprocessing marca un comprobante para que un worker lo vuelva a leer.
func (r *Repository) MarkReprocessing(ctx context.Context, hash string) (bool, error) {
	tag, err := r.db.Exec(ctx, markReprocessingSQL, hash)
	if err != nil {
		return false, fmt.Errorf("marcar reprocesamiento IA: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
