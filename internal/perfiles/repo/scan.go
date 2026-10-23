package repo

import (
	"encoding/json"
	"fmt"

	"glauk/internal/perfiles/model"
)

// scanner abstrae pgx.Row y pgx.Rows para que el escaneo de perfiles viva en un
// único lugar y no se desincronice del contrato de perfilColumns.
type scanner interface {
	Scan(dest ...any) error
}

// scanPerfil decodifica mostrar y monedas con el codec tolerante. Devuelve los
// conflictos de H-01 junto al perfil para que el lector emita el WARN con el
// nombre del socio, que el modelo por sí solo no conoce. También devuelve el
// JSONB crudo de monedas: el lector lo usa para detectar filas históricas que
// aún no declaran tipo/polar_D/polar_P y persistir su forma canónica.
func scanPerfil(row scanner) (model.Perfil, []model.Conflicto, []byte, error) {
	var perfil model.Perfil
	var mostrar, monedas []byte
	err := row.Scan(
		&perfil.IDGrupo,
		&perfil.Nombre,
		&perfil.Rol,
		&perfil.MonedaBase,
		&perfil.SaldoInicial,
		&mostrar,
		&monedas,
		&perfil.Herencia,
	)
	if err != nil {
		return model.Perfil{}, nil, nil, err
	}
	if len(mostrar) > 0 {
		if err := json.Unmarshal(mostrar, &perfil.Mostrar); err != nil {
			return model.Perfil{}, nil, nil, fmt.Errorf("mostrar inválido: %w", err)
		}
	}
	perfil.Monedas = model.Monedas{}
	if len(monedas) > 0 {
		if err := json.Unmarshal(monedas, &perfil.Monedas); err != nil {
			return model.Perfil{}, nil, nil, fmt.Errorf("monedas inválidas: %w", err)
		}
	}
	perfil.Activo = perfil.Mostrar.Tasas
	return perfil, perfil.Monedas.Conflictos(), monedas, nil
}
