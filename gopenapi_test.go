package gopenapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/gabewillen/gopenapi"
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
						w.Write([]byte("Params OK"))
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
	htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecURL: "http://127.0.0.1:8080/openapi.json",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte(htmlContent))
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
	writer.Write(bytes)
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
