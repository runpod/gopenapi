package generator

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/runpod/gopenapi"
)

//go:embed templates/*.tpl
var templateFS embed.FS

type TemplateData struct {
	PackageName string
	ClientName  string // For non-Go languages, this will be "Api" instead of package name
	Operations  []OperationData
}

type OperationData struct {
	OperationId       string
	Method            string
	Path              string
	Description       string
	StructName        string
	MethodName        string // Go method name (properly capitalized camelCase)
	HasPathParams     bool
	HasQueryParams    bool
	HasHeaderParams   bool
	HasRequestBody    bool
	HasResponseBody   bool
	HasAnyParams      bool   // True if any of the above params exist
	ResponseType      string // For simple types like "string", "int", etc. Empty if ResponseFields is used
	PathParams        []ParamData
	QueryParams       []ParamData
	HeaderParams      []ParamData
	RequestBodyFields []FieldData
	ResponseFields    []FieldData
}

type ParamData struct {
	Name            string
	GoName          string
	GoType          string
	ConvertToString string
	AddToParams     string
	SetHeader       string
	PathPattern     string // For path parameter replacement
}

type FieldData struct {
	Name   string
	GoName string
	GoType string
}

// GenerateClientToStdout generates a client for the specified language and outputs to stdout
func GenerateClientToStdout(spec *gopenapi.Spec, language, packageName string) error {
	// Determine template file based on language
	var templateFile string

	switch language {
	case "go":
		templateFile = "templates/go.tpl"
	case "python":
		templateFile = "templates/python.tpl"
	case "typescript":
		templateFile = "templates/typescript.tpl"
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}

	return GenerateClientToWriter(spec, os.Stdout, packageName, templateFile, language)
}

// GenerateClientForLanguage generates a client for the specified language
func GenerateClientForLanguage(spec *gopenapi.Spec, language, outputDir, packageName string) error {
	// Determine template file and output file based on language
	var templateFile, outputFile string

	switch language {
	case "go":
		templateFile = "templates/go.tpl"
		outputFile = filepath.Join(outputDir, "client.go")
	case "python":
		templateFile = "templates/python.tpl"
		outputFile = filepath.Join(outputDir, "client.py")
	case "typescript":
		templateFile = "templates/typescript.tpl"
		outputFile = filepath.Join(outputDir, "client.ts")
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}

	return GenerateClient(spec, outputFile, packageName, templateFile, language)
}

// GenerateClientToWriter generates a client from a gopenapi.Spec and writes to the provided writer
func GenerateClientToWriter(spec *gopenapi.Spec, writer io.Writer, packageName, templateFile, language string) error {
	// Load template from embedded filesystem
	tmplContent, err := templateFS.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Create template with custom functions
	tmpl, err := template.New("client").Funcs(getTemplateFuncs(language)).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Generate template data
	templateData := generateTemplateData(spec, packageName)

	// Execute template
	if err := tmpl.Execute(writer, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// GenerateClient generates a client from a gopenapi.Spec
func GenerateClient(spec *gopenapi.Spec, outputFile, packageName, templateFile, language string) error {
	// Create output directory
	outputDir := filepath.Dir(outputFile)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Use the writer-based function
	return GenerateClientToWriter(spec, outFile, packageName, templateFile, language)
}

// getTemplateFuncs returns template functions for the specified language
func getTemplateFuncs(language string) template.FuncMap {
	funcs := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
	}

	switch language {
	case "python":
		funcs["snake_case"] = toSnakeCase
		funcs["python_type"] = toPythonType
	case "typescript":
		funcs["camel_case"] = toCamelCase
		funcs["typescript_type"] = toTypeScriptType
	}

	return funcs
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	// Handle camelCase and PascalCase
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = re.ReplaceAllString(s, "${1}_${2}")

	// Handle kebab-case and other separators
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")

	return strings.ToLower(s)
}

// toPythonType converts Go types to Python types
func toPythonType(goType string) string {
	switch goType {
	case "string":
		return "str"
	case "int":
		return "int"
	case "float64":
		return "float"
	case "bool":
		return "bool"
	case "[]interface{}":
		return "List[Any]"
	default:
		return "Any"
	}
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}

	// If it's already camelCase (starts with lowercase), return as-is
	if unicode.IsLower(rune(s[0])) {
		return s
	}

	// Convert PascalCase to camelCase
	return strings.ToLower(s[:1]) + s[1:]
}

// toTypeScriptType converts Go types to TypeScript types
func toTypeScriptType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int":
		return "number"
	case "float64":
		return "number"
	case "bool":
		return "boolean"
	case "[]interface{}":
		return "any[]"
	default:
		return "any"
	}
}

func generateTemplateData(spec *gopenapi.Spec, packageName string) *TemplateData {
	var operations []OperationData

	for path, pathItem := range spec.Paths {
		methodOps := map[string]*gopenapi.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
		}

		for method, operation := range methodOps {
			if operation == nil {
				continue
			}

			if operation.OperationId == "" {
				fmt.Fprintf(os.Stderr, "Warning: Operation %s %s is missing operationId and will be skipped\n", method, path)
				continue
			}

			opData := OperationData{
				OperationId: operation.OperationId,
				Method:      method,
				Path:        path,
				Description: operation.Description,
				StructName:  ToStructName(operation.OperationId),
				MethodName:  ToMethodName(operation.OperationId),
			}

			// Process parameters
			grouped := operation.Parameters.Group()

			// Path parameters
			if len(grouped.Path) > 0 {
				opData.HasPathParams = true
				for name, schema := range grouped.Path {
					param := ParamData{
						Name:        name,
						GoName:      ToGoName(name),
						GoType:      SchemaToGoType(schema),
						PathPattern: "{" + name + "}",
					}
					param.ConvertToString = generateConvertToString(param.GoName, param.GoType)
					opData.PathParams = append(opData.PathParams, param)
				}
			}

			// Query parameters
			if len(grouped.Query) > 0 {
				opData.HasQueryParams = true
				for name, schema := range grouped.Query {
					param := ParamData{
						Name:   name,
						GoName: ToGoName(name),
						GoType: SchemaToGoType(schema),
					}
					param.AddToParams = generateAddToParams(param.GoName, param.GoType, name)
					opData.QueryParams = append(opData.QueryParams, param)
				}
			}

			// Header parameters
			if len(grouped.Header) > 0 {
				opData.HasHeaderParams = true
				for name, schema := range grouped.Header {
					param := ParamData{
						Name:   name,
						GoName: ToGoName(name),
						GoType: SchemaToGoType(schema),
					}
					param.SetHeader = generateSetHeader(param.GoName, param.GoType, name)
					opData.HeaderParams = append(opData.HeaderParams, param)
				}
			}

			// Request body
			if operation.RequestBody.Content != nil {
				opData.HasRequestBody = true
				for _, content := range operation.RequestBody.Content {
					if content.Schema.Type != nil {
						requestBodyStructName := opData.StructName + "RequestBody"
						opData.RequestBodyFields = schemaToFieldsWithName(content.Schema, requestBodyStructName)
						break
					}
				}
			}

			// Response body
			if operation.Responses != nil {
				for statusCode, response := range operation.Responses {
					if statusCode >= 200 && statusCode < 300 && response.Content != nil {
						for _, content := range response.Content {
							if content.Schema.Type != nil {
								opData.HasResponseBody = true

								// Check if this is a simple type or a struct
								if content.Schema.Type.Kind() == reflect.Struct {
									// Complex type - create response struct
									responseStructName := opData.StructName + "Response"
									opData.ResponseFields = schemaToFieldsWithName(content.Schema, responseStructName)
									opData.ResponseType = ""

								} else {
									// Simple type - no response struct needed, just use the type directly
									opData.ResponseFields = nil
									opData.ResponseType = SchemaToGoType(content.Schema)

								}
								break
							}
						}
						break
					}
				}
			}

			// Set HasAnyParams
			opData.HasAnyParams = opData.HasPathParams || opData.HasQueryParams || opData.HasHeaderParams || opData.HasRequestBody

			operations = append(operations, opData)
		}
	}

	return &TemplateData{
		PackageName: packageName,
		ClientName:  "", // Always empty - class/struct should just be "Client"
		Operations:  operations,
	}
}

func ToStructName(operationId string) string {
	// Convert operationId to PascalCase struct name
	if operationId == "" {
		return ""
	}

	// Check if it contains separators (underscore, hyphen, etc.)
	hasSeparators := strings.ContainsAny(operationId, "_-.")

	if hasSeparators {
		// Split by separators and convert to PascalCase
		parts := strings.FieldsFunc(operationId, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})

		var result strings.Builder
		for _, part := range parts {
			if len(part) > 0 {
				result.WriteString(strings.ToUpper(part[:1]))
				if len(part) > 1 {
					result.WriteString(strings.ToLower(part[1:]))
				}
			}
		}
		return result.String()
	}

	// For camelCase inputs (no separators), just capitalize the first letter
	if unicode.IsLower(rune(operationId[0])) {
		return strings.ToUpper(operationId[:1]) + operationId[1:]
	}

	// For PascalCase inputs, return as-is but ensure proper casing
	return strings.ToUpper(operationId[:1]) + strings.ToLower(operationId[1:])
}

func ToMethodName(operationId string) string {
	// Convert operationId to PascalCase method name (same as ToStructName for Go)
	if operationId == "" {
		return ""
	}

	// Check if it contains separators (underscore, hyphen, etc.)
	hasSeparators := strings.ContainsAny(operationId, "_-.")

	if hasSeparators {
		// Split by separators and convert to PascalCase
		parts := strings.FieldsFunc(operationId, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})

		var result strings.Builder
		for _, part := range parts {
			if len(part) > 0 {
				result.WriteString(strings.ToUpper(part[:1]))
				if len(part) > 1 {
					result.WriteString(strings.ToLower(part[1:]))
				}
			}
		}
		return result.String()
	}

	// For camelCase inputs (no separators), just capitalize the first letter
	if unicode.IsLower(rune(operationId[0])) {
		return strings.ToUpper(operationId[:1]) + operationId[1:]
	}

	// For PascalCase inputs, return as-is but ensure proper casing
	return strings.ToUpper(operationId[:1]) + strings.ToLower(operationId[1:])
}

func ToGoName(name string) string {
	// Convert parameter name to Go field name (PascalCase)
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}
	return result.String()
}

func SchemaToGoType(schema gopenapi.Schema) string {
	if schema.Type == nil {
		fmt.Fprintf(os.Stderr, "Warning: Schema has nil type, using interface{}\n")
		return "interface{}"
	}

	switch schema.Type {
	case gopenapi.String:
		return "string"
	case gopenapi.Integer:
		return "int"
	case gopenapi.Number:
		return "float64"
	case gopenapi.Boolean:
		return "bool"
	case gopenapi.Array:
		return "[]interface{}"
	default:
		// For other types, use the reflect.Type to determine the Go type
		return typeToGoType(schema.Type)
	}
}

func schemaToFieldsWithName(schema gopenapi.Schema, structName string) []FieldData {
	var fields []FieldData

	if schema.Type == nil || schema.Type.Kind() != reflect.Struct {
		return fields
	}

	t := schema.Type
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		goType := typeToGoType(field.Type)
		if goType == "interface{}" {
			// Use the provided struct name or fall back to reflect type name
			typeName := structName
			if typeName == "" {
				typeName = t.Name()
			}
			if typeName == "" {
				typeName = "unknown"
			}
			fmt.Fprintf(os.Stderr, "Warning: Field %s.%s has type interface{} - consider using a more specific type\n", typeName, field.Name)
		}

		fields = append(fields, FieldData{
			Name:   fieldName,
			GoName: field.Name,
			GoType: goType,
		})
	}

	return fields
}

func typeToGoType(t reflect.Type) string {
	// Handle named types (aliases) by resolving to their underlying type
	if t.PkgPath() != "" && t.Name() != "" {
		// This is a named type (alias), resolve to underlying type
		return typeToGoTypeRecursive(t, make(map[reflect.Type]bool))
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float64"
	case reflect.Bool:
		return "bool"
	case reflect.Slice:
		return "[]" + typeToGoType(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), typeToGoType(t.Elem()))
	case reflect.Ptr:
		return "*" + typeToGoType(t.Elem())
	case reflect.Struct:
		// For struct types, return interface{} as we can't generate the struct inline
		return "interface{}"
	default:
		return "interface{}"
	}
}

// typeToGoTypeRecursive resolves named types to their underlying types with cycle detection
func typeToGoTypeRecursive(t reflect.Type, visited map[reflect.Type]bool) string {
	// Prevent infinite recursion - if we've seen this type before, it's a cycle
	if visited[t] {
		return "interface{}"
	}

	// Mark this type as being processed
	visited[t] = true

	// If this is a named type, we need to check what it's based on
	if t.PkgPath() != "" && t.Name() != "" {
		// For named types, we need to look at the underlying type
		// We can't directly access the underlying type in reflection,
		// but we can create a zero value and check its type

		// Try to create a zero value and see what kind it is
		zeroValue := reflect.Zero(t)

		// Check the kind of the zero value
		switch zeroValue.Kind() {
		case reflect.String:
			return "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return "int"
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "uint"
		case reflect.Float32, reflect.Float64:
			return "float64"
		case reflect.Bool:
			return "bool"
		case reflect.Slice:
			// For slice aliases, get the element type
			elemType := t.Elem()
			if elemType.PkgPath() != "" && elemType.Name() != "" {
				// Element is also a named type
				return "[]" + typeToGoTypeRecursive(elemType, visited)
			}
			return "[]" + typeToGoType(elemType)
		case reflect.Array:
			// For array aliases, get the element type
			elemType := t.Elem()
			if elemType.PkgPath() != "" && elemType.Name() != "" {
				// Element is also a named type
				return fmt.Sprintf("[%d]%s", t.Len(), typeToGoTypeRecursive(elemType, visited))
			}
			return fmt.Sprintf("[%d]%s", t.Len(), typeToGoType(elemType))
		case reflect.Ptr:
			// For pointer aliases, get the element type
			elemType := t.Elem()
			if elemType.PkgPath() != "" && elemType.Name() != "" {
				// Element is also a named type
				return "*" + typeToGoTypeRecursive(elemType, visited)
			}
			return "*" + typeToGoType(elemType)
		case reflect.Struct:
			// Named struct type - return interface{}
			return "interface{}"
		default:
			return "interface{}"
		}
	}

	// Not a named type, use regular resolution
	return typeToGoType(t)
}

func generateConvertToString(goName, goType string) string {
	switch goType {
	case "string":
		return fmt.Sprintf("opts.Path.%s", goName)
	case "int":
		return fmt.Sprintf("strconv.Itoa(opts.Path.%s)", goName)
	case "float64":
		return fmt.Sprintf("strconv.FormatFloat(opts.Path.%s, 'f', -1, 64)", goName)
	case "bool":
		return fmt.Sprintf("strconv.FormatBool(opts.Path.%s)", goName)
	default:
		return fmt.Sprintf("fmt.Sprintf(\"%%v\", opts.Path.%s)", goName)
	}
}

func generateAddToParams(goName, goType, paramName string) string {
	switch goType {
	case "string":
		return fmt.Sprintf("if opts.Query.%s != \"\" {\n\t\tparams.Add(\"%s\", opts.Query.%s)\n\t}", goName, paramName, goName)
	case "int":
		return fmt.Sprintf("if opts.Query.%s != 0 {\n\t\tparams.Add(\"%s\", strconv.Itoa(opts.Query.%s))\n\t}", goName, paramName, goName)
	case "float64":
		return fmt.Sprintf("if opts.Query.%s != 0 {\n\t\tparams.Add(\"%s\", strconv.FormatFloat(opts.Query.%s, 'f', -1, 64))\n\t}", goName, paramName, goName)
	case "bool":
		return fmt.Sprintf("params.Add(\"%s\", strconv.FormatBool(opts.Query.%s))", paramName, goName)
	default:
		return fmt.Sprintf("if opts.Query.%s != nil {\n\t\tparams.Add(\"%s\", fmt.Sprintf(\"%%v\", opts.Query.%s))\n\t}", goName, paramName, goName)
	}
}

func generateSetHeader(goName, goType, headerName string) string {
	switch goType {
	case "string":
		return fmt.Sprintf("if opts.Headers.%s != \"\" {\n\t\treq.Header.Set(\"%s\", opts.Headers.%s)\n\t}", goName, headerName, goName)
	case "int":
		return fmt.Sprintf("if opts.Headers.%s != 0 {\n\t\treq.Header.Set(\"%s\", strconv.Itoa(opts.Headers.%s))\n\t}", goName, headerName, goName)
	case "float64":
		return fmt.Sprintf("if opts.Headers.%s != 0 {\n\t\treq.Header.Set(\"%s\", strconv.FormatFloat(opts.Headers.%s, 'f', -1, 64))\n\t}", goName, headerName, goName)
	case "bool":
		return fmt.Sprintf("req.Header.Set(\"%s\", strconv.FormatBool(opts.Headers.%s))", headerName, goName)
	default:
		return fmt.Sprintf("if opts.Headers.%s != nil {\n\t\treq.Header.Set(\"%s\", fmt.Sprintf(\"%%v\", opts.Headers.%s))\n\t}", goName, headerName, goName)
	}
}
