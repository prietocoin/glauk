package dto

// APIResponse es el contrato estándar de las mutaciones de Comprobantes.
//
// Se conserva el sobre histórico para no romper los paneles que leen
// success/message/data directamente en la raíz. data se omite en errores y
// message se omite cuando el endpoint solo confirma una operación.
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SuccessResponse construye una respuesta exitosa con datos opcionales.
func SuccessResponse(message string, data any) APIResponse {
	return APIResponse{Success: true, Message: message, Data: data}
}

// ErrorResponse construye una respuesta de error legible por el panel.
func ErrorResponse(message string) APIResponse {
	return APIResponse{Success: false, Error: message}
}

// ValidationResponse mantiene el mismo sobre para errores 400.
//
// No se agrega un código interno aquí: el código estable pertenece a la capa
// HTTP, que decide si el fallo es VALIDATION_ERROR, NOT_FOUND o INTERNAL_ERROR.
func ValidationResponse(message string) APIResponse {
	return APIResponse{Success: false, Error: message}
}
