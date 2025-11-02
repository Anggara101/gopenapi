package mapper

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"gopenapi/internal/templates"
)

func TestMapModelsFromSchemas_BasicAndNested(t *testing.T) {
	doc := &openapi3.T{
		Components: &openapi3.Components{},
	}
	doc.Components.Schemas = openapi3.Schemas{}

	// Person schema:
	// - id: integer
	// - name: string
	// - tags: []string
	// - profile: object { age: integer }
	personSchema := &openapi3.Schema{
		Type: &openapi3.Types{openapi3.TypeObject},
		Properties: openapi3.Schemas{
			"id": {
				Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeInteger}},
			},
			"name": {
				Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString}},
			},
			"tags": {
				Value: &openapi3.Schema{
					Type:  &openapi3.Types{openapi3.TypeArray},
					Items: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString}}},
				},
			},
			"profile": {
				Value: &openapi3.Schema{
					Type: &openapi3.Types{openapi3.TypeObject},
					Properties: openapi3.Schemas{
						"age": {Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeInteger}}},
					},
				},
			},
		},
	}
	doc.Components.Schemas["Person"] = &openapi3.SchemaRef{Value: personSchema}

	models := MapModelsFromSchemas(doc)

	// Expect at least Person and PersonProfile to be generated
	if len(models) < 2 {
		t.Fatalf("expected at least 2 models (Person and PersonProfile), got %d", len(models))
	}

	person := findModel(models, "Person")
	if person == nil {
		t.Fatalf("expected model Person to be generated")
	}
	if person.OriginalName != "Person" {
		t.Errorf("expected Person.OriginalName to be 'Person', got %q", person.OriginalName)
	}
	assertField(t, person.Fields, "Id", "int", "id")
	assertField(t, person.Fields, "Name", "string", "name")
	assertField(t, person.Fields, "Tags", "[]string", "tags")
	assertField(t, person.Fields, "Profile", "PersonProfile", "profile")

	personProfile := findModel(models, "PersonProfile")
	if personProfile == nil {
		t.Fatalf("expected nested model PersonProfile to be generated")
	}
	assertField(t, personProfile.Fields, "Age", "int", "age")
}

func TestMapAPIFromPaths_WithGetAndPost(t *testing.T) {
	doc := &openapi3.T{
		Components: &openapi3.Components{},
	}

	// Minimal schema (not strictly required for mapping by ref, but fine to include)
	doc.Components.Schemas = openapi3.Schemas{
		"Person": {Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeObject}}},
	}

	doc.Paths = openapi3.NewPaths()

	// GET /users/{id} -> 200 application/json -> #/components/schemas/Person
	getOp := &openapi3.Operation{
		OperationID: "getUser",
		Description: "Get user",
		Tags:        []string{"Users"},
		Responses:   openapi3.NewResponses(),
	}
	getOp.Responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/Person"},
				},
			},
		},
	})

	// POST /users -> requestBody application/json -> #/components/schemas/Person, 201 response same model
	postOp := &openapi3.Operation{
		OperationID: "createUser",
		Tags:        []string{"Users"},
		RequestBody: &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/Person"},
					},
				},
			},
		},
		Responses: openapi3.NewResponses(),
	}
	postOp.Responses.Set("201", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/Person"},
				},
			},
		},
	})

	doc.Paths.Set("/users/{id}", &openapi3.PathItem{Get: getOp})
	doc.Paths.Set("/users", &openapi3.PathItem{Post: postOp})

	apis := MapAPIFromPaths(doc)

	usersAPIs, ok := apis["users"]
	if !ok {
		t.Fatalf("expected tag 'users' to be present")
	}
	if len(usersAPIs) != 2 {
		t.Fatalf("expected 2 APIs under 'users' tag, got %d", len(usersAPIs))
	}

	getAPI := findAPIByOperationID(apis, "users", "GetUser")
	if getAPI == nil {
		t.Fatalf("expected GET operation with OperationID 'GetUser'")
	}
	if getAPI.Method != "GET" {
		t.Errorf("expected GET method, got %q", getAPI.Method)
	}
	if getAPI.Path != "/users/:id" {
		t.Errorf("expected cleaned path '/users/:id', got %q", getAPI.Path)
	}
	if getAPI.RequestBody != nil {
		t.Errorf("expected GET to have no request body, got %+v", getAPI.RequestBody)
	}
	if getAPI.Response == nil {
		t.Fatalf("expected GET to have a response mapped")
	}
	if getAPI.Response.ModelName != "Person" {
		t.Errorf("expected GET Response.ModelName 'Person', got %q", getAPI.Response.ModelName)
	}
	if len(getAPI.Response.Status) == 0 || getAPI.Response.Status[0] != '2' {
		t.Errorf("expected a 2xx status for GET, got %q", getAPI.Response.Status)
	}

	postAPI := findAPIByOperationID(apis, "users", "CreateUser")
	if postAPI == nil {
		t.Fatalf("expected POST operation with OperationID 'CreateUser'")
	}
	if postAPI.Method != "POST" {
		t.Errorf("expected POST method, got %q", postAPI.Method)
	}
	if postAPI.Path != "/users" {
		t.Errorf("expected path '/users', got %q", postAPI.Path)
	}
	if postAPI.RequestBody == nil {
		t.Fatalf("expected POST to have a request body")
	}
	if postAPI.RequestBody.ModelName != "Person" {
		t.Errorf("expected POST RequestBody.ModelName 'Person', got %q", postAPI.RequestBody.ModelName)
	}
	if postAPI.Response == nil {
		t.Fatalf("expected POST to have a response mapped")
	}
	if postAPI.Response.ModelName != "Person" {
		t.Errorf("expected POST Response.ModelName 'Person', got %q", postAPI.Response.ModelName)
	}
}

func TestCleanPath(t *testing.T) {
	in := "/pets/{id}/owners/{ownerId}"
	want := "/pets/:id/owners/:ownerId"
	if got := cleanPath(in); got != want {
		t.Fatalf("cleanPath(%q) = %q, want %q", in, got, want)
	}
}

// Helpers

func findModel(models []templates.Model, name string) *templates.Model {
	for i := range models {
		if models[i].Name == name {
			return &models[i]
		}
	}
	return nil
}

func assertField(t *testing.T, fields []templates.ModelProp, goName, goType, jsonName string) {
	t.Helper()
	for _, f := range fields {
		if f.GoName == goName {
			if f.GoType != goType {
				t.Fatalf("field %s: expected GoType %q, got %q", goName, goType, f.GoType)
			}
			if f.JSONName != jsonName {
				t.Fatalf("field %s: expected JSONName %q, got %q", goName, jsonName, f.JSONName)
			}
			return
		}
	}
	t.Fatalf("expected field %q not found", goName)
}

func findAPIByOperationID(apis templates.APIs, tag, opID string) *templates.API {
	group, ok := apis[tag]
	if !ok {
		return nil
	}
	for i := range group {
		if group[i].OperationID == opID {
			return &group[i]
		}
	}
	return nil
}
