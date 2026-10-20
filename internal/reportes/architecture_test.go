package reportes

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// allowedImports es la lista cerrada del §3.10. Si aparece pgx, postgres,
// tasas, perfiles, comprobantes o Fiber, el test falla aunque compile.
var allowedImports = map[string]bool{
	"bytes": true, "context": true, "crypto/sha256": true,
	"encoding/base64": true, "errors": true, "fmt": true,
	"image": true, "image/jpeg": true, "regexp": true,
	"strings": true, "sync": true, "time": true,
}

func TestAislamientoHexagonal(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	fset := token.NewFileSet()
	revisados := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		revisados++
		parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsear %s: %v", file, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			spec, ok := node.(*ast.ImportSpec)
			if !ok {
				return true
			}
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Errorf("%s: import inválido %s", file, spec.Path.Value)
				return true
			}
			if !allowedImports[path] {
				t.Errorf("%s importa %q: prohibido por el aislamiento hexagonal", file, path)
			}
			return true
		})
	}
	if revisados != 10 {
		t.Fatalf("archivos de producción revisados = %d, quiero 10", revisados)
	}
}
