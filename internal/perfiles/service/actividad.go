package service

import "glauk/internal/perfiles/model"

// Actividad implementa la tabla de verdad H-02.
//
// mostrar.tasas es la fuente de verdad. La columna legada activo, cuando existe,
// solo puede apagar: un true nunca enciende a un socio apagado en Perfiles. Si la
// columna no existe, el llamador pasa nil y la decisión queda solo en mostrar.
//
//	mostrar.tasas | columna activo | resultado
//	true          | nil/true       | true
//	true          | false          | false
//	false         | nil/true/false | false
//
// La función es pura para que la matriz completa se pruebe sin base de datos y
// para que Tasas, Comprobantes o Reportes puedan reutilizarla por un puerto.
func Actividad(mostrar model.Mostrar, columnaActivo *bool) bool {
	if !mostrar.Tasas {
		return false
	}
	if columnaActivo == nil {
		return true
	}
	return *columnaActivo
}

// ActividadDePerfil deriva el campo público activo de un perfil leído.
func ActividadDePerfil(perfil model.Perfil) bool {
	return Actividad(perfil.Mostrar, nil)
}
