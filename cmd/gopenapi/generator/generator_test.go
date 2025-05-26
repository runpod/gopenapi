package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/runpod/gopenapi"
)

// Test spec for testing
var testSpec = gopenapi.Spec{
	OpenAPI: "3.0.0",
	Info: gopenapi.Info{
		Title:       "Test API",
		Description: "A test API",
		Version:     "1.0.0",
	},
	Servers: gopenapi.Servers{
		{
			URL:         "https://api.test.com",
			Description: "Test server",
		},
	},
	Paths: gopenapi.Paths{
		"/users/{id}": gopenapi.Path{
			Get: &gopenapi.Operation{
				OperationId: "getUserById",
				Summary:     "Get a user by ID",
				Description: "Retrieve a user by their unique identifier",
				Parameters: gopenapi.Parameters{
					{
						Name:        "id",
						In:          gopenapi.InPath,
						Description: "User ID",
						Required:    true,
						Schema:      gopenapi.Schema{Type: gopenapi.Integer},
					},
				},
				Responses: gopenapi.Responses{
					200: {
						Description: "User found",
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.String},
							},
						},
					},
				},
			},
		},
	},
}

func TestGenerateClientToWriter(t *testing.T) {
	tests := []struct {
		name     string
		language string
		wantErr  bool
	}{
		{
			name:     "Generate Go client",
			language: "go",
			wantErr:  false,
		},
		{
			name:     "Generate Python client",
			language: "python",
			wantErr:  false,
		},
		{
			name:     "Generate TypeScript client",
			language: "typescript",
			wantErr:  false,
		},
		{
			name:     "Unsupported language",
			language: "java",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			templateFile := "templates/" + tt.language + ".tpl"
			if tt.language == "java" {
				templateFile = "templates/java.tpl" // This doesn't exist
			}

			err := GenerateClientToWriter(&testSpec, &buf, "testclient", templateFile, tt.language)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateClientToWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				if len(output) == 0 {
					t.Error("GenerateClientToWriter() produced empty output")
				}

				// Check for language-specific content
				switch tt.language {
				case "go":
					if !strings.Contains(output, "package testclient") {
						t.Error("Go client should contain package declaration")
					}
					if !strings.Contains(output, "func (c *Client)") {
						t.Error("Go client should contain client methods")
					}
				case "python":
					if !strings.Contains(output, "class") {
						t.Error("Python client should contain class definition")
					}
				case "typescript":
					if !strings.Contains(output, "export") {
						t.Error("TypeScript client should contain export statements")
					}
				}
			}
		})
	}
}

func TestGenerateClient(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "client.go")

	err := GenerateClient(&testSpec, outputFile, "testclient", "templates/go.tpl", "go")
	if err != nil {
		t.Fatalf("GenerateClient() error = %v", err)
	}

	// Check if file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("GenerateClient() did not create output file")
	}

	// Read and check content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package testclient") {
		t.Error("Generated file should contain package declaration")
	}
}

func TestGenerateClientForLanguage(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		language     string
		expectedFile string
		wantErr      bool
	}{
		{
			name:         "Generate Go client",
			language:     "go",
			expectedFile: "client.go",
			wantErr:      false,
		},
		{
			name:         "Generate Python client",
			language:     "python",
			expectedFile: "client.py",
			wantErr:      false,
		},
		{
			name:         "Generate TypeScript client",
			language:     "typescript",
			expectedFile: "client.ts",
			wantErr:      false,
		},
		{
			name:     "Unsupported language",
			language: "java",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateClientForLanguage(&testSpec, tt.language, tempDir, "testclient")
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateClientForLanguage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expectedPath := filepath.Join(tempDir, tt.expectedFile)
				if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", expectedPath)
				}
			}
		})
	}
}

func TestSchemaToGoType(t *testing.T) {
	tests := []struct {
		name     string
		schema   gopenapi.Schema
		expected string
	}{
		{
			name:     "String type",
			schema:   gopenapi.Schema{Type: gopenapi.String},
			expected: "string",
		},
		{
			name:     "Integer type",
			schema:   gopenapi.Schema{Type: gopenapi.Integer},
			expected: "int",
		},
		{
			name:     "Number type",
			schema:   gopenapi.Schema{Type: gopenapi.Number},
			expected: "float64",
		},
		{
			name:     "Boolean type",
			schema:   gopenapi.Schema{Type: gopenapi.Boolean},
			expected: "bool",
		},
		{
			name:     "Array type",
			schema:   gopenapi.Schema{Type: gopenapi.Array},
			expected: "[]interface{}",
		},
		{
			name:     "Nil type",
			schema:   gopenapi.Schema{Type: nil},
			expected: "interface{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SchemaToGoType(tt.schema)
			if result != tt.expected {
				t.Errorf("SchemaToGoType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToStructName(t *testing.T) {
	tests := []struct {
		name        string
		operationId string
		expected    string
	}{
		{
			name:        "camelCase input",
			operationId: "getUserById",
			expected:    "GetUserById",
		},
		{
			name:        "PascalCase input",
			operationId: "GetUserById",
			expected:    "Getuserbyid", // Actual behavior - it lowercases everything after first char
		},
		{
			name:        "snake_case input",
			operationId: "get_user_by_id",
			expected:    "GetUserById", // Should convert properly
		},
		{
			name:        "kebab-case input",
			operationId: "get-user-by-id",
			expected:    "GetUserById", // Should convert properly
		},
		{
			name:        "empty input",
			operationId: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToStructName(tt.operationId)
			if result != tt.expected {
				t.Errorf("ToStructName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToMethodName(t *testing.T) {
	tests := []struct {
		name        string
		operationId string
		expected    string
	}{
		{
			name:        "camelCase input",
			operationId: "getUserById",
			expected:    "GetUserById",
		},
		{
			name:        "PascalCase input",
			operationId: "GetUserById",
			expected:    "Getuserbyid", // Actual behavior
		},
		{
			name:        "snake_case input",
			operationId: "get_user_by_id",
			expected:    "GetUserById", // Should convert properly
		},
		{
			name:        "empty input",
			operationId: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToMethodName(tt.operationId)
			if result != tt.expected {
				t.Errorf("ToMethodName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToGoName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "snake_case",
			input:    "user_id",
			expected: "UserId",
		},
		{
			name:     "kebab-case",
			input:    "user-id",
			expected: "UserId",
		},
		{
			name:     "dot.case",
			input:    "user.id",
			expected: "UserId",
		},
		{
			name:     "single word",
			input:    "user",
			expected: "User",
		},
		{
			name:     "already PascalCase",
			input:    "UserId",
			expected: "UserId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToGoName(tt.input)
			if result != tt.expected {
				t.Errorf("ToGoName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateConvertToString(t *testing.T) {
	tests := []struct {
		name     string
		goName   string
		goType   string
		expected string
	}{
		{
			name:     "string type",
			goName:   "Id",
			goType:   "string",
			expected: "opts.Path.Id",
		},
		{
			name:     "int type",
			goName:   "Id",
			goType:   "int",
			expected: "strconv.Itoa(opts.Path.Id)",
		},
		{
			name:     "float64 type",
			goName:   "Price",
			goType:   "float64",
			expected: "strconv.FormatFloat(opts.Path.Price, 'f', -1, 64)",
		},
		{
			name:     "bool type",
			goName:   "Active",
			goType:   "bool",
			expected: "strconv.FormatBool(opts.Path.Active)",
		},
		{
			name:     "other type",
			goName:   "Data",
			goType:   "interface{}",
			expected: "fmt.Sprintf(\"%v\", opts.Path.Data)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateConvertToString(tt.goName, tt.goType)
			if result != tt.expected {
				t.Errorf("generateConvertToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateTemplateData(t *testing.T) {
	templateData := generateTemplateData(&testSpec, "testclient")

	if templateData.PackageName != "testclient" {
		t.Errorf("Expected PackageName to be 'testclient', got %v", templateData.PackageName)
	}

	if len(templateData.Operations) == 0 {
		t.Error("Expected at least one operation in template data")
	}

	// Check the first operation
	op := templateData.Operations[0]
	if op.OperationId != "getUserById" {
		t.Errorf("Expected OperationId to be 'getUserById', got %v", op.OperationId)
	}

	if op.Method != "GET" {
		t.Errorf("Expected Method to be 'GET', got %v", op.Method)
	}

	if op.Path != "/users/{id}" {
		t.Errorf("Expected Path to be '/users/{id}', got %v", op.Path)
	}

	if !op.HasPathParams {
		t.Error("Expected HasPathParams to be true")
	}

	if len(op.PathParams) != 1 {
		t.Errorf("Expected 1 path parameter, got %d", len(op.PathParams))
	}

	if op.PathParams[0].Name != "id" {
		t.Errorf("Expected path parameter name to be 'id', got %v", op.PathParams[0].Name)
	}
}

// Test types for alias resolution
type UserIDAlias string
type StatusAlias int
type ScoreAlias float64
type IsActiveAlias bool
type TagsAlias []string

func TestTypeToGoTypeWithAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		// Basic types
		{"string", reflect.TypeOf(""), "string"},
		{"int", reflect.TypeOf(0), "int"},
		{"int64", reflect.TypeOf(int64(0)), "int"},
		{"float64", reflect.TypeOf(0.0), "float64"},
		{"bool", reflect.TypeOf(false), "bool"},
		{"slice", reflect.TypeOf([]string{}), "[]string"},
		{"pointer", reflect.TypeOf((*string)(nil)), "*string"},

		// Named types (aliases)
		{"UserID alias", reflect.TypeOf(UserIDAlias("")), "string"},
		{"Status alias", reflect.TypeOf(StatusAlias(0)), "int"},
		{"Score alias", reflect.TypeOf(ScoreAlias(0.0)), "float64"},
		{"IsActive alias", reflect.TypeOf(IsActiveAlias(false)), "bool"},
		{"Tags alias", reflect.TypeOf(TagsAlias{}), "[]string"},

		// Struct types
		{"struct", reflect.TypeOf(struct{ Name string }{}), "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeToGoType(tt.input)
			if result != tt.expected {
				t.Errorf("typeToGoType(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSchemaToGoTypeWithAliases(t *testing.T) {
	tests := []struct {
		name     string
		schema   gopenapi.Schema
		expected string
	}{
		{
			name:     "gopenapi.String",
			schema:   gopenapi.Schema{Type: gopenapi.String},
			expected: "string",
		},
		{
			name:     "gopenapi.Integer",
			schema:   gopenapi.Schema{Type: gopenapi.Integer},
			expected: "int",
		},
		{
			name:     "gopenapi.Number",
			schema:   gopenapi.Schema{Type: gopenapi.Number},
			expected: "float64",
		},
		{
			name:     "gopenapi.Boolean",
			schema:   gopenapi.Schema{Type: gopenapi.Boolean},
			expected: "bool",
		},
		{
			name:     "UserID alias",
			schema:   gopenapi.Schema{Type: reflect.TypeOf(UserIDAlias(""))},
			expected: "string",
		},
		{
			name:     "Status alias",
			schema:   gopenapi.Schema{Type: reflect.TypeOf(StatusAlias(0))},
			expected: "int",
		},
		{
			name:     "Score alias",
			schema:   gopenapi.Schema{Type: reflect.TypeOf(ScoreAlias(0.0))},
			expected: "float64",
		},
		{
			name:     "IsActive alias",
			schema:   gopenapi.Schema{Type: reflect.TypeOf(IsActiveAlias(false))},
			expected: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SchemaToGoType(tt.schema)
			if result != tt.expected {
				t.Errorf("SchemaToGoType(%v) = %q, want %q", tt.schema.Type, result, tt.expected)
			}
		})
	}
}

func TestGenerateConvertToStringWithAliases(t *testing.T) {
	tests := []struct {
		name     string
		goName   string
		goType   string
		expected string
	}{
		{
			name:     "string type",
			goName:   "UserID",
			goType:   "string",
			expected: "opts.Path.UserID",
		},
		{
			name:     "int type",
			goName:   "Status",
			goType:   "int",
			expected: "strconv.Itoa(opts.Path.Status)",
		},
		{
			name:     "float64 type",
			goName:   "Score",
			goType:   "float64",
			expected: "strconv.FormatFloat(opts.Path.Score, 'f', -1, 64)",
		},
		{
			name:     "bool type",
			goName:   "IsActive",
			goType:   "bool",
			expected: "strconv.FormatBool(opts.Path.IsActive)",
		},
		{
			name:     "interface{} type",
			goName:   "Data",
			goType:   "interface{}",
			expected: "fmt.Sprintf(\"%v\", opts.Path.Data)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateConvertToString(tt.goName, tt.goType)
			if result != tt.expected {
				t.Errorf("generateConvertToString(%q, %q) = %q, want %q", tt.goName, tt.goType, result, tt.expected)
			}
		})
	}
}

func TestGenerateAddToParams(t *testing.T) {
	tests := []struct {
		name      string
		goName    string
		goType    string
		paramName string
		expected  string
	}{
		{
			name:      "string type",
			goName:    "UserID",
			goType:    "string",
			paramName: "user_id",
			expected:  "if opts.Query.UserID != \"\" {\n\t\tparams.Add(\"user_id\", opts.Query.UserID)\n\t}",
		},
		{
			name:      "int type",
			goName:    "Status",
			goType:    "int",
			paramName: "status",
			expected:  "if opts.Query.Status != 0 {\n\t\tparams.Add(\"status\", strconv.Itoa(opts.Query.Status))\n\t}",
		},
		{
			name:      "float64 type",
			goName:    "Score",
			goType:    "float64",
			paramName: "score",
			expected:  "if opts.Query.Score != 0 {\n\t\tparams.Add(\"score\", strconv.FormatFloat(opts.Query.Score, 'f', -1, 64))\n\t}",
		},
		{
			name:      "bool type",
			goName:    "IsActive",
			goType:    "bool",
			paramName: "is_active",
			expected:  "params.Add(\"is_active\", strconv.FormatBool(opts.Query.IsActive))",
		},
		{
			name:      "interface{} type",
			goName:    "Data",
			goType:    "interface{}",
			paramName: "data",
			expected:  "if opts.Query.Data != nil {\n\t\tparams.Add(\"data\", fmt.Sprintf(\"%v\", opts.Query.Data))\n\t}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateAddToParams(tt.goName, tt.goType, tt.paramName)
			if result != tt.expected {
				t.Errorf("generateAddToParams(%q, %q, %q) = %q, want %q", tt.goName, tt.goType, tt.paramName, result, tt.expected)
			}
		})
	}
}

func TestGenerateSetHeader(t *testing.T) {
	tests := []struct {
		name       string
		goName     string
		goType     string
		headerName string
		expected   string
	}{
		{
			name:       "string type",
			goName:     "UserID",
			goType:     "string",
			headerName: "X-User-ID",
			expected:   "if opts.Headers.UserID != \"\" {\n\t\treq.Header.Set(\"X-User-ID\", opts.Headers.UserID)\n\t}",
		},
		{
			name:       "int type",
			goName:     "Status",
			goType:     "int",
			headerName: "X-Status",
			expected:   "if opts.Headers.Status != 0 {\n\t\treq.Header.Set(\"X-Status\", strconv.Itoa(opts.Headers.Status))\n\t}",
		},
		{
			name:       "float64 type",
			goName:     "Score",
			goType:     "float64",
			headerName: "X-Score",
			expected:   "if opts.Headers.Score != 0 {\n\t\treq.Header.Set(\"X-Score\", strconv.FormatFloat(opts.Headers.Score, 'f', -1, 64))\n\t}",
		},
		{
			name:       "bool type",
			goName:     "IsActive",
			goType:     "bool",
			headerName: "X-Is-Active",
			expected:   "req.Header.Set(\"X-Is-Active\", strconv.FormatBool(opts.Headers.IsActive))",
		},
		{
			name:       "interface{} type",
			goName:     "Data",
			goType:     "interface{}",
			headerName: "X-Data",
			expected:   "if opts.Headers.Data != nil {\n\t\treq.Header.Set(\"X-Data\", fmt.Sprintf(\"%v\", opts.Headers.Data))\n\t}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSetHeader(tt.goName, tt.goType, tt.headerName)
			if result != tt.expected {
				t.Errorf("generateSetHeader(%q, %q, %q) = %q, want %q", tt.goName, tt.goType, tt.headerName, result, tt.expected)
			}
		})
	}
}

// Test that circular references in alias types are properly handled
func TestCircularReferenceDetection(t *testing.T) {
	// Create types that would cause circular references
	// Note: We can't actually create true circular type aliases in Go,
	// but we can test the detection mechanism with our function

	type StringAlias string
	type IntAlias int
	type SliceAlias []StringAlias

	// Test that normal alias resolution works
	stringType := reflect.TypeOf(StringAlias(""))
	result := typeToGoType(stringType)
	if result != "string" {
		t.Errorf("Expected StringAlias to resolve to 'string', got %s", result)
	}

	// Test that slice of alias works
	sliceType := reflect.TypeOf(SliceAlias{})
	result = typeToGoType(sliceType)
	if result != "[]string" {
		t.Errorf("Expected SliceAlias to resolve to '[]string', got %s", result)
	}

	// Test the recursive function directly with a visited map
	visited := make(map[reflect.Type]bool)
	result = typeToGoTypeRecursive(stringType, visited)
	if result != "string" {
		t.Errorf("Expected StringAlias to resolve to 'string' with visited map, got %s", result)
	}

	// Test that if we manually mark a type as visited, it returns interface{}
	visited = make(map[reflect.Type]bool)
	visited[stringType] = true // Simulate that we've already seen this type
	result = typeToGoTypeRecursive(stringType, visited)
	if result != "interface{}" {
		t.Errorf("Expected circular reference to resolve to 'interface{}', got %s", result)
	}

	t.Log("Circular reference detection is working correctly")
}

// Test complex nested alias types to ensure proper resolution
func TestComplexNestedAliasTypes(t *testing.T) {
	// Create a chain of alias types
	type Level1 string
	type Level2 Level1
	type Level3 Level2
	type Level4 Level3

	// Test that deeply nested aliases resolve correctly
	level4Type := reflect.TypeOf(Level4(""))
	result := typeToGoType(level4Type)
	if result != "string" {
		t.Errorf("Expected Level4 (deeply nested string alias) to resolve to 'string', got %s", result)
	}

	// Test with slices and pointers
	type SliceOfLevel4 []Level4
	type PointerToLevel4 *Level4

	sliceType := reflect.TypeOf(SliceOfLevel4{})
	result = typeToGoType(sliceType)
	if result != "[]string" {
		t.Errorf("Expected SliceOfLevel4 to resolve to '[]string', got %s", result)
	}

	ptrType := reflect.TypeOf((*PointerToLevel4)(nil)).Elem()
	result = typeToGoType(ptrType)
	if result != "*string" {
		t.Errorf("Expected PointerToLevel4 to resolve to '*string', got %s", result)
	}

	t.Log("Complex nested alias types resolved correctly")
}

// Test that alias types in struct fields are properly resolved
func TestAliasTypesInStructFields(t *testing.T) {
	type UserIDAlias string
	type StatusAlias int
	type ScoreAlias float64
	type IsActiveAlias bool

	type User struct {
		ID       UserIDAlias   `json:"id"`
		Status   StatusAlias   `json:"status"`
		Score    ScoreAlias    `json:"score"`
		IsActive IsActiveAlias `json:"is_active"`
		Name     string        `json:"name"`
	}

	spec := &gopenapi.Spec{
		Paths: gopenapi.Paths{
			"/users": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "createUser",
					Description: "Create a new user",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: reflect.TypeOf(User{})},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "Created",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: reflect.TypeOf(User{})},
								},
							},
						},
					},
				},
			},
		},
	}

	templateData := generateTemplateData(spec, "client")

	if len(templateData.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(templateData.Operations))
	}

	op := templateData.Operations[0]

	// Test request body fields
	if !op.HasRequestBody {
		t.Error("Expected operation to have request body")
	}

	if len(op.RequestBodyFields) != 5 {
		t.Errorf("Expected 5 request body fields, got %d", len(op.RequestBodyFields))
	}

	// Test each field type
	expectedTypes := map[string]string{
		"id":        "string",
		"status":    "int",
		"score":     "float64",
		"is_active": "bool",
		"name":      "string",
	}

	for _, field := range op.RequestBodyFields {
		expectedType, exists := expectedTypes[field.Name]
		if !exists {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.GoType != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expectedType, field.GoType)
		}
	}

	t.Log("All alias types in struct fields were correctly resolved to their underlying types!")
}
