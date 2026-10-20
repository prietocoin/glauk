package store

import (
	"context"

	perfilrepo "glauk/internal/perfiles/repo"
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
	monedas, err := encodeJSON(cambios.Monedas)
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
