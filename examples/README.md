# GopenAPI Client Generator Examples

This directory contains examples of HTTP clients generated from OpenAPI specifications using the gopenapi tool.

## Generated Files

- `client.go` - Go HTTP client
- `client.py` - Python HTTP client  
- `client.ts` - TypeScript HTTP client
- `spec/spec.go` - Example OpenAPI specification in Go

## Usage

### Generate Clients from Example Spec

To generate clients from the example specification:

```bash
# Generate clients for all supported languages
gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -languages go,python,typescript -output ./examples/clients

# Generate only Go client
gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -languages go -output ./examples/clients

# Generate only Python client  
gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -languages python -output ./examples/clients

# Generate only TypeScript client
gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -languages typescript -output ./examples/clients

# Generate to stdout (single language only)
gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -languages go
```

### Generate OpenAPI JSON

To generate OpenAPI JSON specification from the example:

```bash
# Generate to file
gopenapi generate spec -spec examples/spec/spec.go -var ExampleSpec -output openapi.json

# Generate to stdout
gopenapi generate spec -spec examples/spec/spec.go -var ExampleSpec
```

### Command Line Options

**For `generate client`:**
- `-spec` - Go file containing the OpenAPI spec (required)
- `-var` - Variable name containing the spec (required)
- `-languages` - Comma-separated list of languages to generate (default: go)
- `-output` - Output directory for generated clients (if empty, outputs to stdout)
- `-package` - Package name for generated code (default: client)

**For `generate spec`:**
- `-spec` - Go file containing the OpenAPI spec (required)
- `-var` - Variable name containing the spec (required)
- `-output` - Output file for OpenAPI JSON (if empty, outputs to stdout)

### Supported Languages

- **Go** - Generates a type-safe HTTP client with context support
- **Python** - Generates a client using the `requests` library with type hints
- **TypeScript** - Generates a modern async/await client with full type safety

## Using the Generated Clients

### Go Client

To use the generated Go client:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "your-module/client" // Replace with your generated client import
)

func main() {
    // Create client
    client := client.NewClient("https://api.example.com")
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Get a user with type-safe parameters
    user, err := client.GetUserById(ctx, client.GetUserByIdOptions{
        Path: client.GetUserByIdPath{
            Id: 123,
        },
        Query: client.GetUserByIdQuery{
            Include: "profile",
        },
        Headers: client.GetUserByIdHeaders{
            Authorization: "Bearer your-token",
        },
    })
    
    if err != nil {
        // Handle structured error
        if apiErr, ok := err.(*client.Error); ok {
            fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        } else {
            log.Fatal("Network error:", err)
        }
        return
    }
    
    fmt.Printf("User: %+v\n", user)
}
```

### Python Client

```python
from client import Client, APIError

# Create client
client = Client("https://api.example.com")

# Set default headers
client.set_header("Authorization", "Bearer your-token")

try:
    # Get a user with type-safe parameters
    user = client.get_user_by_id(
        path=GetUserByIdPathParams(id=123),
        query=GetUserByIdQueryParams(include="profile")
    )
    print(f"User: {user}")
    
    # Create a user
    new_user = client.create_new_user(
        body=CreateNewUserRequestBody(name="John Doe", email="john@example.com")
    )
    print(f"Created user: {new_user}")
    
except APIError as e:
    print(f"API Error {e.status_code}: {e.message}")
```

### TypeScript Client

```typescript
import { Client, ApiError } from './client';

// Create client with configuration
const client = new Client({
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
  
  // Create a user
  const newUser = await client.createNewUser(
    {}, // path params (empty for this endpoint)
    {}, // query params (empty for this endpoint)
    {}, // headers (optional)
    { name: 'John Doe', email: 'john@example.com' } // request body
  );
  
  console.log('Created user:', newUser);
  
} catch (error) {
  if (error instanceof ApiError) {
    console.error(`API Error ${error.statusCode}: ${error.message}`);
  } else {
    console.error('Network error:', error);
  }
}
```

## Features

### Go Client Features

- Type-safe parameter and response handling
- Context support for request cancellation
- Automatic JSON marshaling/unmarshaling
- Configurable HTTP client
- Structured error handling with detailed error information
- Support for path, query, and header parameters
- Request body validation

### Python Client Features

- Type hints for better IDE support
- Dataclasses for clean, immutable parameter and response objects
- Automatic JSON handling with proper field name conversion
- Session-based requests for connection pooling
- Configurable headers
- Exception-based error handling

### TypeScript Client Features

- Full TypeScript type safety with interfaces for all parameters and responses
- Modern async/await API using fetch
- Configurable timeout and headers
- Automatic JSON serialization/deserialization
- Proper error handling with custom ApiError class
- Support for both Node.js and browser environments

### Error Handling

All generated clients include comprehensive error handling:

- **Structured Error Types**: Custom error types with HTTP status codes, messages, and raw response bodies
- **Consistent Error Handling**: All operations handle HTTP errors consistently across languages
- **Detailed Error Information**: Access to status codes, error messages, and raw response data
- **Type-Safe Error Responses**: Proper typing for error scenarios in all supported languages

## Example Specification

The example specification in `spec/spec.go` demonstrates:

- Multiple HTTP methods (GET, POST)
- Path parameters with type conversion
- Query and header parameters
- Request and response body handling
- Named types with primitive underlying types (like `type ID string`)
- Complex struct types with JSON tags
- Multiple response status codes

This provides a comprehensive example of how to structure your OpenAPI specifications in Go code. 