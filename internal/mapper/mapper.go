package mapper

import (
	"github.com/getkin/kin-openapi/openapi3"
	"gopenapi/internal/templates"
	"gopenapi/internal/utils"
	"strings"
)

func MapModelsFromSchemas(doc *openapi3.T) []templates.Model {
	var models []templates.Model
	for name, schema := range doc.Components.Schemas {
		if schema.Value == nil {
			continue
		}
		parseSchema(name, schema, &models)
	}
	return models
}

func parseSchema(name string, schema *openapi3.SchemaRef, models *[]templates.Model) string {
	if schema == nil || schema.Value == nil {
		return "interface{}"
	}
	if schema.Value.Type.Is(openapi3.TypeString) {
		return "string"
	}
	if schema.Value.Type.Is(openapi3.TypeInteger) {
		return "int"
	}
	if schema.Value.Type.Is(openapi3.TypeNumber) {
		return "float64"
	}
	if schema.Value.Type.Is(openapi3.TypeBoolean) {
		return "bool"
	}
	if schema.Value.Type.Is(openapi3.TypeArray) {
		itemType := parseSchema(name+"Item", schema.Value.Items, models)
		return "[]" + itemType
	}
	if schema.Value.Type.Is(openapi3.TypeObject) {
		modelName := utils.CapitalizeFirstWord(name)
		var fields []templates.ModelProp
		for propName, propSchema := range schema.Value.Properties {
			goType := parseSchema(name+utils.CapitalizeFirstWord(propName), propSchema, models)
			fields = append(fields, templates.ModelProp{
				GoName:      utils.CapitalizeFirstWord(propName),
				GoType:      goType,
				JSONName:    propName,
				Description: propSchema.Value.Description,
			})
		}
		*models = append(*models, templates.Model{
			Name:         modelName,
			OriginalName: name,
			Fields:       fields,
		})
		return modelName
	}
	return "interface{}"
}

func MapAPIFromPaths(doc *openapi3.T) templates.APIs {
	apis := templates.APIs{}
	for path, item := range doc.Paths.Map() {
		for method, operation := range item.Operations() {
			tag := "default"
			if len(operation.Tags) > 0 {
				tag = strings.ToLower(operation.Tags[0])
			}
			var reqBody *templates.RequestBody
			resp := mapResponses(operation.Responses)
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				reqBody = mapRequestBody(operation.RequestBody.Value)
			}
			apis[tag] = append(apis[tag], templates.API{
				OperationID: utils.CapitalizeFirstWord(operation.OperationID),
				Method:      strings.ToUpper(method),
				Path:        cleanPath(path),
				Description: operation.Description,
				RequestBody: reqBody,
				Response:    resp,
			})
		}
	}
	return apis
}

func mapRequestBody(value *openapi3.RequestBody) *templates.RequestBody {
	var reqBody *templates.RequestBody
	for mt, mediaType := range value.Content {
		if mt == "application/json" && mediaType.Schema != nil && mediaType.Schema.Ref != "" {
			parts := strings.Split(mediaType.Schema.Ref, "/")
			schemaName := parts[len(parts)-1]
			reqBody = &templates.RequestBody{
				ModelName: utils.CapitalizeFirstWord(schemaName),
			}
		}
	}
	return reqBody
}

func mapResponses(resp *openapi3.Responses) *templates.Response {
	for status, response := range resp.Map() {
		if status[0] == '2' && response.Value != nil {
			for mt, media := range response.Value.Content {
				if mt == "application/json" && media.Schema != nil && media.Schema.Ref != "" {
					parts := strings.Split(media.Schema.Ref, "/")
					schemaName := parts[len(parts)-1]
					return &templates.Response{
						Status:    status,
						ModelName: utils.CapitalizeFirstWord(schemaName),
					}
				}
			}
		}
	}
	return nil
}

func cleanPath(path string) string {
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	return path
}
