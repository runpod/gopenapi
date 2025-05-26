package spec

import (
	"net/http"

	"github.com/runpod/gopenapi"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ExampleSpec is a sample OpenAPI specification
var ExampleSpec = gopenapi.Spec{
	OpenAPI: "3.0.0",
	Info: gopenapi.Info{
		Title:       "Example API",
		Description: "A simple example API",
		Version:     "1.0.0",
	},
	Servers: gopenapi.Servers{
		{
			URL:         "https://api.example.com",
			Description: "Production server",
		},
	},
	Paths: gopenapi.Paths{
		"/users/{id}": gopenapi.Path{
			Get: &gopenapi.Operation{
				OperationId: "getUserById",
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
					{
						Name:        "Authorization",
						In:          gopenapi.InHeader,
						Description: "Bearer token",
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
					// Sample handler implementation
				}),
			},
		},
		"/test-schema": gopenapi.Path{
			Get: &gopenapi.Operation{
				OperationId: "testSchema",
				Summary:     "Test schema",
				Description: "Test schema",
				Responses: gopenapi.Responses{
					200: {
						Description: "OK",
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: gopenapi.Object[gopenapi.Schema]()},
							},
						},
					},
				},
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
		},
		"/users": gopenapi.Path{
			Get: &gopenapi.Operation{
				OperationId: "listAllUsers",
				Summary:     "List all users",
				Description: "Retrieve a list of all users",
				Responses: gopenapi.Responses{
					200: {
						Description: "List of users",
					},
				},
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Sample handler implementation
				}),
			},
			Post: &gopenapi.Operation{
				OperationId: "createNewUser",
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
					// Sample handler implementation
				}),
			},
		},
	},
}
