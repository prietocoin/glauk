package domain

// QueueItem es una fila cruda de la cola administrativa.
//
// Puede contener PII: la capa HTTP debe responder siempre con no-store.
// Los apuntadores preservan los NULL de una tabla que nació sin restricciones
// y que la consola muestra tal cual para que un operador pueda corregirla.
type QueueItem struct {
	HashLargo    string  `json:"hash_largo"`
	HashCorto    *string `json:"hash_corto"`
	Timestamp    *int64  `json:"timestamp"`
	NombreSocio1 *string `json:"nombre_socio_1"`
	NombreSocio2 *string `json:"nombre_socio_2"`
	Conteo       *int    `json:"conteo"`
	Estado       *string `json:"estado"`
	URLImagen    *string `json:"url_imagen"`
}

// QueueUpdate contiene los campos editables de la consola raw.
type QueueUpdate struct {
	NombreSocio1 *string `json:"nombre_socio_1"`
	NombreSocio2 *string `json:"nombre_socio_2"`
	Estado       *string `json:"estado"`
	Conteo       *int    `json:"conteo"`
	Timestamp    *int64  `json:"timestamp"`
}

// HasChanges permite que el servicio rechace ediciones completamente vacías
// cuando el operador envía un payload sin ningún campo de la cola.
func (q QueueUpdate) HasChanges() bool {
	return q.NombreSocio1 != nil || q.NombreSocio2 != nil || q.Estado != nil ||
		q.Conteo != nil || q.Timestamp != nil
}
