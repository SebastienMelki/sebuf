package httpgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Proto field kind constants for type checking.
const (
	kindString    = "string"
	kindBool      = "bool"
	kindInt32     = "int32"
	kindUint32    = "uint32"
	kindSint32    = "sint32"
	kindSfixed32  = "sfixed32"
	kindInt64     = "int64"
	kindSint64    = "sint64"
	kindSfixed64  = "sfixed64"
	kindUint64    = "uint64"
	kindFixed32   = "fixed32"
	kindFixed64   = "fixed64"
	kindFloat     = "float"
	kindDouble    = "double"
	kindBytes     = "bytes"
	kindEnum      = "enum"
	kindInterface = "interface{}"
)

// RootUnwrapMessage represents a message that should unwrap at the root level.
// This means the entire message serializes to just the unwrap field's value.
type RootUnwrapMessage struct {
	Message      *protogen.Message            // The message with root unwrap
	UnwrapField  *protogen.Field              // The single field with unwrap=true
	IsMap        bool                         // True if the unwrap field is a map
	ValueMessage *protogen.Message            // For maps: the map value message type
	ValueUnwrap  *annotations.UnwrapFieldInfo // For maps: if the value message also has an unwrap field
}

// UnwrapMapField represents a map field whose value type has an unwrap field.
type UnwrapMapField struct {
	Field        *protogen.Field              // The map field
	ValueMessage *protogen.Message            // The map value message type
	UnwrapField  *annotations.UnwrapFieldInfo // The unwrap field info from the value message
}

// GlobalUnwrapInfo holds unwrap field information collected from all files.
// This enables cross-file unwrap resolution within the same Go package.
type GlobalUnwrapInfo struct {
	// UnwrapFields maps message full names to their unwrap field info.
	UnwrapFields map[string]*annotations.UnwrapFieldInfo
}

// NewGlobalUnwrapInfo creates a new GlobalUnwrapInfo instance.
func NewGlobalUnwrapInfo() *GlobalUnwrapInfo {
	return &GlobalUnwrapInfo{
		UnwrapFields: make(map[string]*annotations.UnwrapFieldInfo),
	}
}

// CollectGlobalUnwrapInfo scans all files to be generated and collects unwrap field information.
// This enables the generator to find unwrap annotations on messages defined in other files
// within the same Go package. Returns an error if any unwrap annotation is invalid.
func CollectGlobalUnwrapInfo(files []*protogen.File) (*GlobalUnwrapInfo, error) {
	global := NewGlobalUnwrapInfo()
	for _, file := range files {
		if !file.Generate {
			continue
		}
		if err := collectFileUnwrapFields(file.Messages, global); err != nil {
			return nil, fmt.Errorf("file %s: %w", file.Desc.Path(), err)
		}
	}
	return global, nil
}

func collectFileUnwrapFields(messages []*protogen.Message, global *GlobalUnwrapInfo) error {
	for _, msg := range messages {
		info, err := annotations.GetUnwrapField(msg)
		if err != nil {
			return fmt.Errorf("collecting unwrap fields for message %s: %w", msg.Desc.FullName(), err)
		}
		if info != nil {
			global.UnwrapFields[string(msg.Desc.FullName())] = info
		}
		if err = collectFileUnwrapFields(msg.Messages, global); err != nil {
			return err
		}
	}
	return nil
}

// collectAllUnwrapFields recursively collects all messages with unwrap fields.
// Returns an error if any unwrap annotation is invalid (fail-hard).
func collectAllUnwrapFields(messages []*protogen.Message) (map[string]*annotations.UnwrapFieldInfo, error) {
	result := make(map[string]*annotations.UnwrapFieldInfo)
	if err := collectUnwrapFieldsRecursive(messages, result); err != nil {
		return nil, err
	}
	return result, nil
}

func collectUnwrapFieldsRecursive(messages []*protogen.Message, result map[string]*annotations.UnwrapFieldInfo) error {
	for _, msg := range messages {
		info, err := annotations.GetUnwrapField(msg)
		if err != nil {
			return fmt.Errorf("collecting unwrap fields for message %s: %w", msg.Desc.FullName(), err)
		}
		if info != nil {
			result[string(msg.Desc.FullName())] = info
		}
		if err = collectUnwrapFieldsRecursive(msg.Messages, result); err != nil {
			return err
		}
	}
	return nil
}

// getMapValueMessage returns the message type of a map field's value, or nil if not a message.
func getMapValueMessage(field *protogen.Field) *protogen.Message {
	if field.Message == nil || len(field.Message.Fields) < 2 {
		return nil
	}

	const mapValueFieldNumber = 2
	for _, f := range field.Message.Fields {
		if f.Desc.Number() == mapValueFieldNumber {
			return f.Message
		}
	}
	return nil
}

// emitInlineMarshalChild emits the inline dispatch that forwards opts to a child's
// MarshalJSONSebuf when available and falls back to opts.Marshal otherwise. The dispatch
// is inlined (rather than calling a package-level helper) so two proto files in the same
// Go package can both produce JSON mapping files without redeclaring the helper. Output binds
// the result to local `data` / `err` variables that the caller checks.
func emitInlineMarshalChild(gf *protogen.GeneratedFile, valueExpr string) {
	gf.P("var data []byte")
	gf.P("var err error")
	gf.P(
		"if m, ok := any(",
		valueExpr,
		").(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {",
	)
	gf.P("data, err = m.MarshalJSONSebuf(opts)")
	gf.P("} else {")
	gf.P("data, err = opts.Marshal(", valueExpr, ")")
	gf.P("}")
}
