package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/joeshaw/gengen/genlib"
	"golang.org/x/tools/imports"
)

func main() {
	var (
		outdir     = flag.String("o", ".", "output directory")
		fixImports = flag.Bool("i", true, "run go files through `goimports`")
	)
	flag.Parse()

	if flag.NArg() < 2 {
		cmd := os.Args[0]
		fmt.Fprintf(os.Stderr, "usage: %s [-o <output_dir>] <package> <replacement types...>\n", cmd)
		fmt.Fprintf(os.Stderr, "example: %s -o ./btree github.com/joeshaw/gengen/examples/btree string string\n", cmd)
		os.Exit(1)
	}

	types := make([]string, flag.NArg()-1)
	for i := 1; i < flag.NArg(); i++ {
		types[i-1] = flag.Arg(i)
	}

	// run a "go get <pkg>"
	err := exec.Command("go", "get", flag.Arg(0)).Run()
	if err != nil {
		die(err)
	}

	// resolve the path into which we (might have) just installed it
	pkgpath := findPkgPath(flag.Arg(0))
	if pkgpath == "" {
		die(fmt.Errorf("couldn't find %s", flag.Arg(0)))
	}

	// list the source files
	sourcefiles, err := filepath.Glob(path.Join(pkgpath, "*.go"))
	if err != nil {
		die(err)
	}

	// create a temporary directory
	tdir, err := ioutil.TempDir("", "")
	if err != nil {
		die(err)
	}

	// convert all source files into the tmp dir
	for _, source_path := range sourcefiles {
		dest_path := path.Join(tdir, path.Base(source_path))
		err := convertFile(dest_path, source_path, *fixImports, types...)
		if err != nil {
			die(err)
		}
	}

	// move the converted files into our output dir
	replaceFiles(tdir, *outdir)

	// remove the temporary directory
	os.RemoveAll(tdir)
}

func convertFile(dest_path, source_path string, fixImports bool, types ...string) error {
	buf, err := genlib.Generate(source_path, types...)
	if err != nil {
		return err
	}

	if fixImports {
		buf, err = imports.Process(path.Base(source_path), buf, &imports.Options{
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

	f, err := os.Create(dest_path)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	return nil
}

func replaceFiles(source_dir, dest_dir string) {
	sources, err := filepath.Glob(path.Join(source_dir, "*.go"))
	if err != nil {
		die(err)
	}

	if !exists(dest_dir) {
		os.MkdirAll(dest_dir, 0755)
	}

	for _, source := range sources {
		dest := path.Join(dest_dir, path.Base(source))

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
	for _, dir := range strings.Split(os.Getenv("GOPATH"), ";") {
		fullpath := path.Join(dir, "src", name)
		if exists(fullpath) {
			return fullpath
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
