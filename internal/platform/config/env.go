package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// lookupFunc abstracts the environment so tests never touch the real process.
type lookupFunc func(string) (string, bool)

func envString(l lookupFunc, key, def string) string {
	if raw, ok := l(key); ok {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			return trimmed
		}
	}
	return def
}

func envInt(l lookupFunc, key string, def int) (int, error) {
	raw, ok := l(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un entero: %q", key, raw)
	}
	return value, nil
}

func envBool(l lookupFunc, key string, def bool) (bool, error) {
	raw, ok := l(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("%s debe ser true o false: %q", key, raw)
	}
	return value, nil
}

func envDuration(l lookupFunc, key string, def time.Duration) (time.Duration, error) {
	raw, ok := l(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s debe ser una duración positiva (ej. 25s): %q", key, raw)
	}
	return value, nil
}

func envBytes(l lookupFunc, key string, def int64) (int64, error) {
	raw, ok := l(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	value, err := parseBytes(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s debe ser un tamaño positivo (ej. 20MiB): %q", key, raw)
	}
	return value, nil
}

func envCSV(l lookupFunc, key string, def []string) []string {
	raw, ok := l(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}

// parseBytes accepts raw bytes or binary units. MB and MiB are both 1 << 20
// by contract, so 20MB, 20MiB and 20971520 produce the same value.
func parseBytes(raw string) (int64, error) {
	text := strings.ToUpper(strings.TrimSpace(raw))
	if text == "" {
		return 0, fmt.Errorf("tamaño vacío")
	}
	multiplier := int64(1)
	for _, unit := range []struct {
		suffix string
		factor int64
	}{
		{"GIB", 1 << 30}, {"MIB", 1 << 20}, {"KIB", 1 << 10},
		{"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"B", 1},
	} {
		if strings.HasSuffix(text, unit.suffix) {
			multiplier = unit.factor
			text = strings.TrimSpace(strings.TrimSuffix(text, unit.suffix))
			break
		}
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("tamaño inválido: %q", raw)
	}
	bytes := value * float64(multiplier)
	if bytes > float64(1<<62) {
		return 0, fmt.Errorf("tamaño fuera de rango: %q", raw)
	}
	return int64(bytes), nil
}
