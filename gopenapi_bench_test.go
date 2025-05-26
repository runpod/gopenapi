package gopenapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/runpod/gopenapi"
)

// Benchmark data structures
type BenchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type BenchProduct struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// Stock HTTP handlers for comparison
func stockGetUserHandler(w http.ResponseWriter, r *http.Request) {
	user := BenchUser{
		ID:    123,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func stockCreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var product BenchProduct
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simulate some processing
	product.ID = 456

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(product)
}

// GopenAPI handlers
func gopenapiGetUserHandler(w http.ResponseWriter, r *http.Request) {
	var id int
	err := gopenapi.ValidateRequestPathValue(r, "id", &id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := BenchUser{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	gopenapi.WriteResponse(w, 200, user)
}

func gopenapiCreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var product BenchProduct
	err := gopenapi.ValidateRequestBody(r, &product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simulate some processing
	product.ID = 456

	gopenapi.WriteResponse(w, 201, product)
}

// Setup functions
func setupStockHTTPServer() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", stockGetUserHandler)
	mux.HandleFunc("POST /products", stockCreateProductHandler)
	return mux
}

func setupGopenapiServer() (http.Handler, error) {
	spec := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "Benchmark API",
			Version: "1.0.0",
		},
		Components: gopenapi.Components{
			Schemas: gopenapi.Schemas{
				"User": {
					Type: gopenapi.Object[BenchUser](),
				},
				"Product": {
					Type: gopenapi.Object[BenchProduct](),
				},
			},
		},
		Paths: gopenapi.Paths{
			"/users/{id}": {
				Get: &gopenapi.Operation{
					OperationId: "GetUser",
					Summary:     "Get user by ID",
					Security:    gopenapi.NoSecurity,
					Parameters: gopenapi.Parameters{
						{
							Name:     "id",
							In:       gopenapi.InPath,
							Required: true,
							Schema:   gopenapi.Schema{Type: gopenapi.Integer},
						},
					},
					Responses: gopenapi.Responses{
						200: {
							Description: "User found",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/User",
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(gopenapiGetUserHandler),
				},
			},
			"/products": {
				Post: &gopenapi.Operation{
					OperationId: "CreateProduct",
					Summary:     "Create a new product",
					Security:    gopenapi.NoSecurity,
					RequestBody: gopenapi.RequestBody{
						Required: true,
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
							Description: "Product created",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/Product",
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(gopenapiCreateProductHandler),
				},
			},
		},
		Servers: gopenapi.Servers{
			{URL: "/"},
		},
	}

	return gopenapi.NewServerMux(spec)
}

// Benchmark GET requests
func BenchmarkStockHTTP_GET(b *testing.B) {
	mux := setupStockHTTPServer()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/users/123", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

func BenchmarkGopenapi_GET(b *testing.B) {
	handler, err := setupGopenapiServer()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/users/123", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// Benchmark POST requests with JSON body
func BenchmarkStockHTTP_POST(b *testing.B) {
	mux := setupStockHTTPServer()

	productJSON := `{"name": "Test Product", "price": 99.99, "description": "A test product"}`

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/products", bytes.NewBufferString(productJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				b.Fatalf("Expected status 201, got %d", w.Code)
			}
		}
	})
}

func BenchmarkGopenapi_POST(b *testing.B) {
	handler, err := setupGopenapiServer()
	if err != nil {
		b.Fatal(err)
	}

	productJSON := `{"name": "Test Product", "price": 99.99, "description": "A test product"}`

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/products", bytes.NewBufferString(productJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				b.Fatalf("Expected status 201, got %d", w.Code)
			}
		}
	})
}

// Benchmark server setup/initialization
func BenchmarkStockHTTP_Setup(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = setupStockHTTPServer()
	}
}

func BenchmarkGopenapi_Setup(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := setupGopenapiServer()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Memory allocation benchmarks
func BenchmarkStockHTTP_GET_Allocs(b *testing.B) {
	mux := setupStockHTTPServer()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkGopenapi_GET_Allocs(b *testing.B) {
	handler, err := setupGopenapiServer()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkStockHTTP_POST_Allocs(b *testing.B) {
	mux := setupStockHTTPServer()
	productJSON := `{"name": "Test Product", "price": 99.99, "description": "A test product"}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/products", bytes.NewBufferString(productJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkGopenapi_POST_Allocs(b *testing.B) {
	handler, err := setupGopenapiServer()
	if err != nil {
		b.Fatal(err)
	}

	productJSON := `{"name": "Test Product", "price": 99.99, "description": "A test product"}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/products", bytes.NewBufferString(productJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// Correctness tests to verify both implementations work
func TestBenchmarkCorrectness(t *testing.T) {
	t.Run("stock HTTP GET", func(t *testing.T) {
		mux := setupStockHTTPServer()
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var user BenchUser
		if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
			t.Fatal(err)
		}

		if user.ID != 123 || user.Name != "John Doe" {
			t.Fatalf("Unexpected user data: %+v", user)
		}
	})

	t.Run("gopenapi GET", func(t *testing.T) {
		handler, err := setupGopenapiServer()
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var user BenchUser
		if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
			t.Fatal(err)
		}

		if user.ID != 123 || user.Name != "John Doe" {
			t.Fatalf("Unexpected user data: %+v", user)
		}
	})

	t.Run("stock HTTP POST", func(t *testing.T) {
		mux := setupStockHTTPServer()
		productJSON := `{"name": "Test Product", "price": 99.99, "description": "A test product"}`

		req := httptest.NewRequest("POST", "/products", bytes.NewBufferString(productJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", w.Code)
		}

		var product BenchProduct
		if err := json.NewDecoder(w.Body).Decode(&product); err != nil {
			t.Fatal(err)
		}

		if product.ID != 456 || product.Name != "Test Product" {
			t.Fatalf("Unexpected product data: %+v", product)
		}
	})

	t.Run("gopenapi POST", func(t *testing.T) {
		handler, err := setupGopenapiServer()
		if err != nil {
			t.Fatal(err)
		}

		productJSON := `{"name": "Test Product", "price": 99.99, "description": "A test product"}`

		req := httptest.NewRequest("POST", "/products", bytes.NewBufferString(productJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", w.Code)
		}

		var product BenchProduct
		if err := json.NewDecoder(w.Body).Decode(&product); err != nil {
			t.Fatal(err)
		}

		if product.ID != 456 || product.Name != "Test Product" {
			t.Fatalf("Unexpected product data: %+v", product)
		}
	})
}
