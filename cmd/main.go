package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fgrzl/fetch-gen/internal/generator"
	"github.com/fgrzl/fetch-gen/internal/parser"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func run() error {
	if len(os.Args) < 5 || os.Args[1] != "--input" || os.Args[3] != "--output" {
		fmt.Println("Usage: fetch-gen --input openapi.yaml --output ./src/api.ts [--instance ./path/to/client]")
		return fmt.Errorf("invalid arguments")
	}

	inputPath := os.Args[2]
	inputPath, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for input: %w", err)
	}

	outputPath := os.Args[4]
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output: %w", err)
	}

	instance := "@fgrzl/fetch"
	if len(os.Args) >= 7 && os.Args[5] == "--instance" {
		instance = strings.TrimSuffix(os.Args[6], ".ts")
	}

	data, err := readLocalFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	api, err := parser.ParseDocument(inputPath, data)
	if err != nil {
		return err
	}

	out, err := generator.Generate(api, instance)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	if err := writeLocalFile(outputPath, out); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Generated fetch client: %s\n", outputPath)
	return nil
}
