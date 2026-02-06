package annotations

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// OneofDiscriminatorInfo holds parsed oneof discriminator configuration for a single oneof.
type OneofDiscriminatorInfo struct {
	Oneof         *protogen.Oneof
	Discriminator string         // JSON field name for discriminator (e.g., "type")
	Flatten       bool           // Whether to flatten variant fields to parent level
	Variants      []OneofVariant // Resolved variant info
}

// OneofVariant holds information about a single oneof variant.
type OneofVariant struct {
	Field            *protogen.Field
	DiscriminatorVal string // Value for this variant in the discriminator field
	IsMessage        bool   // Whether variant is a message type (required for flatten)
}

// GetOneofConfig returns the OneofConfig for a oneof, or nil if not annotated.
// Returns nil if discriminator is empty (annotation is treated as absent).
func GetOneofConfig(oneof *protogen.Oneof) *http.OneofConfig {
	options := oneof.Desc.Options()
	if options == nil {
		return nil
	}

	oneofOptions, ok := options.(*descriptorpb.OneofOptions)
	if !ok {
		return nil
	}

	if !proto.HasExtension(oneofOptions, http.E_OneofConfig) {
		return nil
	}

	ext := proto.GetExtension(oneofOptions, http.E_OneofConfig)

	config, ok := ext.(*http.OneofConfig)
	if !ok || config == nil {
		return nil
	}

	if config.GetDiscriminator() == "" {
		return nil // No discriminator = no config
	}

	return config
}

// GetOneofVariantValue returns the custom discriminator value for a oneof variant field.
// Returns empty string if not set (caller should use the proto field name as default).
func GetOneofVariantValue(field *protogen.Field) string {
	options := field.Desc.Options()
	if options == nil {
		return ""
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return ""
	}

	if !proto.HasExtension(fieldOptions, http.E_OneofValue) {
		return ""
	}

	ext := proto.GetExtension(fieldOptions, http.E_OneofValue)

	value, ok := ext.(string)
	if !ok {
		return ""
	}

	return value
}

// GetOneofDiscriminatorInfo resolves the full discriminator info for a oneof.
// Returns nil if the oneof has no oneof_config annotation.
// For each variant, uses oneof_value if set, otherwise proto field name.
func GetOneofDiscriminatorInfo(oneof *protogen.Oneof) *OneofDiscriminatorInfo {
	config := GetOneofConfig(oneof)
	if config == nil {
		return nil
	}

	info := &OneofDiscriminatorInfo{
		Oneof:         oneof,
		Discriminator: config.GetDiscriminator(),
		Flatten:       config.GetFlatten(),
	}

	for _, field := range oneof.Fields {
		variant := OneofVariant{
			Field:     field,
			IsMessage: field.Message != nil,
		}

		// Use custom oneof_value if set, otherwise use proto field name
		customValue := GetOneofVariantValue(field)
		if customValue != "" {
			variant.DiscriminatorVal = customValue
		} else {
			variant.DiscriminatorVal = string(field.Desc.Name())
		}

		info.Variants = append(info.Variants, variant)
	}

	return info
}

// HasOneofDiscriminator returns true if ANY oneof in the message has a discriminator annotation.
func HasOneofDiscriminator(message *protogen.Message) bool {
	for _, oneof := range message.Oneofs {
		if GetOneofConfig(oneof) != nil {
			return true
		}
	}
	return false
}

// ValidateOneofDiscriminator validates a oneof with discriminator annotation.
// Checks for:
// 1. Discriminator name collisions with parent message fields
// 2. When flatten=true: all variants must be message types
// 3. When flatten=true: variant child field names must not collide with parent fields or discriminator.
func ValidateOneofDiscriminator(
	message *protogen.Message,
	oneof *protogen.Oneof,
	config *http.OneofConfig,
) error {
	discriminator := config.GetDiscriminator()

	if err := validateDiscriminatorNameCollision(message, oneof, discriminator); err != nil {
		return err
	}

	if config.GetFlatten() {
		return validateOneofFlatten(message, oneof, discriminator)
	}

	return nil
}

// validateDiscriminatorNameCollision checks discriminator vs parent message fields.
func validateDiscriminatorNameCollision(
	message *protogen.Message,
	oneof *protogen.Oneof,
	discriminator string,
) error {
	for _, field := range message.Fields {
		if field.Oneof == oneof {
			continue // Skip oneof's own fields
		}

		if field.Desc.JSONName() == discriminator {
			return fmt.Errorf(
				"oneof %s.%s: discriminator name %q collides with field %q (JSON: %q)",
				message.Desc.Name(), oneof.Desc.Name(), discriminator,
				field.Desc.Name(), field.Desc.JSONName(),
			)
		}
	}

	return nil
}

// validateOneofFlatten validates flatten-specific rules for a discriminated oneof.
func validateOneofFlatten(
	message *protogen.Message,
	oneof *protogen.Oneof,
	discriminator string,
) error {
	// All variants must be message types when flatten=true
	for _, field := range oneof.Fields {
		if field.Message == nil {
			return fmt.Errorf(
				"oneof %s.%s with flatten=true: variant %q must be a message type (got scalar)",
				message.Desc.Name(), oneof.Desc.Name(), field.Desc.Name(),
			)
		}
	}

	// Collect parent JSON names (non-oneof fields + discriminator)
	reserved := buildReservedNames(message, oneof, discriminator)

	// Check each variant's child fields against reserved names
	for _, variantField := range oneof.Fields {
		if variantField.Message == nil {
			continue // Already caught above
		}

		for _, childField := range variantField.Message.Fields {
			childJSON := childField.Desc.JSONName()
			if source, exists := reserved[childJSON]; exists {
				return fmt.Errorf(
					"oneof %s.%s with flatten=true: variant %q child field %q (JSON: %q) collides with %s",
					message.Desc.Name(), oneof.Desc.Name(),
					variantField.Desc.Name(), childField.Desc.Name(), childJSON, source,
				)
			}
		}
	}

	return nil
}

// buildReservedNames collects JSON names reserved by parent fields and discriminator.
func buildReservedNames(
	message *protogen.Message,
	oneof *protogen.Oneof,
	discriminator string,
) map[string]string {
	reserved := make(map[string]string) // json_name -> source description
	reserved[discriminator] = "discriminator"

	for _, field := range message.Fields {
		if field.Oneof == oneof {
			continue
		}

		reserved[field.Desc.JSONName()] = fmt.Sprintf("parent field %q", field.Desc.Name())
	}

	return reserved
}
