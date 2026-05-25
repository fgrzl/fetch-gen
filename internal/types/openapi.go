package types

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// AdditionalProperties can be either boolean or Schema.
type AdditionalProperties struct {
	Boolean *bool
	Schema  *Schema
}

// SchemaType supports OpenAPI 3.1 / JSON Schema where type can be a string or an array of strings.
// Examples: "string" or ["string", "null"].
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

	return fmt.Errorf("schema type: unsupported YAML shape")
}

func (st *SchemaType) UnmarshalJSON(data []byte) error {
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

	return fmt.Errorf("schema type: unsupported JSON shape")
}

func (st SchemaType) Has(t string) bool {
	for _, v := range st.Values {
		if v == t {
			return true
		}
	}
	return false
}

func (st SchemaType) IsEmpty() bool {
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
	Servers    []Server                         `json:"servers" yaml:"servers"`
}

type Server struct {
	URL string `json:"url" yaml:"url"`
}

type Components struct {
	Schemas    map[string]*Schema    `json:"schemas" yaml:"schemas"`
	Parameters map[string]*Parameter `json:"parameters" yaml:"parameters"`
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
	Required bool                  `json:"required" yaml:"required"`
	Content  map[string]*MediaType `json:"content" yaml:"content"`
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
	Format               string                `json:"format" yaml:"format"`
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
	Ref         string  `json:"$ref" yaml:"$ref"`
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Required    bool    `json:"required" yaml:"required"`
	Schema      *Schema `json:"schema" yaml:"schema"`
	Description string  `json:"description" yaml:"description"`
}
