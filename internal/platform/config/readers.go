package config

import "time"

func readInt(l lookupFunc, key string, def int, dst *int, errs *Errors) {
	value, err := envInt(l, key, def)
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	*dst = value
}

func readInt32(l lookupFunc, key string, def int, dst *int32, errs *Errors) {
	value, err := envInt(l, key, def)
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	if value < 0 || value > 2147483647 {
		*errs = append(*errs, key+" fuera de rango")
		return
	}
	*dst = int32(value)
}

func readBool(l lookupFunc, key string, def bool, dst *bool, errs *Errors) {
	value, err := envBool(l, key, def)
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	*dst = value
}

func readDuration(l lookupFunc, key string, def time.Duration, dst *time.Duration, errs *Errors) {
	value, err := envDuration(l, key, def)
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	*dst = value
}

func readBytes(l lookupFunc, key string, def int64, dst *int64, errs *Errors) {
	value, err := envBytes(l, key, def)
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	*dst = value
}

func readBytesInt(l lookupFunc, key string, def int, dst *int, errs *Errors) {
	value, err := envBytes(l, key, int64(def))
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	*dst = int(value)
}
