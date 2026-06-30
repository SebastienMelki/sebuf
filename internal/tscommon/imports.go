package tscommon

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// errorsModule is the extensionless module path of the shared error-helpers
// file emitted at the output root in modules mode.
const errorsModule = "errors"

// ModuleForFile returns the canonical extensionless TypeScript module path for a
// proto file path, e.g. "anghamna/core/v1/identifiers.proto" ->
// "anghamna/core/v1/identifiers".
func ModuleForFile(protoPath string) string {
	return strings.TrimSuffix(protoPath, ".proto")
}

// RelativeImportSpecifier returns the extensionless, POSIX, "./"-prefixed import
// specifier needed to reach toModule from the file at fromModule. Both arguments
// are extensionless module paths (e.g. "album/v1/service_client",
// "core/v1/identifiers").
func RelativeImportSpecifier(fromModule, toModule string) string {
	fromDir := path.Dir(fromModule)
	var rel string
	switch {
	case fromDir == "." || fromDir == "":
		rel = toModule
	default:
		baseParts := strings.Split(fromDir, "/")
		targetParts := strings.Split(toModule, "/")
		i := 0
		for i < len(baseParts) && i < len(targetParts) && baseParts[i] == targetParts[i] {
			i++
		}
		var out []string
		for j := i; j < len(baseParts); j++ {
			out = append(out, "..")
		}
		out = append(out, targetParts[i:]...)
		rel = strings.Join(out, "/")
	}
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

// importedSymbol records a single imported name and its (possibly aliased) local
// binding within the importing module.
type importedSymbol struct {
	symbol string
	alias  string // equals symbol when there is no collision
}

// ImportTracker collects the cross-module imports a single generated file needs
// and renders the import block. It is only used in modules mode.
type ImportTracker struct {
	typeImports map[string][]importedSymbol // specifier -> imported type symbols
	aliasOf     map[string]string           // "spec\x00symbol" -> local alias
	usedAlias   map[string]string           // local alias -> owning "spec\x00symbol"
	needErrors  bool
	errorsSpec  string
}

// NewImportTracker returns an empty tracker.
func NewImportTracker() *ImportTracker {
	return &ImportTracker{
		typeImports: map[string][]importedSymbol{},
		aliasOf:     map[string]string{},
		usedAlias:   map[string]string{},
	}
}

// NeedType records that `symbol` from module specifier `spec` is referenced and
// returns the local name to use (aliased deterministically on collision).
func (t *ImportTracker) NeedType(spec, symbol string) string {
	key := spec + "\x00" + symbol
	if a, ok := t.aliasOf[key]; ok {
		return a
	}
	alias := symbol
	if owner, taken := t.usedAlias[alias]; taken && owner != key {
		for i := 1; ; i++ {
			cand := fmt.Sprintf("%s_%d", symbol, i)
			if _, used := t.usedAlias[cand]; !used {
				alias = cand
				break
			}
		}
	}
	t.usedAlias[alias] = key
	t.aliasOf[key] = alias
	t.typeImports[spec] = append(t.typeImports[spec], importedSymbol{symbol: symbol, alias: alias})
	return alias
}

// NeedErrors records that the shared error helpers are referenced, importing
// them relative to the given module specifier.
func (t *ImportTracker) NeedErrors(spec string) {
	t.needErrors = true
	t.errorsSpec = spec
}

// Empty reports whether no imports were recorded.
func (t *ImportTracker) Empty() bool {
	return !t.needErrors && len(t.typeImports) == 0
}

// Render writes the import block (value import for error helpers, then sorted
// type-only imports). It emits a trailing blank line when anything was written.
func (t *ImportTracker) Render(p Printer) {
	if t.Empty() {
		return
	}
	if t.needErrors {
		p(`import { ApiError, FieldViolation, ValidationError } from "%s";`, t.errorsSpec)
	}
	specs := make([]string, 0, len(t.typeImports))
	for s := range t.typeImports {
		specs = append(specs, s)
	}
	sort.Strings(specs)
	for _, spec := range specs {
		syms := append([]importedSymbol(nil), t.typeImports[spec]...)
		sort.Slice(syms, func(i, j int) bool { return syms[i].symbol < syms[j].symbol })
		parts := make([]string, 0, len(syms))
		for _, is := range syms {
			if is.alias == is.symbol {
				parts = append(parts, is.symbol)
			} else {
				parts = append(parts, fmt.Sprintf("%s as %s", is.symbol, is.alias))
			}
		}
		p(`import type { %s } from "%s";`, strings.Join(parts, ", "), spec)
	}
	p("")
}

// EmitContext threads modules-mode state through the (otherwise stateless) type
// emitters. A nil context — or one whose ImportStyle is inline — makes every
// type reference resolve to its bare name with no import recorded, preserving
// the historical output byte-for-byte.
type EmitContext struct {
	Options    Options
	SelfModule string // extensionless module path of the file being written
	Imports    *ImportTracker
}

func (c *EmitContext) modules() bool {
	return c != nil && c.Options.ImportStyle == ImportStyleModules && c.Imports != nil
}

// oneofDiscriminated reports whether un-annotated oneofs should render as
// discriminated unions. Orthogonal to import style (works in inline mode too).
func (c *EmitContext) oneofDiscriminated() bool {
	return c != nil && c.Options.OneofStyle == OneofStyleDiscriminated
}

// RefMessage returns the local TypeScript name for a message reference,
// recording a cross-module import when needed.
func (c *EmitContext) RefMessage(msg *protogen.Message) string {
	if msg == nil {
		return ""
	}
	return c.ref(string(msg.Desc.Name()), msg.Desc.ParentFile())
}

// RefEnum returns the local TypeScript name for an enum reference, recording a
// cross-module import when needed.
func (c *EmitContext) RefEnum(enum *protogen.Enum) string {
	if enum == nil {
		return ""
	}
	return c.ref(string(enum.Desc.Name()), enum.Desc.ParentFile())
}

func (c *EmitContext) ref(symbol string, file protoreflect.FileDescriptor) string {
	if !c.modules() {
		return symbol
	}
	mod := ModuleForFile(file.Path())
	if mod == c.SelfModule {
		return symbol
	}
	return c.Imports.NeedType(RelativeImportSpecifier(c.SelfModule, mod), symbol)
}

// NeedErrors records that this file references the shared error helpers.
func (c *EmitContext) NeedErrors() {
	if !c.modules() {
		return
	}
	c.Imports.NeedErrors(RelativeImportSpecifier(c.SelfModule, errorsModule))
}
