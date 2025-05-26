# Schema References Example

This example demonstrates how to use schema references in the gopenapi library.

## Features Demonstrated

1. **Schema Definition in Components**: Define reusable schemas in the `components.schemas` section
2. **Schema References**: Use `$ref` to reference schemas from request bodies, responses, and parameters
3. **Nested References**: Reference schemas that themselves reference other schemas
4. **Automatic Resolution**: Schema references are resolved automatically at server startup

## Key Concepts

### Defining Schemas in Components

```go
Components: gopenapi.Components{
    Schemas: gopenapi.Schemas{
        "User": {
            Type: gopenapi.Object[User](),
        },
        "Product": {
            Type: gopenapi.Object[Product](),
        },
        // Schema that references another schema
        "UserReference": {
            Ref: "#/components/schemas/User",
        },
    },
},
```

### Using Schema References

Instead of defining the schema inline:

```go
RequestBody: gopenapi.RequestBody{
    Content: gopenapi.Content{
        "application/json": {
            Schema: gopenapi.Schema{
                Type: gopenapi.Object[User](), // Inline definition
            },
        },
    },
},
```

You can use a reference:

```go
RequestBody: gopenapi.RequestBody{
    Content: gopenapi.Content{
        "application/json": {
            Schema: gopenapi.Schema{
                Ref: "#/components/schemas/User", // Reference
            },
        },
    },
},
```

## Benefits

1. **DRY (Don't Repeat Yourself)**: Define schemas once, use them multiple times
2. **Maintainability**: Changes to a schema automatically apply everywhere it's referenced
3. **OpenAPI Compliance**: Generates proper OpenAPI 3.0 JSON with `$ref` fields
4. **Performance**: References are resolved once at startup, not on every request

## Reference Format

Schema references support JSON Pointer format for internal references:

- `#/components/schemas/SchemaName` - Reference to a schema in the components section
- `#` indicates a local reference within the same document
- The path after `#` follows JSON Pointer syntax (RFC 6901)

Currently supported internal reference patterns:
- `#/components/schemas/{name}` - Schema objects in the components section

External references (to other files) are not yet supported but may be added in future versions.

## Error Handling

The library provides comprehensive error handling for schema references:

- **Invalid format**: References that don't follow the expected format
- **Missing schemas**: References to schemas that don't exist
- **Circular references**: Schemas that reference each other in a loop

## Running the Example

```bash
go run main.go
```

Then test the endpoints:

```bash
# Create a user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# Create a product
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 19.99}'

# Get user profile (using nested reference)
curl http://localhost:8080/users/123/profile
```

## Implementation Details

- Schema references are resolved during server creation (`NewServer` or `NewServerMux`)
- Resolution happens only once at startup for maximum performance
- The original `$ref` fields are preserved in JSON serialization for OpenAPI compliance
- Resolved type information is used internally for validation and request handling 