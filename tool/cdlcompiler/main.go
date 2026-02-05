package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: cligen <path/to/file.cdl> <package>")
		os.Exit(2)
	}
	inPath := os.Args[1]
	pkgName := os.Args[2]
	inAbs, err := filepath.Abs(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abs path for %s: %v\n", inPath, err)
		os.Exit(1)
	}
	fmt.Printf("Processing %s\n", inAbs)
	if filepath.Ext(inPath) != ".cdl" {
		fmt.Fprintf(os.Stderr, "input must be a .cdl file: %s\n", inPath)
		os.Exit(2)
	}
	content, err := os.ReadFile(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", inPath, err)
		os.Exit(1)
	}
	cdl, err := parseDef(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse %s: %v\n", inPath, err)
		os.Exit(1)
	}

	baseName := strings.TrimSuffix(filepath.Base(inPath), ".cdl")
	outPath := filepath.Join(filepath.Dir(inPath), baseName+".go")
	supPath := filepath.Join(filepath.Dir(inPath), baseName+"_support.go")
	outAbs, err := filepath.Abs(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abs path for %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Writing %s\n", outAbs)
	supAbs, err := filepath.Abs(supPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abs path for %s: %v\n", supPath, err)
		os.Exit(1)
	}
	fmt.Printf("Writing %s\n", supAbs)

	if !isValidIdent(pkgName) {
		fmt.Fprintf(os.Stderr, "invalid package name %q\n", pkgName)
		os.Exit(1)
	}

	src, sup, err := generate(cdl, pkgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", outPath, err)
		os.Exit(1)
	}
	if err := os.WriteFile(supPath, sup, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", supPath, err)
		os.Exit(1)
	}
}
