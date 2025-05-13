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

type OpenAPI struct {
	Paths      map[string]map[string]*Operation `json:"paths" yaml:"paths"`
	Components Components                       `json:"components" yaml:"components"`
}

type Components struct {
	Schemas map[string]*Schema `json:"schemas" yaml:"schemas"`
}

type Operation struct {
	OperationID string               `json:"operationId" yaml:"operationId"`
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
	Type                 string             `json:"type" yaml:"type"`
	Properties           map[string]*Schema `json:"properties" yaml:"properties"`
	Items                *Schema            `json:"items" yaml:"items"`
	Enum                 []any              `json:"enum" yaml:"enum"`
	Ref                  string             `json:"$ref" yaml:"$ref"`
	Description          string             `json:"description" yaml:"description"`
	Required             []string           `json:"required" yaml:"required"`
	AllOf                []*Schema          `json:"allOf" yaml:"allOf"`
	OneOf                []*Schema          `json:"oneOf" yaml:"oneOf"`
	AnyOf                []*Schema          `json:"anyOf" yaml:"anyOf"`
	AdditionalProperties *Schema            `json:"additionalProperties" yaml:"additionalProperties"`
}

type Parameter struct {
	Name     string  `json:"name" yaml:"name"`
	In       string  `json:"in" yaml:"in"`
	Required bool    `json:"required" yaml:"required"`
	Schema   *Schema `json:"schema" yaml:"schema"`
}

type NamedOperation struct {
	ID           string
	Method       string
	DisplayPath  string
	Params       []Parameter
	HasBody      bool
	RequestType  string
	ResponseType string
}

func main() {
	if len(os.Args) < 5 || os.Args[1] != "--input" || os.Args[3] != "--output" {
		fmt.Println("Usage: fetch-gen --input openapi.yaml --output ./src/api.ts [--instance ./path/to/client]")
		os.Exit(1)
	}

	inputPath := os.Args[2]
	inputPath, err := filepath.Abs(inputPath)
	if err != nil {
		panic(err)
	}

	outputPath := os.Args[4]
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		panic(err)
	}

	instance := "@fgrzl/fetch"
	if len(os.Args) >= 7 && os.Args[5] == "--instance" {
		instance = strings.TrimSuffix(os.Args[6], ".ts")
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		panic(err)
	}

	var api OpenAPI
	switch ext := strings.ToLower(filepath.Ext(inputPath)); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &api); err != nil {
			panic(err)
		}
	case ".json":
		if err := json.Unmarshal(data, &api); err != nil {
			panic(err)
		}
	default:
		panic("unsupported file type (must be .yaml or .json)")
	}

	var ops []NamedOperation
	for path, methods := range api.Paths {
		for method, op := range methods {
			id := op.OperationID
			if id == "" {
				methodUpper := strings.ToUpper(method)
				pathClean := strings.ReplaceAll(strings.Trim(path, "/"), "/", "_")
				if pathClean == "" {
					pathClean = "root"
				}
				id = fmt.Sprintf("%s_%s", methodUpper, pathClean)
			}

			params := []Parameter{}
			displayPath := path
			for _, p := range op.Parameters {
				if p.In == "path" {
					params = append(params, *p)
					displayPath = strings.ReplaceAll(displayPath, "{"+p.Name+"}", fmt.Sprintf("${%s}", p.Name))
				}
			}

			// Only application/json
			reqType := ""
			if op.RequestBody != nil {
				if jsonContent, ok := op.RequestBody.Content["application/json"]; ok {
					reqType = resolveType(jsonContent.Schema)
				}
			}

			resType := "any"
			for _, code := range []string{"200", "201", "default"} {
				if resp, ok := op.Responses[code]; ok {
					if jsonContent, ok := resp.Content["application/json"]; ok {
						if jsonContent.Schema != nil {
							resType = resolveType(jsonContent.Schema)
							break
						}
					}
				}
			}

			ops = append(ops, NamedOperation{
				ID:           id,
				Method:       method,
				DisplayPath:  displayPath,
				Params:       params,
				HasBody:      op.RequestBody != nil,
				RequestType:  reqType,
				ResponseType: resType,
			})
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	funcs := template.FuncMap{
		"tsType": resolveType,
		"argList": func(op NamedOperation) string {
			args := []string{}
			for _, p := range op.Params {
				args = append(args, fmt.Sprintf("%s: %s", p.Name, resolveType(p.Schema)))
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
	}

	tmpl := template.Must(template.New("api").Funcs(funcs).Parse(apiTemplate))
	err = tmpl.Execute(f, map[string]any{
		"Schemas":  api.Components.Schemas,
		"Ops":      ops,
		"Instance": instance,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("✅ Generated fetch client: %s\n", outputPath)
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
			return "Record<string, " + resolveType(s.AdditionalProperties) + ">"
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

import client from '{{ .Instance }}';

{{range .Ops}}
export const {{.ID}} = ({{argList .}}): Promise<{{responseType .}}> =>
  client.{{.Method}}(` + "`" + `{{.DisplayPath}}` + "`" + `{{if .HasBody}}, body{{end}});
{{end}}

{{range $name, $schema := .Schemas}}
{{- if $schema.Description}}
/** {{$schema.Description}} */
{{end}}
export interface {{$name}} {
{{- range $prop, $def := $schema.Properties }}
  {{- if $def.Description}}
  /** {{ $def.Description }} */
  {{end}}
  {{$prop}}: {{ tsType $def }};
{{- end }}
}
{{end}}
`
