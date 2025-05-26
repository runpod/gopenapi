package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/runpod/gopenapi/cmd/gopenapi/generator"
	"github.com/runpod/gopenapi/cmd/gopenapi/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "generate":
		if len(os.Args) < 3 {
			printGenerateUsage()
			os.Exit(1)
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "spec":
			generateSpecCommand()
		case "client":
			generateClientCommand()
		default:
			fmt.Fprintf(os.Stderr, "Unknown generate subcommand: %s\n\n", subcommand)
			printGenerateUsage()
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `gopenapi - OpenAPI code generation tool

Usage:
  gopenapi generate spec [flags]    Generate OpenAPI JSON specification
  gopenapi generate client [flags]  Generate API clients
  gopenapi help                     Show this help message

Use "gopenapi generate <subcommand> -help" for more information about a subcommand.
`)
}

func printGenerateUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  gopenapi generate spec [flags]    Generate OpenAPI JSON specification
  gopenapi generate client [flags]  Generate API clients

Use "gopenapi generate <subcommand> -help" for more information about a subcommand.
`)
}

func generateSpecCommand() {
	fs := flag.NewFlagSet("generate spec", flag.ExitOnError)
	specFile := fs.String("spec", "", "Go file containing the OpenAPI spec (required)")
	specVar := fs.String("var", "", "Variable name containing the spec (required, e.g., 'ExampleSpec')")
	output := fs.String("output", "", "Output file for OpenAPI JSON (if empty, outputs to stdout)")
	help := fs.Bool("help", false, "Show help information")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Generate OpenAPI JSON specification from Go code

Usage:
  gopenapi generate spec [flags]

Flags:
  -spec string
        Go file containing the OpenAPI spec (required)
  -var string
        Variable name containing the spec (required, e.g., 'ExampleSpec')
  -output string
        Output file for OpenAPI JSON (if empty, outputs to stdout)
  -help
        Show this help message

Examples:
  gopenapi generate spec -spec examples/spec/spec.go -var ExampleSpec -output openapi.json
  gopenapi generate spec -spec examples/spec/spec.go -var ExampleSpec
`)
	}

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *help {
		fs.Usage()
		return
	}

	if *specFile == "" || *specVar == "" {
		fmt.Fprintf(os.Stderr, "Error: Both -spec and -var flags are required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	spec, err := parser.ParseSpecFromFile(*specFile, *specVar)
	if err != nil {
		log.Fatalf("Failed to parse spec from file: %v", err)
	}

	// Convert spec to OpenAPI JSON
	jsonData, err := parser.SpecToOpenAPIJSON(&spec)
	if err != nil {
		log.Fatalf("Failed to convert spec to OpenAPI JSON: %v", err)
	}

	// Output to file or stdout
	if *output == "" {
		fmt.Print(string(jsonData))
	} else {
		err := os.WriteFile(*output, jsonData, 0644)
		if err != nil {
			log.Fatalf("Failed to write OpenAPI JSON to file: %v", err)
		}
		fmt.Printf("Generated OpenAPI JSON specification: %s\n", *output)
	}
}

func generateClientCommand() {
	fs := flag.NewFlagSet("generate client", flag.ExitOnError)
	specFile := fs.String("spec", "", "Go file containing the OpenAPI spec (required)")
	specVar := fs.String("var", "", "Variable name containing the spec (required, e.g., 'ExampleSpec')")
	outputDir := fs.String("output", "", "Output directory for generated clients (if empty, outputs to stdout)")
	packageName := fs.String("package", "client", "Package name for generated code")
	languages := fs.String("languages", "go", "Comma-separated list of languages to generate (go,python,typescript)")
	help := fs.Bool("help", false, "Show help information")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Generate API clients from OpenAPI specification

Usage:
  gopenapi generate client [flags]

Flags:
  -spec string
        Go file containing the OpenAPI spec (required)
  -var string
        Variable name containing the spec (required, e.g., 'ExampleSpec')
  -output string
        Output directory for generated clients (if empty, outputs to stdout)
  -package string
        Package name for generated code (default "client")
  -languages string
        Comma-separated list of languages to generate (default "go")
        Supported languages: go, python, typescript
  -help
        Show this help message

Examples:
  gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -output ./clients
  gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -languages go,python
  gopenapi generate client -spec examples/spec/spec.go -var ExampleSpec -package myclient
`)
	}

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *help {
		fs.Usage()
		return
	}

	if *specFile == "" || *specVar == "" {
		fmt.Fprintf(os.Stderr, "Error: Both -spec and -var flags are required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	spec, err := parser.ParseSpecFromFile(*specFile, *specVar)
	if err != nil {
		log.Fatalf("Failed to parse spec from file: %v", err)
	}

	// Parse languages
	langs := strings.Split(*languages, ",")
	for i, lang := range langs {
		langs[i] = strings.TrimSpace(lang)
	}

	// Validate languages
	for _, lang := range langs {
		if lang != "go" && lang != "python" && lang != "typescript" {
			log.Fatalf("Unsupported language: %s. Supported languages: go, python, typescript", lang)
		}
	}

	// If output directory is not specified, output to stdout (only works for single language)
	if *outputDir == "" {
		if len(langs) > 1 {
			log.Fatal("Cannot output multiple languages to stdout. Please specify -output directory or use single language.")
		}
		err := generator.GenerateClientToStdout(&spec, langs[0], *packageName)
		if err != nil {
			log.Fatalf("Failed to generate %s client: %v", langs[0], err)
		}
		return
	}

	// Generate clients for each language to files
	for _, lang := range langs {
		err := generator.GenerateClientForLanguage(&spec, lang, *outputDir, *packageName)
		if err != nil {
			log.Fatalf("Failed to generate %s client: %v", lang, err)
		}
		fmt.Printf("Generated %s client in %s\n", lang, *outputDir)
	}
}
