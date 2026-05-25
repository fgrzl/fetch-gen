package parser

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	apitypes "github.com/fgrzl/fetch-gen/internal/types"
	"gopkg.in/yaml.v3"
)

type validationError struct {
	Path    string
	Message string
}

func (e validationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func ParseDocument(inputPath string, data []byte) (*apitypes.OpenAPI, error) {
	var api apitypes.OpenAPI

	switch ext := strings.ToLower(filepath.Ext(inputPath)); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &api); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &api); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file type (must be .yaml or .json)")
	}

	if err := validateOpenAPI(&api); err != nil {
		return nil, err
	}

	return &api, nil
}

func validateOpenAPI(api *apitypes.OpenAPI) error {
	if api == nil {
		return validationError{Message: "openapi document is empty"}
	}

	componentNames := map[string]struct{}{}
	for name := range api.Components.Schemas {
		componentNames[name] = struct{}{}
	}
	for name, parameter := range api.Components.Parameters {
		paramPath := fmt.Sprintf("components.parameters[%q]", name)
		resolved, err := resolveParameter(api, parameter, map[string]struct{}{})
		if err != nil {
			return validationError{Path: paramPath, Message: err.Error()}
		}
		if err := validateParameter(paramPath, resolved, componentNames); err != nil {
			return err
		}
	}

	seenOperationIDs := map[string]string{}
	for path, methods := range api.Paths {
		pathParams, err := extractPathTemplateParams(path)
		if err != nil {
			return validationError{Path: fmt.Sprintf("paths[%q]", path), Message: err.Error()}
		}

		pathParamSet := map[string]struct{}{}
		for _, name := range pathParams {
			pathParamSet[name] = struct{}{}
		}

		for method, op := range methods {
			opPath := fmt.Sprintf("paths[%q][%q]", path, method)
			if op == nil {
				return validationError{Path: opPath, Message: "operation is null"}
			}
			if strings.TrimSpace(op.OperationID) == "" {
				return validationError{Path: opPath + ".operationId", Message: "missing operationId"}
			}
			if previousPath, exists := seenOperationIDs[op.OperationID]; exists {
				return validationError{Path: opPath + ".operationId", Message: fmt.Sprintf("duplicate operationId already used at %s", previousPath)}
			}
			seenOperationIDs[op.OperationID] = opPath

			seenPathParams := map[string]struct{}{}
			for i, param := range op.Parameters {
				paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, i)
				resolved, err := resolveParameter(api, param, map[string]struct{}{})
				if err != nil {
					return validationError{Path: paramPath, Message: err.Error()}
				}
				if err := validateParameter(paramPath, resolved, componentNames); err != nil {
					return err
				}
				if resolved.In == "path" {
					if !resolved.Required {
						return validationError{Path: paramPath + ".required", Message: "path parameters must be required"}
					}
					if _, ok := pathParamSet[resolved.Name]; !ok {
						return validationError{Path: paramPath + ".name", Message: fmt.Sprintf("path parameter %q is not declared in path template %s", resolved.Name, path)}
					}
					seenPathParams[resolved.Name] = struct{}{}
				}
			}

			for _, name := range pathParams {
				if _, ok := seenPathParams[name]; !ok {
					return validationError{Path: opPath + ".parameters", Message: fmt.Sprintf("missing path parameter %q for template %s", name, path)}
				}
			}

			if op.RequestBody != nil {
				if len(op.RequestBody.Content) == 0 {
					return validationError{Path: opPath + ".requestBody.content", Message: "request body has no content"}
				}
				for contentType, media := range op.RequestBody.Content {
					contentPath := fmt.Sprintf("%s.requestBody.content[%q]", opPath, contentType)
					if media == nil {
						return validationError{Path: contentPath, Message: "media type is null"}
					}
					if media.Schema == nil {
						return validationError{Path: contentPath + ".schema", Message: "missing schema"}
					}
					if err := validateSchema(contentPath+".schema", media.Schema, componentNames, map[*apitypes.Schema]struct{}{}); err != nil {
						return err
					}
				}
			}

			if err := validateResponses(opPath, op.Responses, componentNames); err != nil {
				return err
			}
		}
	}

	for name, schema := range api.Components.Schemas {
		if err := validateSchema(fmt.Sprintf("components.schemas[%q]", name), schema, componentNames, map[*apitypes.Schema]struct{}{}); err != nil {
			return err
		}
	}

	return nil
}

func validateParameter(path string, param *apitypes.Parameter, componentNames map[string]struct{}) error {
	if param == nil {
		return validationError{Path: path, Message: "parameter is null"}
	}
	if strings.TrimSpace(param.Name) == "" {
		return validationError{Path: path + ".name", Message: "missing parameter name"}
	}
	if param.In != "path" && param.In != "query" {
		return validationError{Path: path + ".in", Message: fmt.Sprintf("unsupported parameter location %q", param.In)}
	}
	if param.Schema == nil {
		return validationError{Path: path + ".schema", Message: "missing schema"}
	}
	return validateSchema(path+".schema", param.Schema, componentNames, map[*apitypes.Schema]struct{}{})
}

func resolveParameter(api *apitypes.OpenAPI, param *apitypes.Parameter, seen map[string]struct{}) (*apitypes.Parameter, error) {
	if param == nil {
		return nil, fmt.Errorf("parameter is null")
	}
	if param.Ref == "" {
		return param, nil
	}
	const prefix = "#/components/parameters/"
	if !strings.HasPrefix(param.Ref, prefix) {
		return nil, fmt.Errorf("unsupported parameter ref %q", param.Ref)
	}
	name := strings.TrimPrefix(param.Ref, prefix)
	if name == "" || strings.Contains(name, "/") {
		return nil, fmt.Errorf("unsupported parameter ref %q", param.Ref)
	}
	if _, ok := seen[name]; ok {
		return nil, fmt.Errorf("cyclic parameter ref %q", param.Ref)
	}
	component, ok := api.Components.Parameters[name]
	if !ok {
		return nil, fmt.Errorf("unresolved parameter ref %q", param.Ref)
	}
	seen[name] = struct{}{}
	return resolveParameter(api, component, seen)
}

func validateResponses(opPath string, responses map[string]*apitypes.Response, componentNames map[string]struct{}) error {
	if len(responses) == 0 {
		return validationError{Path: opPath + ".responses", Message: "missing responses"}
	}

	for code, resp := range responses {
		responsePath := fmt.Sprintf("%s.responses[%q]", opPath, code)
		if resp == nil {
			return validationError{Path: responsePath, Message: "response is null"}
		}
		if len(resp.Content) == 0 {
			continue
		}
		for contentType, media := range resp.Content {
			contentPath := fmt.Sprintf("%s.content[%q]", responsePath, contentType)
			if media.Schema == nil {
				return validationError{Path: contentPath + ".schema", Message: "missing schema"}
			}
			if err := validateSchema(contentPath+".schema", media.Schema, componentNames, map[*apitypes.Schema]struct{}{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateSchema(path string, s *apitypes.Schema, componentNames map[string]struct{}, seen map[*apitypes.Schema]struct{}) error {
	if s == nil {
		return validationError{Path: path, Message: "missing schema"}
	}
	if _, ok := seen[s]; ok {
		return nil
	}
	seen[s] = struct{}{}

	if s.Ref != "" {
		refName, ok := componentSchemaRefName(s.Ref)
		if !ok {
			return validationError{Path: path + ".$ref", Message: fmt.Sprintf("unsupported ref %q", s.Ref)}
		}
		if _, ok := componentNames[refName]; !ok {
			return validationError{Path: path + ".$ref", Message: fmt.Sprintf("unresolved ref %q", s.Ref)}
		}
		return nil
	}

	for _, schemaType := range s.Type.Values {
		switch schemaType {
		case "string", "integer", "number", "boolean", "null", "array", "object":
		default:
			return validationError{Path: path + ".type", Message: fmt.Sprintf("unsupported schema type %q", schemaType)}
		}
	}

	for i, enumValue := range s.Enum {
		switch enumValue.(type) {
		case nil, string, bool,
			int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64, json.Number:
		default:
			return validationError{Path: fmt.Sprintf("%s.enum[%d]", path, i), Message: fmt.Sprintf("unsupported enum value of type %T", enumValue)}
		}
	}

	for key, subSchema := range s.Properties {
		if subSchema == nil {
			return validationError{Path: fmt.Sprintf("%s.properties[%q]", path, key), Message: "property schema is null"}
		}
		if err := validateSchema(fmt.Sprintf("%s.properties[%q]", path, key), subSchema, componentNames, seen); err != nil {
			return err
		}
	}

	if s.Items != nil {
		if err := validateSchema(path+".items", s.Items, componentNames, seen); err != nil {
			return err
		}
	}

	for i, subSchema := range s.AllOf {
		if subSchema == nil {
			return validationError{Path: fmt.Sprintf("%s.allOf[%d]", path, i), Message: "schema is null"}
		}
		if err := validateSchema(fmt.Sprintf("%s.allOf[%d]", path, i), subSchema, componentNames, seen); err != nil {
			return err
		}
	}

	for i, subSchema := range s.OneOf {
		if subSchema == nil {
			return validationError{Path: fmt.Sprintf("%s.oneOf[%d]", path, i), Message: "schema is null"}
		}
		if err := validateSchema(fmt.Sprintf("%s.oneOf[%d]", path, i), subSchema, componentNames, seen); err != nil {
			return err
		}
	}

	for i, subSchema := range s.AnyOf {
		if subSchema == nil {
			return validationError{Path: fmt.Sprintf("%s.anyOf[%d]", path, i), Message: "schema is null"}
		}
		if err := validateSchema(fmt.Sprintf("%s.anyOf[%d]", path, i), subSchema, componentNames, seen); err != nil {
			return err
		}
	}

	if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
		if err := validateSchema(path+".additionalProperties", s.AdditionalProperties.Schema, componentNames, seen); err != nil {
			return err
		}
	}

	return nil
}

func extractPathTemplateParams(path string) ([]string, error) {
	params := []string{}
	remaining := path

	for {
		start := strings.Index(remaining, "{")
		if start == -1 {
			break
		}
		end := strings.Index(remaining[start+1:], "}")
		if end == -1 {
			return nil, fmt.Errorf("malformed path template: missing closing }")
		}
		name := remaining[start+1 : start+1+end]
		if name == "" {
			return nil, fmt.Errorf("malformed path template: empty parameter name")
		}
		params = append(params, name)
		remaining = remaining[start+1+end+1:]
	}

	return params, nil
}

func componentSchemaRefName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}
