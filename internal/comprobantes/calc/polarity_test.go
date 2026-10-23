package calc

import "testing"

func TestSideSignsPolaridadYHerencia(t *testing.T) {
	positivo := CurrencyConfig{Polaridad: "+"}
	negativo := CurrencyConfig{Polaridad: "-"}
	cases := []struct {
		name         string
		cfg1, cfg2   SocioConfig
		tipo         string
		want1, want2 int
	}{
		{
			name: "deposito_polaridad_positiva_ambos",
			cfg1: SocioConfig{Active: true, Conf: positivo},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "D", want1: 1, want2: 1,
		},
		{
			name: "deposito_polaridad_negativa_ambos",
			cfg1: SocioConfig{Active: true, Conf: negativo},
			cfg2: SocioConfig{Active: true, Conf: negativo},
			tipo: "D", want1: -1, want2: -1,
		},
		{
			name: "pago_polaridad_positiva_ambos",
			cfg1: SocioConfig{Active: true, Conf: positivo},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "P", want1: -1, want2: -1,
		},
		{
			name: "pago_polaridad_negativa_ambos",
			cfg1: SocioConfig{Active: true, Conf: negativo},
			cfg2: SocioConfig{Active: true, Conf: negativo},
			tipo: "P", want1: 1, want2: 1,
		},
		{
			name: "deposito_hereda_socio1",
			cfg1: SocioConfig{Active: true, Conf: positivo, HereDa: true},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "D", want1: -1, want2: 1,
		},
		{
			name: "deposito_hereda_socio2",
			cfg1: SocioConfig{Active: true, Conf: positivo},
			cfg2: SocioConfig{Active: true, Conf: positivo, HereDa: true},
			tipo: "D", want1: 1, want2: -1,
		},
		{
			name: "deposito_ambos_heredan_sin_override",
			cfg1: SocioConfig{Active: true, Conf: positivo, HereDa: true},
			cfg2: SocioConfig{Active: true, Conf: positivo, HereDa: true},
			tipo: "D", want1: 1, want2: 1,
		},
		{
			name: "abono_es_contrapartida",
			cfg1: SocioConfig{Active: true, Conf: positivo},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "A", want1: 1, want2: -1,
		},
		{
			name: "abono_con_herencia_conserva_contrapartida",
			cfg1: SocioConfig{Active: true, Conf: positivo, HereDa: true},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "A", want1: 1, want2: -1,
		},
		{
			name: "general_inactivo_no_ancla",
			cfg1: SocioConfig{},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "D", want1: 1, want2: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1, got2 := SideSigns(tc.cfg1, tc.cfg2, tc.tipo)
			if got1 != tc.want1 || got2 != tc.want2 {
				t.Fatalf("SideSigns = (%d, %d), quiero (%d, %d)", got1, got2, tc.want1, tc.want2)
			}
		})
	}
}

func TestPolarityOf(t *testing.T) {
	if PolarityOf(-0.01) != "-" || PolarityOf(0) != "+" || PolarityOf(14.45) != "+" {
		t.Fatal("PolarityOf")
	}
}
