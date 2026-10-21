package http

import "glauk/internal/reportes"

// resultadoDTO es la proyección estable de un envío exitoso.
type resultadoDTO struct {
	JID       string `json:"jid"`
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Simulated bool   `json:"simulated"`
	Duplicate bool   `json:"duplicate"`
	Bytes     int    `json:"bytes"`
}

func resultadoDe(r reportes.Resultado) resultadoDTO {
	return resultadoDTO{
		JID: r.JID, MessageID: r.MessageID, Status: r.Estado,
		Simulated: r.Simulado, Duplicate: r.Duplicado, Bytes: r.Bytes,
	}
}

// EnviarImagenRequest es el cuerpo de POST /api/v1/reportes/imagen.
// MimeType se acepta por compatibilidad, pero la firma JPEG es la fuente de
// verdad y el servicio la valida antes de tocar WhatsApp.
type EnviarImagenRequest struct {
	Socio     string `json:"socio"`
	RemoteJID string `json:"remoteJid"`
	Caption   string `json:"caption"`
	Base64    string `json:"base64"`
	MimeType  string `json:"mimeType"`
	FileName  string `json:"fileName"`
}

// Solicitud reduce el DTO HTTP al contrato del dominio.
func (r EnviarImagenRequest) Solicitud() reportes.SolicitudImagen {
	return reportes.SolicitudImagen{
		Socio: r.Socio, RemoteJID: r.RemoteJID, Caption: r.Caption,
		Base64: r.Base64, MimeType: r.MimeType, FileName: r.FileName,
	}
}

// EnviarTextoRequest es el cuerpo de POST /api/v1/reportes/texto.
type EnviarTextoRequest struct {
	Socio     string `json:"socio"`
	RemoteJID string `json:"remoteJid"`
	Texto     string `json:"texto"`
}

// Solicitud reduce el DTO HTTP al contrato del dominio.
func (r EnviarTextoRequest) Solicitud() reportes.SolicitudTexto {
	return reportes.SolicitudTexto{
		Socio: r.Socio, RemoteJID: r.RemoteJID, Texto: r.Texto,
	}
}
