# GopenAPI Client Generator Examples

This directory contains examples of HTTP clients generated from OpenAPI specifications using the gopenapi tool.

## Generated Files

- `client.go` - Go HTTP client
- `client.py` - Python HTTP client  
- `example_spec.go` - Example OpenAPI specification in Go

## Usage

### Generate Demo Clients

To generate demo clients for both Go and Python:

```bash
cd cmd/gopenapi
go run . -demo -languages=go,python -output=../../examples
```

### Generate from Go Spec File (Future Feature)

Once Go file parsing is implemented, you'll be able to generate clients from a Go file containing a `gopenapi.Spec`:

```bash
cd cmd/gopenapi
go run . -spec=../../examples/example_spec.go -var=ExampleSpec -languages=go,python -output=../../examples
```

### Command Line Options

- `-demo` - Generate demo client with sample spec
- `-spec` - Go file containing the OpenAPI spec (not yet implemented)
- `-var` - Variable name containing the spec (not yet implemented)
- `-languages` - Comma-separated list of languages to generate (go,python)
- `-output` - Output directory for generated clients (default: examples)
- `-package` - Package name for generated code (default: client)

### Supported Languages

- **Go** - Generates a type-safe HTTP client with context support
- **Python** - Generates a client using the `requests` library with type hints

## Using the Generated Clients

### Go Client

To use the generated Go client, copy the `client.go` file to your project and use it as follows:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
)

// Copy the contents of client.go into your project or import as a package

func main() {
    // Create client
    c := NewClient("https://api.example.com")
    
    // Set default headers
    c.SetHeader("Authorization", "Bearer your-token")
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Get a user
    opts := &GetUserByIdOptions{
        Path: &GetUserByIdPathParams{
            Id: 123,
        },
        Query: &GetUserByIdQueryParams{
            Include: "profile",
        },
        Headers: &GetUserByIdHeaderParams{
            Authorization: "Bearer your-token",
        },
    }
    
    user, err := c.GetUserById(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User: %+v\n", user)
}
```

### Python Client

```python
from client import ClientClient, GetUserByIdPathParams, GetUserByIdQueryParams, CreateNewUserRequestBody

# Create client
client = ClientClient("https://api.example.com")

# Set default headers
client.set_header("Authorization", "Bearer your-token")

# Get a user
path = GetUserByIdPathParams(id=123)
query = GetUserByIdQueryParams(include="profile")

try:
    user = client.get_user_by_id(path=path, query=query)
    print(f"User: {user.name} ({user.email})")
except APIError as e:
    print(f"API Error: {e}")

# Create a user
body = CreateNewUserRequestBody(name="John Doe", email="john@example.com")
new_user = client.create_new_user(body=body)
print(f"Created user: {new_user.id}")
```

## Features

### Go Client Features

- Type-safe parameter and response handling
- Context support for request cancellation
- Automatic JSON marshaling/unmarshaling
- Configurable HTTP client
- Error handling with detailed error information

### Python Client Features

- Type hints for better IDE support
- Dataclasses for clean, immutable parameter and response objects
- Automatic JSON handling with proper field name conversion
- Session-based requests for connection pooling
- Configurable headers
- Exception-based error handling

## Future Enhancements

- [ ] Go file parsing to extract `gopenapi.Spec` from source files
- [ ] Additional language support (TypeScript, Java, etc.)
- [ ] Custom template support
- [ ] OpenAPI 3.1 support
- [ ] Webhook client generation
- [ ] Mock server generation 