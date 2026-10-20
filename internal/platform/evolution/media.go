package evolution

import (
	"context"
	"errors"
	"strings"
)

// SendMedia envía una imagen o documento; solo valida los campos mínimos.
func (c *Client) SendMedia(ctx context.Context, in MediaInput) (Receipt, error) {
	if strings.TrimSpace(in.Number) == "" {
		return Receipt{}, errors.New("evolution: number es obligatorio")
	}
	if strings.TrimSpace(in.Media) == "" {
		return Receipt{}, errors.New("evolution: media es obligatorio")
	}
	if c.Simulated() {
		return c.simulate("media", in.Number, in.Caption), nil
	}

	in.MediaType = firstNonEmpty(in.MediaType, "image")
	in.MimeType = firstNonEmpty(in.MimeType, "image/jpeg")
	return c.send(ctx, c.path("sendMedia"), in)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
