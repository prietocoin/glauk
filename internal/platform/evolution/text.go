package evolution

import (
	"context"
	"errors"
	"strings"
)

// SendText envía un mensaje de texto; no conoce JIDs ni reglas del dominio.
// Solo valida Number y Text; el transporte y los reintentos viven en send.
func (c *Client) SendText(ctx context.Context, in TextInput) (Receipt, error) {
	if err := validateText(in); err != nil {
		return Receipt{}, err
	}
	if c.Simulated() {
		return c.simulate("text", in.Number, in.Text), nil
	}
	return c.send(ctx, c.path("sendText"), in)
}

// validateText only checks the two mandatory Evolution fields.
func validateText(in TextInput) error {
	if strings.TrimSpace(in.Number) == "" {
		return errors.New("evolution: number es obligatorio")
	}
	if strings.TrimSpace(in.Text) == "" {
		return errors.New("evolution: text es obligatorio")
	}
	return nil
}
