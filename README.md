# gopenapi

`gopenapi` is a Go library for building OpenAPI-compliant HTTP services and generating HTTP clients. It provides tools for defining your API using OpenAPI specifications and handles request validation, routing, response generation, and client code generation.

## Features

*   OpenAPI 3.0.x support
*   Request validation
*   Response generation
*   Middleware support
*   Automatic schema generation from Go types
*   Cross-platform client generation (Windows, macOS, Linux)
*   AST-based Go file parsing (no CGO required)

## Installation

### Install the CLI tool

```bash
go install github.com/runpod/gopenapi/cmd/gopenapi@latest
```

### Install the library

```bash
go get github.com/runpod/gopenapi
```

## CLI Usage

The `gopenapi` command-line tool can generate HTTP clients in multiple languages from OpenAPI specifications.

### Generate Clients from Go Files

First, create a Go file with your OpenAPI specification:

```go
// api_spec.go
package main

import (
    "net/http"
    "github.com/runpod/gopenapi"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

var MyAPISpec = gopenapi.Spec{
    OpenAPI: "3.0.0",
    Info: gopenapi.Info{
        Title:   "My API",
        Version: "1.0.0",
    },
    Paths: gopenapi.Paths{
        "/users/{id}": gopenapi.Path{
            Get: &gopenapi.Operation{
                OperationId: "getUserById",
                Summary:     "Get a user by ID",
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
                            gopenapi.ApplicationJSON: {
                                Schema: gopenapi.Schema{Type: gopenapi.Object[User]()},
                            },
                        },
                    },
                },
                Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    // Your handler implementation
                }),
            },
        },
    },
}
```

Then generate clients:

```bash
# Generate clients for all supported languages
gopenapi -spec api_spec.go -var MyAPISpec -languages=go,python,typescript -output=./clients

# Generate only Go client
gopenapi -spec api_spec.go -var MyAPISpec -languages=go -output=./clients

# Generate only Python client  
gopenapi -spec api_spec.go -var MyAPISpec -languages=python -output=./clients

# Generate only TypeScript client
gopenapi -spec api_spec.go -var MyAPISpec -languages=typescript -output=./clients
```

### Command Line Options

- `-spec` - Go file containing the OpenAPI spec (required)
- `-var` - Variable name containing the spec (required, e.g., 'ExampleSpec')
- `-languages` - Comma-separated list of languages to generate (go,python,typescript)
- `-output` - Output directory for generated clients (default: current directory)
- `-package` - Package name for generated code (default: client)

### Generated Client Features

**Go Client:**
- Type-safe parameter and response handling
- Context support for request cancellation
- Automatic JSON marshaling/unmarshaling
- Configurable HTTP client
- Error handling with detailed error information

**Python Client:**
- Type hints for better IDE support
- Dataclasses for clean, immutable parameter and response objects
- Automatic JSON handling with proper field name conversion
- Session-based requests for connection pooling
- Configurable headers
- Exception-based error handling

**TypeScript Client:**
- Full TypeScript type safety with interfaces for all parameters and responses
- Modern async/await API using fetch
- Configurable timeout and headers
- Automatic JSON serialization/deserialization
- Proper error handling with custom ApiError class
- Support for both Node.js and browser environments

### TypeScript Usage Example

```typescript
import { ClientClient, ApiError } from './client';

const client = new ClientClient({
  baseURL: 'https://api.example.com',
  headers: {
    'Authorization': 'Bearer your-token-here'
  },
  timeout: 10000
});

try {
  // Type-safe API calls
  const user = await client.getUserById(
    { id: 123 },                    // path params
    { include: 'profile' },         // query params (optional)
    { authorization: 'Bearer ...' } // headers (optional)
  );
  
  console.log('User:', user);
} catch (error) {
  if (error instanceof ApiError) {
    console.error(`API Error ${error.statusCode}: ${error.message}`);
  } else {
    console.error('Network error:', error);
  }
}
```

## Library Usage

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/runpod/gopenapi" // Replace with your actual import path
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
