package store

import (
	"context"

	perfilesmodel "glauk/internal/perfiles/model"
	perfilrepo "glauk/internal/perfiles/repo"
	"glauk/internal/tasas/domain"
)

// UpdateSocio guarda la configuración de un socio y devuelve cómo quedó.
//
// El SQL no se construye en Tasas: se delega en internal/perfiles/repo, que es el
// único escritor autorizado de perfiles_glaukov (D-12). Tasas conserva su API y su
// forma de respuesta; solo aporta el JSON de monedas ya codificado y las columnas
// opcionales que descubrió con introspección.
//
// Si el nombre no existe devuelve ErrNotFound. El servicio original respondía éxito
// con un socio vacío en ese caso, y el panel mostraba "guardado" sin haber guardado
// nada: silenciar ese desacuerdo es peor que devolver el error.
func UpdateSocio(ctx context.Context, q Querier, nombre string, cambios SocioUpdate) (Socio, error) {
	monedas, err := perfilrepo.CodificarMonedas(monedasCanonicas(cambios.Monedas))
	if err != nil {
		return Socio{}, err
	}
	existentes, err := columnasDe(ctx, q, tablaSocios)
	if err != nil {
		return Socio{}, err
	}
	rows, err := perfilrepo.ActualizarSocioTasas(ctx, q, existentes, perfilrepo.ActualizacionSocioTasas{
		Monedas:    monedas,
		Activo:     cambios.Activo,
		ModoPrueba: cambios.ModoPrueba,
		Rol:        cambios.Rol,
	}, nombre)
	if err != nil {
		return Socio{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return Socio{}, err
		}
		return Socio{}, ErrNotFound
	}
	return scanSocio(rows)
}

// monedasCanonicas traduce la forma de Tasas al superset que Perfiles persiste.
//
// Tasas solo interpreta activo y porcentaje, pero el JSONB es compartido con el
// modal de Perfiles. Al convertir antes de escribir, el codec canónico completa
// tipo, polar_D y polar_P (y sus alias legacy) en vez de reemplazar la fila por
// las dos únicas claves que Tasas entiende. Un activo ausente significa publicar,
// igual que en la lectura histórica.
func monedasCanonicas(monedas map[string]*domain.Moneda) perfilesmodel.Monedas {
	salida := make(perfilesmodel.Monedas, len(monedas))
	for codigo, moneda := range monedas {
		if moneda == nil {
			continue
		}
		activo := true
		if moneda.Activo != nil {
			activo = *moneda.Activo
		}
		salida[codigo] = perfilesmodel.Moneda{
			Activo: activo,
			Porcentaje: perfilesmodel.Porcentajes{
				Deposito: moneda.Porcentaje.Deposito,
				Pago:     moneda.Porcentaje.Pago,
			},
			Def:       moneda.Def,
			Deposito:  moneda.Deposito,
			DepOp:     moneda.DepOp,
			Pago:      moneda.Pago,
			PagoOp:    moneda.PagoOp,
			Tipo:      moneda.Tipo,
			PolarD:    moneda.PolarD,
			PolarP:    moneda.PolarP,
			Polaridad: moneda.Polaridad,
		}
	}
	return salida
}
