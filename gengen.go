package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

const pkgPath = "github.com/joeshaw/gengen/generic"
const genericPkg = "generic"

var genericTypes = []string{"T", "U", "V"}

func generate(filename string, typenames ...string) ([]byte, error) {
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
				//				return ast.NewIdent(typenames[i])
			}
		}

		return node
	}, f).(*ast.File)

	if !astutil.UsesImport(f, pkgPath) {
		astutil.DeleteImport(fset, f, pkgPath)
	}

	var buf bytes.Buffer
	err = format.Node(&buf, fset, f)
	return buf.Bytes(), err
}

func main() {
	var outfile = flag.String("o", "", "output file")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s [-o <output.go>] <file.go> <replacement types...>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "example: %s -o lists.go list_gen.go int string\n", os.Args[0])
		os.Exit(1)
	}

	buf, err := generate(flag.Arg(0), flag.Args()[1:]...)
	if err != nil {
		die(err)
	}

	if *outfile == "" {
		_, err = io.Copy(os.Stdout, bytes.NewBuffer(buf))
	} else {
		err = ioutil.WriteFile(*outfile, buf, 0644)
	}

	if err != nil {
		die(err)
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
