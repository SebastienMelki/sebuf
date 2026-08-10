package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// EmptyBehaviorFieldInfo holds field info with its empty behavior setting.
type EmptyBehaviorFieldInfo struct {
	Field    *protogen.Field
	Behavior http.EmptyBehavior
}

// hasEmptyBehaviorFields returns true if any message field has empty_behavior annotation.
func hasEmptyBehaviorFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if annotations.HasEmptyBehaviorAnnotation(field) {
			return true
		}
	}
	return false
}

// validateEmptyBehaviorAnnotations validates all empty_behavior annotations in a file.
func validateEmptyBehaviorAnnotations(file *protogen.File) error {
	return validateEmptyBehaviorInMessages(file.Messages)
}

func validateEmptyBehaviorInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateEmptyBehaviorAnnotation(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		if err := validateEmptyBehaviorInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// generateEmptyBehaviorFieldMarshal emits the field-level marshal transform for empty_behavior.
func (g *Generator) generateEmptyBehaviorFieldMarshal(gf *protogen.GeneratedFile, fieldInfo *EmptyBehaviorFieldInfo) {
	field := fieldInfo.Field
	goName := field.GoName
	behavior := fieldInfo.Behavior

	gf.P("// Handle empty_behavior for field: ", field.Desc.Name())
	gf.P("if x.", goName, " != nil && proto.Size(x.", goName, ") == 0 {")

	switch behavior {
	case http.EmptyBehavior_EMPTY_BEHAVIOR_NULL:
		gf.P("// EMPTY_BEHAVIOR_NULL: serialize empty message as null")
		emitRawFieldSetForMarshalOptions(gf, field, `[]byte("null")`)
	case http.EmptyBehavior_EMPTY_BEHAVIOR_OMIT:
		gf.P("// EMPTY_BEHAVIOR_OMIT: remove field when message is empty")
		emitRawFieldDeleteAllJSONKeys(gf, field)
	case http.EmptyBehavior_EMPTY_BEHAVIOR_PRESERVE:
		gf.P("// EMPTY_BEHAVIOR_PRESERVE: keep as {} (default protojson behavior)")
		gf.P("// No action needed - protojson already emits {}")
	case http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED:
		gf.P("// EMPTY_BEHAVIOR_UNSPECIFIED: use default (PRESERVE)")
	}

	gf.P("}")
	gf.P()
}
