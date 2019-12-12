package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/joeshaw/gengen/genlib"
	"golang.org/x/tools/imports"
)

var defaultNames = []string{"T", "U", "V"}

func main() {
	var (
		outDir     = flag.String("o", ".", "output directory")
		fixImports = flag.Bool("i", true, "run go files through `goimports`")
	)
	flag.Parse()

	if flag.NArg() < 2 {
		cmd := os.Args[0]
		fmt.Fprintf(os.Stderr, "usage: %s [-o <output_dir>] <package> <replacement types...>\n", cmd)
		fmt.Fprintf(os.Stderr, "example: %s -o ./btree github.com/joeshaw/gengen/examples/btree string string\n", cmd)
		fmt.Fprintf(os.Stderr, "example: %s -o ./btree github.com/joeshaw/gengen/examples/btree T=string U=string\n", cmd)
		os.Exit(1)
	}

	nakedTypeCount := 0

	types := make(map[string]string)
	for i := 1; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		if strings.Contains(arg, "=") {
			kv := strings.SplitN(arg, "=", 2)
			if len(kv[0]) == 1 && strings.Compare(kv[0], "A") >= 0 && strings.Compare(kv[0], "Z") <= 0 {
				types[kv[0]] = kv[1]
			} else {
				fmt.Fprintf(os.Stderr, "'%s' is an invalid name for a type alias must be a letter between 'A' and 'Z'\n", kv[0])
				os.Exit(1)
			}
		} else {
			if nakedTypeCount < len(defaultNames) {
				types[defaultNames[nakedTypeCount]] = arg
				nakedTypeCount++
			} else {
				fmt.Fprintf(os.Stderr, "Passed in more than %d types without an alias\n", len(defaultNames))
				os.Exit(1)
			}
		}
	}

	// run a "go get <pkg>"
	err := exec.Command("go", "get", flag.Arg(0)).Run()
	if err != nil {
		die(err)
	}

	// resolve the path into which we (might have) just installed it
	pkgPath := findPkgPath(flag.Arg(0))
	if pkgPath == "" {
		die(fmt.Errorf("couldn't find %s", flag.Arg(0)))
	}

	// list the source files
	sourceFiles, err := filepath.Glob(filepath.Join(pkgPath, "*.go"))
	if err != nil {
		die(err)
	}

	// create a temporary directory
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		die(err)
	}

	// convert all source files into the tmp dir
	for _, sourcePath := range sourceFiles {
		destPath := filepath.Join(tempDir, filepath.Base(sourcePath))
		err := convertFile(destPath, sourcePath, *fixImports, types)
		if err != nil {
			die(err)
		}
	}

	// move the converted files into our output dir
	replaceFiles(tempDir, *outDir)

	// remove the temporary directory
	os.RemoveAll(tempDir)
}

func convertFile(destPath, sourcePath string, fixImports bool, types map[string]string) error {
	buf, err := genlib.Generate(sourcePath, types)
	if err != nil {
		return err
	}

	if fixImports {
		buf, err = imports.Process(filepath.Base(sourcePath), buf, &imports.Options{
			TabWidth:  8,
			TabIndent: true,
			Comments:  true,
			Fragment:  true,
			AllErrors: false,
		})
		if err != nil {
			return err
		}
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, bytes.NewBuffer(buf))
	if err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

func replaceFiles(sourceDir, destDir string) {
	sources, err := filepath.Glob(filepath.Join(sourceDir, "*.go"))
	if err != nil {
		die(err)
	}

	if !exists(destDir) {
		err := os.MkdirAll(destDir, 0755)
		if err != nil {
			die(err)
		}
	}

	for _, source := range sources {
		dest := filepath.Join(destDir, filepath.Base(source))

		// attempt a simple rename
		err := os.Rename(source, dest)
		if err == nil {
			continue
		}

		// /tmp is often a ramdisk so check for EXDEV
		linkerr, ok := err.(*os.LinkError)
		if !ok {
			die(err)
		}
		errno, ok := linkerr.Err.(syscall.Errno)
		if !ok {
			die(err)
		}
		if errno != syscall.EXDEV {
			die(err)
		}

		// have to copy the bytes explicitly
		if err = copyBytes(source, dest); err != nil {
			die(err)
		}
	}
}

func copyBytes(source, dest string) error {
	sfile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sfile.Close()

	dfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err = io.Copy(dfile, sfile); err != nil {
		dfile.Close()
		return err
	}

	return dfile.Close()
}

func findPkgPath(name string) string {
	for _, dir := range filepath.SplitList(os.Getenv("GOPATH")) {
		fullPath := filepath.Join(dir, "src", name)
		if exists(fullPath) {
			return fullPath
		}
	}
	return ""
}

func exists(fpath string) bool {
	_, err := os.Stat(fpath)
	return !os.IsNotExist(err)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
