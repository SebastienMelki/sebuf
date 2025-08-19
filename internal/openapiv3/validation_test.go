package openapiv3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

func TestApplyStringConstraints(t *testing.T) {
	tests := []struct {
		name           string
		constraints    *validate.FieldRules
		expectedMinLen *int64
		expectedMaxLen *int64
		expectedPattern string
		expectedFormat string
		expectedEnum   []string
		expectedConst  string
	}{
		{
			name: "min and max length",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						MinLen: proto.Uint64(3),
						MaxLen: proto.Uint64(20),
					},
				},
			},
			expectedMinLen: func() *int64 { v := int64(3); return &v }(),
			expectedMaxLen: func() *int64 { v := int64(20); return &v }(),
		},
		{
			name: "email format",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Email: proto.Bool(true),
					},
				},
			},
			expectedFormat: "email",
		},
		{
			name: "uuid format",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Uuid: proto.Bool(true),
					},
				},
			},
			expectedFormat: "uuid",
		},
		{
			name: "uri format",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Uri: proto.Bool(true),
					},
				},
			},
			expectedFormat: "uri",
		},
		{
			name: "hostname format",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Hostname: proto.Bool(true),
					},
				},
			},
			expectedFormat: "hostname",
		},
		{
			name: "ipv4 format",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Ipv4: proto.Bool(true),
					},
				},
			},
			expectedFormat: "ipv4",
		},
		{
			name: "ipv6 format",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Ipv6: proto.Bool(true),
					},
				},
			},
			expectedFormat: "ipv6",
		},
		{
			name: "pattern constraint",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Pattern: proto.String("^[a-zA-Z0-9]+$"),
					},
				},
			},
			expectedPattern: "^[a-zA-Z0-9]+$",
		},
		{
			name: "enum values (in constraint)",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						In: []string{"active", "inactive", "pending"},
					},
				},
			},
			expectedEnum: []string{"active", "inactive", "pending"},
		},
		{
			name: "const value",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_String_{
					String_: &validate.StringRules{
						Const: proto.String("v1.0.0"),
					},
				},
			},
			expectedConst: "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{}
			applyStringConstraints(tt.constraints, schema)
			
			if tt.expectedMinLen != nil {
				require.NotNil(t, schema.MinLength)
				assert.Equal(t, *tt.expectedMinLen, *schema.MinLength)
			}
			
			if tt.expectedMaxLen != nil {
				require.NotNil(t, schema.MaxLength)
				assert.Equal(t, *tt.expectedMaxLen, *schema.MaxLength)
			}
			
			if tt.expectedPattern != "" {
				assert.Equal(t, tt.expectedPattern, schema.Pattern)
			}
			
			if tt.expectedFormat != "" {
				assert.Equal(t, tt.expectedFormat, schema.Format)
			}
			
			if tt.expectedEnum != nil {
				assert.Equal(t, len(tt.expectedEnum), len(schema.Enum))
				for i, enumValue := range tt.expectedEnum {
					assert.Equal(t, enumValue, schema.Enum[i].Value)
				}
			}
			
			if tt.expectedConst != "" {
				require.NotNil(t, schema.Const)
				assert.Equal(t, tt.expectedConst, schema.Const.Value)
			}
		})
	}
}

func TestApplyInt32Constraints(t *testing.T) {
	tests := []struct {
		name               string
		constraints        *validate.FieldRules
		expectedMin        *float64
		expectedMax        *float64
		expectedExclusiveMin *float64
		expectedExclusiveMax *float64
		expectedConst      int32
		expectedEnum       []int32
	}{
		{
			name: "min and max (gte/lte)",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Int32{
					Int32: &validate.Int32Rules{
						Gte: proto.Int32(18),
						Lte: proto.Int32(120),
					},
				},
			},
			expectedMin: func() *float64 { v := 18.0; return &v }(),
			expectedMax: func() *float64 { v := 120.0; return &v }(),
		},
		{
			name: "exclusive min and max (gt/lt)",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Int32{
					Int32: &validate.Int32Rules{
						Gt: proto.Int32(0),
						Lt: proto.Int32(100),
					},
				},
			},
			expectedExclusiveMin: func() *float64 { v := 0.0; return &v }(),
			expectedExclusiveMax: func() *float64 { v := 100.0; return &v }(),
		},
		{
			name: "const value",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Int32{
					Int32: &validate.Int32Rules{
						Const: proto.Int32(42),
					},
				},
			},
			expectedConst: 42,
		},
		{
			name: "enum values",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Int32{
					Int32: &validate.Int32Rules{
						In: []int32{1, 2, 3, 5, 8},
					},
				},
			},
			expectedEnum: []int32{1, 2, 3, 5, 8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{}
			applyInt32Constraints(tt.constraints, schema)
			
			if tt.expectedMin != nil {
				require.NotNil(t, schema.Minimum)
				assert.Equal(t, *tt.expectedMin, *schema.Minimum)
			}
			
			if tt.expectedMax != nil {
				require.NotNil(t, schema.Maximum)
				assert.Equal(t, *tt.expectedMax, *schema.Maximum)
			}
			
			if tt.expectedExclusiveMin != nil {
				require.NotNil(t, schema.ExclusiveMinimum)
				assert.Equal(t, *tt.expectedExclusiveMin, schema.ExclusiveMinimum.B)
			}
			
			if tt.expectedExclusiveMax != nil {
				require.NotNil(t, schema.ExclusiveMaximum)
				assert.Equal(t, *tt.expectedExclusiveMax, schema.ExclusiveMaximum.B)
			}
			
			if tt.expectedConst != 0 {
				require.NotNil(t, schema.Const)
				assert.Equal(t, "42", schema.Const.Value)
			}
			
			if tt.expectedEnum != nil {
				assert.Equal(t, len(tt.expectedEnum), len(schema.Enum))
			}
		})
	}
}

func TestApplyRepeatedConstraints(t *testing.T) {
	tests := []struct {
		name             string
		constraints      *validate.FieldRules
		expectedMinItems *int64
		expectedMaxItems *int64
		expectedUnique   bool
	}{
		{
			name: "min and max items",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Repeated{
					Repeated: &validate.RepeatedRules{
						MinItems: proto.Uint64(1),
						MaxItems: proto.Uint64(10),
					},
				},
			},
			expectedMinItems: func() *int64 { v := int64(1); return &v }(),
			expectedMaxItems: func() *int64 { v := int64(10); return &v }(),
		},
		{
			name: "unique items",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Repeated{
					Repeated: &validate.RepeatedRules{
						Unique: proto.Bool(true),
					},
				},
			},
			expectedUnique: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{}
			applyRepeatedConstraints(tt.constraints, schema)
			
			if tt.expectedMinItems != nil {
				require.NotNil(t, schema.MinItems)
				assert.Equal(t, *tt.expectedMinItems, *schema.MinItems)
			}
			
			if tt.expectedMaxItems != nil {
				require.NotNil(t, schema.MaxItems)
				assert.Equal(t, *tt.expectedMaxItems, *schema.MaxItems)
			}
			
			if tt.expectedUnique {
				require.NotNil(t, schema.UniqueItems)
				assert.True(t, *schema.UniqueItems)
			}
		})
	}
}

func TestApplyMapConstraints(t *testing.T) {
	tests := []struct {
		name                  string
		constraints          *validate.FieldRules
		expectedMinProperties *int64
		expectedMaxProperties *int64
	}{
		{
			name: "min and max pairs",
			constraints: &validate.FieldRules{
				Type: &validate.FieldRules_Map{
					Map: &validate.MapRules{
						MinPairs: proto.Uint64(1),
						MaxPairs: proto.Uint64(20),
					},
				},
			},
			expectedMinProperties: func() *int64 { v := int64(1); return &v }(),
			expectedMaxProperties: func() *int64 { v := int64(20); return &v }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &base.Schema{}
			applyMapConstraints(tt.constraints, schema)
			
			if tt.expectedMinProperties != nil {
				require.NotNil(t, schema.MinProperties)
				assert.Equal(t, *tt.expectedMinProperties, *schema.MinProperties)
			}
			
			if tt.expectedMaxProperties != nil {
				require.NotNil(t, schema.MaxProperties)
				assert.Equal(t, *tt.expectedMaxProperties, *schema.MaxProperties)
			}
		})
	}
}

func TestCheckIfFieldRequired(t *testing.T) {
	tests := []struct {
		name         string
		fieldOptions *descriptorpb.FieldOptions
		expected     bool
	}{
		{
			name:         "no options",
			fieldOptions: nil,
			expected:     false,
		},
		{
			name:         "empty options",
			fieldOptions: &descriptorpb.FieldOptions{},
			expected:     false,
		},
		{
			name: "required field",
			fieldOptions: func() *descriptorpb.FieldOptions {
				opts := &descriptorpb.FieldOptions{}
				proto.SetExtension(opts, validate.E_Field, &validate.FieldRules{
					Required: proto.Bool(true),
				})
				return opts
			}(),
			expected: true,
		},
		{
			name: "not required field",
			fieldOptions: func() *descriptorpb.FieldOptions {
				opts := &descriptorpb.FieldOptions{}
				proto.SetExtension(opts, validate.E_Field, &validate.FieldRules{
					Required: proto.Bool(false),
				})
				return opts
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &protogen.Field{
				Desc: &mockFieldDescriptor{
					options: tt.fieldOptions,
				},
			}
			
			result := checkIfFieldRequired(field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractValidationConstraints(t *testing.T) {
	tests := []struct {
		name        string
		fieldKind   protoreflect.Kind
		fieldOptions *descriptorpb.FieldOptions
		isList      bool
		isMap       bool
		checkSchema func(t *testing.T, schema *base.Schema)
	}{
		{
			name:      "string field with constraints",
			fieldKind: protoreflect.StringKind,
			fieldOptions: func() *descriptorpb.FieldOptions {
				opts := &descriptorpb.FieldOptions{}
				proto.SetExtension(opts, validate.E_Field, &validate.FieldRules{
					Type: &validate.FieldRules_String_{
						String_: &validate.StringRules{
							MinLen: proto.Uint64(5),
							MaxLen: proto.Uint64(50),
						},
					},
				})
				return opts
			}(),
			checkSchema: func(t *testing.T, schema *base.Schema) {
				require.NotNil(t, schema.MinLength)
				require.NotNil(t, schema.MaxLength)
				assert.Equal(t, int64(5), *schema.MinLength)
				assert.Equal(t, int64(50), *schema.MaxLength)
			},
		},
		{
			name:      "repeated field with constraints",
			fieldKind: protoreflect.StringKind,
			isList:    true,
			fieldOptions: func() *descriptorpb.FieldOptions {
				opts := &descriptorpb.FieldOptions{}
				proto.SetExtension(opts, validate.E_Field, &validate.FieldRules{
					Type: &validate.FieldRules_Repeated{
						Repeated: &validate.RepeatedRules{
							MinItems: proto.Uint64(1),
							MaxItems: proto.Uint64(5),
						},
					},
				})
				return opts
			}(),
			checkSchema: func(t *testing.T, schema *base.Schema) {
				require.NotNil(t, schema.MinItems)
				require.NotNil(t, schema.MaxItems)
				assert.Equal(t, int64(1), *schema.MinItems)
				assert.Equal(t, int64(5), *schema.MaxItems)
			},
		},
		{
			name:      "map field with constraints",
			fieldKind: protoreflect.MessageKind,
			isMap:     true,
			fieldOptions: func() *descriptorpb.FieldOptions {
				opts := &descriptorpb.FieldOptions{}
				proto.SetExtension(opts, validate.E_Field, &validate.FieldRules{
					Type: &validate.FieldRules_Map{
						Map: &validate.MapRules{
							MinPairs: proto.Uint64(1),
							MaxPairs: proto.Uint64(10),
						},
					},
				})
				return opts
			}(),
			checkSchema: func(t *testing.T, schema *base.Schema) {
				require.NotNil(t, schema.MinProperties)
				require.NotNil(t, schema.MaxProperties)
				assert.Equal(t, int64(1), *schema.MinProperties)
				assert.Equal(t, int64(10), *schema.MaxProperties)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &protogen.Field{
				Desc: &mockFieldDescriptor{
					kind:    tt.fieldKind,
					isList:  tt.isList,
					isMap:   tt.isMap,
					options: tt.fieldOptions,
				},
			}
			
			schema := &base.Schema{}
			extractValidationConstraints(field, schema)
			
			if tt.checkSchema != nil {
				tt.checkSchema(t, schema)
			}
		})
	}
}