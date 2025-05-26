package main

import (
	"reflect"

	"github.com/runpod/gopenapi"
)

// Define some alias types
type UserID string
type Status int
type Score float64
type IsActive bool

// Example spec with alias types
var ExampleSpecWithAliases = gopenapi.Spec{
	OpenAPI: "3.0.0",
	Info: gopenapi.Info{
		Title:       "Example API with Aliases",
		Description: "An API that demonstrates alias type resolution",
		Version:     "1.0.0",
	},
	Servers: gopenapi.Servers{
		{
			URL:         "https://api.example.com",
			Description: "Example server",
		},
	},
	Paths: gopenapi.Paths{
		"/users/{user_id}": gopenapi.Path{
			Get: &gopenapi.Operation{
				OperationId: "getUserById",
				Summary:     "Get a user by ID",
				Description: "Retrieve a user by their unique identifier",
				Parameters: gopenapi.Parameters{
					{
						Name:        "user_id",
						In:          gopenapi.InPath,
						Description: "User ID (string alias)",
						Required:    true,
						Schema:      gopenapi.Schema{Type: reflect.TypeOf(UserID(""))},
					},
					{
						Name:        "status",
						In:          gopenapi.InQuery,
						Description: "Status filter (int alias)",
						Required:    false,
						Schema:      gopenapi.Schema{Type: reflect.TypeOf(Status(0))},
					},
					{
						Name:        "score",
						In:          gopenapi.InHeader,
						Description: "Score header (float64 alias)",
						Required:    false,
						Schema:      gopenapi.Schema{Type: reflect.TypeOf(Score(0.0))},
					},
					{
						Name:        "is_active",
						In:          gopenapi.InQuery,
						Description: "Active filter (bool alias)",
						Required:    false,
						Schema:      gopenapi.Schema{Type: reflect.TypeOf(IsActive(false))},
					},
				},
				Responses: gopenapi.Responses{
					200: {
						Description: "User found",
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{Type: reflect.TypeOf(UserID(""))},
							},
						},
					},
				},
			},
		},
	},
}
