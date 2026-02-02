// main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// AdditionalProperties can be either boolean or Schema
type AdditionalProperties struct {
	Boolean *bool
	Schema  *Schema
}

// SchemaType supports OpenAPI 3.1 / JSON Schema where `type` can be a string or an array of strings.
// Examples: "string" or ["string","null"].
type SchemaType struct {
	Values []string
}

func (st *SchemaType) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		st.Values = nil
		return nil
	}
	var single string
	if err := node.Decode(&single); err == nil {
		if single == "" {
			st.Values = nil
			return nil
		}
		st.Values = []string{single}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err == nil {
		st.Values = list
		return nil
	}
	// Unknown shape; treat as unset
	st.Values = nil
	return nil
}

func (st *SchemaType) UnmarshalJSON(data []byte) error {
	// null
	if string(data) == "null" {
		st.Values = nil
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single == "" {
			st.Values = nil
			return nil
		}
		st.Values = []string{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		st.Values = list
		return nil
	}
	st.Values = nil
	return nil
}

func (st SchemaType) has(t string) bool {
	for _, v := range st.Values {
		if v == t {
			return true
		}
	}
	return false
}

func (st SchemaType) isEmpty() bool {
	return len(st.Values) == 0
}

func (ap *AdditionalProperties) UnmarshalYAML(node *yaml.Node) error {
	var b bool
	if err := node.Decode(&b); err == nil {
		ap.Boolean = &b
		return nil
	}

	var s Schema
	if err := node.Decode(&s); err != nil {
		return err
	}
	ap.Schema = &s
	return nil
}

func (ap *AdditionalProperties) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		ap.Boolean = &b
		return nil
	}

	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ap.Schema = &s
	return nil
}

type OpenAPI struct {
	Paths      map[string]map[string]*Operation `json:"paths" yaml:"paths"`
	Components Components                       `json:"components" yaml:"components"`
}

type Components struct {
	Schemas map[string]*Schema `json:"schemas" yaml:"schemas"`
}

type Operation struct {
	OperationID string               `json:"operationId" yaml:"operationId"`
	Summary     string               `json:"summary" yaml:"summary"`
	Description string               `json:"description" yaml:"description"`
	RequestBody *RequestBodyWrapper  `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]*Response `json:"responses" yaml:"responses"`
	Parameters  []*Parameter         `json:"parameters" yaml:"parameters"`
}

type RequestBodyWrapper struct {
	Content map[string]*MediaType `json:"content" yaml:"content"`
}

type MediaType struct {
	Schema *Schema `json:"schema" yaml:"schema"`
}

type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

type Schema struct {
	Type                 SchemaType            `json:"type" yaml:"type"`
	Properties           map[string]*Schema    `json:"properties" yaml:"properties"`
	Items                *Schema               `json:"items" yaml:"items"`
	Enum                 []any                 `json:"enum" yaml:"enum"`
	Ref                  string                `json:"$ref" yaml:"$ref"`
	Description          string                `json:"description" yaml:"description"`
	Required             []string              `json:"required" yaml:"required"`
	Nullable             *bool                 `json:"nullable" yaml:"nullable"`
	AllOf                []*Schema             `json:"allOf" yaml:"allOf"`
	OneOf                []*Schema             `json:"oneOf" yaml:"oneOf"`
	AnyOf                []*Schema             `json:"anyOf" yaml:"anyOf"`
	AdditionalProperties *AdditionalProperties `json:"additionalProperties" yaml:"additionalProperties"`
}

type Parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Required    bool    `json:"required" yaml:"required"`
	Schema      *Schema `json:"schema" yaml:"schema"`
	Description string  `json:"description" yaml:"description"`
}

type NamedOperation struct {
	ID           string
	Method       string
	DisplayPath  string
	PathParams   []Parameter
	QueryParams  []Parameter
	HasBody      bool
	RequestType  string
	ResponseType string
	Description  string
}

// TemplateSchema holds schema plus ordered property keys for deterministic iteration
type TemplateSchema struct {
	Name     string
	Schema   *Schema
	PropKeys []string
}

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
	err = os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	instance := "@fgrzl/fetch"
	if len(os.Args) >= 7 && os.Args[5] == "--instance" {
		instance = strings.TrimSuffix(os.Args[6], ".ts")
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var api OpenAPI
	switch ext := strings.ToLower(filepath.Ext(inputPath)); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &api); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &api); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file type (must be .yaml or .json)")
	}

	var ops []NamedOperation
	for path, methods := range api.Paths {
		for method, op := range methods {
			id := op.OperationID
			if id == "" {
				fmt.Printf("missing operation id for %s %s\n", strings.ToUpper(method), path)
				continue
			}
			params := []Parameter{}
			queryParams := []Parameter{}
			displayPath := path
			for _, p := range op.Parameters {
				if p.In == "path" {
					params = append(params, *p)
					displayPath = strings.ReplaceAll(displayPath, "{"+p.Name+"}", fmt.Sprintf("${%s}", p.Name))
				} else if p.In == "query" {
					queryParams = append(queryParams, *p)
				}
			}
			// Combine path and query params for function signature

			// Only application/json
			reqType := ""
			if op.RequestBody != nil {
				if jsonContent, ok := op.RequestBody.Content["application/json"]; ok {
					reqType = resolveType(jsonContent.Schema)
				}
			}

			resType := "any"
			// Check for successful responses in order of preference
			for _, code := range []string{"200", "201", "202", "203", "204", "206", "default"} {
				if resp, ok := op.Responses[code]; ok {
					// Handle 204 No Content specially
					if code == "204" {
						resType = "boolean"
						break
					}
					if jsonContent, ok := resp.Content["application/json"]; ok {
						if jsonContent.Schema != nil {
							resType = resolveType(jsonContent.Schema)
							break
						}
					}
				}
			}

			// If no success response found, check for redirect responses (3xx)
			if resType == "any" {
				for _, code := range []string{"300", "301", "302", "303", "304", "307", "308"} {
					if resp, ok := op.Responses[code]; ok {
						// Redirects typically don't have a JSON body, just headers
						// Check if there's content defined
						if len(resp.Content) == 0 {
							resType = "boolean"
							break
						}
						// If there is content, try to resolve it
						if jsonContent, ok := resp.Content["application/json"]; ok {
							if jsonContent.Schema != nil {
								resType = resolveType(jsonContent.Schema)
								break
							}
						} else {
							// Non-JSON redirect response
							resType = "boolean"
							break
						}
					}
				}
			}

			if method == "delete" {
				method = "del"
			}

			// Get operation description (prefer summary, fallback to description)
			description := op.Summary
			if description == "" {
				description = op.Description
			}

			ops = append(ops, NamedOperation{
				ID:           id,
				Method:       method,
				DisplayPath:  displayPath,
				PathParams:   params,
				QueryParams:  queryParams,
				HasBody:      op.RequestBody != nil,
				RequestType:  reqType,
				ResponseType: resType,
				Description:  description,
			})
		}
	}

	// Ensure deterministic ordering: first by path (DisplayPath), then by operation ID
	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].DisplayPath == ops[j].DisplayPath {
			return ops[i].ID < ops[j].ID
		}
		return ops[i].DisplayPath < ops[j].DisplayPath
	})

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	funcs := template.FuncMap{
		"tsType": resolveType,
		"isAlias": func(s *Schema) bool {
			if s == nil {
				return true
			}
			// If this schema is a ref or a composed/enum/primitive, emit a type alias.
			if s.Ref != "" {
				return true
			}
			if len(s.Enum) > 0 {
				return true
			}
			if len(s.AllOf) > 0 || len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
				return true
			}
			// Objects with additionalProperties are better represented as a Record<> type alias.
			if s.Type.has("object") {
				if s.AdditionalProperties != nil {
					return true
				}
				// If there are no explicit properties, avoid generating an empty interface.
				return len(s.Properties) == 0
			}
			// Arrays/primitives: emit a type alias.
			return !s.Type.isEmpty()
		},
		"argList": func(op NamedOperation) string {
			args := []string{}
			// Add path parameters
			for _, p := range op.PathParams {
				paramType := resolveType(p.Schema)
				if !p.Required {
					paramType += " | undefined"
				}
				args = append(args, fmt.Sprintf("%s: %s", p.Name, paramType))
			}
			// Add query parameters as a single optional object
			if len(op.QueryParams) > 0 {
				queryProps := []string{}
				for _, p := range op.QueryParams {
					paramType := resolveType(p.Schema)
					optional := "?"
					if p.Required {
						optional = ""
					}
					queryProps = append(queryProps, fmt.Sprintf("%s%s: %s", p.Name, optional, paramType))
				}
				queryType := fmt.Sprintf("{ %s }", strings.Join(queryProps, "; "))
				args = append(args, fmt.Sprintf("query?: %s", queryType))
			}
			if op.HasBody {
				bodyType := "any"
				if op.RequestType != "" {
					bodyType = op.RequestType
				}
				args = append(args, fmt.Sprintf("body: %s", bodyType))
			}
			// Always allow passing AbortSignal/timeout/operationId through to the underlying client
			args = append(args, "options?: { signal?: AbortSignal; timeout?: number; operationId?: string }")
			return strings.Join(args, ", ")
		},
		"clientCall": func(op NamedOperation, urlExpr string) string {
			// @fgrzl/fetch method signatures:
			// - get/del/head: (url, params?, options?)
			// - post/put/patch: (url, body?, headers?, options?)
			switch op.Method {
			case "get", "del", "head":
				return fmt.Sprintf("return client.%s(%s, undefined, options);", op.Method, urlExpr)
			case "post", "put", "patch":
				bodyArg := "undefined"
				if op.HasBody {
					bodyArg = "body"
				}
				return fmt.Sprintf("return client.%s(%s, %s, undefined, options);", op.Method, urlExpr, bodyArg)
			default:
				// Fallback: pass URL, optional body, then options
				bodyArg := "undefined"
				if op.HasBody {
					bodyArg = "body"
				}
				return fmt.Sprintf("return client.%s(%s, %s, options);", op.Method, urlExpr, bodyArg)
			}
		},
		"responseType": func(op NamedOperation) string {
			if op.ResponseType != "" {
				return op.ResponseType
			}
			return "any"
		},
		"hasQueryParams": func(op NamedOperation) bool {
			return len(op.QueryParams) > 0
		},
		"upper": strings.ToUpper,
		"add": func(a, b int) int {
			return a + b
		},
		"len": func(slice []NamedOperation) int {
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
	}

	// Build sorted schemas with deterministic property ordering
	sortedSchemaNames := []string{}
	for name := range api.Components.Schemas {
		sortedSchemaNames = append(sortedSchemaNames, name)
	}
	sort.Strings(sortedSchemaNames)

	sortedSchemas := []TemplateSchema{}
	for _, name := range sortedSchemaNames {
		s := api.Components.Schemas[name]
		propKeys := []string{}
		if s != nil && s.Properties != nil {
			for pk := range s.Properties {
				propKeys = append(propKeys, pk)
			}
			sort.Strings(propKeys)
		}
		sortedSchemas = append(sortedSchemas, TemplateSchema{Name: name, Schema: s, PropKeys: propKeys})
	}

	tmpl := template.Must(template.New("api").Funcs(funcs).Parse(apiTemplate))
	err = tmpl.Execute(f, map[string]any{
		"SortedSchemas": sortedSchemas,
		"Ops":           ops,
		"Instance":      instance,
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("✅ Generated fetch client: %s\n", outputPath)
	return nil
}

func resolveType(s *Schema) string {
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
				// Best-effort fallback; keep it a string literal to avoid invalid TS.
				types = append(types, fmt.Sprintf(`"%v"`, v))
			}
		}
		return strings.Join(types, " | ")
	}
	// Composition keywords can appear with or without an explicit `type`.
	// Prefer them over `type` to avoid silently ignoring anyOf/oneOf/allOf.
	if len(s.AllOf) > 0 {
		types := []string{}
		for _, sub := range s.AllOf {
			types = append(types, resolveType(sub))
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
	// Determine base type(s) from `type`.
	// OpenAPI 3.1 allows `type` to be a list including "null".
	typeVals := s.Type.Values
	if len(typeVals) == 0 {
		// Heuristic: object schemas commonly omit `type`.
		if len(s.Properties) > 0 || s.AdditionalProperties != nil {
			typeVals = []string{"object"}
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
			if s.AdditionalProperties != nil {
				if s.AdditionalProperties.Boolean != nil {
					if *s.AdditionalProperties.Boolean {
						return "Record<string, any>"
					}
					// additionalProperties: false - no additional properties allowed
					if len(s.Properties) == 0 {
						return "Record<string, never>"
					}
				} else if s.AdditionalProperties.Schema != nil {
					return "Record<string, " + resolveType(s.AdditionalProperties.Schema) + ">"
				}
			}
			if len(s.Properties) == 0 {
				return "Record<string, any>"
			}
			props := []string{}
			// iterate properties in sorted order for deterministic output
			propKeys := []string{}
			for name := range s.Properties {
				propKeys = append(propKeys, name)
			}
			sort.Strings(propKeys)
			for _, name := range propKeys {
				prop := s.Properties[name]
				props = append(props, fmt.Sprintf("%s: %s", name, resolveType(prop)))
			}
			return "{ " + strings.Join(props, "; ") + " }"
		default:
			return "any"
		}
	}

	if len(typeVals) > 0 {
		parts := make([]string, 0, len(typeVals)+1)
		for _, tv := range typeVals {
			parts = append(parts, mapOne(tv))
		}
		// Back-compat: support OpenAPI 3.0 `nullable: true` as `T | null`.
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

func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
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
  {{$op.ID}}: ({{argList $op}}) => Promise<FetchResponse<{{responseType $op}}>>;
{{- end}}
} {
  return {
{{- range $i, $op := .Ops}}
    {{$op.ID}}: ({{argList $op}}): Promise<FetchResponse<{{responseType $op}}>> => {
{{- if hasQueryParams $op}}
      const queryString = query ? buildQueryParams(query) : '';
      const url = ` + "`" + `{{$op.DisplayPath}}` + "`" + ` + (queryString ? '?' + queryString : '');
			{{clientCall $op "url"}}
{{- else}}
	{{clientCall $op (printf "%c%s%c" 96 $op.DisplayPath 96)}}
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
{{- else}}
/** {{$name}} schema */
{{- end}}
{{- if isAlias $schema }}
export type {{$name}} = {{ tsType $schema }};
{{- else}}
export interface {{$name}} {
{{- range $idx, $prop := $s.PropKeys }}
  {{- $def := index $schema.Properties $prop }}
  {{- if $def.Description }}
  /** {{ $def.Description }} */
  {{- end }}
  {{- $isRequired := contains $schema.Required $prop }}
  {{$prop}}{{if not $isRequired}}?{{end}}: {{ tsType $def }};
{{- end }}
}
{{- end}}
{{end}}
`
