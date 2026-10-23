package repo

import (
	"log/slog"

	"glauk/internal/perfiles/model"
)

// advertirConflictos deja trazabilidad de H-01 sin interrumpir la lectura
// histórica. Recibe el nombre del socio porque el codec de model no lo conoce:
// solo sabe qué campo legacy fue descartado y cuál valor de porcentaje ganó.
//
// El WARN es la señal operativa para detectar datos divergentes entre Tasas y
// Perfiles. La fila se corrige en la siguiente lectura o escritura del socio,
// tal como exige la migración no destructiva.
// El código ISO se incluye siempre en mayúsculas para que la auditoría pueda
// agrupar por moneda sin normalizar los logs. Nunca se registra el JSON completo
// ni un JID: solo el nombre del socio, la moneda, el campo y los dos valores
// numéricos, que son los datos necesarios para reconciliar Tasas y Perfiles.
func advertirConflictos(log *slog.Logger, nombre string, conflictos []model.Conflicto) {
	for _, conflicto := range conflictos {
		log.Warn("monedas: porcentaje gana sobre el valor legacy",
			"socio", nombre,
			"moneda", conflicto.Moneda,
			"campo", conflicto.Campo,
			"descartado", conflicto.Descartado,
			"conservado", conflicto.Conservado,
		)
	}
}
