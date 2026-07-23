package tscommon

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// errorsModule is the extensionless module path of the shared error-helpers
// file emitted at the output root in modules mode.
const errorsModule = "errors"

// Names of the value exports of the shared errors module (see WriteErrorTypes).
const (
	apiErrorName        = "ApiError"
	fieldViolationName  = "FieldViolation"
	validationErrorName = "ValidationError"
)

// errorHelperNames returns the value exports of the shared errors module,
// sorted. Every ImportTracker pre-reserves them as local names so an imported
// proto type or enum whose emitted TS name collides with a helper is
// deterministically aliased (e.g. ApiError_1) instead of producing a duplicate
// identifier next to the helper import.
func errorHelperNames() []string {
	return []string{apiErrorName, fieldViolationName, validationErrorName}
}

// reservedGlobalNames returns the global (DOM / TS-lib) identifiers the
// generated modules reference in template code — fetch, Response, Promise,
// Record, and friends. Pre-reserving them keeps an imported proto type named
// e.g. "Response" from shadowing the global the template relies on, which
// would otherwise typecheck the template code against the proto type.
func reservedGlobalNames() []string {
	return []string{
		"AbortSignal", "AsyncGenerator", "Error", "Headers", "Promise",
		"ReadableStream", "Record", "Request", "Response", "TextDecoder",
		"TextEncoder", "URL", "URLSearchParams", "fetch",
	}
}

// reservedOwnerKey marks a local name as pre-reserved in ImportTracker. The
// leading NUL can never collide with a real "spec\x00symbol" owner key because
// specifiers are never empty.
const reservedOwnerKey = "\x00reserved"

// ModuleForFile returns the canonical extensionless TypeScript module path for a
// proto file path, e.g. "anghamna/core/v1/identifiers.proto" ->
// "anghamna/core/v1/identifiers".
func ModuleForFile(protoPath string) string {
	return strings.TrimSuffix(protoPath, ".proto")
}

// RelativeImportSpecifier returns the POSIX, "./"-prefixed import specifier
// needed to reach toModule from the file at fromModule, suffixed with ".js" so
// the emitted imports resolve under Node ESM / TypeScript nodenext module
// resolution (which requires explicit extensions on relative imports). Both
// arguments are extensionless module paths (e.g. "album/v1/service_client",
// "core/v1/identifiers"); the result is e.g. "./album.js" or
// "../../core/v1/identifiers.js".
func RelativeImportSpecifier(fromModule, toModule string) string {
	fromDir := path.Dir(fromModule)
	var rel string
	switch fromDir {
	case ".", "":
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
	return rel + ".js"
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
	errorSyms   map[string]bool             // error helpers referenced (value import)
	errorsSpec  string
}

// NewImportTracker returns an empty tracker with the error-helper and global
// template names pre-reserved, so type imports can never shadow the shared
// errors module or the globals the generated code relies on.
func NewImportTracker() *ImportTracker {
	t := &ImportTracker{
		typeImports: map[string][]importedSymbol{},
		aliasOf:     map[string]string{},
		usedAlias:   map[string]string{},
		errorSyms:   map[string]bool{},
	}
	for _, name := range errorHelperNames() {
		t.usedAlias[name] = reservedOwnerKey
	}
	for _, name := range reservedGlobalNames() {
		t.usedAlias[name] = reservedOwnerKey
	}
	return t
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

// NeedErrors records that the given shared error helpers are referenced,
// importing them (as a value import) relative to the given module specifier.
func (t *ImportTracker) NeedErrors(spec string, symbols ...string) {
	t.errorsSpec = spec
	for _, s := range symbols {
		t.errorSyms[s] = true
	}
}

// Empty reports whether no imports were recorded.
func (t *ImportTracker) Empty() bool {
	return len(t.errorSyms) == 0 && len(t.typeImports) == 0
}

// Render writes the import block (value import for error helpers, then sorted
// type-only imports). It emits a trailing blank line when anything was written.
func (t *ImportTracker) Render(p Printer) {
	if t.Empty() {
		return
	}
	if len(t.errorSyms) > 0 {
		syms := make([]string, 0, len(t.errorSyms))
		for s := range t.errorSyms {
			syms = append(syms, s)
		}
		sort.Strings(syms)
		p(`import { %s } from "%s";`, strings.Join(syms, ", "), t.errorsSpec)
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

// EmitContext threads the module-emission state through the (otherwise
// stateless) type emitters: the module currently being written and the tracker
// that collects its cross-module imports. A nil context (no import tracker)
// makes every type reference resolve to its bare name with no import recorded.
type EmitContext struct {
	SelfModule         string // extensionless module path of the file being written
	Imports            *ImportTracker
	UseProtoFieldNames bool
}

func (c *EmitContext) modules() bool {
	return c != nil && c.Imports != nil
}

// RefMessage returns the local TypeScript name for a message reference,
// recording a cross-module import when needed.
func (c *EmitContext) RefMessage(msg *protogen.Message) string {
	if msg == nil {
		return ""
	}
	return c.ref(QualifiedTSName(msg.Desc), msg.Desc.ParentFile())
}

// RefEnum returns the local TypeScript name for an enum reference, recording a
// cross-module import when needed.
func (c *EmitContext) RefEnum(enum *protogen.Enum) string {
	if enum == nil {
		return ""
	}
	return c.ref(QualifiedTSName(enum.Desc), enum.Desc.ParentFile())
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

// NeedErrors records that this file references the given shared error helpers.
func (c *EmitContext) NeedErrors(symbols ...string) {
	if !c.modules() || len(symbols) == 0 {
		return
	}
	c.Imports.NeedErrors(RelativeImportSpecifier(c.SelfModule, errorsModule), symbols...)
}

// UsedErrorSymbols returns the error-helper symbols referenced anywhere in the
// given body lines, so a generated file imports only what it uses. Matching is
// on identifier boundaries: an aliased colliding type (ApiError_1) or a longer
// identifier embedding a helper name does not trigger an import.
func UsedErrorSymbols(lines []string) []string {
	var used []string
	for _, sym := range errorHelperNames() {
		pattern := regexp.MustCompile(`\b` + sym + `\b`)
		for _, line := range lines {
			if pattern.MatchString(line) {
				used = append(used, sym)
				break
			}
		}
	}
	return used
}
