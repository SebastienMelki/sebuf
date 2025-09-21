package openapiv3

import (
	"fmt"
	"strconv"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	yaml "go.yaml.in/yaml/v4"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

// extractValidationConstraints extracts buf.validate field options and applies them to the schema.
func extractValidationConstraints(field *protogen.Field, schema *base.Schema) {
	// Get the field descriptor options
	fieldOptions := field.Desc.Options()
	if fieldOptions == nil {
		return
	}

	// Extract the buf.validate field extension
	ext := proto.GetExtension(fieldOptions, validate.E_Field)
	if ext == nil {
		return
	}

	// Type assert to FieldRules
	fieldConstraints, ok := ext.(*validate.FieldRules)
	if !ok || fieldConstraints == nil {
		return
	}

	// Apply constraints based on field type
	switch field.Desc.Kind() {
	case protoreflect.StringKind:
		applyStringConstraints(fieldConstraints, schema)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		applyInt32Constraints(fieldConstraints, schema)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		applyInt64Constraints(fieldConstraints, schema)
	case protoreflect.FloatKind:
		applyFloatConstraints(fieldConstraints, schema)
	case protoreflect.DoubleKind:
		applyDoubleConstraints(fieldConstraints, schema)
	case protoreflect.BoolKind,
		protoreflect.EnumKind,
		protoreflect.BytesKind,
		protoreflect.MessageKind,
		protoreflect.GroupKind:
		// No specific constraints for these types
	}

	// Handle repeated field constraints
	if field.Desc.IsList() {
		applyRepeatedConstraints(fieldConstraints, schema)
	}

	// Handle map field constraints
	if field.Desc.IsMap() {
		applyMapConstraints(fieldConstraints, schema)
	}

	// Handle required constraint
	// Note: Required is handled at the message level, not here
	// This is a marker for the parent message to add this field to required[]
	_ = fieldConstraints.GetRequired()
}

// applyStringConstraints applies string validation constraints to the schema.
func applyStringConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	stringConstraints := constraints.GetString()
	if stringConstraints == nil {
		return
	}

	// Min and max length
	if stringConstraints.HasMinLen() {
		minLen := int64(stringConstraints.GetMinLen()) // #nosec G115
		schema.MinLength = &minLen
	}
	if stringConstraints.HasMaxLen() {
		maxLen := int64(stringConstraints.GetMaxLen()) // #nosec G115
		schema.MaxLength = &maxLen
	}

	// Pattern (regex)
	if stringConstraints.HasPattern() {
		schema.Pattern = stringConstraints.GetPattern()
	}

	// Format constraints
	switch {
	case stringConstraints.GetEmail():
		schema.Format = "email"
	case stringConstraints.GetUuid():
		schema.Format = "uuid"
	case stringConstraints.GetUri():
		schema.Format = "uri"
	case stringConstraints.GetUriRef():
		schema.Format = "uri-reference"
	case stringConstraints.GetAddress():
		// IPv4 or IPv6 address
		schema.Format = "ip"
	case stringConstraints.GetHostname():
		schema.Format = "hostname"
	case stringConstraints.GetIp():
		schema.Format = "ip"
	case stringConstraints.GetIpv4():
		schema.Format = "ipv4"
	case stringConstraints.GetIpv6():
		schema.Format = "ipv6"
	}

	// Enum values (in constraint)
	if len(stringConstraints.GetIn()) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(stringConstraints.GetIn()))
		for _, value := range stringConstraints.GetIn() {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: value,
			})
		}
	}

	// Const value
	if stringConstraints.HasConst() {
		val := stringConstraints.GetConst()
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: val,
		}
	}
}

// applyInt32Constraints applies int32 validation constraints to the schema.
func applyInt32Constraints(constraints *validate.FieldRules, schema *base.Schema) {
	int32Constraints := constraints.GetInt32()
	if int32Constraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if int32Constraints.HasGte() {
		minValue := float64(int32Constraints.GetGte())
		schema.Minimum = &minValue
	}

	// Greater than (exclusive minimum)
	if int32Constraints.HasGt() {
		minValue := float64(int32Constraints.GetGt())
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: minValue}
	}

	// Less than or equal (maximum)
	if int32Constraints.HasLte() {
		maxValue := float64(int32Constraints.GetLte())
		schema.Maximum = &maxValue
	}

	// Less than (exclusive maximum)
	if int32Constraints.HasLt() {
		maxValue := float64(int32Constraints.GetLt())
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: maxValue}
	}

	// Const value
	if int32Constraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: strconv.Itoa(int(int32Constraints.GetConst())),
		}
	}

	// Enum values (in constraint)
	if len(int32Constraints.GetIn()) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(int32Constraints.GetIn()))
		for _, value := range int32Constraints.GetIn() {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: strconv.Itoa(int(value)),
			})
		}
	}
}

// applyInt64Constraints applies int64 validation constraints to the schema.
func applyInt64Constraints(constraints *validate.FieldRules, schema *base.Schema) {
	int64Constraints := constraints.GetInt64()
	if int64Constraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if int64Constraints.HasGte() {
		minValue := float64(int64Constraints.GetGte())
		schema.Minimum = &minValue
	}

	// Greater than (exclusive minimum)
	if int64Constraints.HasGt() {
		minValue := float64(int64Constraints.GetGt())
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: minValue}
	}

	// Less than or equal (maximum)
	if int64Constraints.HasLte() {
		maxValue := float64(int64Constraints.GetLte())
		schema.Maximum = &maxValue
	}

	// Less than (exclusive maximum)
	if int64Constraints.HasLt() {
		maxValue := float64(int64Constraints.GetLt())
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: maxValue}
	}

	// Const value
	if int64Constraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: strconv.FormatInt(int64Constraints.GetConst(), 10),
		}
	}

	// Enum values (in constraint)
	if len(int64Constraints.GetIn()) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(int64Constraints.GetIn()))
		for _, value := range int64Constraints.GetIn() {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: strconv.FormatInt(value, 10),
			})
		}
	}
}

// applyFloatConstraints applies float validation constraints to the schema.
func applyFloatConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	floatConstraints := constraints.GetFloat()
	if floatConstraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if floatConstraints.HasGte() {
		minValue := float64(floatConstraints.GetGte())
		schema.Minimum = &minValue
	}

	// Greater than (exclusive minimum)
	if floatConstraints.HasGt() {
		minValue := float64(floatConstraints.GetGt())
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: minValue}
	}

	// Less than or equal (maximum)
	if floatConstraints.HasLte() {
		maxValue := float64(floatConstraints.GetLte())
		schema.Maximum = &maxValue
	}

	// Less than (exclusive maximum)
	if floatConstraints.HasLt() {
		maxValue := float64(floatConstraints.GetLt())
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: maxValue}
	}

	// Const value
	if floatConstraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%g", floatConstraints.GetConst()),
		}
	}

	// Enum values (in constraint)
	if len(floatConstraints.GetIn()) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(floatConstraints.GetIn()))
		for _, value := range floatConstraints.GetIn() {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%g", value),
			})
		}
	}
}

// applyDoubleConstraints applies double validation constraints to the schema.
func applyDoubleConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	doubleConstraints := constraints.GetDouble()
	if doubleConstraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if doubleConstraints.HasGte() {
		minValue := doubleConstraints.GetGte()
		schema.Minimum = &minValue
	}

	// Greater than (exclusive minimum)
	if doubleConstraints.HasGt() {
		minValue := doubleConstraints.GetGt()
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: minValue}
	}

	// Less than or equal (maximum)
	if doubleConstraints.HasLte() {
		maxValue := doubleConstraints.GetLte()
		schema.Maximum = &maxValue
	}

	// Less than (exclusive maximum)
	if doubleConstraints.HasLt() {
		maxValue := doubleConstraints.GetLt()
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: maxValue}
	}

	// Const value
	if doubleConstraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%g", doubleConstraints.GetConst()),
		}
	}

	// Enum values (in constraint)
	if len(doubleConstraints.GetIn()) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(doubleConstraints.GetIn()))
		for _, value := range doubleConstraints.GetIn() {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%g", value),
			})
		}
	}
}

// applyRepeatedConstraints applies repeated field validation constraints to the schema.
func applyRepeatedConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	repeatedConstraints := constraints.GetRepeated()
	if repeatedConstraints == nil {
		return
	}

	// Min items
	if repeatedConstraints.HasMinItems() {
		minItems := int64(repeatedConstraints.GetMinItems()) // #nosec G115
		schema.MinItems = &minItems
	}

	// Max items
	if repeatedConstraints.HasMaxItems() {
		maxItems := int64(repeatedConstraints.GetMaxItems()) // #nosec G115
		schema.MaxItems = &maxItems
	}

	// Unique items
	if repeatedConstraints.GetUnique() {
		uniqueItems := true
		schema.UniqueItems = &uniqueItems
	}
}

// applyMapConstraints applies map field validation constraints to the schema.
func applyMapConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	mapConstraints := constraints.GetMap()
	if mapConstraints == nil {
		return
	}

	// Min pairs (minProperties)
	if mapConstraints.HasMinPairs() {
		minProps := int64(mapConstraints.GetMinPairs()) // #nosec G115
		schema.MinProperties = &minProps
	}

	// Max pairs (maxProperties)
	if mapConstraints.HasMaxPairs() {
		maxProps := int64(mapConstraints.GetMaxPairs()) // #nosec G115
		schema.MaxProperties = &maxProps
	}
}

// checkIfFieldRequired checks if a field has the required constraint.
func checkIfFieldRequired(field *protogen.Field) bool {
	// Get the field descriptor options
	fieldOptions := field.Desc.Options()
	if fieldOptions == nil {
		return false
	}

	// Extract the buf.validate field extension
	ext := proto.GetExtension(fieldOptions, validate.E_Field)
	if ext == nil {
		return false
	}

	// Type assert to FieldRules
	fieldConstraints, ok := ext.(*validate.FieldRules)
	if !ok || fieldConstraints == nil {
		return false
	}

	return fieldConstraints.GetRequired()
}
