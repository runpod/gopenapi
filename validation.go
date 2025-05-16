package gopenapi

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

type ValidationMiddleware interface {
	Apply(spec *Spec, operation *Operation) (MiddlewareHandler, error)
	ValidatePathValue(operation *Operation, name string, value string) (any, error)
	ValidateBody(operation *Operation, r *http.Request) (any, error)
	ValidateQueryValue(operation *Operation, name string, value string) (any, error)
	ValidateHeaderValue(operation *Operation, name string, value string) (any, error)
	ValidateCookieValue(operation *Operation, name string, value string) (any, error)
	ValidateFormValue(operation *Operation, name string, value string) (any, error)
	ValidateRequest(operation *Operation, r *http.Request) (any, error)
}

type DefaultValidationMiddleware struct {
}

func (v *DefaultValidationMiddleware) Apply(spec *Spec, operation *Operation) (MiddlewareHandler, error) {
	return func(next http.Handler) http.Handler {
		return next
	}, nil
}

func validate(group map[string]Schema, name string, value string) (any, error) {
	schema, ok := group[name]
	if !ok {
		return nil, fmt.Errorf("gopenapi: missing path parameter %s", name)
	}
	return schema.Validate(value)
}

func (v *DefaultValidationMiddleware) ValidatePathValue(operation *Operation, name string, value string) (any, error) {
	return validate(operation.Parameters.group().path, name, value)
}

func (v *DefaultValidationMiddleware) ValidateBody(operation *Operation, request *http.Request) (any, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	contentType := request.Header.Get("Content-Type")
	if contentType == "" {
		if operation.RequestBody.Content != nil {
			return nil, fmt.Errorf("gopenapi: missing content type for request body")
		}
		if len(body) == 0 {
			return nil, nil
		}
		return string(body), nil
	}
	content, ok := operation.RequestBody.Content[MediaType(contentType)]
	if !ok {
		return nil, fmt.Errorf("gopenapi: missing schema for content type %s", contentType)
	}

	return content.Schema.Validate(string(body))
}

func (v *DefaultValidationMiddleware) ValidateQueryValue(operation *Operation, name string, value string) (any, error) {
	return validate(operation.Parameters.group().query, name, value)
}

func (v *DefaultValidationMiddleware) ValidateHeaderValue(operation *Operation, name string, value string) (any, error) {
	return validate(operation.Parameters.group().header, name, value)
}

func (v *DefaultValidationMiddleware) ValidateCookieValue(operation *Operation, name string, value string) (any, error) {
	return validate(operation.Parameters.group().cookie, name, value)
}

func (v *DefaultValidationMiddleware) ValidateFormValue(operation *Operation, name string, value string) (any, error) {
	if operation.RequestBody.Content == nil {
		return nil, fmt.Errorf("gopenapi: no request body defined for form data validation")
	}

	for _, mediaTypeContent := range operation.RequestBody.Content {
		if mediaTypeContent.Schema.Type != nil && mediaTypeContent.Schema.Type.Kind() == reflect.Struct {
			if _, ok := mediaTypeContent.Schema.Type.FieldByName(name); ok {
				// This is a very basic interpretation. A proper implementation would
				// check specific form content types and validate the form data accordingly.
				return nil, fmt.Errorf("gopenapi: ValidateFormValue not fully implemented for complex forms. Parameter %s", name)
			}
		}
	}
	return nil, fmt.Errorf("gopenapi: form field %s schema not found or complex form validation not implemented", name)
}

func (v *DefaultValidationMiddleware) ValidateRequest(operation *Operation, r *http.Request) (any, error) {
	groupedParams := operation.Parameters.group()
	if groupedParams.query != nil {
		for name := range groupedParams.query {
			queryValue := r.URL.Query().Get(name)
			_, err := v.ValidateQueryValue(operation, name, queryValue)
			if err != nil {
				return nil, fmt.Errorf("query parameter validation failed for '%s': %w", name, err)
			}
		}
	}

	if groupedParams.header != nil {
		for name := range groupedParams.header {
			headerValue := r.Header.Get(name)
			_, err := v.ValidateHeaderValue(operation, name, headerValue)
			if err != nil {
				return nil, fmt.Errorf("header parameter validation failed for '%s': %w", name, err)
			}
		}
	}

	if groupedParams.cookie != nil {
		for name := range groupedParams.cookie {
			cookie, err := r.Cookie(name)
			cookieValue := ""
			if err == nil {
				cookieValue = cookie.Value
			} else if err != http.ErrNoCookie {
				return nil, fmt.Errorf("could not retrieve cookie '%s': %w", name, err)
			}
			_, errVal := v.ValidateCookieValue(operation, name, cookieValue)
			if errVal != nil {
				return nil, fmt.Errorf("cookie parameter validation failed for '%s': %w", name, errVal)
			}
		}
	}

	if operation.RequestBody.Content != nil {
		// We need to be careful here. Reading the body consumes it.
		// If validation reads the body, the actual handler won't be able to.
		// A common pattern is to read the body, validate, and then replace r.Body with a new reader.
		// For simplicity in this example, ValidateBody is assumed to be callable,
		// but in a real middleware chain, body handling needs care.
		// The current ValidateBody reads the whole body.
		// This ValidateRequest is more of a high-level orchestrator.
		// We will skip calling ValidateBody here to avoid consuming the body twice if Apply is used.
		// The Apply method should chain these validations appropriately.
		// For now, this method can return a struct of validated parts or just nil,error.
	}

	return nil, nil
}

func ValidateRequestPathValue[T any](r *http.Request, name string, into *T) error {
	spec, ok := SpecFromRequest(r)
	if !ok {
		return fmt.Errorf("gopenapi: no spec for request")
	}
	operation, ok := OperationFromRequest(r)
	if !ok {
		return fmt.Errorf("gopenapi: no operation for request")
	}
	maybeValue, err := spec.ValidationMiddleware.ValidatePathValue(operation, name, r.PathValue(name))
	if err != nil {
		return err
	}
	value, ok := maybeValue.(T)
	if !ok {
		return fmt.Errorf("gopenapi: invalid validated path value type expected %T, got %T", into, maybeValue)
	}
	*into = value
	return nil
}

func ValidateRequestPathValues[T any](r *http.Request, into *T) error {
	valueType := reflect.TypeOf(*into)
	valuesValue := reflect.ValueOf(into).Elem()
	if valueType.Kind() != reflect.Struct {
		return fmt.Errorf("gopenapi: invalid validated path value type %T", into)
	}
	spec, ok := SpecFromRequest(r)
	if !ok {
		return fmt.Errorf("gopenapi: no spec for request")
	}
	operation, ok := OperationFromRequest(r)
	if !ok {
		return fmt.Errorf("gopenapi: no operation for request")
	}
	for i := range valueType.NumField() {
		field := valueType.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")
		fieldName := field.Name
		if len(tag) > 0 && tag[0] != "-" {
			fieldName = tag[0]
		}
		anyValue, err := spec.ValidationMiddleware.ValidatePathValue(operation, fieldName, r.PathValue(fieldName))
		if err != nil {
			return err
		}
		canSet := valuesValue.Field(i).CanSet()
		if !canSet {
			return fmt.Errorf("gopenapi: field %s is not settable", fieldName)
		}
		if !field.Type.AssignableTo(reflect.TypeOf(anyValue)) {
			return fmt.Errorf("gopenapi: field %s is not assignable to %T", fieldName, anyValue)
		}
		valuesValue.Field(i).Set(reflect.ValueOf(anyValue))
	}
	return nil
}

func ValidateRequestBody[T any](r *http.Request, into *T) error {
	spec, ok := SpecFromRequest(r)
	if !ok {
		return fmt.Errorf("gopenapi: no spec for request")
	}
	operation, ok := OperationFromRequest(r)
	if !ok {
		return fmt.Errorf("gopenapi: no operation for request in spec")
	}
	maybeValue, err := spec.ValidationMiddleware.ValidateBody(operation, r)
	if err != nil {
		return err
	}
	value, ok := maybeValue.(*T)
	if !ok {
		return fmt.Errorf("gopenapi: invalid validated body type expected %T, got %T", into, maybeValue)
	}
	*into = *value
	return nil
}
