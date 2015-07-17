package genlib

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

const pkgPath = "github.com/joeshaw/gengen/generic"
const genericPkg = "generic"

var genericTypes = []string{"T", "U", "V"}

func Generate(filename string, typenames ...string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	f = replace(func(node ast.Node) ast.Node {
		se, ok := node.(*ast.SelectorExpr)
		if !ok {
			return node
		}

		x, ok := se.X.(*ast.Ident)
		if !ok || x.Name != genericPkg {
			return node
		}

		for i, t := range genericTypes {
			if se.Sel.Name == t {
				return &ast.Ident{NamePos: 0, Name: typenames[i]}
			}
		}

		return node
	}, f).(*ast.File)

	if !astutil.UsesImport(f, pkgPath) {
		astutil.DeleteImport(fset, f, pkgPath)
	}

	var buf bytes.Buffer
	if err = format.Node(&buf, fset, f); err != nil {
		return nil, err
	}

	return format.Source(buf.Bytes())
}
