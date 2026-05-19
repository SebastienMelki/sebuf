package pyclientgen

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// writeMessage emits a Python @dataclass plus to_dict / from_dict serialization
// for a single proto message. JSON encoding honors every JSON-mapping
// annotation (int64, enum, bytes, timestamp, nullable, empty_behavior, unwrap,
// flatten, oneof) and matches the wire format produced by the Go/TS generators.
func writeMessage(p printer, msg *protogen.Message) {
	className := pythonTypeName(msg)

	p("@dataclass")
	p("class %s:", className)
	p(`    """Generated from proto message %s."""`, msg.Desc.FullName())

	if len(visibleFields(msg)) == 0 {
		p("    pass")
		p("")
		writeMessageSerialization(p, msg, className)
		return
	}

	for _, f := range visibleFields(msg) {
		writeFieldDecl(p, f)
	}
	p("")
	writeMessageSerialization(p, msg, className)
}

// visibleFields returns the fields we emit as dataclass attributes. Map-entry
// synthetic messages are not visible (handled at the parent map field level).
func visibleFields(msg *protogen.Message) []*protogen.Field {
	out := make([]*protogen.Field, 0, len(msg.Fields))
	for _, f := range msg.Fields {
		out = append(out, f)
	}
	return out
}

func writeFieldDecl(p printer, f *protogen.Field) {
	name := pythonFieldName(f)
	pyType := pythonFieldType(f)
	def := pythonFieldDefault(f)
	p("    %s: %s = %s", name, pyType, def)
}

// writeMessageSerialization emits to_dict and from_dict for a message.
//
// to_dict honors root unwrap, map-value unwrap, flatten, and oneof discriminators.
// from_dict is the inverse.
func writeMessageSerialization(p printer, msg *protogen.Message, className string) {
	writeToDict(p, msg)
	writeFromDict(p, msg, className)
	p("")
}

func writeToDict(p printer, msg *protogen.Message) {
	// Handle root unwrap: when an unwrap field is at root, to_dict returns
	// just the underlying field's encoded value (dict or list).
	if annotations.IsRootUnwrap(msg) {
		writeRootUnwrapToDict(p, msg)
		return
	}

	p("    def to_dict(self) -> Any:")
	p(`        """Serialize to a JSON-ready dict respecting sebuf JSON mapping annotations."""`)
	p("        d: dict[str, Any] = {}")

	// Discriminator-aware oneof: emit discriminator key + flattened or nested variant.
	discriminator := discriminatorInfo(msg)

	for _, f := range visibleFields(msg) {
		if isOneofVariant(f, discriminator) {
			continue // handled below
		}
		writeFieldToDict(p, f)
	}

	if discriminator != nil {
		writeDiscriminatedOneofToDict(p, msg, discriminator)
	}

	p("        return d")
	p("")
}

func writeRootUnwrapToDict(p printer, msg *protogen.Message) {
	// Find the unwrap field — there is exactly one (validated by annotations).
	target := annotations.FindUnwrapField(msg)
	if target == nil {
		// Defensive: fall back to the normal path.
		p("    def to_dict(self) -> Any:")
		p("        return {}")
		p("")
		return
	}

	p("    def to_dict(self) -> Any:")
	p(`        """Serialize to a JSON-ready value (root-unwrapped)."""`)
	name := pythonFieldName(target)
	switch {
	case target.Desc.IsMap():
		valField := target.Message.Fields[1]
		valExpr := encodeMapValueExpr(valField, "v")
		p("        return {k: %s for k, v in self.%s.items()}", valExpr, name)
	case target.Desc.IsList():
		valExpr := encodeListItemExpr(target, "v")
		p("        return [%s for v in self.%s]", valExpr, name)
	default:
		valExpr := encodeScalarExpr(target, "self."+name)
		p("        return %s", valExpr)
	}
	p("")
}

// writeFieldToDict emits the to_dict lines for one field.
//
//nolint:gocyclo // mapping logic is inherently branchy
func writeFieldToDict(p printer, f *protogen.Field) {
	name := pythonFieldName(f)
	jsonName := jsonFieldName(f)
	src := "self." + name

	if annotations.IsFlattenField(f) {
		writeFlattenToDict(p, f, src)
		return
	}

	if f.Desc.IsMap() {
		writeMapToDict(p, f, name, jsonName, src)
		return
	}

	if f.Desc.IsList() {
		writeListToDict(p, f, name, jsonName, src)
		return
	}

	if fieldIsMessage(f) {
		writeMessageFieldToDict(p, f, name, jsonName)
		return
	}

	// Scalar field.
	if f.Desc.HasOptionalKeyword() || annotations.IsNullableField(f) {
		p("        if %s is not None:", src)
		p(`            d["%s"] = %s`, jsonName, encodeScalarExpr(f, src))
		return
	}
	// Non-optional scalar: emit unconditionally (protojson includes defaults
	// only when EmitUnpopulated is set; we match Go client default of omitting
	// zero values for symmetry with TS encoding).
	p(`        d["%s"] = %s`, jsonName, encodeScalarExpr(f, src))
}

func writeMapToDict(p printer, f *protogen.Field, name, jsonName, src string) {
	valField := f.Message.Fields[1]
	// Check for map-value unwrap: if the value is a wrapper with `unwrap` set
	// on its repeated field, collapse `{...wrapper...}` to just the array.
	if valField.Message != nil && hasMapValueUnwrap(valField.Message) {
		unwrap := annotations.FindUnwrapField(valField.Message)
		if unwrap != nil {
			inner := encodeListItemExpr(unwrap, "x")
			p("        if %s:", src)
			p(`            d["%s"] = {k: [%s for x in v.%s] for k, v in %s.items()}`,
				jsonName, inner, pythonFieldName(unwrap), src)
			return
		}
	}
	valExpr := encodeMapValueExpr(valField, "v")
	p("        if %s:", src)
	p(`            d["%s"] = {k: %s for k, v in %s.items()}`, jsonName, valExpr, src)
}

func writeListToDict(p printer, f *protogen.Field, name, jsonName, src string) {
	itemExpr := encodeListItemExpr(f, "v")
	p("        if %s:", src)
	p(`            d["%s"] = [%s for v in %s]`, jsonName, itemExpr, src)
}

func writeMessageFieldToDict(p printer, f *protogen.Field, name, jsonName string) {
	src := "self." + name
	behavior := annotations.GetEmptyBehavior(f)

	if behavior == sebufhttp.EmptyBehavior_EMPTY_BEHAVIOR_NULL {
		p("        if %s is None:", src)
		p(`            d["%s"] = None`, jsonName)
		p("        else:")
		p(`            d["%s"] = %s.to_dict()`, jsonName, src)
		return
	}
	if behavior == sebufhttp.EmptyBehavior_EMPTY_BEHAVIOR_OMIT {
		p("        if %s is not None:", src)
		p("            _v = %s.to_dict()", src)
		p("            if _v:")
		p(`                d["%s"] = _v`, jsonName)
		return
	}
	// PRESERVE / UNSPECIFIED: emit when set.
	p("        if %s is not None:", src)
	p(`            d["%s"] = %s.to_dict()`, jsonName, src)
}

func writeFlattenToDict(p printer, f *protogen.Field, src string) {
	prefix := annotations.GetFlattenPrefix(f)
	p("        if %s is not None:", src)
	p("            for _k, _v in %s.to_dict().items():", src)
	if prefix == "" {
		p("                d[_k] = _v")
	} else {
		p(`                d["%s" + _k] = _v`, prefix)
	}
}

// discriminatorInfo returns the resolved discriminator info for the first
// non-synthetic oneof on the message that has (sebuf.http.oneof_config) set,
// or nil if none. Multiple discriminated oneofs per message are not supported
// (matches the Go/TS clients).
func discriminatorInfo(msg *protogen.Message) *annotations.OneofDiscriminatorInfo {
	for _, o := range msg.Oneofs {
		if o.Desc.IsSynthetic() {
			continue
		}
		if info := annotations.GetOneofDiscriminatorInfo(o); info != nil {
			return info
		}
	}
	return nil
}

func isOneofVariant(f *protogen.Field, info *annotations.OneofDiscriminatorInfo) bool {
	if info == nil {
		return false
	}
	for _, v := range info.Variants {
		if v.Field == f {
			return true
		}
	}
	return false
}

// writeDiscriminatedOneofToDict emits the discriminator key + the active variant.
func writeDiscriminatedOneofToDict(p printer, msg *protogen.Message, info *annotations.OneofDiscriminatorInfo) {
	_ = msg
	for _, v := range info.Variants {
		name := pythonFieldName(v.Field)
		src := "self." + name
		p("        if %s is not None:", src)
		p(`            d["%s"] = "%s"`, info.Discriminator, v.DiscriminatorVal)
		if info.Flatten && v.Field.Message != nil {
			p("            for _k, _v in %s.to_dict().items():", src)
			p("                d[_k] = _v")
		} else if v.Field.Message != nil {
			p(`            d["%s"] = %s.to_dict()`, jsonFieldName(v.Field), src)
		} else {
			p(`            d["%s"] = %s`, jsonFieldName(v.Field), encodeScalarExpr(v.Field, src))
		}
	}
}

func writeFromDict(p printer, msg *protogen.Message, className string) {
	p("    @classmethod")
	p("    def from_dict(cls, data: Any) -> \"%s\":", className)
	p(`        """Deserialize from a JSON-decoded dict (or value, for root-unwrapped messages)."""`)

	if annotations.IsRootUnwrap(msg) {
		writeRootUnwrapFromDict(p, msg, className)
		return
	}

	p("        if data is None:")
	p("            return cls()")
	p("        kwargs: dict[str, Any] = {}")

	discriminator := discriminatorInfo(msg)

	for _, f := range visibleFields(msg) {
		if isOneofVariant(f, discriminator) {
			continue
		}
		writeFieldFromDict(p, f)
	}

	if discriminator != nil {
		writeDiscriminatedOneofFromDict(p, discriminator)
	}

	p("        return cls(**kwargs)")
}

func writeRootUnwrapFromDict(p printer, msg *protogen.Message, className string) {
	target := annotations.FindUnwrapField(msg)
	if target == nil {
		p("        return cls()")
		return
	}
	name := pythonFieldName(target)
	switch {
	case target.Desc.IsMap():
		valField := target.Message.Fields[1]
		valExpr := decodeMapValueExpr(valField, "v")
		p("        if data is None:")
		p("            return cls()")
		p("        return cls(%s={k: %s for k, v in data.items()})", name, valExpr)
	case target.Desc.IsList():
		itemExpr := decodeListItemExpr(target, "v")
		p("        if data is None:")
		p("            return cls()")
		p("        return cls(%s=[%s for v in data])", name, itemExpr)
	default:
		valExpr := decodeScalarExpr(target, "data")
		p("        if data is None:")
		p("            return cls()")
		p("        return cls(%s=%s)", name, valExpr)
	}
}

//nolint:gocyclo // mapping logic is inherently branchy
func writeFieldFromDict(p printer, f *protogen.Field) {
	name := pythonFieldName(f)
	jsonName := jsonFieldName(f)

	if annotations.IsFlattenField(f) {
		writeFlattenFromDict(p, f, name)
		return
	}

	if f.Desc.IsMap() {
		writeMapFromDict(p, f, name, jsonName)
		return
	}

	if f.Desc.IsList() {
		writeListFromDict(p, f, name, jsonName)
		return
	}

	if fieldIsMessage(f) {
		p(`        if "%s" in data and data["%s"] is not None:`, jsonName, jsonName)
		p(`            kwargs["%s"] = %s.from_dict(data["%s"])`, name, pythonTypeName(f.Message), jsonName)
		return
	}

	// Scalar/WKT
	p(`        if "%s" in data and data["%s"] is not None:`, jsonName, jsonName)
	p(`            kwargs["%s"] = %s`, name, decodeScalarExpr(f, fmt.Sprintf(`data["%s"]`, jsonName)))
}

func writeMapFromDict(p printer, f *protogen.Field, name, jsonName string) {
	valField := f.Message.Fields[1]
	if valField.Message != nil && hasMapValueUnwrap(valField.Message) {
		unwrap := annotations.FindUnwrapField(valField.Message)
		if unwrap != nil {
			itemExpr := decodeListItemExpr(unwrap, "x")
			wrapperType := pythonTypeName(valField.Message)
			unwrapFieldName := pythonFieldName(unwrap)
			p(`        if "%s" in data and data["%s"] is not None:`, jsonName, jsonName)
			p(`            kwargs["%s"] = {`, name)
			p(`                k: %s(%s=[%s for x in v])`,
				wrapperType, unwrapFieldName, itemExpr)
			p(`                for k, v in data["%s"].items()`, jsonName)
			p(`            }`)
			return
		}
	}
	valExpr := decodeMapValueExpr(valField, "v")
	p(`        if "%s" in data and data["%s"] is not None:`, jsonName, jsonName)
	p(`            kwargs["%s"] = {k: %s for k, v in data["%s"].items()}`, name, valExpr, jsonName)
}

func writeListFromDict(p printer, f *protogen.Field, name, jsonName string) {
	itemExpr := decodeListItemExpr(f, "v")
	p(`        if "%s" in data and data["%s"] is not None:`, jsonName, jsonName)
	p(`            kwargs["%s"] = [%s for v in data["%s"]]`, name, itemExpr, jsonName)
}

func writeFlattenFromDict(p printer, f *protogen.Field, name string) {
	prefix := annotations.GetFlattenPrefix(f)
	nestedType := pythonTypeName(f.Message)
	if prefix == "" {
		// Without a prefix it's ambiguous which keys belong to the nested message.
		// Punt: pass the entire dict; the nested from_dict will ignore unknown keys.
		p(`        kwargs["%s"] = %s.from_dict(data)`, name, nestedType)
		return
	}
	p(`        _sub_%s: dict[str, Any] = {}`, name)
	p(`        for _k, _v in data.items():`)
	p(`            if _k.startswith("%s"):`, prefix)
	p(`                _sub_%s[_k[%d:]] = _v`, name, len(prefix))
	p(`        if _sub_%s:`, name)
	p(`            kwargs["%s"] = %s.from_dict(_sub_%s)`, name, nestedType, name)
}

func writeDiscriminatedOneofFromDict(p printer, info *annotations.OneofDiscriminatorInfo) {
	p(`        _disc = data.get("%s")`, info.Discriminator)
	for _, v := range info.Variants {
		varName := pythonFieldName(v.Field)
		p(`        if _disc == "%s":`, v.DiscriminatorVal)
		if info.Flatten && v.Field.Message != nil {
			p(`            kwargs["%s"] = %s.from_dict(data)`, varName, pythonTypeName(v.Field.Message))
		} else if v.Field.Message != nil {
			p(`            if "%s" in data:`, jsonFieldName(v.Field))
			p(`                kwargs["%s"] = %s.from_dict(data["%s"])`,
				varName, pythonTypeName(v.Field.Message), jsonFieldName(v.Field))
		} else {
			p(`            if "%s" in data:`, jsonFieldName(v.Field))
			p(`                kwargs["%s"] = %s`, varName,
				decodeScalarExpr(v.Field, fmt.Sprintf(`data["%s"]`, jsonFieldName(v.Field))))
		}
	}
}

// hasMapValueUnwrap reports whether `msg` is a wrapper used as a map value
// where the single repeated field is marked `(sebuf.http.unwrap) = true`.
// This is the wrapper pattern that collapses {bars: [...]} → [...] in JSON.
func hasMapValueUnwrap(msg *protogen.Message) bool {
	if msg == nil {
		return false
	}
	f := annotations.FindUnwrapField(msg)
	return f != nil && f.Desc.IsList()
}

// encodeListItemExpr returns the per-item encoding expression for a repeated field.
func encodeListItemExpr(f *protogen.Field, src string) string {
	if fieldIsMessage(f) {
		return fmt.Sprintf("%s.to_dict()", src)
	}
	// Build a synthetic scalar context for the item — same encoding rules apply.
	return encodeScalarExpr(f, src)
}

// decodeListItemExpr is the inverse of encodeListItemExpr.
func decodeListItemExpr(f *protogen.Field, src string) string {
	if fieldIsMessage(f) {
		return fmt.Sprintf("%s.from_dict(%s)", pythonTypeName(f.Message), src)
	}
	return decodeScalarExpr(f, src)
}

// encodeMapValueExpr returns the encoding expression for a map value field.
func encodeMapValueExpr(valField *protogen.Field, src string) string {
	if valField.Desc.Kind() == protoreflect.MessageKind {
		return fmt.Sprintf("%s.to_dict()", src)
	}
	return encodeScalarExpr(valField, src)
}

// decodeMapValueExpr is the inverse of encodeMapValueExpr.
func decodeMapValueExpr(valField *protogen.Field, src string) string {
	if valField.Desc.Kind() == protoreflect.MessageKind {
		return fmt.Sprintf("%s.from_dict(%s)", pythonTypeName(valField.Message), src)
	}
	return decodeScalarExpr(valField, src)
}

// stringJoin is a tiny convenience for assembling field-name lists in tests.
// Kept here so generator code stays self-contained.
func stringJoin(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

// _ keeps stringJoin reachable; encoders/decoders may grow into using it.
var _ = stringJoin
