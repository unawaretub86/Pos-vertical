//go:build ignore

// Script de build: copia frontend/out → web/out
// Uso: go run scripts/copy-frontend.go
// Funciona en Windows, Linux y Mac.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	src := filepath.Join("frontend", "out")
	dst := filepath.Join("web", "out")

	if err := os.RemoveAll(dst); err != nil {
		fmt.Fprintf(os.Stderr, "error limpiando %s: %v\n", dst, err)
		os.Exit(1)
	}

	if err := copyDir(src, dst); err != nil {
		fmt.Fprintf(os.Stderr, "error copiando frontend: %v\n", dst, err)
		os.Exit(1)
	}

	fmt.Println("✓ frontend/out → web/out")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
