package calc

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Profile es la porción de perfiles_glaukov usada por el motor financiero.
type Profile struct {
	Nombre     string                    `json:"nombre"`
	Rol        string                    `json:"rol"`
	MonedaBase string                    `json:"moneda_base"`
	Herencia   bool                      `json:"herencia"`
	Monedas    map[string]CurrencyConfig `json:"monedas"`
}

// CurrencyConfig modela la configuración JSONB de una divisa.
type CurrencyConfig struct {
	Activo     bool   `json:"activo"`
	Tipo       string `json:"tipo"`
	PolarD     string `json:"polar_D"`
	PolarP     string `json:"polar_P"`
	Polaridad  string `json:"polaridad"`
	Porcentaje Spread `json:"porcentaje"`
}

// Spread contiene los porcentajes de depósito y pago.
type Spread struct {
	Deposito Decimal `json:"deposito"`
	Pago     Decimal `json:"pago"`
}

// Decimal acepta números o cadenas numéricas provenientes del JSONB.
type Decimal float64

func (d *Decimal) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if raw == "" || raw == "null" {
		*d = 0
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		*d = 0
		return nil
	}
	*d = Decimal(value)
	return nil
}

// DecodeProfile tolera JSONB vacío o inválido sin romper el listado.
func DecodeProfile(raw []byte) Profile {
	if len(raw) == 0 {
		return Profile{}
	}
	var profile Profile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return Profile{}
	}
	return profile
}

// EffectiveCurrency resuelve herencia de FUNDDA y el contrato por defecto.
func EffectiveCurrency(profile, fundda Profile, code string) CurrencyConfig {
	currencies := profile.Monedas
	if profile.Herencia {
		currencies = fundda.Monedas
	}
	if conf, ok := currencies[code]; ok {
		return conf
	}
	return CurrencyConfig{Tipo: "D", PolarD: "+", PolarP: "-"}
}

// SocioConfig es la configuración efectiva de un lado del snapshot.
type SocioConfig struct {
	Active bool
	Conf   CurrencyConfig
	HereDa bool
	Moneda string
}

// ResolveSocioConfig aplica GENERAL, herencia y defaults de polaridad usando el
// nombre y la moneda base del perfil.
func ResolveSocioConfig(profile Profile, divisa string, fundda Profile) SocioConfig {
	return resolveSocioConfig(profile.Nombre, profile, divisa, fundda, profile.MonedaBase)
}

// ResolveLiveSocioConfig resuelve un lado todavía no liquidado con el nombre y
// la moneda que ya expone el listado. Ambos pueden venir del impacto cuando el
// perfil aún no existe, por eso no se derivan solo del JSONB.
func ResolveLiveSocioConfig(name string, profile Profile, divisa string,
	fundda Profile, moneda string) SocioConfig {
	return resolveSocioConfig(name, profile, divisa, fundda, moneda)
}

func resolveSocioConfig(name string, profile Profile, divisa string,
	fundda Profile, moneda string) SocioConfig {
	if IsGeneral(name) {
		return SocioConfig{Moneda: "USDT"}
	}
	hereda := profile.Herencia
	monedas := profile.Monedas
	if hereda && strings.TrimSpace(fundda.Nombre) != "" {
		monedas = fundda.Monedas
	}
	conf, ok := monedas[divisa]
	if !ok {
		conf = CurrencyConfig{Polaridad: "+"}
	}
	if strings.TrimSpace(conf.Polaridad) == "" {
		conf.Polaridad = "+"
	}
	return SocioConfig{
		Active: true,
		Conf:   conf,
		HereDa: hereda,
		Moneda: CurrencyOf(moneda),
	}
}
