// This file implements the offline mode used only when BaseURL is empty.
// Simulation never touches the network and never reads the API key.
package evolution

import (
	"crypto/sha256"
	"encoding/hex"
)

// Prefix and status are stable so callers can assert simulation unambiguously.
const simulationPrefix = "sim-"
const simulationStatus = "SIMULATED"

// simulate produce un recibo determinista: el mismo destino y contenido generan
// el mismo id, lo que permite probar idempotencia sin red.
// Raw queda nil porque no existe una respuesta HTTP que preservar.
// La API key nunca llega a este camino.
func (c *Client) simulate(kind, number, content string) Receipt {
	return Receipt{
		Status:    simulationStatus,
		MessageID: simulationID(kind, number, content),
	}
}

// simulationID uses a truncated SHA-256 so tests can assert stable identifiers.
// The kind is part of the digest to keep media and text ids distinct.
func simulationID(kind, number, content string) string {
	sum := sha256.Sum256([]byte(kind + "|" + number + "|" + content))
	return simulationPrefix + hex.EncodeToString(sum[:8])
}
