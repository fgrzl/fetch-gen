package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	apitypes "github.com/fgrzl/fetch-gen/internal/types"
)

type namedOperation struct {
	ID           string
	Method       string
	DisplayPath  string
	PathParams   []apitypes.Parameter
	QueryParams  []apitypes.Parameter
	HasBody      bool
	BodyRequired bool
	RequestType  string
	ResponseType string
	Description  string
}

type templateSchema struct {
	Name     string
	Schema   *apitypes.Schema
	PropKeys []string
}

func Generate(api *apitypes.OpenAPI, instance string) ([]byte, error) {
	if api == nil {
		return nil, fmt.Errorf("openapi document is empty")
	}
	if strings.TrimSpace(instance) == "" {
		instance = "@fgrzl/fetch"
	}

	var ops []namedOperation
	for path, methods := range api.Paths {
		for method, op := range methods {
			if op == nil {
				return nil, fmt.Errorf("operation is nil")
			}

			params := []apitypes.Parameter{}
			queryParams := []apitypes.Parameter{}
			displayPath := path
			for _, p := range op.Parameters {
				switch p.In {
				case "path":
					params = append(params, *p)
					displayPath = strings.ReplaceAll(displayPath, "{"+p.Name+"}", fmt.Sprintf("${%s}", p.Name))
				case "query":
					queryParams = append(queryParams, *p)
				}
			}

			reqType := requestTypeForOperation(op)
			resType := responseTypeForOperation(op)

			if method == "delete" {
				method = "del"
			}

			description := op.Summary
			if description == "" {
				description = op.Description
			}

			ops = append(ops, namedOperation{
				ID:           op.OperationID,
				Method:       method,
				DisplayPath:  displayPath,
				PathParams:   params,
				QueryParams:  queryParams,
				HasBody:      op.RequestBody != nil,
				BodyRequired: op.RequestBody != nil && op.RequestBody.Required,
				RequestType:  reqType,
				ResponseType: resType,
				Description:  description,
			})
		}
	}

	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].DisplayPath == ops[j].DisplayPath {
			return ops[i].ID < ops[j].ID
		}
		return ops[i].DisplayPath < ops[j].DisplayPath
	})

	funcs := template.FuncMap{
		"tsType": resolveType,
		"isAlias": func(s *apitypes.Schema) bool {
			if s == nil {
				return true
			}
			if s.Ref != "" {
				return true
			}
			if len(s.Enum) > 0 {
				return true
			}
			if len(s.AllOf) > 0 || len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
				return true
			}
			if s.Type.Has("object") {
				if s.AdditionalProperties != nil {
					return true
				}
				return len(s.Properties) == 0
			}
			return !s.Type.IsEmpty()
		},
		"argList": func(op namedOperation) string {
			args := []string{}
			for _, p := range op.PathParams {
				paramType := resolveType(p.Schema)
				if !p.Required {
					paramType += " | undefined"
				}
				args = append(args, fmt.Sprintf("%s: %s", p.Name, paramType))
			}
			if len(op.QueryParams) > 0 {
				queryProps := []string{}
				for _, p := range op.QueryParams {
					paramType := resolveType(p.Schema)
					optional := "?"
					if p.Required {
						optional = ""
					}
					queryProps = append(queryProps, fmt.Sprintf("%s%s: %s", tsPropertyKey(p.Name), optional, paramType))
				}
				queryType := fmt.Sprintf("{ %s }", strings.Join(queryProps, "; "))
				if hasRequiredQueryParams(op) {
					args = append(args, fmt.Sprintf("query: %s", queryType))
				} else {
					args = append(args, fmt.Sprintf("query?: %s", queryType))
				}
			}
			if op.HasBody {
				bodyType := "any"
				if op.RequestType != "" {
					bodyType = op.RequestType
				}
				if op.BodyRequired {
					args = append(args, fmt.Sprintf("body: %s", bodyType))
				} else {
					args = append(args, fmt.Sprintf("body?: %s", bodyType))
				}
			}
			args = append(args, "options?: { signal?: AbortSignal; timeout?: number; operationId?: string }")
			return strings.Join(args, ", ")
		},
		"clientCall": func(op namedOperation, urlExpr string, optionsVar string) string {
			switch op.Method {
			case "get", "del", "head":
				return fmt.Sprintf("return client.%s(%s, undefined, %s);", op.Method, urlExpr, optionsVar)
			case "post", "put", "patch":
				bodyArg := "undefined"
				if op.HasBody {
					bodyArg = "body"
				}
				return fmt.Sprintf("return client.%s(%s, %s, undefined, %s);", op.Method, urlExpr, bodyArg, optionsVar)
			default:
				bodyArg := "undefined"
				if op.HasBody {
					bodyArg = "body"
				}
				return fmt.Sprintf("return client.%s(%s, %s, %s);", op.Method, urlExpr, bodyArg, optionsVar)
			}
		},
		"responseType": func(op namedOperation) string {
			if op.ResponseType != "" {
				return op.ResponseType
			}
			return "any"
		},
		"hasQueryParams": func(op namedOperation) bool {
			return len(op.QueryParams) > 0
		},
		"upper": strings.ToUpper,
		"add": func(a, b int) int {
			return a + b
		},
		"len": func(slice []namedOperation) int {
			return len(slice)
		},
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"tsPropertyKey":   tsPropertyKey,
		"tsStringLiteral": tsStringLiteral,
	}

	sortedSchemaNames := []string{}
	for name := range api.Components.Schemas {
		sortedSchemaNames = append(sortedSchemaNames, name)
	}
	sort.Strings(sortedSchemaNames)

	sortedSchemas := []templateSchema{}
	for _, name := range sortedSchemaNames {
		s := api.Components.Schemas[name]
		propKeys := []string{}
		if s != nil && s.Properties != nil {
			for pk := range s.Properties {
				propKeys = append(propKeys, pk)
			}
			sort.Strings(propKeys)
		}
		sortedSchemas = append(sortedSchemas, templateSchema{Name: name, Schema: s, PropKeys: propKeys})
	}

	tmpl := template.Must(template.New("api").Funcs(funcs).Parse(apiTemplate))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, map[string]any{
		"SortedSchemas": sortedSchemas,
		"Ops":           ops,
		"Instance":      instance,
	}); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return out.Bytes(), nil
}

func resolveType(s *apitypes.Schema) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		return extractRefName(s.Ref)
	}
	if len(s.Enum) > 0 {
		types := make([]string, 0, len(s.Enum))
		for _, val := range s.Enum {
			switch v := val.(type) {
			case nil:
				types = append(types, "null")
			case string:
				types = append(types, fmt.Sprintf(`"%s"`, v))
			case bool:
				if v {
					types = append(types, "true")
				} else {
					types = append(types, "false")
				}
			case int:
				types = append(types, fmt.Sprintf("%d", v))
			case int64:
				types = append(types, fmt.Sprintf("%d", v))
			case float64:
				types = append(types, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", v), "0"), "."))
			default:
				types = append(types, fmt.Sprintf(`"%v"`, v))
			}
		}
		return strings.Join(types, " | ")
	}
	if len(s.AllOf) > 0 {
		types := []string{}
		for _, sub := range s.AllOf {
			types = append(types, resolveType(sub))
		}
		ownSchema := *s
		ownSchema.AllOf = nil
		if hasOwnSchemaSurface(&ownSchema) {
			types = append(types, resolveType(&ownSchema))
		}
		return strings.Join(types, " & ")
	}
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		union := s.OneOf
		if len(union) == 0 {
			union = s.AnyOf
		}
		types := []string{}
		for _, sub := range union {
			types = append(types, resolveType(sub))
		}
		return strings.Join(types, " | ")
	}
	typeVals := s.Type.Values
	if len(typeVals) == 0 {
		if len(s.Properties) > 0 || s.AdditionalProperties != nil {
			typeVals = []string{"object"}
		} else if s.Items != nil {
			typeVals = []string{"array"}
		}
	}

	mapOne := func(t string) string {
		switch t {
		case "string":
			return "string"
		case "integer", "number":
			return "number"
		case "boolean":
			return "boolean"
		case "null":
			return "null"
		case "array":
			return "Array<" + resolveType(s.Items) + ">"
		case "object":
			return resolveObjectType(s)
		default:
			return "any"
		}
	}

	if len(typeVals) > 0 {
		parts := make([]string, 0, len(typeVals)+1)
		for _, tv := range typeVals {
			parts = append(parts, mapOne(tv))
		}
		if s.Nullable != nil && *s.Nullable {
			if !containsString(parts, "null") {
				parts = append(parts, "null")
			}
		}
		return strings.Join(parts, " | ")
	}

	return "any"
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isTSIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for index, r := range name {
		if index == 0 {
			if r != '_' && r != '$' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && r != '$' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func tsPropertyKey(name string) string {
	if isTSIdentifier(name) {
		return name
	}
	return fmt.Sprintf("%q", name)
}

func tsStringLiteral(value string) string {
	return fmt.Sprintf("%q", value)
}

func hasRequiredQueryParams(op namedOperation) bool {
	for _, p := range op.QueryParams {
		if p.Required {
			return true
		}
	}
	return false
}

func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func requestTypeForOperation(op *apitypes.Operation) string {
	if op == nil || op.RequestBody == nil {
		return ""
	}
	schema := requestContentSchema(op.RequestBody.Content)
	if schema == nil {
		return ""
	}
	return resolveType(schema)
}

func responseTypeForOperation(op *apitypes.Operation) string {
	if op == nil {
		return "any"
	}

	for _, code := range []string{"200", "201", "202", "203", "204", "206", "default"} {
		if resp, ok := op.Responses[code]; ok && resp != nil {
			if code == "204" {
				return "boolean"
			}
			schema := responseContentSchema(resp.Content)
			if schema != nil {
				return resolveType(schema)
			}
		}
	}

	for _, code := range []string{"300", "301", "302", "303", "304", "307", "308"} {
		if resp, ok := op.Responses[code]; ok && resp != nil {
			if len(resp.Content) == 0 {
				return "boolean"
			}
			schema := responseContentSchema(resp.Content)
			if schema != nil {
				return resolveType(schema)
			}
			return "boolean"
		}
	}

	return "any"
}

func requestContentSchema(content map[string]*apitypes.MediaType) *apitypes.Schema {
	if content == nil {
		return nil
	}
	if media, ok := content["application/json"]; ok && media != nil && media.Schema != nil {
		return media.Schema
	}
	keys := make([]string, 0, len(content))
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		media := content[key]
		if media != nil && media.Schema != nil {
			return media.Schema
		}
	}
	return nil
}

func responseContentSchema(content map[string]apitypes.MediaType) *apitypes.Schema {
	if content == nil {
		return nil
	}
	if media, ok := content["application/json"]; ok && media.Schema != nil {
		return media.Schema
	}
	keys := make([]string, 0, len(content))
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		media := content[key]
		if media.Schema != nil {
			return media.Schema
		}
	}
	return nil
}

func hasOwnSchemaSurface(s *apitypes.Schema) bool {
	if s == nil {
		return false
	}
	return s.Ref != "" ||
		len(s.Enum) > 0 ||
		len(s.OneOf) > 0 ||
		len(s.AnyOf) > 0 ||
		!s.Type.IsEmpty() ||
		len(s.Properties) > 0 ||
		s.Items != nil ||
		s.AdditionalProperties != nil
}

func resolveObjectType(s *apitypes.Schema) string {
	propKeys := []string{}
	for name := range s.Properties {
		propKeys = append(propKeys, name)
	}
	sort.Strings(propKeys)

	props := []string{}
	for _, name := range propKeys {
		prop := s.Properties[name]
		optional := "?"
		if containsString(s.Required, name) {
			optional = ""
		}
		props = append(props, fmt.Sprintf("%s%s: %s", tsPropertyKey(name), optional, resolveType(prop)))
	}

	objectLiteral := ""
	if len(props) > 0 {
		objectLiteral = "{ " + strings.Join(props, "; ") + " }"
	}

	additionalType := ""
	if s.AdditionalProperties != nil {
		if s.AdditionalProperties.Boolean != nil {
			if *s.AdditionalProperties.Boolean {
				additionalType = "Record<string, any>"
			} else if objectLiteral == "" {
				additionalType = "Record<string, never>"
			}
		} else if s.AdditionalProperties.Schema != nil {
			additionalType = "Record<string, " + resolveType(s.AdditionalProperties.Schema) + ">"
		}
	}

	if objectLiteral != "" && additionalType != "" {
		return objectLiteral + " & " + additionalType
	}
	if objectLiteral != "" {
		return objectLiteral
	}
	if additionalType != "" {
		return additionalType
	}

	return "Record<string, any>"
}

const apiTemplate = `// Auto-generated by fetch-gen
import type { FetchClient, FetchResponse } from '{{.Instance}}';
import { buildQueryParams } from '{{.Instance}}';

/**
 * Creates an API adapter with typed methods for all OpenAPI operations.
 *
 * @param client - The FetchClient instance to use for HTTP requests
 * @returns An object with typed methods for each API operation
 *
 * @example
 * ` + "```typescript" + `
 * import { createAdapter } from './generated';
 * import client from '@fgrzl/fetch';
 *
 * client.setBaseUrl('https://api.example.com');
 * const api = createAdapter(client);
 *
 * const response = await api.getUsers();
 * if (response.ok) {
 *   console.log(response.data);
 * }
 * ` + "```" + `
 */
export function createAdapter(client: FetchClient): {
{{- range $i, $op := .Ops}}
  /**
   * {{if $op.Description}}{{$op.Description}}{{else}}{{$op.Method | upper}} {{$op.DisplayPath}}{{end}}
   *
{{- range $param := $op.PathParams}}
   * @param {{$param.Name}} - {{if $param.Description}}{{$param.Description}}{{else}}{{$param.Name}} parameter{{end}}
{{- end}}
{{- if hasQueryParams $op}}
   * @param query - Query parameters
{{- end}}
{{- if $op.HasBody}}
   * @param body - Request body
{{- end}}
	 * @param options - Request options (signal, timeout, operationId)
   * @returns Promise resolving to FetchResponse<{{responseType $op}}>
   */
	{{tsPropertyKey $op.ID}}: ({{argList $op}}) => Promise<FetchResponse<{{responseType $op}}>>;
{{- end}}
} {
  return {
{{- range $i, $op := .Ops}}
		{{tsPropertyKey $op.ID}}: ({{argList $op}}): Promise<FetchResponse<{{responseType $op}}>> => {
		const finalOptions = { ...options, operationId: options?.operationId ?? {{tsStringLiteral $op.ID}} };
{{- if hasQueryParams $op}}
      const queryString = query ? buildQueryParams(query) : '';
      const url = ` + "`" + `{{$op.DisplayPath}}` + "`" + ` + (queryString ? '?' + queryString : '');
			{{clientCall $op "url" "finalOptions"}}
{{- else}}
	{{clientCall $op (printf "%c%s%c" 96 $op.DisplayPath 96) "finalOptions"}}
{{- end}}
    }{{if ne (add $i 1) (len $.Ops)}},{{end}}
{{- end}}
  };
}
{{range $i, $s := .SortedSchemas }}
{{- $name := $s.Name }}
{{- $schema := $s.Schema }}

{{- if $schema.Description }}
/** {{$schema.Description}} */
{{- else }}
/** {{$name}} schema */
{{- end }}
{{- if isAlias $schema }}
export type {{$name}} = {{ tsType $schema }};
{{- else }}
export interface {{$name}} {
{{- range $idx, $prop := $s.PropKeys }}
  {{- $def := index $schema.Properties $prop }}
  {{- if $def.Description }}
  /** {{ $def.Description }} */
  {{- end }}
  {{- $isRequired := contains $schema.Required $prop }}
	{{tsPropertyKey $prop}}{{if not $isRequired}}?{{end}}: {{ tsType $def }};
{{- end }}
}
{{- end}}
{{end}}
`
