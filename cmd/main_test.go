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
	assert.Contains(t, code, "getUsers: (query?: { page?: number; limit?: number; status?: \"active\" | \"inactive\" | \"banned\" })", "should generate getUsers method with query object")
	assert.Contains(t, code, "createUser: (body: CreateUserRequest): Promise<FetchResponse<User>>", "should generate createUser method")
	assert.Contains(t, code, "updateUser: (id: string, body: UpdateUserRequest): Promise<FetchResponse<User>>", "should generate updateUser method")
	assert.Contains(t, code, "deleteUser: (id: string, query?: { force?: boolean })", "should generate deleteUser method with query object")

	// Multi-parameter path support
	assert.Contains(t, code, "getUserPost: (user_id: string, post_id: string, query?: { include_comments?: boolean })", "should handle multi-parameter paths with query objects")
	assert.Contains(t, code, "getTeamMember: (org_id: string, team_id: string, member_id: string", "should handle 3-parameter paths")

	// Dynamic URL construction with query parameters
	assert.Contains(t, code, "const url = `/users` + (queryString ? '?' + queryString : '');", "should build dynamic URLs for query params")
	assert.Contains(t, code, "return client.get(url);", "should call GET with dynamic URL")
	assert.Contains(t, code, "return client.post(`/users`, body);", "should call POST with static URL when no query params")
	assert.Contains(t, code, "return client.put(`/users/${id}`, body);", "should call PUT with path params")
	assert.Contains(t, code, "return client.del(url);", "should call DELETE with dynamic URL")

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
	assert.Contains(t, code, "ssoCallback: (provider: string, query?: { code?: string; state?: string }): Promise<FetchResponse<boolean>>", "307 redirect should return boolean")
	assert.Contains(t, code, "ssoLogin: (provider: string, query?: { email?: string; return_url?: string }): Promise<FetchResponse<boolean>>", "307 redirect should return boolean")
	assert.Contains(t, code, "verifyEmail: (query?: { email?: string; token?: string }): Promise<FetchResponse<boolean>>", "307 redirect should return boolean")

	// Check 204 No Content returns boolean
	assert.Contains(t, code, "logout: (): Promise<FetchResponse<boolean>>", "204 No Content should return boolean")
	assert.Contains(t, code, "getJWKS: (): Promise<FetchResponse<JWKSResponse>>", "should return typed response when 200 with content exists")

	// Check normal JSON responses
	assert.Contains(t, code, "detectSSOProviders: (query?: { email?: string }): Promise<FetchResponse<Array<string>>>", "should return array for 200 response")
	assert.Contains(t, code, "getCurrentUser: (): Promise<FetchResponse<UserIdentity>>", "should return UserIdentity for 200 response")
	assert.Contains(t, code, "getVerificationStatus: (): Promise<FetchResponse<EmailVerificationStatus>>", "should return EmailVerificationStatus for 200 response")
	assert.Contains(t, code, "resendVerification: (): Promise<FetchResponse<EmailVerificationStatus>>", "should return EmailVerificationStatus for 200 response")

	// Check generated interfaces
	assert.Contains(t, code, "export interface EmailVerificationStatus", "should generate EmailVerificationStatus interface")
	assert.Contains(t, code, "export interface JWKSResponse", "should generate JWKSResponse interface")
	assert.Contains(t, code, "export interface ProblemDetails", "should generate ProblemDetails interface")
	assert.Contains(t, code, "export interface UserIdentity", "should generate UserIdentity interface")
}
