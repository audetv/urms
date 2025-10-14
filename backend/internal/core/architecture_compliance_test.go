// core/architecture_compliance_test.go
package core_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoreHasNoInfrastructureImports(t *testing.T) {
	files, err := filepath.Glob("./core/**/*.go")
	require.NoError(t, err)

	for _, file := range files {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		require.NoError(t, err)

		for _, imp := range node.Imports {
			if strings.Contains(imp.Path.Value, "infrastructure") {
				t.Errorf("Core file %s imports infrastructure: %s", file, imp.Path.Value)
			}
		}
	}
}
