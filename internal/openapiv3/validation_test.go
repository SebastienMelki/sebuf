package openapiv3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

// Test checkIfFieldRequired function
func TestCheckIfFieldRequired(t *testing.T) {
	tests := []struct {
		name     string
		required bool
		expected bool
	}{
		{
			name:     "Required field",
			required: true,
			expected: true,
		},
		{
			name:     "Optional field",
			required: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &protogen.Field{
				Desc: &mockFieldDescWithValidation{
					name:     "test_field",
					kind:     protoreflect.StringKind,
					required: tt.required,
				},
			}

			result := checkIfFieldRequired(field)
			if result != tt.expected {
				t.Errorf("checkIfFieldRequired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test applyStringConstraints function
func TestApplyStringConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints *validate.StringRules
		checkFn     func(*base.Schema) error
	}{
		{
			name: "Min and max length",
			constraints: &validate.StringRules{
				MinLen: proto.Uint64(5),
				MaxLen: proto.Uint64(100),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MinLength == nil || *schema.MinLength != 5 {
					t.Errorf("Expected MinLength 5, got %v", schema.MinLength)
				}
				if schema.MaxLength == nil || *schema.MaxLength != 100 {
					t.Errorf("Expected MaxLength 100, got %v", schema.MaxLength)
				}
				return nil
			},
		},
		{
			name: "Pattern constraint",
			constraints: &validate.StringRules{
				Pattern: proto.String("^[a-zA-Z0-9_]+$"),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Pattern != "^[a-zA-Z0-9_]+$" {
					t.Errorf("Expected pattern '^[a-zA-Z0-9_]+$', got %s", schema.Pattern)
				}
				return nil
			},
		},
		{
			name: "Email format",
			constraints: &validate.StringRules{
				Email: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "email" {
					t.Errorf("Expected format 'email', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "UUID format",
			constraints: &validate.StringRules{
				Uuid: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "uuid" {
					t.Errorf("Expected format 'uuid', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "URI format",
			constraints: &validate.StringRules{
				Uri: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "uri" {
					t.Errorf("Expected format 'uri', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "URI reference format",
			constraints: &validate.StringRules{
				UriRef: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "uri-reference" {
					t.Errorf("Expected format 'uri-reference', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "IPv4 format",
			constraints: &validate.StringRules{
				Ipv4: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "ipv4" {
					t.Errorf("Expected format 'ipv4', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "IPv6 format",
			constraints: &validate.StringRules{
				Ipv6: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "ipv6" {
					t.Errorf("Expected format 'ipv6', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "Hostname format",
			constraints: &validate.StringRules{
				Hostname: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "hostname" {
					t.Errorf("Expected format 'hostname', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "IP format (generic)",
			constraints: &validate.StringRules{
				Ip: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "ip" {
					t.Errorf("Expected format 'ip', got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "Address format",
			constraints: &validate.StringRules{
				Address: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "ip" {
					t.Errorf("Expected format 'ip' for address, got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name: "Const value",
			constraints: &validate.StringRules{
				Const: proto.String("fixed_value"),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Const == nil || schema.Const.Value != "fixed_value" {
					t.Errorf("Expected const 'fixed_value', got %v", schema.Const)
				}
				return nil
			},
		},
		{
			name: "In constraint (enum values)",
			constraints: &validate.StringRules{
				In: []string{"value1", "value2", "value3"},
			},
			checkFn: func(schema *base.Schema) error {
				if len(schema.Enum) != 3 {
					t.Errorf("Expected 3 enum values, got %d", len(schema.Enum))
				}
				expectedValues := []string{"value1", "value2", "value3"}
				for i, expected := range expectedValues {
					if i >= len(schema.Enum) || schema.Enum[i].Value != expected {
						t.Errorf("Expected enum value %s at index %d", expected, i)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{
				Type: []string{"string"},
			}

			fieldRules := &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: tt.constraints,
				},
			}

			applyStringConstraints(fieldRules, schema)

			if tt.checkFn != nil {
				if err := tt.checkFn(schema); err != nil {
					t.Errorf("Check function failed: %v", err)
				}
			}
		})
	}
}

// Test applyInt32Constraints function
func TestApplyInt32Constraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints *validate.Int32Rules
		checkFn     func(*base.Schema) error
	}{
		{
			name: "Greater than or equal (gte)",
			constraints: &validate.Int32Rules{
				Gte: proto.Int32(10),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 10.0 {
					t.Errorf("Expected minimum 10, got %v", schema.Minimum)
				}
				return nil
			},
		},
		{
			name: "Greater than (gt)",
			constraints: &validate.Int32Rules{
				Gt: proto.Int32(5),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.ExclusiveMinimum == nil || schema.ExclusiveMinimum.B != 5.0 {
					t.Errorf("Expected exclusive minimum 5, got %v", schema.ExclusiveMinimum)
				}
				return nil
			},
		},
		{
			name: "Less than or equal (lte)",
			constraints: &validate.Int32Rules{
				Lte: proto.Int32(100),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Maximum == nil || *schema.Maximum != 100.0 {
					t.Errorf("Expected maximum 100, got %v", schema.Maximum)
				}
				return nil
			},
		},
		{
			name: "Less than (lt)",
			constraints: &validate.Int32Rules{
				Lt: proto.Int32(50),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.ExclusiveMaximum == nil || schema.ExclusiveMaximum.B != 50.0 {
					t.Errorf("Expected exclusive maximum 50, got %v", schema.ExclusiveMaximum)
				}
				return nil
			},
		},
		{
			name: "Const value",
			constraints: &validate.Int32Rules{
				Const: proto.Int32(42),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Const == nil || schema.Const.Value != "42" {
					t.Errorf("Expected const '42', got %v", schema.Const)
				}
				return nil
			},
		},
		{
			name: "In constraint",
			constraints: &validate.Int32Rules{
				In: []int32{1, 2, 3, 5, 8},
			},
			checkFn: func(schema *base.Schema) error {
				if len(schema.Enum) != 5 {
					t.Errorf("Expected 5 enum values, got %d", len(schema.Enum))
				}
				expectedValues := []string{"1", "2", "3", "5", "8"}
				for i, expected := range expectedValues {
					if i >= len(schema.Enum) || schema.Enum[i].Value != expected {
						t.Errorf("Expected enum value %s at index %d", expected, i)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{
				Type: []string{"integer"},
			}

			fieldRules := &validate.FieldRules{
				Type: &validate.FieldRules_Int32{
					Int32: tt.constraints,
				},
			}

			applyInt32Constraints(fieldRules, schema)

			if tt.checkFn != nil {
				if err := tt.checkFn(schema); err != nil {
					t.Errorf("Check function failed: %v", err)
				}
			}
		})
	}
}

// Test applyRepeatedConstraints function
func TestApplyRepeatedConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints *validate.RepeatedRules
		checkFn     func(*base.Schema) error
	}{
		{
			name: "Min items",
			constraints: &validate.RepeatedRules{
				MinItems: proto.Uint64(2),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MinItems == nil || *schema.MinItems != 2 {
					t.Errorf("Expected MinItems 2, got %v", schema.MinItems)
				}
				return nil
			},
		},
		{
			name: "Max items",
			constraints: &validate.RepeatedRules{
				MaxItems: proto.Uint64(10),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MaxItems == nil || *schema.MaxItems != 10 {
					t.Errorf("Expected MaxItems 10, got %v", schema.MaxItems)
				}
				return nil
			},
		},
		{
			name: "Unique items",
			constraints: &validate.RepeatedRules{
				Unique: proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.UniqueItems == nil || !*schema.UniqueItems {
					t.Errorf("Expected UniqueItems true, got %v", schema.UniqueItems)
				}
				return nil
			},
		},
		{
			name: "Combined constraints",
			constraints: &validate.RepeatedRules{
				MinItems: proto.Uint64(1),
				MaxItems: proto.Uint64(5),
				Unique:   proto.Bool(true),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MinItems == nil || *schema.MinItems != 1 {
					t.Errorf("Expected MinItems 1, got %v", schema.MinItems)
				}
				if schema.MaxItems == nil || *schema.MaxItems != 5 {
					t.Errorf("Expected MaxItems 5, got %v", schema.MaxItems)
				}
				if schema.UniqueItems == nil || !*schema.UniqueItems {
					t.Errorf("Expected UniqueItems true, got %v", schema.UniqueItems)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{
				Type: []string{"array"},
			}

			applyRepeatedConstraints(&validate.FieldRules{
				Type: &validate.FieldRules_Repeated{
					Repeated: tt.constraints,
				},
			}, schema)

			if tt.checkFn != nil {
				if err := tt.checkFn(schema); err != nil {
					t.Errorf("Check function failed: %v", err)
				}
			}
		})
	}
}

// Test applyMapConstraints function
func TestApplyMapConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints *validate.MapRules
		checkFn     func(*base.Schema) error
	}{
		{
			name: "Min pairs",
			constraints: &validate.MapRules{
				MinPairs: proto.Uint64(1),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MinProperties == nil || *schema.MinProperties != 1 {
					t.Errorf("Expected MinProperties 1, got %v", schema.MinProperties)
				}
				return nil
			},
		},
		{
			name: "Max pairs",
			constraints: &validate.MapRules{
				MaxPairs: proto.Uint64(20),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MaxProperties == nil || *schema.MaxProperties != 20 {
					t.Errorf("Expected MaxProperties 20, got %v", schema.MaxProperties)
				}
				return nil
			},
		},
		{
			name: "Combined constraints",
			constraints: &validate.MapRules{
				MinPairs: proto.Uint64(2),
				MaxPairs: proto.Uint64(15),
			},
			checkFn: func(schema *base.Schema) error {
				if schema.MinProperties == nil || *schema.MinProperties != 2 {
					t.Errorf("Expected MinProperties 2, got %v", schema.MinProperties)
				}
				if schema.MaxProperties == nil || *schema.MaxProperties != 15 {
					t.Errorf("Expected MaxProperties 15, got %v", schema.MaxProperties)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{
				Type: []string{"object"},
			}

			applyMapConstraints(&validate.FieldRules{
				Type: &validate.FieldRules_Map{
					Map: tt.constraints,
				},
			}, schema)

			if tt.checkFn != nil {
				if err := tt.checkFn(schema); err != nil {
					t.Errorf("Check function failed: %v", err)
				}
			}
		})
	}
}

// Test extractValidationConstraints function integration
func TestExtractValidationConstraints(t *testing.T) {
	tests := []struct {
		name      string
		fieldKind protoreflect.Kind
		rules     *validate.FieldRules
		checkFn   func(*base.Schema) error
	}{
		{
			name:      "String field with email validation",
			fieldKind: protoreflect.StringKind,
			rules: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Email: proto.Bool(true),
					},
				},
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Format != "email" {
					t.Errorf("Expected email format, got %s", schema.Format)
				}
				return nil
			},
		},
		{
			name:      "Int32 field with range validation",
			fieldKind: protoreflect.Int32Kind,
			rules: &validate.FieldRules{
				Type: &validate.FieldRules_Int32{
					Int32: &validate.Int32Rules{
						Gte: proto.Int32(0),
						Lte: proto.Int32(120),
					},
				},
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 0.0 {
					t.Errorf("Expected minimum 0, got %v", schema.Minimum)
				}
				if schema.Maximum == nil || *schema.Maximum != 120.0 {
					t.Errorf("Expected maximum 120, got %v", schema.Maximum)
				}
				return nil
			},
		},
		{
			name:      "Float field with range validation",
			fieldKind: protoreflect.FloatKind,
			rules: &validate.FieldRules{
				Type: &validate.FieldRules_Float{
					Float: &validate.FloatRules{
						Gte: proto.Float32(0.0),
						Lte: proto.Float32(100.0),
					},
				},
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 0.0 {
					t.Errorf("Expected minimum 0, got %v", schema.Minimum)
				}
				if schema.Maximum == nil || *schema.Maximum != 100.0 {
					t.Errorf("Expected maximum 100, got %v", schema.Maximum)
				}
				return nil
			},
		},
		{
			name:      "Double field with range validation",
			fieldKind: protoreflect.DoubleKind,
			rules: &validate.FieldRules{
				Type: &validate.FieldRules_Double{
					Double: &validate.DoubleRules{
						Gte: proto.Float64(-90.0),
						Lte: proto.Float64(90.0),
					},
				},
			},
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != -90.0 {
					t.Errorf("Expected minimum -90, got %v", schema.Minimum)
				}
				if schema.Maximum == nil || *schema.Maximum != 90.0 {
					t.Errorf("Expected maximum 90, got %v", schema.Maximum)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &protogen.Field{
				Desc: &mockFieldDescWithValidation{
					name:  "test_field",
					kind:  tt.fieldKind,
					rules: tt.rules,
				},
			}

			schema := &base.Schema{
				Type: []string{"string"},
			}

			extractValidationConstraints(field, schema)

			if tt.checkFn != nil {
				if err := tt.checkFn(schema); err != nil {
					t.Errorf("Check function failed: %v", err)
				}
			}
		})
	}
}

// Test field without validation constraints
func TestExtractValidationConstraintsNoConstraints(t *testing.T) {
	field := &protogen.Field{
		Desc: &mockFieldDescWithValidation{
			name:        "test_field",
			kind:        protoreflect.StringKind,
			hasValidation: false,
		},
	}

	schema := &base.Schema{
		Type: []string{"string"},
	}

	// Should not panic or modify schema when no constraints
	extractValidationConstraints(field, schema)

	// Schema should remain unchanged
	if len(schema.Type) != 1 || schema.Type[0] != "string" {
		t.Errorf("Schema was modified when no validation constraints present")
	}
}

// === Mock implementations for validation testing ===

type mockFieldDescWithValidation struct {
	name          string
	kind          protoreflect.Kind
	required      bool
	hasValidation bool
	rules         *validate.FieldRules
}

func (f *mockFieldDescWithValidation) Name() protoreflect.Name { return protoreflect.Name(f.name) }
func (f *mockFieldDescWithValidation) FullName() protoreflect.FullName {
	return protoreflect.FullName(f.name)
}
func (f *mockFieldDescWithValidation) IsPlaceholder() bool { return false }
func (f *mockFieldDescWithValidation) Options() protoreflect.ProtoMessage {
	options := &descriptorpb.FieldOptions{}
	if f.hasValidation || f.rules != nil {
		// Create field rules
		rules := f.rules
		if rules == nil && f.required {
			rules = &validate.FieldRules{
				Required: true,
			}
		}
		if rules != nil {
			proto.SetExtension(options, validate.E_Field, rules)
		}
	}
	return options
}
func (f *mockFieldDescWithValidation) Index() int { return 0 }
func (f *mockFieldDescWithValidation) Syntax() protoreflect.Syntax { return protoreflect.Proto3 }
func (f *mockFieldDescWithValidation) Number() protoreflect.FieldNumber { return 1 }
func (f *mockFieldDescWithValidation) Cardinality() protoreflect.Cardinality {
	return protoreflect.Optional
}
func (f *mockFieldDescWithValidation) Kind() protoreflect.Kind { return f.kind }
func (f *mockFieldDescWithValidation) HasJSONName() bool       { return false }
func (f *mockFieldDescWithValidation) JSONName() string       { return f.name }
func (f *mockFieldDescWithValidation) TextName() string       { return f.name }
func (f *mockFieldDescWithValidation) HasPresence() bool      { return false }
func (f *mockFieldDescWithValidation) IsExtension() bool      { return false }
func (f *mockFieldDescWithValidation) IsWeak() bool           { return false }
func (f *mockFieldDescWithValidation) IsPacked() bool         { return false }
func (f *mockFieldDescWithValidation) IsList() bool           { return false }
func (f *mockFieldDescWithValidation) IsMap() bool            { return false }
func (f *mockFieldDescWithValidation) MapKey() protoreflect.FieldDescriptor { return nil }
func (f *mockFieldDescWithValidation) MapValue() protoreflect.FieldDescriptor { return nil }
func (f *mockFieldDescWithValidation) HasDefault() bool { return false }
func (f *mockFieldDescWithValidation) Default() protoreflect.Value { return protoreflect.Value{} }
func (f *mockFieldDescWithValidation) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	return nil
}
func (f *mockFieldDescWithValidation) ContainingOneof() protoreflect.OneofDescriptor { return nil }
func (f *mockFieldDescWithValidation) ContainingMessage() protoreflect.MessageDescriptor {
	return nil
}
func (f *mockFieldDescWithValidation) Enum() protoreflect.EnumDescriptor { return nil }
func (f *mockFieldDescWithValidation) Message() protoreflect.MessageDescriptor { return nil }
func (f *mockFieldDescWithValidation) Parent() protoreflect.Descriptor { return nil }
func (f *mockFieldDescWithValidation) ParentFile() protoreflect.FileDescriptor { return nil }
func (f *mockFieldDescWithValidation) HasOptionalKeyword() bool { return false }