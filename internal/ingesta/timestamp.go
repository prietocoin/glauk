package ingesta

import (
	"fmt"
	"strconv"
	"strings"
)

// flexInt64 acepta segundos, milisegundos y cadenas numéricas en el timestamp
// que entrega Evolution.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(data []byte) error {
	value := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if value == "" || value == "null" {
		*f = 0
		return nil
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		float, floatErr := strconv.ParseFloat(value, 64)
		if floatErr != nil {
			return fmt.Errorf("timestamp inválido: %q", value)
		}
		number = int64(float)
	}
	switch {
	case number > 1_000_000_000_000_000:
		number /= 1_000_000
	case number > 1_000_000_000_000:
		number /= 1_000
	}
	*f = flexInt64(number)
	return nil
}
