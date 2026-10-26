package service

import (
	"context"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/store"
)

// Repository es el puerto que el servicio necesita del almacenamiento.
//
// El repositorio real vive en store; las pruebas inyectan un doble local.
// InTx entrega un Tx acotado para que la lectura del comprobante, el recálculo
// del snapshot y el upsert ocurran bajo la misma transacción de PostgreSQL.
type Repository interface {
	ListRows(ctx context.Context, filters domain.ListFilters) ([]calc.SourceRow, int, error)
	Fundda(ctx context.Context) (calc.Profile, error)
	LatestLot(ctx context.Context) (domain.LoteTasa, error)
	History(ctx context.Context) ([]domain.LoteTasa, error)
	Liquidar(ctx context.Context, liq domain.Liquidacion) error
	ProfilesExist(ctx context.Context, names ...string) (bool, error)
	InTx(ctx context.Context, fn func(store.Tx) error) error
	MarkReprocessing(ctx context.Context, hash string) (bool, error)
	Queue(ctx context.Context) ([]domain.QueueItem, error)
	UpdateQueue(ctx context.Context, hash string, data domain.QueueUpdate) (bool, error)
	DeleteQueue(ctx context.Context, hash string) error
	FiltrosReporte(ctx context.Context, rol string) (domain.ReportFilters, error)
	MovimientosReporte(ctx context.Context, q domain.ReportQuery) ([]domain.ReportRow, error)
	ResumenReporte(ctx context.Context, q domain.ReportQuery) (domain.ReportSummary, error)
}
