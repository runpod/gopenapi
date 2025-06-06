package gopenapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type Middleware interface {
	Apply(spec *Spec, operation *Operation) (MiddlewareHandler, error)
}

type MiddlewareHandler func(next http.Handler) http.Handler

type Operation struct {
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Deprecated  bool       `json:"deprecated,omitempty"`
	Parameters  Parameters `json:"parameters,omitempty"`
	Security    []Security `json:"security,omitempty"`
	// OpenAPI operation ID
	OperationId string `json:"operationId,omitempty"`
	// Request body schema for OpenAPI
	RequestBody RequestBody `json:"requestBody,omitempty"`
	// Response schemas for OpenAPI, keyed by status code
	Responses Responses    `json:"responses,omitempty"`
	Handler   http.Handler `json:"-"`
}

func (o *Operation) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if o.Summary != "" {
		m["summary"] = o.Summary
	}
	if o.Description != "" {
		m["description"] = o.Description
	}
	if len(o.Tags) > 0 {
		m["tags"] = o.Tags
	}
	if o.Deprecated {
		m["deprecated"] = o.Deprecated
	}
	if len(o.Parameters) > 0 {
		m["parameters"] = o.Parameters
	}
	if o.Security != nil {
		m["security"] = o.Security
	}
	if o.OperationId != "" {
		m["operationId"] = o.OperationId
	}
	if o.RequestBody.Content != nil {
		m["requestBody"] = o.RequestBody
	}
	if o.Responses != nil {
		m["responses"] = o.Responses
	}
	return json.Marshal(m)
}

type In string

const (
	InHeader In = "header"
	InQuery  In = "query"
	InPath   In = "path"
	InCookie In = "cookie"
)

func Type[T any]() reflect.Type {
	var v T
	return reflect.TypeOf(v)
}

var Integer = reflect.TypeOf(int(0))
var String = reflect.TypeOf(string(""))
var Number = reflect.TypeOf(float64(0))
var Boolean = reflect.TypeOf(bool(false))
var Array = reflect.TypeOf([]any{})

func Object[T any]() reflect.Type {
	return Type[T]()
}

type Schema struct {
	Type     reflect.Type   `json:"-"`
	Enum     []any          `json:"enum,omitempty"`
	Default  any            `json:"default,omitempty"`
	Example  any            `json:"example,omitempty"`
	Examples map[string]any `json:"examples,omitempty"`
	Ref      string         `json:"$ref,omitempty"`
}

func reflectTypeToJSON(t reflect.Type, schemaJSON map[string]any) error {
	switch t.Kind() {
	case reflect.String:
		schemaJSON["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schemaJSON["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schemaJSON["type"] = "number"
	case reflect.Bool:
		schemaJSON["type"] = "boolean"
	case reflect.Slice, reflect.Array:
		schemaJSON["type"] = "array"
		items := map[string]interface{}{}
		if t.Elem() != nil {
			_ = reflectTypeToJSON(t.Elem(), items)
			// Add items schema if we can determine element type
			schemaJSON["items"] = items
		}
	case reflect.Struct:
		schemaJSON["type"] = "object"

		// Add properties for struct fields
		properties := make(map[string]interface{})
		requiredProps := []string{}

		for i := range t.NumField() {
			field := t.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			// Get JSON field name from tag, fall back to struct field name
			jsonTag := field.Tag.Get("json")
			fieldName := field.Name
			if jsonTag != "" {
				// Parse the json tag to get the name part
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					fieldName = parts[0]
				}

				// Check if this field is required (no omitempty tag)
				if !strings.Contains(jsonTag, "omitempty") {
					requiredProps = append(requiredProps, fieldName)
				}
			}

			// Create schema for this field
			fieldSchema := map[string]any{}
			err := reflectTypeToJSON(field.Type, fieldSchema)
			if err != nil {
				return err
			}

			properties[fieldName] = fieldSchema
		}
		if len(properties) > 0 {
			schemaJSON["properties"] = properties
		}

		if len(requiredProps) > 0 {
			schemaJSON["required"] = requiredProps
		}
	default:
		return fmt.Errorf("unsupported type %s", t.Kind())
	}
	return nil
}

// MarshalJSON implements json.Marshaler to output proper OpenAPI schema format
func (s Schema) MarshalJSON() ([]byte, error) {

	schemaJSON := map[string]interface{}{}
	// Handle type field as string based on reflection.Type
	if s.Ref != "" {
		schemaJSON["$ref"] = s.Ref
	} else if s.Type != nil {
		err := reflectTypeToJSON(s.Type, schemaJSON)
		if err != nil {
			return nil, err
		}
	}

	// Add other fields from the original schema
	if len(s.Enum) > 0 {
		schemaJSON["enum"] = s.Enum
	}
	if s.Default != nil {
		schemaJSON["default"] = s.Default
	}
	if s.Example != nil {
		schemaJSON["example"] = s.Example
	}
	if len(s.Examples) > 0 {
		schemaJSON["examples"] = s.Examples
	}

	return json.Marshal(schemaJSON)
}

func (s Schema) Validate(value string) (any, error) {
	// If this schema has a resolved reference, use the resolved schema
	if s.Ref != "" && s.Type == nil {
		return nil, fmt.Errorf("gopenapi: unresolved schema reference %s", s.Ref)
	}

	switch s.Type {
	case String:
		return value, nil
	case Integer:
		return strconv.Atoi(value)
	case Number:
		return strconv.ParseFloat(value, 64)
	case Boolean:
		return strconv.ParseBool(value)
	default:
		v := reflect.New(s.Type).Interface()
		if err := json.Unmarshal([]byte(value), v); err != nil {
			return nil, err
		}
		return v, nil
	}
}

type Parameters []Parameter

type GroupedParameters struct {
	Path   map[string]Schema
	Query  map[string]Schema
	Header map[string]Schema
	Cookie map[string]Schema
}

var EmptyGroupedParameters = GroupedParameters{
	Path:   make(map[string]Schema),
	Query:  make(map[string]Schema),
	Header: make(map[string]Schema),
	Cookie: make(map[string]Schema),
}

func (p Parameters) Group() GroupedParameters {
	path := make(map[string]Schema)
	query := make(map[string]Schema)
	header := make(map[string]Schema)
	cookie := make(map[string]Schema)
	for _, parameter := range p {
		switch parameter.In {
		case InPath:
			path[parameter.Name] = parameter.Schema
		case InQuery:
			query[parameter.Name] = parameter.Schema
		case InHeader:
			header[parameter.Name] = parameter.Schema
		case InCookie:
			cookie[parameter.Name] = parameter.Schema
		}
	}
	return GroupedParameters{Path: path, Query: query, Header: header, Cookie: cookie}
}

type Parameter struct {
	Name        string `json:"name"`
	In          In     `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	Schema      Schema `json:"schema,omitempty"`
}

type MediaType string

const (
	ApplicationJSON MediaType = "application/json"
	ApplicationXML  MediaType = "application/xml"
	ApplicationYAML MediaType = "application/yaml"
	ApplicationYML  MediaType = "application/yml"
	TextPlain       MediaType = "text/plain"
	TextHTML        MediaType = "text/html"
	TextXML         MediaType = "text/xml"
	TextCSV         MediaType = "text/csv"
	TextJSON        MediaType = "text/json"
	TextYAML        MediaType = "text/yaml"
	TextMarkdown    MediaType = "text/markdown"
	ImagePNG        MediaType = "image/png"
	ImageJPEG       MediaType = "image/jpeg"
	ImageGIF        MediaType = "image/gif"
	ImageSVG        MediaType = "image/svg+xml"
	ImageWEBM       MediaType = "image/webp"
	Image           MediaType = "image/*"
	AudioMP3        MediaType = "audio/mpeg"
	AudioMP4        MediaType = "audio/mp4"
	AudioOGG        MediaType = "audio/ogg"
	AudioWAV        MediaType = "audio/wav"
	AudioWEBM       MediaType = "audio/webm"
	VideoMP4        MediaType = "video/mp4"
	VideoOGG        MediaType = "video/ogg"
	VideoWEBM       MediaType = "video/webm"
	VideoMPEG       MediaType = "video/mpeg"
	VideoMPG        MediaType = "video/mpeg"
)

type Content = map[MediaType]struct {
	Schema Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Content     Content `json:"content,omitempty"`
}

type Ref string

type Tags []string

type Path struct {
	Summary     string     `json:"summary"`
	Description string     `json:"description"`
	Tags        Tags       `json:"tags"`
	Servers     Servers    `json:"servers,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
}

type Paths map[string]Path

type Responses = map[int]struct {
	Description string  `json:"description,omitempty"`
	Content     Content `json:"content,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type Contact struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Email string `json:"email"`
}

type Servers []struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type SecuritySchemeType string

const (
	APIKey        SecuritySchemeType = "apiKey"
	HTTP          SecuritySchemeType = "http"
	OAuth2        SecuritySchemeType = "oauth2"
	OpenIDConnect SecuritySchemeType = "openIdConnect"
)

type Scheme string

const (
	BearerScheme Scheme = "bearer"
	BasicScheme  Scheme = "basic"
)

type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

type SecurityScheme struct {
	Type    SecuritySchemeType `json:"type,omitempty"`
	In      In                 `json:"in,omitempty"`
	Scheme  Scheme             `json:"scheme,omitempty"`
	Flows   *OAuthFlows        `json:"flows,omitempty"`
	Handler MiddlewareHandler  `json:"-"`
}

type SecuritySchemes map[string]SecurityScheme
type Schemas map[string]Schema

type Components struct {
	SecuritySchemes SecuritySchemes `json:"securitySchemes,omitempty"`
	Schemas         Schemas         `json:"schemas,omitempty"`
}

type Security map[string][]string

type SecurityHandler func(w http.ResponseWriter, r *http.Request) error

type DefaultSecurityMiddleware struct {
	spec *Spec
}

func (s *DefaultSecurityMiddleware) Apply(spec *Spec, operation *Operation) (MiddlewareHandler, error) {
	security := operation.Security
	if security == nil {
		security = spec.Security
	}
	handler := http.Handler(operation.Handler)

	for _, security := range security {
		for name := range security {
			maybeScheme, ok := spec.Components.SecuritySchemes[name]
			if !ok || maybeScheme.Handler == nil {
				return nil, fmt.Errorf("gopenapi: security scheme %s not found", name)
			}
			handler = maybeScheme.Handler(handler)
		}
	}
	return func(next http.Handler) http.Handler {
		return handler
	}, nil
}

var NoSecurity []Security = []Security{}

type Spec struct {
	OpenAPI              string               `json:"openapi"`
	Info                 Info                 `json:"info"`
	Paths                Paths                `json:"paths"`
	Servers              Servers              `json:"servers,omitempty"`
	Components           Components           `json:"components"`
	Security             []Security           `json:"security,omitempty"`
	ValidationMiddleware ValidationMiddleware `json:"-"`
	SecurityMiddleware   Middleware           `json:"-"`
}

type Server struct {
	http.Server
	Spec Spec `json:"-"`
}

func formatPattern(method, host, pattern string) string {
	url, err := url.Parse(host)
	if err != nil {
		panic(err)
	}
	pattern = fmt.Sprintf("%s %s%s", method, url.Host, pattern)
	return pattern
}

func handle(spec *Spec, operation *Operation) (http.HandlerFunc, error) {
	handler := http.Handler(operation.Handler)
	for _, middleware := range []Middleware{spec.ValidationMiddleware, spec.SecurityMiddleware} {
		if middleware == nil {
			continue
		}
		middlewareHandler, err := middleware.Apply(spec, operation)
		if err != nil {
			return nil, err
		}
		handler = middlewareHandler(handler)
	}
	handlerContextValue := RequestContext{
		spec:      spec,
		operation: operation,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Add both spec and operation to the request context in a single chain (preserving existing context)
		ctx := context.WithValue(r.Context(), RequestContextKey, handlerContextValue)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}, nil
}

func NewServerMux(spec *Spec) (http.Handler, error) {
	// resolve references first
	if err := resolveRefs(spec); err != nil {
		return nil, fmt.Errorf("failed to resolve schema references: %w", err)
	}

	mux := http.NewServeMux()
	hosts := make([]string, len(spec.Servers))
	if spec.SecurityMiddleware == nil {
		spec.SecurityMiddleware = &DefaultSecurityMiddleware{spec: spec}
	}
	if spec.ValidationMiddleware == nil {
		spec.ValidationMiddleware = &DefaultValidationMiddleware{}
	}
	for i, server := range spec.Servers {
		hosts[i] = server.URL
	}
	for pattern, path := range spec.Paths {
		overrideHosts := hosts
		if path.Servers != nil {
			clear(overrideHosts)
			for _, server := range path.Servers {
				overrideHosts = append(overrideHosts, server.URL)
			}
		}

		for _, host := range overrideHosts {
			for method, operation := range map[string]*Operation{
				http.MethodGet:     path.Get,
				http.MethodPost:    path.Post,
				http.MethodPut:     path.Put,
				http.MethodDelete:  path.Delete,
				http.MethodPatch:   path.Patch,
				http.MethodHead:    path.Head,
				http.MethodOptions: path.Options,
			} {
				if operation != nil && operation.Handler != nil {
					handler, err := handle(spec, operation)
					if err != nil {
						return nil, err
					}
					mux.HandleFunc(formatPattern(method, host, pattern), handler)
				}
			}
		}
	}

	return mux, nil
}

func NewServer(spec *Spec, port string) (*Server, error) {
	handler, err := NewServerMux(spec)
	if err != nil {
		return nil, err
	}
	// resolveRefs(spec)
	baseCtx := context.WithValue(context.Background(), RequestContextKey, RequestContext{
		spec:      spec,
		operation: nil,
	})
	return &Server{
		Server: http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: handler,
			BaseContext: func(l net.Listener) context.Context {
				return baseCtx
			},
		},
		Spec: *spec,
	}, nil
}

func Serve(ctx context.Context, listener net.Listener, spec *Spec) error {
	mux, err := NewServerMux(spec)
	if err != nil {
		return err
	}
	return http.Serve(listener, mux)
}

type key[T any] struct{}

type RequestContext struct {
	spec      *Spec
	operation *Operation
}

var (
	RequestContextKey = key[RequestContext]{}
)

func specFromContext(ctx context.Context) (*Spec, bool) {
	requestCtx, ok := ctx.Value(RequestContextKey).(RequestContext)
	if !ok {
		return nil, false
	}
	return requestCtx.spec, true
}

func SpecFromRequest(r *http.Request) (Spec, bool) {
	spec, ok := specFromContext(r.Context())
	if !ok {
		return Spec{}, false
	}
	return *spec, true
}

func OperationFromRequest(r *http.Request) (*Operation, bool) {
	requestCtx, ok := r.Context().Value(RequestContextKey).(RequestContext)
	if !ok {
		return nil, false
	}
	return requestCtx.operation, true
}

func WriteResponse(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// resolveRefs resolves all schema references in the spec
func resolveRefs(spec *Spec) error {
	// Track which schemas are being resolved to detect circular references
	resolving := make(map[string]bool)

	// Resolve references in operation parameters
	for pathPattern, path := range spec.Paths {
		operations := []*Operation{
			path.Get, path.Post, path.Put, path.Delete,
			path.Patch, path.Head, path.Options, path.Trace,
		}

		for _, operation := range operations {
			if operation == nil {
				continue
			}

			// Resolve parameter schema references
			for i := range operation.Parameters {
				if err := resolveSchemaRefWithTracking(&operation.Parameters[i].Schema, spec, resolving); err != nil {
					return fmt.Errorf("gopenapi.resolveRefs: failed to resolve parameter schema ref in %s: %w", pathPattern, err)
				}
			}

			// Resolve request body schema references
			for mediaType, content := range operation.RequestBody.Content {
				if err := resolveSchemaRefWithTracking(&content.Schema, spec, resolving); err != nil {
					return fmt.Errorf("gopenapi.resolveRefs: failed to resolve request body schema ref for %s in %s: %w", mediaType, pathPattern, err)
				}
				// Update the content in the map since we modified the schema
				operation.RequestBody.Content[mediaType] = content
			}

			// Resolve response schema references
			for statusCode, response := range operation.Responses {
				for mediaType, content := range response.Content {
					if err := resolveSchemaRefWithTracking(&content.Schema, spec, resolving); err != nil {
						return fmt.Errorf("gopenapi.resolveRefs: failed to resolve response schema ref for status %d, media type %s in %s: %w", statusCode, mediaType, pathPattern, err)
					}
					// Update the content in the map since we modified the schema
					response.Content[mediaType] = content
				}
				// Update the response in the map since we modified the content
				operation.Responses[statusCode] = response
			}
		}
	}

	return nil
}

// resolveSchemaRefWithTracking resolves a single schema reference with circular reference detection
func resolveSchemaRefWithTracking(schema *Schema, spec *Spec, resolving map[string]bool) error {
	if schema.Ref == "" {
		return nil
	}

	// Only handle local references (starting with #)
	if !strings.HasPrefix(schema.Ref, "#") {
		return fmt.Errorf("gopenapi.resolveSchemaRefWithTracking: external references not supported: %s", schema.Ref)
	}

	// Check for circular reference
	if resolving[schema.Ref] {
		return fmt.Errorf("gopenapi.resolveSchemaRefWithTracking: circular reference detected for: %s", schema.Ref)
	}

	// Mark this reference as being resolved
	resolving[schema.Ref] = true
	defer delete(resolving, schema.Ref)

	// Resolve the reference using JSON Pointer
	referencedSchema, err := resolveJSONPointer(spec, schema.Ref)
	if err != nil {
		return fmt.Errorf("failed to resolve reference %s: %w", schema.Ref, err)
	}

	// Recursively resolve the referenced schema if it also has a reference
	if err := resolveSchemaRefWithTracking(&referencedSchema, spec, resolving); err != nil {
		return fmt.Errorf("failed to resolve nested reference in %s: %w", schema.Ref, err)
	}

	// Copy the resolved schema properties to the current schema
	// Keep the original Ref for JSON serialization, but copy the Type for validation
	schema.Type = referencedSchema.Type
	if len(referencedSchema.Enum) > 0 {
		schema.Enum = referencedSchema.Enum
	}
	if referencedSchema.Default != nil {
		schema.Default = referencedSchema.Default
	}
	if referencedSchema.Example != nil {
		schema.Example = referencedSchema.Example
	}
	if len(referencedSchema.Examples) > 0 {
		schema.Examples = referencedSchema.Examples
	}

	return nil
}

// resolveJSONPointer resolves a JSON Pointer reference within the spec
func resolveJSONPointer(spec *Spec, ref string) (Schema, error) {
	// Remove the # prefix
	pointer := strings.TrimPrefix(ref, "#")
	if pointer == "" {
		return Schema{}, fmt.Errorf("empty JSON pointer")
	}

	// Split the pointer into segments
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if len(segments) == 0 {
		return Schema{}, fmt.Errorf("invalid JSON pointer: %s", ref)
	}

	// Navigate through the spec based on the pointer segments
	switch segments[0] {
	case "components":
		if len(segments) < 3 {
			return Schema{}, fmt.Errorf("invalid components reference: %s", ref)
		}
		switch segments[1] {
		case "schemas":
			schemaName := segments[2]
			if referencedSchema, exists := spec.Components.Schemas[schemaName]; exists {
				return referencedSchema, nil
			}
			return Schema{}, fmt.Errorf("schema not found: %s", schemaName)
		default:
			return Schema{}, fmt.Errorf("unsupported components reference: %s", ref)
		}
	default:
		return Schema{}, fmt.Errorf("unsupported JSON pointer root: %s", segments[0])
	}
}
