//go:build ignore

// Build script: copies frontend/out → web/out
// Usage: go run scripts/copy-frontend.go
// Works on Windows, Linux and Mac.
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
		fmt.Fprintf(os.Stderr, "error cleaning %s: %v\n", dst, err)
		os.Exit(1)
	}

	if err := copyDir(src, dst); err != nil {
		fmt.Fprintf(os.Stderr, "error copying frontend to %s: %v\n", dst, err)
		os.Exit(1)
	}

	// web/out/.gitkeep is tracked so the go:embed directory exists in a fresh
	// clone; RemoveAll above deleted it, so recreate it empty (US-091 AC-10).
	gitkeep := filepath.Join(dst, ".gitkeep")
	if err := os.WriteFile(gitkeep, nil, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", gitkeep, err)
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
