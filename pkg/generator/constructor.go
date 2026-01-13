package generator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sanity-io/litter"
	"github.com/sosodev/duration"

	"github.com/atombender/go-jsonschema/pkg/codegen"
)

// constructorGenerator generates New* constructor functions for struct types
// that have fields with default values from the JSON schema.
type constructorGenerator struct {
	decl   *codegen.TypeDecl
	output *output
}

// hasDefaults returns true if the struct type has any fields with default values.
func (g *constructorGenerator) hasDefaults() bool {
	st, ok := g.decl.Type.(*codegen.StructType)
	if !ok {
		return false
	}

	for _, f := range st.Fields {
		if f.DefaultValue != nil {
			return true
		}
	}

	return false
}

// generate creates the New* constructor function for the struct type.
func (g *constructorGenerator) generate() func(*codegen.Emitter) error {
	return func(out *codegen.Emitter) error {
		st, ok := g.decl.Type.(*codegen.StructType)
		if !ok {
			return nil
		}

		typeName := g.decl.Name
		out.Commentf("New%s creates a new %s with default values.", typeName, typeName)
		out.Printlnf("func New%s() %s {", typeName, typeName)
		out.Indent(1)
		out.Printlnf("return %s{", typeName)
		out.Indent(1)

		for _, f := range st.Fields {
			if f.DefaultValue != nil {
				// Skip the AdditionalProperties field as it has special handling
				if f.Name == additionalProperties {
					continue
				}

				defaultStr, err := formatDefaultValue(f.Type, f.DefaultValue, out.MaxLineLength())
				if err != nil {
					return fmt.Errorf("cannot format default value for field %s: %w", f.Name, err)
				}

				out.Printlnf("%s: %s,", f.Name, defaultStr)
			}
		}

		out.Indent(-1)
		out.Printlnf("}")
		out.Indent(-1)
		out.Printlnf("}")

		return nil
	}
}

// formatDefaultValue formats a default value for use in generated Go code.
func formatDefaultValue(fieldType codegen.Type, defaultValue interface{}, maxLineLen int32) (string, error) {
	// Handle named types (nested structs with their own defaults)
	if nt, ok := fieldType.(*codegen.NamedType); ok {
		dvm, ok := defaultValue.(map[string]any)
		if ok {
			namedFields := ""

			for _, k := range sortedKeys(dvm) {
				namedFields += fmt.Sprintf("\n%s: %s,", upperFirst(k), litter.Sdump(dvm[k]))
			}

			if namedFields != "" {
				namedFields += "\n"
			}

			return fmt.Sprintf("%s{%s}", nt.Decl.GetName(), namedFields), nil
		}
	}

	// Handle duration type
	if _, ok := fieldType.(codegen.DurationType); ok {
		defaultDurationISO8601, ok := defaultValue.(string)
		if !ok {
			return "", fmt.Errorf("%w: %T given", ErrDefaultDurationIsNotAString, defaultValue)
		}

		if defaultDurationISO8601 == "" {
			return "", ErrDurationIsEmpty
		}

		d, err := duration.Parse(defaultDurationISO8601)
		if err != nil {
			return "", ErrCannotConvertISO8601ToGoFormat
		}

		goDurationStr := d.ToTimeDuration().String()
		// For constructors, we use a constant duration value parsed at init time
		// This is simpler than the validator approach since we're just initializing
		return fmt.Sprintf("func() time.Duration { d, _ := time.ParseDuration(%q); return d }()", goDurationStr), nil
	}

	// Handle slice types
	if err := tryFormatSlice(defaultValue); err == nil {
		return formatSliceValue(fieldType, defaultValue, maxLineLen)
	}

	// Fallback to litter.Sdump
	return strings.TrimSpace(litter.Sdump(defaultValue)), nil
}

// tryFormatSlice checks if the value can be formatted as a slice.
func tryFormatSlice(defaultValue interface{}) error {
	kind := reflect.ValueOf(defaultValue).Kind()
	if kind != reflect.Slice {
		return ErrCannotFindSlideToDump
	}

	_, ok := defaultValue.([]interface{})
	if !ok {
		return ErrInvalidDefaultValue
	}

	return nil
}

// formatSliceValue formats a slice default value.
func formatSliceValue(fieldType codegen.Type, defaultValue interface{}, maxLineLen int32) (string, error) {
	tmpEmitter := codegen.NewEmitter(maxLineLen)

	if err := fieldType.Generate(tmpEmitter); err != nil {
		return "", fmt.Errorf("%w: %w", ErrCannotDumpDefaultSlice, err)
	}

	df, ok := defaultValue.([]interface{})
	if !ok {
		return "", ErrInvalidDefaultValue
	}

	if len(df) == 0 {
		return tmpEmitter.String() + "{}", nil
	}

	tmpEmitter.Printlnf("{")

	for _, value := range df {
		tmpEmitter.Printlnf("%s,", litter.Sdump(value))
	}

	tmpEmitter.Printf("}")

	return tmpEmitter.String(), nil
}
