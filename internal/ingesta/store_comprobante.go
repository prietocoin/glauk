package ingesta

import (
	"context"
	"fmt"

	"glauk/internal/platform/gemini"
)

// upsertExtraccionSQL replica el ensamblador de remitHub y añade url_r2 e
// instancia. estado_ia y procesado_ia son las columnas que el panel ya lee.
const upsertExtraccionSQL = `
INSERT INTO comprobantes_raw (
    hash_largo, monto, moneda, banco, referencia, titular,
    procesado_ia, url_r2, estado_ia, instancia, creado_en
)
VALUES (
    $1::varchar, $2::numeric, $3::varchar, $4::varchar, $5::varchar, $6::varchar,
    TRUE, $7::text, 'PROCESADO', $8::varchar, NOW()
)
ON CONFLICT (hash_largo) DO UPDATE SET
    monto = EXCLUDED.monto,
    moneda = EXCLUDED.moneda,
    banco = EXCLUDED.banco,
    referencia = EXCLUDED.referencia,
    titular = EXCLUDED.titular,
    url_r2 = COALESCE(NULLIF(EXCLUDED.url_r2, ''), comprobantes_raw.url_r2),
    instancia = COALESCE(NULLIF(EXCLUDED.instancia, ''), comprobantes_raw.instancia),
    estado_ia = 'PROCESADO',
    procesado_ia = TRUE`

// GuardarExtraccion escribe la salida de Gemini sin tocar comprobantes_liq:
// la liquidación es un paso posterior y puede fallar por falta de tasas sin
// perder la lectura de la imagen.
func (r *Repository) GuardarExtraccion(ctx context.Context, hash, instancia,
	url string, extraccion gemini.Result) error {
	hash = normalizeHash(hash)
	if _, err := r.db.Exec(ctx, upsertExtraccionSQL,
		hash, extraccion.Monto, extraccion.Moneda, extraccion.Banco,
		extraccion.Referencia, extraccion.Titular, url, instancia); err != nil {
		return r.sqlFail("upsert_comprobantes_raw",
			fmt.Errorf("ingesta: guardar extracción: %w", err))
	}
	return nil
}
