package parser

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/runpod/gopenapi"
	"github.com/runpod/gopenapi/cmd/gopenapi/parser/internal/company"
	"github.com/runpod/gopenapi/cmd/gopenapi/parser/internal/mock"
)

// Test that we can resolve types from other packages
func TestCrossPackageTypeResolution(t *testing.T) {
	// Create a spec that uses types from the standard library and other packages
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Cross Package Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/events": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "createEvent",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									Name      string        `json:"name"`
									Timestamp time.Time     `json:"timestamp"` // Type from time package
									Duration  time.Duration `json:"duration"`  // Another type from time package
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "Event created",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										ID        string    `json:"id"`
										CreatedAt time.Time `json:"created_at"`
									}]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the request body schema
	paths, ok := result["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("Paths should be an object")
	}

	eventsPath, ok := paths["/events"].(map[string]interface{})
	if !ok {
		t.Fatal("Events path should exist")
	}

	postOp, ok := eventsPath["post"].(map[string]interface{})
	if !ok {
		t.Fatal("POST operation should exist")
	}

	requestBody, ok := postOp["requestBody"].(map[string]interface{})
	if !ok {
		t.Fatal("Request body should exist")
	}

	content, ok := requestBody["content"].(map[string]interface{})
	if !ok {
		t.Fatal("Content should exist")
	}

	appJson, ok := content["application/json"].(map[string]interface{})
	if !ok {
		t.Fatal("application/json content should exist")
	}

	schema, ok := appJson["schema"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema should exist")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should exist")
	}

	// Test that time.Time is properly resolved
	timestampProp, ok := properties["timestamp"].(map[string]interface{})
	if !ok {
		t.Fatal("Timestamp property should exist")
	}

	// time.Time should be resolved to a string type in OpenAPI
	timestampType, ok := timestampProp["type"].(string)
	if !ok {
		t.Fatal("Timestamp type should be a string")
	}

	// time.Time should be mapped to string in OpenAPI
	if timestampType != "string" {
		t.Errorf("Expected timestamp type to be 'string', got %s", timestampType)
	}

	// Test that time.Duration is properly resolved
	durationProp, ok := properties["duration"].(map[string]interface{})
	if !ok {
		t.Fatal("Duration property should exist")
	}

	durationType, ok := durationProp["type"].(string)
	if !ok {
		t.Fatal("Duration type should be a string")
	}

	// time.Duration should be mapped to integer (nanoseconds) or string in OpenAPI
	if durationType != "integer" && durationType != "string" {
		t.Errorf("Expected duration type to be 'integer' or 'string', got %s", durationType)
	}

	t.Logf("Successfully resolved cross-package types: timestamp=%s, duration=%s", timestampType, durationType)
}

// Test with more complex nested types from other packages
func TestComplexCrossPackageTypes(t *testing.T) {
	// Test with types that have complex internal structure
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Complex Cross Package Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/complex": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "createComplex",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									// Test with various standard library types
									URL     string        `json:"url"`     // Basic type
									Timeout time.Duration `json:"timeout"` // time package
									When    time.Time     `json:"when"`    // time package
									// Could add more complex types here like net.IP, url.URL, etc.
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
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

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// The test passes if we can generate valid JSON without panicking
	// The actual type resolution will be tested by the previous test
	t.Log("Successfully generated OpenAPI JSON for complex cross-package types")
}

func TestSpecToOpenAPIJSON(t *testing.T) {
	// Test spec for JSON conversion
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:       "Test API",
			Description: "A test API for JSON conversion",
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
					Summary:     "Get user by ID",
					Description: "Retrieve a user by their ID",
					Parameters: gopenapi.Parameters{
						{
							Name:        "id",
							In:          gopenapi.InPath,
							Description: "User ID",
							Required:    true,
							Schema:      gopenapi.Schema{Type: gopenapi.Integer},
						},
						{
							Name:        "include",
							In:          gopenapi.InQuery,
							Description: "Include additional data",
							Required:    false,
							Schema:      gopenapi.Schema{Type: gopenapi.String},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "User found",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										ID   int    `json:"id"`
										Name string `json:"name"`
									}]()},
								},
							},
						},
						404: {
							Description: "User not found",
						},
					},
				},
			},
			"/users": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "createUser",
					Summary:     "Create user",
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
							Description: "User created",
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
		},
	}

	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Test basic structure
	if result["openapi"] != "3.0.0" {
		t.Errorf("Expected openapi version 3.0.0, got %v", result["openapi"])
	}

	// Test info section
	info, ok := result["info"].(map[string]interface{})
	if !ok {
		t.Fatal("Info section should be an object")
	}
	if info["title"] != "Test API" {
		t.Errorf("Expected title 'Test API', got %v", info["title"])
	}
	if info["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", info["version"])
	}

	// Test servers section
	servers, ok := result["servers"].([]interface{})
	if !ok {
		t.Fatal("Servers section should be an array")
	}
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}

	// Test paths section
	paths, ok := result["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("Paths section should be an object")
	}

	// Test specific path
	userPath, ok := paths["/users/{id}"].(map[string]interface{})
	if !ok {
		t.Fatal("User path should exist")
	}

	// Test GET operation
	getOp, ok := userPath["get"].(map[string]interface{})
	if !ok {
		t.Fatal("GET operation should exist")
	}
	if getOp["operationId"] != "getUserById" {
		t.Errorf("Expected operationId 'getUserById', got %v", getOp["operationId"])
	}

	// Test parameters
	params, ok := getOp["parameters"].([]interface{})
	if !ok {
		t.Fatal("Parameters should be an array")
	}
	if len(params) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(params))
	}

	// Test path parameter
	pathParam, ok := params[0].(map[string]interface{})
	if !ok {
		t.Fatal("First parameter should be an object")
	}
	if pathParam["name"] != "id" {
		t.Errorf("Expected parameter name 'id', got %v", pathParam["name"])
	}
	if pathParam["in"] != "path" {
		t.Errorf("Expected parameter in 'path', got %v", pathParam["in"])
	}
	if pathParam["required"] != true {
		t.Errorf("Expected parameter required true, got %v", pathParam["required"])
	}

	// Test responses
	responses, ok := getOp["responses"].(map[string]interface{})
	if !ok {
		t.Fatal("Responses should be an object")
	}
	if _, exists := responses["200"]; !exists {
		t.Error("200 response should exist")
	}
	if _, exists := responses["404"]; !exists {
		t.Error("404 response should exist")
	}
}

func TestParameterLocationToString(t *testing.T) {
	tests := []struct {
		name     string
		location gopenapi.In
		expected string
	}{
		{
			name:     "Path parameter",
			location: gopenapi.InPath,
			expected: "path",
		},
		{
			name:     "Query parameter",
			location: gopenapi.InQuery,
			expected: "query",
		},
		{
			name:     "Header parameter",
			location: gopenapi.InHeader,
			expected: "header",
		},
		{
			name:     "Cookie parameter",
			location: gopenapi.InCookie,
			expected: "cookie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parameterLocationToString(tt.location)
			if result != tt.expected {
				t.Errorf("parameterLocationToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGoTypeToOpenAPIType(t *testing.T) {
	tests := []struct {
		name     string
		goType   reflect.Type
		expected string
	}{
		{
			name:     "String type",
			goType:   reflect.TypeOf(""),
			expected: "string",
		},
		{
			name:     "Int type",
			goType:   reflect.TypeOf(0),
			expected: "integer",
		},
		{
			name:     "Float64 type",
			goType:   reflect.TypeOf(0.0),
			expected: "number",
		},
		{
			name:     "Bool type",
			goType:   reflect.TypeOf(false),
			expected: "boolean",
		},
		{
			name:     "Slice type",
			goType:   reflect.TypeOf([]string{}),
			expected: "array",
		},
		{
			name:     "Struct type",
			goType:   reflect.TypeOf(struct{}{}),
			expected: "object",
		},
		{
			name:     "Pointer type",
			goType:   reflect.TypeOf((*string)(nil)),
			expected: "string",
		},
		{
			name:     "time.Time type",
			goType:   reflect.TypeOf(time.Time{}),
			expected: "string",
		},
		{
			name:     "time.Duration type",
			goType:   reflect.TypeOf(time.Duration(0)),
			expected: "integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goTypeToOpenAPIType(tt.goType)
			if result != tt.expected {
				t.Errorf("goTypeToOpenAPIType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test with types from multiple different packages to ensure generic resolution
func TestMultiplePackageTypeResolution(t *testing.T) {
	// Create a spec that uses types from various packages
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Multi Package Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/test": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "testMultiplePackages",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									// Standard library types
									Timestamp time.Time     `json:"timestamp"`
									Duration  time.Duration `json:"duration"`

									// Basic types for comparison
									Name   string `json:"name"`
									Count  int    `json:"count"`
									Active bool   `json:"active"`
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										Success   bool      `json:"success"`
										CreatedAt time.Time `json:"created_at"`
									}]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the request body schema properties
	paths := result["paths"].(map[string]interface{})
	testPath := paths["/test"].(map[string]interface{})
	postOp := testPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})

	// Test that all types are correctly resolved
	expectedTypes := map[string]string{
		"timestamp": "string",  // time.Time should be string
		"duration":  "integer", // time.Duration should be integer
		"name":      "string",  // Basic string
		"count":     "integer", // Basic int
		"active":    "boolean", // Basic bool
	}

	for fieldName, expectedType := range expectedTypes {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Property %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Property %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Property %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	// Test response types as well
	responses := postOp["responses"].(map[string]interface{})
	response200 := responses["200"].(map[string]interface{})
	responseContent := response200["content"].(map[string]interface{})
	responseAppJson := responseContent["application/json"].(map[string]interface{})
	responseSchema := responseAppJson["schema"].(map[string]interface{})
	responseProperties := responseSchema["properties"].(map[string]interface{})

	// Check response field types
	successProp := responseProperties["success"].(map[string]interface{})
	if successProp["type"] != "boolean" {
		t.Errorf("Response success field should be boolean, got %s", successProp["type"])
	}

	createdAtProp := responseProperties["created_at"].(map[string]interface{})
	if createdAtProp["type"] != "string" {
		t.Errorf("Response created_at field should be string (time.Time), got %s", createdAtProp["type"])
	}

	t.Log("Successfully resolved types from multiple packages without hardcoding!")
}

// Test with mock package types to ensure cross-package type resolution
func TestMockPackageTypeResolution(t *testing.T) {
	// Create a spec that uses types from our mock package
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Mock Package Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/users": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "createUser",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[mock.User]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "User created",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										ID      mock.UserID `json:"id"`
										Success bool        `json:"success"`
									}]()},
								},
							},
						},
					},
				},
			},
			"/products": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "createProduct",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[mock.Product]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "Product created",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[mock.Product]()},
								},
							},
						},
					},
				},
			},
			"/analytics": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "getAnalytics",
					Responses: gopenapi.Responses{
						200: {
							Description: "Analytics data",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[mock.Analytics]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Test User creation endpoint
	paths := result["paths"].(map[string]interface{})
	usersPath := paths["/users"].(map[string]interface{})
	postOp := usersPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})

	// Test User struct field types
	expectedUserFields := map[string]string{
		"id":         "string",  // mock.UserID (string alias)
		"name":       "string",  // string
		"email":      "string",  // string
		"age":        "integer", // int
		"is_active":  "boolean", // mock.IsActive (bool alias)
		"created_at": "string",  // time.Time
		"tags":       "array",   // mock.Tags ([]string alias)
	}

	for fieldName, expectedType := range expectedUserFields {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("User field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("User field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("User field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	// Test Product creation endpoint
	productsPath := paths["/products"].(map[string]interface{})
	productPostOp := productsPath["post"].(map[string]interface{})
	productRequestBody := productPostOp["requestBody"].(map[string]interface{})
	productContent := productRequestBody["content"].(map[string]interface{})
	productAppJson := productContent["application/json"].(map[string]interface{})
	productSchema := productAppJson["schema"].(map[string]interface{})
	productProperties := productSchema["properties"].(map[string]interface{})

	// Test Product struct field types
	expectedProductFields := map[string]string{
		"id":           "integer", // mock.ProductID (int64 alias)
		"name":         "string",  // string
		"price":        "number",  // mock.Price (float64 alias)
		"in_stock":     "boolean", // bool
		"last_updated": "string",  // time.Time
		"duration":     "integer", // time.Duration
		"scores":       "array",   // mock.Scores ([]float64 alias)
	}

	for fieldName, expectedType := range expectedProductFields {
		prop, exists := productProperties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Product field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Product field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Product field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	// Test Analytics endpoint (complex nested structure)
	analyticsPath := paths["/analytics"].(map[string]interface{})
	analyticsGetOp := analyticsPath["get"].(map[string]interface{})
	analyticsResponses := analyticsGetOp["responses"].(map[string]interface{})
	analytics200 := analyticsResponses["200"].(map[string]interface{})
	analyticsContent := analytics200["content"].(map[string]interface{})
	analyticsAppJson := analyticsContent["application/json"].(map[string]interface{})
	analyticsSchema := analyticsAppJson["schema"].(map[string]interface{})
	analyticsProperties := analyticsSchema["properties"].(map[string]interface{})

	// Test that nested structures are properly resolved as objects
	expectedAnalyticsFields := map[string]string{
		"user_metrics":    "object", // mock.UserMetrics
		"product_metrics": "object", // mock.ProductMetrics
		"time_range":      "object", // mock.TimeRange
	}

	for fieldName, expectedType := range expectedAnalyticsFields {
		prop, exists := analyticsProperties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Analytics field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Analytics field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Analytics field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	t.Log("Successfully resolved all mock package types!")
}

// Test with basic alias types from mock package
func TestMockBasicAliasTypes(t *testing.T) {
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Mock Alias Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/test-aliases": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "testAliases",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									UserID    mock.UserID    `json:"user_id"`
									ProductID mock.ProductID `json:"product_id"`
									Price     mock.Price     `json:"price"`
									IsActive  mock.IsActive  `json:"is_active"`
									Tags      mock.Tags      `json:"tags"`
									Scores    mock.Scores    `json:"scores"`
									Status    mock.Status    `json:"status"`
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
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

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to properties
	paths := result["paths"].(map[string]interface{})
	testPath := paths["/test-aliases"].(map[string]interface{})
	postOp := testPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})

	// Test all alias types
	expectedTypes := map[string]string{
		"user_id":    "string",  // mock.UserID (string alias)
		"product_id": "integer", // mock.ProductID (int64 alias)
		"price":      "number",  // mock.Price (float64 alias)
		"is_active":  "boolean", // mock.IsActive (bool alias)
		"tags":       "array",   // mock.Tags ([]string alias)
		"scores":     "array",   // mock.Scores ([]float64 alias)
		"status":     "string",  // mock.Status (string alias)
	}

	for fieldName, expectedType := range expectedTypes {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	t.Log("Successfully resolved all mock alias types!")
}

// Test with nested slice aliases
func TestMockNestedSliceAliases(t *testing.T) {
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Mock Nested Slice Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/test-slices": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "testSlices",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									UserIDs  mock.UserIDs   `json:"user_ids"` // []mock.UserID (slice of string alias)
									Products []mock.Product `json:"products"` // []mock.Product (slice of struct)
									Orders   []mock.Order   `json:"orders"`   // []mock.Order (slice of complex struct)
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
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

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to properties
	paths := result["paths"].(map[string]interface{})
	testPath := paths["/test-slices"].(map[string]interface{})
	postOp := testPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})

	// Test all slice types should be arrays
	expectedTypes := map[string]string{
		"user_ids": "array", // mock.UserIDs ([]mock.UserID)
		"products": "array", // []mock.Product
		"orders":   "array", // []mock.Order
	}

	for fieldName, expectedType := range expectedTypes {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	t.Log("Successfully resolved all mock nested slice types!")
}

// Test edge cases that could cause nil pointer panics
func TestNilPointerEdgeCases(t *testing.T) {
	// This test ensures we handle edge cases that could cause nil pointer panics
	// when dealing with types that don't have package information

	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Nil Pointer Edge Cases Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/edge-cases": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "testEdgeCases",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									// Basic types that shouldn't cause issues
									Name   string `json:"name"`
									Count  int    `json:"count"`
									Active bool   `json:"active"`

									// Types from standard library
									Created time.Time     `json:"created"`
									Timeout time.Duration `json:"timeout"`

									// Mock package types
									UserID    mock.UserID    `json:"user_id"`
									ProductID mock.ProductID `json:"product_id"`
									Price     mock.Price     `json:"price"`

									// Nested structures
									User    mock.User    `json:"user"`
									Product mock.Product `json:"product"`
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
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

	// This should not panic - if it does, we have a nil pointer issue
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() should not error, got: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON should be valid, got error: %v", err)
	}

	// Basic validation that the structure is correct
	paths, ok := result["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("Paths should be an object")
	}

	edgeCasesPath, ok := paths["/edge-cases"].(map[string]interface{})
	if !ok {
		t.Fatal("Edge cases path should exist")
	}

	postOp, ok := edgeCasesPath["post"].(map[string]interface{})
	if !ok {
		t.Fatal("POST operation should exist")
	}

	if postOp["operationId"] != "testEdgeCases" {
		t.Errorf("Expected operationId 'testEdgeCases', got %v", postOp["operationId"])
	}

	t.Log("Successfully handled all edge cases without nil pointer panics!")
}

// Test with various problematic type scenarios
func TestProblematicTypeScenarios(t *testing.T) {
	// Test scenarios that might cause issues in type resolution

	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Problematic Type Scenarios Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/problematic": gopenapi.Path{
				Post: &gopenapi.Operation{
					OperationId: "testProblematicTypes",
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
									// Pointer types
									OptionalString *string      `json:"optional_string,omitempty"`
									OptionalInt    *int         `json:"optional_int,omitempty"`
									OptionalTime   *time.Time   `json:"optional_time,omitempty"`
									OptionalUser   *mock.User   `json:"optional_user,omitempty"`
									OptionalUserID *mock.UserID `json:"optional_user_id,omitempty"`

									// Slice types
									StringSlice []string      `json:"string_slice"`
									IntSlice    []int         `json:"int_slice"`
									TimeSlice   []time.Time   `json:"time_slice"`
									UserSlice   []mock.User   `json:"user_slice"`
									UserIDSlice []mock.UserID `json:"user_id_slice"`

									// Map types (these might be tricky)
									StringMap map[string]string      `json:"string_map"`
									UserMap   map[string]mock.User   `json:"user_map"`
									UserIDMap map[mock.UserID]string `json:"user_id_map"`

									// Interface types
									AnyValue interface{} `json:"any_value"`

									// Nested complex types
									Analytics     mock.Analytics    `json:"analytics"`
									OptionalUser2 mock.OptionalUser `json:"optional_user2"`
								}]()},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[struct {
										Success bool `json:"success"`
									}]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// This should not panic regardless of the complex types
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() should not error with complex types, got: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON should be valid with complex types, got error: %v", err)
	}

	// Navigate to the request body to verify some types were processed
	paths := result["paths"].(map[string]interface{})
	problematicPath := paths["/problematic"].(map[string]interface{})
	postOp := problematicPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})

	// Verify some key types are correctly resolved
	expectedTypes := map[string]string{
		"optional_string": "string", // *string should be string
		"string_slice":    "array",  // []string should be array
		"user_id_slice":   "array",  // []mock.UserID should be array
		"analytics":       "object", // mock.Analytics should be object
		"any_value":       "object", // interface{} should be object
	}

	for fieldName, expectedType := range expectedTypes {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	t.Log("Successfully handled all problematic type scenarios!")
}

// Test that reproduces the exact issue where nested struct field types are not resolved
func TestNestedStructFieldTypeResolution(t *testing.T) {
	// This test reproduces the issue where DeviceSnapshot fields are resolved as interface{}
	// instead of their actual types when the struct comes from an external package

	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Nested Struct Field Resolution Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/devices/{id}": gopenapi.Path{
				Get: &gopenapi.Operation{
					Summary:     "Get device by ID",
					OperationId: "getDeviceById",
					Parameters: gopenapi.Parameters{
						{
							Name:     "id",
							In:       gopenapi.InPath,
							Required: true,
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Success",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{
										Type: gopenapi.Object[mock.DeviceSnapshot](),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the response schema properties
	paths := result["paths"].(map[string]interface{})
	devicesPath := paths["/devices/{id}"].(map[string]interface{})
	getOp := devicesPath["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	response200 := responses["200"].(map[string]interface{})
	content := response200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})

	properties, exists := schema["properties"].(map[string]interface{})
	if !exists {
		t.Fatal("Schema should have properties")
	}

	// Test that DeviceSnapshot field types are correctly resolved
	// These should NOT be interface{} but their actual types
	expectedTypes := map[string]string{
		"ID":          "integer", // mock.ID (uint64 alias)
		"Index":       "integer", // mock.Index (uint64 alias)
		"Kind":        "string",  // mock.Kind (string alias)
		"Memory":      "object",  // mock.Memory (struct)
		"Processes":   "object",  // mock.ProcessMap (map)
		"Errors":      "array",   // []error
		"Temperature": "number",  // float64
	}

	for fieldName, expectedType := range expectedTypes {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("DeviceSnapshot field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("DeviceSnapshot field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("DeviceSnapshot field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	// Also check that Memory struct has its own properties resolved
	memoryProp := properties["Memory"].(map[string]interface{})
	if memoryProp["type"] == "object" {
		if memoryProps, exists := memoryProp["properties"].(map[string]interface{}); exists {
			expectedMemoryTypes := map[string]string{
				"total":     "integer", // uint64
				"used":      "integer", // uint64
				"available": "integer", // uint64
			}

			for fieldName, expectedType := range expectedMemoryTypes {
				if prop, exists := memoryProps[fieldName].(map[string]interface{}); exists {
					if actualType := prop["type"].(string); actualType != expectedType {
						t.Errorf("Memory field %s: expected type %s, got %s", fieldName, expectedType, actualType)
					}
				} else {
					t.Errorf("Memory field %s should exist", fieldName)
				}
			}
		} else {
			t.Error("Memory object should have properties")
		}
	}

	t.Log("Successfully resolved all nested struct field types!")
}

// Test that reproduces the issue with deeply nested external package types
func TestDeeplyNestedExternalTypes(t *testing.T) {
	// Test with a complex structure that has multiple levels of nesting
	// from external packages to ensure all levels are resolved correctly

	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Deeply Nested External Types Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/analytics": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "getAnalytics",
					Responses: gopenapi.Responses{
						200: {
							Description: "Analytics data",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{
										Type: gopenapi.Object[mock.Analytics](),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the response schema
	paths := result["paths"].(map[string]interface{})
	analyticsPath := paths["/analytics"].(map[string]interface{})
	getOp := analyticsPath["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	response200 := responses["200"].(map[string]interface{})
	content := response200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})

	// Test top-level Analytics fields
	expectedAnalyticsTypes := map[string]string{
		"user_metrics":    "object", // mock.UserMetrics
		"product_metrics": "object", // mock.ProductMetrics
		"time_range":      "object", // mock.TimeRange
	}

	for fieldName, expectedType := range expectedAnalyticsTypes {
		prop, exists := properties[fieldName].(map[string]interface{})
		if !exists {
			t.Errorf("Analytics field %s should exist", fieldName)
			continue
		}

		actualType, exists := prop["type"].(string)
		if !exists {
			t.Errorf("Analytics field %s should have a type", fieldName)
			continue
		}

		if actualType != expectedType {
			t.Errorf("Analytics field %s: expected type %s, got %s", fieldName, expectedType, actualType)
		}
	}

	// Test nested UserMetrics fields
	userMetricsProp := properties["user_metrics"].(map[string]interface{})
	if userMetricsProp["type"] == "object" {
		if userMetricsProps, exists := userMetricsProp["properties"].(map[string]interface{}); exists {
			expectedUserMetricsTypes := map[string]string{
				"total_users":  "integer", // int64
				"active_users": "integer", // int64
				"new_users":    "integer", // int64
				"user_ids":     "array",   // mock.UserIDs ([]mock.UserID)
				"last_updated": "string",  // time.Time
			}

			for fieldName, expectedType := range expectedUserMetricsTypes {
				if prop, exists := userMetricsProps[fieldName].(map[string]interface{}); exists {
					if actualType := prop["type"].(string); actualType != expectedType {
						t.Errorf("UserMetrics field %s: expected type %s, got %s", fieldName, expectedType, actualType)
					}
				} else {
					t.Errorf("UserMetrics field %s should exist", fieldName)
				}
			}
		} else {
			t.Error("UserMetrics object should have properties")
		}
	}

	t.Log("Successfully resolved all deeply nested external types!")
}

// Test recursive type resolution for nested structs
func TestRecursiveStructTypeResolution(t *testing.T) {
	// Create test types that mimic the device response structure
	type Bytes int64

	type Snapshot struct {
		Total Bytes `json:"total"`
		Used  Bytes `json:"used"`
		Free  Bytes `json:"free"`
	}

	type Process struct {
		PID    int    `json:"pid"`
		Name   string `json:"name"`
		Memory Bytes  `json:"memory"`
	}

	type DeviceResponse struct {
		ID          uint      `json:"ID"`
		Index       uint      `json:"Index"`
		Kind        string    `json:"Kind"`
		Memory      Snapshot  `json:"Memory"`    // Nested struct
		Processes   []Process `json:"Processes"` // Slice of structs
		Errors      []string  `json:"Errors"`
		Temperature float64   `json:"Temperature"`
	}

	// Create a spec using the nested types
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Device API with Nested Types",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/device/{id}": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "getDeviceById",
					Parameters: gopenapi.Parameters{
						{
							Name:     "id",
							In:       gopenapi.InPath,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.String},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "Device details",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[DeviceResponse]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Parse the JSON to verify structure
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the response schema
	paths := result["paths"].(map[string]interface{})
	devicePath := paths["/device/{id}"].(map[string]interface{})
	getOp := devicePath["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	content := resp200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})

	// Verify the main object
	if schema["type"] != "object" {
		t.Errorf("Expected root schema type to be 'object', got %v", schema["type"])
	}

	properties := schema["properties"].(map[string]interface{})

	// Check Memory field is properly resolved as an object with nested properties
	memoryProp := properties["Memory"].(map[string]interface{})
	if memoryProp["type"] != "object" {
		t.Errorf("Expected Memory type to be 'object', got %v", memoryProp["type"])
	}

	// Verify Memory has nested properties
	memoryProps, ok := memoryProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Memory should have properties")
	}

	// Check nested fields in Memory
	totalProp := memoryProps["total"].(map[string]interface{})
	if totalProp["type"] != "integer" {
		t.Errorf("Expected Memory.total type to be 'integer', got %v", totalProp["type"])
	}

	usedProp := memoryProps["used"].(map[string]interface{})
	if usedProp["type"] != "integer" {
		t.Errorf("Expected Memory.used type to be 'integer', got %v", usedProp["type"])
	}

	freeProp := memoryProps["free"].(map[string]interface{})
	if freeProp["type"] != "integer" {
		t.Errorf("Expected Memory.free type to be 'integer', got %v", freeProp["type"])
	}

	// Check Processes field is an array
	processesProp := properties["Processes"].(map[string]interface{})
	if processesProp["type"] != "array" {
		t.Errorf("Expected Processes type to be 'array', got %v", processesProp["type"])
	}

	// Check other fields
	if properties["ID"].(map[string]interface{})["type"] != "integer" {
		t.Error("ID should be integer")
	}
	if properties["Temperature"].(map[string]interface{})["type"] != "number" {
		t.Error("Temperature should be number")
	}

	t.Log("Successfully resolved nested struct types recursively")
}

// Test even deeper nesting with struct-in-struct-in-struct
func TestDeeplyNestedStructResolution(t *testing.T) {
	type Level3 struct {
		Value string `json:"value"`
	}

	type Level2 struct {
		Name string `json:"name"`
		Deep Level3 `json:"deep"`
	}

	type Level1 struct {
		ID     int    `json:"id"`
		Nested Level2 `json:"nested"`
	}

	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Deeply Nested API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/nested": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "getDeeplyNested",
					Responses: gopenapi.Responses{
						200: {
							Description: "Deeply nested response",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[Level1]()},
								},
							},
						},
					},
				},
			},
		},
	}

	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Print the JSON for debugging
	t.Logf("Generated JSON:\n%s", string(jsonData))

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the nested properties
	paths := result["paths"].(map[string]interface{})
	nestedPath := paths["/nested"].(map[string]interface{})
	getOp := nestedPath["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	content := resp200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})

	// Level 1 properties
	level1Props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Level 1 should have properties")
	}

	nestedProp, ok := level1Props["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("Level 1 should have 'nested' property")
	}

	// Level 2 properties
	level2Props, ok := nestedProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Level 2 (nested) should have properties")
	}

	deepProp, ok := level2Props["deep"].(map[string]interface{})
	if !ok {
		t.Fatal("Level 2 should have 'deep' property")
	}

	// Level 3 properties
	level3Props, ok := deepProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Level 3 (deep) should have properties")
	}

	valueProp, ok := level3Props["value"].(map[string]interface{})
	if !ok {
		t.Fatal("Level 3 should have 'value' property")
	}

	if valueProp["type"] != "string" {
		t.Errorf("Expected deeply nested value type to be 'string', got %v", valueProp["type"])
	}

	t.Log("Successfully resolved deeply nested struct types (3 levels deep)")
}

// Test deeply nested cross-package type resolution
func TestDeeplyNestedCrossPackageTypes(t *testing.T) {
	// This test uses the company.CompanyReport type which has deep nesting across multiple packages:
	// CompanyReport -> Company -> Department -> Team -> Project -> Milestone
	// CompanyReport -> Employee -> ContactInfo -> Address
	// CompanyReport -> Employee -> ContactInfo -> EmergencyContact
	// CompanyReport -> Employee -> Skills -> Competency -> Language

	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Deeply Nested Cross-Package API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/company-report": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "getCompanyReport",
					Responses: gopenapi.Responses{
						200: {
							Description: "Company report with deeply nested types",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[company.CompanyReport]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	// Print the JSON for debugging
	t.Logf("Generated JSON (first 2000 chars):\n%s", string(jsonData[:min(len(jsonData), 2000)]))

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Navigate to the response schema
	paths := result["paths"].(map[string]interface{})
	reportPath := paths["/company-report"].(map[string]interface{})
	getOp := reportPath["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	content := resp200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schema := appJson["schema"].(map[string]interface{})

	// Check CompanyReport properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("CompanyReport should have properties")
	}

	// Check that company field exists and is an object
	companyProp, ok := properties["company"].(map[string]interface{})
	if !ok {
		t.Fatal("CompanyReport should have 'company' property")
	}
	if companyProp["type"] != "object" {
		t.Errorf("Expected company type to be 'object', got %v", companyProp["type"])
	}

	// Check company has nested properties
	companyProps, ok := companyProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Company should have properties")
	}

	// Check departments array
	departmentsProp, ok := companyProps["departments"].(map[string]interface{})
	if !ok {
		t.Fatal("Company should have 'departments' property")
	}
	if departmentsProp["type"] != "array" {
		t.Errorf("Expected departments type to be 'array', got %v", departmentsProp["type"])
	}

	// Check CEO (Employee type) has deep nesting
	ceoProp, ok := companyProps["ceo"].(map[string]interface{})
	if !ok {
		t.Fatal("Company should have 'ceo' property")
	}
	if ceoProp["type"] != "object" {
		t.Errorf("Expected ceo type to be 'object', got %v", ceoProp["type"])
	}

	// Navigate deep into CEO -> contact -> address
	ceoProps, ok := ceoProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("CEO should have properties")
	}

	contactProp, ok := ceoProps["contact"].(map[string]interface{})
	if !ok {
		t.Fatal("CEO should have 'contact' property")
	}

	contactProps, ok := contactProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Contact should have properties")
	}

	addressProp, ok := contactProps["address"].(map[string]interface{})
	if !ok {
		t.Fatal("Contact should have 'address' property")
	}

	addressProps, ok := addressProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Address should have properties")
	}

	// Verify address fields are resolved
	streetProp, ok := addressProps["street"].(map[string]interface{})
	if !ok {
		t.Fatal("Address should have 'street' property")
	}
	if streetProp["type"] != "string" {
		t.Errorf("Expected street type to be 'string', got %v", streetProp["type"])
	}

	// Check headquarters -> coords (multiple levels of local nesting)
	headquartersProp, ok := companyProps["headquarters"].(map[string]interface{})
	if !ok {
		t.Fatal("Company should have 'headquarters' property")
	}

	hqProps, ok := headquartersProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Headquarters should have properties")
	}

	coordsProp, ok := hqProps["coords"].(map[string]interface{})
	if !ok {
		t.Fatal("Headquarters should have 'coords' property")
	}

	coordsProps, ok := coordsProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Coords should have properties")
	}

	latProp, ok := coordsProps["latitude"].(map[string]interface{})
	if !ok {
		t.Fatal("Coords should have 'latitude' property")
	}
	if latProp["type"] != "number" {
		t.Errorf("Expected latitude type to be 'number', got %v", latProp["type"])
	}

	// Test performance -> budget_health (cross-package deep nesting)
	perfProp, ok := properties["performance"].(map[string]interface{})
	if !ok {
		t.Fatal("CompanyReport should have 'performance' property")
	}

	perfProps, ok := perfProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Performance should have properties")
	}

	budgetHealthProp, ok := perfProps["budget_health"].(map[string]interface{})
	if !ok {
		t.Fatal("Performance should have 'budget_health' property")
	}

	budgetHealthProps, ok := budgetHealthProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("BudgetHealth should have properties")
	}

	projectionsProp, ok := budgetHealthProps["projections"].(map[string]interface{})
	if !ok {
		t.Fatal("BudgetHealth should have 'projections' property")
	}

	projectionsProps, ok := projectionsProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Projections should have properties")
	}

	overrunProp, ok := projectionsProps["overrun"].(map[string]interface{})
	if !ok {
		t.Fatal("Projections should have 'overrun' property")
	}
	if overrunProp["type"] != "boolean" {
		t.Errorf("Expected overrun type to be 'boolean', got %v", overrunProp["type"])
	}

	t.Log("Successfully resolved deeply nested cross-package types!")
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Test to debug type resolution issues with external types
func TestDebugExternalTypeResolution(t *testing.T) {
	// Create a simple spec with just one external type
	spec := gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Debug Test API",
			Version: "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/test": gopenapi.Path{
				Get: &gopenapi.Operation{
					OperationId: "test",
					Responses: gopenapi.Responses{
						200: {
							Description: "Test",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[mock.User]()},
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to OpenAPI JSON
	jsonData, err := SpecToOpenAPIJSON(&spec)
	if err != nil {
		t.Fatalf("SpecToOpenAPIJSON() error = %v", err)
	}

	t.Logf("Generated JSON:\n%s", string(jsonData))

	// Check the reflect.Type information
	userType := gopenapi.Object[mock.User]()
	t.Logf("User type: %v", userType)
	t.Logf("User type kind: %v", userType.Kind())
	t.Logf("User type name: %v", userType.Name())
	t.Logf("User type pkg path: %v", userType.PkgPath())

	// Check field types
	for i := 0; i < userType.NumField(); i++ {
		field := userType.Field(i)
		t.Logf("Field %s: type=%v, kind=%v, name=%v, pkgPath=%v",
			field.Name, field.Type, field.Type.Kind(), field.Type.Name(), field.Type.PkgPath())
	}
}
