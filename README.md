# gopenapi

`gopenapi` is a Go library for building OpenAPI-compliant HTTP services. It provides tools for defining your API using OpenAPI specifications and handles request validation, routing, and response generation.

## Features

*   OpenAPI 3.0.x support
*   Request validation
*   Response generation
*   Middleware support
*   Automatic schema generation from Go types

## Installation

```bash
go get github.com/gabewillen/gopenapi 
```


## Usage

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/gabewillen/gopenapi" // Replace with your actual import path
)

// Define a struct for your data
type User struct {
	Name string `json:"name"`
}

// Example handler function
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	// In a real application, you would fetch user data based on r (e.g., path parameters)
	gopenapi.WriteResponse(w, http.StatusOK, User{Name: "John Doe"})
}

func main() {
	spec := &gopenapi.Spec{
		OpenAPI: "3.0.0",
		Info: gopenapi.Info{
			Title:   "My API",
			Version: "1.0.0",
		},
		Servers: gopenapi.Servers{
			{
				URL: "/", 
				Description: "Local server",
			},
		},
		Paths: gopenapi.Paths{
			"/user/{id}": {
				Get: &gopenapi.Operation{
					Summary:     "Get a user by ID",
					OperationId: "getUserById",
					Parameters: gopenapi.Parameters{
						{
							Name:        "id",
							In:          gopenapi.InPath,
							Description: "The ID of the user",
							Required:    true,
							Schema:      gopenapi.Schema{Type: gopenapi.Integer},
						},
					},
					Responses: gopenapi.Responses{
						// Define your 200 response
						http.StatusOK: {
							Description: "Successful response",
							Content: gopenapi.Content{
								gopenapi.ApplicationJSON: {
									Schema: gopenapi.Schema{Type: gopenapi.Object[User]()},
								},
							},
						},
						// You can add other responses like 404, 500 etc.
					},
					Handler: http.HandlerFunc(getUserHandler),
				},
			},
		},
		// You can also define components like schemas globally
		Components: gopenapi.Components{
			Schemas: gopenapi.Schemas{
				"User": {
					Type: gopenapi.Object[User](), // Reuses the User struct defined above
				},
			},
		},
	}

	server, err := gopenapi.NewServer(spec, "8080")
	if err != nil {
		panic(err)
	}

	// You can also serve OpenAPI spec as JSON
	spec.Paths["/openapi.json"] = gopenapi.Path{
		Get: &gopenapi.Operation{
			Summary: "Get OpenAPI Specification",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				jsonSpec, _ := json.MarshalIndent(spec, "", "  ") // Pretty print JSON
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(jsonSpec)
			}),
		},
	}

	// Update the server with the new path for openapi.json
	// This step might differ based on how NewServer and your routing is implemented.
	// For simplicity, we're assuming NewServer can be called again or has a method to update routes.
	// In a real scenario, you'd likely define all paths before calling NewServer the first time.
	server, err = gopenapi.NewServer(spec, "8080") // Re-initialize or update server
	if err != nil {
		panic(err)
	}

	println("Server started on :8080")
	println("Visit http://localhost:8080/user/123 for an example endpoint.")
	println("Visit http://localhost:8080/openapi.json for the OpenAPI spec.")

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
```

*(This is a more detailed example based on your tests. You'll need to adapt it to your specific needs and project structure, and ensure the import paths are correct.)*

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 