package gemini

import (
	"encoding/json"
	"strings"
)

// Result es la extracción normalizada que consume el pipeline.
type Result struct {
	Monto      float64 `json:"monto"`
	Moneda     string  `json:"moneda"`
	Banco      string  `json:"banco"`
	Referencia string  `json:"referencia"`
	Titular    string  `json:"titular"`
}

type request struct {
	Contents         []content  `json:"contents"`
	GenerationConfig generation `json:"generationConfig"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     []byte `json:"data"`
}

type generation struct {
	MimeType    string  `json:"responseMimeType"`
	Temperature float64 `json:"temperature"`
}

type response struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (r response) text() string {
	var builder strings.Builder
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			builder.WriteString(part.Text)
		}
	}
	return strings.TrimSpace(builder.String())
}

// Decode convierte un JSON crudo en Result sin aplicar reglas de negocio.
func Decode(raw []byte) (Result, error) {
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Result{}, err
	}
	return FromMap(fields), nil
}
