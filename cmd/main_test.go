package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const complexSpec = `
openapi: 3.0.0
info:
  title: Advanced API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
    post:
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/User'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
  /users/{id}:
    put:
      operationId: updateUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/User'
      responses:
        '200':
          description: Updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
    delete:
      operationId: delUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '204':
          description: del
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        status:
          type: string
          enum: [active, inactive, banned]
        tags:
          type: array
          items:
            type: string
        metadata:
          type: object
          additionalProperties:
            type: string
        profile:
          type: object
          properties:
            bio:
              type: string
            age:
              type: integer
`

func TestGenerateFullAPIWithComplexModel(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "openapi.yaml")
	outputPath := filepath.Join(tmpDir, "api.ts")

	err := os.WriteFile(inputPath, []byte(complexSpec), 0644)
	assert.NoError(t, err, "should write OpenAPI spec")

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
	assert.Contains(t, code, "getUser: (): Promise<FetchResponse<Array<User>>>", "should generate getUser method")
	assert.Contains(t, code, "createUser: (body: User): Promise<FetchResponse<User>>", "should generate createUser method")
	assert.Contains(t, code, "updateUser: (id: string, body: User): Promise<FetchResponse<User>>", "should generate updateUser method")
	assert.Contains(t, code, "delUser: (id: string): Promise<FetchResponse<boolean>>", "should generate delUser method")

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
