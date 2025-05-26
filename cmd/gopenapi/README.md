# GopenAPI Client Generator

The GopenAPI client generator creates HTTP client code from your `gopenapi.Spec` definitions. It generates type-safe Go clients with proper parameter handling for path, query, header parameters, and request/response bodies.

## Features

- **Type-safe clients**: Generated clients use strongly-typed structs for all parameters
- **Context support**: All methods accept `context.Context` for cancellation and timeouts
- **Flexible options**: Each operation has an options struct containing Path, Query, Headers, and Body parameters as needed
- **Error handling**: Proper error types with status codes and response bodies
- **Template-based**: Uses Go templates for customizable code generation

## Usage

### As a Library Function

```go
package main

import (
    "net/http"
    "github.com/runpod/gopenapi"
)

func main() {
    // Define your API types
    type User struct {
        ID    int    `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    type CreateUserRequest struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    // Create your OpenAPI spec
    spec := gopenapi.Spec{
        OpenAPI: "3.0.0",
        Info: gopenapi.Info{
            Title:       "My API",
            Description: "My awesome API",
            Version:     "1.0.0",
        },
        Servers: gopenapi.Servers{
            {
                URL:         "https://api.mycompany.com",
                Description: "Production server",
            },
        },
        Paths: gopenapi.Paths{
            "/users/{id}": gopenapi.Path{
                Get: &gopenapi.Operation{
                    OperationId: "getUser",
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
                        {
                            Name:        "include",
                            In:          gopenapi.InQuery,
                            Description: "Include additional data",
                            Schema:      gopenapi.Schema{Type: gopenapi.String},
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
                        gopenapi.WriteResponse(w, 200, User{ID: 1, Name: "John", Email: "john@example.com"})
                    }),
                },
            },
            "/users": gopenapi.Path{
                Post: &gopenapi.Operation{
                    OperationId: "createUser",
                    Summary:     "Create a new user",
                    Description: "Create a new user account",
                    RequestBody: gopenapi.RequestBody{
                        Required: true,
                        Content: gopenapi.Content{
                            gopenapi.ApplicationJSON: {
                                Schema: gopenapi.Schema{Type: gopenapi.Object[CreateUserRequest]()},
                            },
                        },
                    },
                    Responses: gopenapi.Responses{
                        201: {
                            Description: "User created",
                            Content: gopenapi.Content{
                                gopenapi.ApplicationJSON: {
                                    Schema: gopenapi.Schema{Type: gopenapi.Object[User]()},
                                },
                            },
                        },
                    },
                    Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        // Your handler implementation
                        gopenapi.WriteResponse(w, 201, User{ID: 2, Name: "Jane", Email: "jane@example.com"})
                    }),
                },
            },
        },
    }

    // Generate the client
    err := gopenapi.GenerateClient(&spec, "my-api-client.go", "myclient", "client.tpl")
    if err != nil {
        log.Fatalf("Failed to generate client: %v", err)
    }

    fmt.Println("Client generated successfully!")
}
```

### Using the Command Line Tool

```bash
# Build the generator
go build -o gopenapi-client-gen ./cmd/gopenapi-client-gen

# Generate a demo client
./gopenapi-client-gen -demo -output example-client.go -package exampleclient

# Use with custom template
./gopenapi-client-gen -demo -output my-client.go -package myclient -template my-template.tpl
```

## Generated Client Usage

The generated client provides a clean, type-safe interface:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "path/to/myclient"
)

func main() {
    // Create client
    client := myclient.NewClient("https://api.mycompany.com")
    client.SetHeader("Authorization", "Bearer your-token")

    // Get a user
    user, err := client.Getuser(context.Background(), &myclient.GetuserOptions{
        Path: &myclient.GetuserPathParams{Id: 123},
        Query: &myclient.GetuserQueryParams{Include: "profile"},
    })
    if err != nil {
        log.Fatalf("Failed to get user: %v", err)
    }
    fmt.Printf("User: %+v\n", user)

    // Create a user
    newUser, err := client.Createuser(context.Background(), &myclient.CreateuserOptions{
        Body: &myclient.CreateuserRequestBody{
            Name: "Alice",
            Email: "alice@example.com",
        },
    })
    if err != nil {
        log.Fatalf("Failed to create user: %v", err)
    }
    fmt.Printf("Created user: %+v\n", newUser)
}
```

## Generated Code Structure

For each operation, the generator creates:

1. **Parameter structs** (if needed):
   - `{OperationName}PathParams` - for path parameters
   - `{OperationName}QueryParams` - for query parameters  
   - `{OperationName}HeaderParams` - for header parameters
   - `{OperationName}RequestBody` - for request body

2. **Options struct**:
   - `{OperationName}Options` - contains pointers to all parameter structs

3. **Response struct** (if the operation returns structured data):
   - `{OperationName}Response` - for the response body

4. **Client method**:
   - `func (c *Client) {OperationName}(ctx context.Context, opts *{OperationName}Options) (*{OperationName}Response, error)`

## Template Customization

The client generator uses Go templates. You can customize the generated code by modifying `client.tpl` or creating your own template file.

The template receives a `ClientGeneratorTemplateData` struct with:
- `PackageName` - the target package name
- `Operations` - slice of `ClientOperationData` with all operation details

## Error Handling

The generated client includes proper error handling:

```go
user, err := client.Getuser(ctx, opts)
if err != nil {
    if apiErr, ok := err.(*myclient.Error); ok {
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        // Access raw response body if needed
        fmt.Printf("Raw body: %s\n", string(apiErr.Body))
    } else {
        fmt.Printf("Network/other error: %v\n", err)
    }
}
```

## Requirements

- Go 1.21+ (for generics support)
- The `client.tpl` template file must be available at generation time
- Operations must have `OperationId` set to generate client methods 