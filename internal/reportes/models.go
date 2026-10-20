package reportes

// SolicitudImagen es la entrada del caso de uso, ya desacoplada de HTTP.
//
// Socio y RemoteJID son alternativos: si RemoteJID trae contenido, gana y no
// se consulta el directorio. Base64 puede incluir el prefijo data:image/jpeg.
type SolicitudImagen struct {
	Socio     string
	RemoteJID string
	Caption   string
	Base64    string
	MimeType  string
	FileName  string
}

// SolicitudTexto es la entrada del caso de uso para reportes sin imagen.
// El texto se recorta a 4096 caracteres antes de llegar al sender.
type SolicitudTexto struct {
	Socio     string
	RemoteJID string
	Texto     string
}

// Resultado es la salida del caso de uso. El handler decide cómo serializarla.
//
// Duplicado solo puede ser true cuando el handler consultó el almacén de
// idempotencia; el servicio puro devuelve el resultado del sender sin mutarlo.
type Resultado struct {
	JID       string
	MessageID string
	Estado    string
	Simulado  bool
	Duplicado bool
	Bytes     int
}

// Dimensiones describe el JPEG validado sin exponer su contenido binario.
// Se usa para auditoría y para completar la respuesta de diagnóstico.
type Dimensiones struct {
	Ancho int
	Alto  int
}

// FiltrosVista es una copia defensiva para que el handler no mute los slices
// que pertenecen al proveedor de movimientos.
func FiltrosVista(f Filtros) Filtros {
	return Filtros{
		Roles:     append([]string(nil), f.Roles...),
		Entidades: append([]string(nil), f.Entidades...),
		Hashes:    append([]string(nil), f.Hashes...),
	}
}
