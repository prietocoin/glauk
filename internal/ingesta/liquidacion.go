package ingesta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
	tasasstore "glauk/internal/tasas/store"
)

const perfilPorGrupoSQL = `
SELECT to_jsonb(p) FROM perfiles_glaukov p
WHERE UPPER(TRIM(p.id_grupo)) = UPPER(TRIM($1))
   OR UPPER(TRIM(p.nombre)) = UPPER(TRIM($1))
   OR p.id_grupo LIKE $2
LIMIT 1`

const funddaPerfilSQL = `
SELECT to_jsonb(p) FROM perfiles_glaukov p
WHERE UPPER(TRIM(p.nombre)) = 'FUNDDA'
LIMIT 1`

// LiquidadorAdapter conecta el pipeline con el motor canónico de comprobantes
// y con el lote vigente de internal/tasas. No duplica fórmulas financieras.
type LiquidadorAdapter struct {
	db DB
}

// NewLiquidador construye el adaptador con el mismo pool que usa el store.
func NewLiquidador(db DB) *LiquidadorAdapter {
	return &LiquidadorAdapter{db: db}
}

var _ Liquidador = (*LiquidadorAdapter)(nil)

// Liquidar resuelve los dos perfiles por id_grupo, toma el lote vigente de
// tasas y congela el snapshot con calc.CalcularSnapshotFinanciero.
func (l *LiquidadorAdapter) Liquidar(ctx context.Context,
	in LiquidacionInput) (LiquidacionResult, error) {
	if l == nil || l.db == nil {
		return LiquidacionResult{}, errors.New("ingesta: liquidador sin base de datos")
	}
	grupo1, grupo2, err := groupsOf(in.Grupos)
	if err != nil {
		return LiquidacionResult{}, err
	}
	perfil1, err := l.perfil(ctx, grupo1)
	if err != nil {
		return LiquidacionResult{}, err
	}
	perfil2, err := l.perfil(ctx, grupo2)
	if err != nil {
		return LiquidacionResult{}, err
	}
	fundda := l.fundda(ctx)

	lotes, err := tasasstore.RecentLotes(ctx, l.db)
	if err != nil {
		return LiquidacionResult{}, fmt.Errorf("ingesta: leer lote de tasas: %w", err)
	}
	if len(lotes) == 0 {
		return LiquidacionResult{}, errors.New("ingesta: no hay lote de tasas vigente")
	}
	lote := lotes[0]
	rawRates, err := json.Marshal(lote.Tasas)
	if err != nil {
		return LiquidacionResult{}, fmt.Errorf("ingesta: serializar tasas: %w", err)
	}
	rates := calc.DecodeRates(rawRates)

	moneda := strings.ToUpper(strings.TrimSpace(in.Moneda))
	if moneda == "" {
		moneda = "USDT"
	}
	tipo := tipoOperacion(perfil1, fundda, moneda)
	snapshot := calc.CalcularSnapshotFinanciero(
		calc.Raw{
			HashLargo:  in.HashLargo,
			Monto:      in.Monto,
			Moneda:     moneda,
			TipoManual: tipo,
			Caption:    in.Caption,
			IDTasa:     lote.IDTasa,
		},
		perfil1, perfil2,
		calc.RateLot{IDTasa: lote.IDTasa, Tasas: rates},
		fundda,
	)

	return LiquidacionResult{Snapshot: snapshot}, nil
}

func (l *LiquidadorAdapter) perfil(ctx context.Context, grupo string) (calc.Profile, error) {
	clean := strings.TrimSpace(grupo)
	if domain.IsPlaceholderSocio(clean) {
		return calc.Profile{}, fmt.Errorf("ingesta: el grupo %q no es un socio real", grupo)
	}
	numeric := strings.SplitN(clean, "@", 2)[0]
	pattern := "%" + numeric + "%"
	var raw []byte
	err := l.db.QueryRow(ctx, perfilPorGrupoSQL, clean, pattern).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return calc.Profile{}, fmt.Errorf("ingesta: el grupo %q no está registrado en perfiles_glaukov", grupo)
	}
	if err != nil {
		return calc.Profile{}, fmt.Errorf("ingesta: leer perfil %q: %w", grupo, err)
	}
	profile := calc.DecodeProfile(raw)
	if strings.TrimSpace(profile.Nombre) == "" {
		profile.Nombre = clean
	}
	if strings.TrimSpace(profile.MonedaBase) == "" {
		profile.MonedaBase = "USDT"
	}
	if strings.TrimSpace(profile.Rol) == "" {
		profile.Rol = "SOCIO"
	}
	return profile, nil
}

func (l *LiquidadorAdapter) fundda(ctx context.Context) calc.Profile {
	var raw []byte
	if err := l.db.QueryRow(ctx, funddaPerfilSQL).Scan(&raw); err != nil {
		return calc.Profile{}
	}
	return calc.DecodeProfile(raw)
}

// tipoOperacion reproduce la regla de glaukapi: abono si la divisa es la base
// del socio 1; si no, el tipo configurado para esa divisa; por defecto P.
func tipoOperacion(perfil1, fundda calc.Profile, moneda string) string {
	if moneda == baseOf(perfil1) && !calc.IsGeneral(perfil1.Nombre) {
		return "A"
	}
	conf := calc.ResolveSocioConfig(perfil1, moneda, fundda).Conf
	if tipo := strings.ToUpper(strings.TrimSpace(conf.Tipo)); tipo != "" {
		return tipo
	}
	return "P"
}

func groupsOf(grupos []string) (string, string, error) {
	first, second := "", ""
	for _, raw := range grupos {
		clean := strings.TrimSpace(raw)
		if domain.IsPlaceholderSocio(clean) {
			continue
		}
		if first == "" {
			first = clean
			continue
		}
		if second == "" && domain.CanonicalSocio(clean) != domain.CanonicalSocio(first) {
			second = clean
		}
	}
	if first == "" || second == "" {
		return "", "", errors.New("ingesta: se requieren dos grupos distintos y válidos para liquidar")
	}
	return first, second, nil
}

func baseOf(profile calc.Profile) string {
	base := strings.ToUpper(strings.TrimSpace(profile.MonedaBase))
	if base == "" {
		return "USDT"
	}
	return base
}
