package ingesta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"glauk/internal/comprobantes/calc"
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

// upsertFinalSQL conserva las 39 columnas de glaukapi. Se ejecuta solo si la
// tabla existe y el operador activó INGESTA_ESCRIBIR_FINAL.
const upsertFinalSQL = `
INSERT INTO comprobantes_final (
    id, hash_largo, hash_corto,
    timestamp_msg, fecha_hora_iso, fecha_fmt, hora_fmt, dia_hora_humano,
    monto, moneda, banco, titular, referencia, tipo, tipo_manual,
    lote, lote_tasa_asignado, es_x2, link_img, url_r2_comprobante,
    nombre_socio_1, socio_1, rol_1, tasa_1, m1_socio, monto_1, me1,
    moneda_base_socio1, polaridad_1,
    nombre_socio_2, socio_2, rol_2, tasa_2, m2_socio, monto_2, me2,
    moneda_base_socio2, polaridad_2, grupo_1, grupo_2, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
    $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29,
    $30, $31, $32, $33, $34, $35, $36, $37, $38, $39::jsonb, $40::jsonb,
    CURRENT_TIMESTAMP
)
ON CONFLICT (hash_largo) DO UPDATE SET
    monto = EXCLUDED.monto, moneda = EXCLUDED.moneda, banco = EXCLUDED.banco,
    titular = EXCLUDED.titular, referencia = EXCLUDED.referencia,
    tipo = EXCLUDED.tipo, tipo_manual = EXCLUDED.tipo_manual,
    lote = EXCLUDED.lote, lote_tasa_asignado = EXCLUDED.lote_tasa_asignado,
    es_x2 = EXCLUDED.es_x2,
    link_img = COALESCE(EXCLUDED.link_img, comprobantes_final.link_img),
    url_r2_comprobante = COALESCE(EXCLUDED.url_r2_comprobante,
        comprobantes_final.url_r2_comprobante),
    nombre_socio_1 = EXCLUDED.nombre_socio_1, socio_1 = EXCLUDED.socio_1,
    rol_1 = EXCLUDED.rol_1, tasa_1 = EXCLUDED.tasa_1,
    m1_socio = EXCLUDED.m1_socio, monto_1 = EXCLUDED.monto_1, me1 = EXCLUDED.me1,
    moneda_base_socio1 = EXCLUDED.moneda_base_socio1,
    polaridad_1 = EXCLUDED.polaridad_1,
    nombre_socio_2 = EXCLUDED.nombre_socio_2, socio_2 = EXCLUDED.socio_2,
    rol_2 = EXCLUDED.rol_2, tasa_2 = EXCLUDED.tasa_2,
    m2_socio = EXCLUDED.m2_socio, monto_2 = EXCLUDED.monto_2, me2 = EXCLUDED.me2,
    moneda_base_socio2 = EXCLUDED.moneda_base_socio2,
    polaridad_2 = EXCLUDED.polaridad_2,
    grupo_1 = EXCLUDED.grupo_1, grupo_2 = EXCLUDED.grupo_2,
    updated_at = CURRENT_TIMESTAMP`

// GuardarLiquidacion escribe el snapshot canónico y, si aplica, la copia
// legacy. Un fallo en comprobantes_final no invalida el trabajo: la tabla
// canónica ya quedó persistida.
func (r *Repository) GuardarLiquidacion(ctx context.Context, in LiquidacionInput,
	result LiquidacionResult, escribirFinal bool) error {
	snapshot := result.Snapshot
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
	if !escribirFinal || !r.finalDisponible(ctx) {
		return nil
	}
	_ = r.guardarFinal(ctx, in, result, socio2)
	return nil
}

func (r *Repository) guardarFinal(ctx context.Context, in LiquidacionInput,
	result LiquidacionResult, socio2 string) error {
	snapshot := result.Snapshot
	iso, fecha, hora, humano := dateParts(in.Timestamp)
	_, err := r.db.Exec(ctx, upsertFinalSQL,
		snapshot.HashLargo, snapshot.HashLargo, in.HashCorto,
		in.Timestamp, iso, fecha, hora, humano,
		in.Monto, in.Moneda, in.Banco, in.Titular, in.Referencia,
		result.TipoManual, result.TipoManual,
		snapshot.LoteTasa, snapshot.LoteTasa, true, in.URLImagen, in.URLImagen,
		snapshot.Socio1, snapshot.Socio1, result.Rol1, snapshot.Tasa1,
		snapshot.Monto1, snapshot.Monto1, snapshot.ME1,
		result.MonedaBase1, result.Polaridad1,
		socio2, socio2, result.Rol2, snapshot.Tasa2,
		snapshot.Monto2, snapshot.Monto2, snapshot.ME2,
		result.MonedaBase2, result.Polaridad2,
		profileJSON(result.Perfil1), profileJSON(result.Perfil2))
	if err != nil {
		return r.sqlFail("upsert_comprobantes_final", err)
	}
	return nil
}

// dateParts usa el huso fijo de Venezuela para no depender de tzdata en el
// contenedor; el legado mostraba la fecha en hora local de Caracas.
func dateParts(timestamp int64) (string, string, string, string) {
	location := time.FixedZone("VET", -4*60*60)
	value := time.Now().In(location)
	if timestamp > 0 {
		value = time.Unix(timestamp, 0).In(location)
	}
	iso := value.Format(time.RFC3339)
	fecha := value.Format("02/01/2006")
	hora := value.Format("15:04")
	return iso, fecha, hora, fecha + " " + hora
}

func profileJSON(profile calc.Profile) string {
	raw, err := json.Marshal(profile)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
