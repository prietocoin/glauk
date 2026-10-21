package adapters

import (
	"context"
	"testing"
	"time"

	comprobantesservice "glauk/internal/comprobantes/service"
	"glauk/internal/reportes"
)

type fakeReportReader struct {
	filtros   comprobantesservice.ReportFilters
	rows      []comprobantesservice.ReportRow
	resumen   comprobantesservice.ReportSummary
	err       error
	lastQuery comprobantesservice.ReportQuery
}

func (f *fakeReportReader) FiltrosReporte(context.Context, string) (comprobantesservice.ReportFilters, error) {
	return f.filtros, f.err
}

func (f *fakeReportReader) MovimientosReporte(_ context.Context,
	q comprobantesservice.ReportQuery) ([]comprobantesservice.ReportRow, error) {
	f.lastQuery = q
	return f.rows, f.err
}

func (f *fakeReportReader) ResumenReporte(_ context.Context,
	q comprobantesservice.ReportQuery) (comprobantesservice.ReportSummary, error) {
	f.lastQuery = q
	return f.resumen, f.err
}

func TestMovimientosFiltrosNoDevuelveNulos(t *testing.T) {
	adapter := NewMovimientosComprobantes(&fakeReportReader{})
	filtros, err := adapter.Filtros(context.Background(), "SOCIO")
	if err != nil {
		t.Fatal(err)
	}
	if filtros.Roles == nil || filtros.Entidades == nil || filtros.Hashes == nil {
		t.Fatalf("los arrays deben ser no nulos: %+v", filtros)
	}
}

func TestMovimientosProyectaConsultaYFilas(t *testing.T) {
	fecha := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	reader := &fakeReportReader{rows: []comprobantesservice.ReportRow{{
		HashLargo: "largo", HashCorto: "corto", Fecha: fecha, Socio: "FUNDABLOCK",
		Rol: "SOCIO", Moneda: "USDT", Monto: 10, TipoOp: "D-USDT",
		Banco: "BANCO", Referencia: "REF", Estado: "OK",
	}}}
	adapter := NewMovimientosComprobantes(reader)
	rows, err := adapter.Movimientos(context.Background(), reportes.ConsultaMovimientos{
		Rol: "SOCIO", Entidad: "FUNDABLOCK", Hash: "corto",
		Desde: fecha, Hasta: fecha, Orden: "fecha_asc", Limite: 25,
	})
	if err != nil || len(rows) != 1 {
		t.Fatalf("Movimientos() = %d, %v", len(rows), err)
	}
	if rows[0].HashLargo != "largo" || rows[0].Monto != 10 || rows[0].Fecha != fecha {
		t.Fatalf("fila = %+v", rows[0])
	}
	q := reader.lastQuery
	if q.Rol != "SOCIO" || q.Entidad != "FUNDABLOCK" || q.Hash != "corto" ||
		q.Orden != "fecha_asc" || q.Limite != 25 || !q.Desde.Equal(fecha) {
		t.Fatalf("consulta traducida = %+v", q)
	}
}

func TestMovimientosResumenNoDevuelveMapaNulo(t *testing.T) {
	adapter := NewMovimientosComprobantes(&fakeReportReader{})
	resumen, err := adapter.Resumen(context.Background(), reportes.ConsultaMovimientos{})
	if err != nil {
		t.Fatal(err)
	}
	if resumen.PorMoneda == nil {
		t.Fatal("PorMoneda no debe ser nil")
	}
}

func TestMovimientosPropagaError(t *testing.T) {
	adapter := NewMovimientosComprobantes(&fakeReportReader{err: context.DeadlineExceeded})
	if _, err := adapter.Filtros(context.Background(), ""); err == nil {
		t.Fatal("se esperaba error")
	}
	if _, err := adapter.Movimientos(context.Background(), reportes.ConsultaMovimientos{}); err == nil {
		t.Fatal("se esperaba error")
	}
	if _, err := adapter.Resumen(context.Background(), reportes.ConsultaMovimientos{}); err == nil {
		t.Fatal("se esperaba error")
	}
}
