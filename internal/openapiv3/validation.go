package openapiv3

import (
	"fmt"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

// extractValidationConstraints extracts buf.validate field options and applies them to the schema
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
	if fieldConstraints.GetRequired() {
		// Note: Required is handled at the message level, not here
		// This is a marker for the parent message to add this field to required[]
	}
}

// applyStringConstraints applies string validation constraints to the schema
func applyStringConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	stringConstraints := constraints.GetString_()
	if stringConstraints == nil {
		return
	}

	// Min and max length
	if stringConstraints.HasMinLen() {
		minLen := int64(stringConstraints.GetMinLen())
		schema.MinLength = &minLen
	}
	if stringConstraints.HasMaxLen() {
		maxLen := int64(stringConstraints.GetMaxLen())
		schema.MaxLength = &maxLen
	}

	// Pattern (regex)
	if stringConstraints.HasPattern() {
		schema.Pattern = stringConstraints.GetPattern()
	}

	// Format constraints
	if stringConstraints.GetEmail() {
		schema.Format = "email"
	} else if stringConstraints.GetUuid() {
		schema.Format = "uuid"
	} else if stringConstraints.GetUri() {
		schema.Format = "uri"
	} else if stringConstraints.GetUriRef() {
		schema.Format = "uri-reference"
	} else if stringConstraints.GetAddress() {
		// IPv4 or IPv6 address
		schema.Format = "ip"
	} else if stringConstraints.GetHostname() {
		schema.Format = "hostname"
	} else if stringConstraints.GetIp() {
		schema.Format = "ip"
	} else if stringConstraints.GetIpv4() {
		schema.Format = "ipv4"
	} else if stringConstraints.GetIpv6() {
		schema.Format = "ipv6"
	}

	// Enum values (in constraint)
	if len(stringConstraints.In) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(stringConstraints.In))
		for _, value := range stringConstraints.In {
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

// applyInt32Constraints applies int32 validation constraints to the schema
func applyInt32Constraints(constraints *validate.FieldRules, schema *base.Schema) {
	int32Constraints := constraints.GetInt32()
	if int32Constraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if int32Constraints.HasGte() {
		min := float64(int32Constraints.GetGte())
		schema.Minimum = &min
	}

	// Greater than (exclusive minimum)
	if int32Constraints.HasGt() {
		min := float64(int32Constraints.GetGt())
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: min}
	}

	// Less than or equal (maximum)
	if int32Constraints.HasLte() {
		max := float64(int32Constraints.GetLte())
		schema.Maximum = &max
	}

	// Less than (exclusive maximum)
	if int32Constraints.HasLt() {
		max := float64(int32Constraints.GetLt())
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: max}
	}

	// Const value
	if int32Constraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%d", int32Constraints.GetConst()),
		}
	}

	// Enum values (in constraint)
	if len(int32Constraints.In) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(int32Constraints.In))
		for _, value := range int32Constraints.In {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%d", value),
			})
		}
	}
}

// applyInt64Constraints applies int64 validation constraints to the schema
func applyInt64Constraints(constraints *validate.FieldRules, schema *base.Schema) {
	int64Constraints := constraints.GetInt64()
	if int64Constraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if int64Constraints.HasGte() {
		min := float64(int64Constraints.GetGte())
		schema.Minimum = &min
	}

	// Greater than (exclusive minimum)
	if int64Constraints.HasGt() {
		min := float64(int64Constraints.GetGt())
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: min}
	}

	// Less than or equal (maximum)
	if int64Constraints.HasLte() {
		max := float64(int64Constraints.GetLte())
		schema.Maximum = &max
	}

	// Less than (exclusive maximum)
	if int64Constraints.HasLt() {
		max := float64(int64Constraints.GetLt())
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: max}
	}

	// Const value
	if int64Constraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%d", int64Constraints.GetConst()),
		}
	}

	// Enum values (in constraint)
	if len(int64Constraints.In) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(int64Constraints.In))
		for _, value := range int64Constraints.In {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%d", value),
			})
		}
	}
}

// applyFloatConstraints applies float validation constraints to the schema
func applyFloatConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	floatConstraints := constraints.GetFloat()
	if floatConstraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if floatConstraints.HasGte() {
		min := float64(floatConstraints.GetGte())
		schema.Minimum = &min
	}

	// Greater than (exclusive minimum)
	if floatConstraints.HasGt() {
		min := float64(floatConstraints.GetGt())
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: min}
	}

	// Less than or equal (maximum)
	if floatConstraints.HasLte() {
		max := float64(floatConstraints.GetLte())
		schema.Maximum = &max
	}

	// Less than (exclusive maximum)
	if floatConstraints.HasLt() {
		max := float64(floatConstraints.GetLt())
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: max}
	}

	// Const value
	if floatConstraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%g", floatConstraints.GetConst()),
		}
	}

	// Enum values (in constraint)
	if len(floatConstraints.In) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(floatConstraints.In))
		for _, value := range floatConstraints.In {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%g", value),
			})
		}
	}
}

// applyDoubleConstraints applies double validation constraints to the schema
func applyDoubleConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	doubleConstraints := constraints.GetDouble()
	if doubleConstraints == nil {
		return
	}

	// Greater than or equal (minimum)
	if doubleConstraints.HasGte() {
		min := doubleConstraints.GetGte()
		schema.Minimum = &min
	}

	// Greater than (exclusive minimum)
	if doubleConstraints.HasGt() {
		min := doubleConstraints.GetGt()
		schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{B: min}
	}

	// Less than or equal (maximum)
	if doubleConstraints.HasLte() {
		max := doubleConstraints.GetLte()
		schema.Maximum = &max
	}

	// Less than (exclusive maximum)
	if doubleConstraints.HasLt() {
		max := doubleConstraints.GetLt()
		schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{B: max}
	}

	// Const value
	if doubleConstraints.HasConst() {
		schema.Const = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%g", doubleConstraints.GetConst()),
		}
	}

	// Enum values (in constraint)
	if len(doubleConstraints.In) > 0 {
		schema.Enum = make([]*yaml.Node, 0, len(doubleConstraints.In))
		for _, value := range doubleConstraints.In {
			schema.Enum = append(schema.Enum, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%g", value),
			})
		}
	}
}

// applyRepeatedConstraints applies repeated field validation constraints to the schema
func applyRepeatedConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	repeatedConstraints := constraints.GetRepeated()
	if repeatedConstraints == nil {
		return
	}

	// Min items
	if repeatedConstraints.HasMinItems() {
		minItems := int64(repeatedConstraints.GetMinItems())
		schema.MinItems = &minItems
	}

	// Max items
	if repeatedConstraints.HasMaxItems() {
		maxItems := int64(repeatedConstraints.GetMaxItems())
		schema.MaxItems = &maxItems
	}

	// Unique items
	if repeatedConstraints.GetUnique() {
		uniqueItems := true
		schema.UniqueItems = &uniqueItems
	}
}

// applyMapConstraints applies map field validation constraints to the schema
func applyMapConstraints(constraints *validate.FieldRules, schema *base.Schema) {
	mapConstraints := constraints.GetMap()
	if mapConstraints == nil {
		return
	}

	// Min pairs (minProperties)
	if mapConstraints.HasMinPairs() {
		minProps := int64(mapConstraints.GetMinPairs())
		schema.MinProperties = &minProps
	}

	// Max pairs (maxProperties)
	if mapConstraints.HasMaxPairs() {
		maxProps := int64(mapConstraints.GetMaxPairs())
		schema.MaxProperties = &maxProps
	}
}

// checkIfFieldRequired checks if a field has the required constraint
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
