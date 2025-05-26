# GopenAPI CLI Tool

The GopenAPI CLI tool generates HTTP client code in multiple languages from your `gopenapi.Spec` definitions. It uses AST parsing to extract OpenAPI specifications from Go files and generates type-safe clients for Go, Python, and TypeScript.

## Features

- **Cross-platform**: Works on Windows, macOS, and Linux without CGO
- **AST-based parsing**: Extracts OpenAPI specs directly from Go source files
- **Multi-language support**: Generates clients for Go, Python, and TypeScript
- **Type-safe clients**: Generated clients use strongly-typed interfaces/structs
- **Template-based**: Uses embedded templates for customizable code generation
- **No external dependencies**: Pure Go implementation with embedded templates

## Installation

```bash
go install github.com/runpod/gopenapi/cmd/gopenapi@latest
```

## Usage

### Basic Usage

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

### Creating a Spec File

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

type CreateUserRequest struct {
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
                    {
                        Name:   "include",
                        In:     gopenapi.InQuery,
                        Schema: gopenapi.Schema{Type: gopenapi.String},
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
        "/users": gopenapi.Path{
            Post: &gopenapi.Operation{
                OperationId: "createUser",
                Summary:     "Create a new user",
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
                }),
            },
        },
    },
}
```

Then generate clients:

```bash
gopenapi -spec api_spec.go -var MyAPISpec -languages=go,python,typescript -output=./clients
```

## Generated Client Features

### Go Client
- Type-safe parameter and response handling
- Context support for request cancellation
- Automatic JSON marshaling/unmarshaling
- Configurable HTTP client
- Error handling with detailed error information

### Python Client
- Type hints for better IDE support
- Dataclasses for clean, immutable parameter and response objects
- Automatic JSON handling with proper field name conversion
- Session-based requests for connection pooling
- Configurable headers
- Exception-based error handling

### TypeScript Client
- Full TypeScript type safety with interfaces for all parameters and responses
- Modern async/await API using fetch
- Configurable timeout and headers
- Automatic JSON serialization/deserialization
- Proper error handling with custom ApiError class
- Support for both Node.js and browser environments

## Usage Examples

### Go Client Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "./clients" // Import your generated client
)

func main() {
    client := clients.NewClient("https://api.example.com")
    client.SetHeader("Authorization", "Bearer your-token")

    // Get a user
    user, err := client.GetUserById(context.Background(), &clients.GetUserByIdOptions{
        Path: &clients.GetUserByIdPathParams{Id: 123},
        Query: &clients.GetUserByIdQueryParams{Include: "profile"},
    })
    if err != nil {
        log.Fatalf("Failed to get user: %v", err)
    }
    fmt.Printf("User: %+v\n", user)
}
```

### Python Client Usage

```python
from clients import ClientClient, APIError

client = ClientClient(
    base_url="https://api.example.com",
    headers={"Authorization": "Bearer your-token"}
)

try:
    # Get a user
    user = client.get_user_by_id(
        path=GetUserByIdPathParams(id=123),
        query=GetUserByIdQueryParams(include="profile")
    )
    print(f"User: {user}")
except APIError as e:
    print(f"API Error {e.status_code}: {e.message}")
```

### TypeScript Client Usage

```typescript
import { ClientClient, ApiError } from './clients/client';

const client = new ClientClient({
  baseURL: 'https://api.example.com',
  headers: {
    'Authorization': 'Bearer your-token'
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

## Generated Code Structure

For each operation, the generator creates:

### Go
- **Parameter structs**: `{OperationName}PathParams`, `{OperationName}QueryParams`, etc.
- **Options struct**: `{OperationName}Options` containing all parameter structs
- **Response struct**: `{OperationName}Response` for structured responses
- **Client method**: `func (c *Client) {OperationName}(ctx context.Context, opts *{OperationName}Options) (*{OperationName}Response, error)`

### Python
- **Dataclass parameters**: `{OperationName}PathParams`, `{OperationName}QueryParams`, etc.
- **Response dataclasses**: `{OperationName}Response` with `from_dict` methods
- **Client method**: `def {operation_name}(self, path: PathParams, query: QueryParams = None, ...) -> Response`

### TypeScript
- **Interface parameters**: `{OperationName}PathParams`, `{OperationName}QueryParams`, etc.
- **Response interfaces**: `{OperationName}Response` for structured responses
- **Client method**: `async {operationName}(path: PathParams, query?: QueryParams, ...) => Promise<Response>`

## Error Handling

All generated clients include comprehensive error handling:

- **Go**: Custom error types with status codes and response bodies
- **Python**: `APIError` exceptions with status codes and messages
- **TypeScript**: `ApiError` class with status codes and response bodies

## Requirements

- Go 1.21+ (for generics support in the library)
- Operations must have `OperationId` set to generate client methods
- The Go file containing the spec must be syntactically valid

## Template Customization

The tool uses embedded templates for code generation. The templates are built into the binary, so no external template files are required. The templates support:

- Custom template functions for each language
- Proper type mapping between Go and target languages
- Configurable naming conventions (camelCase, snake_case, PascalCase)

## Cross-Platform Support

This tool works on all platforms without requiring CGO or external dependencies:
- **Windows**: Full support with AST parsing
- **macOS**: Full support with AST parsing  
- **Linux**: Full support with AST parsing

No need for C compilers or platform-specific build tools! 