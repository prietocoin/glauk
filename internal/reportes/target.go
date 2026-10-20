package reportes

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Un JID válido es un usuario, un grupo o un LID. El rango 5–25 cubre los
// números internacionales y los identificadores de grupo de WhatsApp.
var jidPattern = regexp.MustCompile(
	`^(?:[0-9]{5,25}|[0-9]{5,25}-[0-9]{5,25})@(g\.us|s\.whatsapp\.net|lid)$`,
)

func (s *Service) resolverDestino(ctx context.Context, socio, remoteJID string) (string, error) {
	if jid := strings.TrimSpace(remoteJID); jid != "" {
		return NormalizarJID(jid)
	}

	nombre := strings.TrimSpace(socio)
	if nombre == "" {
		return "", ErrDestinoRequerido
	}

	perfil, err := s.socios.SocioPorNombre(ctx, nombre)
	if errors.Is(err, ErrSocioNoEncontrado) {
		return "", ErrSocioNoEncontrado
	}
	if err != nil {
		return "", fmt.Errorf("resolver socio %q: %w", nombre, err)
	}

	raw := strings.TrimSpace(perfil.IDGrupo)
	if raw == "" || EsPlaceholderJID(raw) {
		return "", fmt.Errorf("%w: socio %q", ErrJIDNoDisponible, perfil.Nombre)
	}
	return NormalizarJID(raw)
}

// NormalizarJID acepta el formato legado @c.us y lo traduce al actual.
// Rechaza espacios, '+', guiones bajos y placeholders como "NOMBRE_JID".
func NormalizarJID(raw string) (string, error) {
	jid := strings.ToLower(strings.TrimSpace(raw))
	jid = strings.TrimPrefix(jid, "whatsapp:")
	if strings.HasSuffix(jid, "@c.us") {
		jid = strings.TrimSuffix(jid, "@c.us") + "@s.whatsapp.net"
	}
	if !jidPattern.MatchString(jid) {
		return "", fmt.Errorf("%w: %q", ErrJIDInvalido, maskJID(raw))
	}
	return jid, nil
}

// maskJID conserva el dominio y los tres primeros caracteres; el resto se tapa.
// Nunca registra el JID completo en logs ni en mensajes de error.
func maskJID(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "***"
	}
	at := strings.LastIndexByte(clean, '@')
	if at < 0 {
		if len(clean) <= 3 {
			return "***"
		}
		return clean[:3] + "***"
	}
	local := clean[:at]
	if len(local) <= 3 {
		return "***" + clean[at:]
	}
	return local[:3] + "***" + clean[at:]
}

// EsPlaceholderJID detecta el default histórico nombre+"_JID" y variantes.
func EsPlaceholderJID(raw string) bool {
	s := strings.ToUpper(strings.TrimSpace(raw))
	return strings.HasSuffix(s, "_JID") || s == "NOMBRE_JID" ||
		strings.ContainsAny(s, " _")
}
