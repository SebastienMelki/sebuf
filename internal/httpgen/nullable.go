package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// hasNullableFields returns true if any field in the message has nullable=true.
func hasNullableFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if annotations.IsNullableField(field) {
			return true
		}
	}
	return false
}

// validateNullableAnnotations validates all nullable annotations in a file.
func validateNullableAnnotations(file *protogen.File) error {
	return validateNullableInMessages(file.Messages)
}

func validateNullableInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateNullableAnnotation(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		if err := validateNullableInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}
