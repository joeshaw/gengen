package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"

	"code.google.com/p/go.tools/astutil"
	"gopkg.in/alecthomas/kingpin.v1"
	"labix.org/v2/pipe"
)

const pkgPath = "github.com/joeshaw/gengen/generic"
const genericPkg = "generic"

var genericTypes = []string{"T", "U", "V"}

func generate(out io.Writer, filename string, typenames ...string) error {
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

	err = format.Node(out, fset, f)
	return err
}

var (
	outFileFlag  = kingpin.Flag("output", "Output to file (defaults to stdout).").Short('o').PlaceHolder("FILE").String()
	rewritesFlag = kingpin.Flag("rewrite", "Rewrite clause for gofmt.").Short('r').Strings()
	filenameArg  = kingpin.Arg("file.go", "Input template filename.").Required().ExistingFile()
	typeArgs     = kingpin.Arg("replacement types...", "Types to substitute.").Required().Strings()
)

func main() {
	kingpin.CommandLine.Help = fmt.Sprintf("example: %s -o list_int_string.go list.go int string", kingpin.CommandLine.Name)
	kingpin.Parse()

	r, w, err := os.Pipe()
	kingpin.FatalIfError(err, "")

	p := pipe.Line(pipe.Read(r))
	for _, rewrite := range *rewritesFlag {
		p = pipe.Line(p, pipe.Exec("gofmt", "-r", rewrite))
	}
	if *outFileFlag != "" {
		p = pipe.Line(p, pipe.WriteFile(*outFileFlag, 0666))
	} else {
		p = pipe.Line(p, pipe.Write(os.Stdout))
	}

	err = generate(w, *filenameArg, *typeArgs...)
	w.Close()
	kingpin.FatalIfError(err, "")

	err = pipe.Run(p)
	kingpin.FatalIfError(err, "")
}
