package store

import (
	"context"

	"glauk/internal/tasas/domain"
)

// Socio es un perfil tal como vive en perfiles_glaukov, más los interruptores
// operativos que no participan del cálculo.
//
// Se separa de domain.Perfil, que es la entrada del calculador, porque estos campos
// son de la fila y no del negocio del cálculo: al calculador no le importa si un
// socio está activo, y quien decide publicar necesita saberlo antes. Embebe el perfil
// para que quien tenga un Socio pueda pasárselo al calculador sin conversión.
type Socio struct {
	domain.Perfil
	Activo     *bool  `json:"activo"`
	ModoPrueba *bool  `json:"modo_prueba"`
	Rol        string `json:"rol"`
	// Mostrar es la fuente de verdad de H-02. Se omite del JSON del panel:
	// la actividad publicada se calcula, no se expone como un campo más.
	Mostrar Mostrar `json:"-"`
}

// Mostrar contiene los interruptores del panel Perfiles que afectan a Tasas.
// Tasas es puntero para distinguir "no configurado" de "apagado": solo true habilita.
type Mostrar struct {
	Tasas *bool `json:"tasas"`
}

// SocioUpdate son los campos que el panel puede cambiar de un socio.
//
// Activo y ModoPrueba son punteros porque la columna admite nulo y el panel envía
// nulo cuando no toca el interruptor: distinguir "ponlo en falso" de "no lo toques"
// es la diferencia entre apagar un socio a propósito y apagarlo sin querer.
//
// El rol tampoco viaja en la petición del panel, así que también es puntero: un rol
// ausente no se escribe. El original hacía `rol = $4` con `rol || 'SOCIO'`, de modo
// que cada guardado de monedas reseteaba el rol del socio; el COALESCE de la
// sentencia convierte esa pérdida silenciosa en una no-operación.
type SocioUpdate struct {
	Monedas    map[string]*domain.Moneda
	Activo     *bool
	ModoPrueba *bool
	Rol        *string
}

// UpdateSocio guarda la configuración de un socio y devuelve cómo quedó.
//
// La sentencia no se escribe aquí: se arma con las columnas que la tabla tiene de
// verdad (ver sentenciaDeGuardado), porque escribir a mano `activo = $2` fue lo que
// rompió la lectura en producción y el guardado tenía la misma trampa esperando.
//
// Si el nombre no existe devuelve ErrNotFound. El servicio original respondía éxito
// con un socio vacío en ese caso, y el panel mostraba "guardado" sin haber guardado
// nada: silenciar ese desacuerdo es peor que devolver el error.
func UpdateSocio(ctx context.Context, q Querier, nombre string, cambios SocioUpdate) (Socio, error) {
	sentencia, args, err := sentenciaDeGuardado(ctx, q, cambios, nombre)
	if err != nil {
		return Socio{}, err
	}
	return queryOne(ctx, q, sentencia, args, scanSocio)
}
