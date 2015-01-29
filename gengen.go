package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

const pkgPath = "github.com/joeshaw/gengen/generic"
const genericPkg = "generic"

var genericTypes = []string{"T", "U", "V"}

func generate(filename string, typenames ...string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
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
				return ast.NewIdent(typenames[i])
			}
		}

		return node
	}, f).(*ast.File)

	if !astutil.UsesImport(f, pkgPath) {
		astutil.DeleteImport(fset, f, pkgPath)
	}

	err = format.Node(os.Stdout, fset, f)
	return err
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <file.go> <replacement types...>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "example: %s list.go int string\n", os.Args[0])
		os.Exit(1)
	}

	err := generate(os.Args[1], os.Args[2:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
