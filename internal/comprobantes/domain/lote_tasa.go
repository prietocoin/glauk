package domain

import "time"

// LoteTasa representa un lote publicado en tasas_glaukov.
//
// Comprobantes solo lo lee: publicar tasas pertenece al dominio Tasas. El
// created_at es opcional porque el fallback T001 no existe en la base y no
// debe inventar una fecha.
type LoteTasa struct {
	IDTasa    string             `json:"id_tasa"`
	Tasas     map[string]float64 `json:"tasas"`
	CreatedAt *time.Time         `json:"created_at,omitempty"`
}

// TasaVigente es la vista mínima del lote más reciente para el panel.
type TasaVigente struct {
	IDTasa    string
	Tasas     map[string]float64
	CreatedAt *time.Time
}

// Lote convierte la vista mínima al contrato JSON histórico.
func (t TasaVigente) Lote() LoteTasa {
	return LoteTasa{IDTasa: t.IDTasa, Tasas: t.Tasas, CreatedAt: t.CreatedAt}
}

// IsFallback indica si el lote es el valor sintético usado cuando la tabla
// todavía no tiene publicaciones.
func (t LoteTasa) IsFallback() bool {
	return t.IDTasa == "T001" && t.CreatedAt == nil
}
