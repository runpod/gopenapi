package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runpod/gopenapi"
	"github.com/runpod/gopenapi/cmd/gopenapi/generator"
	"github.com/runpod/gopenapi/cmd/gopenapi/parser"
)

// Test spec for integration testing
var integrationTestSpec = gopenapi.Spec{
	OpenAPI: "3.0.0",
	Info: gopenapi.Info{
		Title:       "Integration Test API",
		Description: "API for integration testing",
		Version:     "1.0.0",
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

// TestIntegrationGenerateSpec tests the full spec generation pipeline
func TestIntegrationGenerateSpec(t *testing.T) {
	// Test that we can convert a spec to OpenAPI JSON
	jsonData, err := parser.SpecToOpenAPIJSON(&integrationTestSpec)
	if err != nil {
		t.Fatalf("Failed to convert spec to OpenAPI JSON: %v", err)
	}

	// Verify the JSON contains expected content
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "Integration Test API") {
		t.Error("Generated JSON should contain the API title")
	}

	if !strings.Contains(jsonStr, "getUserById") {
		t.Error("Generated JSON should contain the operation ID")
	}

	if !strings.Contains(jsonStr, "/users/{id}") {
		t.Error("Generated JSON should contain the path")
	}
}

// TestIntegrationGenerateClient tests the full client generation pipeline
func TestIntegrationGenerateClient(t *testing.T) {
	tempDir := t.TempDir()

	// Test Go client generation
	err := generator.GenerateClientForLanguage(&integrationTestSpec, "go", tempDir, "testclient")
	if err != nil {
		t.Fatalf("Failed to generate Go client: %v", err)
	}

	// Check that the Go client file was created
	goClientPath := filepath.Join(tempDir, "client.go")
	if _, err := os.Stat(goClientPath); os.IsNotExist(err) {
		t.Error("Go client file was not created")
	}

	// Read and verify the Go client content
	content, err := os.ReadFile(goClientPath)
	if err != nil {
		t.Fatalf("Failed to read generated Go client: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package testclient") {
		t.Error("Go client should contain package declaration")
	}

	if !strings.Contains(contentStr, "func (c *Client) GetUserById") {
		t.Error("Go client should contain the GetUserById method")
	}

	if !strings.Contains(contentStr, "type Error struct") {
		t.Error("Go client should contain Error type definition")
	}
}

// TestIntegrationGenerateClientToWriter tests generating client to a writer
func TestIntegrationGenerateClientToWriter(t *testing.T) {
	var buf bytes.Buffer

	err := generator.GenerateClientToWriter(&integrationTestSpec, &buf, "testclient", "templates/go.tpl", "go")
	if err != nil {
		t.Fatalf("Failed to generate client to writer: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Generated output should not be empty")
	}

	if !strings.Contains(output, "package testclient") {
		t.Error("Generated output should contain package declaration")
	}

	if !strings.Contains(output, "GetUserById") {
		t.Error("Generated output should contain the operation method")
	}
}

// TestIntegrationMultipleLanguages tests generating clients for multiple languages
func TestIntegrationMultipleLanguages(t *testing.T) {
	tempDir := t.TempDir()

	languages := []string{"go", "python", "typescript"}
	expectedFiles := []string{"client.go", "client.py", "client.ts"}

	for i, lang := range languages {
		err := generator.GenerateClientForLanguage(&integrationTestSpec, lang, tempDir, "testclient")
		if err != nil {
			t.Fatalf("Failed to generate %s client: %v", lang, err)
		}

		// Check that the expected file was created
		expectedPath := filepath.Join(tempDir, expectedFiles[i])
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created for language %s", expectedFiles[i], lang)
		}
	}
}

// TestIntegrationErrorHandling tests that error handling is properly integrated
func TestIntegrationErrorHandling(t *testing.T) {
	// Test with an invalid template file
	var buf bytes.Buffer
	err := generator.GenerateClientToWriter(&integrationTestSpec, &buf, "testclient", "templates/nonexistent.tpl", "go")
	if err == nil {
		t.Error("Expected error when using nonexistent template file")
	}

	// Test with unsupported language
	tempDir := t.TempDir()
	err = generator.GenerateClientForLanguage(&integrationTestSpec, "unsupported", tempDir, "testclient")
	if err == nil {
		t.Error("Expected error when using unsupported language")
	}
}

// TestIntegrationComplexSpec tests with a more complex specification
func TestIntegrationComplexSpec(t *testing.T) {
	complexSpec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Complex API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/users": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "listUsers",
					Parameters: gopenapi.Parameters{
						{
							Name:   "limit",
							In:     gopenapi.InQuery,
							Schema: gopenapi.Schema{Type: gopenapi.Integer},
						},
						{
							Name:   "offset",
							In:     gopenapi.InQuery,
							Schema: gopenapi.Schema{Type: gopenapi.Integer},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Array},
								},
							},
						},
					},
				},
				Post: &gopenapi.Operation{
					OperationId: "createUser",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									Name  string `json:"name"`
									Email string `json:"email"`
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "Created",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										ID int `json:"id"`
									}]()},
								},
							},
						},
					},
				},
			},
			"/users/{id}": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "getUserById",
					Parameters: gopenapi.Parameters{
						{
							Name:     "id",
							In:       gopenapi.InPath,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.Integer},
						},
						{
							Name:   "X-Request-ID",
							In:     gopenapi.InHeader,
							Schema: gopenapi.Schema{Type: gopenapi.String},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										ID   int    `json:"id"`
										Name string `json:"name"`
									}]()},
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := generator.GenerateClientToWriter(&complexSpec, &buf, "complexclient", "templates/go.tpl", "go")
	if err != nil {
		t.Fatalf("Failed to generate complex client: %v", err)
	}

	output := buf.String()

	// Check that all operations are generated
	expectedMethods := []string{"ListUsers", "CreateUser", "GetUserById"}
	for _, method := range expectedMethods {
		if !strings.Contains(output, "func (c *Client) "+method) {
			t.Errorf("Generated client should contain method %s", method)
		}
	}

	// Check that parameter handling is generated
	if !strings.Contains(output, "opts.Query.Limit") {
		t.Error("Generated client should handle query parameters")
	}

	if !strings.Contains(output, "opts.Path.Id") {
		t.Error("Generated client should handle path parameters")
	}

	if !strings.Contains(output, "opts.Headers.XRequestID") {
		t.Error("Generated client should handle header parameters")
	}

	// Check that request body handling is generated
	if !strings.Contains(output, "CreateUserRequestBody") {
		t.Error("Generated client should contain request body struct")
	}

	// Check that response handling is generated
	if !strings.Contains(output, "GetUserByIdResponse") {
		t.Error("Generated client should contain response struct")
	}
}
