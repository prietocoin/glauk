package store

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"

	"glauk/internal/tasas/domain"
)

// sqlRecentLotes es la única definición de "lote vigente": ordena por id descendente
// como el servicio vivo (el archivo muerto db.service.js lo hacía por created_at, y
// dos definiciones de lo mismo publican el lote equivocado). Las columnas se nombran
// en lugar de usar SELECT * para que un cambio de esquema falle aquí, no en silencio.
const (
	sqlRecentLotes = `SELECT id_tasa, tasas FROM tasas_glaukov ORDER BY id DESC LIMIT 2`
	sqlLoteByID    = `SELECT id_tasa, tasas FROM tasas_glaukov WHERE LOWER(BTRIM(id_tasa)) = LOWER(BTRIM($1))`
)

// nonDigit replica el saneamiento del identificador del servicio original (/\D/g).
var nonDigit = regexp.MustCompile(`\D`)

// RecentLotes devuelve hasta dos lotes: el vigente primero y el anterior después,
// porque comparar ambos es lo que permite calcular la tendencia con una sola lectura.
// Una tabla vacía no es un error: el llamador decide si eso es "sin tasas" o "nada
// que publicar".
func RecentLotes(ctx context.Context, q Querier) ([]domain.Lote, error) {
	return queryMany(ctx, q, sqlRecentLotes, nil, scanLote)
}

// LoteByID devuelve un lote concreto para la publicación por identificador.
// ErrNotFound conserva la distinción entre "no existe" y un fallo del motor.
func LoteByID(ctx context.Context, q Querier, idTasa string) (domain.Lote, error) {
	return queryOne(ctx, q, sqlLoteByID, []any{idTasa}, scanLote)
}

// NextLoteID deriva el identificador del próximo lote a partir del último.
//
// Reproduce el saneamiento del original: se toman solo los dígitos, se suma uno y se
// rellena a tres posiciones. Un identificador sin dígitos produce T001, que es el
// arranque de la serie, igual que con la tabla vacía.
func NextLoteID(lastID string) string {
	number, err := strconv.Atoi(nonDigit.ReplaceAllString(lastID, ""))
	if err != nil {
		number = 0
	}
	return fmt.Sprintf("T%03d", number+1)
}

// scanLote traduce una fila (id_tasa, tasas) al lote del domain.
//
// id_tasa se lee como anulable porque el cálculo ya sabe qué hacer con un lote sin
// identificador: lo publica como T000. Escanearlo en un string fallaría con un nulo,
// y una fila sin identificador no debe tumbar la publicación.
func scanLote(rows pgx.Rows) (domain.Lote, error) {
	var id *string
	var raw []byte
	if err := rows.Scan(&id, &raw); err != nil {
		return domain.Lote{}, fmt.Errorf("escaneando el lote: %w", err)
	}
	tasas, err := decodeJSON[map[string]any](raw, "tasas")
	if err != nil {
		return domain.Lote{}, err
	}
	return domain.Lote{IDTasa: text(id), Tasas: tasas}, nil
}
