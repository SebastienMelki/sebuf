package pyclientgen

import (
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// collectedTypes captures every message and enum reachable from a file's services
// plus all top-level messages and enums declared in the file.
//
// Python output emits each type at most once with deterministic ordering so golden
// tests stay stable.
type collectedTypes struct {
	messages  map[string]*protogen.Message
	enums     map[string]*protogen.Enum
	errorMsgs map[string]*protogen.Message // *Error messages emitted as Exception classes
}

func collectFileTypes(file *protogen.File) *collectedTypes {
	c := &collectedTypes{
		messages:  make(map[string]*protogen.Message),
		enums:     make(map[string]*protogen.Enum),
		errorMsgs: make(map[string]*protogen.Message),
	}

	// Walk top-level messages and enums (covers everything the user declared).
	for _, msg := range file.Messages {
		c.addMessage(msg)
	}
	for _, enum := range file.Enums {
		c.addEnum(enum)
	}

	// Walk service inputs/outputs to ensure imported messages from other files
	// would be added too (currently we ignore cross-file references — same as TS).
	for _, svc := range file.Services {
		for _, m := range svc.Methods {
			c.addMessage(m.Input)
			c.addMessage(m.Output)
		}
	}

	return c
}

func (c *collectedTypes) addMessage(msg *protogen.Message) {
	if msg == nil {
		return
	}
	if msg.Desc.IsMapEntry() {
		// Map entries are synthetic. Walk fields to capture nested value types,
		// but do not emit a dataclass for the entry itself.
		for _, f := range msg.Fields {
			if f.Message != nil {
				c.addMessage(f.Message)
			}
			if f.Enum != nil {
				c.addEnum(f.Enum)
			}
		}
		return
	}

	name := pythonTypeName(msg)
	if _, ok := c.messages[name]; ok {
		return
	}
	c.messages[name] = msg

	if strings.HasSuffix(string(msg.Desc.Name()), "Error") {
		c.errorMsgs[name] = msg
	}

	for _, nested := range msg.Messages {
		c.addMessage(nested)
	}
	for _, nestedEnum := range msg.Enums {
		c.addEnum(nestedEnum)
	}
	for _, f := range msg.Fields {
		if f.Message != nil {
			c.addMessage(f.Message)
		}
		if f.Enum != nil {
			c.addEnum(f.Enum)
		}
	}
}

func (c *collectedTypes) addEnum(enum *protogen.Enum) {
	if enum == nil {
		return
	}
	name := pythonEnumName(enum)
	if _, ok := c.enums[name]; ok {
		return
	}
	c.enums[name] = enum
}

// OrderedMessages returns messages sorted by their Python class name for
// deterministic output. Error messages are returned by ErrorMessages instead
// and excluded here so we don't emit a dataclass and an Exception for the same type.
func (c *collectedTypes) OrderedMessages() []*protogen.Message {
	out := make([]*protogen.Message, 0, len(c.messages))
	for name, msg := range c.messages {
		if _, isErr := c.errorMsgs[name]; isErr {
			continue
		}
		out = append(out, msg)
	}
	sort.Slice(out, func(i, j int) bool {
		return pythonTypeName(out[i]) < pythonTypeName(out[j])
	})
	return out
}

// OrderedEnums returns enums sorted by Python class name.
func (c *collectedTypes) OrderedEnums() []*protogen.Enum {
	out := make([]*protogen.Enum, 0, len(c.enums))
	for _, e := range c.enums {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return pythonEnumName(out[i]) < pythonEnumName(out[j])
	})
	return out
}

// OrderedErrors returns *Error messages sorted by Python class name.
func (c *collectedTypes) OrderedErrors() []*protogen.Message {
	out := make([]*protogen.Message, 0, len(c.errorMsgs))
	for _, m := range c.errorMsgs {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		return pythonTypeName(out[i]) < pythonTypeName(out[j])
	})
	return out
}

// pythonTypeName returns the Python class name for a proto message. For nested
// messages we flatten using underscore separation (Outer.Inner -> Outer_Inner)
// because Python lacks proto's nested-name resolution.
func pythonTypeName(msg *protogen.Message) string {
	// GoIdent.GoName uses Go's convention (Outer_Inner) which matches what we want.
	return msg.GoIdent.GoName
}

// pythonEnumName returns the Python class name for a proto enum.
func pythonEnumName(enum *protogen.Enum) string {
	return enum.GoIdent.GoName
}
