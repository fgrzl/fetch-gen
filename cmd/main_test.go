package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldReturnErrorGivenMissingCliArgumentsWhenRunningThenFail(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	os.Args = []string{"fetch-gen"}

	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid arguments")
}

func TestShouldWriteOutputFileGivenValidOpenAPISpecWhenRunningThenCreateOutput(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join("..", "tests", "fixtures", "openapi-test.yaml")
	outputPath := filepath.Join(tmpDir, "api.ts")
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	os.Args = []string{
		"fetch-gen",
		"--input", inputPath,
		"--output", outputPath,
	}

	err := run()
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}
