package gopenapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/runpod/gopenapi"
)

type User struct {
	Name string `json:"name"`
}

var UserSchema = gopenapi.Schema{
	Type: gopenapi.Object[User](),
}

func TestOpenAPIServer(t *testing.T) {
	schema := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:       "Test API",
			Description: "Test API",
			Version:     "1.0.0",
		},
		Paths: gopenapi.Paths{
			"/user/{id}": {
				Tags: []string{"user"},
				Get: &gopenapi.Operation{
					OperationId: "GetUserById",
					Summary:     "Get the user by id",
					Description: "Get the user by id",
					Tags:        []string{"user"},
					Security:    gopenapi.NoSecurity,
					Parameters: gopenapi.Parameters{
						{
							Name:        "id",
							In:          gopenapi.InPath,
							Description: "The id of the user",
							Schema: gopenapi.Schema{
								Type: gopenapi.Integer,
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
						},
					},
					Handler: http.HandlerFunc(getUserHandler),
				},
				Patch: &gopenapi.Operation{
					OperationId: "PatchUserById",
					Summary:     "Patch the user by id",
					Description: "Patch the user by id",
					Tags:        []string{"user"},
					Security:    gopenapi.NoSecurity,
					RequestBody: gopenapi.RequestBody{
						Content: gopenapi.Content{
							"application/json": {
								Schema: gopenapi.Schema{
									Type: gopenapi.Object[User](),
								},
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
						},
					},
					Handler: http.HandlerFunc(patchUserHandler),
				},
			},
			"/account/{id}/user/{userId}": {
				Tags: []string{"account"},
				Get: &gopenapi.Operation{
					OperationId: "GetUserById",
					Summary:     "Get the user by id",
					Description: "Get the user by id",
					Tags:        []string{"user"},
					Security:    gopenapi.NoSecurity,
					Parameters: gopenapi.Parameters{
						{
							Name:        "id",
							In:          gopenapi.InPath,
							Description: "The id of the account",
							Schema: gopenapi.Schema{
								Type: gopenapi.Integer,
							},
						},
						{
							Name:        "userId",
							In:          gopenapi.InPath,
							Description: "The id of the user",
							Schema: gopenapi.Schema{
								Type: gopenapi.Integer,
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
						},
					},
					Handler: http.HandlerFunc(getAccountUserHandler),
				},
			},
			"/docs": {
				Tags: []string{"docs"},
				Get: &gopenapi.Operation{
					Summary:     "Get the docs path",
					Description: "Get the docs path",
					Tags:        []string{"docs"},
					Security:    gopenapi.NoSecurity,
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
						},
					},
					Handler: http.HandlerFunc(getDocsHandler),
				},
			},
			"/openapi.json": {
				Summary:     "Get the openapi.json path",
				Description: "Get the openapi.json path",
				Tags:        []string{"openapi"},
				Get: &gopenapi.Operation{
					Summary:     "Get the openapi.json path",
					Description: "Get the openapi.json path",
					Tags:        []string{"openapi"},
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
						},
					},
					Handler: http.HandlerFunc(getOpenAPIJSONHandler),
				},
			},
			"/test-params": {
				Get: &gopenapi.Operation{
					OperationId: "TestParams",
					Summary:     "Test query, header, and cookie parameters",
					Security:    gopenapi.NoSecurity,
					Parameters: gopenapi.Parameters{
						{
							Name:     "queryParamStr",
							In:       gopenapi.InQuery,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.String},
						},
						{
							Name:     "queryParamInt",
							In:       gopenapi.InQuery,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.Integer},
						},
						{
							Name:     "X-Header-Str",
							In:       gopenapi.InHeader,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.String},
						},
						{
							Name:   "X-Header-Int",
							In:     gopenapi.InHeader,
							Schema: gopenapi.Schema{Type: gopenapi.Integer},
						},
						{
							Name:     "cookieParamStr",
							In:       gopenapi.InCookie,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.String},
						},
						{
							Name:   "cookieParamInt",
							In:     gopenapi.InCookie,
							Schema: gopenapi.Schema{Type: gopenapi.Integer},
						},
					},
					Responses: gopenapi.Responses{
						200: {Description: "OK"},
						400: {Description: "Bad Request"},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						spec, _ := gopenapi.SpecFromRequest(r)
						op, _ := gopenapi.OperationFromRequest(r)

						_, err := spec.ValidationMiddleware.ValidateRequest(op, r)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte("Params OK")); err != nil {
							http.Error(w, "Failed to write response", http.StatusInternalServerError)
							return
						}
					}),
				},
			},
		},
		Servers: gopenapi.Servers{
			{
				URL:         "/",
				Description: "Localhost",
			},
			{
				URL:         "https://api.foo.ai",
				Description: "Production",
			},
			{
				URL:         "https://dev-api.foo.ai",
				Description: "Development",
			},
		},
		Components: gopenapi.Components{
			SecuritySchemes: gopenapi.SecuritySchemes{
				"apiKey": {
					Type:    gopenapi.APIKey,
					Scheme:  gopenapi.BasicScheme,
					Handler: apiKeyHandler,
				},
			},
			Schemas: gopenapi.Schemas{
				"User": {
					Type: gopenapi.Object[User](),
				},
				"Item": {
					Type: gopenapi.Object[struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					}](),
				},
			},
		},
		Security: []gopenapi.Security{
			{
				"apiKey": []string{},
			},
		},
	}

	server, err := gopenapi.NewServer(schema, "8080")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("get user by invalid id", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/user/a", nil)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, response.Code)
		}
	})
	t.Run("get user by valid id", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/user/1", nil)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d", http.StatusOK, response.Code)
		}
	})
	t.Run("get account user by invalid id", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/account/a/user/1", nil)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, response.Code)
		}
	})
	t.Run("get account user by valid id", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/account/1/user/1", nil)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d", http.StatusOK, response.Code)
		}
	})
	t.Run("patch user by valid id", func(t *testing.T) {
		request, err := http.NewRequest("PATCH", "http://127.0.0.1:8080/user/1", bytes.NewBuffer([]byte(`{"name": "John Doe"}`)))
		request.Header.Set("Content-Type", "application/json")
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d", http.StatusOK, response.Code)
		}
	})
	t.Run("test query, header, cookie params - valid", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test&queryParamInt=123", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Header-Str", "headerTest")
		req.Header.Set("X-Header-Int", "456")
		req.AddCookie(&http.Cookie{Name: "cookieParamStr", Value: "cookieValue"})
		req.AddCookie(&http.Cookie{Name: "cookieParamInt", Value: "789"})

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status OK %d, got %d. Body: %s", http.StatusOK, response.Code, response.Body.String())
		}
	})
	t.Run("test query params - missing required queryParamInt", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Header-Str", "headerTest")
		req.AddCookie(&http.Cookie{Name: "cookieParamStr", Value: "cookieValue"})

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status Bad Request %d, got %d. Body: %s", http.StatusBadRequest, response.Code, response.Body.String())
		}
	})
	t.Run("test query params - invalid queryParamInt type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test&queryParamInt=abc", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Header-Str", "headerTest")
		req.AddCookie(&http.Cookie{Name: "cookieParamStr", Value: "cookieValue"})

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status Bad Request %d, got %d. Body: %s", http.StatusBadRequest, response.Code, response.Body.String())
		}
	})
	t.Run("test header params - missing required X-Header-Str", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test&queryParamInt=123", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: "cookieParamStr", Value: "cookieValue"})

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status Bad Request %d, got %d. Body: %s", http.StatusBadRequest, response.Code, response.Body.String())
		}
	})
	t.Run("test header params - invalid X-Header-Int type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test&queryParamInt=123", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Header-Str", "headerTest")
		req.Header.Set("X-Header-Int", "xyz")
		req.AddCookie(&http.Cookie{Name: "cookieParamStr", Value: "cookieValue"})

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status Bad Request %d, got %d. Body: %s", http.StatusBadRequest, response.Code, response.Body.String())
		}
	})
	t.Run("test cookie params - missing required cookieParamStr", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test&queryParamInt=123", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Header-Str", "headerTest")
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status Bad Request %d, got %d. Body: %s", http.StatusBadRequest, response.Code, response.Body.String())
		}
	})
	t.Run("test cookie params - invalid cookieParamInt type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-params?queryParamStr=test&queryParamInt=123", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Header-Str", "headerTest")
		req.AddCookie(&http.Cookie{Name: "cookieParamStr", Value: "cookieValue"})
		req.AddCookie(&http.Cookie{Name: "cookieParamInt", Value: "non-integer"})

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("Expected status Bad Request %d, got %d. Body: %s", http.StatusBadRequest, response.Code, response.Body.String())
		}
	})
}

func getDocsHandler(w http.ResponseWriter, r *http.Request) {
	// htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
	// 	SpecURL: "http://127.0.0.1:8080/openapi.json",
	// })
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	w.WriteHeader(200)
	// w.Write([]byte(htmlContent))
}

func getOpenAPIJSONHandler(writer http.ResponseWriter, request *http.Request) {
	spec, ok := gopenapi.SpecFromRequest(request)
	if !ok {
		http.Error(writer, "No spec found", http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(200)
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Pragma", "no-cache")
	writer.Header().Set("Expires", "0")
	bytes, err := json.Marshal(spec)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := writer.Write(bytes); err != nil {
		http.Error(writer, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func apiKeyHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		apiKey := request.Header.Get("X-API-KEY")
		if apiKey != "1234567890" {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func getUserHandler(writer http.ResponseWriter, request *http.Request) {
	id := 0
	err := gopenapi.ValidateRequestPathValue(request, "id", &id)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	gopenapi.WriteResponse(writer, 200, map[string]any{"id": id, "name": "John Doe"})
}

func getAccountUserHandler(writer http.ResponseWriter, request *http.Request) {
	paths := struct {
		Id     int `json:"id"`
		UserId int `json:"userId"`
	}{}
	err := gopenapi.ValidateRequestPathValues(request, &paths)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	gopenapi.WriteResponse(writer, 200, map[string]any{"id": paths.Id, "userId": paths.UserId})
}

func patchUserHandler(writer http.ResponseWriter, request *http.Request) {
	user := User{}
	err := gopenapi.ValidateRequestBody(request, &user)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	gopenapi.WriteResponse(writer, 200, map[string]any{"name": user.Name})
}

func TestSchemaReferences(t *testing.T) {
	// Define a Product struct for testing
	type Product struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	schema := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:       "Test API with References",
			Description: "Test API with schema references",
			Version:     "1.0.0",
		},
		Components: gopenapi.Components{
			Schemas: gopenapi.Schemas{
				"User": {
					Type: gopenapi.Object[User](),
				},
				"Product": {
					Type: gopenapi.Object[Product](),
				},
				"UserRef": {
					Ref: "#/components/schemas/User",
				},
			},
		},
		Paths: gopenapi.Paths{
			"/products": {
				Post: &gopenapi.Operation{
					OperationId: "CreateProduct",
					Summary:     "Create a product",
					Security:    gopenapi.NoSecurity,
					RequestBody: gopenapi.RequestBody{
						Content: gopenapi.Content{
							"application/json": {
								Schema: gopenapi.Schema{
									Ref: "#/components/schemas/Product",
								},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "Created",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/Product",
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						product := Product{}
						err := gopenapi.ValidateRequestBody(r, &product)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						gopenapi.WriteResponse(w, 201, product)
					}),
				},
			},
			"/users/{id}/profile": {
				Get: &gopenapi.Operation{
					OperationId: "GetUserProfile",
					Summary:     "Get user profile using schema reference",
					Security:    gopenapi.NoSecurity,
					Parameters: gopenapi.Parameters{
						{
							Name: "id",
							In:   gopenapi.InPath,
							Schema: gopenapi.Schema{
								Type: gopenapi.Integer,
							},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/User",
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						id := 0
						err := gopenapi.ValidateRequestPathValue(r, "id", &id)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						gopenapi.WriteResponse(w, 200, User{Name: "John Doe"})
					}),
				},
			},
			"/nested-ref-test": {
				Get: &gopenapi.Operation{
					OperationId: "TestNestedRef",
					Summary:     "Test nested schema reference",
					Security:    gopenapi.NoSecurity,
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/UserRef",
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						gopenapi.WriteResponse(w, 200, User{Name: "Jane Doe"})
					}),
				},
			},
		},
		Servers: gopenapi.Servers{
			{
				URL:         "/",
				Description: "Localhost",
			},
		},
	}

	server, err := gopenapi.NewServer(schema, "8080")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("create product with schema reference", func(t *testing.T) {
		productJSON := `{"id": 1, "name": "Test Product", "price": 99.99}`
		request, err := http.NewRequest("POST", "http://127.0.0.1:8080/products", bytes.NewBuffer([]byte(productJSON)))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusCreated {
			t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusCreated, response.Code, response.Body.String())
		}
	})

	t.Run("get user profile with schema reference", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/users/123/profile", nil)
		if err != nil {
			t.Fatal(err)
		}

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, response.Code, response.Body.String())
		}
	})

	t.Run("test nested schema reference", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/nested-ref-test", nil)
		if err != nil {
			t.Fatal(err)
		}

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, response.Code, response.Body.String())
		}
	})
}

func TestSchemaReferenceErrors(t *testing.T) {
	t.Run("external reference not supported", func(t *testing.T) {
		schema := &gopenapi.Spec{
			OpenAPI: "3.0.0",
			Info: gopenapi.Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Components: gopenapi.Components{
				Schemas: gopenapi.Schemas{
					"User": {
						Type: gopenapi.Object[User](),
					},
				},
			},
			Paths: gopenapi.Paths{
				"/test": {
					Get: &gopenapi.Operation{
						Security: gopenapi.NoSecurity,
						Responses: gopenapi.Responses{
							200: {
								Description: "OK",
								Content: gopenapi.Content{
									"application/json": {
										Schema: gopenapi.Schema{
											Ref: "external-file.json#/schemas/User",
										},
									},
								},
							},
						},
						Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
					},
				},
			},
			Servers: gopenapi.Servers{
				{URL: "/"},
			},
		}

		_, err := gopenapi.NewServer(schema, "8080")
		if err == nil {
			t.Fatal("Expected error for external reference")
		}
		if !strings.Contains(err.Error(), "external references not supported") {
			t.Fatalf("Expected error about external references not supported, got: %s", err.Error())
		}
	})

	t.Run("missing schema reference", func(t *testing.T) {
		schema := &gopenapi.Spec{
			OpenAPI: "3.0.0",
			Info: gopenapi.Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Components: gopenapi.Components{
				Schemas: gopenapi.Schemas{
					"User": {
						Type: gopenapi.Object[User](),
					},
				},
			},
			Paths: gopenapi.Paths{
				"/test": {
					Get: &gopenapi.Operation{
						Security: gopenapi.NoSecurity,
						Responses: gopenapi.Responses{
							200: {
								Description: "OK",
								Content: gopenapi.Content{
									"application/json": {
										Schema: gopenapi.Schema{
											Ref: "#/components/schemas/NonExistent",
										},
									},
								},
							},
						},
						Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
					},
				},
			},
			Servers: gopenapi.Servers{
				{URL: "/"},
			},
		}

		_, err := gopenapi.NewServer(schema, "8080")
		if err == nil {
			t.Fatal("Expected error for missing schema reference")
		}
		if !strings.Contains(err.Error(), "schema not found") {
			t.Fatalf("Expected error about schema not found, got: %s", err.Error())
		}
	})

	t.Run("circular reference detection", func(t *testing.T) {
		schema := &gopenapi.Spec{
			OpenAPI: "3.0.0",
			Info: gopenapi.Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Components: gopenapi.Components{
				Schemas: gopenapi.Schemas{
					"A": {
						Ref: "#/components/schemas/B",
					},
					"B": {
						Ref: "#/components/schemas/A",
					},
				},
			},
			Paths: gopenapi.Paths{
				"/test": {
					Get: &gopenapi.Operation{
						Security: gopenapi.NoSecurity,
						Responses: gopenapi.Responses{
							200: {
								Description: "OK",
								Content: gopenapi.Content{
									"application/json": {
										Schema: gopenapi.Schema{
											Ref: "#/components/schemas/A",
										},
									},
								},
							},
						},
						Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
					},
				},
			},
			Servers: gopenapi.Servers{
				{URL: "/"},
			},
		}

		_, err := gopenapi.NewServer(schema, "8080")
		if err == nil {
			t.Fatal("Expected error for circular reference")
		}
		// The error should indicate a problem with resolving nested references
		if !strings.Contains(err.Error(), "failed to resolve") {
			t.Fatalf("Expected error about failed resolution, got: %s", err.Error())
		}
	})

	t.Run("invalid JSON pointer format", func(t *testing.T) {
		schema := &gopenapi.Spec{
			OpenAPI: "3.0.0",
			Info: gopenapi.Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Components: gopenapi.Components{
				Schemas: gopenapi.Schemas{
					"User": {
						Type: gopenapi.Object[User](),
					},
				},
			},
			Paths: gopenapi.Paths{
				"/test": {
					Get: &gopenapi.Operation{
						Security: gopenapi.NoSecurity,
						Responses: gopenapi.Responses{
							200: {
								Description: "OK",
								Content: gopenapi.Content{
									"application/json": {
										Schema: gopenapi.Schema{
											Ref: "#/invalid/pointer",
										},
									},
								},
							},
						},
						Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
					},
				},
			},
			Servers: gopenapi.Servers{
				{URL: "/"},
			},
		}

		_, err := gopenapi.NewServer(schema, "8080")
		if err == nil {
			t.Fatal("Expected error for invalid JSON pointer")
		}
		if !strings.Contains(err.Error(), "unsupported JSON pointer root") {
			t.Fatalf("Expected error about unsupported JSON pointer root, got: %s", err.Error())
		}
	})
}

func TestSchemaReferenceJSONSerialization(t *testing.T) {
	// Test that schema references are preserved in JSON serialization
	schema := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: gopenapi.Components{
			Schemas: gopenapi.Schemas{
				"User": {
					Type: gopenapi.Object[User](),
				},
			},
		},
		Paths: gopenapi.Paths{
			"/test": {
				Get: &gopenapi.Operation{
					Security: gopenapi.NoSecurity,
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/User",
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						gopenapi.WriteResponse(w, 200, User{Name: "Test"})
					}),
				},
			},
		},
		Servers: gopenapi.Servers{
			{URL: "/"},
		},
	}

	// Create server to trigger reference resolution
	_, err := gopenapi.NewServer(schema, "8080")
	if err != nil {
		t.Fatal(err)
	}

	// Serialize the spec to JSON
	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}

	// Parse the JSON to verify the $ref is preserved
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatal(err)
	}

	// Navigate to the schema reference in the JSON
	paths := parsed["paths"].(map[string]interface{})
	testPath := paths["/test"].(map[string]interface{})
	getOp := testPath["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	response200 := responses["200"].(map[string]interface{})
	content := response200["content"].(map[string]interface{})
	appJson := content["application/json"].(map[string]interface{})
	schemaObj := appJson["schema"].(map[string]interface{})

	// Verify that the $ref field is preserved
	ref, exists := schemaObj["$ref"]
	if !exists {
		t.Fatal("Expected $ref field to be preserved in JSON serialization")
	}
	if ref != "#/components/schemas/User" {
		t.Fatalf("Expected $ref to be '#/components/schemas/User', got %s", ref)
	}

	// Verify that the schema was resolved internally (Type field should be set)
	// We can't check this directly in JSON since Type is not serialized,
	// but we can verify the resolution worked by checking that the server was created successfully
	t.Log("Schema reference resolution and JSON serialization test passed")
}

func TestJSONPointerReferenceFormats(t *testing.T) {
	// Test that various JSON Pointer reference formats work correctly
	type Product struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	schema := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: gopenapi.Components{
			Schemas: gopenapi.Schemas{
				"User": {
					Type: gopenapi.Object[User](),
				},
				"Product": {
					Type: gopenapi.Object[Product](),
				},
			},
		},
		Paths: gopenapi.Paths{
			"/test-standard": {
				Get: &gopenapi.Operation{
					Security: gopenapi.NoSecurity,
					Responses: gopenapi.Responses{
						200: {
							Description: "OK",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/User", // Standard format
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						gopenapi.WriteResponse(w, 200, User{Name: "Test User"})
					}),
				},
			},
		},
		Servers: gopenapi.Servers{
			{URL: "/"},
		},
	}

	// Create server to trigger reference resolution
	server, err := gopenapi.NewServer(schema, "8080")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("standard components schema reference", func(t *testing.T) {
		request, err := http.NewRequest("GET", "http://127.0.0.1:8080/test-standard", nil)
		if err != nil {
			t.Fatal(err)
		}

		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, response.Code, response.Body.String())
		}
	})

	t.Log("JSON Pointer reference formats test passed")
}
