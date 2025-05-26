package main

import (
	"github.com/runpod/gopenapi"
)

// TestStruct uses external types to test resolution
type TestStruct struct {
	Info gopenapi.Info `json:"info"`
}

var testSpec = gopenapi.Spec{
	OpenAPI: "3.0.0",
	Info: gopenapi.Info{
		Title:   "Path Resolution Test",
		Version: "1.0.0",
	},
	Paths: gopenapi.Paths{
		"/test": gopenapi.Path{
			Get: &gopenapi.Operation{
				OperationId: "getTest",
				Responses: gopenapi.Responses{
					200: {
						Description: "Test response",
						Content: gopenapi.Content{
							gopenapi.ApplicationJSON: {
								Schema: gopenapi.Schema{
									Type: gopenapi.Object[TestStruct](),
								},
							},
						},
					},
				},
			},
		},
	},
}
