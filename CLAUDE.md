# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-jsonschema is a CLI tool that generates Go data types and structs from JSON Schema definitions, including unmarshalling code with validation.

## Build Commands

```bash
make build          # Build using goreleaser (snapshot)
make test           # Run all tests with coverage
make lint-go        # Run golangci-lint
make format-go      # Format Go code
```

Run a single test:
```bash
go test -v -run TestCore ./tests
```

## Regenerating Test Golden Files

When changes affect generated code output:
```bash
export OVERWRITE_EXPECTED_GO_FILE="true"
make test
```

## Architecture

### Code Generation Pipeline

1. **Schema Loading** (`pkg/schemas/`): Parses JSON/YAML schema files into `Schema` and `Type` structs. The `Loader` interface handles file resolution and caching.

2. **Generator** (`pkg/generator/generate.go`): Main orchestrator. Creates `schemaGenerator` instances per schema file. Maps schema IDs to output files/packages.

3. **Schema Generator** (`pkg/generator/schema_generator.go`): Converts schema types to Go types. Handles `$ref` resolution, `allOf`/`anyOf` merging, enum generation, and cycle detection.

4. **Codegen** (`pkg/codegen/`): Go AST-like model (`TypeDecl`, `StructType`, `NamedType`, etc.) and `Emitter` for rendering Go source.

5. **Formatters** (`pkg/generator/formatter.go`, `json_formatter.go`, `yaml_formatter.go`): Generate `UnmarshalJSON`/`UnmarshalYAML` methods with validation.

6. **Validators** (`pkg/generator/validator.go`): Generate validation code for required fields, string patterns, numeric bounds, etc.

### Go Workspaces

The project uses Go workspaces (`go.work`) to test generated code. The `tests/` module imports generated code from `tests/data/` subdirectories, allowing both generation logic and generated code to be tested together.

### Key CLI Flags

- `--schema-package=URI=PACKAGE`: Map schema $id to Go package
- `--schema-output=URI=FILENAME`: Map schema $id to output file
- `--only-models`: Skip generating unmarshal/validation methods
- `--min-sized-ints`: Use sized int types based on min/max values
- `--default-constructors`: Generate `New*` constructor functions

## Conventions

- Uses Conventional Commits (feat:, fix:, chore:, etc.)
- Linting config: `.rules/.golangci.yml`
