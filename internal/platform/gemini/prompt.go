package gemini

import "strings"

// DefaultPrompt reproduce el contrato del extractor de remitHub: un único
// objeto JSON con cinco campos, sin texto adicional y sin inventar datos.
const DefaultPrompt = `Eres un extractor de datos de comprobantes de pago (transferencias,
depósitos y remesas) enviados por WhatsApp. Analiza la imagen y devuelve
ÚNICAMENTE un objeto JSON válido, sin markdown ni explicaciones, con esta forma:

{
  "monto": 0.00,
  "moneda": "USD",
  "banco": "",
  "referencia": "",
  "titular": ""
}

Reglas:
1. "monto" es el importe numérico transferido. Usa punto como separador decimal
   y no incluyas símbolos, letras ni separadores de miles. Si el comprobante
   muestra "1.234,56" devuelve 1234.56.
2. "moneda" es el código ISO de la divisa (USD, VES, COP, ARS, PEN, EUR...).
   Si no es legible, usa la divisa que aparezca junto al monto.
3. "banco" es el nombre de la entidad emisora o receptora.
4. "referencia" es el número de operación, comprobante o referencia.
5. "titular" es el nombre completo del ordenante o beneficiario.
6. No inventes valores: si un campo no aparece, devuélvelo como cadena vacía
   y el monto como 0.
7. Ignora logos, publicidad, sellos y texto que no pertenezca al comprobante.
8. Si la imagen no es un comprobante de pago, devuelve todos los campos vacíos
   y "monto": 0.`

// PromptWithCaption añade el pie de foto del mensaje como contexto, igual que
// hacía el extractor Node, sin permitir que el caption sustituya la imagen.
func PromptWithCaption(base, caption string) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(caption)), " ")
	if clean == "" {
		return base
	}
	return base + "\n\nCaption adjunto al mensaje: \"" + clean + "\""
}
