package main

import (
	"embed"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/runpod/gopenapi"
	"golang.org/x/tools/go/packages"
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

func main() {
	var (
		specFile    = flag.String("spec", "", "Go file containing the OpenAPI spec (required)")
		specVar     = flag.String("var", "", "Variable name containing the spec (required, e.g., 'ExampleSpec')")
		outputDir   = flag.String("output", ".", "Output directory for generated clients")
		packageName = flag.String("package", "client", "Package name for generated code")
		languages   = flag.String("languages", "go", "Comma-separated list of languages to generate (go,python,typescript)")
	)
	flag.Parse()

	if *specFile == "" || *specVar == "" {
		log.Fatal("Both -spec and -var flags are required")
	}

	spec, err := parseSpecFromFile(*specFile, *specVar)
	if err != nil {
		log.Fatalf("Failed to parse spec from file: %v", err)
	}

	// Parse languages
	langs := strings.Split(*languages, ",")
	for i, lang := range langs {
		langs[i] = strings.TrimSpace(lang)
	}

	// Generate clients for each language
	for _, lang := range langs {
		err := generateClientForLanguage(&spec, lang, *outputDir, *packageName)
		if err != nil {
			log.Fatalf("Failed to generate %s client: %v", lang, err)
		}
		fmt.Printf("Generated %s client in %s\n", lang, *outputDir)
	}
}

// parseSpecFromFile parses a Go file and extracts the specified gopenapi.Spec variable
func parseSpecFromFile(filename, varName string) (gopenapi.Spec, error) {
	return parseSpecViaPackages(filename, varName)
}

// parseSpecViaPackages parses the Go file using go/packages and extracts the gopenapi.Spec
func parseSpecViaPackages(filename, varName string) (gopenapi.Spec, error) {
	// Get the directory containing the file
	dir := filepath.Dir(filename)

	// Load the package
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Dir:   dir,
		Fset:  token.NewFileSet(),
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return gopenapi.Spec{}, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return gopenapi.Spec{}, fmt.Errorf("no packages found")
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return gopenapi.Spec{}, fmt.Errorf("package has errors: %v", pkg.Errors)
	}

	// Find the file in the package
	var targetFile *ast.File
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return gopenapi.Spec{}, fmt.Errorf("failed to get absolute path: %w", err)
	}

	for _, file := range pkg.Syntax {
		filePos := pkg.Fset.Position(file.Pos()).Filename
		absFilePos, err := filepath.Abs(filePos)
		if err != nil {
			continue
		}
		if absFilePos == absFilename {
			targetFile = file
			break
		}
	}

	if targetFile == nil {
		return gopenapi.Spec{}, fmt.Errorf("file %s not found in package", filename)
	}

	// Find the variable declaration and extract its value
	var specLiteral *ast.CompositeLit
	found := false

	ast.Inspect(targetFile, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range valueSpec.Names {
						if name.Name == varName {
							if i < len(valueSpec.Values) {
								if compLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
									specLiteral = compLit
									found = true
									return false
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	if !found {
		return gopenapi.Spec{}, fmt.Errorf("variable %s not found in file %s", varName, filename)
	}

	if specLiteral == nil {
		return gopenapi.Spec{}, fmt.Errorf("variable %s is not a composite literal", varName)
	}

	// Parse the composite literal into a gopenapi.Spec with type resolution
	spec, err := parseSpecFromASTWithTypes(specLiteral, pkg)
	if err != nil {
		return gopenapi.Spec{}, fmt.Errorf("failed to parse spec from AST: %w", err)
	}

	return spec, nil
}

// parseSpecFromASTWithTypes converts an AST composite literal to a gopenapi.Spec with type resolution
func parseSpecFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Spec, error) {
	spec := gopenapi.Spec{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "OpenAPI":
					if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
						spec.OpenAPI = strings.Trim(basicLit.Value, `"`)
					}
				case "Info":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						info, err := parseInfoFromAST(compLit, pkg.Fset, nil)
						if err != nil {
							return spec, fmt.Errorf("failed to parse Info: %w", err)
						}
						spec.Info = info
					}
				case "Servers":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						servers, err := parseServersFromAST(compLit, pkg.Fset, nil)
						if err != nil {
							return spec, fmt.Errorf("failed to parse Servers: %w", err)
						}
						spec.Servers = servers
					}
				case "Paths":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						paths, err := parsePathsFromASTWithTypes(compLit, pkg)
						if err != nil {
							return spec, fmt.Errorf("failed to parse Paths: %w", err)
						}
						spec.Paths = paths
					}
				}
			}
		}
	}

	return spec, nil
}

// parsePathsFromASTWithTypes parses gopenapi.Paths from AST with type resolution
func parsePathsFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Paths, error) {
	paths := make(gopenapi.Paths)

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			// Get the path string
			var pathStr string
			if basicLit, ok := kv.Key.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
				pathStr = strings.Trim(basicLit.Value, `"`)
			}

			// Parse the path item
			if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
				pathItem, err := parsePathItemFromASTWithTypes(compLit, pkg)
				if err != nil {
					return paths, fmt.Errorf("failed to parse path item for %s: %w", pathStr, err)
				}
				paths[pathStr] = pathItem
			}
		}
	}

	return paths, nil
}

// parsePathItemFromASTWithTypes parses gopenapi.Path from AST with type resolution
func parsePathItemFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Path, error) {
	pathItem := gopenapi.Path{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				if unaryExpr, ok := kv.Value.(*ast.UnaryExpr); ok && unaryExpr.Op == token.AND {
					if compLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
						operation, err := parseOperationFromASTWithTypes(compLit, pkg)
						if err != nil {
							return pathItem, fmt.Errorf("failed to parse operation %s: %w", ident.Name, err)
						}

						switch strings.ToUpper(ident.Name) {
						case "GET":
							pathItem.Get = &operation
						case "POST":
							pathItem.Post = &operation
						case "PUT":
							pathItem.Put = &operation
						case "DELETE":
							pathItem.Delete = &operation
						case "PATCH":
							pathItem.Patch = &operation
						case "HEAD":
							pathItem.Head = &operation
						case "OPTIONS":
							pathItem.Options = &operation
						}
					}
				}
			}
		}
	}

	return pathItem, nil
}

// parseOperationFromASTWithTypes parses gopenapi.Operation from AST with type resolution
func parseOperationFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Operation, error) {
	operation := gopenapi.Operation{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "OperationId", "Summary", "Description":
					if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
						value := strings.Trim(basicLit.Value, `"`)
						switch ident.Name {
						case "OperationId":
							operation.OperationId = value
						case "Summary":
							operation.Summary = value
						case "Description":
							operation.Description = value
						}
					}
				case "Parameters":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						params, err := parseParametersFromASTWithTypes(compLit, pkg)
						if err != nil {
							return operation, fmt.Errorf("failed to parse parameters: %w", err)
						}
						operation.Parameters = params
					}
				case "Responses":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						responses, err := parseResponsesFromASTWithTypes(compLit, pkg)
						if err != nil {
							return operation, fmt.Errorf("failed to parse responses: %w", err)
						}
						operation.Responses = responses
					}
				case "RequestBody":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						requestBody, err := parseRequestBodyFromASTWithTypes(compLit, pkg)
						if err != nil {
							return operation, fmt.Errorf("failed to parse request body: %w", err)
						}
						operation.RequestBody = requestBody
					}
				case "Handler":
					// Skip handler parsing for now as it's complex and not needed for client generation
					operation.Handler = nil
				}
			}
		}
	}

	return operation, nil
}

// parseParametersFromASTWithTypes parses gopenapi.Parameters from AST with type resolution
func parseParametersFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Parameters, error) {
	var params gopenapi.Parameters

	for _, elt := range lit.Elts {
		if compLit, ok := elt.(*ast.CompositeLit); ok {
			param := gopenapi.Parameter{}
			for _, paramElt := range compLit.Elts {
				if kv, ok := paramElt.(*ast.KeyValueExpr); ok {
					if ident, ok := kv.Key.(*ast.Ident); ok {
						switch ident.Name {
						case "Name", "Description":
							if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
								value := strings.Trim(basicLit.Value, `"`)
								switch ident.Name {
								case "Name":
									param.Name = value
								case "Description":
									param.Description = value
								}
							}
						case "Required":
							if ident, ok := kv.Value.(*ast.Ident); ok {
								param.Required = ident.Name == "true"
							}
						case "In":
							// Parse parameter location (path, query, header)
							if selectorExpr, ok := kv.Value.(*ast.SelectorExpr); ok {
								switch selectorExpr.Sel.Name {
								case "InPath":
									param.In = gopenapi.InPath
								case "InQuery":
									param.In = gopenapi.InQuery
								case "InHeader":
									param.In = gopenapi.InHeader
								}
							}
						case "Schema":
							// Parse schema with type resolution
							if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
								schema, err := parseSchemaFromASTWithTypes(compLit, pkg)
								if err != nil {
									return params, fmt.Errorf("failed to parse schema: %w", err)
								}
								param.Schema = schema
							}
						}
					}
				}
			}
			params = append(params, param)
		}
	}

	return params, nil
}

// parseSchemaFromASTWithTypes parses gopenapi.Schema from AST with type resolution
func parseSchemaFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Schema, error) {
	schema := gopenapi.Schema{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Type" {
				// Parse type with resolution
				if selectorExpr, ok := kv.Value.(*ast.SelectorExpr); ok {
					switch selectorExpr.Sel.Name {
					case "String":
						schema.Type = gopenapi.String
					case "Integer":
						schema.Type = gopenapi.Integer
					case "Number":
						schema.Type = gopenapi.Number
					case "Boolean":
						schema.Type = gopenapi.Boolean
					case "Array":
						schema.Type = gopenapi.Array
					}
				} else if callExpr, ok := kv.Value.(*ast.CallExpr); ok {
					// Handle gopenapi.Object[Type]() calls with type resolution
					if indexExpr, ok := callExpr.Fun.(*ast.IndexExpr); ok {
						if selectorExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok && selectorExpr.Sel.Name == "Object" {
							// This is a generic call like gopenapi.Object[SomeType]()
							// Handle both local types (ident) and imported types (selector)
							var resolvedType reflect.Type

							if ident, ok := indexExpr.Index.(*ast.Ident); ok {
								// Local type like Object[User]()
								resolvedType = lookupTypeInPackage(ident, pkg)
							} else if selector, ok := indexExpr.Index.(*ast.SelectorExpr); ok {
								// Imported type like Object[gopenapi.Schema]()
								resolvedType = lookupImportedType(selector, pkg)
							}

							if resolvedType != nil {
								schema.Type = resolvedType
							} else {
								fmt.Fprintf(os.Stderr, "Warning: Could not resolve type for Object[%s](), falling back to interface{}\n", getTypeNameFromExpr(indexExpr.Index))
								schema.Type = gopenapi.Object[interface{}]()
							}
						}
					}
				}
			}
		}
	}

	return schema, nil
}

// resolveTypeFromAST resolves a type from AST using package type information
func resolveTypeFromAST(expr ast.Expr, pkg *packages.Package) reflect.Type {
	if pkg.TypesInfo == nil {
		return nil
	}

	if typeInfo := pkg.TypesInfo.TypeOf(expr); typeInfo != nil {
		// Convert types.Type to reflect.Type
		// This is a simplified conversion - in practice you might need more sophisticated handling
		switch typeInfo.String() {
		case "string":
			return reflect.TypeOf("")
		case "int":
			return reflect.TypeOf(0)
		case "float64":
			return reflect.TypeOf(0.0)
		case "bool":
			return reflect.TypeOf(false)
		default:
			// For complex types, try to get the underlying type
			// This is where the real type resolution happens
			if typeInfo.Underlying() != nil {
				return getReflectTypeFromGoType(typeInfo)
			}
		}
	}

	return nil
}

// getReflectTypeFromGoType converts a go/types.Type to reflect.Type
func getReflectTypeFromGoType(t interface{}) reflect.Type {
	// This is a simplified implementation
	// In practice, you'd need to handle all the different types.Type variants
	// For now, return interface{} as a fallback
	return reflect.TypeOf((*interface{})(nil)).Elem()
}

// lookupTypeInPackage looks up a type by AST identifier and returns a reflect.Type
func lookupTypeInPackage(ident *ast.Ident, pkg *packages.Package) reflect.Type {
	if pkg.TypesInfo == nil || pkg.TypesInfo.Uses == nil {
		return nil
	}

	// Use TypesInfo.Uses to get the object for this identifier
	obj := pkg.TypesInfo.Uses[ident]
	if obj == nil {
		return nil
	}

	// Get the underlying type
	if typeObj, ok := obj.(*types.TypeName); ok {
		return createReflectTypeFromGoTypes(typeObj.Type(), pkg)
	}

	return nil
}

// lookupImportedType looks up an imported type by AST selector expression and returns a reflect.Type
func lookupImportedType(selector *ast.SelectorExpr, pkg *packages.Package) reflect.Type {
	if pkg.TypesInfo == nil || pkg.TypesInfo.Uses == nil {
		return nil
	}

	// Use TypesInfo.Uses to get the object for this selector
	obj := pkg.TypesInfo.Uses[selector.Sel]
	if obj == nil {
		return nil
	}

	// Get the underlying type
	if typeObj, ok := obj.(*types.TypeName); ok {
		return createReflectTypeFromGoTypes(typeObj.Type(), pkg)
	}

	return nil
}

// getTypeNameFromExpr extracts a readable type name from an AST expression for warnings
func getTypeNameFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if pkg, ok := e.X.(*ast.Ident); ok {
			return pkg.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	default:
		return "unknown"
	}
}

// createReflectTypeFromGoTypes creates a reflect.Type from go/types.Type
func createReflectTypeFromGoTypes(t types.Type, pkg *packages.Package) reflect.Type {
	switch typ := t.(type) {
	case *types.Named:
		// For named types, check the underlying type
		underlying := typ.Underlying()
		switch underlyingType := underlying.(type) {
		case *types.Struct:
			// Complex struct type - create a struct type
			return createStructType(underlyingType, pkg)
		case *types.Basic:
			// Named type with primitive underlying type (like type ID string)
			// Return the underlying primitive type
			return getReflectTypeFromGoTypesType(underlyingType)
		default:
			// For other underlying types (slices, arrays, etc.), use the underlying type
			return getReflectTypeFromGoTypesType(underlying)
		}
	case *types.Struct:
		return createStructType(typ, pkg)
	default:
		return getReflectTypeFromGoTypesType(t)
	}
}

// createStructType creates a reflect.Type for a struct from go/types.Struct
func createStructType(structType *types.Struct, pkg *packages.Package) reflect.Type {
	numFields := structType.NumFields()
	fields := make([]reflect.StructField, numFields)

	for i := 0; i < numFields; i++ {
		field := structType.Field(i)
		tag := ""
		if structType.Tag(i) != "" {
			tag = structType.Tag(i)
		}

		fieldType := getReflectTypeFromGoTypesType(field.Type())

		fields[i] = reflect.StructField{
			Name: field.Name(),
			Type: fieldType,
			Tag:  reflect.StructTag(tag),
		}
	}

	return reflect.StructOf(fields)
}

// getReflectTypeFromGoTypesType converts basic go/types.Type to reflect.Type
func getReflectTypeFromGoTypesType(t types.Type) reflect.Type {
	switch typ := t.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.String:
			return reflect.TypeOf("")
		case types.Int:
			return reflect.TypeOf(0)
		case types.Int8:
			return reflect.TypeOf(int8(0))
		case types.Int16:
			return reflect.TypeOf(int16(0))
		case types.Int32:
			return reflect.TypeOf(int32(0))
		case types.Int64:
			return reflect.TypeOf(int64(0))
		case types.Uint:
			return reflect.TypeOf(uint(0))
		case types.Uint8:
			return reflect.TypeOf(uint8(0))
		case types.Uint16:
			return reflect.TypeOf(uint16(0))
		case types.Uint32:
			return reflect.TypeOf(uint32(0))
		case types.Uint64:
			return reflect.TypeOf(uint64(0))
		case types.Float32:
			return reflect.TypeOf(float32(0))
		case types.Float64:
			return reflect.TypeOf(float64(0))
		case types.Bool:
			return reflect.TypeOf(false)
		default:
			return reflect.TypeOf((*interface{})(nil)).Elem()
		}
	case *types.Slice:
		elemType := getReflectTypeFromGoTypesType(typ.Elem())
		return reflect.SliceOf(elemType)
	case *types.Pointer:
		elemType := getReflectTypeFromGoTypesType(typ.Elem())
		return reflect.PtrTo(elemType)
	default:
		return reflect.TypeOf((*interface{})(nil)).Elem()
	}
}

// parseResponsesFromASTWithTypes parses gopenapi.Responses from AST with type resolution
func parseResponsesFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Responses, error) {
	responses := make(gopenapi.Responses)

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			// Get status code
			var statusCode int
			if basicLit, ok := kv.Key.(*ast.BasicLit); ok && basicLit.Kind == token.INT {
				if code, err := strconv.Atoi(basicLit.Value); err == nil {
					statusCode = code
				}
			}

			// Parse response
			if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
				response, err := parseResponseFromASTWithTypes(compLit, pkg)
				if err != nil {
					return responses, fmt.Errorf("failed to parse response for status %d: %w", statusCode, err)
				}
				responses[statusCode] = response
			}
		}
	}

	return responses, nil
}

// parseResponseFromASTWithTypes parses response struct from AST with type resolution
func parseResponseFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (struct {
	Description string           `json:"description,omitempty"`
	Content     gopenapi.Content `json:"content,omitempty"`
}, error) {
	response := struct {
		Description string           `json:"description,omitempty"`
		Content     gopenapi.Content `json:"content,omitempty"`
	}{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "Description":
					if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
						response.Description = strings.Trim(basicLit.Value, `"`)
					}
				case "Content":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						content, err := parseContentFromASTWithTypes(compLit, pkg)
						if err != nil {
							return response, fmt.Errorf("failed to parse content: %w", err)
						}
						response.Content = content
					}
				}
			}
		}
	}

	return response, nil
}

// parseContentFromASTWithTypes parses gopenapi.Content from AST with type resolution
func parseContentFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.Content, error) {
	content := make(gopenapi.Content)

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			// Get media type
			var mediaType gopenapi.MediaType
			if selectorExpr, ok := kv.Key.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "ApplicationJSON" {
					mediaType = gopenapi.ApplicationJSON
				}
			}

			// Parse media type object
			if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
				mediaTypeObj := struct {
					Schema gopenapi.Schema `json:"schema,omitempty"`
				}{}
				for _, mediaElt := range compLit.Elts {
					if kv, ok := mediaElt.(*ast.KeyValueExpr); ok {
						if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Schema" {
							if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
								schema, err := parseSchemaFromASTWithTypes(compLit, pkg)
								if err != nil {
									return content, fmt.Errorf("failed to parse schema: %w", err)
								}
								mediaTypeObj.Schema = schema
							}
						}
					}
				}
				content[mediaType] = mediaTypeObj
			}
		}
	}

	return content, nil
}

// parseRequestBodyFromASTWithTypes parses gopenapi.RequestBody from AST with type resolution
func parseRequestBodyFromASTWithTypes(lit *ast.CompositeLit, pkg *packages.Package) (gopenapi.RequestBody, error) {
	requestBody := gopenapi.RequestBody{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "Required":
					if ident, ok := kv.Value.(*ast.Ident); ok {
						requestBody.Required = ident.Name == "true"
					}
				case "Content":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						content, err := parseContentFromASTWithTypes(compLit, pkg)
						if err != nil {
							return requestBody, fmt.Errorf("failed to parse content: %w", err)
						}
						requestBody.Content = content
					}
				}
			}
		}
	}

	return requestBody, nil
}

// parseSpecViaAST parses the Go file using AST and extracts the gopenapi.Spec
func parseSpecViaAST(filename, varName string) (gopenapi.Spec, error) {
	// Parse the Go file
	fset := token.NewFileSet()
	src, err := os.ReadFile(filename)
	if err != nil {
		return gopenapi.Spec{}, fmt.Errorf("failed to read file: %w", err)
	}

	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return gopenapi.Spec{}, fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Find the variable declaration and extract its value
	var specLiteral *ast.CompositeLit
	found := false

	ast.Inspect(node, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range valueSpec.Names {
						if name.Name == varName {
							if i < len(valueSpec.Values) {
								if compLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
									specLiteral = compLit
									found = true
									return false
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	if !found {
		return gopenapi.Spec{}, fmt.Errorf("variable %s not found in file %s", varName, filename)
	}

	if specLiteral == nil {
		return gopenapi.Spec{}, fmt.Errorf("variable %s is not a composite literal", varName)
	}

	// Parse the composite literal into a gopenapi.Spec
	spec, err := parseSpecFromAST(specLiteral, fset, src)
	if err != nil {
		return gopenapi.Spec{}, fmt.Errorf("failed to parse spec from AST: %w", err)
	}

	return spec, nil
}

// parseSpecFromAST converts an AST composite literal to a gopenapi.Spec
func parseSpecFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Spec, error) {
	spec := gopenapi.Spec{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "OpenAPI":
					if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
						spec.OpenAPI = strings.Trim(basicLit.Value, `"`)
					}
				case "Info":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						info, err := parseInfoFromAST(compLit, fset, src)
						if err != nil {
							return spec, fmt.Errorf("failed to parse Info: %w", err)
						}
						spec.Info = info
					}
				case "Servers":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						servers, err := parseServersFromAST(compLit, fset, src)
						if err != nil {
							return spec, fmt.Errorf("failed to parse Servers: %w", err)
						}
						spec.Servers = servers
					}
				case "Paths":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						paths, err := parsePathsFromAST(compLit, fset, src)
						if err != nil {
							return spec, fmt.Errorf("failed to parse Paths: %w", err)
						}
						spec.Paths = paths
					}
				}
			}
		}
	}

	return spec, nil
}

// parseInfoFromAST parses gopenapi.Info from AST
func parseInfoFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Info, error) {
	info := gopenapi.Info{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
					value := strings.Trim(basicLit.Value, `"`)
					switch ident.Name {
					case "Title":
						info.Title = value
					case "Description":
						info.Description = value
					case "Version":
						info.Version = value
					}
				}
			}
		}
	}

	return info, nil
}

// parseServersFromAST parses gopenapi.Servers from AST
func parseServersFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Servers, error) {
	var servers gopenapi.Servers

	for _, elt := range lit.Elts {
		if compLit, ok := elt.(*ast.CompositeLit); ok {
			server := struct {
				URL         string `json:"url"`
				Description string `json:"description"`
			}{}
			for _, serverElt := range compLit.Elts {
				if kv, ok := serverElt.(*ast.KeyValueExpr); ok {
					if ident, ok := kv.Key.(*ast.Ident); ok {
						if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
							value := strings.Trim(basicLit.Value, `"`)
							switch ident.Name {
							case "URL":
								server.URL = value
							case "Description":
								server.Description = value
							}
						}
					}
				}
			}
			servers = append(servers, server)
		}
	}

	return servers, nil
}

// parsePathsFromAST parses gopenapi.Paths from AST
func parsePathsFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Paths, error) {
	paths := make(gopenapi.Paths)

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			// Get the path string
			var pathStr string
			if basicLit, ok := kv.Key.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
				pathStr = strings.Trim(basicLit.Value, `"`)
			}

			// Parse the path item
			if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
				pathItem, err := parsePathItemFromAST(compLit, fset, src)
				if err != nil {
					return paths, fmt.Errorf("failed to parse path item for %s: %w", pathStr, err)
				}
				paths[pathStr] = pathItem
			}
		}
	}

	return paths, nil
}

// parsePathItemFromAST parses gopenapi.Path from AST
func parsePathItemFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Path, error) {
	pathItem := gopenapi.Path{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				if unaryExpr, ok := kv.Value.(*ast.UnaryExpr); ok && unaryExpr.Op == token.AND {
					if compLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
						operation, err := parseOperationFromAST(compLit, fset, src)
						if err != nil {
							return pathItem, fmt.Errorf("failed to parse operation %s: %w", ident.Name, err)
						}

						switch strings.ToUpper(ident.Name) {
						case "GET":
							pathItem.Get = &operation
						case "POST":
							pathItem.Post = &operation
						case "PUT":
							pathItem.Put = &operation
						case "DELETE":
							pathItem.Delete = &operation
						case "PATCH":
							pathItem.Patch = &operation
						case "HEAD":
							pathItem.Head = &operation
						case "OPTIONS":
							pathItem.Options = &operation
						}
					}
				}
			}
		}
	}

	return pathItem, nil
}

// parseOperationFromAST parses gopenapi.Operation from AST
func parseOperationFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Operation, error) {
	operation := gopenapi.Operation{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "OperationId", "Summary", "Description":
					if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
						value := strings.Trim(basicLit.Value, `"`)
						switch ident.Name {
						case "OperationId":
							operation.OperationId = value
						case "Summary":
							operation.Summary = value
						case "Description":
							operation.Description = value
						}
					}
				case "Parameters":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						params, err := parseParametersFromAST(compLit, fset, src)
						if err != nil {
							return operation, fmt.Errorf("failed to parse parameters: %w", err)
						}
						operation.Parameters = params
					}
				case "Responses":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						responses, err := parseResponsesFromAST(compLit, fset, src)
						if err != nil {
							return operation, fmt.Errorf("failed to parse responses: %w", err)
						}
						operation.Responses = responses
					}
				case "RequestBody":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						requestBody, err := parseRequestBodyFromAST(compLit, fset, src)
						if err != nil {
							return operation, fmt.Errorf("failed to parse request body: %w", err)
						}
						operation.RequestBody = requestBody
					}
				case "Handler":
					// Skip handler parsing for now as it's complex and not needed for client generation
					operation.Handler = nil
				}
			}
		}
	}

	return operation, nil
}

// parseParametersFromAST parses gopenapi.Parameters from AST
func parseParametersFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Parameters, error) {
	var params gopenapi.Parameters

	for _, elt := range lit.Elts {
		if compLit, ok := elt.(*ast.CompositeLit); ok {
			param := gopenapi.Parameter{}
			for _, paramElt := range compLit.Elts {
				if kv, ok := paramElt.(*ast.KeyValueExpr); ok {
					if ident, ok := kv.Key.(*ast.Ident); ok {
						switch ident.Name {
						case "Name", "Description":
							if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
								value := strings.Trim(basicLit.Value, `"`)
								switch ident.Name {
								case "Name":
									param.Name = value
								case "Description":
									param.Description = value
								}
							}
						case "Required":
							if ident, ok := kv.Value.(*ast.Ident); ok {
								param.Required = ident.Name == "true"
							}
						case "In":
							// Parse parameter location (path, query, header)
							if selectorExpr, ok := kv.Value.(*ast.SelectorExpr); ok {
								switch selectorExpr.Sel.Name {
								case "InPath":
									param.In = gopenapi.InPath
								case "InQuery":
									param.In = gopenapi.InQuery
								case "InHeader":
									param.In = gopenapi.InHeader
								}
							}
						case "Schema":
							// Parse schema - simplified for basic types
							if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
								schema, err := parseSchemaFromAST(compLit, fset, src)
								if err != nil {
									return params, fmt.Errorf("failed to parse schema: %w", err)
								}
								param.Schema = schema
							}
						}
					}
				}
			}
			params = append(params, param)
		}
	}

	return params, nil
}

// parseSchemaFromAST parses gopenapi.Schema from AST (simplified)
func parseSchemaFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Schema, error) {
	schema := gopenapi.Schema{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Type" {
				// Parse type - this is simplified and handles basic types
				if selectorExpr, ok := kv.Value.(*ast.SelectorExpr); ok {
					switch selectorExpr.Sel.Name {
					case "String":
						schema.Type = gopenapi.String
					case "Integer":
						schema.Type = gopenapi.Integer
					case "Number":
						schema.Type = gopenapi.Number
					case "Boolean":
						schema.Type = gopenapi.Boolean
					case "Array":
						schema.Type = gopenapi.Array
					}
				} else if callExpr, ok := kv.Value.(*ast.CallExpr); ok {
					// Handle gopenapi.Object[Type]() calls
					if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "Object" {
							// For Object types, we'll set a placeholder type
							// The actual type information would need more complex parsing
							schema.Type = gopenapi.Object[interface{}]()
						}
					}
				}
			}
		}
	}

	return schema, nil
}

// parseResponsesFromAST parses gopenapi.Responses from AST
func parseResponsesFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Responses, error) {
	responses := make(gopenapi.Responses)

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			// Get status code
			var statusCode int
			if basicLit, ok := kv.Key.(*ast.BasicLit); ok && basicLit.Kind == token.INT {
				if code, err := strconv.Atoi(basicLit.Value); err == nil {
					statusCode = code
				}
			}

			// Parse response
			if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
				response, err := parseResponseFromAST(compLit, fset, src)
				if err != nil {
					return responses, fmt.Errorf("failed to parse response for status %d: %w", statusCode, err)
				}
				responses[statusCode] = response
			}
		}
	}

	return responses, nil
}

// parseResponseFromAST parses response struct from AST
func parseResponseFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (struct {
	Description string           `json:"description,omitempty"`
	Content     gopenapi.Content `json:"content,omitempty"`
}, error) {
	response := struct {
		Description string           `json:"description,omitempty"`
		Content     gopenapi.Content `json:"content,omitempty"`
	}{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "Description":
					if basicLit, ok := kv.Value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
						response.Description = strings.Trim(basicLit.Value, `"`)
					}
				case "Content":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						content, err := parseContentFromAST(compLit, fset, src)
						if err != nil {
							return response, fmt.Errorf("failed to parse content: %w", err)
						}
						response.Content = content
					}
				}
			}
		}
	}

	return response, nil
}

// parseContentFromAST parses gopenapi.Content from AST
func parseContentFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.Content, error) {
	content := make(gopenapi.Content)

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			// Get media type
			var mediaType gopenapi.MediaType
			if selectorExpr, ok := kv.Key.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "ApplicationJSON" {
					mediaType = gopenapi.ApplicationJSON
				}
			}

			// Parse media type object
			if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
				mediaTypeObj := struct {
					Schema gopenapi.Schema `json:"schema,omitempty"`
				}{}
				for _, mediaElt := range compLit.Elts {
					if kv, ok := mediaElt.(*ast.KeyValueExpr); ok {
						if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Schema" {
							if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
								schema, err := parseSchemaFromAST(compLit, fset, src)
								if err != nil {
									return content, fmt.Errorf("failed to parse schema: %w", err)
								}
								mediaTypeObj.Schema = schema
							}
						}
					}
				}
				content[mediaType] = mediaTypeObj
			}
		}
	}

	return content, nil
}

// parseRequestBodyFromAST parses gopenapi.RequestBody from AST
func parseRequestBodyFromAST(lit *ast.CompositeLit, fset *token.FileSet, src []byte) (gopenapi.RequestBody, error) {
	requestBody := gopenapi.RequestBody{}

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				switch ident.Name {
				case "Required":
					if ident, ok := kv.Value.(*ast.Ident); ok {
						requestBody.Required = ident.Name == "true"
					}
				case "Content":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						content, err := parseContentFromAST(compLit, fset, src)
						if err != nil {
							return requestBody, fmt.Errorf("failed to parse content: %w", err)
						}
						requestBody.Content = content
					}
				}
			}
		}
	}

	return requestBody, nil
}

// generateClientForLanguage generates a client for the specified language
func generateClientForLanguage(spec *gopenapi.Spec, language, outputDir, packageName string) error {
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

// GenerateClient generates a client from a gopenapi.Spec
func GenerateClient(spec *gopenapi.Spec, outputFile, packageName, templateFile, language string) error {
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
	templateData := generateTemplateData(spec, packageName, language)

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

	// Execute template
	if err := tmpl.Execute(outFile, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// getTemplateFuncs returns template functions for the specified language
func getTemplateFuncs(language string) template.FuncMap {
	funcs := template.FuncMap{
		"title": strings.Title,
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

func generateTemplateData(spec *gopenapi.Spec, packageName string, language string) *TemplateData {
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
				StructName:  toStructName(operation.OperationId),
				MethodName:  toMethodName(operation.OperationId),
			}

			// Process parameters
			grouped := operation.Parameters.Group()

			// Path parameters
			if len(grouped.Path) > 0 {
				opData.HasPathParams = true
				for name, schema := range grouped.Path {
					param := ParamData{
						Name:        name,
						GoName:      toGoName(name),
						GoType:      schemaToGoType(schema),
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
						GoName: toGoName(name),
						GoType: schemaToGoType(schema),
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
						GoName: toGoName(name),
						GoType: schemaToGoType(schema),
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
									opData.ResponseType = schemaToGoType(content.Schema)

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

func toStructName(operationId string) string {
	// Convert operationId to PascalCase struct name
	// For camelCase inputs, just capitalize the first letter
	// For other formats, convert to proper PascalCase
	if operationId == "" {
		return ""
	}

	// Check if it's already in camelCase (starts with lowercase, contains uppercase)
	if unicode.IsLower(rune(operationId[0])) {
		// Just capitalize the first letter for camelCase
		return strings.ToUpper(operationId[:1]) + operationId[1:]
	}

	// If it's not camelCase, convert it to PascalCase
	parts := strings.FieldsFunc(operationId, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	if len(parts) == 0 {
		return strings.ToUpper(operationId[:1]) + strings.ToLower(operationId[1:])
	}

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

func toMethodName(operationId string) string {
	// For camelCase operationIds, just capitalize the first letter
	// For other formats, convert to proper camelCase first
	if operationId == "" {
		return ""
	}

	// Check if it's already in camelCase (starts with lowercase, contains uppercase)
	if unicode.IsLower(rune(operationId[0])) {
		// Just capitalize the first letter
		return strings.ToUpper(operationId[:1]) + operationId[1:]
	}

	// If it's not camelCase, convert it first
	parts := strings.FieldsFunc(operationId, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	if len(parts) == 0 {
		return strings.ToUpper(operationId[:1]) + strings.ToLower(operationId[1:])
	}

	var result strings.Builder
	for i, part := range parts {
		if len(part) > 0 {
			if i == 0 {
				// First part: capitalize first letter, lowercase the rest
				result.WriteString(strings.ToUpper(part[:1]))
				if len(part) > 1 {
					result.WriteString(strings.ToLower(part[1:]))
				}
			} else {
				// Subsequent parts: capitalize first letter, lowercase the rest
				result.WriteString(strings.ToUpper(part[:1]))
				if len(part) > 1 {
					result.WriteString(strings.ToLower(part[1:]))
				}
			}
		}
	}
	return result.String()
}

func toGoName(name string) string {
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

func schemaToGoType(schema gopenapi.Schema) string {
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

func schemaToFields(schema gopenapi.Schema) []FieldData {
	return schemaToFieldsWithName(schema, "")
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
	default:
		return "interface{}"
	}
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
