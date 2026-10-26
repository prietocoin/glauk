package ingesta

import (
	"context"
	"fmt"
)

const upsertLiquidacionSQL = `
INSERT INTO comprobantes_liq (
    hash_largo, socio_1, tipo_op1, monto_1, tasa_1, me1,
    socio_2, tipo_op2, monto_2, tasa_2, me2, lote_tasa, actualizado_en
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
ON CONFLICT (hash_largo) DO UPDATE SET
    socio_1 = EXCLUDED.socio_1, tipo_op1 = EXCLUDED.tipo_op1,
    monto_1 = EXCLUDED.monto_1, tasa_1 = EXCLUDED.tasa_1, me1 = EXCLUDED.me1,
    socio_2 = EXCLUDED.socio_2, tipo_op2 = EXCLUDED.tipo_op2,
    monto_2 = EXCLUDED.monto_2, tasa_2 = EXCLUDED.tasa_2, me2 = EXCLUDED.me2,
    lote_tasa = EXCLUDED.lote_tasa, actualizado_en = NOW()`

const perfilExisteSQL = `
SELECT EXISTS (
    SELECT 1
    FROM perfiles_glaukov
    WHERE UPPER(TRIM(nombre)) = UPPER(TRIM($1::text))
)`

// GuardarLiquidacion persiste únicamente el snapshot canónico en
// comprobantes_liq. La copia legacy comprobantes_final ya no se escribe: no
// tenía lectores en ninguno de los módulos y su error se descartaba, de modo
// que solo podía producir deriva silenciosa. comprobantes_liq es la fuente de
// verdad contable.
func (r *Repository) GuardarLiquidacion(ctx context.Context,
	result LiquidacionResult) error {
	snapshot := result.Snapshot
	snapshot.HashLargo = normalizeHash(snapshot.HashLargo)
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf("ingesta: liquidación inválida: %w", err)
	}
	if err := r.validarPerfiles(ctx, snapshot.Socio1, *snapshot.Socio2); err != nil {
		return err
	}
	socio2 := ""
	if snapshot.Socio2 != nil {
		socio2 = *snapshot.Socio2
	}
	if _, err := r.db.Exec(ctx, upsertLiquidacionSQL,
		snapshot.HashLargo, snapshot.Socio1, snapshot.TipoOp1,
		snapshot.Monto1, snapshot.Tasa1, snapshot.ME1,
		socio2, snapshot.TipoOp2, snapshot.Monto2, snapshot.Tasa2,
		snapshot.ME2, snapshot.LoteTasa); err != nil {
		return r.sqlFail("upsert_comprobantes_liq",
			fmt.Errorf("ingesta: guardar liquidación: %w", err))
	}
	return nil
}

func (r *Repository) validarPerfiles(ctx context.Context, socios ...string) error {
	for _, socio := range socios {
		var existe bool
		if err := r.db.QueryRow(ctx, perfilExisteSQL, socio).Scan(&existe); err != nil {
			return r.sqlFail("validar_perfil_liquidacion",
				fmt.Errorf("ingesta: validar socio %q: %w", socio, err))
		}
		if !existe {
			return fmt.Errorf("ingesta: el socio %q no está registrado en perfiles_glaukov", socio)
		}
	}
	return nil
}
