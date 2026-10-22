package domain

import (
	"errors"
	"testing"
	"time"
)

func TestMoneyHelpers(t *testing.T) {
	if got := NormalizeCurrency(" cop ", "USDT"); got != "COP" {
		t.Fatalf("NormalizeCurrency = %q", got)
	}
	if got := NormalizeCurrency("", "USDT"); got != "USDT" {
		t.Fatalf("fallback = %q", got)
	}
	if got := FirstNonEmpty("", "  ", " T052 "); got != "T052" {
		t.Fatalf("FirstNonEmpty = %q", got)
	}
	if got := FirstNonEmpty("", " "); got != "" {
		t.Fatalf("FirstNonEmpty vacío = %q", got)
	}
	if PositiveOrOne(4) != 4 || PositiveOrOne(0) != 1 || PositiveOrOne(-2) != 1 {
		t.Fatal("PositiveOrOne")
	}
	money := NewMoney(12.5, " usdt ")
	if money.Amount != 12.5 || money.Currency != "USDT" {
		t.Fatalf("NewMoney = %+v", money)
	}
}

func TestListFiltersNormalizeValidateYOrden(t *testing.T) {
	filters := ListFilters{
		Socio: " A ", Rol: " B ", FechaInicio: " 2026-01-01 ",
		FechaFin: "2026-01-31", Hash: " H ", Orden: "fecha_asc",
	}
	filters.Normalize()
	if filters.Socio != "A" || filters.Rol != "B" || filters.Hash != "H" ||
		filters.FechaInicio != "2026-01-01" || !filters.OrderAsc() {
		t.Fatalf("Normalize = %+v", filters)
	}
	if err := filters.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	defaultOrder := ListFilters{Orden: "otro"}
	defaultOrder.Normalize()
	if defaultOrder.Orden != "fecha_desc" || defaultOrder.OrderAsc() {
		t.Fatalf("orden por defecto = %+v", defaultOrder)
	}
	paginado := ListFilters{Limit: 500, Offset: -5}
	paginado.Normalize()
	if paginado.Limit != MaxListLimit || paginado.Offset != 0 {
		t.Fatalf("paginación fuera de rango = %+v", paginado)
	}
	sinPaginar := ListFilters{}
	sinPaginar.Normalize()
	if sinPaginar.Limit != DefaultListLimit || sinPaginar.Offset != 0 {
		t.Fatalf("paginación por defecto = %+v", sinPaginar)
	}
	invalid := ListFilters{FechaInicio: "2026-13-40"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("fecha inválida aceptada")
	}
	invalid = ListFilters{FechaFin: "ayer"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("fecha fin inválida aceptada")
	}
}

func TestReportQueryNormalize(t *testing.T) {
	query := ReportQuery{Rol: " todos ", Entidad: " FUNDA ", Hash: " H ", Orden: "asc", Limite: 900}
	query.Normalize()
	if query.Rol != "" || query.Entidad != "FUNDA" || query.Hash != "H" ||
		query.Orden != "fecha_desc" || query.Limite != 500 {
		t.Fatalf("Normalize = %+v", query)
	}
	query = ReportQuery{Orden: "fecha_asc"}
	query.Normalize()
	if query.Orden != "fecha_asc" || query.Limite != 100 {
		t.Fatalf("defaults = %+v", query)
	}
}

func TestLotesYLiquidacion(t *testing.T) {
	created := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	view := TasaVigente{IDTasa: "T052", Tasas: map[string]float64{"COP": 4000}, CreatedAt: &created}
	lote := view.Lote()
	if lote.IDTasa != "T052" || lote.Tasas["COP"] != 4000 || lote.CreatedAt == nil {
		t.Fatalf("Lote = %+v", lote)
	}
	if !(LoteTasa{IDTasa: "T001"}).IsFallback() {
		t.Fatal("T001 sin fecha debe ser fallback")
	}
	if (LoteTasa{IDTasa: "T001", CreatedAt: &created}).IsFallback() {
		t.Fatal("T001 con fecha no es fallback")
	}
	if (LoteTasa{IDTasa: "T052"}).IsFallback() {
		t.Fatal("T052 no es fallback")
	}
	empty := ""
	second := "SOCIO2"
	if (Liquidacion{}).HasSocio2() || (Liquidacion{Socio2: &empty}).HasSocio2() {
		t.Fatal("socio2 vacío no debe contar")
	}
	if !(Liquidacion{Socio2: &second}).HasSocio2() {
		t.Fatal("socio2 real debe contar")
	}
	if !(QueueUpdate{Estado: &second}).HasChanges() {
		t.Fatal("QueueUpdate con estado debe tener cambios")
	}
	if (QueueUpdate{}).HasChanges() {
		t.Fatal("QueueUpdate vacío no tiene cambios")
	}
	if !(QueueUpdate{Conteo: new(int)}).HasChanges() ||
		!(QueueUpdate{Timestamp: new(int64)}).HasChanges() ||
		!(QueueUpdate{NombreSocio1: &empty}).HasChanges() ||
		!(QueueUpdate{NombreSocio2: &second}).HasChanges() {
		t.Fatal("QueueUpdate no detectó todos los campos")
	}
}

func TestValidationErrors(t *testing.T) {
	plain := NewValidation("dato inválido")
	if plain.Error() != "dato inválido" {
		t.Fatalf("NewValidation = %v", plain)
	}
	formatted := Validationf("%s debe ser positivo", "monto")
	var validation *ValidationError
	if !errors.As(formatted, &validation) || validation.Message != "monto debe ser positivo" {
		t.Fatalf("Validationf = %v", formatted)
	}
	if !errors.Is(ErrComprobanteNotFound, ErrComprobanteNotFound) {
		t.Fatal("error centinela")
	}
}
