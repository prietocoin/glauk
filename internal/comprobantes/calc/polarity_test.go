package calc

import "testing"

func TestSideSignsPolaridadYHerencia(t *testing.T) {
	positivo := CurrencyConfig{Polaridad: "+"}
	negativo := CurrencyConfig{Polaridad: "-"}
	cases := []struct {
		name         string
		cfg1, cfg2   SocioConfig
		tipo, socio1 string
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
			name: "abono_con_fundda_como_socio1",
			cfg1: SocioConfig{Active: true, Conf: positivo},
			cfg2: SocioConfig{Active: true, Conf: positivo},
			tipo: "A", socio1: " FUNDDA ", want1: -1, want2: 1,
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
			got1, got2 := SideSigns(tc.cfg1, tc.cfg2, tc.tipo, tc.socio1)
			if got1 != tc.want1 || got2 != tc.want2 {
				t.Fatalf("SideSigns = (%d, %d), quiero (%d, %d)", got1, got2, tc.want1, tc.want2)
			}
		})
	}
}

func TestEffectivePolarityUsaContratoYRespaldo(t *testing.T) {
	cases := []struct {
		name string
		conf CurrencyConfig
		tipo string
		want string
	}{
		{"deposito_usa_polar_D", CurrencyConfig{PolarD: "-", PolarP: "+", Polaridad: "+"}, "D", "-"},
		{"pago_usa_polar_P", CurrencyConfig{PolarD: "-", PolarP: "+", Polaridad: "+"}, "P", "+"},
		{"deposito_legacy", CurrencyConfig{Polaridad: "-"}, "D", "-"},
		{"pago_legacy_es_contrapartida", CurrencyConfig{Polaridad: "+"}, "P", "-"},
		{"pago_legacy_negativo_es_contrapartida", CurrencyConfig{Polaridad: "-"}, "P", "+"},
		{"sin_datos_deposito_es_positivo", CurrencyConfig{}, "D", "+"},
		{"sin_datos_pago_es_negativo", CurrencyConfig{}, "P", "-"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectivePolarity(tc.conf, tc.tipo); got != tc.want {
				t.Fatalf("EffectivePolarity = %q, quiero %q", got, tc.want)
			}
		})
	}
}

func TestNaturalSignContratoDYP(t *testing.T) {
	contrato := SocioConfig{Active: true, Conf: CurrencyConfig{
		PolarD: "-", PolarP: "+", Polaridad: "+",
	}}
	if got := NaturalSign(contrato, "D"); got != -1 {
		t.Fatalf("D con polar_D=- = %d", got)
	}
	if got := NaturalSign(contrato, "P"); got != 1 {
		t.Fatalf("P con polar_P=+ = %d", got)
	}
	if got := NaturalSign(SocioConfig{}, "D"); got != 1 {
		t.Fatalf("GENERAL = %d", got)
	}
	if OperationNature("") != "D" || OperationNature(" pago ") != "P" || OperationNature("abono") != "A" {
		t.Fatal("OperationNature")
	}
}

func TestPolarityOf(t *testing.T) {
	if PolarityOf(-0.01) != "-" || PolarityOf(0) != "+" || PolarityOf(14.45) != "+" {
		t.Fatal("PolarityOf")
	}
}
