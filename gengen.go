package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/joeshaw/gengen/genlib"
)

func main() {
	var outfile = flag.String("o", "", "output file")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s [-o <output.go>] <file.go> <replacement types...>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "example: %s -o lists.go list_gen.go int string\n", os.Args[0])
		os.Exit(1)
	}

	buf, err := genlib.Generate(flag.Arg(0), flag.Args()[1:]...)
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
