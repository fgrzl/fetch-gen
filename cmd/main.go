// main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// AdditionalProperties can be either boolean or Schema
type AdditionalProperties struct {
	Boolean *bool
	Schema  *Schema
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
	Type                 string                `json:"type" yaml:"type"`
	Properties           map[string]*Schema    `json:"properties" yaml:"properties"`
	Items                *Schema               `json:"items" yaml:"items"`
	Enum                 []any                 `json:"enum" yaml:"enum"`
	Ref                  string                `json:"$ref" yaml:"$ref"`
	Description          string                `json:"description" yaml:"description"`
	Required             []string              `json:"required" yaml:"required"`
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

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	funcs := template.FuncMap{
		"tsType": resolveType,
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
			return strings.Join(args, ", ")
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
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
	}

	tmpl := template.Must(template.New("api").Funcs(funcs).Parse(apiTemplate))
	err = tmpl.Execute(f, map[string]any{
		"Schemas":  api.Components.Schemas,
		"Ops":      ops,
		"Instance": instance,
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
		types := make([]string, len(s.Enum))
		for i, val := range s.Enum {
			types[i] = fmt.Sprintf(`"%v"`, val)
		}
		return strings.Join(types, " | ")
	}
	switch s.Type {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "Array<" + resolveType(s.Items) + ">"
	case "object":
		if s.AdditionalProperties != nil {
			if s.AdditionalProperties.Boolean != nil {
				if *s.AdditionalProperties.Boolean {
					return "Record<string, any>"
				} else {
					// additionalProperties: false - no additional properties allowed
					if len(s.Properties) == 0 {
						return "Record<string, never>"
					}
				}
			} else if s.AdditionalProperties.Schema != nil {
				return "Record<string, " + resolveType(s.AdditionalProperties.Schema) + ">"
			}
		}
		if len(s.Properties) == 0 {
			return "Record<string, any>"
		}
		props := []string{}
		for name, prop := range s.Properties {
			props = append(props, fmt.Sprintf("%s: %s", name, resolveType(prop)))
		}
		return "{ " + strings.Join(props, "; ") + " }"
	}
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
	return "any"
}

func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

const apiTemplate = `// Auto-generated by fetch-gen
import type { FetchClient, FetchResponse } from '{{.Instance}}';
import { buildQueryParams } from '{{.Instance}}';

/**
 * Creates an API adapter with typed methods for all OpenAPI operations
 * @param client The FetchClient instance to use for HTTP requests
 * @returns An object with typed methods for each API operation
 */
export function createAdapter(client: FetchClient): {
	{{- range $i, $op := .Ops}}
	/**
	 * {{if $op.Description}}{{$op.Description}}{{else}}{{$op.Method | upper}} {{$op.DisplayPath}}{{end}}
	{{- range $param := $op.PathParams}}
	 * @param {{$param.Name}} {{if $param.Description}}{{$param.Description}}{{else}}{{$param.Name}} parameter{{end}}
	{{- end}}
	{{- if hasQueryParams $op}}
	 * @param query Query parameters
	{{- end}}
	{{- if $op.HasBody}}
	 * @param body Request body
	{{- end}}
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
			return client.{{$op.Method}}(url{{if $op.HasBody}}, body{{end}});
			{{- else}}
			return client.{{$op.Method}}(` + "`" + `{{$op.DisplayPath}}` + "`" + `{{if $op.HasBody}}, body{{end}});
			{{- end}}
		},
		{{- end}}
	};
}

{{range $name, $schema := .Schemas}}
{{if $schema.Description}}/** {{$schema.Description}} */{{else}}/** {{$name}} schema */{{end}}
export interface {{$name}} {
{{- range $prop, $def := $schema.Properties }}
	{{- if $def.Description}}
	/** {{ $def.Description }} */
	{{- end}}
	{{- $isRequired := contains $schema.Required $prop}}
	{{$prop}}{{if not $isRequired}}?{{end}}: {{ tsType $def }};
{{- end }}
}

{{end}}
`
