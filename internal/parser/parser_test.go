package parser_test

import (
	"strings"
	"testing"

	"github.com/fgrzl/fetch-gen/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validYAMLDocument = doc(
	"paths:",
	"  /users:",
	"    get:",
	"      operationId: getUsers",
	"      responses:",
	"        \"200\":",
	"          description: ok",
)

var validJSONDocument = doc(
	"{",
	"  \"paths\": {",
	"    \"/users\": {",
	"      \"get\": {",
	"        \"operationId\": \"getUsers\",",
	"        \"responses\": {",
	"          \"200\": {",
	"            \"description\": \"ok\"",
	"          }",
	"        }",
	"      }",
	"    }",
	"  }",
	"}",
)

func doc(lines ...string) string {
	return strings.Join(lines, "\n")
}

func TestShouldParseYAMLGivenValidDocumentWhenParsingThenReturnOpenAPI(t *testing.T) {
	api, err := parser.ParseDocument("openapi.yaml", []byte(validYAMLDocument))
	require.NoError(t, err)
	assert.Equal(t, "getUsers", api.Paths["/users"]["get"].OperationID)
}

func TestShouldParseJSONGivenValidDocumentWhenParsingThenReturnOpenAPI(t *testing.T) {
	api, err := parser.ParseDocument("openapi.json", []byte(validJSONDocument))
	require.NoError(t, err)
	assert.Equal(t, "getUsers", api.Paths["/users"]["get"].OperationID)
}

func TestShouldRejectUnsupportedFileTypeGivenTextInputWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.txt", []byte(validYAMLDocument))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported file type")
}

func TestShouldRejectMissingOperationIDGivenOperationWithoutOperationIdWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users:",
		"    get:",
		"      responses:",
		"        \"200\":",
		"          description: ok",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing operationId")
}

func TestShouldRejectMissingPathParameterGivenPathTemplateWithoutParameterWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users/{id}:",
		"    get:",
		"      operationId: getUser",
		"      responses:",
		"        \"200\":",
		"          description: ok",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing path parameter")
}

func TestShouldRejectDuplicateOperationIDGivenRepeatedOperationIdsWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users:",
		"    get:",
		"      operationId: getUsers",
		"      responses:",
		"        \"200\":",
		"          description: ok",
		"  /admins:",
		"    get:",
		"      operationId: getUsers",
		"      responses:",
		"        \"200\":",
		"          description: ok",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate operationId")
}

func TestShouldRejectUnresolvedRefGivenMissingComponentSchemaWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users:",
		"    get:",
		"      operationId: getUsers",
		"      responses:",
		"        \"200\":",
		"          description: ok",
		"          content:",
		"            application/json:",
		"              schema:",
		"                $ref: '#/components/schemas/Missing'",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unresolved ref")
}

func TestShouldRejectMalformedPathTemplateGivenMissingClosingBraceWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users/{id:",
		"    get:",
		"      operationId: getUsers",
		"      responses:",
		"        \"200\":",
		"          description: ok",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "malformed path template")
}

func TestShouldRejectUnsupportedParameterLocationGivenHeaderParameterWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users:",
		"    get:",
		"      operationId: getUsers",
		"      parameters:",
		"        - name: x-trace-id",
		"          in: header",
		"          schema:",
		"            type: string",
		"      responses:",
		"        \"200\":",
		"          description: ok",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported parameter location")
}

func TestShouldRejectMissingResponsesGivenOperationWithoutResponsesWhenParsingThenReturnError(t *testing.T) {
	_, err := parser.ParseDocument("openapi.yaml", []byte(doc(
		"paths:",
		"  /users:",
		"    get:",
		"      operationId: getUsers",
	)))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing responses")
}
