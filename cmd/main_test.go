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

	err := run()
	assert.NoError(t, err, "should run successfully")

	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err, "should generate api.ts")
	code := string(content)

	// Function structure - should use createAdapter pattern
	assert.Contains(t, code, "export function createAdapter(client: FetchClient)", "should generate createAdapter function")
	assert.Contains(t, code, "import type { FetchClient, FetchResponse }", "should import FetchResponse type")

	// New query object pattern functions
	assert.Contains(t, code, "getUsers: (query?: { page?: number; limit?: number; status?: \"active\" | \"inactive\" | \"banned\" }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string })", "should generate getUsers method with query object")
	assert.Contains(t, code, "createUser: (body: CreateUserRequest, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<User>>", "should generate createUser method")
	assert.Contains(t, code, "updateUser: (id: string, body: UpdateUserRequest, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<User>>", "should generate updateUser method")
	assert.Contains(t, code, "deleteUser: (id: string, query?: { force?: boolean }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string })", "should generate deleteUser method with query object")

	// Multi-parameter path support
	assert.Contains(t, code, "getUserPost: (user_id: string, post_id: string, query?: { include_comments?: boolean }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string })", "should handle multi-parameter paths with query objects")
	assert.Contains(t, code, "getTeamMember: (org_id: string, team_id: string, member_id: string", "should handle 3-parameter paths")

	// Dynamic URL construction with query parameters
	assert.Contains(t, code, "const url = `/users` + (queryString ? '?' + queryString : '');", "should build dynamic URLs for query params")
	assert.Contains(t, code, "return client.get(url, undefined, finalOptions);", "should call GET with dynamic URL and finalOptions")
	assert.Contains(t, code, "return client.post(`/users`, body, undefined, finalOptions);", "should call POST with static URL when no query params and finalOptions")
	assert.Contains(t, code, "return client.put(`/users/${id}`, body, undefined, finalOptions);", "should call PUT with path params and finalOptions")
	assert.Contains(t, code, "return client.del(url, undefined, finalOptions);", "should call DELETE with dynamic URL and finalOptions")
	assert.Contains(t, code, "const finalOptions = { ...options, operationId: options?.operationId ?? 'getUsers' };", "should create finalOptions with operationId fallback")

	// Query parameter handling with imported helper function
	assert.Contains(t, code, "import { buildQueryParams } from '@fgrzl/fetch';", "should import buildQueryParams from @fgrzl/fetch")
	assert.Contains(t, code, "const queryString = query ? buildQueryParams(query) : '';", "should use imported buildQueryParams helper")

	// TypeScript model details - check User interface
	assert.Contains(t, code, `status: "active" | "inactive" | "banned" | "pending";`, "should contain enum with all values")
	assert.Contains(t, code, `tags?: Array<string>;`, "should contain optional array")
	assert.Contains(t, code, `metadata?: Record<string, string>;`, "should contain optional map")

	// Check for nested object in profile (should be inlined)
	assert.Contains(t, code, `profile?: {`, "should have profile object")
	assert.Contains(t, code, `website: string`, "should have website property")
	assert.Contains(t, code, `avatar: string`, "should have avatar property")
	assert.Contains(t, code, `bio: string`, "should have bio property")
	assert.Contains(t, code, `age: number`, "should have age property")
	assert.Contains(t, code, `location: string`, "should have location property")
	assert.Contains(t, code, `preferences: {`, "should have preferences nested object")
	assert.Contains(t, code, `theme: "light" | "dark" | "auto"`, "should have theme enum")
	assert.Contains(t, code, `notifications: boolean`, "should have notifications boolean")
}

func TestGenerateRedirectResponses(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join("..", "auth-api.yaml")
	outputPath := filepath.Join(tmpDir, "auth-api.ts")

	os.Args = []string{
		"fetch-gen",
		"--input", inputPath,
		"--output", outputPath,
	}

	err := run()
	assert.NoError(t, err, "should run successfully")

	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err, "should generate auth-api.ts")
	code := string(content)

	// Check redirect responses (307) return boolean
	assert.Contains(t, code, "ssoCallback: (provider: string, query?: { code?: string; state?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>", "307 redirect should return boolean")
	assert.Contains(t, code, "ssoLogin: (provider: string, query?: { email?: string; return_url?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>", "307 redirect should return boolean")
	assert.Contains(t, code, "verifyEmail: (query?: { email?: string; token?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>", "307 redirect should return boolean")

	// Check 204 No Content returns boolean
	assert.Contains(t, code, "logout: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>", "204 No Content should return boolean")
	assert.Contains(t, code, "getJWKS: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<JWKSResponse>>", "should return typed response when 200 with content exists")

	// Check normal JSON responses
	assert.Contains(t, code, "detectSSOProviders: (query?: { email?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<Array<string>>>", "should return array for 200 response")
	assert.Contains(t, code, "getCurrentUser: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<UserIdentity>>", "should return UserIdentity for 200 response")
	assert.Contains(t, code, "getVerificationStatus: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<EmailVerificationStatus>>", "should return EmailVerificationStatus for 200 response")
	assert.Contains(t, code, "resendVerification: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<EmailVerificationStatus>>", "should return EmailVerificationStatus for 200 response")

	// Check generated interfaces
	assert.Contains(t, code, "export interface EmailVerificationStatus", "should generate EmailVerificationStatus interface")
	assert.Contains(t, code, "export interface JWKSResponse", "should generate JWKSResponse interface")
	assert.Contains(t, code, "export interface ProblemDetails", "should generate ProblemDetails interface")
	assert.Contains(t, code, "export interface UserIdentity", "should generate UserIdentity interface")
}

func TestGenerateAnyOfSchemaAsTypeAlias(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join("..", "openapi-anyof.yaml")
	outputPath := filepath.Join(tmpDir, "anyof.ts")

	os.Args = []string{
		"fetch-gen",
		"--input", inputPath,
		"--output", outputPath,
	}

	err := run()
	assert.NoError(t, err, "should run successfully")

	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err, "should generate anyof.ts")
	code := string(content)

	// anyOf component schemas should be emitted as a union type alias.
	assert.Contains(t, code, "export type Pet = Cat | Dog;", "should generate Pet as a union type alias")
	assert.NotContains(t, code, "export interface Pet", "should not generate Pet as an interface")

	// Member schemas should still be regular interfaces.
	assert.Contains(t, code, "export interface Cat", "should generate Cat interface")
	assert.Contains(t, code, "export interface Dog", "should generate Dog interface")
}

func TestGenerateNullableAndNonStringEnums(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join("..", "openapi-nullable-enum.yaml")
	outputPath := filepath.Join(tmpDir, "nullable-enum.ts")

	os.Args = []string{
		"fetch-gen",
		"--input", inputPath,
		"--output", outputPath,
	}

	err := run()
	assert.NoError(t, err, "should run successfully")

	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err, "should generate nullable-enum.ts")
	code := string(content)

	// OpenAPI 3.1 `type: [string, null]` should become `string | null`.
	assert.Contains(t, code, "export type MaybeString = string | null;", "should generate MaybeString as string | null")

	// Enums should emit correct TS literals, including null.
	assert.Contains(t, code, "status: \"active\" | \"inactive\" | null;", "should include null in string enum")
	assert.Contains(t, code, "count?: 0 | 1 | 2;", "should emit numeric enum literals")
	assert.Contains(t, code, "flag?: true | false;", "should emit boolean enum literals")
}
