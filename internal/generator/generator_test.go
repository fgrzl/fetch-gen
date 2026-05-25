package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fgrzl/fetch-gen/internal/generator"
	"github.com/fgrzl/fetch-gen/internal/parser"
	apitypes "github.com/fgrzl/fetch-gen/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateCodeFromFixture(t *testing.T, fixture string) string {
	t.Helper()

	fixturePath, err := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures", fixture))
	require.NoError(t, err)

	content, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	api, err := parser.ParseDocument(fixturePath, content)
	require.NoError(t, err)

	output, err := generator.Generate(api, "")
	require.NoError(t, err)

	return string(output)
}

func generateCodeFromAPI(t *testing.T, api *apitypes.OpenAPI, instance string) string {
	t.Helper()

	output, err := generator.Generate(api, instance)
	require.NoError(t, err)

	return string(output)
}

func TestShouldReturnErrorGivenNilOpenAPIDocumentWhenGeneratingThenFail(t *testing.T) {
	_, err := generator.Generate(nil, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "openapi document is empty")
}

func TestShouldUseDefaultInstanceGivenBlankInstanceWhenGeneratingThenImportDefaultClient(t *testing.T) {
	output, err := generator.Generate(&apitypes.OpenAPI{}, "")
	require.NoError(t, err)
	assert.Contains(t, string(output), "from '@fgrzl/fetch';")
}

func TestShouldUseCustomInstanceGivenProvidedInstanceWhenGeneratingThenImportCustomClient(t *testing.T) {
	output, err := generator.Generate(&apitypes.OpenAPI{}, "./src/custom")
	require.NoError(t, err)
	assert.Contains(t, string(output), "from './src/custom';")
}

func TestShouldExportCreateAdapterGivenEmptyOpenAPIDocumentWhenGeneratingThenEmitFactory(t *testing.T) {
	output, err := generator.Generate(&apitypes.OpenAPI{}, "")
	require.NoError(t, err)
	assert.Contains(t, string(output), "export function createAdapter(client: FetchClient)")
}

func TestShouldGenerateQueryObjectGivenComplexOpenAPIDocumentWhenGeneratingThenUseQueryParams(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-test.yaml")

	assert.Contains(t, code, "getUsers: (query?: { page?: number; limit?: number; status?: \"active\" | \"inactive\" | \"banned\" }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string })")
	assert.Contains(t, code, "import { buildQueryParams } from '@fgrzl/fetch';")
	assert.Contains(t, code, "const queryString = query ? buildQueryParams(query) : '';")
}

func TestShouldGenerateRequestBodyGivenComplexOpenAPIDocumentWhenGeneratingThenIncludeBodyArgument(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-test.yaml")

	assert.Contains(t, code, "createUser: (body: CreateUserRequest, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<User>>")
}

func TestShouldGeneratePathParametersGivenComplexOpenAPIDocumentWhenGeneratingThenIncludePathArguments(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-test.yaml")

	assert.Contains(t, code, "updateUser: (id: string, body: UpdateUserRequest, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<User>>")
	assert.Contains(t, code, "getTeamMember: (org_id: string, team_id: string, member_id: string")
}

func TestShouldGenerateDeleteOperationGivenComplexOpenAPIDocumentWhenGeneratingThenIncludeQueryArguments(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-test.yaml")

	assert.Contains(t, code, "deleteUser: (id: string, query?: { force?: boolean }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string })")
}

func TestShouldGenerateUserSchemaGivenComplexOpenAPIDocumentWhenGeneratingThenEmitNestedProperties(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-test.yaml")

	assert.Contains(t, code, `status: "active" | "inactive" | "banned" | "pending";`)
	assert.Contains(t, code, `tags?: Array<string>;`)
	assert.Contains(t, code, `metadata?: Record<string, string>;`)
	assert.Contains(t, code, `profile?: {`)
	assert.Contains(t, code, `preferences?: {`)
	assert.Contains(t, code, `theme?: "light" | "dark" | "auto"`)
	assert.Contains(t, code, `notifications?: boolean`)
}

func TestShouldGenerateBooleanResponsesGivenAuthApiWhenGeneratingThenReturnBoolean(t *testing.T) {
	code := generateCodeFromFixture(t, "auth-api.yaml")

	assert.Contains(t, code, "ssoCallback: (provider: string, query?: { code?: string; state?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>")
	assert.Contains(t, code, "ssoLogin: (provider: string, query?: { email?: string; return_url?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>")
	assert.Contains(t, code, "verifyEmail: (query?: { email?: string; token?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>")
	assert.Contains(t, code, "logout: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>")
}

func TestShouldGenerateTypedResponsesGivenAuthApiWhenGeneratingThenReturnResponseTypes(t *testing.T) {
	code := generateCodeFromFixture(t, "auth-api.yaml")

	assert.Contains(t, code, "getJWKS: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<JWKSResponse>>")
	assert.Contains(t, code, "detectSSOProviders: (query?: { email?: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<Array<string>>>")
	assert.Contains(t, code, "getCurrentUser: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<UserIdentity>>")
	assert.Contains(t, code, "getVerificationStatus: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<EmailVerificationStatus>>")
	assert.Contains(t, code, "resendVerification: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<EmailVerificationStatus>>")
	assert.Contains(t, code, "export interface EmailVerificationStatus")
	assert.Contains(t, code, "export interface JWKSResponse")
	assert.Contains(t, code, "export interface ProblemDetails")
	assert.Contains(t, code, "export interface UserIdentity")
}

func TestShouldGenerateAnyOfSchemaGivenUnionSchemaWhenGeneratingThenEmitTypeAlias(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-anyof.yaml")

	assert.Contains(t, code, "export type Pet = Cat | Dog;")
	assert.NotContains(t, code, "export interface Pet")
	assert.Contains(t, code, "export interface Cat")
	assert.Contains(t, code, "export interface Dog")
}

func TestShouldGenerateNullableAndNonStringEnumsGivenNullableEnumSchemaWhenGeneratingThenIncludeNull(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-nullable-enum.yaml")

	assert.Contains(t, code, "export type MaybeString = string | null;")
	assert.Contains(t, code, "status: \"active\" | \"inactive\" | null;")
	assert.Contains(t, code, "count?: 0 | 1 | 2;")
	assert.Contains(t, code, "flag?: true | false;")
}

func TestShouldGenerateSchemaEdgeCasesGivenMixedSchemaSurfacesWhenGeneratingThenPreserveDeclaredShape(t *testing.T) {
	code := generateCodeFromFixture(t, "openapi-schema-edge-cases.yaml")

	assert.Contains(t, code, `config: { baseUrl: string; headers?: Record<string, string>; timeout?: number };`)
	assert.Contains(t, code, `export type ObjectWithPropertiesAndAdditionalProperties = { kind: string; label?: string } & Record<string, boolean>;`)
	assert.Contains(t, code, `export type AllOfWithSiblingProperties = { id: string } & { enabled?: boolean } & { source: string };`)
}

func TestShouldGenerateClosedObjectGivenFalseAdditionalPropertiesWhenGeneratingThenUseNeverRecord(t *testing.T) {
	closed := false
	code := generateCodeFromAPI(t, &apitypes.OpenAPI{
		Components: apitypes.Components{
			Schemas: map[string]*apitypes.Schema{
				"ClosedMap": {
					Type:                 apitypes.SchemaType{Values: []string{"object"}},
					AdditionalProperties: &apitypes.AdditionalProperties{Boolean: &closed},
				},
			},
		},
	}, "")

	assert.Contains(t, code, "export type ClosedMap = Record<string, never>;")
}

func TestShouldUseMultipartRequestBodyGivenNonJsonContentWhenGeneratingThenUseTypedBody(t *testing.T) {
	code := generateCodeFromAPI(t, &apitypes.OpenAPI{
		Paths: map[string]map[string]*apitypes.Operation{
			"/upload": {
				"post": {
					OperationID: "uploadFile",
					RequestBody: &apitypes.RequestBodyWrapper{
						Content: map[string]*apitypes.MediaType{
							"multipart/form-data": {
								Schema: &apitypes.Schema{
									Type: apitypes.SchemaType{Values: []string{"object"}},
									Properties: map[string]*apitypes.Schema{
										"fileName": {Type: apitypes.SchemaType{Values: []string{"string"}}},
									},
									Required: []string{"fileName"},
								},
							},
						},
					},
					Responses: map[string]*apitypes.Response{
						"204": {Description: "No Content"},
					},
				},
			},
		},
	}, "")

	assert.Contains(t, code, "uploadFile: (body: { fileName: string }, options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<boolean>>")
}

func TestShouldUseDefaultResponseGivenDefaultTextResponseWhenGeneratingThenUseTypedSchema(t *testing.T) {
	code := generateCodeFromAPI(t, &apitypes.OpenAPI{
		Paths: map[string]map[string]*apitypes.Operation{
			"/status": {
				"get": {
					OperationID: "getStatus",
					Responses: map[string]*apitypes.Response{
						"default": {
							Content: map[string]apitypes.MediaType{
								"text/plain": {
									Schema: &apitypes.Schema{Type: apitypes.SchemaType{Values: []string{"string"}}},
								},
							},
						},
					},
				},
			},
		},
	}, "")

	assert.Contains(t, code, "getStatus: (options?: { signal?: AbortSignal; timeout?: number; operationId?: string }): Promise<FetchResponse<string>>")
}
