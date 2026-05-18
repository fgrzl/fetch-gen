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
	clean := filepath.Clean(path)
	dir, file := filepath.Split(clean)
	if file == "" || file == "." {
		return errors.New("path must name a file")
	}
	if strings.Contains(file, "..") {
		return errors.New("invalid file name")
	}
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, file), data, 0o600)
}
