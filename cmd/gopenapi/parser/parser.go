package parser

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/runpod/gopenapi"
	"golang.org/x/tools/go/packages"
)

// ParseSpecFromFile parses a Go file and extracts the specified gopenapi.Spec variable
func ParseSpecFromFile(filename, varName string) (gopenapi.Spec, error) {
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
						info := gopenapi.Info{}
						for _, infoElt := range compLit.Elts {
							if kv, ok := infoElt.(*ast.KeyValueExpr); ok {
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
						spec.Info = info
					}
				case "Servers":
					if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
						var servers gopenapi.Servers
						for _, serverElt := range compLit.Elts {
							if compLit, ok := serverElt.(*ast.CompositeLit); ok {
								server := struct {
									URL         string `json:"url"`
									Description string `json:"description"`
								}{}
								for _, serverFieldElt := range compLit.Elts {
									if kv, ok := serverFieldElt.(*ast.KeyValueExpr); ok {
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
					// Handle different types of call expressions
					if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "TypeOf" {
							// This is a reflect.TypeOf() call
							if len(callExpr.Args) > 0 {
								// Get the type from the argument to TypeOf
								resolvedType := resolveTypeFromAST(callExpr.Args[0], pkg)
								if resolvedType != nil {
									schema.Type = resolvedType
								} else {
									fmt.Fprintf(os.Stderr, "Warning: Could not resolve type for reflect.TypeOf(), falling back to interface{}\n")
									schema.Type = reflect.TypeOf((*interface{})(nil)).Elem()
								}
							}
						}
					} else if indexExpr, ok := callExpr.Fun.(*ast.IndexExpr); ok {
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
		// Use the improved type resolution function
		return createReflectTypeFromGoTypes(typeInfo)
	}

	return nil
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
		return createReflectTypeFromGoTypes(typeObj.Type())
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
		return createReflectTypeFromGoTypes(typeObj.Type())
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
func createReflectTypeFromGoTypes(t types.Type) reflect.Type {
	switch typ := t.(type) {
	case *types.Named:
		// For named types, we need to preserve the package path and name information
		// Try to get the actual reflect.Type for well-known types
		var pkgPath string
		var typeName string

		if typ.Obj() != nil {
			typeName = typ.Obj().Name()
			if typ.Obj().Pkg() != nil {
				pkgPath = typ.Obj().Pkg().Path()
			}
		}

		// Try to get the actual type by creating a zero value
		if pkgPath != "" && typeName != "" {
			if actualType := getActualReflectType(pkgPath, typeName); actualType != nil {
				return actualType
			}
		}

		// Fallback: check the underlying type
		underlying := typ.Underlying()
		switch underlyingType := underlying.(type) {
		case *types.Struct:
			// Complex struct type - create a struct type
			return createStructType(underlyingType)
		case *types.Basic:
			// Named type with primitive underlying type (like type ID string)
			// Return the underlying primitive type
			return getReflectTypeFromGoTypesType(underlyingType)
		default:
			// For other underlying types (slices, arrays, etc.), use the underlying type
			return getReflectTypeFromGoTypesType(underlying)
		}
	case *types.Struct:
		return createStructType(typ)
	default:
		return getReflectTypeFromGoTypesType(t)
	}
}

// getActualReflectType tries to get the actual reflect.Type for well-known types
func getActualReflectType(pkgPath, typeName string) reflect.Type {
	// For well-known standard library types, we can get the actual reflect.Type
	switch pkgPath + "." + typeName {
	case "time.Time":
		return reflect.TypeOf((*time.Time)(nil)).Elem()
	case "time.Duration":
		return reflect.TypeOf((*time.Duration)(nil)).Elem()
	}

	// For other types, we can't easily get the actual reflect.Type without importing
	// the package, so return nil to use the fallback
	return nil
}

// createStructType creates a reflect.Type for a struct from go/types.Struct
func createStructType(structType *types.Struct) reflect.Type {
	numFields := structType.NumFields()
	fields := make([]reflect.StructField, numFields)

	for i := range numFields {
		field := structType.Field(i)
		tag := ""
		if structType.Tag(i) != "" {
			tag = structType.Tag(i)
		}

		// Use the recursive type resolution to properly handle named types
		fieldType := createReflectTypeFromGoTypes(field.Type())

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
			return reflect.TypeOf((*any)(nil)).Elem()
		}
	case *types.Slice:
		// Use recursive resolution for slice elements
		elemType := createReflectTypeFromGoTypes(typ.Elem())
		return reflect.SliceOf(elemType)
	case *types.Pointer:
		// Use recursive resolution for pointer elements
		elemType := createReflectTypeFromGoTypes(typ.Elem())
		return reflect.PointerTo(elemType)
	case *types.Map:
		// Handle map types
		keyType := createReflectTypeFromGoTypes(typ.Key())
		valueType := createReflectTypeFromGoTypes(typ.Elem())
		return reflect.MapOf(keyType, valueType)
	case *types.Named:
		// This should be handled by createReflectTypeFromGoTypes, but add as fallback
		return createReflectTypeFromGoTypes(typ)
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

// SpecToOpenAPIJSON converts a gopenapi.Spec to OpenAPI JSON format
func SpecToOpenAPIJSON(spec *gopenapi.Spec) ([]byte, error) {
	// Create OpenAPI JSON structure
	openAPISpec := map[string]interface{}{
		"openapi": spec.OpenAPI,
		"info": map[string]interface{}{
			"title":       spec.Info.Title,
			"description": spec.Info.Description,
			"version":     spec.Info.Version,
		},
	}

	// Add servers if present
	if len(spec.Servers) > 0 {
		servers := make([]map[string]interface{}, len(spec.Servers))
		for i, server := range spec.Servers {
			servers[i] = map[string]interface{}{
				"url":         server.URL,
				"description": server.Description,
			}
		}
		openAPISpec["servers"] = servers
	}

	// Add paths
	if len(spec.Paths) > 0 {
		paths := make(map[string]interface{})
		for path, pathItem := range spec.Paths {
			pathObj := make(map[string]interface{})

			// Add operations for each HTTP method
			if pathItem.Get != nil {
				pathObj["get"] = operationToJSON(pathItem.Get)
			}
			if pathItem.Post != nil {
				pathObj["post"] = operationToJSON(pathItem.Post)
			}
			if pathItem.Put != nil {
				pathObj["put"] = operationToJSON(pathItem.Put)
			}
			if pathItem.Delete != nil {
				pathObj["delete"] = operationToJSON(pathItem.Delete)
			}
			if pathItem.Patch != nil {
				pathObj["patch"] = operationToJSON(pathItem.Patch)
			}
			if pathItem.Head != nil {
				pathObj["head"] = operationToJSON(pathItem.Head)
			}
			if pathItem.Options != nil {
				pathObj["options"] = operationToJSON(pathItem.Options)
			}

			paths[path] = pathObj
		}
		openAPISpec["paths"] = paths
	}

	// Marshal to JSON with proper indentation
	return json.MarshalIndent(openAPISpec, "", "  ")
}

// operationToJSON converts a gopenapi.Operation to JSON format
func operationToJSON(op *gopenapi.Operation) map[string]interface{} {
	operation := map[string]interface{}{}

	if op.OperationId != "" {
		operation["operationId"] = op.OperationId
	}
	if op.Summary != "" {
		operation["summary"] = op.Summary
	}
	if op.Description != "" {
		operation["description"] = op.Description
	}

	// Add parameters
	if len(op.Parameters) > 0 {
		params := make([]map[string]interface{}, len(op.Parameters))
		for i, param := range op.Parameters {
			paramObj := map[string]interface{}{
				"name":        param.Name,
				"in":          parameterLocationToString(param.In),
				"required":    param.Required,
				"description": param.Description,
				"schema":      schemaToJSON(param.Schema),
			}
			params[i] = paramObj
		}
		operation["parameters"] = params
	}

	// Add request body
	if op.RequestBody.Content != nil {
		requestBody := map[string]interface{}{
			"required": op.RequestBody.Required,
			"content":  contentToJSON(op.RequestBody.Content),
		}
		operation["requestBody"] = requestBody
	}

	// Add responses
	if len(op.Responses) > 0 {
		responses := make(map[string]interface{})
		for statusCode, response := range op.Responses {
			responseObj := map[string]interface{}{
				"description": response.Description,
			}
			if response.Content != nil {
				responseObj["content"] = contentToJSON(response.Content)
			}
			responses[fmt.Sprintf("%d", statusCode)] = responseObj
		}
		operation["responses"] = responses
	}

	return operation
}

// parameterLocationToString converts parameter location to string
func parameterLocationToString(location gopenapi.In) string {
	switch location {
	case gopenapi.InPath:
		return "path"
	case gopenapi.InQuery:
		return "query"
	case gopenapi.InHeader:
		return "header"
	case gopenapi.InCookie:
		return "cookie"
	default:
		return "query"
	}
}

// schemaToJSON converts a gopenapi.Schema to JSON format
func schemaToJSON(schema gopenapi.Schema) map[string]interface{} {
	schemaObj := map[string]interface{}{}

	if schema.Type != nil {
		switch schema.Type {
		case gopenapi.String:
			schemaObj["type"] = "string"
		case gopenapi.Integer:
			schemaObj["type"] = "integer"
		case gopenapi.Number:
			schemaObj["type"] = "number"
		case gopenapi.Boolean:
			schemaObj["type"] = "boolean"
		case gopenapi.Array:
			schemaObj["type"] = "array"
		default:
			// For complex types (structs), use object type
			if schema.Type.Kind() == reflect.Struct {
				schemaObj["type"] = "object"
				// Add properties based on struct fields
				properties := generateStructProperties(schema.Type)
				if len(properties) > 0 {
					schemaObj["properties"] = properties
				}
			} else {
				schemaObj["type"] = goTypeToOpenAPIType(schema.Type)
			}
		}
	}

	return schemaObj
}

// generateStructProperties recursively generates properties for struct types
func generateStructProperties(t reflect.Type) map[string]interface{} {
	properties := make(map[string]interface{})

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

		// Generate schema for this field
		fieldSchema := generateFieldSchema(field.Type)
		properties[fieldName] = fieldSchema
	}

	return properties
}

// generateFieldSchema generates the schema for a single field type
func generateFieldSchema(t reflect.Type) map[string]interface{} {
	schema := map[string]interface{}{}

	// Handle special types first
	if t.PkgPath() != "" && t.Name() != "" {
		typeFullName := t.PkgPath() + "." + t.Name()
		switch typeFullName {
		case "time.Time":
			schema["type"] = "string"
			return schema
		case "time.Duration":
			schema["type"] = "integer"
			return schema
		}
	}

	// Handle types by kind
	switch t.Kind() {
	case reflect.String:
		schema["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		// TODO: Add items schema for array elements
	case reflect.Struct:
		schema["type"] = "object"
		// Recursively generate properties for nested structs
		properties := generateStructProperties(t)
		if len(properties) > 0 {
			schema["properties"] = properties
		}
	case reflect.Ptr:
		// For pointers, use the element type
		return generateFieldSchema(t.Elem())
	case reflect.Map:
		schema["type"] = "object"
		// TODO: Add additionalProperties for map values
	default:
		schema["type"] = "object"
	}

	return schema
}

// contentToJSON converts gopenapi.Content to JSON format
func contentToJSON(content gopenapi.Content) map[string]interface{} {
	contentObj := make(map[string]interface{})

	for mediaType, mediaTypeObj := range content {
		contentObj[string(mediaType)] = map[string]interface{}{
			"schema": schemaToJSON(mediaTypeObj.Schema),
		}
	}

	return contentObj
}

// goTypeToOpenAPIType converts Go reflect.Type to OpenAPI type string
func goTypeToOpenAPIType(t reflect.Type) string {
	// Handle named types
	if t.PkgPath() != "" && t.Name() != "" {
		// Special handling for well-known types that should have specific OpenAPI mappings
		// regardless of their underlying Go structure
		typeFullName := t.PkgPath() + "." + t.Name()
		switch typeFullName {
		case "time.Time":
			return "string" // time.Time should be represented as string in OpenAPI (RFC3339)
		case "time.Duration":
			return "integer" // Duration as nanoseconds
			// Add more well-known types as needed, but keep it minimal
			// For example: net.IP, url.URL, etc.
		}

		// For other named types, check the underlying type
		// This handles type aliases like `type UserID string` from any package
		switch t.Kind() {
		case reflect.String:
			return "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "integer"
		case reflect.Float32, reflect.Float64:
			return "number"
		case reflect.Bool:
			return "boolean"
		case reflect.Slice, reflect.Array:
			return "array"
		case reflect.Struct:
			return "object"
		case reflect.Ptr:
			return goTypeToOpenAPIType(t.Elem())
		default:
			return "object"
		}
	}

	// Handle basic types
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Struct:
		return "object"
	case reflect.Ptr:
		return goTypeToOpenAPIType(t.Elem())
	default:
		return "object"
	}
}
