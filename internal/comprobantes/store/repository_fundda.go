package store

import (
	"context"
	"fmt"

	"glauk/internal/comprobantes/calc"
)

// Fundda obtiene el perfil usado como plantilla de herencia del listado.
//
// Es una lectura fuera de transacción: el listado proyecta muchas filas y la
// plantilla cambia con muy poca frecuencia. Si la tabla no tiene FUNDDA, el
// perfil vacío hace que EffectiveCurrency caiga en el contrato por defecto.
//
// La ausencia de fila no es un error de negocio: el listado debe seguir
// funcionando con los defaults históricos y solo falla si PostgreSQL rechaza
// la consulta. El escaneo se comparte con la variante transaccional para que
// ambas rutas decodifiquen exactamente el mismo JSONB.
//
// FUNDDA puede cambiar mientras un operador edita un comprobante; la variante
// transaccional vuelve a leerla después del lock para que el snapshot use la
// versión vigente en ese instante.
func (r *Repository) Fundda(ctx context.Context) (calc.Profile, error) {
	profile, err := scanProfile(r.db.QueryRow(ctx, funddaTxSQL), "FUNDDA")
	if err != nil {
		return calc.Profile{}, fmt.Errorf("consultar plantilla FUNDDA: %w", err)
	}
	return profile, nil
}
