package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/runpod/gopenapi"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func main() {
	spec := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:       "Schema References Example",
			Description: "Example demonstrating schema references in gopenapi",
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
				// Example of a schema that references another schema
				"UserReference": {
					Ref: "#/components/schemas/User",
				},
			},
		},
		Paths: gopenapi.Paths{
			"/users": {
				Post: &gopenapi.Operation{
					OperationId: "CreateUser",
					Summary:     "Create a new user",
					Security:    gopenapi.NoSecurity,
					RequestBody: gopenapi.RequestBody{
						Required: true,
						Content: gopenapi.Content{
							"application/json": {
								Schema: gopenapi.Schema{
									Ref: "#/components/schemas/User", // Using schema reference
								},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "User created successfully",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/User", // Using schema reference
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var user User
						err := gopenapi.ValidateRequestBody(r, &user)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}

						// In a real application, you would save the user to a database
						user.ID = 123 // Simulate assigned ID

						gopenapi.WriteResponse(w, 201, user)
					}),
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
									Ref: "#/components/schemas/Product", // Using schema reference
								},
							},
						},
					},
					Responses: gopenapi.Responses{
						201: {
							Description: "Product created successfully",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/Product", // Using schema reference
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var product Product
						err := gopenapi.ValidateRequestBody(r, &product)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}

						// In a real application, you would save the product to a database
						product.ID = 456 // Simulate assigned ID

						gopenapi.WriteResponse(w, 201, product)
					}),
				},
			},
			"/users/{id}/profile": {
				Get: &gopenapi.Operation{
					OperationId: "GetUserProfile",
					Summary:     "Get user profile using nested schema reference",
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
							Description: "User profile",
							Content: gopenapi.Content{
								"application/json": {
									Schema: gopenapi.Schema{
										Ref: "#/components/schemas/UserReference", // Using nested reference
									},
								},
							},
						},
					},
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var id int
						err := gopenapi.ValidateRequestPathValue(r, "id", &id)
						if err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}

						user := User{
							ID:    id,
							Name:  "John Doe",
							Email: "john@example.com",
						}

						gopenapi.WriteResponse(w, 200, user)
					}),
				},
			},
		},
		Servers: gopenapi.Servers{
			{
				URL:         "http://localhost:8080",
				Description: "Local development server",
			},
		},
	}

	server, err := gopenapi.NewServer(spec, "8080")
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Schema references have been resolved automatically at startup!")
	fmt.Println("\nTry these endpoints:")
	fmt.Println("POST /users - Create a user with JSON body")
	fmt.Println("POST /products - Create a product with JSON body")
	fmt.Println("GET /users/123/profile - Get user profile")

	log.Fatal(server.ListenAndServe())
}
