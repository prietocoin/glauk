package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var errNoJSON = errors.New("gemini: la respuesta no contiene un objeto JSON")

func errInvalidJSON(err error) error {
	return fmt.Errorf("gemini: JSON inválido: %w", err)
}

// ParseJSON acepta la respuesta del modelo aunque venga envuelta en markdown
// o con texto adicional: busca el primer objeto y el último cierre.
func ParseJSON(text string) (Result, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return Result{}, errNoJSON
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(text[start:end+1]), &fields); err != nil {
		return Result{}, errInvalidJSON(err)
	}
	return FromMap(fields), nil
}

// FromMap normaliza alias habituales del prompt y devuelve el contrato.
func FromMap(fields map[string]any) Result {
	lookup := make(map[string]any, len(fields))
	for key, value := range fields {
		lookup[strings.ToLower(strings.TrimSpace(key))] = value
	}
	return Result{
		Monto:      parseAmount(pick(lookup, "monto", "amount", "importe", "valor")),
		Moneda:     cleanText(pick(lookup, "moneda", "currency", "divisa"), 12),
		Banco:      cleanText(pick(lookup, "banco", "bank", "entidad", "institucion"), 80),
		Referencia: cleanText(pick(lookup, "referencia", "reference", "ref", "comprobante", "numero"), 80),
		Titular:    cleanText(pick(lookup, "titular", "nombre_titular", "beneficiario", "nombre"), 120),
	}
}

// Normalize aplica el recorte final antes de escribir en PostgreSQL.
func (r Result) Normalize() Result {
	return Result{
		Monto:      r.Monto,
		Moneda:     strings.ToUpper(cleanText(r.Moneda, 12)),
		Banco:      cleanText(r.Banco, 80),
		Referencia: cleanText(r.Referencia, 80),
		Titular:    cleanText(r.Titular, 120),
	}
}

func pick(fields map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := fields[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func cleanText(value any, limit int) string {
	var text string
	switch typed := value.(type) {
	case string:
		text = typed
	case json.Number:
		text = typed.String()
	case float64:
		text = strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		text = ""
	}
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}

// parseAmount acepta "1.234,56", "1,234.56", "1234.56" y números JSON.
func parseAmount(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case json.Number:
		number, _ := typed.Float64()
		return number
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		return parseAmountString(typed)
	}
	return 0
}

func parseAmountString(raw string) float64 {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r == '.', r == ',', r == '-':
			return r
		}
		return -1
	}, raw)
	if cleaned == "" {
		return 0
	}

	lastDot := strings.LastIndex(cleaned, ".")
	lastComma := strings.LastIndex(cleaned, ",")
	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot {
			cleaned = strings.ReplaceAll(cleaned, ".", "")
			cleaned = strings.Replace(cleaned, ",", ".", 1)
		} else {
			cleaned = strings.ReplaceAll(cleaned, ",", "")
		}
	case lastComma >= 0:
		cleaned = normalizeSingleSeparator(cleaned, ",")
	case lastDot >= 0:
		cleaned = normalizeSingleSeparator(cleaned, ".")
	}

	number, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return number
}

func normalizeSingleSeparator(value, separator string) string {
	parts := strings.Split(value, separator)
	if len(parts) == 1 {
		return value
	}
	decimal := parts[len(parts)-1]
	if len(decimal) > 0 && len(decimal) <= 2 {
		return strings.Join(parts[:len(parts)-1], "") + "." + decimal
	}
	return strings.Join(parts, "")
}
