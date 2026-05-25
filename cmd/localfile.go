package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func readLocalFile(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}
	if strings.Contains(path, "\x00") {
		return nil, errors.New("invalid path")
	}
	clean := filepath.Clean(path)
	dir, file := filepath.Split(clean)
	if file == "" || file == "." {
		return nil, errors.New("path must name a file")
	}
	if strings.Contains(file, "..") {
		return nil, errors.New("invalid file name")
	}
	if dir == "" {
		dir = "."
	}
	return fs.ReadFile(os.DirFS(dir), file)
}

func writeLocalFile(path string, data []byte) error {
	if path == "" {
		return errors.New("path is empty")
	}
	if strings.Contains(path, "\x00") {
		return errors.New("invalid path")
	}
	if !filepath.IsAbs(path) {
		return errors.New("path must be absolute")
	}
	clean := filepath.Clean(path)
	if clean != path {
		return errors.New("invalid path")
	}
	dir := filepath.Dir(clean)
	if dir == "" || dir == "." {
		return errors.New("path must name a file")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	// #nosec G703 -- path is resolved to an absolute, cleaned path before writing.
	return os.WriteFile(clean, data, 0o600)
}
