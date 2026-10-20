package reportes

import "errors"

var (
	// ErrDestinoRequerido indica que falta socio y remoteJid.
	ErrDestinoRequerido = errors.New("debe indicar socio o remoteJid")
	// ErrSocioNoEncontrado indica que el directorio no conoce el socio.
	ErrSocioNoEncontrado = errors.New("socio no encontrado")
	// ErrJIDNoDisponible indica que el perfil no tiene un JID utilizable.
	ErrJIDNoDisponible = errors.New("el socio no tiene un JID configurado")
	// ErrJIDInvalido indica que el JID no cumple el formato de WhatsApp.
	ErrJIDInvalido = errors.New("el JID no tiene un formato de WhatsApp válido")
	// ErrImagenRequerida indica que no se recibió base64.
	ErrImagenRequerida = errors.New("base64 es obligatorio")
	// ErrImagenMuyGrande indica que el JPEG supera el límite de bytes.
	ErrImagenMuyGrande = errors.New("la imagen excede el tamaño máximo permitido")
	// ErrImagenFormato indica que el contenido no es un JPEG válido.
	ErrImagenFormato = errors.New("la imagen debe ser JPEG")
	// ErrImagenDimensiones indica que el JPEG no cumple los límites de tamaño.
	ErrImagenDimensiones = errors.New("las dimensiones de la imagen no son válidas")
	// ErrTextoRequerido indica que el reporte de texto está vacío.
	ErrTextoRequerido = errors.New("el texto es obligatorio")
	// ErrEnvioFallido envuelve cualquier fallo del sender de WhatsApp.
	ErrEnvioFallido = errors.New("no se pudo enviar el reporte")
	// ErrEnvioEnCurso indica que otra petición reservó la misma clave.
	ErrEnvioEnCurso = errors.New("ya hay un envío en curso para esta clave")
	// ErrIdempotenciaLlena indica que el almacén alcanzó su límite.
	ErrIdempotenciaLlena = errors.New("el almacén de idempotencia está lleno")
)
