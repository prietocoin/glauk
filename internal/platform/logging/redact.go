package logging

import "strings"

// MaskKey preserves only the first four and last two characters. Short keys
// are fully hidden so the mask never reveals most of the secret.
func MaskKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if len(trimmed) <= 8 {
		return "****"
	}
	return trimmed[:4] + "****" + trimmed[len(trimmed)-2:]
}

// MaskJID keeps the domain and five leading digits, matching the documented
// example 5491123456789@g.us -> 54911****@g.us.
func MaskJID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	at := strings.LastIndexByte(trimmed, '@')
	if at <= 0 {
		return "****"
	}
	local, domain := trimmed[:at], trimmed[at:]
	if len(local) <= 5 {
		return "****" + domain
	}
	return local[:5] + "****" + domain
}

// MaskPhone preserves a short prefix and suffix so support can correlate a
// number without exposing it in logs.
func MaskPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if len(trimmed) <= 6 {
		return "****"
	}
	return trimmed[:4] + "****" + trimmed[len(trimmed)-2:]
}
