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
	"syscall"

	"github.com/joeshaw/gengen/genlib"
	"golang.org/x/tools/imports"
)

const (
	usageTemplate = `usage: %s [-o <output_dir>] <package> <replacement types...>
example: %s -o ./btree github.com/joeshaw/gengen/examples/btree string string
`

	osEx        = 0x1
	osWrite     = 0x2
	osRead      = 0x4
	osUserShift = 0x6

	osUserR   = osRead << osUserShift
	osUserW   = osWrite << osUserShift
	osUserX   = osEx << osUserShift
	osUserRW  = osUserR | osUserW
	osUserRWX = osUserRW | osUserX
)

func main() {
	var (
		outDir     = flag.String("o", ".", "output directory")
		fixImports = flag.Bool("i", true, "run go files through `goimports`")
	)
	flag.Parse()

	if flag.NArg() < 2 {
		cmd := os.Args[0]
		fmt.Fprintf(os.Stderr, usageTemplate, cmd, cmd)
		os.Exit(1)
	}

	types := make([]string, flag.NArg()-1)
	for i := 1; i < flag.NArg(); i++ {
		types[i-1] = flag.Arg(i)
	}

	// run a "go get <pkg>"
	// TODO(joeshaw): users probably should go get template-packages themselves
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
		err := convertFile(destPath, sourcePath, *fixImports, types...)
		if err != nil {
			die(err)
		}
	}

	// move the converted files into our output dir
	replaceFiles(tempDir, *outDir)

	// remove the temporary directory
	os.RemoveAll(tempDir)
}

func convertFile(
	destPath, sourcePath string, fixImports bool, types ...string,
) error {
	buf, err := genlib.Generate(sourcePath, types...)
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
	defer f.Close()

	_, err = io.Copy(f, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	return nil
}

func replaceFiles(sourceDir, destDir string) {
	sources, err := filepath.Glob(filepath.Join(sourceDir, "*.go"))
	if err != nil {
		die(err)
	}

	if !exists(destDir) {
		err := os.MkdirAll(destDir, osUserRWX)
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
	defer dfile.Close()

	_, err = io.Copy(dfile, sfile)
	return err
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
