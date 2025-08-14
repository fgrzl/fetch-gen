package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFullAPIWithComplexModel(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join("..", "openapi-test.yaml")
	outputPath := filepath.Join(tmpDir, "api.ts")

	os.Args = []string{
		"fetch-gen",
		"--input", inputPath,
		"--output", outputPath,
	}

	main()

	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err, "should generate api.ts")
	code := string(content)

	// Function structure - should use createApi pattern
	assert.Contains(t, code, "export function createApi(client: FetchClient)", "should generate createApi function")
	assert.Contains(t, code, "import type { FetchClient, FetchResponse }", "should import FetchResponse type")
	assert.Contains(t, code, "getUsers: (): Promise<FetchResponse<Array<User>>>", "should generate getUsers method")
	assert.Contains(t, code, "createUser: (body: CreateUserRequest): Promise<FetchResponse<User>>", "should generate createUser method")
	assert.Contains(t, code, "updateUser: (id: string, body: UpdateUserRequest): Promise<FetchResponse<User>>", "should generate updateUser method")
	assert.Contains(t, code, "deleteUser: (id: string): Promise<FetchResponse<boolean>>", "should generate deleteUser method")

	// Client calls
	assert.Contains(t, code, "client.get(`/users`)", "should call GET")
	assert.Contains(t, code, "client.post(`/users`", "should call POST")
	assert.Contains(t, code, "client.put(`/users/${id}`", "should call PUT")
	assert.Contains(t, code, "client.del(`/users/${id}`)", "should call DELETE")

	// TypeScript model details
	assert.Contains(t, code, `status: "active" | "inactive" | "banned";`, "should contain enum")
	assert.Contains(t, code, `tags: Array<string>;`, "should contain array")
	assert.Contains(t, code, `metadata: Record<string, string>;`, "should contain map")
	assert.Contains(t, code, `profile: { bio: string; age: number };`, "should inline nested object")
}
