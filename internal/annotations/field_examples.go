package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// GetFieldExamples extracts example values from field options.
// Returns nil if no field examples annotation is present.
func GetFieldExamples(field *protogen.Field) []string {
	options := field.Desc.Options()
	if options == nil {
		return nil
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return nil
	}

	ext := proto.GetExtension(fieldOptions, http.E_FieldExamples)
	if ext == nil {
		return nil
	}

	fieldExamples, ok := ext.(*http.FieldExamples)
	if !ok || fieldExamples == nil {
		return nil
	}

	return fieldExamples.GetValues()
}
