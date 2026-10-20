package service

import "errors"

// Errores de dominio que la capa HTTP traduce a un código estable. Se declaran
// aquí, separados de los casos de uso, para que http no dependa de los detalles
// del repositorio ni de Fiber para distinguir un 400 de un 404.
//
// Contrato de traducción:
//
//	ErrNombreRequerido  -> 400 NOMBRE_REQUERIDO
//	ErrEntidadProtegida -> 400 ENTIDAD_PROTEGIDA
//	pgx.ErrNoRows       -> 404 NO_ENCONTRADO
//	otros               -> 500 ERROR_INTERNO
//
// El mensaje de los errores de dominio es seguro para el operador. Los errores
// de persistencia no se serializan: response.go devuelve un texto genérico para
// no filtrar SQL ni credenciales. Los alias históricos conservan las claves
// error y code del envelope de Perfiles. Los códigos son estables y forman parte
// del contrato: no se traducen ni se derivan del texto del error. Cambiarlos
// requeriría actualizar a los consumidores que distinguen un 400 de un 404 por
// code y no solo por el status HTTP.
var (
	// ErrNombreRequerido evita upserts con una identidad vacía.
	ErrNombreRequerido = errors.New("el nombre del socio es obligatorio")

	// ErrEntidadProtegida protege las matrices GENERAL y FUNDDA de un borrado
	// accidental. GENERAL además no puede apagarse por PATCH.
	ErrEntidadProtegida = errors.New("no se puede eliminar o desactivar la entidad matriz")
)
